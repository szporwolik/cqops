package aprs

import (
	"math"
	"strconv"
	"strings"
)

// ParsePositionPacket attempts to decode an APRS position report from a raw
// packet string. Returns the StationRecord and true if successful.
//
// Supported formats:
//   - Uncompressed: !DDMM.hhN/DDDMM.hhWr/...
//   - Uncompressed with course/speed: =DDMM.hhN/DDDMM.hhWr/...
//   - Uncompressed with timestamp: @DDHHMMzDDMM.hhN/DDDMM.hhWr/...
//   - Mic-E compressed: '0...  or  `0...
//   - Base-91 compressed: !L54BLSVg@#  (LoRa/Direwolf)
//
// Symbols: /r = car, /[ = runner, /> = car, /- = house, etc.
func ParsePositionPacket(raw string) (StationRecord, bool) {
	var sr StationRecord

	// Split header from body: CALLSIGN>DEST,PATH:BODY
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) < 2 {
		return sr, false
	}

	// Extract callsign from header.
	header := parts[0]
	if idx := strings.Index(header, ">"); idx >= 0 {
		sr.Callsign = strings.TrimSpace(header[:idx])
	} else {
		return sr, false
	}
	if sr.Callsign == "" {
		return sr, false
	}

	body := parts[1]
	if len(body) < 2 {
		return sr, false
	}

	dataType := body[0]

	// Mic-E compressed position (data types ' and `).
	if dataType == '\'' || dataType == '`' {
		if tryDecodeMicE(header, body, &sr) {
			return sr, true
		}
		return sr, false
	}

	// Object: ;OBJNAME  *HHMMSSh<position>
	// The object name is used as the callsign for the map marker.
	if dataType == ';' && tryDecodeObject(body, &sr) {
		return sr, true
	}

	// Item: )ITEMNAME!<position> or )ITEMNAME_<position>
	if dataType == ')' && tryDecodeItem(body, &sr) {
		return sr, true
	}

	// ---- Uncompressed position formats ----
	// '!' is ambiguous: it starts both uncompressed (!DDMM.hhN/...) and
	// Base-91 compressed (!L54BLSVg@#) positions. Try Base-91 first
	// (strict charset validation), then fall back to uncompressed.

	bodyUncomp := body
	switch dataType {
	case '@':
		if len(bodyUncomp) < 8 {
			return sr, false
		}
		bodyUncomp = bodyUncomp[8:] // skip @ and timestamp
		if len(bodyUncomp) < 10 {
			return sr, false
		}
	case '!', '=', '/':
		bodyUncomp = bodyUncomp[1:] // skip data type indicator
	default:
		return sr, false
	}

	// For '!', '=', and '/', the format is ambiguous (uncompressed vs Base-91).
	// Heuristic: uncompressed always starts with two digits (lat degrees).
	// Base-91 compressed starts with a symbol table character (/\A-Za-j).
	if (dataType == '!' || dataType == '=' || dataType == '/') && len(body) > 2 &&
		!(body[1] >= '0' && body[1] <= '9') {
		if tryDecodeBase91(body, &sr) {
			return sr, true
		}
	}

	if tryDecodeUncompressed(bodyUncomp, &sr) {
		return sr, true
	}

	return sr, false
}

