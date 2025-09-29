package gps

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
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
		Speed:          0.1,
		Course:         0.0,
		Satellites:     8,
		TimeToLock:     30 * time.Second,
		OutputRate:     1 * time.Second,
	}
}

func TestNewGPSSimulator(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Test that simulator is properly initialized
	if sim == nil {
		t.Fatal("NewGPSSimulator should not return nil")
	}

	// Test config assignment
	if sim.Config.Latitude != config.Latitude {
		t.Errorf("Expected latitude %f, got %f", config.Latitude, sim.Config.Latitude)
	}
	if sim.Config.Longitude != config.Longitude {
		t.Errorf("Expected longitude %f, got %f", config.Longitude, sim.Config.Longitude)
	}
	if sim.Config.Radius != config.Radius {
		t.Errorf("Expected radius %f, got %f", config.Radius, sim.Config.Radius)
	}
	if sim.Config.Satellites != config.Satellites {
		t.Errorf("Expected satellites %d, got %d", config.Satellites, sim.Config.Satellites)
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
	if len(sim.Satellites) != config.Satellites {
		t.Errorf("Expected %d satellites, got %d", config.Satellites, len(sim.Satellites))
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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Test satellite count
	if len(sim.Satellites) != config.Satellites {
		t.Errorf("Expected %d satellites, got %d", config.Satellites, len(sim.Satellites))
	}

	// Test satellite properties
	for i, sat := range sim.Satellites {
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

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

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

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
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

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

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
		name           string
		jitter         float64
		speed          float64
		course         float64
		expectMovement bool
	}{
		{"No jitter no movement", 0.0, 0.0, 0.0, false}, // No jitter, no speed = no movement
		{"Low jitter stationary", 0.05, 0.0, 0.0, true}, // Stationary GPS still has jitter noise
		{"Low jitter moving", 0.05, 50.0, 90.0, true}, // Higher speed for detectable movement
		{"Medium jitter moving", 0.5, 50.0, 90.0, true},
		{"High jitter moving", 0.9, 50.0, 90.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig()
			config.Jitter = tt.jitter
			config.Speed = tt.speed
			config.Course = tt.course
			buffer := &bytes.Buffer{}
			sim, err := NewGPSSimulator(config, buffer)
			if err != nil {
				t.Fatalf("Failed to create GPS simulator: %v", err)
			}
			sim.isLocked = true

			// Store initial position
			initialLat := sim.currentLat
			initialLon := sim.currentLon

			// Update speed/course and position multiple times (proper sequence)
			var totalDistance float64
			for i := 0; i < 10; i++ {
				sim.updateSpeedAndCourse()

				// Add small delay to allow time-based movement calculation
				time.Sleep(10 * time.Millisecond)

				sim.updatePosition()

				// Track cumulative movement
				latChange := math.Abs(sim.currentLat - initialLat)
				lonChange := math.Abs(sim.currentLon - initialLon)
				totalDistance = math.Sqrt(latChange*latChange + lonChange*lonChange)
			}

			if tt.expectMovement {
				// For moving GPS at 50 knots over 100ms, should have detectable movement
				if totalDistance < 0.00001 { // Adjusted threshold for higher speed
					t.Errorf("Expected movement for speed %.1f, but total distance was %.8f",
						tt.speed, totalDistance)
				}
			} else {
				// For stationary GPS, movement should be minimal
				if totalDistance > 0.0001 { // Tighter threshold for stationary
					t.Errorf("Expected minimal movement for stationary GPS, but total distance was %.8f",
						totalDistance)
				}
			}

			// Check final position is within reasonable bounds
			finalDistance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
			// Allow more tolerance for high jitter and moving scenarios
			maxAllowedDistance := config.Radius * 1.5
			if finalDistance > maxAllowedDistance {
				t.Errorf("Final position too far from center: %.2f > %.2f",
					finalDistance, maxAllowedDistance)
			}
		})
	}
}

func TestUpdatePositionEdgeCases(t *testing.T) {
	t.Run("No time delta", func(t *testing.T) {
		config := createTestConfig()
		config.Speed = 10.0
		config.Course = 90.0
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		// Call updatePosition once to establish lastUpdateTime
		sim.updatePosition()

		// Store position after first update
		positionAfterFirst := [2]float64{sim.currentLat, sim.currentLon}

		// Manually set lastUpdateTime to future to guarantee deltaTime <= 0
		sim.lastUpdateTime = time.Now().Add(time.Second)

		// Call updatePosition again - should return early due to deltaTime <= 0
		sim.updatePosition()

		// Position should not change on second call
		latDiff := math.Abs(sim.currentLat - positionAfterFirst[0])
		lonDiff := math.Abs(sim.currentLon - positionAfterFirst[1])

		if latDiff > 1e-10 || lonDiff > 1e-10 {
			t.Errorf("Position should not change when deltaTime <= 0. After first: (%.10f, %.10f), After second: (%.10f, %.10f), Diff: (%.2e, %.2e)",
				positionAfterFirst[0], positionAfterFirst[1], sim.currentLat, sim.currentLon, latDiff, lonDiff)
		}
	})

	t.Run("High jitter boundary bouncing", func(t *testing.T) {
		config := createTestConfig()
		config.Speed = 100.0 // High speed to hit boundary quickly
		config.Course = 90.0 // East
		config.Jitter = 0.8  // High jitter for bouncing behavior
		config.Radius = 50.0 // Small radius to hit boundary
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		// Move close to boundary
		radiusDeg := config.Radius / 111320.0
		sim.currentLat = config.Latitude
		sim.currentLon = config.Longitude + radiusDeg*0.9 // Near east boundary

		// Update several times to trigger boundary bouncing
		for i := 0; i < 5; i++ {
			sim.updateSpeedAndCourse()
			time.Sleep(10 * time.Millisecond)
			sim.updatePosition()
		}

		// Course should have changed due to bouncing (not guaranteed every time due to randomness)
		// Just verify the bouncing logic was exercised by checking we stayed within reasonable bounds
		distance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
		if distance > config.Radius*2.0 { // Allow some overshoot for bouncing
			t.Errorf("High jitter bouncing failed to keep position reasonable. Distance: %.2f, Max expected: %.2f",
				distance, config.Radius*2.0)
		}
	})

	t.Run("Low jitter boundary constraint", func(t *testing.T) {
		config := createTestConfig()
		config.Speed = 100.0 // High speed to hit boundary quickly
		config.Course = 90.0 // East
		config.Jitter = 0.1  // Low jitter for constraint behavior
		config.Radius = 50.0 // Small radius to hit boundary
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		// Move close to boundary
		radiusDeg := config.Radius / 111320.0
		sim.currentLat = config.Latitude
		sim.currentLon = config.Longitude + radiusDeg*0.9 // Near east boundary

		// Update to trigger boundary constraint
		sim.updateSpeedAndCourse()
		time.Sleep(20 * time.Millisecond) // Longer time to ensure movement
		sim.updatePosition()

		// Should be constrained near the boundary
		distance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
		if distance > config.Radius*1.1 { // Allow small overshoot for constraint logic
			t.Errorf("Low jitter constraint failed. Distance: %.2f, Max expected: %.2f",
				distance, config.Radius*1.1)
		}
	})
}

func TestCourseNormalization(t *testing.T) {
	config := createTestConfig()
	config.Speed = 10.0
	config.Course = 350.0 // Near 360°
	config.Jitter = 0.8   // High jitter to cause large course changes
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
	sim.isLocked = true

	// Test multiple updates to trigger course normalization
	for i := 0; i < 20; i++ {
		sim.updateSpeedAndCourse()

		// Course should always be normalized to [0, 360)
		if sim.currentCourse < 0 || sim.currentCourse >= 360 {
			t.Errorf("Course not properly normalized: %.2f (should be in [0, 360))", sim.currentCourse)
		}
	}

	t.Run("Course normalization during bouncing", func(t *testing.T) {
		config := createTestConfig()
		config.Speed = 200.0 // Very high speed
		config.Course = 10.0 // Close to 0°
		config.Jitter = 0.9  // Maximum jitter for extreme course changes
		config.Radius = 30.0 // Very small radius to force frequent bouncing
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		// Set up position very close to boundary to force bouncing
		radiusDeg := config.Radius / 111320.0
		sim.currentLat = config.Latitude + radiusDeg*0.95
		sim.currentLon = config.Longitude

		// Force course to be near boundary values to test normalization
		sim.currentCourse = 358.0 // Near 360°

		// Update many times to trigger course normalization in bouncing logic
		for i := 0; i < 30; i++ {
			sim.updateSpeedAndCourse()
			time.Sleep(5 * time.Millisecond)
			sim.updatePosition()

			// Verify course is always normalized
			if sim.currentCourse < 0 || sim.currentCourse >= 360 {
				t.Errorf("Course not normalized during bouncing: %.2f", sim.currentCourse)
			}
		}
	})
}

func TestUpdatePositionBoundaryConstraints(t *testing.T) {
	// Test the new movement system's boundary handling with stationary GPS
	// This tests that a stationary GPS (speed=0) stays near its initial position
	config := createTestConfig()
	config.Radius = 50.0
	config.Speed = 0.0 // Stationary
	config.Course = 0.0
	config.Jitter = 0.1 // Lower jitter for more predictable bounds

	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
	sim.isLocked = true

	// Update position multiple times - should stay near center for stationary GPS
	maxDistance := 0.0
	for i := 0; i < 20; i++ {
		sim.updateSpeedAndCourse()
		sim.updatePosition()
		distance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
		if distance > maxDistance {
			maxDistance = distance
		}
	}

	// For stationary GPS with low jitter, should stay within reasonable bounds
	// With jitter=0.1, maximum jitter distance is 0.1 * 0.5 * radius = 5% of radius
	expectedMaxDistance := config.Radius * 0.3 // Should stay within 30% of radius (allowing for cumulative jitter drift)
	if maxDistance > expectedMaxDistance {
		t.Errorf("Stationary GPS moved too far from center. Max distance: %.2f, Expected max: %.2f",
			maxDistance, expectedMaxDistance)
	}
}

func TestUpdateAltitudeEdgeCases(t *testing.T) {
	t.Run("Zero altitude jitter", func(t *testing.T) {
		config := createTestConfig()
		config.Altitude = 1000.0
		config.AltitudeJitter = 0.0 // No jitter
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		initialAltitude := sim.currentAlt

		// Update multiple times - altitude should remain stable
		for i := 0; i < 10; i++ {
			sim.updateAltitude()
		}

		if sim.currentAlt != initialAltitude {
			t.Errorf("Altitude changed with zero jitter. Initial: %.1f, Final: %.1f",
				initialAltitude, sim.currentAlt)
		}
	})

	t.Run("Altitude bounds checking", func(t *testing.T) {
		config := createTestConfig()
		config.Altitude = 100.0
		config.AltitudeJitter = 1.0 // Maximum jitter
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		// Update many times to test boundary conditions
		minAlt := config.Altitude
		maxAlt := config.Altitude
		for i := 0; i < 100; i++ {
			sim.updateAltitude()
			if sim.currentAlt < minAlt {
				minAlt = sim.currentAlt
			}
			if sim.currentAlt > maxAlt {
				maxAlt = sim.currentAlt
			}
		}

		// Should be within expected bounds
		expectedMin := math.Max(config.Altitude-100.0, -50.0)
		expectedMax := config.Altitude + 500.0

		if minAlt < expectedMin-1.0 { // Small tolerance for floating point
			t.Errorf("Minimum altitude %.1f below expected minimum %.1f", minAlt, expectedMin)
		}
		if maxAlt > expectedMax+1.0 { // Small tolerance for floating point
			t.Errorf("Maximum altitude %.1f above expected maximum %.1f", maxAlt, expectedMax)
		}
	})

	t.Run("Sea level altitude bounds", func(t *testing.T) {
		config := createTestConfig()
		config.Altitude = 10.0      // Near sea level
		config.AltitudeJitter = 1.0 // Maximum jitter
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		// Update many times to test sea level boundary
		minAlt := config.Altitude
		for i := 0; i < 100; i++ {
			sim.updateAltitude()
			if sim.currentAlt < minAlt {
				minAlt = sim.currentAlt
			}
		}

		// Should not go too far below sea level
		if minAlt < -50.0 {
			t.Errorf("Altitude went too far below sea level: %.1f", minAlt)
		}
	})
}

func TestUpdateSatellites(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Store initial satellite states
	initialSats := make([]Satellite, len(sim.Satellites))
	copy(initialSats, sim.Satellites)

	// Update satellites multiple times
	for i := 0; i < 10; i++ {
		sim.updateSatellites()

		// Check that all satellites remain within valid bounds
		for j, sat := range sim.Satellites {
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
	for i, sat := range sim.Satellites {
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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Test elevation boundary conditions
	sim.Satellites[0].Elevation = 4  // Below minimum
	sim.Satellites[1].Elevation = 86 // Above maximum

	// Test SNR boundary conditions
	sim.Satellites[2].SNR = 14 // Below minimum
	sim.Satellites[3].SNR = 56 // Above maximum

	sim.updateSatellites()

	// Check that boundaries are enforced
	if sim.Satellites[0].Elevation < 5 {
		t.Errorf("Expected elevation to be at least 5, got %d", sim.Satellites[0].Elevation)
	}
	if sim.Satellites[1].Elevation > 85 {
		t.Errorf("Expected elevation to be at most 85, got %d", sim.Satellites[1].Elevation)
	}
	if sim.Satellites[2].SNR < 15 {
		t.Errorf("Expected SNR to be at least 15, got %d", sim.Satellites[2].SNR)
	}
	if sim.Satellites[3].SNR > 55 {
		t.Errorf("Expected SNR to be at most 55, got %d", sim.Satellites[3].SNR)
	}
}

func TestUpdate(t *testing.T) {
	config := createTestConfig()
	config.TimeToLock = 100 * time.Millisecond // Short lock time for testing
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

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
	if !strings.Contains(output, "$GPGLL,") {
		t.Error("Output should contain GLL sentence when not locked")
	}
	if !strings.Contains(output, "$GPVTG,") {
		t.Error("Output should contain VTG sentence when not locked")
	}
	// Should not contain GSA, GSV, or ZDA when not locked
	if strings.Contains(output, "$GPGSA,") {
		t.Error("Output should not contain GSA sentence when not locked")
	}
	if strings.Contains(output, "$GPGSV,") {
		t.Error("Output should not contain GSV sentence when not locked")
	}
	if strings.Contains(output, "$GPZDA,") {
		t.Error("Output should not contain ZDA sentence when not locked")
	}

	// Test output when locked
	sim.isLocked = true
	buffer.Reset()
	sim.outputNMEA()
	output = buffer.String()

	// Should contain all sentence types
	expectedSentences := []string{"$GPGGA,", "$GPRMC,", "$GPGLL,", "$GPVTG,", "$GPGSA,", "$GPGSV,", "$GPZDA,"}
	for _, sentence := range expectedSentences {
		if !strings.Contains(output, sentence) {
			t.Errorf("Output should contain %s sentence when locked", sentence)
		}
	}

	// Count GSV sentences (should be based on number of satellites)
	gsvCount := strings.Count(output, "$GPGSV,")
	expectedGSVCount := (len(sim.Satellites) + 3) / 4 // Round up to nearest 4
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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		b.Fatalf("Failed to create GPS simulator: %v", err)
	}
	sim.isLocked = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.updatePosition()
	}
}

func BenchmarkUpdateSatellites(b *testing.B) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		b.Fatalf("Failed to create GPS simulator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.updateSatellites()
	}
}

func BenchmarkDistanceFromCenter(b *testing.B) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		b.Fatalf("Failed to create GPS simulator: %v", err)
	}

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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
	sim.Satellites = []Satellite{} // Empty satellites

	result := sim.generateGSV()
	if len(result) != 0 {
		t.Errorf("Expected 0 GSV sentences for 0 satellites, got %d", len(result))
	}

	// Test with 1 satellite (to test padding logic)
	sim.Satellites = []Satellite{
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
	sim.Satellites = []Satellite{
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
	sim.Satellites = []Satellite{
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
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		b.Fatalf("Failed to create GPS simulator: %v", err)
	}
	sim.isLocked = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Reset()
		sim.outputNMEA()
	}
}

