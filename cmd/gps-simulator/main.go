package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go.bug.st/serial"
	"github.com/Bucknalla/go-gps-simulator/gps"
)

// Version information - populated at build time via ldflags
var (
	Version   = "dev"     // Will be set to git tag if available, otherwise "dev"
	Commit    = "unknown" // Will be set to git commit hash
	BuildDate = "unknown" // Will be set to build timestamp
)

func main() {
	var config gps.Config
	var showVersion bool

	// Define command line flags
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.Float64Var(&config.Latitude, "lat", 37.7749, "Initial latitude (decimal degrees)")
	flag.Float64Var(&config.Longitude, "lon", -122.4194, "Initial longitude (decimal degrees)")
	flag.Float64Var(&config.Radius, "radius", 100.0, "Wandering radius in meters")
	flag.Float64Var(&config.Altitude, "altitude", 45.0, "Starting altitude in meters")
	flag.Float64Var(&config.Jitter, "jitter", 0.0, "GPS position jitter factor (0.0=stable, 1.0=high jitter)")
	flag.Float64Var(&config.AltitudeJitter, "altitude-jitter", 0.0, "Altitude jitter factor (0.0=stable, 1.0=high variation)")
	flag.Float64Var(&config.Speed, "speed", 0.0, "Static speed in knots")
	flag.Float64Var(&config.Course, "course", 0.0, "Static course in degrees (0-359)")
	flag.IntVar(&config.Satellites, "satellites", 8, "Number of satellites to simulate (4-12)")
	flag.DurationVar(&config.TimeToLock, "lock-time", 2*time.Second, "Time to GPS lock simulation")
	flag.DurationVar(&config.OutputRate, "rate", 1*time.Second, "NMEA output rate")
	flag.StringVar(&config.SerialPort, "serial", "", "Serial port for NMEA output (e.g., /dev/ttyUSB0, COM1)")
	flag.IntVar(&config.BaudRate, "baud", 9600, "Serial port baud rate")
	flag.BoolVar(&config.Quiet, "quiet", false, "Suppress info messages (only output NMEA data)")
	flag.BoolVar(&config.GPXEnabled, "gpx", false, "Generate GPX track file with timestamp-based filename")
	flag.DurationVar(&config.Duration, "duration", 0, "How long to run the simulation (e.g., 30s, 5m, 1h). Default is indefinite")
	flag.StringVar(&config.ReplayFile, "replay", "", "GPX file to replay instead of simulating (e.g., track.gpx)")
	flag.Float64Var(&config.ReplaySpeed, "replay-speed", 1.0, "Replay speed multiplier (1.0=real-time, 2.0=2x speed, 0.5=half speed)")
	flag.BoolVar(&config.ReplayLoop, "replay-loop", false, "Loop the GPX replay continuously (default: stop after one pass)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nGPS NMEA0183 Simulator\n")
		fmt.Fprintf(os.Stderr, "Simulates a GPS receiver outputting NMEA sentences with configurable parameters.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Handle version flag
	if showVersion {
		if Version != "dev" {
			fmt.Printf("v%s\n", Version)
		} else {
			fmt.Printf("%s\n", Commit)
		}
		os.Exit(0)
	}

	// Validate input parameters
	if config.Satellites < 4 || config.Satellites > 12 {
		log.Fatal("Number of satellites must be between 4 and 12")
	}

	if config.Radius < 0 {
		log.Fatal("Radius must be positive")
	}

	if config.Jitter < 0.0 || config.Jitter > 1.0 {
		log.Fatal("Jitter must be between 0.0 and 1.0")
	}

	if config.AltitudeJitter < 0.0 || config.AltitudeJitter > 1.0 {
		log.Fatal("Altitude jitter must be between 0.0 and 1.0")
	}

	if config.BaudRate <= 0 {
		log.Fatal("Baud rate must be positive")
	}

	if config.Speed < 0.0 {
		log.Fatal("Speed must be non-negative")
	}

	if config.Course < 0.0 || config.Course >= 360.0 {
		log.Fatal("Course must be between 0.0 and 359.9 degrees")
	}

	if config.ReplaySpeed <= 0.0 {
		log.Fatal("Replay speed must be positive")
	}

	// Handle GPX filename generation and validation
	if config.GPXEnabled {
		// Require duration when GPX is enabled
		if config.Duration <= 0 {
			log.Fatal("Duration greater than 0 must be specified when using -gpx flag (e.g., -duration 30s)")
		}
		// Always generate timestamp-based filename when -gpx flag is used
		config.GPXFile = fmt.Sprintf("%s.gpx", time.Now().Format("20060102_150405"))
	}

	// Setup output writer (serial port or stdout)
	var nmeaWriter io.Writer = os.Stdout
	var serialPort serial.Port

	if config.SerialPort != "" {
		mode := &serial.Mode{
			BaudRate: config.BaudRate,
			Parity:   serial.NoParity,
			DataBits: 8,
			StopBits: serial.OneStopBit,
		}

		var err error
		serialPort, err = serial.Open(config.SerialPort, mode)
		if err != nil {
			log.Fatalf("Failed to open serial port %s: %v", config.SerialPort, err)
		}
		defer serialPort.Close()
		nmeaWriter = serialPort

		if !config.Quiet {
			fmt.Fprintf(os.Stderr, "Opened serial port: %s at %d baud\n", config.SerialPort, config.BaudRate)
		}
	}

	// Log to stderr so it doesn't interfere with NMEA output
	if !config.Quiet {
		if config.ReplayFile != "" {
			fmt.Fprintf(os.Stderr, "Starting GPS replay from: %s\n", config.ReplayFile)
			fmt.Fprintf(os.Stderr, "Replay speed: %.1fx\n", config.ReplaySpeed)
		} else {
			fmt.Fprintf(os.Stderr, "Starting GPS simulator...\n")
			fmt.Fprintf(os.Stderr, "Initial position: %.6f, %.6f, %.1fm\n", config.Latitude, config.Longitude, config.Altitude)
			fmt.Fprintf(os.Stderr, "Wandering radius: %.1f meters\n", config.Radius)
			fmt.Fprintf(os.Stderr, "GPS jitter: %.1f (%.0f%% jitter)\n", config.Jitter, config.Jitter*100)
			fmt.Fprintf(os.Stderr, "Altitude jitter: %.1f (%.0f%% variation)\n", config.AltitudeJitter, config.AltitudeJitter*100)
			fmt.Fprintf(os.Stderr, "Speed: %.1f knots\n", config.Speed)
			fmt.Fprintf(os.Stderr, "Course: %.1f degrees\n", config.Course)
		}
		fmt.Fprintf(os.Stderr, "Satellites: %d\n", config.Satellites)
		fmt.Fprintf(os.Stderr, "Time to lock: %v\n", config.TimeToLock)
		fmt.Fprintf(os.Stderr, "Output rate: %v\n", config.OutputRate)
		if config.SerialPort != "" {
			fmt.Fprintf(os.Stderr, "NMEA output: %s (%d baud)\n", config.SerialPort, config.BaudRate)
		} else {
			fmt.Fprintf(os.Stderr, "NMEA output: stdout\n")
		}
		fmt.Fprintf(os.Stderr, "\nPress Ctrl+C to stop\n\n")
	}

	// Start GPS simulation
	simulator, err := gps.NewGPSSimulator(config, nmeaWriter)
	if err != nil {
		log.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Show GPX file info if enabled
	if config.GPXEnabled && !config.Quiet {
		fmt.Fprintf(os.Stderr, "GPX output: %s\n", config.GPXFile)
	}

	simulator.Run()
}
