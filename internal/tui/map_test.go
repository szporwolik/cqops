package tui

import (
	"strings"
	"testing"
)

// =============================================================================
// NativeMapDimensions
// =============================================================================

func TestNativeMapDimensions(t *testing.T) {
	if NativeMapHeight <= 0 {
		t.Errorf("NativeMapHeight = %d; want > 0", NativeMapHeight)
	}
	if NativeMapWidth <= 0 {
		t.Errorf("NativeMapWidth = %d; want > 0", NativeMapWidth)
	}
	// The map is ASCII art — should have reasonable dimensions for a world map.
	if NativeMapHeight < 10 {
		t.Errorf("NativeMapHeight = %d; expected at least 10 rows", NativeMapHeight)
	}
	if NativeMapWidth < 20 {
		t.Errorf("NativeMapWidth = %d; expected at least 20 columns", NativeMapWidth)
	}
}

// =============================================================================
// mercatorXY
// =============================================================================

func TestMercatorXY(t *testing.T) {
	tests := []struct {
		lat, lon     float64
		w, h         int
		wantSentinel bool // expecting (-1, -1) sentinel
		wantInBounds bool // expecting coords to be within [0,w) × [0,h)
	}{
		// Valid coordinates — should return in-bounds pixel coords.
		{lat: 52.0, lon: 21.0, w: 80, h: 24, wantInBounds: true},   // Warsaw area
		{lat: 50.0, lon: 20.0, w: 80, h: 24, wantInBounds: true},   // Poland
		{lat: -34.0, lon: -58.0, w: 80, h: 24, wantInBounds: true}, // Buenos Aires
		// (0, 0) is the sentinel for "no location" — returns (-1, -1).
		{lat: 0, lon: 0, w: 80, h: 24, wantSentinel: true},
		// Very small map dimensions — should not panic, should return in-bounds.
		{lat: 52.0, lon: 21.0, w: 1, h: 1, wantInBounds: true},
		{lat: 52.0, lon: 21.0, w: 5, h: 3, wantInBounds: true},
	}
	for _, tt := range tests {
		x, y := mercatorXY(tt.lat, tt.lon, tt.w, tt.h)
		if tt.wantSentinel {
			if x != -1 || y != -1 {
				t.Errorf("mercatorXY(%v, %v, %d, %d) = (%d, %d); want (-1, -1)",
					tt.lat, tt.lon, tt.w, tt.h, x, y)
			}
		} else if tt.wantInBounds {
			if x < 0 || x >= tt.w {
				t.Errorf("mercatorXY(%v, %v, %d, %d) x = %d; want 0 <= x < %d",
					tt.lat, tt.lon, tt.w, tt.h, x, tt.w)
			}
			if y < 0 || y >= tt.h {
				t.Errorf("mercatorXY(%v, %v, %d, %d) y = %d; want 0 <= y < %d",
					tt.lat, tt.lon, tt.w, tt.h, y, tt.h)
			}
		}
	}
}

// =============================================================================
// renderWorldMap — no-panic and dimension checks
// =============================================================================

func TestRenderWorldMap_NoPanic(t *testing.T) {
	// These calls should never panic, regardless of input.
	call := func(lat1, lon1, lat2, lon2 float64, w, h int) (result string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("renderWorldMap(%v,%v,%v,%v, %d,%d) panicked: %v",
					lat1, lon1, lat2, lon2, w, h, r)
			}
		}()
		result = renderWorldMap(lat1, lon1, lat2, lon2, w, h)
		return
	}

	// Normal usage: two valid locations, reasonable dimensions.
	got := call(52.0, 21.0, 50.0, 20.0, 80, 24)
	if got == "" {
		t.Error("renderWorldMap returned empty for normal inputs")
	}

	// Very small width (but height sufficient) — should still work.
	call(52.0, 21.0, 50.0, 20.0, 30, 24)

	// Very small height — returns "" (height non-negotiable).
	got2 := call(52.0, 21.0, 50.0, 20.0, 80, 5)
	if got2 != "" {
		t.Errorf("renderWorldMap with height=5 returned non-empty; want empty (height non-negotiable)")
	}

	// No locations (0, 0 sentinel for both).
	call(0, 0, 0, 0, 80, 24)
}

