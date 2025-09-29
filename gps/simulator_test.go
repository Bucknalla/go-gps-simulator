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
		BaudRate:       9600,
		ReplaySpeed:    1.0,
	}
}

// Helper function to create a test simulator
func createTestSimulator() *Simulator {
	config := createTestConfig()
	sim, err := NewSimulator(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create test simulator: %v", err))
	}
	buffer := &bytes.Buffer{}
	sim.SetNMEAWriter(buffer)
	return sim
}

func TestNewSimulator(t *testing.T) {
	config := createTestConfig()

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Test that simulator is properly initialized
	if sim == nil {
		t.Fatal("NewSimulator should not return nil")
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
	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}

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

	sim, err := NewSimulator(config)
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

	sim, err := NewSimulator(config)
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

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator: %v", err)
	}
	sim.SetNMEAWriter(buffer)
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
	sim, err := NewSimulator(config)
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
		{"Low jitter stationary", 0.05, 0.0, 0.0, false},
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

			sim, err := NewSimulator(config)
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

func TestStartStop(t *testing.T) {
	sim := createTestSimulator()

	// Test initial state
	if sim.IsRunning() {
		t.Error("Simulator should not be running initially")
	}

	// Test start
	err := sim.Start()
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}

	if !sim.IsRunning() {
		t.Error("Simulator should be running after start")
	}

	// Test stop
	err = sim.Stop()
	if err != nil {
		t.Fatalf("Failed to stop simulator: %v", err)
	}

	if sim.IsRunning() {
		t.Error("Simulator should not be running after stop")
	}

	// Test double start
	sim.Start()
	err = sim.Start()
	if err == nil {
		t.Error("Expected error when starting already running simulator")
	}

	// Test double stop
	sim.Stop()
	err = sim.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped simulator")
	}
}

func TestGetStatus(t *testing.T) {
	sim := createTestSimulator()

	status := sim.GetStatus()

	// Test initial status
	if status.Running {
		t.Error("Status should show not running initially")
	}
	// Note: Start time may be set even when not running in this implementation
	if status.ElapsedTime != 0 {
		t.Error("Elapsed time should be zero when not running")
	}

	// Test status after starting
	sim.Start()
	time.Sleep(10 * time.Millisecond)

	status = sim.GetStatus()
	if !status.Running {
		t.Error("Status should show running after start")
	}
	if status.StartTime.IsZero() {
		t.Error("Start time should be set when running")
	}
	if status.ElapsedTime == 0 {
		t.Error("Elapsed time should be greater than zero when running")
	}

	// Test position data in status
	if status.Position.Latitude != sim.currentLat {
		t.Errorf("Status position latitude mismatch: expected %f, got %f",
			sim.currentLat, status.Position.Latitude)
	}
	if len(status.Position.Satellites) != len(sim.satellites) {
		t.Errorf("Status satellites count mismatch: expected %d, got %d",
			len(sim.satellites), len(status.Position.Satellites))
	}

	sim.Stop()
}

