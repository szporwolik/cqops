package qso

import (
	"testing"
)

func TestIsValidLocator(t *testing.T) {
	tests := []struct {
		loc string
		ok  bool
	}{
		// Valid 2-char (WSJT-X compatibility).
		{"KO", true},
		{"JO", true},
		{"AA", true},
		{"RR", true},

		// Valid 4-char.
		{"KO00", true},
		{"JO90", true},
		{"FN31", true},
		{"AA00", true},
		{"RR99", true},

		// Valid 6-char.
		{"KO00CA", true},
		{"JO90XB", true},
		{"FN31PR", true},
		{"AA00AA", true},
		{"RR99XX", true},

		// Valid 8-char.
		{"KO00CA12", true},
		{"JO90XB99", true},
		{"FN31PR00", true},

		// Invalid 2-char.
		{"ZZ", false}, // out of range
		{"K0", false}, // digit in field
		{"99", false}, // digits only
		{"A", false},  // 1 char
		{"", false},

		// Too short / invalid length.
		{"K", false},
		{"KO0", false},
		{"KO000", false},     // 5 chars
		{"KO00C", false},     // 5 chars
		{"KO00CAA", false},   // 7 chars
		{"KO00CA123", false}, // 9 chars
		{"KO00CAAA", false},  // 8 chars but wrong format (4th pair must be digits)

		// Invalid ranges.
		{"ZZ99", false},   // first pair: Z out of range (A-R)
		{"KOAA", false},   // second pair: letters instead of digits
		{"KO00ZA", false}, // third pair: Z out of range (A-X)

		// Invalid characters.
		{"KO 00", false},
		{"KO/00", false},
		{"KO.00", false},
		{"KO,00", false},
		{"KO;00", false},
		{"KO\"00", false},

		// Lowercase.
		{"ko00", true},
		{"jo90xb", true},
		{"ko00ca12", true},

		// With spaces.
		{"  KO00  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.loc, func(t *testing.T) {
			got := IsValidLocator(tt.loc)
			if got != tt.ok {
				t.Errorf("IsValidLocator(%q) = %v, want %v", tt.loc, got, tt.ok)
			}
		})
	}
}

func TestNormalizeLocator(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"  ko00  ", "KO00"},
		{"JO90XB", "JO90XB"},
		{"ko00ca12", "KO00CA12"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		got := NormalizeLocator(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeLocator(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
