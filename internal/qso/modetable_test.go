package qso

import "testing"

func TestIsValidMode(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"SSB", true},
		{"CW", true},
		{"FT8", true},
		{"MFSK", true},
		{"PSK", true},
		{"RTTY", true},
		{"USB", false},
		{"LSB", false},
		{"FT4", false},
		{"BOGUS", false},
		{"", false},
		{"ssb", true},
		{"Cw", true},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			if got := IsValidMode(tt.mode); got != tt.expected {
				t.Errorf("IsValidMode(%q) = %v, want %v", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestIsValidSubmode(t *testing.T) {
	tests := []struct {
		mode, submode string
		expected      bool
	}{
		{"SSB", "USB", true},
		{"SSB", "LSB", true},
		{"SSB", "", true},
		{"SSB", "BOGUS", false},
		{"CW", "PCW", true},
		{"CW", "", true},
		{"CW", "USB", false},
		{"FT8", "", true},
		{"MFSK", "FT4", true},
		{"MFSK", "FT8", false}, // FT8 is a top-level mode (ADIF 3.1.7), not an MFSK submode
		{"MFSK", "FT2", true},  // FT2 is an MFSK submode (ADIF 3.1.7)
		{"DIGITALVOICE", "DMR", true},
		{"DIGITALVOICE", "DSTAR", true},
		{"DIGITALVOICE", "C4FM", true},
		{"BOGUS", "X", false},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"/"+tt.submode, func(t *testing.T) {
			if got := IsValidSubmode(tt.mode, tt.submode); got != tt.expected {
				t.Errorf("IsValidSubmode(%q, %q) = %v, want %v", tt.mode, tt.submode, got, tt.expected)
			}
		})
	}
}

func TestNormalizeMode(t *testing.T) {
	tests := []struct {
		mode, submode     string
		wantMode, wantSub string
	}{
		{"SSB", "USB", "SSB", "USB"},
		{"CW", "", "CW", ""},
		{"USB", "", "SSB", "USB"},
		{"LSB", "", "SSB", "LSB"},
		{"USB", "USB", "SSB", "USB"},
		{"LSB", "LSB", "SSB", "LSB"},
		{"C4FM", "", "DIGITALVOICE", "C4FM"},
		{"DSTAR", "", "DIGITALVOICE", "DSTAR"},
		{"JT4A", "", "JT4", "JT4A"},
		{"JT65A", "", "JT65", "JT65A"},
		{"PSK31", "", "PSK", "PSK31"},
		{"PSK63", "", "PSK", "PSK63"},
		{"PSK125", "", "PSK", "PSK125"},
		{"QPSK31", "", "PSK", "QPSK31"},
		{"MFSK8", "", "MFSK", "MFSK8"},
		{"MFSK16", "", "MFSK", "MFSK16"},
		{"FT8", "", "FT8", ""}, // FT8 is a top-level mode (ADIF 3.1.7)
		{"FT4", "", "MFSK", "FT4"},
		{"MFSK", "FT4", "MFSK", "FT4"},
		{"MFSK", "FT8", "FT8", ""}, // MFSK+FT8 normalizes to standalone FT8 (legacy / non-standard)
		{"ft8", "", "FT8", ""},
		{"ft4", "", "MFSK", "FT4"},
		{"Ft8", "", "FT8", ""},
		{"", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"/"+tt.submode, func(t *testing.T) {
			gotMode, gotSub := NormalizeMode(tt.mode, tt.submode)
			if gotMode != tt.wantMode || gotSub != tt.wantSub {
				t.Errorf("NormalizeMode(%q, %q) = (%q, %q), want (%q, %q)",
					tt.mode, tt.submode, gotMode, gotSub, tt.wantMode, tt.wantSub)
			}
		})
	}
}

func TestFlrigModeMap(t *testing.T) {
	tests := map[string]string{
		"USB":    "SSB",
		"LSB":    "SSB",
		"CW":     "CW",
		"CWR":    "CW",
		"RTTY":   "RTTY",
		"RTTYR":  "RTTY",
		"AM":     "AM",
		"FM":     "FM",
		"WFM":    "FM",
		"PKT":    "PKT",
		"PKT-L":  "PKT",
		"PKT-U":  "PKT",
		"PKT-FM": "PKT",
		"BOGUS":  "BOGUS",
	}
	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			if got := NormalizeRigMode(raw); got != want {
				t.Errorf("NormalizeRigMode(%q) = %q, want %q", raw, got, want)
			}
		})
	}
}
