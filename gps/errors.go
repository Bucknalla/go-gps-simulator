package gps

import "errors"

// Common errors returned by the GPS simulator
var (
	ErrInvalidSatelliteCount   = errors.New("number of satellites must be between 4 and 12")
	ErrInvalidRadius           = errors.New("radius must be positive")
	ErrInvalidJitter           = errors.New("jitter must be between 0.0 and 1.0")
	ErrInvalidAltitudeJitter   = errors.New("altitude jitter must be between 0.0 and 1.0")
	ErrInvalidBaudRate         = errors.New("baud rate must be positive")
	ErrInvalidSpeed            = errors.New("speed must be non-negative")
	ErrInvalidCourse           = errors.New("course must be between 0.0 and 359.9 degrees")
	ErrInvalidReplaySpeed      = errors.New("replay speed must be positive")
	ErrSimulatorNotRunning     = errors.New("simulator is not running")
	ErrSimulatorAlreadyRunning = errors.New("simulator is already running")
)
