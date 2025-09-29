package gps

import "time"

// Config holds all configuration options for the GPS simulator
type Config struct {
	Latitude       float64
	Longitude      float64
	Radius         float64 // in meters
	Altitude       float64 // starting altitude in meters
	Jitter         float64 // GPS jitter factor (0.0-1.0)
	AltitudeJitter float64 // altitude jitter factor (0.0-1.0)
	Speed          float64 // static speed in knots
	Course         float64 // static course in degrees (0-359)
	Satellites     int
	TimeToLock     time.Duration
	OutputRate     time.Duration
	SerialPort     string        // Serial port device (e.g., /dev/ttyUSB0, COM1)
	BaudRate       int           // Serial baud rate
	Quiet          bool          // Suppress informational messages
	GPXEnabled     bool          // Enable GPX file generation with timestamp filename
	GPXFile        string        // Generated GPX filename (internal use)
	Duration       time.Duration // How long to run the simulation (0 = run indefinitely)
	ReplayFile     string        // GPX file to replay (empty = normal simulation mode)
	ReplaySpeed    float64       // Replay speed multiplier (1.0 = real-time, 2.0 = 2x speed, etc.)
	ReplayLoop     bool          // Whether to loop the replay (false = stop after one pass, true = loop continuously)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		Latitude:       37.7749, // San Francisco
		Longitude:      -122.4194,
		Radius:         100.0,
		Altitude:       45.0,
		Jitter:         0.0,
		AltitudeJitter: 0.0,
		Speed:          0.0,
		Course:         0.0,
		Satellites:     8,
		TimeToLock:     2 * time.Second,
		OutputRate:     1 * time.Second,
		BaudRate:       9600,
		Quiet:          false,
		GPXEnabled:     false,
		Duration:       0,
		ReplaySpeed:    1.0,
		ReplayLoop:     false,
	}
}

// Validate checks if the configuration is valid and returns an error if not
func (c *Config) Validate() error {
	if c.Satellites < 4 || c.Satellites > 12 {
		return ErrInvalidSatelliteCount
	}
	if c.Radius < 0 {
		return ErrInvalidRadius
	}
	if c.Jitter < 0.0 || c.Jitter > 1.0 {
		return ErrInvalidJitter
	}
	if c.AltitudeJitter < 0.0 || c.AltitudeJitter > 1.0 {
		return ErrInvalidAltitudeJitter
	}
	if c.BaudRate <= 0 {
		return ErrInvalidBaudRate
	}
	if c.Speed < 0.0 {
		return ErrInvalidSpeed
	}
	if c.Course < 0.0 || c.Course >= 360.0 {
		return ErrInvalidCourse
	}
	if c.ReplaySpeed <= 0.0 {
		return ErrInvalidReplaySpeed
	}
	return nil
}
