package gps

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		expected string
	}{
		{
			name:     "Simple GGA sentence",
			sentence: "$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,",
			expected: "47",
		},
		{
			name:     "Simple RMC sentence",
			sentence: "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W",
			expected: "6A",
		},
		{
			name:     "Empty fields",
			sentence: "$GPGGA,,,,,,,,,,,,,,,",
			expected: "7A",
		},
		{
			name:     "Single character after $",
			sentence: "$A",
			expected: "41",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateChecksum(tt.sentence)
			if result != tt.expected {
				t.Errorf("calculateChecksum(%q) = %q, want %q", tt.sentence, result, tt.expected)
			}
		})
	}
}

func TestFormatNMEA(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		expected string
	}{
		{
			name:     "Simple sentence",
			sentence: "$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,",
			expected: "$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47\r\n",
		},
		{
			name:     "RMC sentence",
			sentence: "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W",
			expected: "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNMEA(tt.sentence)
			if result != tt.expected {
				t.Errorf("formatNMEA(%q) = %q, want %q", tt.sentence, result, tt.expected)
			}
		})
	}
}

// Helper function to create a test GPS simulator
func createTestSimulator() *GPSSimulator {
	config := Config{
		Latitude:   37.7749,
		Longitude:  -122.4194,
		Radius:     100.0,
		Jitter:     0.5,
		Speed:      0.1,
		Course:     0.0,
		Satellites: 8,
		TimeToLock: 30 * time.Second,
		OutputRate: 1 * time.Second,
	}

	sim := &GPSSimulator{
		Config:     config,
		currentLat: config.Latitude,
		currentLon: config.Longitude,
		isLocked:   true,
		Satellites: []Satellite{
			{ID: 1, Elevation: 45, Azimuth: 90, SNR: 35},
			{ID: 2, Elevation: 60, Azimuth: 180, SNR: 40},
			{ID: 3, Elevation: 30, Azimuth: 270, SNR: 25},
			{ID: 4, Elevation: 75, Azimuth: 0, SNR: 45},
		},
		nmeaWriter: &bytes.Buffer{},
	}

	return sim
}

func TestGenerateGGA(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)

	result := sim.generateGGA(testTime)

	// Check that the result is properly formatted
	if !strings.HasPrefix(result, "$GPGGA,") {
		t.Errorf("generateGGA should start with '$GPGGA,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateGGA should end with \\r\\n, got: %s", result)
	}

	// Check that it contains a checksum
	if !strings.Contains(result, "*") {
		t.Errorf("generateGGA should contain checksum separator '*', got: %s", result)
	}

	// Check time format (should be HHMMSS)
	if !strings.Contains(result, "123456") {
		t.Errorf("generateGGA should contain time '123456', got: %s", result)
	}

	// Check that coordinates are present (should contain latitude and longitude)
	parts := strings.Split(result, ",")
	if len(parts) < 15 {
		t.Errorf("generateGGA should have at least 15 comma-separated fields, got %d", len(parts))
	}

	// Verify quality indicator is set (should be "1" for GPS fix)
	if len(parts) > 6 && parts[6] != "1" {
		t.Errorf("generateGGA quality indicator should be '1', got: %s", parts[6])
	}
}

func TestGenerateNoFixGGA(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)

	result := sim.generateNoFixGGA(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPGGA,") {
		t.Errorf("generateNoFixGGA should start with '$GPGGA,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateNoFixGGA should end with \\r\\n, got: %s", result)
	}

	// Check that quality indicator is 0 (no fix)
	parts := strings.Split(result, ",")
	if len(parts) > 6 && parts[6] != "0" {
		t.Errorf("generateNoFixGGA quality indicator should be '0', got: %s", parts[6])
	}

	// Check that coordinate fields are empty
	if len(parts) > 2 && parts[2] != "" {
		t.Errorf("generateNoFixGGA latitude should be empty, got: %s", parts[2])
	}
	if len(parts) > 4 && parts[4] != "" {
		t.Errorf("generateNoFixGGA longitude should be empty, got: %s", parts[4])
	}
}

