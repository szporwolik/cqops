package tui

import (
	"fmt"
	"strings"

	"github.com/szporwolik/cqops/internal/geo"
)

func gridToLatLon(grid string) (float64, float64) {
	lat, lon, _ := gridToLatLonOK(grid)
	return lat, lon
}

// gridToLatLonOK reports whether the locator was valid, so callers can tell a
// real 0,0 (equator/prime meridian) from a parse failure.
func gridToLatLonOK(grid string) (float64, float64, bool) {
	lat, lon, err := geo.GridToLatLon(grid)
	if err != nil {
		return 0, 0, false
	}
	return lat, lon, true
}

func parseCoord(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}