// tryDecodeUncompressed parses standard uncompressed APRS positions:
//
//	DDMM.hhN/DDDMM.hhWr/...comment
func tryDecodeUncompressed(body string, sr *StationRecord) bool {

	// Parse latitude: DDMM.hhX (X = N or S)
	if len(body) < 9 {
		return false
	}
	latDeg, err := strconv.Atoi(body[0:2])
	if err != nil {
		return false
	}
	latMin, err := strconv.ParseFloat(body[2:7], 64)
	if err != nil {
		return false
	}
	latHemi := body[7]

	// Parse symbol table and longitude.
	body = body[8:]
	if len(body) < 1 {
		return false
	}
	symTable := body[0]
	body = body[1:]

	// Parse longitude: DDDMM.hhX (X = E or W)
	if len(body) < 10 {
		return false
	}
	lonDeg, err := strconv.Atoi(body[0:3])
	if err != nil {
		return false
	}
	lonMin, err := strconv.ParseFloat(body[3:8], 64)
	if err != nil {
		return false
	}
	lonHemi := body[8]
	symCode := body[9]

	sr.Symbol = string(symTable) + string(symCode)

	lat := float64(latDeg) + latMin/60.0
	if latHemi == 'S' {
		lat = -lat
	}
	lon := float64(lonDeg) + lonMin/60.0
	if lonHemi == 'W' {
		lon = -lon
	}

	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return false
	}

	sr.Lat = math.Round(lat*10000) / 10000
	sr.Lon = math.Round(lon*10000) / 10000

	// Optional: course/speed and comment after the symbol.
	body = body[10:]
	if len(body) >= 7 && body[0] == '/' {
		courseEnd := strings.IndexByte(body[1:], '/')
		if courseEnd >= 0 && courseEnd <= 3 {
			courseEnd++
			if c, err := strconv.Atoi(body[1:courseEnd]); err == nil && c >= 0 && c <= 360 {
				sr.Course = c
			}
			body = body[courseEnd+1:]
		}
	}
	sr.Comment = strings.TrimSpace(body)

	return true
}

// ===========================================================================
// Mic-E compressed position decoder
// ===========================================================================
//
// Mic-E reuses the AX.25 destination address field to encode latitude and
// longitude offset, then packs the remainder (longitude, course, speed,
// symbol) into the information field. Based on go-aprs-fap by Hessu.
//
// Data type indicators:
//
//	'  (0x27) — original Mic-E (pre-GPS)
//	`  (0x60) — newer Mic-E (GPS-based, most common today)

// micEDestEntry holds the decoded digit and flags for a Mic-E destination char.
type micEDestEntry struct {
	digit   int
	msgBit  int // 0=standard, 1=custom
	isNorth bool
}

// micEDestTable maps destination address characters to their Mic-E values.
// Characters are ASCII values from '0' to 'Z'. Full range from go-aprs-fap.
var micEDestTable = [256]micEDestEntry{}

func init() {
	// Build the Mic-E destination table (same values as go-aprs-fap/perl-aprs-fap).
	// Standard digits 0-9 (ASCII 48-57): digit = value, msgBit = 0.
	for c := '0'; c <= '9'; c++ {
		micEDestTable[c] = micEDestEntry{digit: int(c - '0'), msgBit: 0}
	}
	// A-I (65-73): digit = c-'A', msgBit = 1.
	for c := 'A'; c <= 'I'; c++ {
		micEDestTable[c] = micEDestEntry{digit: int(c - 'A'), msgBit: 1}
	}
	// J (74): digit = 9, msgBit = 1.
	micEDestTable['J'] = micEDestEntry{digit: 9, msgBit: 1}
	// K (75): digit = 0, msgBit = 1.
	micEDestTable['K'] = micEDestEntry{digit: 0, msgBit: 1}
	// L (76): digit = 0, msgBit = 0.
	micEDestTable['L'] = micEDestEntry{digit: 0, msgBit: 0}
	// P-Z (80-90): digit = c-'P', msgBit = 1, isNorth = true.
	for c := 'P'; c <= 'Z'; c++ {
		micEDestTable[c] = micEDestEntry{digit: int(c - 'P'), msgBit: 1, isNorth: true}
	}
}

