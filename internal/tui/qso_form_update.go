package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// QSO form field navigation
// =============================================================================

// commitCall commits the current callsign field value for path calculation
// and invalidates the path cache. Returns the normalized call, or empty string
// if the callsign is invalid (clears pathCall in that case).
func (m *Model) commitCall() string {
	cur := qso.NormalizeCall(m.fields[fieldCall].Value())
	if cur != "" && !qso.IsValidCall(cur) {
		m.pathCall = ""
		m.cachedPathSig = ""
		return ""
	}
	m.pathCall = cur
	m.cachedPathSig = ""
	return cur
}

// lookupCallCmd returns a batch of lookup commands (QRZ + WL + filtered table)
// for the given callsign. Returns nil if the call is empty or invalid.
func (m *Model) lookupCallCmd(call string) tea.Cmd {
	if call == "" || !qso.IsValidCall(call) {
		return nil
	}
	var cmds []tea.Cmd
	if m.App.Config.QRZ.Enabled && m.App.Config.QRZ.User != "" {
		cmds = append(cmds, m.qrzLookup(call))
	}
	wl := m.App.Logbook.Wavelog
	if wl != nil && wl.Enabled && wl.APIKey != "" {
		cmds = append(cmds, m.wlLookup(call))
	}
	cmds = append(cmds, m.updateFilteredTable())
	return tea.Batch(cmds...)
}

// focusField sets focus to the specified QSO form field.
func (m *Model) focusField(f field) {
	m.retainFocused = false
	m.onFieldExit()
	m.fields[m.focus].Blur()
	m.focus = f
	m.fields[m.focus].Focus()
}