func TestRenderWorldMap_Dimensions(t *testing.T) {
	// When map is rendered, it should respect dimensions (no clipping panic).
	// The native map is ~112 columns wide; width controls horizontal padding,
	// not clipping. Height is non-negotiable: if height < native map height,
	// returns "".
	got := renderWorldMap(52.0, 21.0, 50.0, 20.0, 80, 24)
	if got == "" {
		t.Fatal("renderWorldMap returned empty")
	}

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	// Map height should be NativeMapHeight + possibly a legend line.
	if len(lines) < NativeMapHeight {
		t.Errorf("map height %d < NativeMapHeight %d", len(lines), NativeMapHeight)
	}

	// Each line should be non-empty.
	for i, line := range lines {
		if len(line) == 0 {
			t.Errorf("line %d is empty", i)
		}
	}

	// Vertical clipping: height too small → empty string.
	if got2 := renderWorldMap(52.0, 21.0, 50.0, 20.0, 80, 5); got2 != "" {
		t.Error("renderWorldMap with height=5 should return empty (height non-negotiable)")
	}
}

// =============================================================================
// gridToLatLon (map-specific edge cases already covered in grid_test.go)
// Covered here: sentinel (0,0) return for invalid input.
// =============================================================================

func TestGridToLatLon_SentinelOnEmpty(t *testing.T) {
	lat, lon := gridToLatLon("")
	if lat != 0 || lon != 0 {
		t.Errorf("gridToLatLon(\"\") = (%v, %v); want (0, 0)", lat, lon)
	}
}

// =============================================================================
// lonLatToMapPixel — equirectangular coordinate mapping for 1280×640 map
// =============================================================================

func TestLonLatToMapPixel(t *testing.T) {
	tests := []struct {
		lon, lat     float64
		w, h         int
		wantX, wantY int
	}{
		// Centre of the map.
		{lon: 0, lat: 0, w: 1280, h: 640, wantX: 640, wantY: 320},
		// Left edge.
		{lon: -180, lat: 0, w: 1280, h: 640, wantX: 0, wantY: 320},
		// Right edge — clamped.
		{lon: 180, lat: 0, w: 1280, h: 640, wantX: 1279, wantY: 320},
		// Top edge.
		{lon: 0, lat: 90, w: 1280, h: 640, wantX: 640, wantY: 0},
		// Bottom edge — clamped.
		{lon: 0, lat: -90, w: 1280, h: 640, wantX: 640, wantY: 639},
		// Specific location: lon=20, lat=50.
		{lon: 20, lat: 50, w: 1280, h: 640, wantX: 711, wantY: 142},
		// Out of range — clamped.
		{lon: 200, lat: 0, w: 1280, h: 640, wantX: 1279, wantY: 320},
		{lon: -200, lat: 0, w: 1280, h: 640, wantX: 0, wantY: 320},
		{lon: 0, lat: 100, w: 1280, h: 640, wantX: 640, wantY: 0},
		{lon: 0, lat: -100, w: 1280, h: 640, wantX: 640, wantY: 639},
		// Small map.
		{lon: 0, lat: 0, w: 10, h: 5, wantX: 5, wantY: 3},
	}

	for _, tt := range tests {
		x, y := lonLatToMapPixel(tt.lon, tt.lat, tt.w, tt.h)
		if x != tt.wantX || y != tt.wantY {
			t.Errorf("lonLatToMapPixel(%v, %v, %d, %d) = (%d, %d); want (%d, %d)",
				tt.lon, tt.lat, tt.w, tt.h, x, y, tt.wantX, tt.wantY)
		}
	}
}
