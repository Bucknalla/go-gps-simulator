package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// calculateChecksum calculates the NMEA checksum for a sentence
func calculateChecksum(sentence string) string {
	var checksum byte
	for i := 1; i < len(sentence); i++ { // Skip the '$' character
		checksum ^= sentence[i]
	}
	return fmt.Sprintf("%02X", checksum)
}

// formatNMEA formats a complete NMEA sentence with checksum
func formatNMEA(sentence string) string {
	checksum := calculateChecksum(sentence)
	return fmt.Sprintf("%s*%s\r\n", sentence, checksum)
}

// generateGGA generates a GGA (Global Positioning System Fix Data) sentence
func (s *GPSSimulator) generateGGA(timestamp time.Time) string {
	timeStr := timestamp.UTC().Format("150405") // HHMMSS

	// Convert coordinates to NMEA format (DDMM.MMMMM)
	latDeg := int(math.Abs(s.currentLat))
	latMin := (math.Abs(s.currentLat) - float64(latDeg)) * 60
	latHem := "N"
	if s.currentLat < 0 {
		latHem = "S"
	}

	lonDeg := int(math.Abs(s.currentLon))
	lonMin := (math.Abs(s.currentLon) - float64(lonDeg)) * 60
	lonHem := "E"
	if s.currentLon < 0 {
		lonHem = "W"
	}

	// Quality indicator: 1 = GPS fix
	quality := "1"
	numSats := fmt.Sprintf("%02d", len(s.satellites))
	hdop := "1.2"                                 // Horizontal dilution of precision
	altitude := fmt.Sprintf("%.1f", s.currentAlt) // Current altitude above mean sea level
	altUnit := "M"
	geoidSep := "0.0" // Geoidal separation
	sepUnit := "M"
	dgpsAge := "" // Age of DGPS data
	dgpsID := ""  // DGPS station ID

	sentence := fmt.Sprintf("$GPGGA,%s,%02d%07.4f,%s,%03d%07.4f,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s",
		timeStr,
		latDeg, latMin, latHem,
		lonDeg, lonMin, lonHem,
		quality, numSats, hdop,
		altitude, altUnit,
		geoidSep, sepUnit,
		dgpsAge, dgpsID)

	return formatNMEA(sentence)
}

// generateNoFixGGA generates a GGA sentence when there's no GPS fix
func (s *GPSSimulator) generateNoFixGGA(timestamp time.Time) string {
	timeStr := timestamp.UTC().Format("150405")

	sentence := fmt.Sprintf("$GPGGA,%s,,,,,0,00,,,,,,,,,", timeStr)
	return formatNMEA(sentence)
}

// generateRMC generates an RMC (Recommended Minimum) sentence
func (s *GPSSimulator) generateRMC(timestamp time.Time) string {
	timeStr := timestamp.UTC().Format("150405") // HHMMSS
	dateStr := timestamp.UTC().Format("020106") // DDMMYY

	// Convert coordinates to NMEA format
	latDeg := int(math.Abs(s.currentLat))
	latMin := (math.Abs(s.currentLat) - float64(latDeg)) * 60
	latHem := "N"
	if s.currentLat < 0 {
		latHem = "S"
	}

	lonDeg := int(math.Abs(s.currentLon))
	lonMin := (math.Abs(s.currentLon) - float64(lonDeg)) * 60
	lonHem := "E"
	if s.currentLon < 0 {
		lonHem = "W"
	}

	status := "A"                                  // A = Active, V = Void
	speed := fmt.Sprintf("%.1f", s.currentSpeed)   // Speed over ground in knots (with jitter applied)
	course := fmt.Sprintf("%.1f", s.currentCourse) // Course over ground in degrees (with jitter applied)
	magVar := ""                                   // Magnetic variation
	magVarDir := ""                                // Direction of magnetic variation
	mode := "A"                                    // A = Autonomous, D = DGPS, E = DR

	sentence := fmt.Sprintf("$GPRMC,%s,%s,%02d%07.4f,%s,%03d%07.4f,%s,%s,%s,%s,%s,%s,%s",
		timeStr, status,
		latDeg, latMin, latHem,
		lonDeg, lonMin, lonHem,
		speed, course, dateStr,
		magVar, magVarDir, mode)

	return formatNMEA(sentence)
}

// generateNoFixRMC generates an RMC sentence when there's no GPS fix
func (s *GPSSimulator) generateNoFixRMC(timestamp time.Time) string {
	timeStr := timestamp.UTC().Format("150405")
	dateStr := timestamp.UTC().Format("020106")

	sentence := fmt.Sprintf("$GPRMC,%s,V,,,,,,,,%s,,,N", timeStr, dateStr)
	return formatNMEA(sentence)
}

// generateGSA generates a GSA (GPS DOP and active satellites) sentence
func (s *GPSSimulator) generateGSA() string {
	mode1 := "A" // A = Automatic, M = Manual
	mode2 := "3" // 1 = No fix, 2 = 2D fix, 3 = 3D fix

	// List up to 12 satellite IDs being used for fix
	var satIDs []string
	for i, sat := range s.satellites {
		if i < 12 {
			satIDs = append(satIDs, fmt.Sprintf("%02d", sat.ID))
		}
	}

	// Pad with empty fields to make 12 total
	for len(satIDs) < 12 {
		satIDs = append(satIDs, "")
	}

	pdop := "2.1" // Position dilution of precision
	hdop := "1.2" // Horizontal dilution of precision
	vdop := "1.8" // Vertical dilution of precision

	sentence := fmt.Sprintf("$GPGSA,%s,%s,%s,%s,%s,%s",
		mode1, mode2,
		strings.Join(satIDs, ","),
		pdop, hdop, vdop)

	return formatNMEA(sentence)
}

