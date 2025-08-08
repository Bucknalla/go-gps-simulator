package main

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"
)

// Helper function to create a test config
func createTestConfig() Config {
	return Config{
		Latitude:       37.7749,
		Longitude:      -122.4194,
		Radius:         100.0,
		Altitude:       45.0,
		Jitter:         0.5,
		AltitudeJitter: 0.1,
		Satellites:     8,
		TimeToLock:     30 * time.Second,
		OutputRate:     1 * time.Second,
	}
}

func TestNewGPSSimulator(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}

	sim := NewGPSSimulator(config, buffer)

	// Test that simulator is properly initialized
	if sim == nil {
		t.Fatal("NewGPSSimulator should not return nil")
	}

	// Test config assignment
	if sim.config.Latitude != config.Latitude {
		t.Errorf("Expected latitude %f, got %f", config.Latitude, sim.config.Latitude)
	}
	if sim.config.Longitude != config.Longitude {
		t.Errorf("Expected longitude %f, got %f", config.Longitude, sim.config.Longitude)
	}
	if sim.config.Radius != config.Radius {
		t.Errorf("Expected radius %f, got %f", config.Radius, sim.config.Radius)
	}
	if sim.config.Satellites != config.Satellites {
		t.Errorf("Expected satellites %d, got %d", config.Satellites, sim.config.Satellites)
	}

	// Test initial position
	if sim.currentLat != config.Latitude {
		t.Errorf("Expected initial latitude %f, got %f", config.Latitude, sim.currentLat)
	}
	if sim.currentLon != config.Longitude {
		t.Errorf("Expected initial longitude %f, got %f", config.Longitude, sim.currentLon)
	}
	if sim.currentAlt != config.Altitude {
		t.Errorf("Expected initial altitude %f, got %f", config.Altitude, sim.currentAlt)
	}

	// Test initial lock state
	if sim.isLocked {
		t.Error("GPS should not be locked initially")
	}

	// Test satellites initialization
	if len(sim.satellites) != config.Satellites {
		t.Errorf("Expected %d satellites, got %d", config.Satellites, len(sim.satellites))
	}

	// Test writer assignment
	if sim.nmeaWriter != buffer {
		t.Error("NMEA writer should be assigned correctly")
	}

	// Test timing setup
	if sim.startTime.IsZero() {
		t.Error("Start time should be set")
	}
	if sim.lockTime.IsZero() {
		t.Error("Lock time should be set")
	}
}

func TestInitializeSatellites(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	// Test satellite count
	if len(sim.satellites) != config.Satellites {
		t.Errorf("Expected %d satellites, got %d", config.Satellites, len(sim.satellites))
	}

	// Test satellite properties
	for i, sat := range sim.satellites {
		// Test ID assignment
		expectedID := i + 1
		if sat.ID != expectedID {
			t.Errorf("Satellite %d should have ID %d, got %d", i, expectedID, sat.ID)
		}

		// Test elevation range (10-80 degrees)
		if sat.Elevation < 10 || sat.Elevation > 80 {
			t.Errorf("Satellite %d elevation %d should be between 10-80 degrees", i, sat.Elevation)
		}

		// Test azimuth range (0-359 degrees)
		if sat.Azimuth < 0 || sat.Azimuth >= 360 {
			t.Errorf("Satellite %d azimuth %d should be between 0-359 degrees", i, sat.Azimuth)
		}

		// Test SNR range (20-50 dB)
		if sat.SNR < 20 || sat.SNR > 50 {
			t.Errorf("Satellite %d SNR %d should be between 20-50 dB", i, sat.SNR)
		}
	}
}

func TestAltitudeSimulation(t *testing.T) {
	config := createTestConfig()
	config.Altitude = 1000.0
	config.AltitudeJitter = 0.5
	buffer := &bytes.Buffer{}

	sim := NewGPSSimulator(config, buffer)

	// Test initial altitude
	if sim.currentAlt != 1000.0 {
		t.Errorf("Expected initial altitude 1000.0, got %f", sim.currentAlt)
	}

	// Force GPS lock to enable altitude updates
	sim.isLocked = true

	// Capture initial altitude
	initialAltitude := sim.currentAlt

	// Update altitude multiple times and check for variation
	sim.updateAltitude()
	sim.updateAltitude()
	sim.updateAltitude()

	// With jitter 0.5, altitude should change
	if sim.currentAlt == initialAltitude {
		t.Error("Expected altitude to change with jitter > 0")
	}

	// Test altitude bounds - should stay within reasonable range
	if sim.currentAlt < 900.0 || sim.currentAlt > 1500.0 {
		t.Errorf("Altitude %f is outside expected bounds (900-1500m)", sim.currentAlt)
	}
}

