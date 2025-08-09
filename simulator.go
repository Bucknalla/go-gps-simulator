package main

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"
)

type GPSSimulator struct {
	config         Config
	currentLat     float64
	currentLon     float64
	currentAlt     float64
	currentSpeed   float64 // Current speed with jitter applied (knots)
	currentCourse  float64 // Current course with jitter applied (degrees)
	isLocked       bool
	lockTime       time.Time
	startTime      time.Time
	lastUpdateTime time.Time
	satellites     []Satellite
	nmeaWriter     io.Writer
}

type Satellite struct {
	ID        int
	Elevation int // degrees above horizon
	Azimuth   int // degrees from north
	SNR       int // signal-to-noise ratio
}

func NewGPSSimulator(config Config, nmeaWriter io.Writer) *GPSSimulator {
	now := time.Now()
	sim := &GPSSimulator{
		config:         config,
		currentLat:     config.Latitude,
		currentLon:     config.Longitude,
		currentAlt:     config.Altitude,
		currentSpeed:   config.Speed,
		currentCourse:  config.Course,
		isLocked:       false,
		startTime:      now,
		lockTime:       now.Add(config.TimeToLock),
		lastUpdateTime: now,
		nmeaWriter:     nmeaWriter,
	}

	// Initialize satellites
	sim.initializeSatellites()

	return sim
}

func (s *GPSSimulator) initializeSatellites() {
	s.satellites = make([]Satellite, s.config.Satellites)

	for i := 0; i < s.config.Satellites; i++ {
		s.satellites[i] = Satellite{
			ID:        i + 1,
			Elevation: rand.Intn(70) + 10, // 10-80 degrees
			Azimuth:   rand.Intn(360),     // 0-359 degrees
			SNR:       rand.Intn(30) + 20, // 20-50 dB
		}
	}
}

func (s *GPSSimulator) Run() {
	ticker := time.NewTicker(s.config.OutputRate)
	defer ticker.Stop()

	for range ticker.C {
		s.update()
		s.outputNMEA()
	}
}

func (s *GPSSimulator) update() {
	now := time.Now()

	// Check if GPS should be locked
	if !s.isLocked && now.After(s.lockTime) {
		s.isLocked = true
		if !s.config.Quiet {
			fmt.Fprintf(os.Stderr, "GPS LOCKED after %v\n", now.Sub(s.startTime))
		}
	}

	// Update position if locked
	if s.isLocked {
		s.updateSpeedAndCourse()
		s.updatePosition()
		s.updateAltitude()
	}

	// Update satellites
	s.updateSatellites()
}

func (s *GPSSimulator) updateSpeedAndCourse() {
	// Apply jitter to speed and course based on jitter configuration
	var speedVariation, courseVariation float64

	if s.config.Jitter < 0.2 {
		// Low jitter: minimal variation (±5% speed, ±2° course)
		speedVariation = 0.05
		courseVariation = 2.0
	} else if s.config.Jitter < 0.7 {
		// Medium jitter: moderate variation (±10-30% speed, ±5-15° course)
		speedVariation = 0.10 + (s.config.Jitter-0.2)*0.40 // 10% to 30%
		courseVariation = 5.0 + (s.config.Jitter-0.2)*20.0 // 5° to 15°
	} else {
		// High jitter: large variation (±50% speed, ±30° course)
		speedVariation = 0.30 + (s.config.Jitter-0.7)*0.67  // 30% to 50%
		courseVariation = 15.0 + (s.config.Jitter-0.7)*50.0 // 15° to 30°
	}

	// Apply speed variation
	speedDelta := (rand.Float64() - 0.5) * 2 * s.config.Speed * speedVariation
	s.currentSpeed = s.config.Speed + speedDelta
	if s.currentSpeed < 0 {
		s.currentSpeed = 0 // Speed cannot be negative
	}

	// Apply course variation
	courseDelta := (rand.Float64() - 0.5) * 2 * courseVariation
	s.currentCourse = s.config.Course + courseDelta

	// Normalize course to 0-359.9 range
	for s.currentCourse < 0 {
		s.currentCourse += 360
	}
	for s.currentCourse >= 360 {
		s.currentCourse -= 360
	}
}

