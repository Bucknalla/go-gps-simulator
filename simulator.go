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
	config     Config
	currentLat float64
	currentLon float64
	isLocked   bool
	lockTime   time.Time
	startTime  time.Time
	satellites []Satellite
	nmeaWriter io.Writer
}

type Satellite struct {
	ID        int
	Elevation int // degrees above horizon
	Azimuth   int // degrees from north
	SNR       int // signal-to-noise ratio
}

func NewGPSSimulator(config Config, nmeaWriter io.Writer) *GPSSimulator {
	sim := &GPSSimulator{
		config:     config,
		currentLat: config.Latitude,
		currentLon: config.Longitude,
		isLocked:   false,
		startTime:  time.Now(),
		lockTime:   time.Now().Add(config.TimeToLock),
		nmeaWriter: nmeaWriter,
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
		fmt.Fprintf(os.Stderr, "GPS LOCKED after %v\n", now.Sub(s.startTime))
	}

	// Update position if locked
	if s.isLocked {
		s.updatePosition()
	}

	// Update satellites
	s.updateSatellites()
}

func (s *GPSSimulator) updatePosition() {
	// Convert radius from meters to degrees (approximate)
	radiusInDegrees := s.config.Radius / 111000.0 // rough conversion

	// Calculate maximum step size based on jitter
	// Low jitter = small, smooth steps; High jitter = large, random jumps
	maxStepRatio := 0.01 + (s.config.Jitter * 0.3) // 1% to 31% of radius per step
	maxStep := radiusInDegrees * maxStepRatio

	var newLat, newLon float64

	if s.config.Jitter < 0.1 {
		// Very low jitter: smooth, predictable movement in small circles
		time := float64(time.Now().UnixNano()) / 1e9 / 60.0 // slow circular motion
		angle := time * 2 * math.Pi
		distance := maxStep * 2 // small consistent radius

		deltaLat := distance * math.Cos(angle)
		deltaLon := distance * math.Sin(angle) / math.Cos(s.config.Latitude*math.Pi/180)

		newLat = s.config.Latitude + deltaLat
		newLon = s.config.Longitude + deltaLon
	} else {
		// Blend smooth movement with random jumps based on jitter
		smoothWeight := 1.0 - s.config.Jitter
		randomWeight := s.config.Jitter

		// Smooth component: gradual drift from current position
		smoothAngle := rand.Float64() * 2 * math.Pi
		smoothDistance := rand.Float64() * maxStep * 0.3 // smaller smooth steps

		smoothDeltaLat := smoothDistance * math.Cos(smoothAngle)
		smoothDeltaLon := smoothDistance * math.Sin(smoothAngle) / math.Cos(s.currentLat*math.Pi/180)

		// Random component: larger jumps
		randomAngle := rand.Float64() * 2 * math.Pi
		randomDistance := rand.Float64() * maxStep

		randomDeltaLat := randomDistance * math.Cos(randomAngle)
		randomDeltaLon := randomDistance * math.Sin(randomAngle) / math.Cos(s.config.Latitude*math.Pi/180)

		// Combine smooth and random movement
		totalDeltaLat := (smoothDeltaLat * smoothWeight) + (randomDeltaLat * randomWeight)
		totalDeltaLon := (smoothDeltaLon * smoothWeight) + (randomDeltaLon * randomWeight)

		newLat = s.currentLat + totalDeltaLat
		newLon = s.currentLon + totalDeltaLon
	}

	// Ensure new position is within the specified radius from center
	if s.distanceFromCenter(newLat, newLon) <= s.config.Radius {
		s.currentLat = newLat
		s.currentLon = newLon
	} else {
		// If outside radius, move towards center with some randomness
		centerLat := s.config.Latitude
		centerLon := s.config.Longitude

		// Move partway back towards center
		pullback := 0.1 + (rand.Float64() * 0.2) // 10-30% pullback
		s.currentLat += (centerLat - s.currentLat) * pullback
		s.currentLon += (centerLon - s.currentLon) * pullback
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