func TestAltitudeStability(t *testing.T) {
	config := createTestConfig()
	config.Altitude = 500.0
	config.AltitudeJitter = 0.0 // No jitter
	buffer := &bytes.Buffer{}

	sim := NewGPSSimulator(config, buffer)
	sim.isLocked = true

	initialAltitude := sim.currentAlt

	// Update altitude multiple times
	for i := 0; i < 10; i++ {
		sim.updateAltitude()
	}

	// With zero jitter, altitude should remain stable
	if sim.currentAlt != initialAltitude {
		t.Errorf("Expected altitude to remain stable at %f, got %f", initialAltitude, sim.currentAlt)
	}
}

func TestAltitudeInNMEA(t *testing.T) {
	config := createTestConfig()
	config.Altitude = 2500.0
	buffer := &bytes.Buffer{}

	sim := NewGPSSimulator(config, buffer)
	sim.isLocked = true

	// Generate NMEA output
	sim.outputNMEA()

	output := buffer.String()

	// Check that GGA sentence contains the altitude
	if !strings.Contains(output, "2500.0,M") {
		t.Errorf("Expected NMEA output to contain altitude '2500.0,M', got: %s", output)
	}

	// Update altitude and check again
	buffer.Reset()
	sim.currentAlt = 3000.5
	sim.outputNMEA()

	output = buffer.String()
	if !strings.Contains(output, "3000.5,M") {
		t.Errorf("Expected NMEA output to contain updated altitude '3000.5,M', got: %s", output)
	}
}

func TestDistanceFromCenter(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	tests := []struct {
		name      string
		lat       float64
		lon       float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "Same position",
			lat:       config.Latitude,
			lon:       config.Longitude,
			expected:  0.0,
			tolerance: 0.1,
		},
		{
			name:      "One degree north",
			lat:       config.Latitude + 1.0,
			lon:       config.Longitude,
			expected:  111000.0, // Approximately 111km per degree
			tolerance: 5000.0,   // 5km tolerance
		},
		{
			name:      "One degree east",
			lat:       config.Latitude,
			lon:       config.Longitude + 1.0,
			expected:  85000.0, // Varies by latitude
			tolerance: 10000.0, // 10km tolerance
		},
		{
			name:      "Small distance",
			lat:       config.Latitude + 0.001, // ~111 meters north
			lon:       config.Longitude,
			expected:  111.0,
			tolerance: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := sim.distanceFromCenter(tt.lat, tt.lon)
			if math.Abs(distance-tt.expected) > tt.tolerance {
				t.Errorf("Distance from center: expected ~%f, got %f (tolerance: %f)",
					tt.expected, distance, tt.tolerance)
			}
		})
	}
}

func TestUpdatePosition(t *testing.T) {
	tests := []struct {
		name   string
		jitter float64
	}{
		{"Low jitter", 0.05},
		{"Medium jitter", 0.5},
		{"High jitter", 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig()
			config.Jitter = tt.jitter
			buffer := &bytes.Buffer{}
			sim := NewGPSSimulator(config, buffer)
			sim.isLocked = true

			// Store initial position
			initialLat := sim.currentLat
			initialLon := sim.currentLon

			// Update position multiple times
			for i := 0; i < 10; i++ {
				sim.updatePosition()

				// Check that position is within the specified radius
				distance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
				if distance > config.Radius {
					t.Errorf("Position update %d: distance %f exceeds radius %f",
						i, distance, config.Radius)
				}

				// For very low jitter, movement should be more predictable
				if tt.jitter < 0.1 {
					// Position should change but not dramatically
					latChange := math.Abs(sim.currentLat - initialLat)
					lonChange := math.Abs(sim.currentLon - initialLon)
					if latChange > 0.01 || lonChange > 0.01 {
						t.Errorf("Low jitter should produce small movements, got lat change: %f, lon change: %f",
							latChange, lonChange)
					}
				}
			}
		})
	}
}

