package main

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

// TrackPoint represents a single point in a GPX track
type TrackPoint struct {
	Lat       float64   `xml:"lat,attr"`
	Lon       float64   `xml:"lon,attr"`
	Elevation float64   `xml:"ele"`
	Time      time.Time `xml:"time"`
}

// GPXWriter handles writing GPS data to a GPX file
type GPXWriter struct {
	filename    string
	gpx         *GPX
	file        *os.File
	isFirstTime bool
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
		filename:    filename,
		gpx:         gpx,
		file:        file,
		isFirstTime: true,
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