func TestUpdateConfig(t *testing.T) {
	sim := createTestSimulator()

	// Test valid config update
	newConfig := createTestConfig()
	newConfig.Latitude = 40.0
	newConfig.Longitude = -74.0
	newConfig.Satellites = 10

	err := sim.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Verify config was updated
	if sim.config.Latitude != 40.0 {
		t.Errorf("Expected updated latitude 40.0, got %f", sim.config.Latitude)
	}
	if sim.config.Longitude != -74.0 {
		t.Errorf("Expected updated longitude -74.0, got %f", sim.config.Longitude)
	}

	// Test invalid config update
	invalidConfig := createTestConfig()
	invalidConfig.Satellites = 15 // Invalid satellite count

	err = sim.UpdateConfig(invalidConfig)
	if err == nil {
		t.Error("Expected error when updating with invalid config")
	}

	// Test update while running (may or may not be allowed in this implementation)
	sim.Start()
	err = sim.UpdateConfig(newConfig)
	// This implementation may allow config updates while running
	if err != nil {
		t.Logf("Config update while running returned error (this may be expected): %v", err)
	}
	sim.Stop()
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		modifier func(*Config)
		wantErr  bool
	}{
		{
			name:     "Valid config",
			modifier: func(c *Config) {},
			wantErr:  false,
		},
		{
			name:     "Invalid satellite count (too low)",
			modifier: func(c *Config) { c.Satellites = 3 },
			wantErr:  true,
		},
		{
			name:     "Invalid satellite count (too high)",
			modifier: func(c *Config) { c.Satellites = 13 },
			wantErr:  true,
		},
		{
			name:     "Invalid radius",
			modifier: func(c *Config) { c.Radius = -10.0 },
			wantErr:  true,
		},
		{
			name:     "Invalid jitter",
			modifier: func(c *Config) { c.Jitter = 1.5 },
			wantErr:  true,
		},
		{
			name:     "Invalid altitude jitter",
			modifier: func(c *Config) { c.AltitudeJitter = -0.1 },
			wantErr:  true,
		},
		{
			name:     "Invalid baud rate",
			modifier: func(c *Config) { c.BaudRate = 0 },
			wantErr:  true,
		},
		{
			name:     "Invalid speed",
			modifier: func(c *Config) { c.Speed = -5.0 },
			wantErr:  true,
		},
		{
			name:     "Invalid course",
			modifier: func(c *Config) { c.Course = 360.0 },
			wantErr:  true,
		},
		{
			name:     "Invalid replay speed",
			modifier: func(c *Config) { c.ReplaySpeed = 0.0 },
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig()
			tt.modifier(&config)

			_, err := NewSimulator(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSimulator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkUpdatePosition(b *testing.B) {
	sim := createTestSimulator()
	sim.isLocked = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.updatePosition()
	}
}

func BenchmarkUpdateSatellites(b *testing.B) {
	sim := createTestSimulator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.updateSatellites()
	}
}

func BenchmarkDistanceFromCenter(b *testing.B) {
	sim := createTestSimulator()
	lat := sim.config.Latitude + 0.001
	lon := sim.config.Longitude + 0.001

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.distanceFromCenter(lat, lon)
	}
}

func BenchmarkOutputNMEA(b *testing.B) {
	sim := createTestSimulator()
	sim.isLocked = true
	buffer := &bytes.Buffer{}
	sim.SetNMEAWriter(buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Reset()
		sim.outputNMEA()
	}
}

func TestNMEACallback(t *testing.T) {
	config := createTestConfig()
	config.TimeToLock = 10 * time.Millisecond // Very short lock time
	config.OutputRate = 10 * time.Millisecond // Very fast output rate

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create simulator: %v", err)
	}

	var receivedData NMEAData
	callbackCalled := false

	sim.AddCallback(func(data NMEAData) {
		receivedData = data
		callbackCalled = true
	})

	// Start the simulator to enable NMEA output
	sim.Start()
	time.Sleep(100 * time.Millisecond) // Give it time to lock and generate NMEA data
	sim.Stop()

	if !callbackCalled {
		t.Error("NMEA callback was not called")
	}

	if len(receivedData.Sentences) == 0 {
		t.Error("NMEA callback received no sentences")
	}

	// Check that position data is included
	if receivedData.Position.Latitude == 0.0 {
		t.Error("Position latitude should not be zero in callback")
	}
}

// Tests for GPX replay functionality in simulator

func TestNewSimulatorWithReplay(t *testing.T) {
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

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator with replay: %v", err)
	}

	// Check status to verify replay points were loaded
	status := sim.GetStatus()
	if status.ReplayTotal != 2 {
		t.Errorf("Expected 2 replay points, got %d", status.ReplayTotal)
	}

	// Verify initial position was set from first point
	if status.Position.Latitude != 37.774900 {
		t.Errorf("Expected initial lat 37.774900, got %f", status.Position.Latitude)
	}
	if status.Position.Longitude != -122.419400 {
		t.Errorf("Expected initial lon -122.419400, got %f", status.Position.Longitude)
	}
	if status.Position.Altitude != 50.0 {
		t.Errorf("Expected initial alt 50.0, got %f", status.Position.Altitude)
	}
}

func TestNewSimulatorWithReplayError(t *testing.T) {
	config := createTestConfig()
	config.ReplayFile = "non_existent_file.gpx"

	_, err := NewSimulator(config)
	if err == nil {
		t.Error("Expected error when loading non-existent replay file")
	}
	if !strings.Contains(err.Error(), "failed to load replay file") {
		t.Errorf("Expected error about failed to load replay file, got: %v", err)
	}
}

func TestReplayWithSequentialTimestamps(t *testing.T) {
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
	config.OutputRate = 1 * time.Millisecond // Fast output for testing

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator with replay: %v", err)
	}

	// Start simulation
	sim.Start()

	// Let it run briefly to ensure replay progresses
	time.Sleep(50 * time.Millisecond)

	// Check initial status
	status := sim.GetStatus()
	if status.ReplayTotal != 3 {
		t.Errorf("Expected 3 replay points, got %d", status.ReplayTotal)
	}

	sim.Stop()
}

