package qso

import (
	"testing"
)

func TestIsValidCall(t *testing.T) {
	tests := []struct {
		call string
		ok   bool
	}{
		// Valid standard calls.
		{"SP9MOA", true},
		{"VK3A", true},
		{"DJ7NT", true},
		{"K1ABC", true},
		{"W1AW", true},
		{"JA1ABC", true},
		{"EA8XYZ", true},
		{"9A1A", true},
		{"3B8CF", true},
		{"ZL1ABC", true},

		// Valid with slash (portable/mobile).
		{"SP9ABC/P", true},
		{"DL/SP9ABC", true},
		{"SP9ABC/M", true},
		{"SP9ABC/R", true},
		{"F/SP9ABC/P", true},
		{"EA8/SP9MOA", true},

		// Edge cases.
		{"A1A", true},         // 3 chars, min length
		{"A1234567890", true}, // longish but under 20

		// Too short.
		{"AB", false},
		{"A1", false},
		{"", false},

		// Too long.
		{"A12345678901234567890", false}, // 21 chars

		// Missing digit (ham callsigns always have digits).
		{"SPAMOA", false},
		{"ABC", false},

		// Missing letter.
		{"12345", false},
		{"123", false},

		// Invalid characters.
		{"SP9 MOA", false},  // space
		{"SP9\tMOA", false}, // tab
		{"SP9;MOA", false},  // semicolon
		{"SP9,MOA", false},  // comma
		{"SP9.MOA", false},  // dot
		{"SP9\\MOA", false}, // backslash
		{"SP9\"MOA", false}, // quote
		{"SP9'MOA", false},  // single quote
		{"SP9[MOA", false},  // bracket
		{"SP9{MOA", false},  // brace
		{"SP9<MOA", false},  // angle bracket
		{"SP9😀MOA", false},  // emoji

		// Slash abuse.
		{"/SP9MOA", false},  // slash at start
		{"SP9MOA/", false},  // slash at end
		{"SP9//MOA", false}, // double slash
		{"//SP9MOA", false}, // double slash at start
		{"/", false},        // just slash
		{"//", false},       // just slashes

		// Lowercase (uppercased by NormalizeCall, test here too).
		{"sp9moa", true},
		{"sp9abc/p", true},
	}

	for _, tt := range tests {
		t.Run(tt.call, func(t *testing.T) {
			got := IsValidCall(tt.call)
			if got != tt.ok {
				t.Errorf("IsValidCall(%q) = %v, want %v", tt.call, got, tt.ok)
			}
		})
	}
}

func TestNormalizeCall(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"  sp9moa  ", "SP9MOA"},
		{"SP9MOA", "SP9MOA"},
		{"sp9abc/p", "SP9ABC/P"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		got := NormalizeCall(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeCall(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