func TestGenerateRMC(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)

	result := sim.generateRMC(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPRMC,") {
		t.Errorf("generateRMC should start with '$GPRMC,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateRMC should end with \\r\\n, got: %s", result)
	}

	// Check time format
	if !strings.Contains(result, "123456") {
		t.Errorf("generateRMC should contain time '123456', got: %s", result)
	}

	// Check date format (should be DDMMYY)
	if !strings.Contains(result, "150124") {
		t.Errorf("generateRMC should contain date '150124', got: %s", result)
	}

	parts := strings.Split(result, ",")
	if len(parts) < 12 {
		t.Errorf("generateRMC should have at least 12 comma-separated fields, got %d", len(parts))
	}

	// Check status (should be "A" for active)
	if len(parts) > 2 && parts[2] != "A" {
		t.Errorf("generateRMC status should be 'A', got: %s", parts[2])
	}
}

func TestGenerateNoFixRMC(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)

	result := sim.generateNoFixRMC(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPRMC,") {
		t.Errorf("generateNoFixRMC should start with '$GPRMC,', got: %s", result)
	}

	parts := strings.Split(result, ",")

	// Check status (should be "V" for void/invalid)
	if len(parts) > 2 && parts[2] != "V" {
		t.Errorf("generateNoFixRMC status should be 'V', got: %s", parts[2])
	}

	// Check that coordinate fields are empty
	if len(parts) > 3 && parts[3] != "" {
		t.Errorf("generateNoFixRMC latitude should be empty, got: %s", parts[3])
	}
	if len(parts) > 5 && parts[5] != "" {
		t.Errorf("generateNoFixRMC longitude should be empty, got: %s", parts[5])
	}
}

func TestGenerateRMCWithSpeedAndCourse(t *testing.T) {
	// Create a simulator with custom speed and course
	config := Config{
		Latitude:   37.7749,
		Longitude:  -122.4194,
		Radius:     100.0,
		Jitter:     0.5,
		Speed:      12.5,
		Course:     270.0,
		Satellites: 8,
		TimeToLock: 30 * time.Second,
		OutputRate: 1 * time.Second,
	}

	now := time.Now()
	sim := &GPSSimulator{
		Config:         config,
		currentLat:     config.Latitude,
		currentLon:     config.Longitude,
		currentSpeed:   config.Speed,
		currentCourse:  config.Course,
		isLocked:       true,
		lastUpdateTime: now,
	}

	testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)
	result := sim.generateRMC(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPRMC,") {
		t.Errorf("generateRMC should start with '$GPRMC,', got: %s", result)
	}

	// Parse the RMC sentence to check speed and course fields
	parts := strings.Split(result, ",")
	if len(parts) < 9 {
		t.Fatalf("RMC sentence should have at least 9 fields, got %d", len(parts))
	}

	// Check speed field (index 7) - should match currentSpeed
	expectedSpeed := "12.5"
	if parts[7] != expectedSpeed {
		t.Errorf("Expected speed %s, got %s", expectedSpeed, parts[7])
	}

	// Check course field (index 8) - should match currentCourse
	expectedCourse := "270.0"
	if parts[8] != expectedCourse {
		t.Errorf("Expected course %s, got %s", expectedCourse, parts[8])
	}
}

