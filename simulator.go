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
	gpxWriter      *GPXWriter
	// Replay mode fields
	replayPoints    []TrackPoint
	replayIndex     int
	replayStartTime time.Time
	replayCompleted bool // Track if we've completed one full pass through the replay
}

type Satellite struct {
	ID        int
	Elevation int // degrees above horizon
	Azimuth   int // degrees from north
	SNR       int // signal-to-noise ratio
}

func NewGPSSimulator(config Config, nmeaWriter io.Writer) (*GPSSimulator, error) {
	now := time.Now()
	sim := &GPSSimulator{
		config:          config,
		currentLat:      config.Latitude,
		currentLon:      config.Longitude,
		currentAlt:      config.Altitude,
		currentSpeed:    config.Speed,
		currentCourse:   config.Course,
		isLocked:        false,
		startTime:       now,
		lockTime:        now.Add(config.TimeToLock),
		lastUpdateTime:  now,
		nmeaWriter:      nmeaWriter,
		replayIndex:     0,
		replayStartTime: now,
		replayCompleted: false,
	}

	// Load GPX file for replay mode
	if config.ReplayFile != "" {
		points, err := ReadGPXFile(config.ReplayFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load replay file: %v", err)
		}
		sim.replayPoints = points

		// Set initial position from first track point
		if len(points) > 0 {
			sim.currentLat = points[0].Lat
			sim.currentLon = points[0].Lon
			sim.currentAlt = points[0].Elevation
		}
	}

	// Initialize GPX writer if GPX is enabled
	if config.GPXEnabled {
		gpxWriter, err := NewGPXWriter(config.GPXFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create GPX writer: %v", err)
		}
		sim.gpxWriter = gpxWriter
	}

	// Initialize satellites
	sim.initializeSatellites()

	return sim, nil
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

	// Ensure GPX writer is closed when simulation ends
	defer s.Close()

	// Set up duration timer if specified
	var durationTimer *time.Timer
	var durationChan <-chan time.Time
	if s.config.Duration > 0 {
		durationTimer = time.NewTimer(s.config.Duration)
		durationChan = durationTimer.C
		defer durationTimer.Stop()

		if !s.config.Quiet {
			fmt.Fprintf(os.Stderr, "Simulation will run for %v\n", s.config.Duration)
		}
	}

	for {
		select {
		case <-ticker.C:
			s.update()
			s.outputNMEA()
			s.updateGPX()

			// Check if replay is completed and looping is disabled
			if s.config.ReplayFile != "" && !s.config.ReplayLoop && s.replayCompleted {
				if !s.config.Quiet {
					fmt.Fprintf(os.Stderr, "\nGPX replay completed\n")
				}
				return
			}
		case <-durationChan:
			if !s.config.Quiet {
				fmt.Fprintf(os.Stderr, "\nSimulation completed after %v\n", s.config.Duration)
			}
			return
		}
	}
}

// Close closes any open resources (like GPX writer)
func (s *GPSSimulator) Close() {
	if s.gpxWriter != nil {
		if !s.config.Quiet {
			fmt.Fprintf(os.Stderr, "Writing GPX file: %s with %d track points\n",
				s.config.GPXFile, s.gpxWriter.GetTrackPointCount())
		}
		err := s.gpxWriter.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error closing GPX file: %v\n", err)
		}
	}
}

// updateGPX adds current position to GPX track if GPX writer is enabled and GPS is locked
func (s *GPSSimulator) updateGPX() {
	if s.gpxWriter != nil && s.isLocked {
		s.gpxWriter.AddTrackPoint(s.currentLat, s.currentLon, s.currentAlt, time.Now())

		// Write to file periodically to avoid losing data if program is interrupted
		// Write every 10 points to balance between performance and data safety
		if s.gpxWriter.GetTrackPointCount()%10 == 0 {
			err := s.gpxWriter.WriteToFile()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing GPX data: %v\n", err)
			}
		}
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
		if s.config.ReplayFile != "" {
			s.updateReplayPosition()
		} else {
			s.updateSpeedAndCourse()
			s.updatePosition()
			s.updateAltitude()
		}
	}

	// Update satellites
	s.updateSatellites()
}