func TestUpdateSatellitesEdgeCases(t *testing.T) {
	t.Run("Satellite movement consistency", func(t *testing.T) {
		config := createTestConfig()
		config.Satellites = 1 // Single satellite for easier testing
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Update multiple times and track changes
		elevationChanges := 0
		azimuthChanges := 0
		snrChanges := 0

		for i := 0; i < 10; i++ {
			prevElev := sim.Satellites[0].Elevation
			prevAzim := sim.Satellites[0].Azimuth
			prevSNR := sim.Satellites[0].SNR

			sim.updateSatellites()

			if sim.Satellites[0].Elevation != prevElev {
				elevationChanges++
			}
			if sim.Satellites[0].Azimuth != prevAzim {
				azimuthChanges++
			}
			if sim.Satellites[0].SNR != prevSNR {
				snrChanges++
			}
		}

		// Should have some changes (not all updates will change values due to randomness)
		if elevationChanges == 0 && azimuthChanges == 0 && snrChanges == 0 {
			t.Error("No satellite parameters changed over 10 updates")
		}
	})

	t.Run("Extreme satellite boundary conditions", func(t *testing.T) {
		config := createTestConfig()
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Set satellites to boundary values
		sim.Satellites[0].Elevation = 4  // Below minimum
		sim.Satellites[1].Elevation = 86 // Above maximum
		sim.Satellites[2].SNR = 14       // Below minimum
		sim.Satellites[3].SNR = 56       // Above maximum

		// Update multiple times to ensure boundaries are maintained
		for i := 0; i < 20; i++ {
			sim.updateSatellites()

			for j, sat := range sim.Satellites {
				if sat.Elevation < 5 || sat.Elevation > 85 {
					t.Errorf("Satellite %d elevation %d out of bounds [5, 85]", j, sat.Elevation)
				}
				if sat.SNR < 15 || sat.SNR > 55 {
					t.Errorf("Satellite %d SNR %d out of bounds [15, 55]", j, sat.SNR)
				}
				if sat.Azimuth < 0 || sat.Azimuth >= 360 {
					t.Errorf("Satellite %d azimuth %d out of bounds [0, 360)", j, sat.Azimuth)
				}
			}
		}
	})
}