func TestUpdateSpeedAndCourse(t *testing.T) {
	tests := []struct {
		name                string
		jitter              float64
		baseSpeed           float64
		baseCourse          float64
		expectedSpeedRange  [2]float64 // [min, max] expected speed range
		expectedCourseRange [2]float64 // [min, max] expected course range
	}{
		{
			name:                "Low jitter",
			jitter:              0.1,
			baseSpeed:           10.0,
			baseCourse:          90.0,
			expectedSpeedRange:  [2]float64{9.5, 10.5},  // ±5%
			expectedCourseRange: [2]float64{88.0, 92.0}, // ±2°
		},
		{
			name:                "Medium jitter",
			jitter:              0.5,
			baseSpeed:           10.0,
			baseCourse:          90.0,
			expectedSpeedRange:  [2]float64{7.0, 13.0},   // ±30%
			expectedCourseRange: [2]float64{75.0, 105.0}, // ±15°
		},
		{
			name:                "High jitter",
			jitter:              0.9,
			baseSpeed:           10.0,
			baseCourse:          90.0,
			expectedSpeedRange:  [2]float64{5.0, 15.0},   // ±50%
			expectedCourseRange: [2]float64{60.0, 120.0}, // ±30°
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Latitude:   37.7749,
				Longitude:  -122.4194,
				Jitter:     tt.jitter,
				Speed:      tt.baseSpeed,
				Course:     tt.baseCourse,
				Satellites: 8,
			}

			now := time.Now()
			sim := &GPSSimulator{
				Config:         config,
				currentSpeed:   config.Speed,
				currentCourse:  config.Course,
				lastUpdateTime: now,
			}

			// Test multiple updates to check variation range
			speedMin, speedMax := tt.baseSpeed, tt.baseSpeed
			courseMin, courseMax := tt.baseCourse, tt.baseCourse

			for i := 0; i < 50; i++ { // Run multiple times to test range
				sim.updateSpeedAndCourse()

				if sim.currentSpeed < speedMin {
					speedMin = sim.currentSpeed
				}
				if sim.currentSpeed > speedMax {
					speedMax = sim.currentSpeed
				}

				if sim.currentCourse < courseMin {
					courseMin = sim.currentCourse
				}
				if sim.currentCourse > courseMax {
					courseMax = sim.currentCourse
				}
			}

			// Check that variations are within expected ranges
			if speedMin < tt.expectedSpeedRange[0] || speedMax > tt.expectedSpeedRange[1] {
				t.Errorf("Speed range [%.1f, %.1f] outside expected [%.1f, %.1f]",
					speedMin, speedMax, tt.expectedSpeedRange[0], tt.expectedSpeedRange[1])
			}

			if courseMin < tt.expectedCourseRange[0] || courseMax > tt.expectedCourseRange[1] {
				t.Errorf("Course range [%.1f, %.1f] outside expected [%.1f, %.1f]",
					courseMin, courseMax, tt.expectedCourseRange[0], tt.expectedCourseRange[1])
			}

			// Speed should never be negative
			if speedMin < 0 {
				t.Errorf("Speed should never be negative, got %.1f", speedMin)
			}

			// Course should be normalized to 0-359 range
			if courseMin < 0 || courseMax >= 360 {
				t.Errorf("Course should be in range [0, 360), got [%.1f, %.1f]", courseMin, courseMax)
			}
		})
	}
}

func TestGenerateGSA(t *testing.T) {
	sim := createTestSimulator()

	result := sim.generateGSA()

	// Check basic format
	if !strings.HasPrefix(result, "$GPGSA,") {
		t.Errorf("generateGSA should start with '$GPGSA,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateGSA should end with \\r\\n, got: %s", result)
	}

	parts := strings.Split(result, ",")
	if len(parts) < 18 {
		t.Errorf("generateGSA should have at least 18 comma-separated fields, got %d", len(parts))
	}

	// Check mode1 (should be "A" for automatic)
	if len(parts) > 1 && parts[1] != "A" {
		t.Errorf("generateGSA mode1 should be 'A', got: %s", parts[1])
	}

	// Check mode2 (should be "3" for 3D fix)
	if len(parts) > 2 && parts[2] != "3" {
		t.Errorf("generateGSA mode2 should be '3', got: %s", parts[2])
	}

	// Check that satellite IDs are present
	satCount := 0
	for i := 3; i < 15 && i < len(parts); i++ {
		if parts[i] != "" {
			satCount++
		}
	}
	if satCount != len(sim.Satellites) {
		t.Errorf("generateGSA should contain %d satellite IDs, got %d", len(sim.Satellites), satCount)
	}
}

func TestGenerateGSV(t *testing.T) {
	sim := createTestSimulator()

	results := sim.generateGSV()

	if len(results) == 0 {
		t.Errorf("generateGSV should return at least one sentence")
	}

	// Check that all results are properly formatted
	for i, result := range results {
		if !strings.HasPrefix(result, "$GPGSV,") {
			t.Errorf("generateGSV[%d] should start with '$GPGSV,', got: %s", i, result)
		}

		if !strings.HasSuffix(result, "\r\n") {
			t.Errorf("generateGSV[%d] should end with \\r\\n, got: %s", i, result)
		}

		parts := strings.Split(result, ",")
		if len(parts) < 4 {
			t.Errorf("generateGSV[%d] should have at least 4 comma-separated fields, got %d", i, len(parts))
		}

		// Check total sentences field
		if len(parts) > 1 && parts[1] == "" {
			t.Errorf("generateGSV[%d] total sentences field should not be empty", i)
		}

		// Check sentence number field
		if len(parts) > 2 && parts[2] == "" {
			t.Errorf("generateGSV[%d] sentence number field should not be empty", i)
		}

		// Check total satellites field
		if len(parts) > 3 && parts[3] == "" {
			t.Errorf("generateGSV[%d] total satellites field should not be empty", i)
		}
	}
}

