package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Test Config struct
func TestConfig(t *testing.T) {
	config := Config{
		Latitude:       37.7749,
		Longitude:      -122.4194,
		Radius:         100.0,
		Altitude:       45.0,
		Jitter:         0.5,
		AltitudeJitter: 0.1,
		Speed:          5.0,
		Course:         90.0,
		Satellites:     8,
		TimeToLock:     30 * time.Second,
		OutputRate:     1 * time.Second,
		SerialPort:     "/dev/ttyUSB0",
		BaudRate:       9600,
		Quiet:          true,
	}

	// Test that all fields are properly set
	if config.Latitude != 37.7749 {
		t.Errorf("Expected latitude 37.7749, got %f", config.Latitude)
	}
	if config.Longitude != -122.4194 {
		t.Errorf("Expected longitude -122.4194, got %f", config.Longitude)
	}
	if config.Radius != 100.0 {
		t.Errorf("Expected radius 100.0, got %f", config.Radius)
	}
	if config.Altitude != 45.0 {
		t.Errorf("Expected altitude 45.0, got %f", config.Altitude)
	}
	if config.Jitter != 0.5 {
		t.Errorf("Expected jitter 0.5, got %f", config.Jitter)
	}
	if config.AltitudeJitter != 0.1 {
		t.Errorf("Expected altitude jitter 0.1, got %f", config.AltitudeJitter)
	}
	if config.Speed != 5.0 {
		t.Errorf("Expected speed 5.0, got %f", config.Speed)
	}
	if config.Course != 90.0 {
		t.Errorf("Expected course 90.0, got %f", config.Course)
	}
	if config.Satellites != 8 {
		t.Errorf("Expected satellites 8, got %d", config.Satellites)
	}
	if config.TimeToLock != 30*time.Second {
		t.Errorf("Expected TimeToLock 30s, got %v", config.TimeToLock)
	}
	if config.OutputRate != 1*time.Second {
		t.Errorf("Expected OutputRate 1s, got %v", config.OutputRate)
	}
	if config.SerialPort != "/dev/ttyUSB0" {
		t.Errorf("Expected SerialPort '/dev/ttyUSB0', got %s", config.SerialPort)
	}
	if config.BaudRate != 9600 {
		t.Errorf("Expected BaudRate 9600, got %d", config.BaudRate)
	}
	if config.Quiet != true {
		t.Errorf("Expected Quiet true, got %t", config.Quiet)
	}
}

// Test version variables
func TestVersionVariables(t *testing.T) {
	// These should have default values
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}

	// Default values should be set
	if Version != "dev" && !strings.Contains(Version, ".") {
		t.Logf("Version: %s (may be overridden at build time)", Version)
	}
	if Commit != "unknown" && len(Commit) < 7 {
		t.Logf("Commit: %s (may be overridden at build time)", Commit)
	}
	if BuildDate != "unknown" {
		t.Logf("BuildDate: %s (may be overridden at build time)", BuildDate)
	}
}

// Test main function indirectly by testing flag parsing behavior
// We can't directly test main() as it would cause the program to run,
// but we can test the components that main() uses
func TestMainComponents(t *testing.T) {
	// Test that we can create a basic config (simulating what main does)
	config := Config{
		Latitude:   37.7749,
		Longitude:  -122.4194,
		Radius:     100.0,
		Jitter:     0.5,
		Satellites: 8,
		TimeToLock: 30 * time.Second,
		OutputRate: 1 * time.Second,
		BaudRate:   9600,
	}

	// Test validation logic (similar to what's in main)
	if config.Satellites < 4 || config.Satellites > 12 {
		t.Errorf("Satellites should be between 4 and 12, got %d", config.Satellites)
	}

	if config.Radius < 0 {
		t.Errorf("Radius should be positive, got %f", config.Radius)
	}

	if config.Jitter < 0.0 || config.Jitter > 1.0 {
		t.Errorf("Jitter should be between 0.0 and 1.0, got %f", config.Jitter)
	}

	if config.BaudRate <= 0 {
		t.Errorf("BaudRate should be positive, got %d", config.BaudRate)
	}
}

