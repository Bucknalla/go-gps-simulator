package gps

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

// GPX represents the root GPX document structure
type GPX struct {
	XMLName xml.Name `xml:"gpx"`
	Version string   `xml:"version,attr"`
	Creator string   `xml:"creator,attr"`
	Xmlns   string   `xml:"xmlns,attr"`
	Track   Track    `xml:"trk"`
	Routes  []Route  `xml:"rte"`
}

// Track represents a GPX track
type Track struct {
	Name         string       `xml:"name"`
	TrackSegment TrackSegment `xml:"trkseg"`
}

// TrackSegment represents a segment of a GPX track
type TrackSegment struct {
	TrackPoints []TrackPoint `xml:"trkpt"`
}

// Route represents a GPX route
type Route struct {
	Name        string       `xml:"name"`
	RoutePoints []RoutePoint `xml:"rtept"`
}

// RoutePoint represents a single point in a GPX route
type RoutePoint struct {
	Lat       float64   `xml:"lat,attr"`
	Lon       float64   `xml:"lon,attr"`
	Elevation float64   `xml:"ele"`
	Time      time.Time `xml:"time"`
}

// GPXWriter handles writing GPS data to a GPX file
type GPXWriter struct {
	filename string
	gpx      *GPX
	file     *os.File
}

// NewGPXWriter creates a new GPX writer
func NewGPXWriter(filename string) (*GPXWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create GPX file %s: %v", filename, err)
	}

	gpx := &GPX{
		Version: "1.1",
		Creator: "go-gps-simulator",
		Xmlns:   "http://www.topografix.com/GPX/1/1",
		Track: Track{
			Name: "GPS Simulator Track",
			TrackSegment: TrackSegment{
				TrackPoints: []TrackPoint{},
			},
		},
	}

	writer := &GPXWriter{
		filename: filename,
		gpx:      gpx,
		file:     file,
	}

	return writer, nil
}

// AddTrackPoint adds a new track point to the GPX file
func (w *GPXWriter) AddTrackPoint(lat, lon, elevation float64, timestamp time.Time) {
	trackPoint := TrackPoint{
		Lat:       lat,
		Lon:       lon,
		Elevation: elevation,
		Time:      timestamp.UTC(),
	}

	w.gpx.Track.TrackSegment.TrackPoints = append(w.gpx.Track.TrackSegment.TrackPoints, trackPoint)
}

// WriteToFile writes the current GPX data to the file
func (w *GPXWriter) WriteToFile() error {
	// Seek to the beginning of the file
	_, err := w.file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to beginning of file: %v", err)
	}

	// Truncate the file to remove any existing content
	err = w.file.Truncate(0)
	if err != nil {
		return fmt.Errorf("failed to truncate file: %v", err)
	}

	// Write XML header
	_, err = w.file.WriteString(xml.Header)
	if err != nil {
		return fmt.Errorf("failed to write XML header: %v", err)
	}

	// Marshal and write the GPX data
	encoder := xml.NewEncoder(w.file)
	encoder.Indent("", "  ")
	err = encoder.Encode(w.gpx)
	if err != nil {
		return fmt.Errorf("failed to encode GPX data: %v", err)
	}

	// Flush to ensure data is written
	err = w.file.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync file: %v", err)
	}

	return nil
}

// Close closes the GPX file
func (w *GPXWriter) Close() error {
	if w.file != nil {
		// Write final data before closing
		err := w.WriteToFile()
		if err != nil {
			w.file.Close()
			return err
		}
		return w.file.Close()
	}
	return nil
}

// GetTrackPointCount returns the number of track points currently stored
func (w *GPXWriter) GetTrackPointCount() int {
	return len(w.gpx.Track.TrackSegment.TrackPoints)
}

// ReadGPXFile reads and parses a GPX file, returning the track points
func ReadGPXFile(filename string) ([]TrackPoint, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPX file %s: %v", filename, err)
	}
	defer file.Close()

	var gpx GPX
	decoder := xml.NewDecoder(file)
	err = decoder.Decode(&gpx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GPX file %s: %v", filename, err)
	}

	var points []TrackPoint

	// Try to get points from tracks first
	if len(gpx.Track.TrackSegment.TrackPoints) > 0 {
		points = gpx.Track.TrackSegment.TrackPoints
	} else if len(gpx.Routes) > 0 && len(gpx.Routes[0].RoutePoints) > 0 {
		// Convert route points to track points
		routePoints := gpx.Routes[0].RoutePoints
		points = make([]TrackPoint, len(routePoints))
		for i, rp := range routePoints {
			points[i] = TrackPoint{
				Lat:       rp.Lat,
				Lon:       rp.Lon,
				Elevation: rp.Elevation,
				Time:      rp.Time,
			}
		}
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no track points or route points found in GPX file %s", filename)
	}

	return points, nil
}