// nextField moves focus to the next QSO form field in sequence.
func (m *Model) nextField() {
	m.onFieldExit()

	if m.retainFocused {
		m.retainFocused = false
		m.focus = 0
		m.fields[m.focus].Focus()
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == fieldComment {
		m.retainFocused = true
	} else {
		m.focus = (m.focus + 1) % fieldCount
		m.fields[m.focus].Focus()
	}
}

// prevField moves focus to the previous QSO form field in sequence.
func (m *Model) prevField() {
	m.onFieldExit()

	if m.retainFocused {
		m.retainFocused = false
		m.focus = fieldComment
		m.fields[m.focus].Focus()
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == 0 {
		m.retainFocused = true
	} else {
		m.focus--
		m.fields[m.focus].Focus()
	}
}

// =============================================================================
// QSO form field cycling (band, mode, submode)
// =============================================================================

// cycleFieldUp cycles the focused field's value upward.
func (m *Model) cycleFieldUp() {
	switch m.focus {
	case fieldBand:
		m.cycleBand(1)
	case fieldMode:
		m.cycleMode(1)
	case fieldSubmode:
		m.cycleSubmode(1)
	}
}

// cycleFieldDown cycles the focused field's value downward.
func (m *Model) cycleFieldDown() {
	switch m.focus {
	case fieldBand:
		m.cycleBand(-1)
	case fieldMode:
		m.cycleMode(-1)
	case fieldSubmode:
		m.cycleSubmode(-1)
	}
}

// cycleBand cycles the band field in the given direction (1 = up, -1 = down).
func (m *Model) cycleBand(dir int) {
	b := strings.ToUpper(strings.TrimSpace(m.fields[fieldBand].Value()))
	b = qso.NormalizeBand(b)
	list := qso.AllBands()
	idx := indexOfStr(list, b)
	idx += dir
	if idx < 0 {
		idx = len(list) - 1
	} else if idx >= len(list) {
		idx = 0
	}
	m.fields[fieldBand].SetValue(list[idx])
	m.autoFillSSBSubmode()
}

// cycleMode cycles the mode field in the given direction (1 = up, -1 = down).
func (m *Model) cycleMode(dir int) {
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	mode, _ = qso.NormalizeMode(mode, "")
	if !qso.IsValidMode(mode) {
		mode = ""
	}
	list := qso.CycleModes()
	idx := indexOfStr(list, mode)
	idx += dir
	if idx < 0 {
		idx = len(list) - 1
	} else if idx >= len(list) {
		idx = 0
	}
	m.fields[fieldMode].SetValue(list[idx])
	m.fields[fieldSubmode].SetValue("")
}

// cycleSubmode cycles the submode field in the given direction.
func (m *Model) cycleSubmode(dir int) {
	cur := strings.ToUpper(strings.TrimSpace(m.fields[fieldSubmode].Value()))
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	mode, _ = qso.NormalizeMode(mode, "")
	list := qso.SubmodesFor(mode)
	if len(list) == 0 {
		m.fields[fieldSubmode].SetValue("")
		return
	}
	idx := indexOfStr(list, cur)
	idx += dir
	if idx < 0 {
		idx = len(list) - 1
	} else if idx >= len(list) {
		idx = 0
	}
	m.fields[fieldSubmode].SetValue(list[idx])
}

// indexOfStr returns the index of s in list (case-insensitive), or -1.
func indexOfStr(list []string, s string) int {
	for i, v := range list {
		if strings.EqualFold(v, s) {
			return i
		}
	}
	return -1
}

// =============================================================================
// QSO form autofill methods
// =============================================================================

// autoFillRST auto-fills RST sent/rcvd based on mode defaults.
func (m *Model) autoFillRST() {
	if m.fields[fieldRSTSent].Value() != "" || m.fields[fieldRSTRcvd].Value() != "" {
		return
	}
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	if mode == "CW" {
		m.fields[fieldRSTSent].SetValue("599")
		m.fields[fieldRSTRcvd].SetValue("599")
	} else {
		m.fields[fieldRSTSent].SetValue("59")
		m.fields[fieldRSTRcvd].SetValue("59")
	}
}

// applyFreqDefaults derives band, mode, and submode from the frequency field.
func (m *Model) applyFreqDefaults() {
	freqStr := strings.TrimSpace(m.fields[fieldFreq].Value())
	if freqStr == "" {
		return
	}
	var freq float64
	fmt.Sscanf(freqStr, "%f", &freq)
	if freq <= 0 {
		return
	}

	band := qso.DeriveBand(freq)
	if band != "" {
		m.fields[fieldBand].SetValue(band)
	}

	low, _, _ := qso.BandRange(band)
	if low >= 50 {
		m.fields[fieldMode].SetValue("FM")
	} else {
		m.fields[fieldMode].SetValue("SSB")
	}

	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	switch mode {
	case "SSB":
		if freq < 10.0 {
			m.fields[fieldSubmode].SetValue("LSB")
		} else {
			m.fields[fieldSubmode].SetValue("USB")
		}
	case "FM":
		m.fields[fieldSubmode].SetValue("")
	}
}

// autoFillSSBSubmode delegates to applyFreqDefaults for SSB submode logic.
func (m *Model) autoFillSSBSubmode() {
	m.applyFreqDefaults()
}

// =============================================================================
// QSO form field update on typing
// =============================================================================

// updateFocused handles generic keypress input for the focused QSO form field.
// Each field has specific side effects documented inline.
func (m *Model) updateFocused(msg tea.KeyPressMsg) {
	if m.retainFocused {
		return
	}

	f := m.focus
	prevVal := m.fields[f].Value()
	m.fields[f], _ = m.fields[f].Update(msg)

	switch f {
	case fieldCall:
		// Uppercase. If call changed: invalidate all call-dependent state —
		// partner data, lookups, filtered table, path, and stats.
		m.fields[f].SetValue(strings.ToUpper(m.fields[f].Value()))
		if cur := strings.TrimSpace(m.fields[f].Value()); cur != strings.TrimSpace(prevVal) {
			m.partnerData = nil
			m.wlPrivateData = nil
			m.wlLookupDone = false
			m.screen = screenQSO
			m.clearFilteredTable()
			m.invalidatePartnerMapCache()
			m.pathCall = ""
			m.pathGrid = ""
			m.cachedPathSig = ""
			m.cachedLogStatsSig = ""
		}

	case fieldBand:
		// Changing band manually: derive default mode/submode.
		if m.fields[f].Value() != prevVal {
			m.applyFreqDefaults()
		}

	case fieldGrid:
		// Auto-format grid locator.
		m.fields[f].SetValue(formatLocator(m.fields[f].Value()))

	case fieldDate, fieldTime:
		// Manual edit: stop auto-updating date/time.
		if m.fields[f].Value() != prevVal {
			m.dateTimeAuto = false
		}
	}
	// fieldFreq: band/mode derivation deferred to focus exit (see onFieldExit).
}

// =============================================================================
// QSO form field exit — committed values & autofill
// =============================================================================

// onFieldExit is called when leaving a field (via nextField/prevField/focusField).
// It commits values for path calculation and triggers autofill / lookups.
func (m *Model) onFieldExit() {
	switch m.focus {
	case fieldCall:
		cur := m.commitCall()
		m.autoFillRST()
		m.autoFillSSBSubmode()
		if cur == "" {
			raw := strings.TrimSpace(m.fields[fieldCall].Value())
			if raw != "" {
				m.toasts.Warn("Not a valid callsign")
			}
			break
		}
		// Defer lookup via flag — onFieldExit can't return commands.
		if cur != "" && !strings.EqualFold(cur, m.qrzLastCall) {
			m.qrzNeed = true
			m.qrzCall = cur
		}

	case fieldGrid:
		m.pathGrid = strings.ToUpper(strings.TrimSpace(m.fields[fieldGrid].Value()))

	case fieldFreq:
		m.applyFreqDefaults()
	}
}

// clearForm resets all QSO form fields to defaults, preserving retained comment
// and rig-related field values.
func (m *Model) clearForm() {
	retainedComment := ""
	if m.retainComment {
		retainedComment = m.fields[fieldComment].Value()
	}

	rig := [5]struct {
		idx   field
		value string
	}{
		{fieldBand, m.fields[fieldBand].Value()},
		{fieldFreq, m.fields[fieldFreq].Value()},
		{fieldMode, m.fields[fieldMode].Value()},
		{fieldSubmode, m.fields[fieldSubmode].Value()},
		{fieldTXPower, m.fields[fieldTXPower].Value()},
	}

	for i := field(0); i < fieldCount; i++ {
		m.fields[i].SetValue("")
		m.fields[i].Blur()
	}
	now := time.Now().UTC()
	dateFmt, timeFmt := dateTimeFormats(m.width)
	m.fields[fieldDate].SetValue(now.Format(dateFmt))
	m.fields[fieldTime].SetValue(now.Format(timeFmt))

	for _, r := range rig {
		if r.value != "" {
			m.fields[r.idx].SetValue(r.value)
		}
	}
	if retainedComment != "" {
		m.fields[fieldComment].SetValue(retainedComment)
	}
	m.dateTimeAuto = true
	m.retainFocused = false
	m.focus = fieldCall
	m.fields[m.focus].Focus()
	m.partnerData = nil
	m.wlPrivateData = nil
	m.wlLookupDone = false
	m.clearFilteredTable()
	m.pathCall = ""
	m.pathGrid = ""
}