func TestGenerateGSVMultipleSentences(t *testing.T) {
	// Create simulator with many satellites to test multiple GSV sentences
	sim := createTestSimulator()
	sim.Satellites = make([]Satellite, 12) // Maximum satellites
	for i := 0; i < 12; i++ {
		sim.Satellites[i] = Satellite{
			ID:        i + 1,
			Elevation: 45,
			Azimuth:   i * 30,
			SNR:       35,
		}
	}

	results := sim.generateGSV()

	expectedSentences := 3 // 12 satellites / 4 per sentence = 3 sentences
	if len(results) != expectedSentences {
		t.Errorf("generateGSV with 12 satellites should return %d sentences, got %d", expectedSentences, len(results))
	}

	// Check that each sentence has the correct total count
	for i, result := range results {
		parts := strings.Split(result, ",")
		if len(parts) > 1 && parts[1] != "3" {
			t.Errorf("generateGSV[%d] should indicate 3 total sentences, got: %s", i, parts[1])
		}
		if len(parts) > 3 && parts[3] != "12" {
			t.Errorf("generateGSV[%d] should indicate 12 total satellites, got: %s", i, parts[3])
		}
	}
}

func TestCoordinateConversion(t *testing.T) {
	tests := []struct {
		name         string
		lat          float64
		lon          float64
		expectLat    string // Expected latitude format in NMEA
		expectLon    string // Expected longitude format in NMEA
		expectLatHem string
		expectLonHem string
	}{
		{
			name:         "San Francisco coordinates",
			lat:          37.7749,
			lon:          -122.4194,
			expectLat:    "3746.494",  // Should be close to this format
			expectLon:    "12225.164", // Should be close to this format
			expectLatHem: "N",
			expectLonHem: "W",
		},
		{
			name:         "Southern hemisphere",
			lat:          -33.8688,
			lon:          151.2093,
			expectLatHem: "S",
			expectLonHem: "E",
		},
		{
			name:         "Zero coordinates",
			lat:          0.0,
			lon:          0.0,
			expectLatHem: "N",
			expectLonHem: "E",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := createTestSimulator()
			sim.currentLat = tt.lat
			sim.currentLon = tt.lon

			testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)
			result := sim.generateGGA(testTime)

			parts := strings.Split(result, ",")
			if len(parts) < 6 {
				t.Fatalf("GGA sentence should have at least 6 fields")
			}

			// Check hemisphere indicators
			if parts[3] != tt.expectLatHem {
				t.Errorf("Expected latitude hemisphere %s, got %s", tt.expectLatHem, parts[3])
			}
			if parts[5] != tt.expectLonHem {
				t.Errorf("Expected longitude hemisphere %s, got %s", tt.expectLonHem, parts[5])
			}

			// For non-zero coordinates, check that coordinate fields are not empty
			if tt.lat != 0.0 && parts[2] == "" {
				t.Errorf("Latitude field should not be empty for non-zero latitude")
			}
			if tt.lon != 0.0 && parts[4] == "" {
				t.Errorf("Longitude field should not be empty for non-zero longitude")
			}
		})
	}
}

func TestGenerateVTG(t *testing.T) {
	sim := createTestSimulator()

	result := sim.generateVTG()

	// Check basic format
	if !strings.HasPrefix(result, "$GPVTG,") {
		t.Errorf("generateVTG should start with '$GPVTG,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateVTG should end with \\r\\n, got: %s", result)
	}

	// Split sentence and remove checksum part
	sentencePart := strings.Split(result, "*")[0]
	parts := strings.Split(sentencePart, ",")
	if len(parts) < 10 {
		t.Errorf("generateVTG should have at least 10 comma-separated fields, got %d", len(parts))
	}

	// Check that course is present (field 1)
	if len(parts) > 1 && parts[1] == "" {
		t.Errorf("generateVTG course field should not be empty")
	}

	// Check course reference (field 2) should be "T"
	if len(parts) > 2 && parts[2] != "T" {
		t.Errorf("generateVTG course reference should be 'T', got: %s", parts[2])
	}

	// Check magnetic reference (field 4) should be "M"
	if len(parts) > 4 && parts[4] != "M" {
		t.Errorf("generateVTG magnetic reference should be 'M', got: %s", parts[4])
	}

	// Check speed in knots is present (field 5)
	if len(parts) > 5 && parts[5] == "" {
		t.Errorf("generateVTG speed in knots should not be empty")
	}

	// Check knots unit (field 6) should be "N"
	if len(parts) > 6 && parts[6] != "N" {
		t.Errorf("generateVTG knots unit should be 'N', got: %s", parts[6])
	}

	// Check speed in km/h is present (field 7)
	if len(parts) > 7 && parts[7] == "" {
		t.Errorf("generateVTG speed in km/h should not be empty")
	}

	// Check km/h unit (field 8) should be "K"
	if len(parts) > 8 && parts[8] != "K" {
		t.Errorf("generateVTG km/h unit should be 'K', got: %s", parts[8])
	}

	// Check mode (field 9) should be "A"
	if len(parts) > 9 && parts[9] != "A" {
		t.Errorf("generateVTG mode should be 'A', got: %s", parts[9])
	}
}