func TestUpdatePositionBoundaryConstraints(t *testing.T) {
	config := createTestConfig()
	config.Radius = 50.0 // Small radius for testing
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)
	sim.isLocked = true

	// Move to edge of radius
	sim.currentLat = config.Latitude + 0.0004  // ~44 meters north
	sim.currentLon = config.Longitude + 0.0004 // Close to radius

	initialDistance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)

	// Track positions that exceed radius
	exceedCount := 0
	maxExceedance := 0.0

	// Update position multiple times
	for i := 0; i < 20; i++ {
		sim.updatePosition()
		distance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)

		// Allow small overshoots due to the pullback mechanism
		// The algorithm may temporarily exceed the radius before correction
		if distance > config.Radius {
			exceedCount++
			exceedance := distance - config.Radius
			if exceedance > maxExceedance {
				maxExceedance = exceedance
			}

			// Should not exceed radius by more than a reasonable amount (10% tolerance)
			tolerance := config.Radius * 0.1 // 10% tolerance
			if exceedance > tolerance {
				t.Errorf("Update %d: position too far outside radius. Distance: %f, Radius: %f, Exceedance: %f, Max allowed: %f",
					i, distance, config.Radius, exceedance, tolerance)
			}
		}
	}

	// Most updates should stay within radius
	if float64(exceedCount)/20.0 > 0.5 {
		t.Errorf("Too many position updates exceeded radius: %d out of 20 (max exceedance: %f)",
			exceedCount, maxExceedance)
	}

	// Should have moved closer to center due to pullback mechanism over time
	finalDistance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
	if finalDistance >= initialDistance {
		// Allow some tolerance since pullback is probabilistic
		tolerance := 1.0 // 1 meter tolerance
		if finalDistance-initialDistance > tolerance {
			t.Errorf("Expected pullback to reduce distance. Initial: %f, Final: %f, Difference: %f",
				initialDistance, finalDistance, finalDistance-initialDistance)
		}
	}
}

