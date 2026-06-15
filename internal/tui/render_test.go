package tui

import (
	"strings"
	"testing"
)

func TestContentHeight(t *testing.T) {
	tests := []struct {
		terminalH int
		wantMin   int
	}{
		{24, 19},  // 24 - 5 = 19
		{10, 5},   // 10 - 5 = 5
		{5, 3},    // clamped to min 3
		{0, 3},    // clamped to min 3
		{-1, 3},   // clamped to min 3
		{100, 95}, // large terminal
	}
	for _, tt := range tests {
		got := contentHeight(tt.terminalH)
		if got < tt.wantMin {
			t.Errorf("contentHeight(%d) = %d; want >= %d", tt.terminalH, got, tt.wantMin)
		}
	}
}

func TestFillBody(t *testing.T) {
	tests := []struct {
		content  string
		height   int
		wantRows int
	}{
		{"hello", 3, 3},
		{"line1\nline2", 5, 5},
		{"line1\nline2\nline3", 2, 3}, // already taller than requested
	}
	for _, tt := range tests {
		got := fillBody(tt.content, tt.height)
		h := lipglossHeight(got)
		if h != tt.wantRows {
			t.Errorf("fillBody(%q, %d) height = %d; want %d", tt.content, tt.height, h, tt.wantRows)
		}
	}
}

// lipglossHeight is a test helper that counts newlines to determine rendered height.
func lipglossHeight(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}
