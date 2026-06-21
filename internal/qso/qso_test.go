package qso

import "testing"

func TestParseSerial(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"   ", 0},
		{"599 001", 1},
		{"001", 1},
		{"42", 42},
		{"599", 599},
		{" 599 001 ", 1},
		{"10212", 10212},
		{"ABC", 0},
		{"599ABC", 599},
		{"ABC599", 599},
		{"599 001 002", 2},
	}
	for _, tt := range tests {
		got := ParseSerial(tt.input)
		if got != tt.want {
			t.Errorf("ParseSerial(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
