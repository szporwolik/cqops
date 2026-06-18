package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/store"
)

// dxcTimeWindows is the list of time filter options in minutes (0 = all).
var dxcTimeWindows = []int{0, 60, 30, 15, 10, 5}

// dxcColWidths maps DXC column titles to minimum widths (matching editorColWidths).
var dxcColWidths = map[string]int{
	"Time": 8, "Freq": 8, "Band": 5, "Mode": 4, "DX Call": 10, "Spotter": 10, "Comment": 8,
}

// dxcColOrder is the fixed display order: Time, Freq, Band, Mode, DX Call, Spotter, Comment.
var dxcColOrder = []string{"Time", "Freq", "Band", "Mode", "DX Call", "Spotter", "Comment"}

// dxcColValue returns the display value for a DXC column and spot.
func dxcColValue(col string, s *store.DXCSpot) string {
	switch col {
	case "Time":
		return time.Unix(s.ReceivedAt, 0).UTC().Format("15:04:05")
	case "Freq":
		return fmt.Sprintf("%.1f", s.Frequency)
	case "Band":
		return s.Band
	case "Mode":
		return s.Mode
	case "DX Call":
		return s.DXCall
	case "Spotter":
		return s.Spotter
	case "Comment":
		return s.Comment
	}
	return ""
}

// dxcFilteredSpots returns spots filtered by current band and time settings.
func (m *Model) dxcFilteredSpots() []store.DXCSpot {
	spots, err := store.QueryDXCSpots(m.App.DB)
	if err != nil {
		return nil
	}

	// Time filter.
	if m.dxcTimeFilter > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(m.dxcTimeFilter) * time.Minute).Unix()
		var filtered []store.DXCSpot
		for _, s := range spots {
			if s.ReceivedAt >= cutoff {
				filtered = append(filtered, s)
			}
		}
		spots = filtered
	}

	// Band filter.
	if m.dxcBandFilter != "" {
		var filtered []store.DXCSpot
		for _, s := range spots {
			if m.dxcBandFilter == "other" {
				if s.Band == "" {
					filtered = append(filtered, s)
				}
			} else if s.Band == m.dxcBandFilter {
				filtered = append(filtered, s)
			}
		}
		spots = filtered
	}

	// Mode filter.
	if m.dxcModeFilter != "" {
		var filtered []store.DXCSpot
		for _, s := range spots {
			if s.Mode == m.dxcModeFilter {
				filtered = append(filtered, s)
			}
		}
		spots = filtered
	}

	// When a specific band is selected, sort by frequency descending
	// so the highest frequency in the band appears at the top.
	if m.dxcBandFilter != "" {
		sort.Slice(spots, func(i, j int) bool {
			return spots[i].Frequency > spots[j].Frequency
		})
	}

	return spots
}

// dxcAvailableBands returns a sorted list of unique bands present in the spots,
// plus "other" if any spots lack a band classification.
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
	sort.Strings(bands)
	if hasOther {
		bands = append(bands, "other")
	}
	return bands
}

// dxcAvailableModes returns a sorted list of unique modes present in the spots.
func (m *Model) dxcAvailableModes() []string {
	spots, err := store.QueryDXCSpots(m.App.DB)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, s := range spots {
		if s.Mode != "" {
			seen[s.Mode] = true
		}
	}
	var modes []string
	for mo := range seen {
		modes = append(modes, mo)
	}
	sort.Strings(modes)
	return modes
}

