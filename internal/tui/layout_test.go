package tui

import (
	"testing"
)

// =============================================================================
// stripNonDigits
// =============================================================================

func TestStripNonDigits(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", ""},
		{"abc", ""},
		{"123", "123"},
		{"1a2b3c", "123"},
		{"123.456", "123456"},
		{"  42  ", "42"},
		{"abc123def456", "123456"},
	}
	for _, tt := range tests {
		got := stripNonDigits(tt.input)
		if got != tt.want {
			t.Errorf("stripNonDigits(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

// =============================================================================
// formatDate / formatTime
// =============================================================================

func TestFormatDate(t *testing.T) {
	tests := []struct {
		adif, want string
	}{
		{"20240501", "2024-05-01"},
		{"20240101", "2024-01-01"},
		{"20241231", "2024-12-31"},
		{"2024050", "—"}, // too short
		{"", "—"},
		{"abcd", "—"},
	}
	for _, tt := range tests {
		got := formatDate(tt.adif)
		if got != tt.want {
			t.Errorf("formatDate(%q) = %q; want %q", tt.adif, got, tt.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		adif, want string
	}{
		{"123045", "12:30:45"},
		{"000000", "00:00:00"},
		{"235959", "23:59:59"},
		{"12304", "—"},
		{"", "—"},
	}
	for _, tt := range tests {
		got := formatTime(tt.adif)
		if got != tt.want {
			t.Errorf("formatTime(%q) = %q; want %q", tt.adif, got, tt.want)
		}
	}
}

// =============================================================================
// tern
// =============================================================================

func TestTern(t *testing.T) {
	if got := tern(true, "yes", "no"); got != "yes" {
		t.Errorf("tern(true) = %q; want yes", got)
	}
	if got := tern(false, "yes", "no"); got != "no" {
		t.Errorf("tern(false) = %q; want no", got)
	}
}
