package tui

import (
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/store"
)

// handleDXCUpdate routes messages to the DXC table for keyboard navigation
// and handles filter keybindings.
func (m *Model) handleDXCUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "f4":
			m.screen = screenQSO
			return m, cmd

		case "pgup":
			// Cycle time window forward.
			m.dxc.timeIdx = (m.dxc.timeIdx + 1) % len(dxcTimeWindows)
			m.dxc.timeFilter = dxcTimeWindows[m.dxc.timeIdx]
			m.dxc.tableReady = false
			return m, cmd

		case "pgdown":
			// Cycle time window backward.
			m.dxc.timeIdx--
			if m.dxc.timeIdx < 0 {
				m.dxc.timeIdx = len(dxcTimeWindows) - 1
			}
			m.dxc.timeFilter = dxcTimeWindows[m.dxc.timeIdx]
			m.dxc.tableReady = false
			return m, cmd

		case "home":
			// Cycle band filter forward.
			choices := m.dxcBandChoices()
			if len(choices) > 0 {
				m.dxc.bandIdx = (m.dxc.bandIdx + 1) % len(choices)
				m.dxc.bandFilter = choices[m.dxc.bandIdx]
			}
			m.dxc.tableReady = false
			return m, cmd

		case "end":
			// Cycle band filter backward.
			choices := m.dxcBandChoices()
			if len(choices) > 0 {
				m.dxc.bandIdx--
				if m.dxc.bandIdx < 0 {
					m.dxc.bandIdx = len(choices) - 1
				}
				m.dxc.bandFilter = choices[m.dxc.bandIdx]
			}
			m.dxc.tableReady = false
			return m, cmd

		case `\`:
			// Cycle continent filter forward.
			choices := m.dxcContChoices()
			if len(choices) > 0 {
				m.dxc.contIdx = (m.dxc.contIdx + 1) % len(choices)
				m.dxc.contFilter = choices[m.dxc.contIdx]
			}
			m.dxc.tableReady = false
			return m, cmd

		case "enter":
			// Fill QSO form with highlighted spot, tune rig, and jump to form.
			m.dxcFillFromSelected()
			cmd = tea.Batch(cmd, m.dxcTuneCmd())
			m.screen = screenQSO
			return m, cmd

		case "insert":
			// Cycle mode filter forward.
			modes := m.dxcAvailableModes()
			choices := []string{""} // "" means all
			choices = append(choices, modes...)
			if len(choices) > 0 {
				m.dxc.modeIdx = (m.dxc.modeIdx + 1) % len(choices)
				m.dxc.modeFilter = choices[m.dxc.modeIdx]
			}
			m.dxc.tableReady = false
			return m, cmd

		case "delete":
			// Cycle mode filter backward.
			modes := m.dxcAvailableModes()
			choices := []string{""}
			choices = append(choices, modes...)
			if len(choices) > 0 {
				m.dxc.modeIdx--
				if m.dxc.modeIdx < 0 {
					m.dxc.modeIdx = len(choices) - 1
				}
				m.dxc.modeFilter = choices[m.dxc.modeIdx]
			}
			m.dxc.tableReady = false
			return m, cmd

		case "backspace":
			// Clear all filters.
			m.dxc.timeFilter = 0
			m.dxc.timeIdx = 0
			m.dxc.bandFilter = ""
			m.dxc.bandIdx = 0
			m.dxc.contFilter = ""
			m.dxc.contIdx = 0
			m.dxc.modeFilter = ""
			m.dxc.modeIdx = 0
			m.dxc.tableReady = false
			return m, cmd
		}
	}

	if m.dxc.tableReady {
		// Space is reserved for tune — don't forward to the table.
		if kp, ok := msg.(tea.KeyPressMsg); ok && (kp.String() == " " || kp.String() == "space" || kp.Code == ' ') {
			cmd = tea.Batch(cmd, m.dxcTuneCmd())
			return m, cmd
		}
		t, c := m.dxc.table.Update(msg)
		m.dxc.table = t
		m.updateDXCSelectedCall()
		if c != nil {
			cmd = tea.Batch(cmd, c)
		}
	}

	return m, cmd
}

// dxcFillFromSelected fills the QSO form with the currently highlighted DXC spot.
func (m *Model) dxcFillFromSelected() {
	spot, ok := m.dxcSpotAtCursor()
	if !ok {
		return
	}

	// Fill callsign.
	m.fields[fieldCall].SetValue(spot.DXCall)
	m.lookup.partnerData = nil
	m.lookup.wlPrivateData = nil
	m.lookup.wlLookupDone = false
	m.invalidatePartnerMapCache()
	// Clear QRZ-populated fields so old callsign data does not bleed.
	m.fields[fieldName].SetValue("")
	m.fields[fieldQTH].SetValue("")
	m.fields[fieldGrid].SetValue("")
	m.fields[fieldCountry].SetValue("")

	// Fill frequency: when WSJT-X is offline, use DXC spot frequency.
	// Always set band and mode from the spot — these are known regardless
	// of flrig/WSJT-X state.
	if spot.Band != "" {
		m.fields[fieldBand].SetValue(spot.Band)
	}
	if spot.Mode != "" {
		m.fields[fieldMode].SetValue(spot.Mode)
	}
	if !m.wsjtx.online {
		freqMHz := spot.Frequency / 1000
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.5f", freqMHz))
		// Don't call applyFreqDefaults — band and mode are already set from
		// the spot above. Only auto-fill SSB submode if needed.
		m.autoFillSSBSubmode()
		applog.Info("DXC: populated QSO form from spot",
			"call", spot.DXCall,
			"freq", fmt.Sprintf("%.1f kHz", spot.Frequency),
		)
	} else {
		applog.Info("DXC: populated QSO form call from spot",
			"call", spot.DXCall,
		)
	}

	// Parse spot comment for reference designators (SOTA, POTA, WWFF, IOTA)
	// and auto-fill the corresponding QSO form fields.
	m.parseSpotCommentForRefs(spot.Comment)

	// Commit the callsign (normalizes, sets pathCall). Lookups (QRZ, Wavelog)
	// are dispatched by onFieldExit when the user leaves the call field — no
	// need to double-dispatch here.
	cur := m.commitCall()
	if cur != "" {
		m.autoFillRST()
		m.autoFillSSBSubmode()
		m.dxccAutoFill()
	}
}

// dxcSpotAtCursor returns the DXC spot at the current table cursor position.
// Returns the spot data captured at the last cursor move — NOT a fresh DB query,
// so the frequency matches what the user sees in the table.
func (m *Model) dxcSpotAtCursor() (store.DXCSpot, bool) {
	if m.dxc.selectedCall == "" || !m.dxc.tableReady {
		return store.DXCSpot{}, false
	}
	if m.dxc.selectedSpot.DXCall == "" {
		return store.DXCSpot{}, false
	}
	applog.Debug("DXC: dxcSpotAtCursor",
		"call", m.dxc.selectedSpot.DXCall,
		"freq_khz", m.dxc.selectedSpot.Frequency,
	)
	return m.dxc.selectedSpot, true
}

// updateDXCSelectedCall updates the cached selected spot from the current
// table cursor position. Call after cursor movement or table rebuild.
// Stores the FULL spot data so frequency is locked at cursor-move time.
func (m *Model) updateDXCSelectedCall() {
	if !m.dxc.tableReady {
		m.dxc.selectedCall = ""
		m.dxc.selectedSpot = store.DXCSpot{}
		return
	}
	cursor := m.dxc.table.Cursor()
	spots := m.dxc.cachedSpots
	if len(spots) == 0 {
		spots = m.dxcFilteredSpots()
	}
	prev := m.dxc.selectedCall
	if cursor >= 0 && cursor < len(spots) {
		m.dxc.selectedSpot = spots[cursor]
		m.dxc.selectedCall = m.dxc.selectedSpot.DXCall
	} else {
		m.dxc.selectedCall = ""
		m.dxc.selectedSpot = store.DXCSpot{}
	}
	if m.dxc.selectedCall != prev {
		applog.Debug("DXC: selected call changed",
			"cursor", cursor,
			"prev", prev,
			"new", m.dxc.selectedCall,
			"freq_khz", m.dxc.selectedSpot.Frequency,
		)
	}
}

// wellKnownWWFFPrefixes lists the most common country-specific WWFF prefixes
// (top ~10 ham nations). The generic "WWFF" prefix is always checked first.
var wellKnownWWFFPrefixes = []string{
	"KFF", "DLFF", "SPFF", "FFF", "GFF", "JAFF",
	"IFF", "EAFF", "OZFF", "VKFF", "VEFF", "ONFF",
}

// parseSpotCommentForRefs scans a DXC spot comment for SOTA, POTA, WWFF, and
// IOTA reference designators and fills the corresponding QSO form fields.
// Only fills empty fields — existing values are never overwritten.
func (m *Model) parseSpotCommentForRefs(comment string) {
	if comment == "" {
		return
	}
	upper := strings.ToUpper(strings.TrimSpace(comment))

	// IOTA: two letters, dash, digits (e.g. "EU-005", "NA-001")
	if m.fields[fieldIOTA].Value() == "" {
		re := regexp.MustCompile(`\b([A-Z]{2}-\d{3,4})\b`)
		if match := re.FindStringSubmatch(upper); match != nil {
			m.fields[fieldIOTA].SetValue(match[1])
		}
	}

	// SOTA: country/association prefix, slash, summit code (e.g. "SP/BZ-001")
	if m.fields[fieldSOTA].Value() == "" {
		re := regexp.MustCompile(`\b([A-Z0-9]+/[A-Z0-9]+-\d{3,4})\b`)
		if match := re.FindStringSubmatch(upper); match != nil {
			m.fields[fieldSOTA].SetValue(match[1])
		}
	}

	// WWFF: check BEFORE POTA — WWFF prefixes (e.g. DLFF, VEFF) would
	// otherwise match the broader POTA pattern first.
	if m.fields[fieldWWFF].Value() == "" {
		prefixes := append([]string{"WWFF"}, wellKnownWWFFPrefixes...)
		pattern := `\b((?:` + strings.Join(prefixes, "|") + `)-\d{3,5})\b`
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(upper); match != nil {
			m.fields[fieldWWFF].SetValue(match[1])
		}
	}

	// POTA: country prefix, dash, digits (e.g. "SP-0001", "K-0001").
	// Exclude WWFF-like patterns (any prefix ending in "FF") and KHz/DB- noise.
	if m.fields[fieldPOTA].Value() == "" {
		re := regexp.MustCompile(`\b([A-Z0-9]{1,4}-\d{4,6})\b`)
		for _, match := range re.FindAllStringSubmatch(upper, -1) {
			ref := match[1]
			// Skip if it looks like a WWFF reference (prefix ending in FF).
			if strings.HasSuffix(strings.SplitN(ref, "-", 2)[0], "FF") {
				continue
			}
			if strings.Contains(ref, "KHZ") || strings.Contains(ref, "DB-") {
				continue
			}
			m.fields[fieldPOTA].SetValue(ref)
			break
		}
	}
}
