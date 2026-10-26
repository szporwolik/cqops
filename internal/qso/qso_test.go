package qso

import "testing"

func TestSerialFromExchange(t *testing.T) {
	tests := []struct {
		exchange string
		rst      string
		want     int
	}{
		// Plain serials, with and without the report in front.
		{"", "", 0},
		{"   ", "", 0},
		{"001", "", 1},
		{"42", "", 42},
		{"10212", "", 10212},
		{"599", "", 599},
		{"599 001", "", 1},
		{" 599 001 ", "", 1},
		{"599 001 002", "", 2},
		{"59 15", "59", 15},
		{"599 001", "599", 1},
		{"5NN 042", "599", 42},

		// The report must never become the serial: reading the number out of
		// "55 MA" used to return 55 and send it to Wavelog.
		{"55 MA", "55", 0},
		{"57 MA", "57", 0},
		{"55", "55", 0},

		// Digits embedded in a state, grid or code are not serials.
		{"55 JO90", "55", 0},
		{"JO90", "", 0},
		{"JO90AA", "", 0},
		{"ABC599", "", 0},
		{"599ABC", "", 0},
		{"ABC", "", 0},

		// A serial followed by another token still counts.
		{"59 15 KW", "59", 15},
		{"59 001/JO90", "59", 1},

		// More digits than a serial can hold.
		{"59 12345678901234567890", "59", 0},
	}
	for _, tt := range tests {
		got := SerialFromExchange(tt.exchange, tt.rst)
		if got != tt.want {
			t.Errorf("SerialFromExchange(%q, %q) = %d, want %d", tt.exchange, tt.rst, got, tt.want)
		}
	}
}

// TestNormalizeExchangeKeepsStateExchangeNumberless: a non-serial exchange
// ("MA", "JO90") must not produce STX/SRX at all — the ADIF export omits them
// and Wavelog gets no invented serial.
func TestNormalizeExchangeKeepsStateExchangeNumberless(t *testing.T) {
	state := QSO{RSTSent: "55", RSTRcvd: "57", ExchSent: "55 MA", ExchRcvd: "57 JO90"}
	state.NormalizeExchange()
	if state.STX != 0 || state.SRX != 0 {
		t.Errorf("serials = %d/%d, want 0/0", state.STX, state.SRX)
	}
	if state.STXString != "MA" || state.SRXString != "JO90" {
		t.Errorf("exchange strings = %q/%q, want MA/JO90", state.STXString, state.SRXString)
	}

	serial := QSO{RSTSent: "59", RSTRcvd: "59", ExchSent: "59 15", ExchRcvd: "59 26"}
	serial.NormalizeExchange()
	if serial.STX != 15 || serial.SRX != 26 {
		t.Errorf("serials = %d/%d, want 15/26", serial.STX, serial.SRX)
	}
}
