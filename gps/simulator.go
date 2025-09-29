package gps

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Simulator represents the GPS simulator
type Simulator struct {
	mu             sync.RWMutex
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
	replayCompleted bool
	// Control fields
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
	callbacks []func(NMEAData)
}

// NewSimulator creates a new GPS simulator instance
func NewSimulator(config Config) (*Simulator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	sim := &Simulator{
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
		replayIndex:     0,
		replayStartTime: now,
		replayCompleted: false,
		running:         false,
		callbacks:       make([]func(NMEAData), 0),
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
	if config.GPXEnabled && config.GPXFile != "" {
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

// SetNMEAWriter sets the writer for NMEA output
func (s *Simulator) SetNMEAWriter(writer io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nmeaWriter = writer
}

// AddCallback adds a callback function that will be called with each NMEA data update
func (s *Simulator) AddCallback(callback func(NMEAData)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, callback)
}

// Start starts the GPS simulation
func (s *Simulator) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrSimulatorAlreadyRunning
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.ticker = time.NewTicker(s.config.OutputRate)
	s.running = true
	s.startTime = time.Now()
	s.lockTime = s.startTime.Add(s.config.TimeToLock)

	go s.run()
	return nil
}

// Stop stops the GPS simulation
func (s *Simulator) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrSimulatorNotRunning
	}

	s.cancel()
	s.ticker.Stop()
	s.running = false

	// Close GPX writer if enabled
	if s.gpxWriter != nil {
		s.gpxWriter.Close()
	}

	return nil
}

// IsRunning returns whether the simulator is currently running
func (s *Simulator) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus returns the current simulator status
func (s *Simulator) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var elapsedTime time.Duration
	if s.running {
		elapsedTime = time.Since(s.startTime)
	}

	return Status{
		Running:     s.running,
		StartTime:   s.startTime,
		ElapsedTime: elapsedTime,
		Position: Position{
			Latitude:   s.currentLat,
			Longitude:  s.currentLon,
			Altitude:   s.currentAlt,
			Speed:      s.currentSpeed,
			Course:     s.currentCourse,
			IsLocked:   s.isLocked,
			Satellites: s.satellites,
			Timestamp:  time.Now(),
		},
		Config:          s.config,
		ReplayIndex:     s.replayIndex,
		ReplayTotal:     len(s.replayPoints),
		ReplayCompleted: s.replayCompleted,
	}
}

// UpdateConfig updates the simulator configuration (can be called while running)
func (s *Simulator) UpdateConfig(newConfig Config) error {
	if err := newConfig.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update configuration
	oldRate := s.config.OutputRate
	s.config = newConfig

	// If output rate changed and simulator is running, restart ticker
	if s.running && oldRate != newConfig.OutputRate {
		s.ticker.Stop()
		s.ticker = time.NewTicker(newConfig.OutputRate)
	}

	return nil
}

// run is the main simulation loop
func (s *Simulator) run() {
	defer func() {
		if s.gpxWriter != nil {
			s.gpxWriter.Close()
		}
	}()

	var durationTimer *time.Timer
	var durationChan <-chan time.Time
	if s.config.Duration > 0 {
		durationTimer = time.NewTimer(s.config.Duration)
		durationChan = durationTimer.C
		defer durationTimer.Stop()
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			s.update()
			s.outputNMEA()
			s.updateGPX()

			// Check if replay is completed and looping is disabled
			if s.config.ReplayFile != "" && !s.config.ReplayLoop && s.replayCompleted {
				s.Stop()
				return
			}
		case <-durationChan:
			s.Stop()
			return
		}
	}
}

