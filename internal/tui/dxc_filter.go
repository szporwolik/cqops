package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// dxcFilteredSpots returns spots filtered by current band, time, and mode settings.
func (m *Model) dxcFilteredSpots() []store.DXCSpot {
	spots, err := store.QueryDXCSpots(m.App.DB)
	if err != nil {
		return nil
	}

	// Time filter.
	if m.dxc.timeFilter > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(m.dxc.timeFilter) * time.Minute).Unix()
		var filtered []store.DXCSpot
		for _, s := range spots {
			if s.ReceivedAt >= cutoff {
				filtered = append(filtered, s)
			}
		}
		spots = filtered
	}

	// Band filter.
	if m.dxc.bandFilter != "" {
		var filtered []store.DXCSpot
		for _, s := range spots {
			if m.dxc.bandFilter == "other" {
				if s.Band == "" {
					filtered = append(filtered, s)
				}
			} else if s.Band == m.dxc.bandFilter {
				filtered = append(filtered, s)
			}
		}
		spots = filtered
	}

	// Mode filter — uses pre-computed mode_cat column.
	if m.dxc.modeFilter != "" {
		var filtered []store.DXCSpot
		for _, s := range spots {
			if s.ModeCat == m.dxc.modeFilter {
				filtered = append(filtered, s)
			}
		}
		spots = filtered
	}

	// When a specific band is selected, sort by frequency descending
	// so the highest frequency in the band appears at the top.
	if m.dxc.bandFilter != "" {
		sort.Slice(spots, func(i, j int) bool {
			return spots[i].Frequency > spots[j].Frequency
		})
	}

	m.dxc.cachedSpots = spots
	return spots
}

// dxcAvailableBands returns bands sorted by frequency (wavelength), plus "other".
func (m *Model) dxcAvailableBands() []string {
	spots, err := store.QueryDXCSpots(m.App.DB)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	hasOther := false
	for _, s := range spots {
		if s.Band == "" {
			hasOther = true
		} else {
			seen[s.Band] = true
		}
	}
	var bands []string
	for b := range seen {
		bands = append(bands, b)
	}
	// Sort by frequency (lowest freq / longest wavelength first).
	sort.Slice(bands, func(i, j int) bool {
		return qso.BandIndex(bands[i]) < qso.BandIndex(bands[j])
	})
	if hasOther {
		bands = append(bands, "other")
	}
	return bands
}

// dxcBandChoices returns the ordered list of band filter choices: "" (all),
// sorted band names, then "other" if present.
func (m *Model) dxcBandChoices() []string {
	bands := m.dxcAvailableBands()
	choices := []string{""}
	hasOther := false
	for _, b := range bands {
		if b == "other" {
			hasOther = true
		} else {
			choices = append(choices, b)
		}
	}
	if hasOther {
		choices = append(choices, "other")
	}
	return choices
}

// dxcAvailableModes returns the mode filter categories: CW, DIGI, PHONE.
func (m *Model) dxcAvailableModes() []string {
	return []string{"CW", "DIGI", "PHONE"}
}

// spotModeCategory maps an individual spot mode to a filter category.
// Categories: CW, DIGI (digital), PHONE (voice).
func spotModeCategory(mode string) string {
	switch strings.ToUpper(mode) {
	case "CW", "CW-L", "CW-U", "CWL", "CWU", "CW-R":
		return "CW"
	case "FT8", "FT4", "RTTY", "PSK", "JT65", "JT9", "MSK144", "FSK", "DATA", "DATA-U", "DATA-L", "DATA-FM":
		return "DIGI"
	case "USB", "LSB", "AM", "FM":
		return "PHONE"
	default:
		return ""
	}
}
