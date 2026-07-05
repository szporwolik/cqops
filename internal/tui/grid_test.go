package tui

import (
	"math"
	"strings"
	"testing"
)

// =============================================================================
// gridDistanceKm
// =============================================================================

func TestGridDistanceKm(t *testing.T) {
	tests := []struct {
		own, partner string
		wantMin      float64 // distance should be >= wantMin
		wantZero     bool    // distance should be exactly 0
	}{
		// Known grid pair: KO00ca to KO02lg (~233 km per hamradio lib)
		{own: "KO00ca", partner: "KO02lg", wantMin: 230},
		// Same grid → zero distance
		{own: "KO00ca", partner: "KO00ca", wantZero: true},
		{own: "JO90", partner: "JO90", wantZero: true},
		// Empty own grid
		{own: "", partner: "KO02lg", wantZero: true},
		// Empty partner grid
		{own: "KO00ca", partner: "", wantZero: true},
		// Both empty
		{own: "", partner: "", wantZero: true},
		// Invalid grid
		{own: "INVALID", partner: "KO02lg", wantZero: true},
		{own: "KO00ca", partner: "XXXXXX", wantZero: true},
	}
	for _, tt := range tests {
		got := gridDistanceKm(tt.own, tt.partner)
		if tt.wantZero {
			if got != 0 {
				t.Errorf("gridDistanceKm(%q, %q) = %v; want 0", tt.own, tt.partner, got)
			}
		} else if got < tt.wantMin {
			t.Errorf("gridDistanceKm(%q, %q) = %v; want >= %v", tt.own, tt.partner, got, tt.wantMin)
		}
	}
}

// =============================================================================
// gridBearing / gridBearingDeg
// =============================================================================

func TestGridBearingDeg(t *testing.T) {
	tests := []struct {
		own, partner string
		wantBelow    float64 // bearing should be >= 0 and < wantBelow
		wantNeg      bool    // expecting -1 (error/empty)
	}{
		// Known bearing from KO00ca to KO02lg (roughly NNE)
		{own: "KO00ca", partner: "KO02lg", wantBelow: 360},
		// Same grid → any bearing is valid (typically 0 from library)
		{own: "KO00ca", partner: "KO00ca", wantBelow: 360},
		// Empty grids
		{own: "", partner: "KO02lg", wantNeg: true},
		{own: "KO00ca", partner: "", wantNeg: true},
		{own: "", partner: "", wantNeg: true},
		// Invalid grids
		{own: "INVALID", partner: "KO02lg", wantNeg: true},
	}
	for _, tt := range tests {
		got := gridBearingDeg(tt.own, tt.partner)
		if tt.wantNeg {
			if got >= 0 {
				t.Errorf("gridBearingDeg(%q, %q) = %v; want -1", tt.own, tt.partner, got)
			}
		} else if got < 0 || got >= tt.wantBelow {
			t.Errorf("gridBearingDeg(%q, %q) = %v; want 0 <= bearing < %v", tt.own, tt.partner, got, tt.wantBelow)
		}
	}
}

func TestGridBearing(t *testing.T) {
	// gridBearing returns formatted bearing string.
	tests := []struct {
		own, partner string
		wantEmpty    bool
	}{
		{own: "KO00ca", partner: "KO02lg", wantEmpty: false},
		{own: "", partner: "KO02lg", wantEmpty: true},
		{own: "KO00ca", partner: "", wantEmpty: true},
	}
	for _, tt := range tests {
		got := gridBearing(tt.own, tt.partner)
		if tt.wantEmpty && got != "" {
			t.Errorf("gridBearing(%q, %q) = %q; want empty", tt.own, tt.partner, got)
		}
		if !tt.wantEmpty && got == "" {
			t.Errorf("gridBearing(%q, %q) = empty; want non-empty bearing", tt.own, tt.partner)
		}
	}
}

// =============================================================================
// formatDistance
// =============================================================================

