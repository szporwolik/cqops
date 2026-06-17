package qso

import (
	"regexp"
	"strings"
)

// allowedCallRe matches strings consisting only of A-Z, 0-9, and /.
var allowedCallRe = regexp.MustCompile(`^[A-Z0-9/]+$`)

// NormalizeCall trims spaces and converts to uppercase.
func NormalizeCall(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// IsValidCall returns true if the callsign passes basic format validation.
// It does not attempt to validate national prefix rules — only obvious
// garbage, typos, and injection-like input are rejected.
func IsValidCall(s string) bool {
	s = NormalizeCall(s)
	if s == "" {
		return false
	}
	if len(s) < 3 || len(s) > 20 {
		return false
	}
	if !allowedCallRe.MatchString(s) {
		return false
	}
	if strings.Contains(s, "//") {
		return false
	}
	if s[0] == '/' || s[len(s)-1] == '/' {
		return false
	}
	hasLetter := strings.ContainsAny(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := strings.ContainsAny(s, "0123456789")
	return hasLetter && hasDigit
}