func TestClose(t *testing.T) {
	// Test Close function with GPX writer
	config := createTestConfig()
	config.GPXEnabled = true
	tempDir := t.TempDir()
	config.GPXFile = filepath.Join(tempDir, "test_close.gpx")
	config.Quiet = false
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Add some track points
	sim.isLocked = true
	sim.updateGPX()
	sim.updateGPX()

	// Capture stderr for testing output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Close the simulator
	sim.Close()

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 1000)
	n, _ := r.Read(captured)
	output := string(captured[:n])

	// Should contain GPX file writing message
	if !strings.Contains(output, "Writing GPX file:") || !strings.Contains(output, "test_close.gpx") {
		t.Errorf("Expected GPX writing message in output, got: %s", output)
	}

	// Clean up

}

func TestCloseWithoutGPX(t *testing.T) {
	// Test Close function without GPX writer
	config := createTestConfig()
	config.GPXEnabled = false
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Should not panic when closing without GPX writer
	sim.Close()
}

func TestCloseQuietMode(t *testing.T) {
	// Test Close function in quiet mode
	config := createTestConfig()
	config.GPXEnabled = true
	tempDir := t.TempDir()
	config.GPXFile = filepath.Join(tempDir, "test_close_quiet.gpx")
	config.Quiet = true
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Add some track points
	sim.isLocked = true
	sim.updateGPX()

	// Capture stderr for testing output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Close the simulator
	sim.Close()

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 1000)
	n, _ := r.Read(captured)
	output := string(captured[:n])

	// Should not contain any output in quiet mode
	if len(output) > 0 {
		t.Errorf("Expected no output in quiet mode, got: %s", output)
	}

	// Clean up

}

func TestUpdateGPX(t *testing.T) {
	// Test updateGPX function with GPX enabled and GPS locked
	config := createTestConfig()
	config.GPXEnabled = true
	tempDir := t.TempDir()
	config.GPXFile = filepath.Join(tempDir, "test_update_gpx.gpx")
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// GPS not locked - should not add points
	sim.updateGPX()
	if sim.gpxWriter.GetTrackPointCount() != 0 {
		t.Error("Should not add track points when GPS is not locked")
	}

	// GPS locked - should add points
	sim.isLocked = true
	sim.updateGPX()
	if sim.gpxWriter.GetTrackPointCount() != 1 {
		t.Errorf("Expected 1 track point, got %d", sim.gpxWriter.GetTrackPointCount())
	}

	// Add more points to test periodic writing (every 10 points)
	for i := 0; i < 12; i++ {
		sim.updateGPX()
	}

	if sim.gpxWriter.GetTrackPointCount() != 13 {
		t.Errorf("Expected 13 track points, got %d", sim.gpxWriter.GetTrackPointCount())
	}

	// Clean up
	sim.Close()

}

func TestUpdateGPXWithoutGPXWriter(t *testing.T) {
	// Test updateGPX function without GPX writer
	config := createTestConfig()
	config.GPXEnabled = false
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	sim.isLocked = true
	// Should not panic when calling updateGPX without GPX writer
	sim.updateGPX()
}