// buildDXCTable constructs the bubbles/table for DXC spots.
// Follows the exact same pattern as LogbookEditor.buildTable in logbook_editor_table.go.
func (m *Model) buildDXCTable() {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	tableH := h - 6
	if tableH < 5 {
		tableH = 5
	}
	bodyW := w - 4
	if bodyW < 20 {
		bodyW = 20
	}
	if bodyW > partnerMapMaxW {
		bodyW = partnerMapMaxW
	}

	// Use the fixed column order — no tier selection (always same 6 columns).
	names := dxcColOrder

	// Build columns from the fixed order.
	var cols []table.Column
	minTotal := 0
	for _, n := range names {
		cw := dxcColWidths[n]
		minTotal += cw
		cols = append(cols, table.Column{Title: n, Width: cw})
	}
	gaps := len(cols) - 1
	extra := bodyW - gaps - minTotal
	if extra > 0 {
		dist := 0
		for i := range cols {
			var share int
			switch cols[i].Title {
			case "Comment":
				share = extra * 4 / 10
			case "Spotter":
				share = extra * 3 / 10
			case "DX Call":
				share = extra * 2 / 10
			}
			cols[i].Width += share
			dist += share
		}
		if leftover := extra - dist; leftover > 0 {
			cols[len(cols)-1].Width += leftover
		}
	}

	spots := m.dxcFilteredSpots()
	m.dxcSpotCount = len(spots)

	filtered := m.dxcBandFilter != "" || m.dxcModeFilter != ""
	bandHighlight := S.Info

	var rows []table.Row
	for _, s := range spots {
		s := s // capture
		var row table.Row
		for _, n := range names {
			v := dxcColValue(n, &s)
			if v == "" {
				v = "\u2014"
			}
			if filtered && n == "Band" && v != "\u2014" && m.dxcBandFilter != "" {
				v = bandHighlight.Render(v)
			}
			if filtered && n == "Mode" && v != "\u2014" && m.dxcModeFilter != "" {
				v = bandHighlight.Render(v)
			}
			row = append(row, v)
		}
		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableH),
		table.WithWidth(bodyW),
	)
	sty := table.DefaultStyles()
	sty.Header = sty.Header.
		BorderForeground(P.TextDim).
		BorderBottom(true).
		Bold(false).
		Foreground(P.Text)
	// Highlight Band/Mode header when filter is active.
	if filtered {
		sty.Header = sty.Header.Foreground(P.Cursor)
	}
	t.SetStyles(sty)
	t.Focus()

	m.dxcTable = t
	m.dxcTableReady = true
	m.dxcBuiltW = w
	m.dxcBuiltH = h
	m.updateDXCSelectedCall()
}

