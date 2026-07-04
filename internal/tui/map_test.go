package tui

import (
	"testing"
)

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
