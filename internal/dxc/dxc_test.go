package dxc

import (
	"testing"
)

func TestParseSpot(t *testing.T) {
	tests := []struct {
		line        string
		wantOK      bool
		wantDX      string
		wantFreq    float64
		wantSpotter string
	}{
		{"DX de SP9SPM:  14074.0  K1ABC  FT8 TNX", true, "K1ABC", 14074.0, "SP9SPM"},
		{"DX de N0NBH:     3800.0   W1AW     Hello World", true, "W1AW", 3800.0, "N0NBH"},
		{"DX de EA3XXX:  28001.5  JA1ABC", true, "JA1ABC", 28001.5, "EA3XXX"},
		{"DX de IZ1:  7000.0  DL1ABC  59 QSL", true, "DL1ABC", 7000.0, "IZ1"},
		// Non-spot lines
		{"Hello from cluster", false, "", 0, ""},
		{"Welcome to DXSpider", false, "", 0, ""},
		{"", false, "", 0, ""},
		// Edge cases
		{"DX de : 14000.0 CALL", false, "", 0, ""},
		{"DX de SP9:  x  CALL", false, "", 0, ""},
		{"DX de SP9:  14000.0", false, "", 0, ""},
	}

	for _, tt := range tests {
		s, ok := parseSpot(tt.line)
		if ok != tt.wantOK {
			t.Errorf("parseSpot(%q) ok=%v, want %v", tt.line, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if s.DXCall != tt.wantDX {
			t.Errorf("parseSpot(%q) DXCall=%q, want %q", tt.line, s.DXCall, tt.wantDX)
		}
		if s.Frequency != tt.wantFreq {
			t.Errorf("parseSpot(%q) Freq=%f, want %f", tt.line, s.Frequency, tt.wantFreq)
		}
		if s.Spotter != tt.wantSpotter {
			t.Errorf("parseSpot(%q) Spotter=%q, want %q", tt.line, s.Spotter, tt.wantSpotter)
		}
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("", "", "SP9SPM")
	if c.host != "dxspots.com" {
		t.Errorf("default host = %q, want dxspots.com", c.host)
	}
	if c.port != "7300" {
		t.Errorf("default port = %q, want 7300", c.port)
	}
	if c.login != "SP9SPM" {
		t.Errorf("login = %q, want SP9SPM", c.login)
	}
}
