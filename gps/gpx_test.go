package gps

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewGPXWriter(t *testing.T) {
	// Test creating a new GPX writer
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_track.gpx")

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	// Check that the writer was initialized correctly
	if writer.filename != tempFile {
		t.Errorf("Expected filename %s, got %s", tempFile, writer.filename)
	}

	if writer.gpx == nil {
		t.Error("GPX structure should be initialized")
	}

	if writer.gpx.Version != "1.1" {
		t.Errorf("Expected GPX version 1.1, got %s", writer.gpx.Version)
	}

	if writer.gpx.Creator != "go-gps-simulator" {
		t.Errorf("Expected creator 'go-gps-simulator', got %s", writer.gpx.Creator)
	}

	if writer.gpx.Xmlns != "http://www.topografix.com/GPX/1/1" {
		t.Errorf("Expected GPX namespace, got %s", writer.gpx.Xmlns)
	}
}

func TestNewGPXWriterInvalidPath(t *testing.T) {
	// Test creating GPX writer with invalid path
	_, err := NewGPXWriter("/invalid/path/test.gpx")
	if err == nil {
		t.Error("Expected error for invalid file path, got nil")
	}
}

func TestAddTrackPoint(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_track_points.gpx")

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	// Add some track points
	points := []struct {
		lat, lon, elevation float64
		time                time.Time
	}{
		{37.7749, -122.4194, 50.0, time.Now()},
		{37.7750, -122.4193, 51.0, time.Now().Add(time.Second)},
		{37.7751, -122.4192, 52.0, time.Now().Add(2 * time.Second)},
	}

	for _, point := range points {
		writer.AddTrackPoint(point.lat, point.lon, point.elevation, point.time)
	}

	// Check that track points were added
	if writer.GetTrackPointCount() != len(points) {
		t.Errorf("Expected %d track points, got %d", len(points), writer.GetTrackPointCount())
	}

	// Verify the track structure exists (we can't access internal structure directly)
	// The track point count verification above is sufficient to confirm points were added
}

func TestWriteToFile(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_write.gpx")

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}

	// Add a track point
	writer.AddTrackPoint(37.7749, -122.4194, 50.0, time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC))

	// Write to file
	err = writer.WriteToFile()
	if err != nil {
		t.Fatalf("Failed to write GPX file: %v", err)
	}

	writer.Close()

	// Read the file back and verify content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GPX file: %v", err)
	}

	contentStr := string(content)

	// Check XML declaration
	if !strings.HasPrefix(contentStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("GPX file should start with XML declaration")
	}

	// Check GPX root element
	if !strings.Contains(contentStr, "<gpx") {
		t.Error("GPX file should contain <gpx> element")
	}

	// Check track point data (lat/lon are attributes, ele/time are elements)
	if !strings.Contains(contentStr, `lat="37.7749"`) {
		t.Error("GPX file should contain latitude data")
	}
	if !strings.Contains(contentStr, `lon="-122.4194"`) {
		t.Error("GPX file should contain longitude data")
	}
	if !strings.Contains(contentStr, "<ele>50</ele>") {
		t.Error("GPX file should contain elevation data")
	}
	if !strings.Contains(contentStr, "<time>2024-01-15T10:30:45Z</time>") {
		t.Error("GPX file should contain time data")
	}
}

func TestCloseGPXWriter(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_close.gpx")

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}

	// Add a track point
	writer.AddTrackPoint(37.7749, -122.4194, 50.0, time.Now())

	// Close should write the file
	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close GPX writer: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("GPX file should exist after closing")
	}

	// Verify file content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GPX file: %v", err)
	}

	if len(content) == 0 {
		t.Error("GPX file should not be empty")
	}
}

func TestGPXStructureValidation(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_validation.gpx")

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	// Add track points with various data
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)
	writer.AddTrackPoint(37.7749, -122.4194, 50.5, testTime)

	// Write and read back to validate XML structure
	err = writer.WriteToFile()
	if err != nil {
		t.Fatalf("Failed to write GPX file: %v", err)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GPX file: %v", err)
	}

	// Parse XML to ensure it's valid
	var gpx GPX
	err = xml.Unmarshal(content, &gpx)
	if err != nil {
		t.Fatalf("GPX file contains invalid XML: %v", err)
	}

	// Validate structure
	if gpx.Version != "1.1" {
		t.Errorf("Expected version 1.1, got %s", gpx.Version)
	}
	if gpx.Creator != "go-gps-simulator" {
		t.Errorf("Expected creator 'go-gps-simulator', got %s", gpx.Creator)
	}

	// Verify track structure exists
	if len(gpx.Track.TrackSegment.TrackPoints) != 1 {
		t.Errorf("Expected 1 track point, got %d", len(gpx.Track.TrackSegment.TrackPoints))
	}

	if len(gpx.Track.TrackSegment.TrackPoints) > 0 {
		point := gpx.Track.TrackSegment.TrackPoints[0]
		if point.Lat != 37.7749 {
			t.Errorf("Expected lat 37.7749, got %f", point.Lat)
		}
		if point.Lon != -122.4194 {
			t.Errorf("Expected lon -122.4194, got %f", point.Lon)
		}
		if point.Elevation != 50.5 {
			t.Errorf("Expected elevation 50.5, got %f", point.Elevation)
		}
	}
}

// Note: ParseGPX functionality is not available in this GPS library version
// The tests above cover the GPX writing functionality that is available