func TestFormatDistance(t *testing.T) {
	tests := []struct {
		km   float64
		unit string
		want string
	}{
		{km: 100, unit: "metric", want: "100 km"},
		{km: 100, unit: "imperial", want: "62 mi"},
		{km: 0, unit: "metric", want: ""},
		{km: -1, unit: "metric", want: ""},
		{km: 1, unit: "metric", want: "1 km"},
		{km: 1.4, unit: "metric", want: "1 km"},
		{km: 1.6, unit: "metric", want: "2 km"},
		// Unknown unit defaults to km
		{km: 50, unit: "nmi", want: "50 km"},
		// imperial rounding
		{km: 10, unit: "imperial", want: "6 mi"},
	}
	for _, tt := range tests {
		got := formatDistance(tt.km, tt.unit)
		if got != tt.want {
			t.Errorf("formatDistance(%v, %q) = %q; want %q", tt.km, tt.unit, got, tt.want)
		}
	}
}

// =============================================================================
// distanceLine (integration)
// =============================================================================

func TestDistanceLine(t *testing.T) {
	// Valid grids produce a distance+bearing line.
	got := distanceLine("KO00ca", "KO02lg", "km")
	if got == "" {
		t.Error("distanceLine returned empty for valid grids")
	}
	if !strings.Contains(got, "·") {
		t.Errorf("distanceLine = %q; expected '·' separator", got)
	}

	// Empty grids still produce a fallback line.
	got2 := distanceLine("", "", "km")
	if got2 == "" {
		t.Error("distanceLine returned empty for empty grids (should get fallback)")
	}
}

// =============================================================================
// formatLocator
// =============================================================================

func TestFormatLocator(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		// Standard 4-char: uppercase first 2, lowercase nothing else
		{"ko00", "KO00"},
		{"KO00", "KO00"},
		{"Ko00", "KO00"},
		// 6-char Maidenhead: uppercase first 2, keep next 2 as-is, lowercase remainder
		{"ko00ca", "KO00ca"},
		{"KO00CA", "KO00ca"},
		{"Ko00Ca", "KO00ca"},
		// < 4 chars: just uppercase
		{"ab", "AB"},
		{"a", "A"},
		// Empty
		{"", ""},
		// With spaces
		{"  ko00  ", "KO00"},
		// 8-char extended
		{"KO00ca12", "KO00ca12"},
		{"KO00CA12", "KO00ca12"},
	}
	for _, tt := range tests {
		got := formatLocator(tt.input)
		if got != tt.want {
			t.Errorf("formatLocator(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

// =============================================================================
// gridToLatLon
// =============================================================================

func TestGridToLatLon(t *testing.T) {
	tests := []struct {
		grid       string
		wantZero   bool
		wantLatMin float64 // latitude should be >= wantLatMin
	}{
		// Valid grids should produce non-zero coordinates.
		{grid: "KO00ca", wantLatMin: 50}, // Poland area, lat ~52
		{grid: "KO02lg", wantLatMin: 50}, // ~52
		{grid: "JO90", wantLatMin: 49},   // ~50
		// Invalid grid → (0,0).
		{grid: "INVALID", wantZero: true},
		{grid: "", wantZero: true},
		// Southern hemisphere example: grid near Buenos Aires
		{grid: "GF05", wantLatMin: -40}, // ~-35
	}
	for _, tt := range tests {
		lat, lon := gridToLatLon(tt.grid)
		if tt.wantZero {
			if lat != 0 || lon != 0 {
				t.Errorf("gridToLatLon(%q) = (%v, %v); want (0, 0)", tt.grid, lat, lon)
			}
		} else {
			if lat == 0 && lon == 0 {
				t.Errorf("gridToLatLon(%q) = (0, 0); want non-zero coords", tt.grid)
			}
			if lat < tt.wantLatMin {
				t.Errorf("gridToLatLon(%q) lat = %v; want >= %v", tt.grid, lat, tt.wantLatMin)
			}
		}
	}
}

// =============================================================================
// parseCoord
// =============================================================================

func TestParseCoord(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"52.3", 52.3},
		{"-1.5", -1.5},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
		{"  42.0  ", 42.0},
	}
	for _, tt := range tests {
		got := parseCoord(tt.input)
		if math.Abs(got-tt.want) > 0.0001 {
			t.Errorf("parseCoord(%q) = %v; want %v", tt.input, got, tt.want)
		}
	}
}
