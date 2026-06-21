package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
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
		m.rc.pathCall = ""
		m.rc.pathSig = ""
		return ""
	}
	m.rc.pathCall = cur
	m.rc.pathSig = ""
	return cur
}

// lookupCallCmd returns a batch of lookup commands (QRZ + WL + DXC + filtered table)
// for the given callsign. Returns nil if the call is empty or invalid.
func (m *Model) lookupCallCmd(call string) tea.Cmd {
	if call == "" || !qso.IsValidCall(call) {
		return nil
	}
	var cmds []tea.Cmd
	if m.App.Config.Integrations.QRZ.Enabled && m.App.Config.Integrations.QRZ.User != "" {
		cmds = append(cmds, m.qrzLookup(call))
	}
	wl := m.App.Logbook.Wavelog
	if wl != nil && wl.Enabled && wl.APIKey != "" {
		cmds = append(cmds, m.wlLookup(call))
	}
	cmds = append(cmds, m.dxcSpotLookupCmd(call))
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
		// Skip hidden fields.
		for m.isFieldHidden(m.focus) && m.focus != fieldComment {
			m.focus = (m.focus + 1) % fieldCount
		}
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
		// Skip hidden fields.
		for m.isFieldHidden(m.focus) && m.focus > 0 {
			m.focus--
		}
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
			m.lookup.partnerData = nil
			m.lookup.wlPrivateData = nil
			m.lookup.wlLookupDone = false
			m.gridSource = gridSourceNone
			m.screen = screenQSO
			m.clearFilteredTable()
			m.invalidatePartnerMapCache()
			m.rc.pathCall = ""
			m.rc.pathGrid = ""
			m.rc.pathSig = ""
			m.rc.logStatsSig = ""
			m.updateSCP()
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
	// REF fields (SOTA/POTA/WWFF/IOTA): uppercasing and name resolution
	// are deferred to focus exit (onFieldExit) for performance.
}

// =============================================================================
// QSO form field exit — committed values & autofill
// =============================================================================

// onFieldExit is called when leaving a field (via nextField/prevField/focusField).
// It commits values for path calculation, validates the field (showing a toast
// if invalid), and triggers autofill / lookups.
func (m *Model) onFieldExit() {
	// Show validation toast for the field being left, if the value is non-empty
	// but invalid. Empty is OK — handled at save time by qso.ValidateForSave.
	if hint := m.qsoFieldHint(m.focus); hint != "" {
		m.toasts.Warn(hint)
	}

	switch m.focus {
	case fieldCall:
		cur := m.commitCall()
		m.autoFillRST()
		m.autoFillSSBSubmode()
		if cur == "" {
			// Callsign was cleared — reset last-looked-up state so that
			// re-entering the same call triggers a fresh lookup.
			m.lookup.qrzLastCall = ""
			break
		}
		// Defer lookup via flag — onFieldExit can't return commands.
		if cur != "" && !strings.EqualFold(cur, m.lookup.qrzLastCall) {
			m.lookup.qrzNeed = true
			m.lookup.qrzCall = cur
		}
		// Fill country/continent from DXCC — runs regardless of QRZ status.
		m.dxccAutoFill()
		// Callsign committed — hide SCP suggestions.
		m.scpMatches = nil
		m.scpCacheKey = ""

		// Contest prefill: fill exchange fields from active contest config.
		m.prefillContestExchange()

	case fieldGrid:
		m.rc.pathGrid = strings.ToUpper(strings.TrimSpace(m.fields[fieldGrid].Value()))
		m.gridSource = gridSourceManual
		m.invalidatePartnerMapCache()

	case fieldFreq:
		m.applyFreqDefaults()

	case fieldRSTSent, fieldRSTRcvd:
		// Changing RST should recalculate pre-filled exchange fields.
		m.prefillContestExchange()

	case fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA:
		m.fields[m.focus].SetValue(strings.ToUpper(m.fields[m.focus].Value()))
		m.ref.refNamesDirty = true
		m.applyRefGridAndQTH()
	}

	// Always re-check dupe on any field exit — cheap DB query.
	applog.Debug("dupe: onFieldExit calling checkDupe", "focus", int(m.focus))
	m.checkDupe()
}

// checkDupe sets m.dupe to true when the current call/band/mode/date
// combination already exists in the database, unless the existing QSO
// has different reference data (e.g. different SOTA summit same day).
func (m *Model) checkDupe() {
	applog.Debug("dupe: checkDupe called", "focus", int(m.focus))
	m.dupe = false
	if m.App == nil || m.App.DB == nil {
		applog.Debug("dupe: DB not available", "appNil", m.App == nil, "dbNil", m.App != nil && m.App.DB == nil)
		return
	}
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	band := qso.NormalizeBand(m.fields[fieldBand].Value())
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	date := qso.StripNonDigits(m.fields[fieldDate].Value())
	if call == "" || band == "" || mode == "" || date == "" {
		applog.Debug("dupe: fields empty", "call", call, "band", band, "mode", mode, "date", date)
		return
	}
	isDupe, existing := store.IsDuplicateQSO(m.App.DB, call, band, mode, date)
	if !isDupe || existing == nil {
		applog.Debug("dupe: no match", "call", call, "band", band, "mode", mode, "date", date)
		return
	}
	// If any reference field differs, it's not a dupe (e.g. different summit).
	formSOTA := strings.TrimSpace(m.fields[fieldSOTA].Value())
	formPOTA := strings.TrimSpace(m.fields[fieldPOTA].Value())
	formWWFF := strings.TrimSpace(m.fields[fieldWWFF].Value())
	formIOTA := strings.TrimSpace(m.fields[fieldIOTA].Value())
	if formSOTA != existing.SOTA || formPOTA != existing.POTA ||
		formWWFF != existing.WWFF || formIOTA != existing.IOTA {
		applog.Debug("dupe: ref mismatch — not a dupe", "formSOTA", formSOTA, "dbSOTA", existing.SOTA)
		return
	}
	m.dupe = true
	applog.Debug("dupe: DETECTED", "call", call, "band", band, "mode", mode, "date", date)
}

// clearForm resets the entire QSO form for a new QSO: clears fields (with
// retention of rig values and comment), partner lookup data, navigation state,
// and the recent-QSO filter table.  Partner data is preserved when the user is
// actively viewing the Partner/Image screen.
func (m *Model) clearForm() {
	m.dupe = false
	m.resetQSOFields()
	m.resetPartnerLookup()
	m.resetNavigation()
	m.clearFilteredTable()
	m.scpMatches = nil
	m.scpCacheKey = ""
}

// resetQSOFields blanks every field, restores date/time to UTC now, and
// re-applies retained rig values (band, freq, mode, submode, power) and
// the retain-comment setting.  Focus moves to the Call field.
func (m *Model) resetQSOFields() {
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
	m.gridSource = gridSourceNone
	m.ref.refNamesDirty = true
	m.focus = fieldCall
	m.fields[m.focus].Focus()
}

// resetPartnerLookup clears QRZ and Wavelog lookup data so the next
// lookup request starts from a clean state.  Skipped when the user is
// actively viewing the Partner or Image screen.
func (m *Model) resetPartnerLookup() {
	if m.screen != screenPartner && m.screen != screenImage {
		m.lookup.partnerData = nil
		m.lookup.wlPrivateData = nil
		m.lookup.wlLookupDone = false
	}
}

// resetNavigation clears the cached path line and the committed call/grid
// values so the next path calculation starts fresh.
func (m *Model) resetNavigation() {
	m.rc.pathCall = ""
	m.rc.pathGrid = ""
	m.rc.pathLine = ""
	m.rc.pathSig = ""
}
