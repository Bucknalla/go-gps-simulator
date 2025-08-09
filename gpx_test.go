package main

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewGPXWriter(t *testing.T) {
	// Test creating a new GPX writer
	tempFile := "test_track.gpx"
	defer os.Remove(tempFile) // Clean up after test

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
	tempFile := "test_trackpoint.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	// Test adding a track point
	testTime := time.Date(2025, 8, 9, 12, 30, 45, 0, time.UTC)
	writer.AddTrackPoint(37.7749, -122.4194, 45.0, testTime)

	// Check that the track point was added
	if len(writer.gpx.Track.TrackSegment.TrackPoints) != 1 {
		t.Errorf("Expected 1 track point, got %d", len(writer.gpx.Track.TrackSegment.TrackPoints))
	}

	trackPoint := writer.gpx.Track.TrackSegment.TrackPoints[0]
	if trackPoint.Lat != 37.7749 {
		t.Errorf("Expected latitude 37.7749, got %f", trackPoint.Lat)
	}
	if trackPoint.Lon != -122.4194 {
		t.Errorf("Expected longitude -122.4194, got %f", trackPoint.Lon)
	}
	if trackPoint.Elevation != 45.0 {
		t.Errorf("Expected elevation 45.0, got %f", trackPoint.Elevation)
	}
	if !trackPoint.Time.Equal(testTime) {
		t.Errorf("Expected time %v, got %v", testTime, trackPoint.Time)
	}
}

func TestAddMultipleTrackPoints(t *testing.T) {
	tempFile := "test_multiple_points.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	// Add multiple track points
	baseTime := time.Date(2025, 8, 9, 12, 0, 0, 0, time.UTC)
	expectedPoints := 5

	for i := 0; i < expectedPoints; i++ {
		lat := 37.7749 + float64(i)*0.001
		lon := -122.4194 + float64(i)*0.001
		elevation := 45.0 + float64(i)*2.0
		pointTime := baseTime.Add(time.Duration(i) * time.Minute)

		writer.AddTrackPoint(lat, lon, elevation, pointTime)
	}

	// Check that all points were added
	if len(writer.gpx.Track.TrackSegment.TrackPoints) != expectedPoints {
		t.Errorf("Expected %d track points, got %d", expectedPoints, len(writer.gpx.Track.TrackSegment.TrackPoints))
	}

	// Verify the first and last points
	firstPoint := writer.gpx.Track.TrackSegment.TrackPoints[0]
	lastPoint := writer.gpx.Track.TrackSegment.TrackPoints[expectedPoints-1]

	if firstPoint.Lat != 37.7749 {
		t.Errorf("First point latitude incorrect: expected 37.7749, got %f", firstPoint.Lat)
	}

	expectedLastLat := 37.7749 + float64(expectedPoints-1)*0.001
	if lastPoint.Lat != expectedLastLat {
		t.Errorf("Last point latitude incorrect: expected %f, got %f", expectedLastLat, lastPoint.Lat)
	}
}

func TestGetTrackPointCount(t *testing.T) {
	tempFile := "test_count.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	// Initially should be 0
	if writer.GetTrackPointCount() != 0 {
		t.Errorf("Expected 0 track points initially, got %d", writer.GetTrackPointCount())
	}

	// Add some points and check count
	testTime := time.Now()
	for i := 0; i < 3; i++ {
		writer.AddTrackPoint(37.7749, -122.4194, 45.0, testTime)
	}

	if writer.GetTrackPointCount() != 3 {
		t.Errorf("Expected 3 track points, got %d", writer.GetTrackPointCount())
	}
}

func TestWriteToFile(t *testing.T) {
	tempFile := "test_write.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}

	// Add a track point
	testTime := time.Date(2025, 8, 9, 12, 30, 45, 0, time.UTC)
	writer.AddTrackPoint(37.7749, -122.4194, 45.0, testTime)

	// Write to file
	err = writer.WriteToFile()
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}

	writer.Close()

	// Read the file and verify content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GPX file: %v", err)
	}

	contentStr := string(content)

	// Check for XML declaration
	if !strings.Contains(contentStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("GPX file should contain XML declaration")
	}

	// Check for GPX root element
	if !strings.Contains(contentStr, "<gpx version=\"1.1\"") {
		t.Error("GPX file should contain GPX root element with version")
	}

	// Check for creator
	if !strings.Contains(contentStr, "creator=\"go-gps-simulator\"") {
		t.Error("GPX file should contain creator attribute")
	}

	// Check for namespace
	if !strings.Contains(contentStr, "xmlns=\"http://www.topografix.com/GPX/1/1\"") {
		t.Error("GPX file should contain GPX namespace")
	}

	// Check for track point data
	if !strings.Contains(contentStr, "lat=\"37.7749\"") {
		t.Error("GPX file should contain latitude data")
	}

	if !strings.Contains(contentStr, "lon=\"-122.4194\"") {
		t.Error("GPX file should contain longitude data")
	}

	if !strings.Contains(contentStr, "<ele>45</ele>") {
		t.Error("GPX file should contain elevation data")
	}

	if !strings.Contains(contentStr, "<time>2025-08-09T12:30:45Z</time>") {
		t.Error("GPX file should contain time data")
	}
}