func TestUpdateSatellites(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	// Store initial satellite states
	initialSats := make([]Satellite, len(sim.satellites))
	copy(initialSats, sim.satellites)

	// Update satellites multiple times
	for i := 0; i < 10; i++ {
		sim.updateSatellites()

		// Check that all satellites remain within valid bounds
		for j, sat := range sim.satellites {
			// Check elevation bounds
			if sat.Elevation < 5 || sat.Elevation > 85 {
				t.Errorf("Update %d: Satellite %d elevation %d out of bounds (5-85)",
					i, j, sat.Elevation)
			}

			// Check azimuth bounds
			if sat.Azimuth < 0 || sat.Azimuth >= 360 {
				t.Errorf("Update %d: Satellite %d azimuth %d out of bounds (0-359)",
					i, j, sat.Azimuth)
			}

			// Check SNR bounds
			if sat.SNR < 15 || sat.SNR > 55 {
				t.Errorf("Update %d: Satellite %d SNR %d out of bounds (15-55)",
					i, j, sat.SNR)
			}

			// ID should remain unchanged
			if sat.ID != initialSats[j].ID {
				t.Errorf("Update %d: Satellite %d ID changed from %d to %d",
					i, j, initialSats[j].ID, sat.ID)
			}
		}
	}

	// Check that at least some satellites have changed
	changed := false
	for i, sat := range sim.satellites {
		if sat.Elevation != initialSats[i].Elevation ||
			sat.Azimuth != initialSats[i].Azimuth ||
			sat.SNR != initialSats[i].SNR {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("Expected at least some satellite parameters to change after updates")
	}
}

func TestUpdateSatellitesBoundaryConditions(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	// Test elevation boundary conditions
	sim.satellites[0].Elevation = 4  // Below minimum
	sim.satellites[1].Elevation = 86 // Above maximum

	// Test SNR boundary conditions
	sim.satellites[2].SNR = 14 // Below minimum
	sim.satellites[3].SNR = 56 // Above maximum

	sim.updateSatellites()

	// Check that boundaries are enforced
	if sim.satellites[0].Elevation < 5 {
		t.Errorf("Expected elevation to be at least 5, got %d", sim.satellites[0].Elevation)
	}
	if sim.satellites[1].Elevation > 85 {
		t.Errorf("Expected elevation to be at most 85, got %d", sim.satellites[1].Elevation)
	}
	if sim.satellites[2].SNR < 15 {
		t.Errorf("Expected SNR to be at least 15, got %d", sim.satellites[2].SNR)
	}
	if sim.satellites[3].SNR > 55 {
		t.Errorf("Expected SNR to be at most 55, got %d", sim.satellites[3].SNR)
	}
}

func TestUpdate(t *testing.T) {
	config := createTestConfig()
	config.TimeToLock = 100 * time.Millisecond // Short lock time for testing
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	// Initially should not be locked
	if sim.isLocked {
		t.Error("GPS should not be locked initially")
	}

	// Update before lock time
	sim.update()
	if sim.isLocked {
		t.Error("GPS should not be locked before lock time")
	}

	// Wait for lock time and update
	time.Sleep(config.TimeToLock + 10*time.Millisecond)
	sim.update()
	if !sim.isLocked {
		t.Error("GPS should be locked after lock time")
	}

	// Store initial position
	initialLat := sim.currentLat
	initialLon := sim.currentLon

	// Update again - position should change now that it's locked
	sim.update()
	if sim.currentLat == initialLat && sim.currentLon == initialLon {
		// Position might not change every update due to randomness, so this is not a hard failure
		t.Logf("Position did not change after GPS lock (this may be normal)")
	}
}

func TestOutputNMEA(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	// Test output when not locked
	buffer.Reset()
	sim.outputNMEA()
	output := buffer.String()

	// Should contain no-fix sentences
	if !strings.Contains(output, "$GPGGA,") {
		t.Error("Output should contain GGA sentence when not locked")
	}
	if !strings.Contains(output, "$GPRMC,") {
		t.Error("Output should contain RMC sentence when not locked")
	}
	// Should not contain GSA or GSV when not locked
	if strings.Contains(output, "$GPGSA,") {
		t.Error("Output should not contain GSA sentence when not locked")
	}
	if strings.Contains(output, "$GPGSV,") {
		t.Error("Output should not contain GSV sentence when not locked")
	}

	// Test output when locked
	sim.isLocked = true
	buffer.Reset()
	sim.outputNMEA()
	output = buffer.String()

	// Should contain all sentence types
	expectedSentences := []string{"$GPGGA,", "$GPRMC,", "$GPGSA,", "$GPGSV,"}
	for _, sentence := range expectedSentences {
		if !strings.Contains(output, sentence) {
			t.Errorf("Output should contain %s sentence when locked", sentence)
		}
	}

	// Count GSV sentences (should be based on number of satellites)
	gsvCount := strings.Count(output, "$GPGSV,")
	expectedGSVCount := (len(sim.satellites) + 3) / 4 // Round up to nearest 4
	if gsvCount != expectedGSVCount {
		t.Errorf("Expected %d GSV sentences, got %d", expectedGSVCount, gsvCount)
	}

	// Verify all sentences end with \r\n
	sentences := strings.Split(output, "\r\n")
	for i, sentence := range sentences {
		if sentence == "" {
			continue // Skip empty strings from split
		}
		if !strings.HasPrefix(sentence, "$GP") {
			t.Errorf("Sentence %d should start with $GP: %s", i, sentence)
		}
		if !strings.Contains(sentence, "*") {
			t.Errorf("Sentence %d should contain checksum separator: %s", i, sentence)
		}
	}
}

func TestOutputNMEAChecksums(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)
	sim.isLocked = true

	buffer.Reset()
	sim.outputNMEA()
	output := buffer.String()

	// Split into individual sentences
	lines := strings.Split(output, "\r\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Verify checksum
		parts := strings.Split(line, "*")
		if len(parts) != 2 {
			t.Errorf("Sentence should have exactly one checksum separator: %s", line)
			continue
		}

		nmeaPart := parts[0]
		checksumPart := parts[1]
		expectedChecksum := calculateChecksum(nmeaPart)

		if checksumPart != expectedChecksum {
			t.Errorf("Invalid checksum for sentence: %s. Expected %s, got %s",
				line, expectedChecksum, checksumPart)
		}
	}
}

func TestSatelliteStruct(t *testing.T) {
	sat := Satellite{
		ID:        15,
		Elevation: 45,
		Azimuth:   180,
		SNR:       35,
	}

	if sat.ID != 15 {
		t.Errorf("Expected ID 15, got %d", sat.ID)
	}
	if sat.Elevation != 45 {
		t.Errorf("Expected elevation 45, got %d", sat.Elevation)
	}
	if sat.Azimuth != 180 {
		t.Errorf("Expected azimuth 180, got %d", sat.Azimuth)
	}
	if sat.SNR != 35 {
		t.Errorf("Expected SNR 35, got %d", sat.SNR)
	}
}

// Benchmark tests for performance
func BenchmarkUpdatePosition(b *testing.B) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)
	sim.isLocked = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.updatePosition()
	}
}

func BenchmarkUpdateSatellites(b *testing.B) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.updateSatellites()
	}
}

func BenchmarkDistanceFromCenter(b *testing.B) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	lat := config.Latitude + 0.001
	lon := config.Longitude + 0.001

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.distanceFromCenter(lat, lon)
	}
}