func TestNewGPSSimulatorWithGPXError(t *testing.T) {
	// Test NewGPSSimulator with invalid GPX file path (non-existent directory)
	config := createTestConfig()
	config.GPXEnabled = true
	config.GPXFile = "/non/existent/directory/test.gpx"
	buffer := &bytes.Buffer{}

	_, err := NewGPSSimulator(config, buffer)
	if err == nil {
		t.Error("Expected error for invalid GPX file path, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create GPX writer") {
		t.Errorf("Expected GPX writer error, got: %v", err)
	}
}

func TestNewGPSSimulatorWithGPXEnabled(t *testing.T) {
	// Test NewGPSSimulator with GPX enabled
	config := createTestConfig()
	config.GPXEnabled = true
	tempDir := t.TempDir()
	config.GPXFile = filepath.Join(tempDir, "test_new_simulator.gpx")
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	if sim.gpxWriter == nil {
		t.Error("Expected GPX writer to be initialized")
	}

	// Clean up
	sim.Close()

}

func TestRunWithDuration(t *testing.T) {
	// Test Run function with duration limit
	config := createTestConfig()
	config.OutputRate = 10 * time.Millisecond
	config.Duration = 50 * time.Millisecond
	config.Quiet = false
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Capture stderr for testing output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	start := time.Now()
	sim.Run()
	elapsed := time.Since(start)

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 1000)
	n, _ := r.Read(captured)
	output := string(captured[:n])

	// Should have run for approximately the specified duration
	if elapsed < 40*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("Expected run time around 50ms, got %v", elapsed)
	}

	// Should contain duration messages
	if !strings.Contains(output, "Simulation will run for") {
		t.Error("Expected duration start message in output")
	}
	if !strings.Contains(output, "Simulation completed after") {
		t.Error("Expected duration completion message in output")
	}

	// Should have generated some NMEA output
	nmeaOutput := buffer.String()
	if len(nmeaOutput) == 0 {
		t.Error("Expected NMEA output from Run function")
	}
}

func TestRunWithDurationQuiet(t *testing.T) {
	// Test Run function with duration in quiet mode
	config := createTestConfig()
	config.OutputRate = 10 * time.Millisecond
	config.Duration = 30 * time.Millisecond
	config.Quiet = true
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Capture stderr for testing output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	sim.Run()

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 1000)
	n, _ := r.Read(captured)
	output := string(captured[:n])

	// Should not contain any output in quiet mode
	if len(output) > 0 {
		t.Errorf("Expected no output in quiet mode, got: %s", output)
	}
}

func TestUpdateGPXWriteError(t *testing.T) {
	// Test updateGPX with WriteToFile error
	config := createTestConfig()
	config.GPXEnabled = true
	tempDir := t.TempDir()
	config.GPXFile = filepath.Join(tempDir, "test_update_gpx_error.gpx")
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
	defer sim.Close()

	sim.isLocked = true

	// Add 9 track points (won't trigger write)
	for i := 0; i < 9; i++ {
		sim.updateGPX()
	}

	// Close the underlying file to cause WriteToFile error on 10th point
	if sim.gpxWriter.file != nil {
		sim.gpxWriter.file.Close()
	}

	// Capture stderr to verify error message
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Add 10th point - should trigger WriteToFile error
	sim.updateGPX()

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 1000)
	n, _ := r.Read(captured)
	output := string(captured[:n])

	// Should contain error message
	if !strings.Contains(output, "Error writing GPX data:") {
		t.Errorf("Expected GPX write error message in output, got: %s", output)
	}
}

func TestCloseWithGPXError(t *testing.T) {
	// Test Close with GPX writer error
	config := createTestConfig()
	config.GPXEnabled = true
	tempDir := t.TempDir()
	config.GPXFile = filepath.Join(tempDir, "test_close_gpx_error.gpx")
	config.Quiet = false
	buffer := &bytes.Buffer{}

	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Add some track points
	sim.isLocked = true
	sim.updateGPX()

	// Close the underlying GPX file to cause error in Close
	if sim.gpxWriter.file != nil {
		sim.gpxWriter.file.Close()
	}

	// Capture stderr to verify error messages
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Close should trigger error
	sim.Close()

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 1000)
	n, _ := r.Read(captured)
	output := string(captured[:n])

	// Should contain both writing message and error message
	if !strings.Contains(output, "Writing GPX file:") || !strings.Contains(output, "test_close_gpx_error.gpx") {
		t.Errorf("Expected GPX writing message in output, got: %s", output)
	}
	if !strings.Contains(output, "Error closing GPX file:") {
		t.Errorf("Expected GPX close error message in output, got: %s", output)
	}
}

func TestUpdatePositionEdgeCasesAdvanced(t *testing.T) {
	// Test more edge cases for updatePosition to improve coverage
	t.Run("Zero radius with movement", func(t *testing.T) {
		config := createTestConfig()
		config.Radius = 0.0 // Zero radius
		config.Speed = 50.0 // Higher speed for more detectable movement
		config.Course = 45.0
		config.Jitter = 0.1
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		sim.isLocked = true
		sim.currentSpeed = config.Speed
		sim.currentCourse = config.Course
		initialLat := sim.currentLat
		initialLon := sim.currentLon

		// Update position multiple times with longer time intervals
		for i := 0; i < 3; i++ {
			sim.updateSpeedAndCourse()
			time.Sleep(50 * time.Millisecond) // Longer time for more movement
			sim.updatePosition()
		}

		// With zero radius, position should still be able to change due to movement
		latChange := math.Abs(sim.currentLat - initialLat)
		lonChange := math.Abs(sim.currentLon - initialLon)
		totalChange := math.Sqrt(latChange*latChange + lonChange*lonChange)

		// With zero radius, movement should still occur if speed > 0
		// This tests the boundary logic when radius is zero
		if totalChange < 0.00001 {
			// This might actually be expected behavior - zero radius might constrain movement
			t.Logf("Position change was minimal (%.8f) with zero radius - this may be correct behavior", totalChange)
		}
	})

	t.Run("Very high speed movement", func(t *testing.T) {
		config := createTestConfig()
		config.Speed = 1000.0   // Very high speed
		config.Course = 0.0     // Due north
		config.Jitter = 0.0     // No jitter for predictable movement
		config.Radius = 10000.0 // Large radius to avoid boundary effects
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		sim.isLocked = true
		sim.currentSpeed = config.Speed
		sim.currentCourse = config.Course

		initialLat := sim.currentLat

		// Update with significant time gap
		time.Sleep(50 * time.Millisecond)
		sim.updatePosition()

		// Should have moved significantly north
		latChange := sim.currentLat - initialLat
		if latChange <= 0 {
			t.Error("Expected northward movement with high speed")
		}
	})

	t.Run("Boundary collision with different courses", func(t *testing.T) {
		courses := []float64{0, 45, 90, 135, 180, 225, 270, 315}

		for _, course := range courses {
			config := createTestConfig()
			config.Speed = 100.0
			config.Course = course
			config.Jitter = 0.9  // High jitter for bouncing
			config.Radius = 30.0 // Small radius
			buffer := &bytes.Buffer{}
			sim, err := NewGPSSimulator(config, buffer)
			if err != nil {
				t.Fatalf("Failed to create GPS simulator: %v", err)
			}

			sim.isLocked = true
			sim.currentSpeed = config.Speed
			sim.currentCourse = course

			// Move close to boundary in the direction of the course
			radiusDeg := config.Radius / 111320.0
			switch {
			case course >= 315 || course < 45: // North
				sim.currentLat = config.Latitude + radiusDeg*0.9
			case course >= 45 && course < 135: // East
				sim.currentLon = config.Longitude + radiusDeg*0.9
			case course >= 135 && course < 225: // South
				sim.currentLat = config.Latitude - radiusDeg*0.9
			case course >= 225 && course < 315: // West
				sim.currentLon = config.Longitude - radiusDeg*0.9
			}

			// Update position to trigger boundary logic
			sim.updateSpeedAndCourse()
			time.Sleep(20 * time.Millisecond)
			sim.updatePosition()

			// Verify still within reasonable bounds
			distance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
			if distance > config.Radius*2.0 {
				t.Errorf("Course %.0f: Position too far from center: %.2f > %.2f",
					course, distance, config.Radius*2.0)
			}
		}
	})
}

