package ref

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// normalizeForSearch strips diacritics and lowercases for accent/case-insensitive
// substring search. "Ćwilin" → "cwilin", "Österreich" → "osterreich".
func normalizeForSearch(s string) string {
	t := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)),
		norm.NFC,
	)
	result, _, err := transform.String(t, s)
	if err != nil {
		return strings.ToLower(s)
	}
	return strings.ToLower(result)
}