// initializeSatellites initializes the satellite array
func (s *Simulator) initializeSatellites() {
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

// update updates the GPS position and satellite information
func (s *Simulator) update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check if GPS should be locked
	if !s.isLocked && now.After(s.lockTime) {
		s.isLocked = true
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

// updateSpeedAndCourse applies jitter to speed and course
func (s *Simulator) updateSpeedAndCourse() {
	var speedVariation, courseVariation float64

	if s.config.Jitter == 0.0 {
		speedVariation = 0.0
		courseVariation = 0.0
	} else if s.config.Jitter < 0.2 {
		speedVariation = 0.05
		courseVariation = 2.0
	} else if s.config.Jitter < 0.7 {
		speedVariation = 0.10 + (s.config.Jitter-0.2)*0.40
		courseVariation = 5.0 + (s.config.Jitter-0.2)*20.0
	} else {
		speedVariation = 0.30 + (s.config.Jitter-0.7)*0.67
		courseVariation = 15.0 + (s.config.Jitter-0.7)*50.0
	}

	// Apply speed variation
	speedDelta := (rand.Float64() - 0.5) * 2 * s.config.Speed * speedVariation
	s.currentSpeed = s.config.Speed + speedDelta
	if s.currentSpeed < 0 {
		s.currentSpeed = 0
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

// updatePosition updates the GPS position based on speed and course
func (s *Simulator) updatePosition() {
	now := time.Now()
	deltaTime := now.Sub(s.lastUpdateTime).Seconds()
	s.lastUpdateTime = now

	if deltaTime <= 0 {
		return
	}

	// Convert speed from knots to meters per second
	speedMPS := s.currentSpeed * 0.514444

	// Calculate distance traveled
	distanceMeters := speedMPS * deltaTime

	// Calculate new position using proper spherical Earth calculations
	newLat, newLon := s.calculateNewPosition(s.currentLat, s.currentLon, distanceMeters, s.currentCourse)

	// Apply radius constraint
	if s.distanceFromCenter(newLat, newLon) > s.config.Radius {
		if s.config.Jitter > 0.5 {
			// High jitter: bounce off boundaries with random course change
			randomCourseChange := (rand.Float64() - 0.5) * 60.0
			s.currentCourse += randomCourseChange

			// Normalize course
			for s.currentCourse < 0 {
				s.currentCourse += 360
			}
			for s.currentCourse >= 360 {
				s.currentCourse -= 360
			}

			// Recalculate with new course using spherical calculation
			newLat, newLon = s.calculateNewPosition(s.currentLat, s.currentLon, distanceMeters, s.currentCourse)
		} else {
			// Low jitter: constrain to radius boundary
			// Calculate bearing from center to new position
			bearing := s.calculateBearing(s.config.Latitude, s.config.Longitude, newLat, newLon)

			// Place position exactly at radius boundary
			newLat, newLon = s.calculateNewPosition(s.config.Latitude, s.config.Longitude, s.config.Radius, bearing)
		}
	}

	s.currentLat = newLat
	s.currentLon = newLon
}

// updateAltitude applies altitude jitter
func (s *Simulator) updateAltitude() {
	if s.config.AltitudeJitter > 0 {
		maxChange := 1.0 + (s.config.AltitudeJitter * 20.0)
		change := (rand.Float64() - 0.5) * 2 * maxChange

		newAltitude := s.currentAlt + change

		minAltitude := s.config.Altitude - 100.0
		maxAltitude := s.config.Altitude + 500.0

		if minAltitude < -50.0 {
			minAltitude = -50.0
		}

		if newAltitude < minAltitude {
			newAltitude = minAltitude
		} else if newAltitude > maxAltitude {
			newAltitude = maxAltitude
		}

		s.currentAlt = newAltitude
	}
}

// updateSatellites simulates satellite movement
func (s *Simulator) updateSatellites() {
	for i := range s.satellites {
		s.satellites[i].Elevation += rand.Intn(3) - 1
		s.satellites[i].Azimuth = (s.satellites[i].Azimuth + rand.Intn(3) - 1 + 360) % 360

		if s.satellites[i].Elevation < 5 {
			s.satellites[i].Elevation = 5
		}
		if s.satellites[i].Elevation > 85 {
			s.satellites[i].Elevation = 85
		}

		s.satellites[i].SNR += rand.Intn(6) - 3
		if s.satellites[i].SNR < 15 {
			s.satellites[i].SNR = 15
		}
		if s.satellites[i].SNR > 55 {
			s.satellites[i].SNR = 55
		}
	}
}

// distanceFromCenter calculates distance from the configured center point
func (s *Simulator) distanceFromCenter(lat, lon float64) float64 {
	return s.calculateDistance(s.config.Latitude, s.config.Longitude, lat, lon)
}

// calculateDistance calculates distance between two points using Haversine formula
func (s *Simulator) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

// calculateNewPosition calculates a new lat/lon position given starting point, distance, and bearing
// Uses the spherical Earth model for accurate calculations at all speeds and distances
func (s *Simulator) calculateNewPosition(lat, lon, distance, bearing float64) (newLat, newLon float64) {
	const R = 6371000.0 // Earth's radius in meters

	// Convert to radians
	latRad := lat * math.Pi / 180.0
	lonRad := lon * math.Pi / 180.0
	bearingRad := bearing * math.Pi / 180.0

	// Calculate angular distance
	angularDistance := distance / R

	// Calculate new latitude
	newLatRad := math.Asin(math.Sin(latRad)*math.Cos(angularDistance) +
		math.Cos(latRad)*math.Sin(angularDistance)*math.Cos(bearingRad))

	// Calculate new longitude
	newLonRad := lonRad + math.Atan2(
		math.Sin(bearingRad)*math.Sin(angularDistance)*math.Cos(latRad),
		math.Cos(angularDistance)-math.Sin(latRad)*math.Sin(newLatRad))

	// Convert back to degrees
	newLat = newLatRad * 180.0 / math.Pi
	newLon = newLonRad * 180.0 / math.Pi

	// Normalize longitude to -180 to +180 range
	for newLon > 180 {
		newLon -= 360
	}
	for newLon < -180 {
		newLon += 360
	}

	return newLat, newLon
}

// calculateBearing calculates the bearing from point 1 to point 2
func (s *Simulator) calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
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

// updateGPX adds current position to GPX track if enabled
func (s *Simulator) updateGPX() {
	if s.gpxWriter != nil && s.isLocked {
		s.gpxWriter.AddTrackPoint(s.currentLat, s.currentLon, s.currentAlt, time.Now())

		// Write to file periodically
		if s.gpxWriter.GetTrackPointCount()%10 == 0 {
			s.gpxWriter.WriteToFile()
		}
	}
}

// outputNMEA generates and outputs NMEA sentences
func (s *Simulator) outputNMEA() {
	timestamp := time.Now()
	var sentences []string

	if s.isLocked {
		sentences = append(sentences, s.generateGGA(timestamp))
		sentences = append(sentences, s.generateRMC(timestamp))
		sentences = append(sentences, s.generateGLL(timestamp))
		sentences = append(sentences, s.generateVTG())
		sentences = append(sentences, s.generateGSA())
		gsv := s.generateGSV()
		sentences = append(sentences, gsv...)
		sentences = append(sentences, s.generateZDA(timestamp))
	} else {
		sentences = append(sentences, s.generateNoFixGGA(timestamp))
		sentences = append(sentences, s.generateNoFixRMC(timestamp))
		sentences = append(sentences, s.generateNoFixGLL(timestamp))
		sentences = append(sentences, s.generateNoFixVTG())
	}

	// Output to writer if set
	if s.nmeaWriter != nil {
		for _, sentence := range sentences {
			fmt.Fprint(s.nmeaWriter, sentence)
		}
	}

	// Call callbacks with NMEA data
	nmeaData := NMEAData{
		Sentences: sentences,
		Position: Position{
			Latitude:   s.currentLat,
			Longitude:  s.currentLon,
			Altitude:   s.currentAlt,
			Speed:      s.currentSpeed,
			Course:     s.currentCourse,
			IsLocked:   s.isLocked,
			Satellites: s.satellites,
			Timestamp:  timestamp,
		},
		Timestamp: timestamp,
	}

	for _, callback := range s.callbacks {
		go callback(nmeaData) // Call async to avoid blocking
	}
}

// Implementation continues in the next part due to length...