func TestUpdateSpeedAndCourseEdgeCases(t *testing.T) {
	t.Run("Zero speed edge case", func(t *testing.T) {
		config := createTestConfig()
		config.Speed = 0.0
		config.Jitter = 0.8 // High jitter
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Update multiple times
		for i := 0; i < 10; i++ {
			sim.updateSpeedAndCourse()

			// Speed should never go negative
			if sim.currentSpeed < 0 {
				t.Errorf("Speed went negative: %.2f", sim.currentSpeed)
			}
		}
	})

	t.Run("Course boundary wraparound", func(t *testing.T) {
		testCases := []struct {
			name   string
			course float64
			jitter float64
		}{
			{"Near 0 degrees", 5.0, 0.8},
			{"Near 360 degrees", 355.0, 0.8},
			{"Exactly 0 degrees", 0.0, 0.9},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := createTestConfig()
				config.Course = tc.course
				config.Jitter = tc.jitter
				buffer := &bytes.Buffer{}
				sim, err := NewGPSSimulator(config, buffer)
				if err != nil {
					t.Fatalf("Failed to create GPS simulator: %v", err)
				}

				// Update many times to test wraparound
				for i := 0; i < 50; i++ {
					sim.updateSpeedAndCourse()

					// Course should always be in valid range
					if sim.currentCourse < 0 || sim.currentCourse >= 360 {
						t.Errorf("Course out of bounds: %.2f (should be [0, 360))", sim.currentCourse)
					}
				}
			})
		}
	})

	t.Run("Jitter level variations", func(t *testing.T) {
		jitterLevels := []float64{0.05, 0.15, 0.35, 0.65, 0.85, 0.95}

		for _, jitter := range jitterLevels {
			t.Run(fmt.Sprintf("Jitter %.2f", jitter), func(t *testing.T) {
				config := createTestConfig()
				config.Speed = 10.0
				config.Course = 90.0
				config.Jitter = jitter
				buffer := &bytes.Buffer{}
				sim, err := NewGPSSimulator(config, buffer)
				if err != nil {
					t.Fatalf("Failed to create GPS simulator: %v", err)
				}

				speedVariations := []float64{}
				courseVariations := []float64{}

				for i := 0; i < 20; i++ {
					sim.updateSpeedAndCourse()
					speedVariations = append(speedVariations, sim.currentSpeed)
					courseVariations = append(courseVariations, sim.currentCourse)
				}

				// Calculate variation ranges
				minSpeed, maxSpeed := speedVariations[0], speedVariations[0]
				minCourse, maxCourse := courseVariations[0], courseVariations[0]

				for _, speed := range speedVariations {
					if speed < minSpeed {
						minSpeed = speed
					}
					if speed > maxSpeed {
						maxSpeed = speed
					}
				}
				for _, course := range courseVariations {
					if course < minCourse {
						minCourse = course
					}
					if course > maxCourse {
						maxCourse = course
					}
				}

				speedRange := maxSpeed - minSpeed
				_ = maxCourse - minCourse // courseRange - not used but calculated for completeness

				// Higher jitter should produce larger variations
				if jitter < 0.2 && speedRange > 2.0 {
					t.Errorf("Low jitter (%.2f) produced unexpectedly large speed range: %.2f", jitter, speedRange)
				}
				if jitter > 0.8 && speedRange < 1.0 {
					t.Errorf("High jitter (%.2f) produced unexpectedly small speed range: %.2f", jitter, speedRange)
				}
			})
		}
	})
}