func TestGenerateNoFixVTG(t *testing.T) {
	sim := createTestSimulator()

	result := sim.generateNoFixVTG()

	// Check basic format
	if !strings.HasPrefix(result, "$GPVTG,") {
		t.Errorf("generateNoFixVTG should start with '$GPVTG,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateNoFixVTG should end with \\r\\n, got: %s", result)
	}

	// Check that it ends with "N" for not valid
	if !strings.Contains(result, ",N*") {
		t.Errorf("generateNoFixVTG should contain ',N*' for not valid status, got: %s", result)
	}

	// Split sentence and remove checksum part
	sentencePart := strings.Split(result, "*")[0]
	parts := strings.Split(sentencePart, ",")
	// Check that most fields are empty (no fix)
	for i := 1; i < 9; i++ { // Fields 1-8 should be empty
		if len(parts) > i && parts[i] != "" {
			t.Errorf("generateNoFixVTG field %d should be empty, got: %s", i, parts[i])
		}
	}
}

func TestVTGSpeedConversion(t *testing.T) {
	// Create simulator with known speed
	config := Config{
		Latitude:   37.7749,
		Longitude:  -122.4194,
		Speed:      10.0, // 10 knots
		Course:     90.0, // 90 degrees
		Satellites: 8,
	}

	now := time.Now()
	sim := &GPSSimulator{
		Config:         config,
		currentSpeed:   config.Speed,
		currentCourse:  config.Course,
		isLocked:       true,
		lastUpdateTime: now,
	}

	result := sim.generateVTG()
	// Split sentence and remove checksum part
	sentencePart := strings.Split(result, "*")[0]
	parts := strings.Split(sentencePart, ",")

	if len(parts) < 8 {
		t.Fatalf("VTG sentence should have at least 8 fields")
	}

	// Check speed in knots (should be 10.0)
	expectedKnots := "10.0"
	if parts[5] != expectedKnots {
		t.Errorf("Expected speed in knots %s, got %s", expectedKnots, parts[5])
	}

	// Check speed in km/h (should be 10.0 * 1.852 = 18.52)
	expectedKmh := "18.5" // Rounded to 1 decimal place
	if parts[7] != expectedKmh {
		t.Errorf("Expected speed in km/h %s, got %s", expectedKmh, parts[7])
	}

	// Check course (should be 90.0)
	expectedCourse := "90.0"
	if parts[1] != expectedCourse {
		t.Errorf("Expected course %s, got %s", expectedCourse, parts[1])
	}
}

func TestNMEAChecksumValidation(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)

	// Test that generated sentences have valid checksums
	sentences := []string{
		sim.generateGGA(testTime),
		sim.generateRMC(testTime),
		sim.generateGSA(),
		sim.generateVTG(),
	}

	// Add GSV sentences
	gsv := sim.generateGSV()
	sentences = append(sentences, gsv...)

	for i, sentence := range sentences {
		// Split sentence and checksum
		parts := strings.Split(sentence, "*")
		if len(parts) != 2 {
			t.Errorf("Sentence %d should contain exactly one '*' separator, got: %s", i, sentence)
			continue
		}

		nmeaPart := parts[0]
		checksumPart := strings.TrimSuffix(parts[1], "\r\n")

		// Calculate expected checksum
		expectedChecksum := calculateChecksum(nmeaPart)

		if checksumPart != expectedChecksum {
			t.Errorf("Sentence %d has incorrect checksum. Expected %s, got %s. Sentence: %s",
				i, expectedChecksum, checksumPart, sentence)
		}
	}
}

