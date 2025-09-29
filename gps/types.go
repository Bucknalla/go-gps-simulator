package gps

import "time"

// Satellite represents a GPS satellite
type Satellite struct {
	ID        int
	Elevation int // degrees above horizon
	Azimuth   int // degrees from north
	SNR       int // signal-to-noise ratio
}

// TrackPoint represents a point in a GPS track
type TrackPoint struct {
	Lat       float64   `xml:"lat,attr"`
	Lon       float64   `xml:"lon,attr"`
	Elevation float64   `xml:"ele"`
	Time      time.Time `xml:"time"`
}

// Position represents the current GPS position and status
type Position struct {
	Latitude   float64     `json:"latitude"`
	Longitude  float64     `json:"longitude"`
	Altitude   float64     `json:"altitude"`
	Speed      float64     `json:"speed"`  // knots
	Course     float64     `json:"course"` // degrees
	IsLocked   bool        `json:"is_locked"`
	Satellites []Satellite `json:"satellites"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Status represents the current simulator status
type Status struct {
	Running         bool          `json:"running"`
	StartTime       time.Time     `json:"start_time,omitempty"`
	ElapsedTime     time.Duration `json:"elapsed_time"`
	Position        Position      `json:"position"`
	Config          Config        `json:"config"`
	ReplayIndex     int           `json:"replay_index,omitempty"`
	ReplayTotal     int           `json:"replay_total,omitempty"`
	ReplayCompleted bool          `json:"replay_completed,omitempty"`
}

// NMEAData contains NMEA sentence data
type NMEAData struct {
	Sentences []string  `json:"sentences"`
	Position  Position  `json:"position"`
	Timestamp time.Time `json:"timestamp"`
}