func (s *GPSSimulator) updatePosition() {
	now := time.Now()
	deltaTime := now.Sub(s.lastUpdateTime).Seconds()
	s.lastUpdateTime = now

	// If no time has passed, don't update position
	if deltaTime <= 0 {
		return
	}

	// Convert speed from knots to meters per second
	// 1 knot = 0.514444 meters per second
	speedMPS := s.currentSpeed * 0.514444

	// Calculate distance traveled in this time interval
	distanceMeters := speedMPS * deltaTime

	// Convert course from degrees to radians (course is measured clockwise from north)
	// In math, 0° is east and angles increase counter-clockwise
	// In navigation, 0° is north and angles increase clockwise
	// Convert navigation course to math angle: mathAngle = 90° - navCourse
	mathAngleRad := (90.0 - s.currentCourse) * math.Pi / 180.0

	// Calculate position change in meters
	deltaEast := distanceMeters * math.Cos(mathAngleRad)  // Eastward displacement
	deltaNorth := distanceMeters * math.Sin(mathAngleRad) // Northward displacement

	// Convert meters to degrees (approximate)
	// At the equator: 1 degree latitude ≈ 111,320 meters
	// 1 degree longitude varies by latitude: ≈ 111,320 * cos(latitude) meters
	deltaLatDeg := deltaNorth / 111320.0
	deltaLonDeg := deltaEast / (111320.0 * math.Cos(s.currentLat*math.Pi/180.0))

	// Calculate new position
	newLat := s.currentLat + deltaLatDeg
	newLon := s.currentLon + deltaLonDeg

	// Apply radius constraint - if we're moving outside the configured radius,
	// either constrain the movement or apply some random jitter to change direction
	if s.distanceFromCenter(newLat, newLon) > s.config.Radius {
		if s.config.Jitter > 0.5 {
			// High jitter: add some randomness to course to "bounce" off boundaries
			randomCourseChange := (rand.Float64() - 0.5) * 60.0 // ±30° change
			s.currentCourse += randomCourseChange

			// Normalize course
			for s.currentCourse < 0 {
				s.currentCourse += 360
			}
			for s.currentCourse >= 360 {
				s.currentCourse -= 360
			}

			// Recalculate with new course
			mathAngleRad = (90.0 - s.currentCourse) * math.Pi / 180.0
			deltaEast = distanceMeters * math.Cos(mathAngleRad)
			deltaNorth = distanceMeters * math.Sin(mathAngleRad)
			deltaLatDeg = deltaNorth / 111320.0
			deltaLonDeg = deltaEast / (111320.0 * math.Cos(s.currentLat*math.Pi/180.0))

			newLat = s.currentLat + deltaLatDeg
			newLon = s.currentLon + deltaLonDeg
		} else {
			// Low jitter: constrain to radius boundary
			// Calculate direction from center to new position
			centerLat := s.config.Latitude
			centerLon := s.config.Longitude

			bearing := math.Atan2(
				(newLon-centerLon)*math.Cos(centerLat*math.Pi/180.0),
				newLat-centerLat,
			)

			// Place new position at radius boundary in that direction
			radiusDegLat := s.config.Radius / 111320.0
			radiusDegLon := s.config.Radius / (111320.0 * math.Cos(centerLat*math.Pi/180.0))

			newLat = centerLat + radiusDegLat*math.Cos(bearing)
			newLon = centerLon + radiusDegLon*math.Sin(bearing)/math.Cos(centerLat*math.Pi/180.0)
		}
	}

	// Update current position
	s.currentLat = newLat
	s.currentLon = newLon
}

func (s *GPSSimulator) updateAltitude() {
	// Apply altitude jitter based on configuration
	if s.config.AltitudeJitter > 0 {
		// Calculate maximum altitude change per update
		// Low jitter = small changes; High jitter = large changes
		maxChange := 1.0 + (s.config.AltitudeJitter * 20.0) // 1-21 meters max change

		// Generate random altitude change
		change := (rand.Float64() - 0.5) * 2 * maxChange // -maxChange to +maxChange

		// Apply change
		newAltitude := s.currentAlt + change

		// Keep altitude within reasonable bounds (prevent negative or extreme altitudes)
		minAltitude := s.config.Altitude - 100.0 // Allow 100m below starting altitude
		maxAltitude := s.config.Altitude + 500.0 // Allow 500m above starting altitude

		if minAltitude < -50.0 {
			minAltitude = -50.0 // Don't go too far below sea level
		}

		if newAltitude < minAltitude {
			newAltitude = minAltitude
		} else if newAltitude > maxAltitude {
			newAltitude = maxAltitude
		}

		s.currentAlt = newAltitude
	}
}

func (s *GPSSimulator) distanceFromCenter(lat, lon float64) float64 {
	// Haversine formula for distance calculation
	const R = 6371000 // Earth's radius in meters

	lat1 := s.config.Latitude * math.Pi / 180
	lat2 := lat * math.Pi / 180
	deltaLat := (lat - s.config.Latitude) * math.Pi / 180
	deltaLon := (lon - s.config.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (s *GPSSimulator) updateSatellites() {
	// Simulate satellite movement and signal changes
	for i := range s.satellites {
		// Slightly adjust elevation and azimuth
		s.satellites[i].Elevation += rand.Intn(3) - 1 // -1, 0, or 1
		s.satellites[i].Azimuth = (s.satellites[i].Azimuth + rand.Intn(3) - 1 + 360) % 360

		// Keep elevation within bounds
		if s.satellites[i].Elevation < 5 {
			s.satellites[i].Elevation = 5
		}
		if s.satellites[i].Elevation > 85 {
			s.satellites[i].Elevation = 85
		}

		// Simulate SNR variations
		s.satellites[i].SNR += rand.Intn(6) - 3 // -3 to +3
		if s.satellites[i].SNR < 15 {
			s.satellites[i].SNR = 15
		}
		if s.satellites[i].SNR > 55 {
			s.satellites[i].SNR = 55
		}
	}
}

func (s *GPSSimulator) outputNMEA() {
	timestamp := time.Now()

	if s.isLocked {
		// Output GGA sentence (Global Positioning System Fix Data)
		fmt.Fprint(s.nmeaWriter, s.generateGGA(timestamp))

		// Output RMC sentence (Recommended Minimum)
		fmt.Fprint(s.nmeaWriter, s.generateRMC(timestamp))

		// Output GSA sentence (GPS DOP and active satellites)
		fmt.Fprint(s.nmeaWriter, s.generateGSA())

		// Output GSV sentences (GPS Satellites in view)
		gsv := s.generateGSV()
		for _, sentence := range gsv {
			fmt.Fprint(s.nmeaWriter, sentence)
		}
	} else {
		// Output sentences indicating no fix
		fmt.Fprint(s.nmeaWriter, s.generateNoFixGGA(timestamp))
		fmt.Fprint(s.nmeaWriter, s.generateNoFixRMC(timestamp))
	}

	// No extra blank lines - NMEA sentences should be continuous
}
