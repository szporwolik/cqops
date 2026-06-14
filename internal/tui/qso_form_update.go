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

// focusField sets focus to the specified QSO form field.
func (m *Model) focusField(f field) {
	m.retainFocused = false
	if m.focus == fieldFreq {
		m.applyFreqDefaults()
	}
	m.fields[m.focus].Blur()
	m.focus = f
	m.fields[m.focus].Focus()
}

// nextField moves focus to the next QSO form field in sequence.
func (m *Model) nextField() {
	if m.focus == fieldFreq {
		m.applyFreqDefaults()
	}
	wasCall := m.focus == fieldCall

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
	if wasCall {
		cur := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		if cur != "" && !strings.EqualFold(cur, m.qrzLastCall) {
			m.qrzNeed = true
			m.qrzCall = cur
		}
		m.autoFillRST()
		m.autoFillSSBSubmode()
	}
}

// prevField moves focus to the previous QSO form field in sequence.
func (m *Model) prevField() {
	if m.focus == fieldFreq {
		m.applyFreqDefaults()
	}
	wasCall := m.focus == fieldCall

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
	if wasCall {
		cur := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		if cur != "" && !strings.EqualFold(cur, m.qrzLastCall) {
			m.qrzNeed = true
			m.qrzCall = cur
		}
		m.autoFillRST()
		m.autoFillSSBSubmode()
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
// QSO form field update and clear
// =============================================================================

// updateFocused handles generic keypress input for the focused QSO form field.
func (m *Model) updateFocused(msg tea.KeyPressMsg) {
	if m.retainFocused {
		return
	}
	prevCall := strings.TrimSpace(m.fields[fieldCall].Value())
	prevVal := m.fields[m.focus].Value()
	prevFreq := m.fields[fieldFreq].Value()
	m.fields[m.focus], _ = m.fields[m.focus].Update(msg)

	// Frequency changed: derive band/mode/submode on field exit (deferred to focus change).
	if m.focus == fieldFreq && m.fields[fieldFreq].Value() != prevFreq {
		// Deferred.
	}
	// Band changed: derive mode/submode from the new band.
	if m.focus == fieldBand && m.fields[m.focus].Value() != prevVal {
		m.applyFreqDefaults()
	}
	// Date/time manually changed: disable auto-update.
	if (m.focus == fieldDate || m.focus == fieldTime) && m.fields[m.focus].Value() != prevVal {
		m.dateTimeAuto = false
	}
	// Call: auto-uppercase.
	if m.focus == fieldCall {
		m.fields[m.focus].SetValue(strings.ToUpper(m.fields[m.focus].Value()))
	}
	// Grid: auto-format.
	if m.focus == fieldGrid {
		m.fields[m.focus].SetValue(formatLocator(m.fields[m.focus].Value()))
	}
	// Call changed: clear stale QRZ/WL data.
	if m.focus == fieldCall {
		cur := strings.TrimSpace(m.fields[fieldCall].Value())
		if cur != prevCall {
			if m.partnerData != nil && !strings.EqualFold(m.partnerData.Callsign, cur) {
				m.partnerData = nil
				m.wlPrivateData = nil
				m.screen = screenQSO
				m.fields[fieldGrid].SetValue("")
				m.fields[fieldQTH].SetValue("")
				m.fields[fieldCountry].SetValue("")
			}
		}
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
	m.fields[fieldDate].SetValue(now.Format("2006-01-02"))
	m.fields[fieldTime].SetValue(now.Format("15:04:05"))

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
	m.screen = screenQSO
}
