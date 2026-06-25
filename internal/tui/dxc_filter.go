package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// dxcFilteredSpots returns spots filtered by current band, time, and mode settings.
// Uses cached spots when available and filters haven't changed.
// The cache is invalidated by handleDXCSpotsStored when new spots arrive,
// and by dxcInvalidateSpotCache when the DXC connection resets.
func (m *Model) dxcFilteredSpots() []store.DXCSpot {
	// Check cache validity: must match current filter state AND have data.
	if m.dxc.cachedSpots != nil &&
		m.dxc.cachedBandFilter == m.dxc.bandFilter &&
		m.dxc.cachedTimeFilter == m.dxc.timeFilter &&
		m.dxc.cachedContFilter == m.dxc.contFilter &&
		m.dxc.cachedModeFilter == m.dxc.modeFilter {
		return m.dxc.cachedSpots
	}

	spots, err := store.QueryDXCSpots(m.App.DB)
	if err != nil {
		return nil
	}

	// Populate band and continent filter caches from the raw query result
	// so dxcAvailableBands/Continents don't need their own DB scan.
	if m.dxc.cachedBands == nil || m.dxc.cachedConts == nil {
		m.populateDXCFilterCaches(spots)
	}

	// Single-pass in-memory filter — avoids allocating 3-4 intermediate slices.
	// Pre-compute filter predicates once.
	var cutoff int64
	if m.dxc.timeFilter > 0 {
		cutoff = time.Now().UTC().Add(-time.Duration(m.dxc.timeFilter) * time.Minute).Unix()
	}
	bandFilter := m.dxc.bandFilter
	contFilter := m.dxc.contFilter
	modeFilter := m.dxc.modeFilter
	hasFilter := cutoff > 0 || bandFilter != "" || contFilter != "" || modeFilter != ""

	if hasFilter {
		// Single pass: keep matching spots by writing to spots[:n].
		n := 0
		for _, s := range spots {
			if cutoff > 0 && s.ReceivedAt < cutoff {
				continue
			}
			if bandFilter != "" {
				if bandFilter == "other" {
					if s.Band != "" {
						continue
					}
				} else if s.Band != bandFilter {
					continue
				}
			}
			if contFilter != "" && s.DXCont != contFilter {
				continue
			}
			if modeFilter != "" && s.ModeCat != modeFilter {
				continue
			}
			spots[n] = s
			n++
		}
		spots = spots[:n]
	}

	// When a specific band is selected, sort by frequency descending
	// so the highest frequency in the band appears at the top.
	// Only sort when the band filter changes (not on every spot refresh)
	// since spots already arrive time-ordered from the database.
	if bandFilter != "" && bandFilter != m.dxc.cachedSortBand {
		sort.Slice(spots, func(i, j int) bool {
			return spots[i].Frequency > spots[j].Frequency
		})
	}

	m.dxc.cachedSpots = spots
	m.dxc.cachedBandFilter = bandFilter
	m.dxc.cachedTimeFilter = m.dxc.timeFilter
	m.dxc.cachedContFilter = contFilter
	m.dxc.cachedModeFilter = modeFilter
	m.dxc.cachedSortBand = bandFilter
	return spots
}

// dxcInvalidateSpotCache clears the cached filtered spots so the next call
// to dxcFilteredSpots re-queries the database. Called after new spots are
// stored or when the DXC client reconnects.
func (m *Model) dxcInvalidateSpotCache() {
	m.dxc.cachedSpots = nil
	m.dxc.cachedBandFilter = ""
	m.dxc.cachedTimeFilter = -1
	m.dxc.cachedContFilter = ""
	m.dxc.cachedModeFilter = ""
	m.dxc.cachedSortBand = ""
}

// populateDXCFilterCaches extracts band and continent lists from raw spots
// and stores them so dxcAvailableBands/Continents avoid a separate DB scan.
func (m *Model) populateDXCFilterCaches(spots []store.DXCSpot) {
	seenBands := map[string]bool{}
	hasOther := false
	seenConts := map[string]bool{}
	for _, s := range spots {
		if s.Band == "" {
			hasOther = true
		} else {
			seenBands[s.Band] = true
		}
		if s.DXCont != "" {
			seenConts[s.DXCont] = true
		}
	}
	var bands []string
	for b := range seenBands {
		bands = append(bands, b)
	}
	sort.Slice(bands, func(i, j int) bool {
		return qso.BandIndex(bands[i]) < qso.BandIndex(bands[j])
	})
	if hasOther {
		bands = append(bands, "other")
	}
	m.dxc.cachedBands = bands

	var conts []string
	for c := range seenConts {
		conts = append(conts, c)
	}
	sort.Strings(conts)
	m.dxc.cachedConts = conts
}

// dxcAvailableBands returns bands sorted by frequency (wavelength), plus "other".
// Cache is pre-populated by dxcFilteredSpots; the DB fallback handles cold starts.
func (m *Model) dxcAvailableBands() []string {
	if m.dxc.cachedBands != nil {
		return m.dxc.cachedBands
	}
	// Cold cache — query DB directly (rare; normally populated by dxcFilteredSpots).
	spots, err := store.QueryDXCSpots(m.App.DB)
	if err != nil {
		return nil
	}
	m.populateDXCFilterCaches(spots)
	return m.dxc.cachedBands
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

// dxcAvailableContinents returns unique continent codes from spots, sorted.
// Cache is pre-populated by dxcFilteredSpots; the DB fallback handles cold starts.
func (m *Model) dxcAvailableContinents() []string {
	if m.dxc.cachedConts != nil {
		return m.dxc.cachedConts
	}
	// Cold cache — query DB directly (rare; normally populated by dxcFilteredSpots).
	spots, err := store.QueryDXCSpots(m.App.DB)
	if err != nil {
		return nil
	}
	m.populateDXCFilterCaches(spots)
	return m.dxc.cachedConts
}

// dxcContChoices returns continent filter choices: "" (all) then sorted codes.
func (m *Model) dxcContChoices() []string {
	conts := m.dxcAvailableContinents()
	choices := []string{""}
	choices = append(choices, conts...)
	return choices
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