func tryDecodeMicE(header, body string, sr *StationRecord) bool {
	// Extract destination address from header: CALLSIGN>DEST,PATH
	rest := header[strings.IndexByte(header, '>')+1:]
	dest := rest
	if idx := strings.IndexByte(rest, ','); idx >= 0 {
		dest = rest[:idx]
	}
	// Strip SSID.
	if idx := strings.LastIndexByte(dest, '-'); idx >= 0 {
		dest = dest[:idx]
	}

	if len(dest) < 6 {
		return false
	}

	// Skip data type indicator (` or ') — body[0] is the actual indicator.
	body = body[1:]
	if len(body) < 8 {
		return false
	}

	// Decode latitude from destination (same algorithm as go-aprs-fap).
	latDigits := make([]int, 6)
	isNorth := false
	lonOffset := 0
	isWest := false

	for i := 0; i < 6; i++ {
		c := dest[i]
		entry := micEDestTable[c]
		if entry.digit > 9 && i == 0 {
			// d1 must have a digit 0-9; entries not in table have zero-value.
		}
		latDigits[i] = entry.digit

		if i == 3 && entry.isNorth {
			isNorth = true
		}
		if i == 4 && c >= 'P' && c <= 'Z' {
			lonOffset = 100
		}
		if i == 5 && c >= 'P' && c <= 'Z' {
			isWest = true
		}
	}

	latDeg := float64(latDigits[0]*10 + latDigits[1])
	latMin := float64(latDigits[2]*10+latDigits[3]) + float64(latDigits[4]*10+latDigits[5])/100.0
	lat := latDeg + latMin/60.0
	if !isNorth {
		lat = -lat
	}

	// Decode longitude from information field.
	lonDeg := int(body[0]) - 28 + lonOffset
	if lonDeg >= 180 && lonDeg <= 189 {
		lonDeg -= 80
	} else if lonDeg >= 190 && lonDeg <= 199 {
		lonDeg -= 190
	}

	lonMin := int(body[1]) - 28
	if lonMin >= 60 {
		lonMin -= 60
	}
	lonHMin := int(body[2]) - 28

	lon := float64(lonDeg) + (float64(lonMin)+float64(lonHMin)/100.0)/60.0
	if isWest {
		lon = -lon
	}

	// Validate.
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return false
	}

	sr.Lat = math.Round(lat*10000) / 10000
	sr.Lon = math.Round(lon*10000) / 10000

	// Speed and course from bytes 3-5 (sp, dc, se).
	if len(body) >= 6 {
		sp := int(body[3]) - 28
		dc := int(body[4]) - 28
		se := int(body[5]) - 28
		speedKnots := float64(sp*10 + dc/10)
		if speedKnots >= 800 {
			speedKnots -= 800
		}
		course := (dc%10)*100 + se
		if course >= 400 {
			course -= 400
		}
		sr.Course = course
		sr.SpeedKmH = int(speedKnots * 1.852)
	}

	// Symbol table and code at bytes 6-7.
	// In Mic-E, body[6]=code, body[7]=table (the opposite of uncompressed).
	// Standard symbol order is "table+code".
	if len(body) >= 8 {
		sr.Symbol = string(body[7]) + string(body[6])
	}

	// Comment is everything after byte 7.
	if len(body) > 8 {
		sr.Comment = strings.TrimSpace(body[8:])
	}

	return true
}

// ===========================================================================
// Base-91 compressed position decoder
// ===========================================================================
//
// Base-91 compression packs latitude (90° range) and longitude (180° range)
// into 4 bytes each using ASCII value offset by 33 ('!' = 0).
//
// Format after '!' data type:
//
//	<sym_table><lat4><lon4><sym_code>
//
// Based on go-aprs-fap by Hessu.

func tryDecodeBase91(body string, sr *StationRecord) bool {
	// Skip the data type indicator.
	body = body[1:]
	if len(body) < 10 {
		return false
	}

	// Symbol table is the first byte.
	symTable := body[0]

	// Decode latitude from bytes 1-4.
	latVal := (int(body[1])-33)*91*91*91 +
		(int(body[2])-33)*91*91 +
		(int(body[3])-33)*91 +
		(int(body[4]) - 33)
	lat := 90.0 - float64(latVal)/380926.0

	// Decode longitude from bytes 5-8.
	lonVal := (int(body[5])-33)*91*91*91 +
		(int(body[6])-33)*91*91 +
		(int(body[7])-33)*91 +
		(int(body[8]) - 33)
	lon := -180.0 + float64(lonVal)/190463.0

	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return false
	}

	sr.Lat = math.Round(lat*10000) / 10000
	sr.Lon = math.Round(lon*10000) / 10000

	// Symbol code is at position 9.
	sr.Symbol = string(symTable) + string(body[9])

	// Comment follows the symbol.
	if len(body) > 10 {
		sr.Comment = strings.TrimSpace(body[10:])
	}

	return true
}

