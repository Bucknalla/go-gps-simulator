package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Bucknalla/go-gps-simulator/gps"
	"go.bug.st/serial"
)

// Version information - populated at build time via ldflags
var (
	Version   = "dev"     // Will be set to git tag if available, otherwise "dev"
	Commit    = "unknown" // Will be set to git commit hash
	BuildDate = "unknown" // Will be set to build timestamp
)

// Config is now imported from the gps package
type Config = gps.Config

func main() {
	// Start with default config and override with flags
	config := gps.DefaultConfig()
	var showVersion bool

	// Define command line flags
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.Float64Var(&config.Latitude, "lat", config.Latitude, "Initial latitude (decimal degrees)")
	flag.Float64Var(&config.Longitude, "lon", config.Longitude, "Initial longitude (decimal degrees)")
	flag.Float64Var(&config.Radius, "radius", config.Radius, "Wandering radius in meters")
	flag.Float64Var(&config.Altitude, "altitude", config.Altitude, "Starting altitude in meters")
	flag.Float64Var(&config.Jitter, "jitter", config.Jitter, "GPS position jitter factor (0.0=stable, 1.0=high jitter)")
	flag.Float64Var(&config.AltitudeJitter, "altitude-jitter", config.AltitudeJitter, "Altitude jitter factor (0.0=stable, 1.0=high variation)")
	flag.Float64Var(&config.Speed, "speed", config.Speed, "Static speed in knots")
	flag.Float64Var(&config.Course, "course", config.Course, "Static course in degrees (0-359)")
	flag.IntVar(&config.Satellites, "satellites", config.Satellites, "Number of satellites to simulate (4-12)")
	flag.DurationVar(&config.TimeToLock, "lock-time", config.TimeToLock, "Time to GPS lock simulation")
	flag.DurationVar(&config.OutputRate, "rate", config.OutputRate, "NMEA output rate")
	flag.StringVar(&config.SerialPort, "serial", config.SerialPort, "Serial port for NMEA output (e.g., /dev/ttyUSB0, COM1)")
	flag.IntVar(&config.BaudRate, "baud", config.BaudRate, "Serial port baud rate")
	flag.BoolVar(&config.Quiet, "quiet", config.Quiet, "Suppress info messages (only output NMEA data)")
	flag.BoolVar(&config.GPXEnabled, "gpx", config.GPXEnabled, "Generate GPX track file with timestamp-based filename")
	flag.DurationVar(&config.Duration, "duration", config.Duration, "How long to run the simulation (e.g., 30s, 5m, 1h). Default is indefinite")
	flag.StringVar(&config.ReplayFile, "replay", config.ReplayFile, "GPX file to replay instead of simulating (e.g., track.gpx)")
	flag.Float64Var(&config.ReplaySpeed, "replay-speed", config.ReplaySpeed, "Replay speed multiplier (1.0=real-time, 2.0=2x speed, 0.5=half speed)")
	flag.BoolVar(&config.ReplayLoop, "replay-loop", config.ReplayLoop, "Loop the GPX replay continuously (default: stop after one pass)")

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

	// Validate config using the library's validation
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
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

	// Create GPS simulator using the new library
	simulator, err := gps.NewSimulator(config)
	if err != nil {
		log.Fatalf("Failed to create GPS simulator: %v", err)
	}

	// Set the NMEA writer
	simulator.SetNMEAWriter(nmeaWriter)

	// Show GPX file info if enabled
	if config.GPXEnabled && !config.Quiet {
		fmt.Fprintf(os.Stderr, "GPX output: %s\n", config.GPXFile)
	}

	// Start the simulator
	if err := simulator.Start(); err != nil {
		log.Fatalf("Failed to start GPS simulator: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either the simulator to finish (if duration is set) or a signal
	if config.Duration > 0 {
		// If duration is set, wait for that duration or a signal
		select {
		case <-time.After(config.Duration):
			// Duration elapsed, simulator should have stopped itself
		case <-sigChan:
			// Signal received, stop the simulator
			if err := simulator.Stop(); err != nil {
				log.Printf("Error stopping simulator: %v", err)
			}
		}
	} else {
		// No duration set, run indefinitely until signal
		<-sigChan
		if err := simulator.Stop(); err != nil {
			log.Printf("Error stopping simulator: %v", err)
		}
	}

	// Wait a bit for the simulator to cleanly shut down
	time.Sleep(100 * time.Millisecond)
}