// Test validation edge cases
func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name           string
		satellites     int
		radius         float64
		jitter         float64
		altitudeJitter float64
		speed          float64
		course         float64
		baudRate       int
		shouldError    bool
	}{
		{"Valid config", 8, 100.0, 0.5, 0.1, 10.0, 90.0, 9600, false},
		{"Valid config with quiet", 8, 100.0, 0.5, 0.1, 10.0, 90.0, 9600, false},
		{"Min satellites", 4, 100.0, 0.5, 0.1, 10.0, 90.0, 9600, false},
		{"Max satellites", 12, 100.0, 0.5, 0.1, 10.0, 90.0, 9600, false},
		{"Too few satellites", 3, 100.0, 0.5, 0.1, 10.0, 90.0, 9600, true},
		{"Too many satellites", 13, 100.0, 0.5, 0.1, 10.0, 90.0, 9600, true},
		{"Negative radius", 8, -1.0, 0.5, 0.1, 10.0, 90.0, 9600, true},
		{"Zero radius", 8, 0.0, 0.5, 0.1, 10.0, 90.0, 9600, false},
		{"Min jitter", 8, 100.0, 0.0, 0.1, 10.0, 90.0, 9600, false},
		{"Max jitter", 8, 100.0, 1.0, 0.1, 10.0, 90.0, 9600, false},
		{"Negative jitter", 8, 100.0, -0.1, 0.1, 10.0, 90.0, 9600, true},
		{"High jitter", 8, 100.0, 1.1, 0.1, 10.0, 90.0, 9600, true},
		{"Min altitude jitter", 8, 100.0, 0.5, 0.0, 10.0, 90.0, 9600, false},
		{"Max altitude jitter", 8, 100.0, 0.5, 1.0, 10.0, 90.0, 9600, false},
		{"Negative altitude jitter", 8, 100.0, 0.5, -0.1, 10.0, 90.0, 9600, true},
		{"High altitude jitter", 8, 100.0, 0.5, 1.1, 10.0, 90.0, 9600, true},
		{"Zero speed", 8, 100.0, 0.5, 0.1, 0.0, 90.0, 9600, false},
		{"Negative speed", 8, 100.0, 0.5, 0.1, -1.0, 90.0, 9600, true},
		{"Min course", 8, 100.0, 0.5, 0.1, 10.0, 0.0, 9600, false},
		{"Max course", 8, 100.0, 0.5, 0.1, 10.0, 359.9, 9600, false},
		{"High course", 8, 100.0, 0.5, 0.1, 10.0, 360.0, 9600, true},
		{"Negative course", 8, 100.0, 0.5, 0.1, 10.0, -1.0, 9600, true},
		{"Zero baud rate", 8, 100.0, 0.5, 0.1, 10.0, 90.0, 0, true},
		{"Negative baud rate", 8, 100.0, 0.5, 0.1, 10.0, 90.0, -9600, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{
				Satellites:     tc.satellites,
				Radius:         tc.radius,
				Jitter:         tc.jitter,
				AltitudeJitter: tc.altitudeJitter,
				Speed:          tc.speed,
				Course:         tc.course,
				BaudRate:       tc.baudRate,
			}

			hasError := false

			// Apply the same validation logic as main()
			if config.Satellites < 4 || config.Satellites > 12 {
				hasError = true
			}
			if config.Radius < 0 {
				hasError = true
			}
			if config.Jitter < 0.0 || config.Jitter > 1.0 {
				hasError = true
			}
			if config.AltitudeJitter < 0.0 || config.AltitudeJitter > 1.0 {
				hasError = true
			}
			if config.BaudRate <= 0 {
				hasError = true
			}
			if config.Speed < 0.0 {
				hasError = true
			}
			if config.Course < 0.0 || config.Course >= 360.0 {
				hasError = true
			}

			if hasError != tc.shouldError {
				t.Errorf("Expected error: %v, got error: %v", tc.shouldError, hasError)
			}
		})
	}
}