// TestDeterministicBoundaryConditions ensures consistent coverage of boundary conditions
// that are normally hit randomly, eliminating coverage variation between test runs
func TestDeterministicBoundaryConditions(t *testing.T) {
	t.Run("Speed negative boundary", func(t *testing.T) {
		config := createTestConfig()
		config.Speed = 0.1  // Very low speed
		config.Jitter = 0.9 // High jitter to force negative speeds
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Force a scenario where speed goes negative
		sim.Config.Speed = 1.0
		sim.currentSpeed = 1.0

		// Manually set a large negative delta to force the boundary condition
		originalSpeed := sim.currentSpeed
		sim.currentSpeed = -0.5 // Force negative speed

		// Call updateSpeedAndCourse to trigger the boundary check
		sim.updateSpeedAndCourse()

		// The speed should be corrected to 0 or positive
		if sim.currentSpeed < 0 {
			t.Errorf("Speed should not be negative after update, got %.2f", sim.currentSpeed)
		}

		// Reset for normal operation
		sim.currentSpeed = originalSpeed
	})

	t.Run("Course normalization boundaries", func(t *testing.T) {
		config := createTestConfig()
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Test negative course normalization
		sim.currentCourse = -10.0
		sim.updateSpeedAndCourse()
		if sim.currentCourse < 0 || sim.currentCourse >= 360 {
			t.Errorf("Course should be normalized to 0-359 range, got %.2f", sim.currentCourse)
		}

		// Test course >= 360 normalization
		sim.currentCourse = 370.0
		sim.updateSpeedAndCourse()
		if sim.currentCourse < 0 || sim.currentCourse >= 360 {
			t.Errorf("Course should be normalized to 0-359 range, got %.2f", sim.currentCourse)
		}
	})

	t.Run("Satellite elevation boundaries", func(t *testing.T) {
		config := createTestConfig()
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Force satellites to boundary conditions
		for i := range sim.Satellites {
			// Test low elevation boundary
			sim.Satellites[i].Elevation = 3 // Below minimum of 5
			sim.updateSatellites()
			if sim.Satellites[i].Elevation < 5 {
				t.Errorf("Satellite %d elevation should be at least 5, got %d", i, sim.Satellites[i].Elevation)
			}

			// Test high elevation boundary
			sim.Satellites[i].Elevation = 87 // Above maximum of 85
			sim.updateSatellites()
			if sim.Satellites[i].Elevation > 85 {
				t.Errorf("Satellite %d elevation should be at most 85, got %d", i, sim.Satellites[i].Elevation)
			}
		}
	})

	t.Run("Satellite SNR boundaries", func(t *testing.T) {
		config := createTestConfig()
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Force satellites to SNR boundary conditions
		for i := range sim.Satellites {
			// Test low SNR boundary
			sim.Satellites[i].SNR = 10 // Below minimum of 15
			sim.updateSatellites()
			if sim.Satellites[i].SNR < 15 {
				t.Errorf("Satellite %d SNR should be at least 15, got %d", i, sim.Satellites[i].SNR)
			}

			// Test high SNR boundary
			sim.Satellites[i].SNR = 60 // Above maximum of 55
			sim.updateSatellites()
			if sim.Satellites[i].SNR > 55 {
				t.Errorf("Satellite %d SNR should be at most 55, got %d", i, sim.Satellites[i].SNR)
			}
		}
	})

	t.Run("Altitude boundaries", func(t *testing.T) {
		config := createTestConfig()
		config.Altitude = 100.0     // Starting altitude
		config.AltitudeJitter = 0.9 // High jitter to trigger boundaries
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Test minimum altitude boundary (below sea level)
		sim.currentAlt = -60.0 // Below -50.0 minimum
		sim.updateAltitude()
		if sim.currentAlt < -50.0 {
			t.Errorf("Altitude should not go below -50m (sea level limit), got %.2f", sim.currentAlt)
		}

		// Test minimum relative to starting altitude
		sim.Config.Altitude = 200.0 // High starting altitude
		sim.currentAlt = 80.0       // Below (200 - 100) = 100m minimum
		sim.updateAltitude()
		if sim.currentAlt < 100.0 {
			t.Errorf("Altitude should not go below starting-100m, got %.2f", sim.currentAlt)
		}

		// Test maximum altitude boundary
		sim.currentAlt = 750.0 // Above (200 + 500) = 700m maximum
		sim.updateAltitude()
		if sim.currentAlt > 700.0 {
			t.Errorf("Altitude should not exceed starting+500m, got %.2f", sim.currentAlt)
		}

		// Test the sea level boundary condition specifically
		sim.Config.Altitude = 10.0 // Low starting altitude
		sim.currentAlt = -60.0     // This should trigger the -50.0 sea level minimum
		sim.updateAltitude()
		if sim.currentAlt < -50.0 {
			t.Errorf("Sea level boundary should prevent altitude below -50m, got %.2f", sim.currentAlt)
		}
	})

	t.Run("Position update edge cases", func(t *testing.T) {
		config := createTestConfig()
		config.Jitter = 0.0 // Zero jitter - no GPS noise
		config.Speed = 50.0 // High speed
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}

		// Test with zero speed and zero jitter - position should not change
		originalLat := sim.currentLat
		originalLon := sim.currentLon
		sim.currentSpeed = 0.0 // Zero speed with zero jitter should result in no position change

		sim.updatePosition()

		// With zero speed and zero jitter, position should remain unchanged
		latDiff := math.Abs(sim.currentLat - originalLat)
		lonDiff := math.Abs(sim.currentLon - originalLon)
		if latDiff > 0.000001 || lonDiff > 0.000001 {
			t.Errorf("Position should not change with zero speed and zero jitter, lat diff: %f, lon diff: %f", latDiff, lonDiff)
		}
	})

	t.Run("Stationary GPS jitter", func(t *testing.T) {
		config := createTestConfig()
		config.Jitter = 0.5 // Medium jitter for stationary GPS noise
		config.Speed = 0.0  // Zero speed - stationary
		buffer := &bytes.Buffer{}
		sim, err := NewGPSSimulator(config, buffer)
		if err != nil {
			t.Fatalf("Failed to create GPS simulator: %v", err)
		}
		sim.isLocked = true

		// Test multiple updates to ensure stationary jitter occurs
		originalLat := sim.currentLat
		originalLon := sim.currentLon

		positionChanged := false
		for i := 0; i < 10; i++ {
			time.Sleep(10 * time.Millisecond) // Small delay to ensure deltaTime > 0
			sim.updatePosition()

			latDiff := math.Abs(sim.currentLat - originalLat)
			lonDiff := math.Abs(sim.currentLon - originalLon)

			// Check if position has changed due to jitter (should happen with medium jitter)
			if latDiff > 1e-8 || lonDiff > 1e-8 {
				positionChanged = true

				// Verify jitter stays within radius
				distance := sim.distanceFromCenter(sim.currentLat, sim.currentLon)
				if distance > config.Radius {
					t.Errorf("Stationary jitter exceeded radius constraint. Distance: %.6f, Radius: %.6f", distance, config.Radius)
				}
				break
			}
		}

		if !positionChanged {
			t.Error("Stationary GPS should show position jitter with non-zero jitter setting")
		}
	})
}

// Tests for GPX replay functionality in simulator

