// Package geo provides coordinate, Maidenhead locator, distance, and bearing
// utilities shared across CQOps packages. Package geo has no dependencies on
// other CQOps packages and is safe to import from app, tui, dashboard, or qso.
package geo

import (
	"fmt"
	"math"
	"strings"
)

// =============================================================================
// Maidenhead grid locator ↔ decimal lat/lon
// =============================================================================

// GridToLatLon converts a Maidenhead grid locator to decimal latitude and
// longitude. Supports 2, 4, 6, 8, and 10 character grids. Returns the centre
// of the grid cell at the given precision. Returns an error for grids under
// 4 characters (2-char locators are too coarse for position display).
func GridToLatLon(grid string) (float64, float64, error) {
	grid = strings.ToUpper(strings.TrimSpace(grid))
	if len(grid) < 4 {
		return 0, 0, fmt.Errorf("grid too short: %q", grid)
	}
	// Validate Maidenhead field ranges.
	if grid[0] < 'A' || grid[0] > 'R' || grid[1] < 'A' || grid[1] > 'R' {
		return 0, 0, fmt.Errorf("invalid grid field: %q", grid[:2])
	}
	if grid[2] < '0' || grid[2] > '9' || grid[3] < '0' || grid[3] > '9' {
		return 0, 0, fmt.Errorf("invalid grid square: %q", grid[2:4])
	}
	if len(grid) >= 6 {
		if grid[4] < 'A' || grid[4] > 'X' || grid[5] < 'A' || grid[5] > 'X' {
			return 0, 0, fmt.Errorf("invalid grid subsquare: %q", grid[4:6])
		}
	}
	if len(grid) >= 8 {
		if grid[6] < '0' || grid[6] > '9' || grid[7] < '0' || grid[7] > '9' {
			return 0, 0, fmt.Errorf("invalid grid extended: %q", grid[6:8])
		}
	}
	if len(grid) >= 10 {
		if grid[8] < 'A' || grid[8] > 'X' || grid[9] < 'A' || grid[9] > 'X' {
			return 0, 0, fmt.Errorf("invalid grid further extended: %q", grid[8:10])
		}
	}
	lon := float64(grid[0]-'A')*20 - 180
	lat := float64(grid[1]-'A')*10 - 90
	lon += float64(grid[2]-'0') * 2
	lat += float64(grid[3]-'0') * 1
	if len(grid) >= 6 {
		lon += float64(grid[4]-'A') * (5.0 / 60.0)
		lat += float64(grid[5]-'A') * (2.5 / 60.0)
		if len(grid) >= 8 {
			lon += float64(grid[6]-'0') * (0.5 / 60.0)
			lat += float64(grid[7]-'0') * (0.25 / 60.0)
			if len(grid) >= 10 {
				lon += float64(grid[8]-'A') * (0.5 / 60.0 / 24.0)
				lat += float64(grid[9]-'A') * (0.25 / 60.0 / 24.0)
				lon += 0.5 / 60.0 / 48.0  // centre of 10-char cell
				lat += 0.25 / 60.0 / 48.0 // centre of 10-char cell
			} else {
				lon += 0.25 / 60.0  // centre of 8-char cell
				lat += 0.125 / 60.0 // centre of 8-char cell
			}
		} else {
			lon += 2.5 / 60.0  // centre of 6-char cell
			lat += 1.25 / 60.0 // centre of 6-char cell
		}
	} else {
		lon += 1.0 // centre of 2° square (4-char grid)
		lat += 0.5 // centre of 1° square
	}
	return lat, lon, nil
}

// LatLonToGrid converts a decimal latitude/longitude to a 4-character
// Maidenhead grid locator (field + square). Suitable for approximate
// country-level positioning.
func LatLonToGrid(lat, lon float64) string {
	lon += 180
	lat += 90
	fLon := int(lon / 20)
	fLat := int(lat / 10)
	lon = math.Mod(lon, 20)
	lat = math.Mod(lat, 10)
	sLon := int(lon / 2)
	sLat := int(lat)
	return fmt.Sprintf("%c%c%d%d", 'A'+fLon, 'A'+fLat, sLon, sLat)
}

// =============================================================================
// Distance and bearing
// =============================================================================

const earthRadiusKm = 6371.0

// HaversineKm computes the great-circle distance between two points in
// kilometres using the haversine formula.
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// BearingDeg computes the initial bearing (forward azimuth) from point 1 to
// point 2, in degrees (0–360). Result is normalised to [0, 360).
func BearingDeg(lat1, lon1, lat2, lon2 float64) float64 {
	dLon := (lon2 - lon1) * math.Pi / 180
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	y := math.Sin(dLon) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLon)
	return math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
}

// =============================================================================
// Validation
// =============================================================================

// IsSentinelGrid returns true if grid is a known placeholder/sentinel value
// (e.g. AA00aa returned by QRZ when the real grid is unknown).
func IsSentinelGrid(grid string) bool {
	g := strings.ToUpper(strings.TrimSpace(grid))
	return g == "AA00AA"
}