func TestGenerateGLL(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 123000000, time.UTC) // With milliseconds

	result := sim.generateGLL(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPGLL,") {
		t.Errorf("generateGLL should start with '$GPGLL,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateGLL should end with \\r\\n, got: %s", result)
	}

	// Check that it contains a checksum
	if !strings.Contains(result, "*") {
		t.Errorf("generateGLL should contain checksum separator '*', got: %s", result)
	}

	// Check time format (should be HHMMSS.SS)
	if !strings.Contains(result, "123456.12") {
		t.Errorf("generateGLL should contain time '123456.12', got: %s", result)
	}

	// Parse the GLL sentence
	parts := strings.Split(result, ",")
	if len(parts) < 8 {
		t.Errorf("generateGLL should have at least 8 comma-separated fields, got %d", len(parts))
	}

	// Check that coordinates are present
	if len(parts) > 1 && parts[1] == "" {
		t.Errorf("generateGLL latitude field should not be empty")
	}
	if len(parts) > 3 && parts[3] == "" {
		t.Errorf("generateGLL longitude field should not be empty")
	}

	// Check hemisphere indicators
	if len(parts) > 2 && parts[2] != "N" && parts[2] != "S" {
		t.Errorf("generateGLL latitude hemisphere should be 'N' or 'S', got: %s", parts[2])
	}
	if len(parts) > 4 && parts[4] != "E" && parts[4] != "W" {
		t.Errorf("generateGLL longitude hemisphere should be 'E' or 'W', got: %s", parts[4])
	}

	// Check status (should be "A" for active)
	if len(parts) > 6 && parts[6] != "A" {
		t.Errorf("generateGLL status should be 'A', got: %s", parts[6])
	}

	// Check mode (should be "A" for autonomous)
	sentencePart := strings.Split(parts[7], "*")[0] // Remove checksum
	if sentencePart != "A" {
		t.Errorf("generateGLL mode should be 'A', got: %s", sentencePart)
	}
}

func TestGenerateNoFixGLL(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 123000000, time.UTC)

	result := sim.generateNoFixGLL(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPGLL,") {
		t.Errorf("generateNoFixGLL should start with '$GPGLL,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateNoFixGLL should end with \\r\\n, got: %s", result)
	}

	// Check that coordinate fields are empty
	parts := strings.Split(result, ",")
	if len(parts) > 1 && parts[1] != "" {
		t.Errorf("generateNoFixGLL latitude should be empty, got: %s", parts[1])
	}
	if len(parts) > 3 && parts[3] != "" {
		t.Errorf("generateNoFixGLL longitude should be empty, got: %s", parts[3])
	}

	// Check status (should be "V" for invalid) - field index 6
	if len(parts) > 6 && parts[6] != "V" {
		t.Errorf("generateNoFixGLL status should be 'V', got: %s", parts[6])
	}

	// Check mode (should be "N" for not valid) - field index 7
	if len(parts) > 7 {
		sentencePart := strings.Split(parts[7], "*")[0] // Remove checksum
		if sentencePart != "N" {
			t.Errorf("generateNoFixGLL mode should be 'N', got: %s", sentencePart)
		}
	}
}

func TestGenerateZDA(t *testing.T) {
	sim := createTestSimulator()
	testTime := time.Date(2024, 1, 15, 12, 34, 56, 123000000, time.UTC)

	result := sim.generateZDA(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPZDA,") {
		t.Errorf("generateZDA should start with '$GPZDA,', got: %s", result)
	}

	if !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateZDA should end with \\r\\n, got: %s", result)
	}

	// Check that it contains a checksum
	if !strings.Contains(result, "*") {
		t.Errorf("generateZDA should contain checksum separator '*', got: %s", result)
	}

	// Parse the ZDA sentence
	sentencePart := strings.Split(result, "*")[0] // Remove checksum part
	parts := strings.Split(sentencePart, ",")
	if len(parts) < 7 {
		t.Errorf("generateZDA should have at least 7 comma-separated fields, got %d", len(parts))
	}

	// Check time format (should be HHMMSS.SS)
	if len(parts) > 1 && parts[1] != "123456.12" {
		t.Errorf("generateZDA should contain time '123456.12', got: %s", parts[1])
	}

	// Check day
	if len(parts) > 2 && parts[2] != "15" {
		t.Errorf("generateZDA should contain day '15', got: %s", parts[2])
	}

	// Check month
	if len(parts) > 3 && parts[3] != "01" {
		t.Errorf("generateZDA should contain month '01', got: %s", parts[3])
	}

	// Check year
	if len(parts) > 4 && parts[4] != "2024" {
		t.Errorf("generateZDA should contain year '2024', got: %s", parts[4])
	}

	// Check local zone hours (should be "00" for UTC)
	if len(parts) > 5 && parts[5] != "00" {
		t.Errorf("generateZDA should contain local zone hours '00', got: %s", parts[5])
	}

	// Check local zone minutes (should be "00" for UTC)
	if len(parts) > 6 && parts[6] != "00" {
		t.Errorf("generateZDA should contain local zone minutes '00', got: %s", parts[6])
	}
}