// ===========================================================================
// Object / Item decoders
// ===========================================================================
//
// Object: ;OBJNAME  *HHMMSSh<position>
// Item:   )ITEMNAME!<position> or )ITEMNAME_<position>
//
// Objects and Items are APRS constructs that let one station report the
// position of another entity (repeater, event, weather station, etc.).
// The object name (9 chars, space-padded) is used as the callsign for
// map display.

// tryDecodeObject parses an APRS Object packet.
// Format: ;OBJNAME  *HHMMSSh<position data>
func tryDecodeObject(body string, sr *StationRecord) bool {
	body = body[1:]     // skip ';'
	if len(body) < 27 { // 9 name + 1 alive + 7 timestamp + 10 compressed pos
		return false
	}

	// Object name is 9 characters.
	sr.Callsign = strings.TrimSpace(body[:9])
	if sr.Callsign == "" {
		return false
	}

	// Alive/killed indicator at position 9.
	aliveChar := body[9]
	if aliveChar != '*' && aliveChar != '_' {
		return false
	}

	// Skip timestamp at positions 10-16 (7 chars: HHMMSSh).
	posBody := body[17:]

	// Parse position using the uncompressed or compressed parsers.
	return parseObjectPosition(posBody, sr)
}

// tryDecodeItem parses an APRS Item packet.
// Format: )ITEMNAME!<position data> or )ITEMNAME_<position data>
func tryDecodeItem(body string, sr *StationRecord) bool {
	body = body[1:] // skip ')'
	if len(body) < 18 {
		return false
	}

	// Item name is 3-9 characters, terminated by ! (alive) or _ (killed).
	nameEnd := -1
	for i := 0; i < len(body) && i < 10; i++ {
		if body[i] == '!' || body[i] == '_' {
			nameEnd = i
			break
		}
	}
	if nameEnd < 0 {
		return false
	}

	sr.Callsign = body[:nameEnd]
	if sr.Callsign == "" {
		return false
	}

	// Position follows the alive/killed indicator.
	posBody := body[nameEnd+1:]
	return parseObjectPosition(posBody, sr)
}

// parseObjectPosition delegates to the uncompressed or compressed position
// parsers based on the first character of the position body.
func parseObjectPosition(posBody string, sr *StationRecord) bool {
	if len(posBody) < 10 {
		return false
	}

	// Uncompressed: starts with digit or space.
	if (posBody[0] >= '0' && posBody[0] <= '9') || posBody[0] == ' ' {
		return tryDecodeUncompressed(posBody, sr)
	}

	// Compressed: starts with symbol table character.
	if posBody[0] >= '!' && posBody[0] <= '{' {
		return tryDecodeBase91Position(posBody, sr)
	}

	return false
}

// tryDecodeBase91Position parses a compressed position without the leading '!'.
// Format: <sym_table><lat4><lon4><sym_code>
func tryDecodeBase91Position(body string, sr *StationRecord) bool {
	if len(body) < 10 {
		return false
	}

	symTable := body[0]

	latVal := (int(body[1])-33)*91*91*91 +
		(int(body[2])-33)*91*91 +
		(int(body[3])-33)*91 +
		(int(body[4]) - 33)
	lat := 90.0 - float64(latVal)/380926.0

	lonVal := (int(body[5])-33)*91*91*91 +
		(int(body[6])-33)*91*91 +
		(int(body[7])-33)*91 +
		(int(body[8]) - 33)
	lon := -180.0 + float64(lonVal)/190463.0

	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return false
	}

	sr.Lat = math.Round(lat*10000) / 10000
	sr.Lon = math.Round(lon*10000) / 10000
	sr.Symbol = string(symTable) + string(body[9])

	if len(body) > 10 {
		sr.Comment = strings.TrimSpace(body[10:])
	}

	return true
}