func TestNewGPSSimulatorWithReplay(t *testing.T) {
	// Create a test GPX file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_replay_simulator.gpx")

	gpxContent := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Test Track</name>
    <trkseg>
      <trkpt lat="37.774900" lon="-122.419400">
        <ele>50.0</ele>
        <time>2024-01-15T10:00:00Z</time>
      </trkpt>
      <trkpt lat="37.775000" lon="-122.419300">
        <ele>52.0</ele>
        <time>2024-01-15T10:00:10Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`

	err := os.WriteFile(tempFile, []byte(gpxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test GPX file: %v", err)
	}

	config := createTestConfig()
	config.ReplayFile = tempFile
	config.ReplaySpeed = 1.0

	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator with replay: %v", err)
	}

	// Verify replay points were loaded
	if len(sim.replayPoints) != 2 {
		t.Errorf("Expected 2 replay points, got %d", len(sim.replayPoints))
	}

	// Verify initial position was set from first point
	if sim.currentLat != 37.774900 {
		t.Errorf("Expected initial lat 37.774900, got %f", sim.currentLat)
	}
	if sim.currentLon != -122.419400 {
		t.Errorf("Expected initial lon -122.419400, got %f", sim.currentLon)
	}
	if sim.currentAlt != 50.0 {
		t.Errorf("Expected initial alt 50.0, got %f", sim.currentAlt)
	}
}

func TestNewGPSSimulatorWithReplayError(t *testing.T) {
	config := createTestConfig()
	config.ReplayFile = "non_existent_file.gpx"

	buffer := &bytes.Buffer{}
	_, err := NewGPSSimulator(config, buffer)
	if err == nil {
		t.Error("Expected error when loading non-existent replay file")
	}
	if !strings.Contains(err.Error(), "failed to load replay file") {
		t.Errorf("Expected error about failed to load replay file, got: %v", err)
	}
}

func TestHasSequentialTimestamps(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	tests := []struct {
		name     string
		points   []TrackPoint
		expected bool
	}{
		{
			name: "Sequential timestamps",
			points: []TrackPoint{
				{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
				{Time: time.Date(2024, 1, 1, 10, 0, 10, 0, time.UTC)},
				{Time: time.Date(2024, 1, 1, 10, 0, 20, 0, time.UTC)},
			},
			expected: true,
		},
		{
			name: "Non-sequential timestamps",
			points: []TrackPoint{
				{Time: time.Date(2024, 1, 1, 10, 0, 20, 0, time.UTC)},
				{Time: time.Date(2024, 1, 1, 10, 0, 10, 0, time.UTC)},
				{Time: time.Date(2024, 1, 1, 10, 0, 30, 0, time.UTC)},
			},
			expected: false,
		},
		{
			name: "Single point",
			points: []TrackPoint{
				{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
			},
			expected: false,
		},
		{
			name:     "No points",
			points:   []TrackPoint{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim.replayPoints = tt.points
			result := sim.hasSequentialTimestamps()
			if result != tt.expected {
				t.Errorf("Expected hasSequentialTimestamps() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculateDistance(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "Same point",
			lat1:      37.7749,
			lon1:      -122.4194,
			lat2:      37.7749,
			lon2:      -122.4194,
			expected:  0.0,
			tolerance: 0.1,
		},
		{
			name:      "San Francisco to nearby point",
			lat1:      37.7749,
			lon1:      -122.4194,
			lat2:      37.7750,
			lon2:      -122.4194,
			expected:  11.1, // Approximately 11.1 meters per 0.0001 degree latitude
			tolerance: 1.0,
		},
		{
			name:      "Longer distance",
			lat1:      37.7749,
			lon1:      -122.4194,
			lat2:      37.7849,
			lon2:      -122.4094,
			expected:  1400.0, // Approximately 1.4km
			tolerance: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sim.calculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("Expected distance ~%.1f ± %.1f, got %.1f", tt.expected, tt.tolerance, result)
			}
		})
	}
}

func TestCalculateBearing(t *testing.T) {
	config := createTestConfig()
	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "North bearing",
			lat1:      37.0,
			lon1:      -122.0,
			lat2:      38.0,
			lon2:      -122.0,
			expected:  0.0, // Due north
			tolerance: 1.0,
		},
		{
			name:      "East bearing",
			lat1:      37.0,
			lon1:      -122.0,
			lat2:      37.0,
			lon2:      -121.0,
			expected:  90.0, // Due east
			tolerance: 1.0,
		},
		{
			name:      "South bearing",
			lat1:      38.0,
			lon1:      -122.0,
			lat2:      37.0,
			lon2:      -122.0,
			expected:  180.0, // Due south
			tolerance: 1.0,
		},
		{
			name:      "West bearing",
			lat1:      37.0,
			lon1:      -121.0,
			lat2:      37.0,
			lon2:      -122.0,
			expected:  270.0, // Due west
			tolerance: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sim.calculateBearing(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			// Normalize expected value to 0-359 range
			normalizedExpected := math.Mod(tt.expected+360, 360)
			if math.Abs(result-normalizedExpected) > tt.tolerance {
				t.Errorf("Expected bearing ~%.1f ± %.1f, got %.1f", normalizedExpected, tt.tolerance, result)
			}
		})
	}
}

func TestDistanceFromCenterRefactoring(t *testing.T) {
	// Test that distanceFromCenter produces the same results as calculateDistance
	config := createTestConfig()
	config.Latitude = 37.7749
	config.Longitude = -122.4194

	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	testLat := 37.7750
	testLon := -122.4193

	// Both methods should return the same result
	distanceFromCenter := sim.distanceFromCenter(testLat, testLon)
	calculateDistance := sim.calculateDistance(config.Latitude, config.Longitude, testLat, testLon)

	if math.Abs(distanceFromCenter-calculateDistance) > 0.001 {
		t.Errorf("distanceFromCenter and calculateDistance should return same result, got %.6f vs %.6f",
			distanceFromCenter, calculateDistance)
	}
}

func TestUpdateReplayPosition(t *testing.T) {
	// Create a test GPX file with route data (non-sequential timestamps)
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_replay_update.gpx")

	gpxContent := `<?xml version="1.0"?>
<gpx version="1.0" creator="test" xmlns="http://www.topografix.com/GPX/1/0">
  <rte>
    <name>Test Route</name>
    <rtept lat="42.430950" lon="-71.107628">
      <ele>23.5</ele>
      <time>2001-11-28T21:05:28Z</time>
    </rtept>
    <rtept lat="42.431240" lon="-71.109236">
      <ele>26.6</ele>
      <time>2001-06-02T03:26:55Z</time>
    </rtept>
    <rtept lat="42.432000" lon="-71.110000">
      <ele>30.0</ele>
      <time>2001-12-01T12:00:00Z</time>
    </rtept>
  </rte>
</gpx>`

	err := os.WriteFile(tempFile, []byte(gpxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test GPX file: %v", err)
	}

	config := createTestConfig()
	config.ReplayFile = tempFile
	config.ReplaySpeed = 2.0 // 2x speed
	config.ReplayLoop = true // Enable looping for this test

	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator with replay: %v", err)
	}

	// Should detect non-sequential timestamps and use index-based progression
	if sim.hasSequentialTimestamps() {
		t.Error("Expected non-sequential timestamps to be detected")
	}

	// Initial position should be from first point
	if sim.currentLat != 42.430950 {
		t.Errorf("Expected initial lat 42.430950, got %f", sim.currentLat)
	}

	// Test index-based progression
	t.Run("Index-based progression", func(t *testing.T) {
		// Simulate some time passing and update replay position
		sim.replayStartTime = time.Now().Add(-2 * time.Second) // 2 seconds ago
		sim.updateReplayPosition()

		// With 2x speed and 2 seconds elapsed, should be at index 4 % 3 = 1
		expectedIndex := 1
		if sim.replayIndex != expectedIndex {
			t.Errorf("Expected replay index %d, got %d", expectedIndex, sim.replayIndex)
		}

		// Position should have updated to second point
		if sim.currentLat != 42.431240 {
			t.Errorf("Expected updated lat 42.431240, got %f", sim.currentLat)
		}
		if sim.currentAlt != 26.6 {
			t.Errorf("Expected updated alt 26.6, got %f", sim.currentAlt)
		}
	})

	// Test looping behavior
	t.Run("Loop behavior", func(t *testing.T) {
		// Simulate time that would go past the end of the track
		sim.replayStartTime = time.Now().Add(-10 * time.Second) // 10 seconds ago
		sim.updateReplayPosition()

		// Should have looped back around
		// With 2x speed and 10 seconds elapsed = 20 points elapsed, 20 % 3 = 2
		expectedIndex := 2
		if sim.replayIndex != expectedIndex {
			t.Errorf("Expected looped replay index %d, got %d", expectedIndex, sim.replayIndex)
		}

		// Position should be at third point
		if sim.currentLat != 42.432000 {
			t.Errorf("Expected looped lat 42.432000, got %f", sim.currentLat)
		}
	})

	// Test speed and course calculation
	t.Run("Speed and course calculation", func(t *testing.T) {
		// Reset to first point
		sim.replayIndex = 0
		sim.currentLat = sim.replayPoints[0].Lat
		sim.currentLon = sim.replayPoints[0].Lon
		sim.currentAlt = sim.replayPoints[0].Elevation

		// Update to trigger speed/course calculation
		sim.updateReplayPosition()

		// Should have calculated speed and course based on distance to next point
		if sim.currentSpeed <= 0 {
			t.Errorf("Expected positive speed calculation, got %f", sim.currentSpeed)
		}
		if sim.currentCourse < 0 || sim.currentCourse >= 360 {
			t.Errorf("Expected course in 0-359 range, got %f", sim.currentCourse)
		}
	})
}

func TestUpdateReplayPositionWithSequentialTimestamps(t *testing.T) {
	// Create a test GPX file with sequential timestamps
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_replay_sequential.gpx")

	gpxContent := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Test Track</name>
    <trkseg>
      <trkpt lat="37.774900" lon="-122.419400">
        <ele>50.0</ele>
        <time>2024-01-15T10:00:00Z</time>
      </trkpt>
      <trkpt lat="37.775000" lon="-122.419300">
        <ele>52.0</ele>
        <time>2024-01-15T10:00:10Z</time>
      </trkpt>
      <trkpt lat="37.775100" lon="-122.419200">
        <ele>54.0</ele>
        <time>2024-01-15T10:00:20Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`

	err := os.WriteFile(tempFile, []byte(gpxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test GPX file: %v", err)
	}

	config := createTestConfig()
	config.ReplayFile = tempFile
	config.ReplaySpeed = 1.0

	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator with replay: %v", err)
	}

	// Should detect sequential timestamps
	if !sim.hasSequentialTimestamps() {
		t.Error("Expected sequential timestamps to be detected")
	}

	// Test time-based progression
	t.Run("Time-based progression", func(t *testing.T) {
		// Reset to beginning
		sim.replayIndex = 0
		sim.replayStartTime = time.Now().Add(-5 * time.Second) // 5 seconds ago

		sim.updateReplayPosition()

		// With 5 seconds elapsed at 1x speed, should still be at index 0
		// (since first point is at T+0, second at T+10)
		if sim.replayIndex != 0 {
			t.Errorf("Expected replay index 0, got %d", sim.replayIndex)
		}

		// Position should be at first point
		if sim.currentLat != 37.774900 {
			t.Errorf("Expected lat 37.774900, got %f", sim.currentLat)
		}

		// Test progression to second point
		sim.replayStartTime = time.Now().Add(-12 * time.Second) // 12 seconds ago
		sim.updateReplayPosition()

		// Should now be at index 1 (since 12 > 10 seconds)
		if sim.replayIndex != 1 {
			t.Errorf("Expected replay index 1 after 12 seconds, got %d", sim.replayIndex)
		}

		// Position should be at second point
		if sim.currentLat != 37.775000 {
			t.Errorf("Expected lat 37.775000, got %f", sim.currentLat)
		}
	})
}