func TestGLLCoordinateFormats(t *testing.T) {
	tests := []struct {
		name         string
		lat          float64
		lon          float64
		expectLatHem string
		expectLonHem string
	}{
		{
			name:         "Northern/Eastern hemisphere",
			lat:          37.7749,
			lon:          122.4194,
			expectLatHem: "N",
			expectLonHem: "E",
		},
		{
			name:         "Southern/Western hemisphere",
			lat:          -33.8688,
			lon:          -151.2093,
			expectLatHem: "S",
			expectLonHem: "W",
		},
		{
			name:         "Zero coordinates",
			lat:          0.0,
			lon:          0.0,
			expectLatHem: "N",
			expectLonHem: "E",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := createTestSimulator()
			sim.currentLat = tt.lat
			sim.currentLon = tt.lon

			testTime := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)
			result := sim.generateGLL(testTime)

			parts := strings.Split(result, ",")
			if len(parts) < 5 {
				t.Fatalf("GLL sentence should have at least 5 fields")
			}

			// Check hemisphere indicators
			if parts[2] != tt.expectLatHem {
				t.Errorf("Expected latitude hemisphere %s, got %s", tt.expectLatHem, parts[2])
			}
			if parts[4] != tt.expectLonHem {
				t.Errorf("Expected longitude hemisphere %s, got %s", tt.expectLonHem, parts[4])
			}

			// For non-zero coordinates, check that coordinate fields are not empty
			if tt.lat != 0.0 && parts[1] == "" {
				t.Errorf("Latitude field should not be empty for non-zero latitude")
			}
			if tt.lon != 0.0 && parts[3] == "" {
				t.Errorf("Longitude field should not be empty for non-zero longitude")
			}
		})
	}
}

func TestZDADifferentTimes(t *testing.T) {
	sim := createTestSimulator()

	tests := []struct {
		name          string
		testTime      time.Time
		expectedTime  string
		expectedDay   string
		expectedMonth string
		expectedYear  string
	}{
		{
			name:          "New Year's Day",
			testTime:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedTime:  "000000.00",
			expectedDay:   "01",
			expectedMonth: "01",
			expectedYear:  "2024",
		},
		{
			name:          "Leap year date",
			testTime:      time.Date(2024, 2, 29, 23, 59, 59, 999000000, time.UTC),
			expectedTime:  "235959.99",
			expectedDay:   "29",
			expectedMonth: "02",
			expectedYear:  "2024",
		},
		{
			name:          "End of year",
			testTime:      time.Date(2023, 12, 31, 12, 30, 45, 500000000, time.UTC),
			expectedTime:  "123045.50",
			expectedDay:   "31",
			expectedMonth: "12",
			expectedYear:  "2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sim.generateZDA(tt.testTime)

			sentencePart := strings.Split(result, "*")[0]
			parts := strings.Split(sentencePart, ",")

			if len(parts) < 5 {
				t.Fatalf("ZDA sentence should have at least 5 fields")
			}

			if parts[1] != tt.expectedTime {
				t.Errorf("Expected time %s, got %s", tt.expectedTime, parts[1])
			}
			if parts[2] != tt.expectedDay {
				t.Errorf("Expected day %s, got %s", tt.expectedDay, parts[2])
			}
			if parts[3] != tt.expectedMonth {
				t.Errorf("Expected month %s, got %s", tt.expectedMonth, parts[3])
			}
			if parts[4] != tt.expectedYear {
				t.Errorf("Expected year %s, got %s", tt.expectedYear, parts[4])
			}
		})
	}
}
