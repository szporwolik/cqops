package qso

import (
	"regexp"
	"strings"
)

// locatorRe validates Maidenhead grid locators: 2/4/6/8 chars, standard field ranges.
// 2-char form (e.g. "KO") accepted for WSJT-X compatibility.
var locatorRe = regexp.MustCompile(`^[A-R]{2}([0-9]{2}([A-X]{2}([0-9]{2})?)?)?$`)

// NormalizeLocator trims spaces and converts to uppercase.
func NormalizeLocator(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// IsValidLocator returns true if s is a valid Maidenhead grid locator
// (2, 4, 6, or 8 characters with correct field ranges).
func IsValidLocator(s string) bool {
	s = NormalizeLocator(s)
	if s == "" {
		return false
	}
	return locatorRe.MatchString(s)
}