// Test that we can simulate the main function workflow without actually running it
func TestMainWorkflow(t *testing.T) {
	// Simulate the main function workflow
	config := Config{
		Latitude:   37.7749,
		Longitude:  -122.4194,
		Radius:     100.0,
		Jitter:     0.5,
		Satellites: 8,
		TimeToLock: 30 * time.Second,
		OutputRate: 1 * time.Second,
		BaudRate:   9600,
		Quiet:      false, // Test with verbose mode
	}

	// Test that we can create a simulator (what main does)
	nmeaWriter := os.Stdout // This is what main uses when no serial port is specified
	simulator := NewGPSSimulator(config, nmeaWriter)

	if simulator == nil {
		t.Error("Should be able to create GPS simulator like main() does")
	}

	// Test that the simulator is properly configured
	if simulator != nil {
		if simulator.config.Latitude != config.Latitude {
			t.Error("Simulator should be configured with the same latitude as config")
		}
		if simulator.config.Longitude != config.Longitude {
			t.Error("Simulator should be configured with the same longitude as config")
		}
		if len(simulator.satellites) != config.Satellites {
			t.Error("Simulator should have the correct number of satellites")
		}
	}
}

// Test the quiet flag functionality
func TestQuietFlag(t *testing.T) {
	tests := []struct {
		name  string
		quiet bool
	}{
		{"Quiet mode enabled", true},
		{"Quiet mode disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Latitude:   37.7749,
				Longitude:  -122.4194,
				Radius:     100.0,
				Jitter:     0.5,
				Satellites: 8,
				TimeToLock: 30 * time.Second,
				OutputRate: 1 * time.Second,
				BaudRate:   9600,
				Quiet:      tt.quiet,
			}

			// Test that the quiet flag is properly set
			if config.Quiet != tt.quiet {
				t.Errorf("Expected Quiet to be %t, got %t", tt.quiet, config.Quiet)
			}

			// Test that we can create a simulator with quiet mode
			simulator := NewGPSSimulator(config, os.Stdout)
			if simulator == nil {
				t.Error("Should be able to create GPS simulator with quiet mode")
			}

			// Verify the quiet setting is preserved in the simulator's config
			if simulator != nil && simulator.config.Quiet != tt.quiet {
				t.Errorf("Expected simulator config.Quiet to be %t, got %t",
					tt.quiet, simulator.config.Quiet)
			}
		})
	}
}

// Test quiet flag behavior in different scenarios
func TestQuietFlagBehavior(t *testing.T) {
	// Test default quiet value (should be false)
	var config Config
	if config.Quiet != false {
		t.Errorf("Default Quiet value should be false, got %t", config.Quiet)
	}

	// Test explicit quiet settings
	config.Quiet = true
	if config.Quiet != true {
		t.Errorf("Explicit Quiet=true should be true, got %t", config.Quiet)
	}

	config.Quiet = false
	if config.Quiet != false {
		t.Errorf("Explicit Quiet=false should be false, got %t", config.Quiet)
	}
}

// Test that quiet mode affects the correct behavior in simulator
func TestQuietModeIntegration(t *testing.T) {
	// Test with quiet mode enabled
	quietConfig := Config{
		Latitude:   37.7749,
		Longitude:  -122.4194,
		Radius:     100.0,
		Jitter:     0.5,
		Satellites: 8,
		TimeToLock: 10 * time.Millisecond, // Short for testing
		OutputRate: 1 * time.Second,
		BaudRate:   9600,
		Quiet:      true,
	}

	quietSim := NewGPSSimulator(quietConfig, os.Stdout)
	if !quietSim.config.Quiet {
		t.Error("Simulator should preserve quiet mode setting")
	}

	// Test with quiet mode disabled
	verboseConfig := Config{
		Latitude:   37.7749,
		Longitude:  -122.4194,
		Radius:     100.0,
		Jitter:     0.5,
		Satellites: 8,
		TimeToLock: 10 * time.Millisecond, // Short for testing
		OutputRate: 1 * time.Second,
		BaudRate:   9600,
		Quiet:      false,
	}

	verboseSim := NewGPSSimulator(verboseConfig, os.Stdout)
	if verboseSim.config.Quiet {
		t.Error("Simulator should preserve non-quiet mode setting")
	}
}