// dxcView renders the DXC cluster spots table.
// Follows the exact same View pattern as LogbookEditor (logbook_editor_render.go).
func (m *Model) dxcView() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	bodyW := w - 4
	if bodyW < 20 {
		bodyW = 20
	}
	if bodyW > partnerMapMaxW {
		bodyW = partnerMapMaxW
	}

	// Rebuild the table when dimensions change (same as F7 editor).
	if m.dxcTableReady && (m.width != m.dxcBuiltW || m.height != m.dxcBuiltH) {
		m.dxcTableReady = false
	}
	if !m.dxcTableReady {
		m.buildDXCTable()
	}

	contentH := contentHeight(h)

	// Spacer row shows active filters — labels dimmed, values visible.
	timeVal := "all"
	if m.dxcTimeFilter > 0 {
		timeVal = fmt.Sprintf("%dm", m.dxcTimeFilter)
	}
	bandVal := "all"
	if m.dxcBandFilter != "" {
		bandVal = m.dxcBandFilter
	}
	modeVal := "all"
	if m.dxcModeFilter != "" {
		modeVal = m.dxcModeFilter
	}
	filterInfo := " " + DimStyle.Render("Filters:") + " " +
		DimStyle.Render("Time") + " " + ValueStyle.Render(timeVal) +
		" " + DimStyle.Render("|") + " " +
		DimStyle.Render("Band") + " " + ValueStyle.Render(bandVal) +
		" " + DimStyle.Render("|") + " " +
		DimStyle.Render("Mode") + " " + ValueStyle.Render(modeVal) +
		" " + DimStyle.Render("|") + " " +
		DimStyle.Render("Spots") + " " + ValueStyle.Render(fmt.Sprintf("%d", m.dxcSpotCount))
	spacer := lipgloss.NewStyle().Width(bodyW).Render(filterInfo)

	tablePart := lipgloss.NewStyle().
		MaxWidth(bodyW).
		Height(contentH - 1).
		Render(m.dxcTable.View())
	return lipgloss.JoinVertical(lipgloss.Left, spacer, tablePart)
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
			m.dxcTimeIdx = (m.dxcTimeIdx + 1) % len(dxcTimeWindows)
			m.dxcTimeFilter = dxcTimeWindows[m.dxcTimeIdx]
			m.dxcTableReady = false
			return m, cmd

		case "pgdown":
			// Cycle time window backward.
			m.dxcTimeIdx--
			if m.dxcTimeIdx < 0 {
				m.dxcTimeIdx = len(dxcTimeWindows) - 1
			}
			m.dxcTimeFilter = dxcTimeWindows[m.dxcTimeIdx]
			m.dxcTableReady = false
			return m, cmd

		case "home":
			// Cycle band filter forward.
			choices := m.dxcBandChoices()
			if len(choices) > 0 {
				m.dxcBandIdx = (m.dxcBandIdx + 1) % len(choices)
				m.dxcBandFilter = choices[m.dxcBandIdx]
			}
			m.dxcTableReady = false
			return m, cmd

		case "end":
			// Cycle band filter backward.
			choices := m.dxcBandChoices()
			if len(choices) > 0 {
				m.dxcBandIdx--
				if m.dxcBandIdx < 0 {
					m.dxcBandIdx = len(choices) - 1
				}
				m.dxcBandFilter = choices[m.dxcBandIdx]
			}
			m.dxcTableReady = false
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
				m.dxcModeIdx = (m.dxcModeIdx + 1) % len(choices)
				m.dxcModeFilter = choices[m.dxcModeIdx]
			}
			m.dxcTableReady = false
			return m, cmd

		case "delete":
			// Cycle mode filter backward.
			modes := m.dxcAvailableModes()
			choices := []string{""}
			choices = append(choices, modes...)
			if len(choices) > 0 {
				m.dxcModeIdx--
				if m.dxcModeIdx < 0 {
					m.dxcModeIdx = len(choices) - 1
				}
				m.dxcModeFilter = choices[m.dxcModeIdx]
			}
			m.dxcTableReady = false
			return m, cmd

		case "backspace":
			// Clear all filters.
			m.dxcTimeFilter = 0
			m.dxcTimeIdx = 0
			m.dxcBandFilter = ""
			m.dxcBandIdx = 0
			m.dxcModeFilter = ""
			m.dxcModeIdx = 0
			m.dxcTableReady = false
			return m, cmd
		}
	}

	if m.dxcTableReady {
		t, c := m.dxcTable.Update(msg)
		m.dxcTable = t
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
	m.partnerData = nil
	m.wlPrivateData = nil
	m.wlLookupDone = false
	m.invalidatePartnerMapCache()

	// Fill frequency: when WSJT-X is offline, use DXC spot frequency.
	if !m.wsjtxOnline {
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
}

// dxcSpotAtCursor returns the DXC spot at the current table cursor position.
// Returns the spot data captured at the last cursor move — NOT a fresh DB query,
// so the frequency matches what the user sees in the table.
func (m *Model) dxcSpotAtCursor() (store.DXCSpot, bool) {
	if m.dxcSelectedCall == "" || !m.dxcTableReady {
		return store.DXCSpot{}, false
	}
	if m.dxcSelectedSpot.DXCall == "" {
		return store.DXCSpot{}, false
	}
	applog.Debug("DXC: dxcSpotAtCursor",
		"call", m.dxcSelectedSpot.DXCall,
		"freq_khz", m.dxcSelectedSpot.Frequency,
	)
	return m.dxcSelectedSpot, true
}

// updateDXCSelectedCall updates the cached selected spot from the current
// table cursor position. Call after cursor movement or table rebuild.
// Stores the FULL spot data so frequency is locked at cursor-move time.
func (m *Model) updateDXCSelectedCall() {
	if !m.dxcTableReady {
		m.dxcSelectedCall = ""
		m.dxcSelectedSpot = store.DXCSpot{}
		return
	}
	cursor := m.dxcTable.Cursor()
	spots := m.dxcFilteredSpots()
	prev := m.dxcSelectedCall
	if cursor >= 0 && cursor < len(spots) {
		m.dxcSelectedSpot = spots[cursor]
		m.dxcSelectedCall = m.dxcSelectedSpot.DXCall
	} else {
		m.dxcSelectedCall = ""
		m.dxcSelectedSpot = store.DXCSpot{}
	}
	if m.dxcSelectedCall != prev {
		applog.Debug("DXC: selected call changed",
			"cursor", cursor,
			"prev", prev,
			"new", m.dxcSelectedCall,
			"freq_khz", m.dxcSelectedSpot.Frequency,
		)
	}
}

// dxcTuneCmd returns a tea.Cmd that tunes flrig to the highlighted spot's
// frequency and mode. Returns the result as a dxcTuneResultMsg for toasts.
// Captures spot data at call time so it reflects exactly what the user selected.
func (m *Model) dxcTuneCmd() tea.Cmd {
	if !m.rigConnected || m.wsjtxOnline || m.flrigClient == nil {
		applog.Info("DXC: tune skipped",
			"rigConnected", m.rigConnected,
			"wsjtxOnline", m.wsjtxOnline,
			"hasClient", m.flrigClient != nil,
		)
		return nil
	}
	spot, ok := m.dxcSpotAtCursor()
	if !ok {
		return nil
	}
	freqHz := int64(spot.Frequency * 1000)
	freqMHz := float64(freqHz) / 1_000_000
	call := spot.DXCall
	flrigModeIdx := flrigModeIndex(m.flrigModes, spotModeToFlrigMode(spot.Mode))
	flrigModeName := spot.Mode
	client := m.flrigClient

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		if err := client.SetFrequency(ctx, freqHz); err != nil {
			applog.Warn("DXC: tune rig freq failed",
				"call", call, "freq_mhz", fmt.Sprintf("%.5f", freqMHz), "error", err,
			)
			return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("freq: %w", err)}
		}

		// Wait for rig to settle, then verify the actual VFO frequency.
		time.Sleep(400 * time.Millisecond)
		status, err := client.Status(ctx)
		var verifyMsg string
		if err == nil && status.Connected {
			actualMHz := status.FrequencyMHz
			diff := actualMHz - freqMHz
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.001 {
				verifyMsg = fmt.Sprintf(" (rig at %.5f MHz)", actualMHz)
				applog.Warn("DXC: tune freq mismatch",
					"call", call,
					"requested_mhz", fmt.Sprintf("%.5f", freqMHz),
					"actual_mhz", fmt.Sprintf("%.5f", actualMHz),
				)
			}
		}

		if flrigModeIdx >= 0 {
			if err := client.SetMode(ctx, flrigModeIdx); err != nil {
				applog.Warn("DXC: tune rig mode failed",
					"call", call, "mode", flrigModeName, "idx", flrigModeIdx, "error", err,
				)
				return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("mode: %w", err)}
			}
		}
		applog.Info("DXC: rig tuned OK",
			"call", call,
			"freq_mhz", fmt.Sprintf("%.5f", freqMHz),
			"mode", flrigModeName,
			"verify", verifyMsg,
		)
		return dxcTuneResultMsg{call: call, freqMHz: freqMHz, mode: flrigModeName, verify: verifyMsg}
	}
}

// spotModeToFlrigMode maps a DXC spot mode string to a flrig-compatible mode.
// CW maps to CW-L (lower sideband) which is the standard for HF CW operation.
func spotModeToFlrigMode(spotMode string) string {
	switch strings.ToUpper(spotMode) {
	case "USB":
		return "USB"
	case "LSB":
		return "LSB"
	case "CW":
		return "CW-L"
	case "FM":
		return "FM"
	case "AM":
		return "AM"
	case "RTTY", "FSK":
		return "RTTY"
	case "FT8", "FT4", "JT65", "JT9", "MSK144", "PSK", "DATA":
		return "DATA-U"
	default:
		return ""
	}
}
