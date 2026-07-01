package tui

import (
	"fmt"
	"strings"

	"github.com/szporwolik/cqops/internal/geo"
)

func gridToLatLon(grid string) (float64, float64) {
	lat, lon, err := geo.GridToLatLon(grid)
	if err != nil {
		return 0, 0
	}
	return lat, lon
}

func parseCoord(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}