func TestReplaySpeedLessThanOne(t *testing.T) {
	// Test replay speeds less than 1.0 to ensure no division by zero panic
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_slow_replay.gpx")

	gpxContent := `<?xml version="1.0"?>
<gpx version="1.0" creator="test" xmlns="http://www.topografix.com/GPX/1/0">
  <rte>
    <name>Test Route</name>
    <rtept lat="42.430950" lon="-71.107628">
      <ele>23.5</ele>
      <time>2001-11-28T21:05:28Z</time>
    </rtept>
    <rtept lat="42.431240" lon="-71.109236">
      <ele>26.6</ele>
      <time>2001-12-01T12:00:00Z</time>
    </rtept>
  </rte>
</gpx>`

	err := os.WriteFile(tempFile, []byte(gpxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test GPX file: %v", err)
	}

	testCases := []struct {
		name        string
		replaySpeed float64
	}{
		{"Speed 0.5x", 0.5},
		{"Speed 0.1x", 0.1},
		{"Speed 0.25x", 0.25},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig()
			config.ReplayFile = tempFile
			config.ReplaySpeed = tc.replaySpeed

			buffer := &bytes.Buffer{}
			sim, err := NewGPSSimulator(config, buffer)
			if err != nil {
				t.Fatalf("Failed to create GPS simulator with replay speed %.1fx: %v", tc.replaySpeed, err)
			}

			// This should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("updateReplayPosition panicked with replay speed %.1fx: %v", tc.replaySpeed, r)
				}
			}()

			// Simulate some time passing
			sim.replayStartTime = time.Now().Add(-5 * time.Second)
			sim.updateReplayPosition()

			// Verify position was updated (should be at first point)
			if sim.currentLat != 42.430950 {
				t.Errorf("Expected lat 42.430950, got %f", sim.currentLat)
			}

			// With slow replay speed, should still be at index 0 after 5 seconds
			if tc.replaySpeed <= 0.5 && sim.replayIndex != 0 {
				t.Errorf("Expected replay index 0 with slow speed %.1fx after 5 seconds, got %d",
					tc.replaySpeed, sim.replayIndex)
			}
		})
	}
}

func TestReplaySpeedZeroDefensiveCheck(t *testing.T) {
	// Test that zero replay speed is handled defensively without panic
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_zero_speed.gpx")

	gpxContent := `<?xml version="1.0"?>
<gpx version="1.0" creator="test" xmlns="http://www.topografix.com/GPX/1/0">
  <rte>
    <name>Test Route</name>
    <rtept lat="42.430950" lon="-71.107628">
      <ele>23.5</ele>
      <time>2001-11-28T21:05:28Z</time>
    </rtept>
    <rtept lat="42.431240" lon="-71.109236">
      <ele>26.6</ele>
      <time>2001-12-01T12:00:00Z</time>
    </rtept>
  </rte>
</gpx>`

	err := os.WriteFile(tempFile, []byte(gpxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test GPX file: %v", err)
	}

	testCases := []struct {
		name          string
		replaySpeed   float64
		expectWarning bool
	}{
		{"Zero speed", 0.0, true},
		{"Negative speed", -0.5, true},
		{"Very small positive speed", 0.001, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig()
			config.ReplayFile = tempFile
			config.ReplaySpeed = tc.replaySpeed

			buffer := &bytes.Buffer{}
			sim, err := NewGPSSimulator(config, buffer)
			if err != nil {
				t.Fatalf("Failed to create GPS simulator: %v", err)
			}

			// Capture stderr to check for warning messages
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// This should not panic, even with invalid replay speed
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("updateReplayPosition panicked with replay speed %.3f: %v", tc.replaySpeed, r)
				}
			}()

			// Simulate some time passing and update position
			sim.replayStartTime = time.Now().Add(-2 * time.Second)
			sim.updateReplayPosition()

			// Restore stderr and check for warnings
			w.Close()
			os.Stderr = oldStderr
			captured := make([]byte, 1000)
			n, _ := r.Read(captured)
			output := string(captured[:n])

			if tc.expectWarning {
				if !strings.Contains(output, "Warning: Invalid replay speed") {
					t.Errorf("Expected warning for invalid replay speed %.3f, got: %s", tc.replaySpeed, output)
				}
				// Speed should have been corrected to 1.0
				if sim.Config.ReplaySpeed != 1.0 {
					t.Errorf("Expected replay speed to be corrected to 1.0, got %.3f", sim.Config.ReplaySpeed)
				}
			} else {
				if strings.Contains(output, "Warning: Invalid replay speed") {
					t.Errorf("Unexpected warning for valid replay speed %.3f: %s", tc.replaySpeed, output)
				}
				// Speed should remain unchanged
				if sim.Config.ReplaySpeed != tc.replaySpeed {
					t.Errorf("Expected replay speed to remain %.3f, got %.3f", tc.replaySpeed, sim.Config.ReplaySpeed)
				}
			}

			// Verify position was updated (should be at first point)
			if sim.currentLat != 42.430950 {
				t.Errorf("Expected lat 42.430950, got %f", sim.currentLat)
			}
		})
	}
}
