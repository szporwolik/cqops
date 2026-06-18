package tui

import (
	"fmt"

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
			// Fill QSO form with highlighted spot and jump to form.
			m.dxcFillFromSelected()
			m.screen = screenQSO
			return m, cmd

		case "tab":
			// Fill QSO form with highlighted spot and tune rig.
			m.dxcFillFromSelected()
			cmd = tea.Batch(cmd, m.dxcTuneCmd())
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

	// Fill frequency: when WSJT-X is offline, use DXC spot frequency.
	if !m.wsjtx.online {
		freqMHz := spot.Frequency / 1000
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.5f", freqMHz))
		m.applyFreqDefaults()
		applog.Info("DXC: populated QSO form from spot",
			"call", spot.DXCall,
			"freq", fmt.Sprintf("%.1f kHz", spot.Frequency),
		)
	} else {
		applog.Info("DXC: populated QSO form call from spot",
			"call", spot.DXCall,
		)
	}

	// Commit the callsign (normalizes, sets pathCall) and flag a deferred
	// lookup — same logic as onFieldExit for the call field.
	cur := m.commitCall()
	if cur != "" {
		m.autoFillRST()
		m.autoFillSSBSubmode()
		m.lookup.qrzNeed = true
		m.lookup.qrzCall = cur
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
