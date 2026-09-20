package aprs

import "strings"

// CleanComment returns an APRS comment safe for terminal display. Emoji,
// pictographs, and decorative Unicode symbols (☀, ❄, ★, ➡, …) render as
// tofu boxes on many terminal fonts, so they are removed; variation
// selectors are dropped and collapsed whitespace is normalized. Letters
// and punctuation — including accented Latin — are preserved.
func CleanComment(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	pendingSpace := false
	for _, r := range s {
		if isCommentSymbol(r) {
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if b.Len() > 0 {
				pendingSpace = true
			}
			continue
		}
		if pendingSpace {
			b.WriteByte(' ')
			pendingSpace = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isCommentSymbol reports whether r is an emoji, pictograph, or decorative
// symbol that typically lacks a glyph on terminal fonts.
func isCommentSymbol(r rune) bool {
	switch {
	case r >= 0xFE00 && r <= 0xFE0F: // variation selectors
		return true
	case r >= 0x1F000 && r <= 0x1FAFF: // emoji, pictographs, symbols
		return true
	case r >= 0x2600 && r <= 0x27BF: // miscellaneous symbols + dingbats
		return true
	case r >= 0x2B00 && r <= 0x2BFF: // arrows and geometric shapes
		return true
	}
	return false
}