func TestXMLStructureValidity(t *testing.T) {
	tempFile := "test_xml_validity.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}

	// Add multiple track points
	baseTime := time.Date(2025, 8, 9, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		lat := 37.7749 + float64(i)*0.001
		lon := -122.4194 + float64(i)*0.001
		elevation := 45.0 + float64(i)*2.0
		pointTime := baseTime.Add(time.Duration(i) * time.Minute)

		writer.AddTrackPoint(lat, lon, elevation, pointTime)
	}

	err = writer.WriteToFile()
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}

	writer.Close()

	// Read and parse the XML to ensure it's valid
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GPX file: %v", err)
	}

	var parsedGPX GPX
	err = xml.Unmarshal(content, &parsedGPX)
	if err != nil {
		t.Fatalf("Failed to parse GPX XML: %v", err)
	}

	// Verify the parsed structure
	if parsedGPX.Version != "1.1" {
		t.Errorf("Parsed GPX version incorrect: expected 1.1, got %s", parsedGPX.Version)
	}

	if parsedGPX.Creator != "go-gps-simulator" {
		t.Errorf("Parsed GPX creator incorrect: expected go-gps-simulator, got %s", parsedGPX.Creator)
	}

	if len(parsedGPX.Track.TrackSegment.TrackPoints) != 3 {
		t.Errorf("Parsed GPX should have 3 track points, got %d", len(parsedGPX.Track.TrackSegment.TrackPoints))
	}

	// Verify first track point
	firstPoint := parsedGPX.Track.TrackSegment.TrackPoints[0]
	if firstPoint.Lat != 37.7749 {
		t.Errorf("First track point latitude incorrect: expected 37.7749, got %f", firstPoint.Lat)
	}
	if firstPoint.Lon != -122.4194 {
		t.Errorf("First track point longitude incorrect: expected -122.4194, got %f", firstPoint.Lon)
	}
	if firstPoint.Elevation != 45.0 {
		t.Errorf("First track point elevation incorrect: expected 45.0, got %f", firstPoint.Elevation)
	}
}

func TestCloseWithoutFile(t *testing.T) {
	// Test closing a writer that was never used to write to file
	tempFile := "test_close_empty.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}

	// Close without writing anything
	err = writer.Close()
	if err != nil {
		t.Errorf("Close should not return error even with empty GPX: %v", err)
	}

	// File should exist but be empty or contain minimal GPX structure
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("GPX file should exist after close")
	}
}

func TestWriteToFileError(t *testing.T) {
	// Test WriteToFile with file permission error
	tempFile := "test_write_error.gpx"

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()
	defer os.Remove(tempFile)

	// Add a track point
	testTime := time.Date(2025, 8, 9, 12, 30, 45, 0, time.UTC)
	writer.AddTrackPoint(37.7749, -122.4194, 45.0, testTime)

	// Close the file first to simulate a closed file error
	if writer.file != nil {
		writer.file.Close()
		writer.file = nil
	}

	// Now try to write - should return an error
	err = writer.WriteToFile()
	if err == nil {
		t.Error("Expected error when writing to closed file, got nil")
	}
}

func TestCloseError(t *testing.T) {
	// Test Close with file error
	tempFile := "test_close_error.gpx"

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer os.Remove(tempFile)

	// Add a track point
	testTime := time.Date(2025, 8, 9, 12, 30, 45, 0, time.UTC)
	writer.AddTrackPoint(37.7749, -122.4194, 45.0, testTime)

	// Close the underlying file to simulate an error
	if writer.file != nil {
		writer.file.Close()
		// Don't set to nil - keep the reference so Close() will try to use it
	}

	// Now try to close - should return an error from WriteToFile
	err = writer.Close()
	if err == nil {
		t.Error("Expected error when closing with closed file, got nil")
	}
}

func TestCloseAlreadyClosed(t *testing.T) {
	// Test Close when already closed
	tempFile := "test_close_already_closed.gpx"

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer os.Remove(tempFile)

	// Close once
	err = writer.Close()
	if err != nil {
		t.Errorf("First close should not return error: %v", err)
	}

	// Close again - should return error because file is already closed
	err = writer.Close()
	if err == nil {
		t.Error("Second close should return error because file is already closed")
	}
}

func TestWriteToFileWithEmptyTrack(t *testing.T) {
	// Test WriteToFile with no track points
	tempFile := "test_write_empty.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	// Write without adding any track points
	err = writer.WriteToFile()
	if err != nil {
		t.Fatalf("Failed to write empty GPX file: %v", err)
	}

	// Read the file and verify it contains valid GPX structure
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GPX file: %v", err)
	}

	contentStr := string(content)

	// Check for basic GPX structure even with no track points
	if !strings.Contains(contentStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("GPX file should contain XML declaration")
	}
	if !strings.Contains(contentStr, "<gpx version=\"1.1\"") {
		t.Error("GPX file should contain GPX root element")
	}
	if !strings.Contains(contentStr, "<trk>") {
		t.Error("GPX file should contain track element")
	}
}

func TestTrackName(t *testing.T) {
	tempFile := "test_track_name.gpx"
	defer os.Remove(tempFile)

	writer, err := NewGPXWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create GPX writer: %v", err)
	}
	defer writer.Close()

	expectedTrackName := "GPS Simulator Track"
	if writer.gpx.Track.Name != expectedTrackName {
		t.Errorf("Expected track name '%s', got '%s'", expectedTrackName, writer.gpx.Track.Name)
	}

	// Write to file and verify the track name is preserved
	writer.AddTrackPoint(37.7749, -122.4194, 45.0, time.Now())
	err = writer.WriteToFile()
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}

	writer.Close()

	// Read and verify track name in file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GPX file: %v", err)
	}

	if !strings.Contains(string(content), "<name>GPS Simulator Track</name>") {
		t.Error("GPX file should contain track name")
	}
}
