package tui

import (
	"fmt"

	"github.com/ftl/hamradio/locator"
)

func gridDistance(ownGrid, partnerGrid string) string {
	if ownGrid == "" || partnerGrid == "" {
		return ""
	}
	own, err := locator.Parse(ownGrid)
	if err != nil {
		return ""
	}
	partner, err := locator.Parse(partnerGrid)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%.0f km", float64(locator.Distance(own, partner)))
}

func gridBearing(ownGrid, partnerGrid string) string {
	if ownGrid == "" || partnerGrid == "" {
		return ""
	}
	own, err := locator.Parse(ownGrid)
	if err != nil {
		return ""
	}
	partner, err := locator.Parse(partnerGrid)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%.0f°", float64(locator.Azimuth(own, partner)))
}
