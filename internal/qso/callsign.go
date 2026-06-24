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

// DeriveBaseCall extracts the base callsign from a possibly prefixed/suffixed
// callsign. Examples: "DL/SP9SPM/P" → "SP9SPM", "SP9SPM" → "SP9SPM".
// Returns empty string if no valid callsign part is found.
// Used for indexed database lookups where LIKE '%/call' patterns are too slow.
func DeriveBaseCall(call string) string {
	call = NormalizeCall(call)
	if call == "" {
		return ""
	}
	parts := strings.Split(call, "/")
	// Collect parts that look like real callsigns (have both letters and digits).
	var candidates []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		hasLetter := strings.ContainsAny(p, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		hasDigit := strings.ContainsAny(p, "0123456789")
		if hasLetter && hasDigit && len(p) >= 3 {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return call // fallback: return the whole callsign
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	// Multiple candidates — prefer the one that looks most like a standard
	// callsign (letter...digit...letter pattern, typical length 4-7).
	for _, c := range candidates {
		if len(c) >= 4 && len(c) <= 8 && isStandardCallPattern(c) {
			return c
		}
	}
	return candidates[0]
}

// isStandardCallPattern checks if s looks like a typical amateur callsign:
// one or two letters, a digit, then one to four letters (e.g. K1ABC, SP9XYZ, EA3XYZ).
var standardCallRe = regexp.MustCompile(`^[A-Z]{1,2}[0-9][A-Z]{1,4}$`)

func isStandardCallPattern(s string) bool {
	return standardCallRe.MatchString(s)
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
