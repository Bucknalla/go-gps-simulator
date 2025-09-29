package gps

import (
	"bytes"
	"fmt"
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

func TestGenerateGGA(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = true
	sim.currentLat = 37.7749
	sim.currentLon = -122.4194
	sim.currentAlt = 45.0

	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	result := sim.generateGGA(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPGGA,") {
		t.Errorf("generateGGA should start with '$GPGGA,', got: %s", result)
	}

	// Check that it contains expected time
	if !strings.Contains(result, "103045") {
		t.Errorf("generateGGA should contain time '103045', got: %s", result)
	}

	// Check that it contains coordinates
	if !strings.Contains(result, "3746.4940,N") {
		t.Errorf("generateGGA should contain latitude '3746.4940,N', got: %s", result)
	}
	if !strings.Contains(result, "12225.1640,W") {
		t.Errorf("generateGGA should contain longitude '12225.1640,W', got: %s", result)
	}

	// Check that it contains altitude
	if !strings.Contains(result, "45.0,M") {
		t.Errorf("generateGGA should contain altitude '45.0,M', got: %s", result)
	}

	// Check that it ends with checksum and CRLF
	if !strings.Contains(result, "*") || !strings.HasSuffix(result, "\r\n") {
		t.Errorf("generateGGA should end with checksum and CRLF, got: %s", result)
	}
}

func TestGenerateRMC(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = true
	sim.currentLat = 37.7749
	sim.currentLon = -122.4194
	sim.currentSpeed = 5.5
	sim.currentCourse = 90.0

	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	result := sim.generateRMC(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPRMC,") {
		t.Errorf("generateRMC should start with '$GPRMC,', got: %s", result)
	}

	// Check that it contains expected time and date
	if !strings.Contains(result, "103045") {
		t.Errorf("generateRMC should contain time '103045', got: %s", result)
	}
	if !strings.Contains(result, "150124") {
		t.Errorf("generateRMC should contain date '150124', got: %s", result)
	}

	// Check status (should be 'A' for active when locked)
	parts := strings.Split(result, ",")
	if len(parts) > 2 && parts[2] != "A" {
		t.Errorf("generateRMC should have status 'A' when locked, got: %s", parts[2])
	}

	// Check that it contains coordinates
	if !strings.Contains(result, "3746.4940,N") {
		t.Errorf("generateRMC should contain latitude '3746.4940,N', got: %s", result)
	}
	if !strings.Contains(result, "12225.1640,W") {
		t.Errorf("generateRMC should contain longitude '12225.1640,W', got: %s", result)
	}

	// Check speed and course
	if !strings.Contains(result, "5.5") {
		t.Errorf("generateRMC should contain speed '5.5', got: %s", result)
	}
	if !strings.Contains(result, "90.0") {
		t.Errorf("generateRMC should contain course '90.0', got: %s", result)
	}
}

func TestGenerateGLL(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = true
	sim.currentLat = 37.7749
	sim.currentLon = -122.4194

	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	result := sim.generateGLL(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPGLL,") {
		t.Errorf("generateGLL should start with '$GPGLL,', got: %s", result)
	}

	// Check that it contains coordinates
	if !strings.Contains(result, "3746.4940,N") {
		t.Errorf("generateGLL should contain latitude '3746.4940,N', got: %s", result)
	}
	if !strings.Contains(result, "12225.1640,W") {
		t.Errorf("generateGLL should contain longitude '12225.1640,W', got: %s", result)
	}

	// Check that it contains time
	if !strings.Contains(result, "103045") {
		t.Errorf("generateGLL should contain time '103045', got: %s", result)
	}

	// Check status (should be 'A' for active when locked)
	parts := strings.Split(result, ",")
	if len(parts) > 6 && parts[6] != "A" {
		t.Errorf("generateGLL should have status 'A' when locked, got: %s", parts[6])
	}
}

func TestGenerateVTG(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = true
	sim.currentSpeed = 10.5
	sim.currentCourse = 45.0

	result := sim.generateVTG()

	// Check basic format
	if !strings.HasPrefix(result, "$GPVTG,") {
		t.Errorf("generateVTG should start with '$GPVTG,', got: %s", result)
	}

	// Check that it contains course
	if !strings.Contains(result, "45.0,T") {
		t.Errorf("generateVTG should contain course '45.0,T', got: %s", result)
	}

	// Check that it contains speed in knots
	if !strings.Contains(result, "10.5,N") {
		t.Errorf("generateVTG should contain speed '10.5,N', got: %s", result)
	}

	// Check that it contains speed in km/h (knots * 1.852)
	expectedKmh := 10.5 * 1.852
	expectedKmhStr := fmt.Sprintf("%.1f,K", expectedKmh)
	if !strings.Contains(result, expectedKmhStr) {
		t.Errorf("generateVTG should contain speed '%s', got: %s", expectedKmhStr, result)
	}
}

func TestGenerateGSA(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = true

	result := sim.generateGSA()

	// Check basic format
	if !strings.HasPrefix(result, "$GPGSA,") {
		t.Errorf("generateGSA should start with '$GPGSA,', got: %s", result)
	}

	// Check mode (should be 'A' for automatic)
	parts := strings.Split(result, ",")
	if len(parts) > 1 && parts[1] != "A" {
		t.Errorf("generateGSA should have mode 'A', got: %s", parts[1])
	}

	// Check fix type (should be '3' for 3D fix when locked)
	if len(parts) > 2 && parts[2] != "3" {
		t.Errorf("generateGSA should have fix type '3' when locked, got: %s", parts[2])
	}

	// Check that satellite IDs are present
	foundSatelliteIDs := false
	for i := 3; i < 15 && i < len(parts); i++ {
		if parts[i] != "" {
			foundSatelliteIDs = true
			break
		}
	}
	if !foundSatelliteIDs {
		t.Error("generateGSA should contain satellite IDs")
	}

	// Check DOP values are present
	if len(parts) < 18 {
		t.Errorf("generateGSA should have 18 parts, got %d", len(parts))
	}
}

func TestGenerateGSV(t *testing.T) {
	sim := createTestSimulator()

	result := sim.generateGSV()

	// Should return multiple sentences for 8 satellites (2 sentences)
	expectedSentences := (len(sim.satellites) + 3) / 4
	if len(result) != expectedSentences {
		t.Errorf("generateGSV should return %d sentences for %d satellites, got %d",
			expectedSentences, len(sim.satellites), len(result))
	}

	for i, sentence := range result {
		// Check basic format
		if !strings.HasPrefix(sentence, "$GPGSV,") {
			t.Errorf("generateGSV sentence %d should start with '$GPGSV,', got: %s", i, sentence)
		}

		parts := strings.Split(sentence, ",")
		if len(parts) < 4 {
			t.Errorf("generateGSV sentence %d should have at least 4 parts, got %d", i, len(parts))
			continue
		}

		// Check total sentences count
		totalSentences := parts[1]
		if totalSentences != fmt.Sprintf("%d", expectedSentences) {
			t.Errorf("generateGSV sentence %d should show total sentences %d, got %s",
				i, expectedSentences, totalSentences)
		}

		// Check sentence number
		sentenceNum := parts[2]
		if sentenceNum != fmt.Sprintf("%d", i+1) {
			t.Errorf("generateGSV sentence %d should show sentence number %d, got %s",
				i, i+1, sentenceNum)
		}

		// Check total satellites in view
		totalSats := parts[3]
		if totalSats != fmt.Sprintf("%02d", len(sim.satellites)) {
			t.Errorf("generateGSV sentence %d should show total satellites %02d, got %s",
				i, len(sim.satellites), totalSats)
		}
	}
}

func TestGenerateZDA(t *testing.T) {
	sim := createTestSimulator()

	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	result := sim.generateZDA(testTime)

	// Check basic format
	if !strings.HasPrefix(result, "$GPZDA,") {
		t.Errorf("generateZDA should start with '$GPZDA,', got: %s", result)
	}

	// Check that it contains expected time and date components
	if !strings.Contains(result, "103045") {
		t.Errorf("generateZDA should contain time '103045', got: %s", result)
	}
	if !strings.Contains(result, "15") {
		t.Errorf("generateZDA should contain day '15', got: %s", result)
	}
	if !strings.Contains(result, "01") {
		t.Errorf("generateZDA should contain month '01', got: %s", result)
	}
	if !strings.Contains(result, "2024") {
		t.Errorf("generateZDA should contain year '2024', got: %s", result)
	}
}

func TestNoFixSentences(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = false // Ensure not locked

	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	// Test no-fix GGA
	gga := sim.generateNoFixGGA(testTime)
	if !strings.HasPrefix(gga, "$GPGGA,") {
		t.Errorf("generateNoFixGGA should start with '$GPGGA,', got: %s", gga)
	}
	if !strings.Contains(gga, "103045") {
		t.Errorf("generateNoFixGGA should contain time '103045', got: %s", gga)
	}
	// Should have empty coordinates and quality 0
	parts := strings.Split(gga, ",")
	if len(parts) > 6 && parts[6] != "0" {
		t.Errorf("generateNoFixGGA should have quality '0', got: %s", parts[6])
	}

	// Test no-fix RMC
	rmc := sim.generateNoFixRMC(testTime)
	if !strings.HasPrefix(rmc, "$GPRMC,") {
		t.Errorf("generateNoFixRMC should start with '$GPRMC,', got: %s", rmc)
	}
	if !strings.Contains(rmc, "103045") {
		t.Errorf("generateNoFixRMC should contain time '103045', got: %s", rmc)
	}
	// Should have status 'V' for invalid
	parts = strings.Split(rmc, ",")
	if len(parts) > 2 && parts[2] != "V" {
		t.Errorf("generateNoFixRMC should have status 'V', got: %s", parts[2])
	}

	// Test no-fix GLL
	gll := sim.generateNoFixGLL(testTime)
	if !strings.HasPrefix(gll, "$GPGLL,") {
		t.Errorf("generateNoFixGLL should start with '$GPGLL,', got: %s", gll)
	}
	// Should have status 'V' for invalid
	parts = strings.Split(gll, ",")
	if len(parts) > 6 && parts[6] != "V" {
		t.Errorf("generateNoFixGLL should have status 'V', got: %s", parts[6])
	}

	// Test no-fix VTG
	vtg := sim.generateNoFixVTG()
	if !strings.HasPrefix(vtg, "$GPVTG,") {
		t.Errorf("generateNoFixVTG should start with '$GPVTG,', got: %s", vtg)
	}
	// Should have empty fields and mode 'N'
	parts = strings.Split(vtg, ",")
	lastPart := strings.Split(parts[len(parts)-1], "*")[0] // Remove checksum
	if lastPart != "N" {
		t.Errorf("generateNoFixVTG should end with mode 'N', got: %s", lastPart)
	}
}

func TestNMEAChecksumValidation(t *testing.T) {
	// Test various NMEA sentences to ensure checksums are calculated correctly
	testSentences := []string{
		"$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,",
		"$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W",
		"$GPGLL,4807.038,N,01131.000,E,123519,A,A",
		"$GPVTG,054.7,T,034.4,M,005.5,N,010.2,K",
		"$GPGSA,A,3,01,02,03,04,05,06,07,08,09,10,11,12,1.0,1.0,1.0",
	}

	for _, sentence := range testSentences {
		formatted := formatNMEA(sentence)

		// Extract the checksum
		parts := strings.Split(formatted, "*")
		if len(parts) != 2 {
			t.Errorf("Formatted sentence should have exactly one '*': %s", formatted)
			continue
		}

		checksumPart := strings.TrimSuffix(parts[1], "\r\n")
		expectedChecksum := calculateChecksum(sentence)

		if checksumPart != expectedChecksum {
			t.Errorf("Checksum mismatch for %s: expected %s, got %s",
				sentence, expectedChecksum, checksumPart)
		}
	}
}

func TestOutputNMEALocked(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = true
	buffer := &bytes.Buffer{}
	sim.SetNMEAWriter(buffer)

	sim.outputNMEA()
	output := buffer.String()

	// Should contain all sentence types when locked
	expectedSentences := []string{"$GPGGA,", "$GPRMC,", "$GPGLL,", "$GPVTG,", "$GPGSA,", "$GPGSV,", "$GPZDA,"}
	for _, sentence := range expectedSentences {
		if !strings.Contains(output, sentence) {
			t.Errorf("Output should contain %s sentence when locked", sentence)
		}
	}

	// Count GSV sentences (should be based on number of satellites)
	gsvCount := strings.Count(output, "$GPGSV,")
	expectedGSVCount := (len(sim.satellites) + 3) / 4 // Round up to nearest 4
	if gsvCount != expectedGSVCount {
		t.Errorf("Expected %d GSV sentences, got %d", expectedGSVCount, gsvCount)
	}

	// Verify all sentences end with \r\n
	sentences := strings.Split(output, "\r\n")
	for i, sentence := range sentences {
		if sentence == "" {
			continue // Skip empty strings from split
		}
		if !strings.HasPrefix(sentence, "$GP") {
			t.Errorf("Sentence %d should start with $GP: %s", i, sentence)
		}
		if !strings.Contains(sentence, "*") {
			t.Errorf("Sentence %d should contain checksum separator: %s", i, sentence)
		}
	}
}

func TestOutputNMEANotLocked(t *testing.T) {
	sim := createTestSimulator()
	sim.isLocked = false
	buffer := &bytes.Buffer{}
	sim.SetNMEAWriter(buffer)

	sim.outputNMEA()
	output := buffer.String()

	// Should contain no-fix sentences
	if !strings.Contains(output, "$GPGGA,") {
		t.Error("Output should contain GGA sentence when not locked")
	}
	if !strings.Contains(output, "$GPRMC,") {
		t.Error("Output should contain RMC sentence when not locked")
	}
	if !strings.Contains(output, "$GPGLL,") {
		t.Error("Output should contain GLL sentence when not locked")
	}
	if !strings.Contains(output, "$GPVTG,") {
		t.Error("Output should contain VTG sentence when not locked")
	}

	// Should not contain GSA, GSV, or ZDA when not locked
	if strings.Contains(output, "$GPGSA,") {
		t.Error("Output should not contain GSA sentence when not locked")
	}
	if strings.Contains(output, "$GPGSV,") {
		t.Error("Output should not contain GSV sentence when not locked")
	}
	if strings.Contains(output, "$GPZDA,") {
		t.Error("Output should not contain ZDA sentence when not locked")
	}
}

func TestCoordinateConversion(t *testing.T) {
	tests := []struct {
		name           string
		lat            float64
		lon            float64
		expectedLatDMS string
		expectedLonDMS string
		expectedLatHem string
		expectedLonHem string
	}{
		{
			name:           "San Francisco",
			lat:            37.7749,
			lon:            -122.4194,
			expectedLatDMS: "3746.4940",
			expectedLonDMS: "12225.1640",
			expectedLatHem: "N",
			expectedLonHem: "W",
		},
		{
			name:           "Sydney",
			lat:            -33.8688,
			lon:            151.2093,
			expectedLatDMS: "3352.1280",
			expectedLonDMS: "15112.5580",
			expectedLatHem: "S",
			expectedLonHem: "E",
		},
		{
			name:           "London",
			lat:            51.5074,
			lon:            -0.1278,
			expectedLatDMS: "5130.4440",
			expectedLonDMS: "00007.6680",
			expectedLatHem: "N",
			expectedLonHem: "W",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := createTestSimulator()
			sim.isLocked = true
			sim.currentLat = tt.lat
			sim.currentLon = tt.lon

			testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
			gga := sim.generateGGA(testTime)

			if !strings.Contains(gga, tt.expectedLatDMS+","+tt.expectedLatHem) {
				t.Errorf("GGA should contain latitude '%s,%s', got: %s",
					tt.expectedLatDMS, tt.expectedLatHem, gga)
			}
			if !strings.Contains(gga, tt.expectedLonDMS+","+tt.expectedLonHem) {
				t.Errorf("GGA should contain longitude '%s,%s', got: %s",
					tt.expectedLonDMS, tt.expectedLonHem, gga)
			}
		})
	}
}
