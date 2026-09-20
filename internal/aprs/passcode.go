package aprs

import (
	"strconv"
	"strings"
)

// BaseCall normalizes an APRS callsign to its uppercase base form: the SSID
// ("-xx") and portable suffixes ("/P", "/M", ...) are stripped. Servers
// and own-station filtering compare the base callsign only.
func BaseCall(callsign string) string {
	c := strings.ToUpper(strings.TrimSpace(callsign))
	if i := strings.IndexByte(c, '-'); i >= 0 {
		c = c[:i]
	}
	if i := strings.IndexByte(c, '/'); i >= 0 {
		c = c[:i]
	}
	return c
}

// CanonicalCall normalizes a callsign for identity comparison: uppercase
// base plus an explicit SSID. Per the APRS 1.0.1 spec an omitted SSID
// defaults to 0, so "SP9SPM" and "SP9SPM-0" are the same station.
// Portable suffixes ("SP9SPM-7/P") are stripped from the SSID.
func CanonicalCall(callsign string) string {
	c := strings.ToUpper(strings.TrimSpace(callsign))
	base, ssid := c, ""
	if i := strings.IndexByte(c, '-'); i >= 0 {
		base, ssid = c[:i], c[i+1:]
	} else if i := strings.IndexByte(c, '/'); i >= 0 {
		base = c[:i]
	}
	if j := strings.IndexAny(ssid, "/"); j >= 0 {
		ssid = ssid[:j]
	}
	if ssid == "" {
		ssid = "0"
	}
	return base + "-" + ssid
}

// Passcode computes the standard APRS-IS passcode for a callsign using the
// well-known 0x73e2 hash. The callsign is normalized to its base form first
// because servers verify the base callsign only.
//
// The hash processes characters in pairs: each even-position character is
// folded in shifted left 8 bits, each odd-position character unshifted.
// An odd-length base callsign contributes only the shifted last character.
// Verified against live APRS-IS servers (aprsc).
func Passcode(callsign string) string {
	c := BaseCall(callsign)

	hash := 0x73e2
	for i := 0; i < len(c); i += 2 {
		hash ^= int(c[i]) << 8
		if i+1 < len(c) {
			hash ^= int(c[i+1])
		}
	}
	return strconv.Itoa(hash & 0x7fff)
}
