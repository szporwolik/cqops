package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestContentHeight(t *testing.T) {
	tests := []struct {
		terminalH int
		wantMin   int
	}{
		{24, 20},  // 24 - 4 = 20
		{10, 6},   // 10 - 4 = 6
		{5, 3},    // clamped to min 3
		{0, 3},    // clamped to min 3
		{-1, 3},   // clamped to min 3
		{100, 96}, // large terminal
	}
	for _, tt := range tests {
		got := contentHeight(tt.terminalH)
		if got < tt.wantMin {
			t.Errorf("contentHeight(%d) = %d; want >= %d", tt.terminalH, got, tt.wantMin)
		}
	}
}

func TestSafeWidth(t *testing.T) {
	tests := []struct {
		w    int
		want int
	}{
		{80, 80},
		{30, 30},
		{29, 30}, // clamped to min
		{0, 30},  // clamped to min
		{-5, 30}, // clamped to min
	}
	for _, tt := range tests {
		got := safeWidth(tt.w)
		if got != tt.want {
			t.Errorf("safeWidth(%d) = %d; want %d", tt.w, got, tt.want)
		}
	}
}

func TestSafeHeight(t *testing.T) {
	tests := []struct {
		h, min, want int
	}{
		{10, 5, 10},
		{5, 5, 5},
		{3, 5, 5},  // clamped to min
		{0, 5, 5},  // clamped to min
		{-1, 5, 5}, // clamped to min
	}
	for _, tt := range tests {
		got := safeHeight(tt.h, tt.min)
		if got != tt.want {
			t.Errorf("safeHeight(%d, %d) = %d; want %d", tt.h, tt.min, got, tt.want)
		}
	}
}

func TestTruncWithEllipsis(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"hello", 10, "hello"},              // shorter than max
		{"hello world", 8, "hello w\u2026"}, // truncated with ellipsis
		{"abc", 2, "abc"},                   // max < 3, returns original
		{"abcd", 4, "abcd"},                 // fits exactly, not truncated
		{"abcde", 4, "abc\u2026"},           // one char too many, truncated
		{"", 10, ""},                        // empty string
	}
	for _, tt := range tests {
		got := truncWithEllipsis(tt.s, tt.max)
		if got != tt.want {
			t.Errorf("truncWithEllipsis(%q, %d) = %q; want %q", tt.s, tt.max, got, tt.want)
		}
		// Verify display width never exceeds max (using lipgloss.Width, not len)
		if tt.max >= 3 && len(tt.s) > tt.max {
			w := lipglossWidth(got)
			if w > tt.max {
				t.Errorf("truncWithEllipsis(%q, %d) produced %q (display width %d > max %d)", tt.s, tt.max, got, w, tt.max)
			}
		}
	}
}

func TestEmptyState(t *testing.T) {
	got := emptyState()
	if got == "" {
		t.Error("emptyState() returned empty string")
	}
	if !strings.Contains(got, "\u2014") {
		t.Errorf("emptyState() = %q; expected em-dash", got)
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

// lipglossWidth returns the display width using lipgloss.Width.
func lipglossWidth(s string) int {
	return lipgloss.Width(s)
}