func (s *GPSSimulator) updateSpeedAndCourse() {
	// Apply jitter to speed and course based on jitter configuration
	var speedVariation, courseVariation float64

	if s.config.Jitter == 0.0 {
		// Zero jitter: no variation at all
		speedVariation = 0.0
		courseVariation = 0.0
	} else if s.config.Jitter < 0.2 {
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
	return s.calculateDistance(s.config.Latitude, s.config.Longitude, lat, lon)
}

// hasSequentialTimestamps checks if the replay points have sequential timestamps
func (s *GPSSimulator) hasSequentialTimestamps() bool {
	if len(s.replayPoints) < 2 {
		return false
	}

	// Check if timestamps are generally increasing
	for i := 0; i < len(s.replayPoints)-1; i++ {
		if s.replayPoints[i+1].Time.Before(s.replayPoints[i].Time) {
			return false
		}
	}
	return true
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

		// Output GLL sentence (Geographic Position - Latitude/Longitude)
		fmt.Fprint(s.nmeaWriter, s.generateGLL(timestamp))

		// Output VTG sentence (Track Made Good and Ground Speed)
		fmt.Fprint(s.nmeaWriter, s.generateVTG())

		// Output GSA sentence (GPS DOP and active satellites)
		fmt.Fprint(s.nmeaWriter, s.generateGSA())

		// Output GSV sentences (GPS Satellites in view)
		gsv := s.generateGSV()
		for _, sentence := range gsv {
			fmt.Fprint(s.nmeaWriter, sentence)
		}

		// Output ZDA sentence (UTC Date and Time)
		fmt.Fprint(s.nmeaWriter, s.generateZDA(timestamp))
	} else {
		// Output sentences indicating no fix
		fmt.Fprint(s.nmeaWriter, s.generateNoFixGGA(timestamp))
		fmt.Fprint(s.nmeaWriter, s.generateNoFixRMC(timestamp))
		fmt.Fprint(s.nmeaWriter, s.generateNoFixGLL(timestamp))
		fmt.Fprint(s.nmeaWriter, s.generateNoFixVTG())
	}

	// No extra blank lines - NMEA sentences should be continuous
}

// updateReplayPosition updates position based on GPX replay data
func (s *GPSSimulator) updateReplayPosition() {
	if len(s.replayPoints) == 0 {
		return
	}

	// Defensive check for invalid replay speed
	if s.config.ReplaySpeed <= 0 {
		// Log error and use default speed to prevent panic
		fmt.Fprintf(os.Stderr, "Warning: Invalid replay speed %.2f, using default 1.0x\n", s.config.ReplaySpeed)
		s.config.ReplaySpeed = 1.0
	}

	now := time.Now()
	elapsedTime := now.Sub(s.replayStartTime)

	// Apply replay speed multiplier
	adjustedTime := time.Duration(float64(elapsedTime) * s.config.ReplaySpeed)

	// Check if timestamps are sequential for time-based progression
	useTimestamps := s.hasSequentialTimestamps()

	if useTimestamps {
		// Time-based progression using GPX timestamps
		targetTime := s.replayPoints[0].Time.Add(adjustedTime)

		// Find the track point that should be active at this time
		newIndex := 0
		for i := 0; i < len(s.replayPoints); i++ {
			if targetTime.After(s.replayPoints[i].Time) || targetTime.Equal(s.replayPoints[i].Time) {
				newIndex = i
			} else {
				break
			}
		}

		// If target time is past the last timestamp, we've completed the replay
		if targetTime.After(s.replayPoints[len(s.replayPoints)-1].Time) {
			newIndex = len(s.replayPoints) // This will trigger completion check
		}

		s.replayIndex = newIndex
	} else {
		// Index-based progression when timestamps are not sequential
		// Progress through points at a steady rate (1 point per second at 1x speed)
		pointInterval := time.Duration(float64(time.Second) / s.config.ReplaySpeed)
		pointsSinceStart := int(elapsedTime / pointInterval)

		if s.config.ReplayLoop {
			s.replayIndex = pointsSinceStart % len(s.replayPoints)
		} else {
			s.replayIndex = pointsSinceStart
		}
	}

	// If we've reached the end, handle completion/looping
	if s.replayIndex >= len(s.replayPoints) {
		s.replayCompleted = true
		if s.config.ReplayLoop {
			// Loop back to start if looping is enabled
			s.replayIndex = 0
			s.replayStartTime = now
		}
		return
	}

	// Update current position from track point
	currentPoint := s.replayPoints[s.replayIndex]
	s.currentLat = currentPoint.Lat
	s.currentLon = currentPoint.Lon
	s.currentAlt = currentPoint.Elevation

	// Calculate speed and course from next point if available
	if s.replayIndex < len(s.replayPoints)-1 {
		nextPoint := s.replayPoints[s.replayIndex+1]

		// Calculate distance and time between points
		distance := s.calculateDistance(s.currentLat, s.currentLon, nextPoint.Lat, nextPoint.Lon)

		var timeDiff float64
		if useTimestamps {
			timeDiff = nextPoint.Time.Sub(currentPoint.Time).Seconds()
		} else {
			// Use a fixed time interval for non-sequential timestamps
			timeDiff = 1.0 // 1 second between points
		}

		if timeDiff > 0 {
			// Convert m/s to knots (1 m/s = 1.94384 knots)
			s.currentSpeed = (distance / timeDiff) * 1.94384

			// Calculate course (bearing) to next point
			s.currentCourse = s.calculateBearing(s.currentLat, s.currentLon, nextPoint.Lat, nextPoint.Lon)
		}
	}
}

// calculateBearing calculates the bearing from point 1 to point 2
func (s *GPSSimulator) calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLonRad := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(deltaLonRad) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLonRad)

	bearing := math.Atan2(y, x) * 180 / math.Pi

	// Normalize to 0-359 degrees
	if bearing < 0 {
		bearing += 360
	}

	return bearing
}

// calculateDistance calculates the distance between two points using the Haversine formula
// This is the primary implementation used by other distance calculation methods
func (s *GPSSimulator) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth's radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
