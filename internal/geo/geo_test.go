package geo

import (
	"math"
	"testing"
)

func TestGridToLatLon_4Char(t *testing.T) {
	lat, lon, err := GridToLatLon("KO00")
	if err != nil {
		t.Fatal(err)
	}
	// Centre of 2°×1° square: 20°E + 1° = 21°E, 50°N + 0.5° = 50.5°N
	if lat < 50 || lat > 51 {
		t.Errorf("lat = %v, want ~50.5", lat)
	}
	if lon < 20 || lon > 22 {
		t.Errorf("lon = %v, want ~21", lon)
	}
}

func TestGridToLatLon_6Char(t *testing.T) {
	lat, lon, err := GridToLatLon("KO00CA")
	if err != nil {
		t.Fatal(err)
	}
	// KO00CA: 20° + 10′ + 2.5′ = ~20.2083°E, 50° + 0′ + 1.25′ = ~50.0208°N
	if lat < 49 || lat > 51 {
		t.Errorf("lat = %v, want ~50.02", lat)
	}
	if lon < 19 || lon > 21 {
		t.Errorf("lon = %v, want ~20.21", lon)
	}
}

func TestGridToLatLon_8Char(t *testing.T) {
	lat, lon, err := GridToLatLon("KO00CA02")
	if err != nil {
		t.Fatal(err)
	}
	// KO00CA02: subsquare 'C'→2×5′, 'A'→0×2.5′, ext '0'→0×0.5′, '2'→2×0.25′
	// lon = 20° + 0° + 10′ + 2.5′ + 0 + 0.25′ = 20.2125°
	// lat = 50° + 0° + 0′ + 1.25′ + 0.5′ + 0.125′ = 50.03125°
	if lat < 49 || lat > 51 {
		t.Errorf("lat = %v, want ~50.03", lat)
	}
	if lon < 19 || lon > 21 {
		t.Errorf("lon = %v, want ~20.21", lon)
	}
}

func TestGridToLatLon_10Char(t *testing.T) {
	lat, lon, err := GridToLatLon("KO00CA02WH")
	if err != nil {
		t.Fatal(err)
	}
	if lat == 0 && lon == 0 {
		t.Error("expected non-zero coords for 10-char grid")
	}
	if lat < 49 || lat > 51 {
		t.Errorf("lat = %v, expected near 50°N", lat)
	}
}

func TestGridToLatLon_TooShort(t *testing.T) {
	_, _, err := GridToLatLon("KO")
	if err == nil {
		t.Error("expected error for 2-char grid")
	}
	_, _, err = GridToLatLon("")
	if err == nil {
		t.Error("expected error for empty grid")
	}
}

func TestGridToLatLon_Lowercase(t *testing.T) {
	lat1, lon1, _ := GridToLatLon("KO00ca")
	lat2, lon2, _ := GridToLatLon("KO00CA")
	if math.Abs(lat1-lat2) > 0.0001 || math.Abs(lon1-lon2) > 0.0001 {
		t.Errorf("mixed case KO00ca != uppercase KO00CA: (%v,%v) vs (%v,%v)",
			lat1, lon1, lat2, lon2)
	}
}

func TestLatLonToGrid_Valid(t *testing.T) {
	g := LatLonToGrid(50.02, 20.21)
	if g != "KO00" {
		t.Errorf("LatLonToGrid(50.02, 20.21) = %q, want KO00", g)
	}
}

func TestLatLonToGrid_Southern(t *testing.T) {
	g := LatLonToGrid(-34.6, -58.4) // Buenos Aires
	if g != "GF05" {
		t.Errorf("LatLonToGrid(-34.6, -58.4) = %q, want GF05", g)
	}
}

func TestHaversineKm_KnownDistance(t *testing.T) {
	// Warsaw (52.23, 21.01) to Kraków (50.06, 19.94) ≈ 250 km
	d := HaversineKm(52.23, 21.01, 50.06, 19.94)
	if d < 200 || d > 300 {
		t.Errorf("Warsaw-Kraków distance = %v km, expected ~250 km", d)
	}
}

func TestHaversineKm_Zero(t *testing.T) {
	d := HaversineKm(52.0, 21.0, 52.0, 21.0)
	if d != 0 {
		t.Errorf("same-point distance = %v, want 0", d)
	}
}

func TestBearingDeg_North(t *testing.T) {
	b := BearingDeg(50, 20, 52, 20) // due north
	if b < -1 || b > 1 {
		t.Errorf("bearing north = %v°, want ~0°", b)
	}
}

func TestBearingDeg_East(t *testing.T) {
	b := BearingDeg(50, 20, 50, 22) // due east
	if b < 85 || b > 95 {
		t.Errorf("bearing east = %v°, want ~90°", b)
	}
}

func TestIsSentinelGrid(t *testing.T) {
	if !IsSentinelGrid("AA00aa") {
		t.Error("AA00aa should be sentinel")
	}
	if !IsSentinelGrid("aa00aa") {
		t.Error("aa00aa should be sentinel (case-insensitive)")
	}
	if IsSentinelGrid("KO00CA") {
		t.Error("KO00CA should not be sentinel")
	}
	if IsSentinelGrid("") {
		t.Error("empty should not be sentinel")
	}
}
