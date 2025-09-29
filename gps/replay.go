package gps

import (
	"time"
)

// updateReplayPosition updates position based on GPX replay data
func (s *Simulator) updateReplayPosition() {
	if len(s.replayPoints) == 0 {
		return
	}

	// Defensive check for invalid replay speed
	if s.config.ReplaySpeed <= 0 {
		s.config.ReplaySpeed = 1.0
	}

	now := time.Now()
	elapsedTime := now.Sub(s.replayStartTime)

	// Apply replay speed multiplier
	adjustedTime := time.Duration(float64(elapsedTime) * s.config.ReplaySpeed)

	// Check if timestamps are sequential for time-based progression
	useTimestamps := s.hasSequentialTimestamps()

	if useTimestamps {
		// Time-based progression using GPX timestamps
		targetTime := s.replayPoints[0].Time.Add(adjustedTime)

		// Find the track point that should be active at this time
		newIndex := 0
		for i := 0; i < len(s.replayPoints); i++ {
			if targetTime.After(s.replayPoints[i].Time) || targetTime.Equal(s.replayPoints[i].Time) {
				newIndex = i
			} else {
				break
			}
		}

		// If target time is past the last timestamp, we've completed the replay
		if targetTime.After(s.replayPoints[len(s.replayPoints)-1].Time) {
			newIndex = len(s.replayPoints) // This will trigger completion check
		}

		s.replayIndex = newIndex
	} else {
		// Index-based progression when timestamps are not sequential
		// Progress through points at a steady rate (1 point per second at 1x speed)
		pointInterval := time.Duration(float64(time.Second) / s.config.ReplaySpeed)
		pointsSinceStart := int(elapsedTime / pointInterval)

		if s.config.ReplayLoop {
			s.replayIndex = pointsSinceStart % len(s.replayPoints)
		} else {
			s.replayIndex = pointsSinceStart
		}
	}

	// If we've reached the end, handle completion/looping
	if s.replayIndex >= len(s.replayPoints) {
		s.replayCompleted = true
		if s.config.ReplayLoop {
			// Loop back to start if looping is enabled
			s.replayIndex = 0
			s.replayStartTime = now
		}
		return
	}

	// Update current position from track point
	currentPoint := s.replayPoints[s.replayIndex]
	s.currentLat = currentPoint.Lat
	s.currentLon = currentPoint.Lon
	s.currentAlt = currentPoint.Elevation

	// Calculate speed and course from next point if available
	if s.replayIndex < len(s.replayPoints)-1 {
		nextPoint := s.replayPoints[s.replayIndex+1]

		// Calculate distance and time between points
		distance := s.calculateDistance(s.currentLat, s.currentLon, nextPoint.Lat, nextPoint.Lon)

		var timeDiff float64
		if useTimestamps {
			timeDiff = nextPoint.Time.Sub(currentPoint.Time).Seconds()
		} else {
			// Use a fixed time interval for non-sequential timestamps
			timeDiff = 1.0 // 1 second between points
		}

		if timeDiff > 0 {
			// Convert m/s to knots (1 m/s = 1.94384 knots)
			s.currentSpeed = (distance / timeDiff) * 1.94384

			// Calculate course (bearing) to next point
			s.currentCourse = s.calculateBearing(s.currentLat, s.currentLon, nextPoint.Lat, nextPoint.Lon)
		}
	}
}

// hasSequentialTimestamps checks if the replay points have sequential timestamps
func (s *Simulator) hasSequentialTimestamps() bool {
	if len(s.replayPoints) < 2 {
		return false
	}

	// Check if timestamps are generally increasing
	for i := 0; i < len(s.replayPoints)-1; i++ {
		if s.replayPoints[i+1].Time.Before(s.replayPoints[i].Time) {
			return false
		}
	}
	return true
}