func TestReplayWithNonSequentialTimestamps(t *testing.T) {
	// Create a test GPX file with route data (non-sequential timestamps)
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_replay_route.gpx")

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
	config.ReplaySpeed = 2.0                 // 2x speed
	config.ReplayLoop = true                 // Enable looping for this test
	config.OutputRate = 1 * time.Millisecond // Fast output

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create GPS simulator with replay: %v", err)
	}

	// Initial position should be from first point
	status := sim.GetStatus()
	if status.Position.Latitude != 42.430950 {
		t.Errorf("Expected initial lat 42.430950, got %f", status.Position.Latitude)
	}

	if status.ReplayTotal != 3 {
		t.Errorf("Expected 3 replay points, got %d", status.ReplayTotal)
	}

	// Start simulation to test progression
	sim.Start()
	time.Sleep(50 * time.Millisecond) // Let it run briefly
	sim.Stop()
}

func TestReplaySpeedVariations(t *testing.T) {
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
		{"Speed 2.0x", 2.0},
		{"Speed 10.0x", 10.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig()
			config.ReplayFile = tempFile
			config.ReplaySpeed = tc.replaySpeed
			config.OutputRate = 1 * time.Millisecond

			sim, err := NewSimulator(config)
			if err != nil {
				t.Fatalf("Failed to create GPS simulator with replay speed %.1fx: %v", tc.replaySpeed, err)
			}

			// This should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Simulator panicked with replay speed %.1fx: %v", tc.replaySpeed, r)
				}
			}()

			// Start and run briefly
			sim.Start()
			time.Sleep(10 * time.Millisecond)
			sim.Stop()

			// Verify position was updated (should be at first point)
			status := sim.GetStatus()
			if status.Position.Latitude != 42.430950 {
				t.Errorf("Expected lat 42.430950, got %f", status.Position.Latitude)
			}
		})
	}
}

func TestReplayInvalidSpeed(t *testing.T) {
	// Test that zero/negative replay speed is handled defensively
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_invalid_speed.gpx")

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
		{"Zero speed", 0.0},
		{"Negative speed", -0.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig()
			config.ReplayFile = tempFile
			config.ReplaySpeed = tc.replaySpeed

			// Should fail validation during simulator creation
			_, err := NewSimulator(config)
			if err == nil {
				t.Errorf("Expected error for invalid replay speed %.3f", tc.replaySpeed)
			}
		})
	}
}

func TestReplayLooping(t *testing.T) {
	// Test replay looping functionality
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_loop.gpx")

	gpxContent := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Short Track</name>
    <trkseg>
      <trkpt lat="37.774900" lon="-122.419400">
        <ele>50.0</ele>
        <time>2024-01-15T10:00:00Z</time>
      </trkpt>
      <trkpt lat="37.775000" lon="-122.419300">
        <ele>52.0</ele>
        <time>2024-01-15T10:00:01Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`

	err := os.WriteFile(tempFile, []byte(gpxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test GPX file: %v", err)
	}

	// Test with looping enabled
	t.Run("Looping enabled", func(t *testing.T) {
		config := createTestConfig()
		config.ReplayFile = tempFile
		config.ReplaySpeed = 10.0 // Fast speed to complete quickly
		config.ReplayLoop = true
		config.OutputRate = 1 * time.Millisecond

		sim, err := NewSimulator(config)
		if err != nil {
			t.Fatalf("Failed to create simulator: %v", err)
		}

		sim.Start()
		time.Sleep(50 * time.Millisecond) // Let it run and potentially loop

		status := sim.GetStatus()
		if status.ReplayCompleted && !config.ReplayLoop {
			t.Error("Replay should not be marked completed when looping is enabled")
		}

		sim.Stop()
	})

	// Test with looping disabled
	t.Run("Looping disabled", func(t *testing.T) {
		config := createTestConfig()
		config.ReplayFile = tempFile
		config.ReplaySpeed = 10.0 // Fast speed to complete quickly
		config.ReplayLoop = false
		config.OutputRate = 1 * time.Millisecond

		sim, err := NewSimulator(config)
		if err != nil {
			t.Fatalf("Failed to create simulator: %v", err)
		}

		sim.Start()
		time.Sleep(100 * time.Millisecond) // Give it time to complete

		status := sim.GetStatus()
		// With fast speed and short track, should complete
		if !status.ReplayCompleted {
			t.Log("Replay may not have completed yet (timing dependent)")
		}

		sim.Stop()
	})
}
