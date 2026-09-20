package aprs

import "strings"

// PacketType classifies a raw APRS packet into the station type codes used
// by Yaesu FTM-series radios (shown next to the symbol in the station list).
//
// Yaesu FTM-series APRS station type display:
//
//	Display  Description
//	E        Mic-E: Displayed when a beacon of a Mic-E station is received
//	P        Position: Displayed when the beacon from a fixed station (FIXED)
//	         or a mobile station (MOVING) is received
//	p        Position: Displayed when the beacon of a fixed station (FIXED)
//	         or a mobile station (MOVING) is received (compression type)
//	W        Weather report: Displayed when the beacon of a meteorological
//	         station is received
//	w        Weather report: Displayed when the beacon of a meteorological
//	         station is received (compression type)
//	O        Object: Displayed when the beacon of an object station is received
//	o        Object: Displayed when the beacon of an object station is received
//	         (compression type)
//	I        Item: Displayed when the beacon of an item station is received
//	i        Item: Displayed when the beacon of an item station is received
//	         (compression type)
//	K        Killed Object/Item: Displayed when a deleted object station or
//	         item station is received
//	k        Killed Object/Item: Displayed when a deleted object station or
//	         item station is received (compression type)
//	S        Status: Displayed when the beacon of a status station is received
//	G        Raw NMEA: Displayed when Raw NMEA data (GGA / GLL / RMC) is received
//	?        Other: Displayed when a beacon that cannot be interpreted is received
//	Emg      Displayed when an emergency signal from a Mic-E station is received
//
// Emergency (Emg) is not detected yet — Mic-E emergency decoding may be added
// later. Killed objects are not stored in the cache, so K/k only appear for
// packets classified directly.
func PacketType(raw string) string {
	if raw == "" {
		return ""
	}
	// Most packets are CALL>DEST,PATH:BODY; objects (;) and items ())
	// carry no header and are their own body.
	body := raw
	if idx := strings.IndexByte(raw, ':'); idx >= 0 {
		body = raw[idx+1:]
	}
	if body == "" {
		return "?"
	}
	switch body[0] {
	case '\'', '`': // Mic-E position
		return "E"
	case ';': // Object
		return "O"
	case ')': // Item
		return "I"
	case '_': // Weather report
		return "W"
	case '$': // Raw NMEA
		return "G"
	case '>': // Status
		return "S"
	case '@', '=': // Uncompressed position (with timestamp / course-speed)
		return "P"
	case '!', '/':
		// Ambiguous: uncompressed positions start with a digit, Base-91
		// compressed ones with a symbol-table character.
		if len(body) > 1 && !(body[1] >= '0' && body[1] <= '9') {
			return "p"
		}
		return "P"
	default:
		return "?"
	}
}