// Test Run function with timeout to avoid infinite loop
func TestRun(t *testing.T) {
	config := createTestConfig()
	config.OutputRate = 10 * time.Millisecond // Very fast for testing
	config.TimeToLock = 5 * time.Millisecond  // Quick lock
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)

	// Run for a short duration
	done := make(chan bool)
	go func() {
		time.Sleep(50 * time.Millisecond) // Run for 50ms
		done <- true
	}()

	go func() {
		sim.Run()
	}()

	<-done

	// Check that some NMEA output was generated
	output := buffer.String()
	if len(output) == 0 {
		t.Error("Expected NMEA output from Run function")
	}

	// Should contain GPS sentences
	if !strings.Contains(output, "$GP") {
		t.Error("Expected GPS sentences in output")
	}
}

// Test GSV with edge cases to improve coverage
func TestGenerateGSVEdgeCases(t *testing.T) {
	// Test with 0 satellites
	config := createTestConfig()
	config.Satellites = 0
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)
	sim.satellites = []Satellite{} // Empty satellites

	result := sim.generateGSV()
	if len(result) != 0 {
		t.Errorf("Expected 0 GSV sentences for 0 satellites, got %d", len(result))
	}

	// Test with 1 satellite (to test padding logic)
	sim.satellites = []Satellite{
		{ID: 1, Elevation: 45, Azimuth: 90, SNR: 35},
	}

	result = sim.generateGSV()
	if len(result) != 1 {
		t.Errorf("Expected 1 GSV sentence for 1 satellite, got %d", len(result))
	}

	// Check that padding is applied (should have empty fields)
	sentence := result[0]
	if !strings.Contains(sentence, ",,,,") {
		t.Error("Expected padding fields in GSV sentence for single satellite")
	}

	// Test with 3 satellites (to test partial padding)
	sim.satellites = []Satellite{
		{ID: 1, Elevation: 45, Azimuth: 90, SNR: 35},
		{ID: 2, Elevation: 60, Azimuth: 180, SNR: 40},
		{ID: 3, Elevation: 30, Azimuth: 270, SNR: 25},
	}

	result = sim.generateGSV()
	if len(result) != 1 {
		t.Errorf("Expected 1 GSV sentence for 3 satellites, got %d", len(result))
	}

	// Should have one empty field set (4th satellite position)
	sentence = result[0]
	if !strings.Contains(sentence, ",,,,") {
		t.Error("Expected one empty field set in GSV sentence for 3 satellites")
	}

	// Test with exactly 4 satellites (no padding needed)
	sim.satellites = []Satellite{
		{ID: 1, Elevation: 45, Azimuth: 90, SNR: 35},
		{ID: 2, Elevation: 60, Azimuth: 180, SNR: 40},
		{ID: 3, Elevation: 30, Azimuth: 270, SNR: 25},
		{ID: 4, Elevation: 75, Azimuth: 0, SNR: 45},
	}

	result = sim.generateGSV()
	if len(result) != 1 {
		t.Errorf("Expected 1 GSV sentence for 4 satellites, got %d", len(result))
	}

	// Should not have padding for 4 satellites
	sentence = result[0]
	if strings.Contains(sentence, ",,,,") {
		t.Error("Should not have padding fields in GSV sentence for 4 satellites")
	}
}

// Test RMC edge cases to improve coverage
func TestGenerateRMCEdgeCases(t *testing.T) {
	sim := createTestSimulator()

	// Test with negative coordinates to cover hemisphere logic
	sim.currentLat = -33.8688 // Sydney, Australia (Southern hemisphere)
	sim.currentLon = 151.2093 // Sydney, Australia (Eastern hemisphere)

	testTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC) // End of year
	result := sim.generateRMC(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPRMC,") {
		t.Errorf("generateRMC should start with '$GPRMC,', got: %s", result)
	}

	// Check that it contains expected time and date
	if !strings.Contains(result, "235959") {
		t.Errorf("generateRMC should contain time '235959', got: %s", result)
	}
	if !strings.Contains(result, "311224") {
		t.Errorf("generateRMC should contain date '311224', got: %s", result)
	}

	// Check hemisphere indicators
	parts := strings.Split(result, ",")
	if len(parts) > 4 && parts[4] != "S" {
		t.Errorf("Expected Southern hemisphere 'S', got: %s", parts[4])
	}
	if len(parts) > 6 && parts[6] != "E" {
		t.Errorf("Expected Eastern hemisphere 'E', got: %s", parts[6])
	}
}

func BenchmarkOutputNMEA(b *testing.B) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim := NewGPSSimulator(config, buffer)
	sim.isLocked = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Reset()
		sim.outputNMEA()
	}
}
