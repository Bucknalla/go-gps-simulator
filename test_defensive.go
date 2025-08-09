package main

import (
	"bytes"
	"fmt"
)

func main() {
	// Test defensive check by directly creating simulator with zero replay speed
	config := Config{
		Latitude:       37.7749,
		Longitude:      -122.4194,
		Altitude:       100.0,
		Speed:          5.0,
		Course:         90.0,
		Radius:         1000.0,
		Jitter:         0.1,
		AltitudeJitter: 0.1,
		Satellites:     8,
		TimeToLock:     1000000, // Very long to prevent GPS lock
		OutputRate:     1000000,
		ReplayFile:     "fells_loop.gpx",
		ReplaySpeed:    0.0, // This should trigger defensive check
	}

	buffer := &bytes.Buffer{}
	sim, err := NewGPSSimulator(config, buffer)
	if err != nil {
		fmt.Printf("Error creating simulator: %v\n", err)
		return
	}
	defer sim.Close()

	// Force GPS lock to trigger replay position update
	sim.isLocked = true

	fmt.Println("Calling updateReplayPosition with zero replay speed...")
	sim.updateReplayPosition() // This should not panic and should show warning
	fmt.Printf("Success! Replay speed corrected to: %.1f\n", sim.config.ReplaySpeed)
}