// generateGSV generates GSV (GPS Satellites in view) sentences
func (s *GPSSimulator) generateGSV() []string {
	var sentences []string

	totalSats := len(s.satellites)
	totalSentences := (totalSats + 3) / 4 // Round up to nearest 4

	for sentenceNum := 1; sentenceNum <= totalSentences; sentenceNum++ {
		startIdx := (sentenceNum - 1) * 4
		endIdx := startIdx + 4
		if endIdx > totalSats {
			endIdx = totalSats
		}

		sentence := fmt.Sprintf("$GPGSV,%d,%d,%02d",
			totalSentences, sentenceNum, totalSats)

		// Add satellite data (up to 4 satellites per sentence)
		for i := startIdx; i < endIdx; i++ {
			sat := s.satellites[i]
			sentence += fmt.Sprintf(",%02d,%02d,%03d,%02d",
				sat.ID, sat.Elevation, sat.Azimuth, sat.SNR)
		}

		// Pad with empty fields if less than 4 satellites in this sentence
		fieldsToAdd := 4 - (endIdx - startIdx)
		for i := 0; i < fieldsToAdd; i++ {
			sentence += ",,,,"
		}

		sentences = append(sentences, formatNMEA(sentence))
	}

	return sentences
}

// generateVTG generates a VTG (Track Made Good and Ground Speed) sentence
func (s *GPSSimulator) generateVTG() string {
	// Course over ground (true)
	courseTrue := fmt.Sprintf("%.1f", s.currentCourse)
	courseTrueRef := "T" // T = True

	// Course over ground (magnetic) - we'll leave this empty as we don't simulate magnetic variation
	courseMagnetic := ""
	courseMagneticRef := "M" // M = Magnetic

	// Speed over ground in knots
	speedKnots := fmt.Sprintf("%.1f", s.currentSpeed)
	speedKnotsUnit := "N" // N = Knots

	// Speed over ground in kilometers per hour
	// 1 knot = 1.852 km/h
	speedKmh := fmt.Sprintf("%.1f", s.currentSpeed*1.852)
	speedKmhUnit := "K" // K = Kilometers per hour

	mode := "A" // A = Autonomous, D = DGPS, E = DR

	sentence := fmt.Sprintf("$GPVTG,%s,%s,%s,%s,%s,%s,%s,%s,%s",
		courseTrue, courseTrueRef,
		courseMagnetic, courseMagneticRef,
		speedKnots, speedKnotsUnit,
		speedKmh, speedKmhUnit,
		mode)

	return formatNMEA(sentence)
}

// generateNoFixVTG generates a VTG sentence when there's no GPS fix
func (s *GPSSimulator) generateNoFixVTG() string {
	sentence := "$GPVTG,,,,,,,,,N" // N = Not valid
	return formatNMEA(sentence)
}

// generateGLL generates a GLL (Geographic Position - Latitude/Longitude) sentence
func (s *GPSSimulator) generateGLL(timestamp time.Time) string {
	utcTime := timestamp.UTC()
	timeStr := fmt.Sprintf("%02d%02d%02d.%02d",
		utcTime.Hour(), utcTime.Minute(), utcTime.Second(), utcTime.Nanosecond()/10000000) // HHMMSS.SS

	// Convert coordinates to NMEA format (DDMM.MMMMM)
	latDeg := int(math.Abs(s.currentLat))
	latMin := (math.Abs(s.currentLat) - float64(latDeg)) * 60
	latHem := "N"
	if s.currentLat < 0 {
		latHem = "S"
	}

	lonDeg := int(math.Abs(s.currentLon))
	lonMin := (math.Abs(s.currentLon) - float64(lonDeg)) * 60
	lonHem := "E"
	if s.currentLon < 0 {
		lonHem = "W"
	}

	status := "A" // A = Data valid, V = Data invalid
	mode := "A"   // A = Autonomous, D = DGPS, E = DR

	sentence := fmt.Sprintf("$GPGLL,%02d%07.4f,%s,%03d%07.4f,%s,%s,%s,%s",
		latDeg, latMin, latHem,
		lonDeg, lonMin, lonHem,
		timeStr, status, mode)

	return formatNMEA(sentence)
}

// generateNoFixGLL generates a GLL sentence when there's no GPS fix
func (s *GPSSimulator) generateNoFixGLL(timestamp time.Time) string {
	utcTime := timestamp.UTC()
	timeStr := fmt.Sprintf("%02d%02d%02d.%02d",
		utcTime.Hour(), utcTime.Minute(), utcTime.Second(), utcTime.Nanosecond()/10000000) // HHMMSS.SS

	sentence := fmt.Sprintf("$GPGLL,,,,,%s,V,N", timeStr) // V = Invalid, N = Not valid
	return formatNMEA(sentence)
}

// generateZDA generates a ZDA (UTC Date and Time) sentence
func (s *GPSSimulator) generateZDA(timestamp time.Time) string {
	utcTime := timestamp.UTC()

	timeStr := fmt.Sprintf("%02d%02d%02d.%02d",
		utcTime.Hour(), utcTime.Minute(), utcTime.Second(), utcTime.Nanosecond()/10000000) // HHMMSS.SS
	day := fmt.Sprintf("%02d", utcTime.Day())
	month := fmt.Sprintf("%02d", utcTime.Month())
	year := fmt.Sprintf("%04d", utcTime.Year())

	// Local zone hours and minutes (we'll use UTC, so both are 00)
	localZoneHours := "00"
	localZoneMinutes := "00"

	sentence := fmt.Sprintf("$GPZDA,%s,%s,%s,%s,%s,%s",
		timeStr, day, month, year, localZoneHours, localZoneMinutes)

	return formatNMEA(sentence)
}
