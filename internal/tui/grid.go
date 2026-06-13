package tui

import (
	"fmt"
	"strings"

	"github.com/ftl/hamradio/locator"
)

func gridDistanceKm(ownGrid, partnerGrid string) float64 {
	if ownGrid == "" || partnerGrid == "" {
		return 0
	}
	own, err := locator.Parse(ownGrid)
	if err != nil {
		return 0
	}
	partner, err := locator.Parse(partnerGrid)
	if err != nil {
		return 0
	}
	return float64(locator.Distance(own, partner))
}

func gridBearing(ownGrid, partnerGrid string) string {
	if ownGrid == "" || partnerGrid == "" {
		return ""
	}
	deg := gridBearingDeg(ownGrid, partnerGrid)
	if deg < 0 {
		return ""
	}
	return fmt.Sprintf("%.0f°", deg)
}

func gridBearingDeg(ownGrid, partnerGrid string) float64 {
	if ownGrid == "" || partnerGrid == "" {
		return -1
	}
	own, err := locator.Parse(ownGrid)
	if err != nil {
		return -1
	}
	partner, err := locator.Parse(partnerGrid)
	if err != nil {
		return -1
	}
	return float64(locator.Azimuth(own, partner))
}

func formatDistance(km float64, unit string) string {
	if km <= 0 {
		return ""
	}
	if unit == "mi" {
		return fmt.Sprintf("%.0f mi", km*0.621371)
	}
	return fmt.Sprintf("%.0f km", km)
}

func distanceLine(ownGrid, partnerGrid, unit string) string {
	km := gridDistanceKm(ownGrid, partnerGrid)
	bear := gridBearing(ownGrid, partnerGrid)
	if bear == "" {
		bear = "0°"
	}
	distStr := formatDistance(km, unit)
	if distStr == "" {
		distStr = "0 km"
	}
	return fmt.Sprintf("%s  ·  %s", distStr, bear)
}

func formatLocator(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return strings.ToUpper(s)
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case i < 2:
			if r >= 'a' && r <= 'z' {
				b.WriteRune(r - 32)
			} else {
				b.WriteRune(r)
			}
		case i < 4:
			b.WriteRune(r)
		default:
			if r >= 'A' && r <= 'Z' {
				b.WriteRune(r + 32)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
