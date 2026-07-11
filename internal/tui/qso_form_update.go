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
	if !m.Offline && m.inetOnline {
		if m.App.Config.Integrations.QRZ.Enabled && m.App.Config.Integrations.QRZ.User != "" {
			cmds = append(cmds, m.qrzLookup(call))
		}
		wl := m.App.Logbook.Wavelog
		if wl != nil && wl.Enabled && wl.APIKey != "" {
			cmds = append(cmds, m.wlLookup(call))
		}
	}
	cmds = append(cmds, m.dxcSpotLookupCmd(call))
	cmds = append(cmds, m.updateFilteredTable())
	return tea.Batch(cmds...)
}

// focusField sets focus to the specified QSO form field.
func (m *Model) focusField(f field) {
	m.keepFocused = false
	m.onFieldExit()
	m.fields[m.focus].Blur()
	m.focus = f
	m.fields[m.focus].Focus()
}

// contestAutoFocusExchRcvd moves focus to the Exch Rcvd field after
// lookups complete, so the operator can type the received exchange and
// press Enter to log — a tight contest workflow.
func (m *Model) contestAutoFocusExchRcvd() {
	if m.App == nil || m.App.Config == nil {
		return
	}
	if m.App.Logbook.ActiveContest == "" {
		return
	}
	if m.screen != screenQSO || m.confirm != nil || m.spotDialog != nil {
		return
	}
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	if call == "" || !m.lookupsCompleteForCall(call) {
		return
	}
	// Don't steal focus if the user has already moved away from the call field.
	if m.focus != fieldCall {
		return
	}
	// Move cursor to end of Exch Rcvd so typing immediately appends.
	m.focusField(fieldExchRcvd)
	m.fields[fieldExchRcvd].SetCursor(len(m.fields[fieldExchRcvd].Value()))
}

// nextField moves focus to the next QSO form field in sequence.
func (m *Model) nextField() {
	m.onFieldExit()

	if m.keepFocused {
		if m.keepSubFocus == 0 {
			m.keepSubFocus = 1 // Keep → Retain
		} else {
			// Retain → Comment
			m.keepFocused = false
			m.keepSubFocus = 0
			m.focus = fieldComment
			m.fields[m.focus].Focus()
		}
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == fieldComment {
		m.keepFocused = true
		m.keepSubFocus = 0
	} else {
		col, pos := m.fieldColumnPos(m.focus)
		next := m.horizontalTabTarget(col, pos, +1)
		m.focus = next
		for m.isFieldHidden(m.focus) {
			col2, pos2 := m.fieldColumnPos(m.focus)
			m.focus = m.horizontalTabTarget(col2, pos2, +1)
		}
		m.fields[m.focus].Focus()
	}
}

// prevField moves focus to the previous QSO form field — horizontal column
// cycling in reverse (Right → Middle → Left → Right).
func (m *Model) prevField() {
	m.onFieldExit()

	if m.keepFocused {
		if m.keepSubFocus == 1 {
			m.keepSubFocus = 0 // Retain → Keep
		} else {
			m.keepFocused = false
			m.keepSubFocus = 0
			m.focus = fieldComment
			m.fields[m.focus].Focus()
		}
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == fieldComment {
		m.keepFocused = true
		m.keepSubFocus = 1 // Comment → Retain
		return
	}
	// Shift+Tab from field 0 wraps horizontally to right column, not to checkbox row.
	col, pos := m.fieldColumnPos(m.focus)
	next := m.horizontalTabTarget(col, pos, -1)
	m.focus = next
	for m.isFieldHidden(m.focus) {
		col2, pos2 := m.fieldColumnPos(m.focus)
		m.focus = m.horizontalTabTarget(col2, pos2, -1)
	}
	m.fields[m.focus].Focus()
}

// formCols is the pre-built QSO form column layout — immutable.
var formCols = [3][]field{formLeft, formMiddle, formRight}

// formColumns returns the three QSO form column arrays.
func formColumns() [3][]field { return formCols }

// fieldColumnPos returns the column index (0=left, 1=middle, 2=right, 3=comment)
// and the row position within that column for the given field.
func (m *Model) fieldColumnPos(f field) (col, pos int) {
	if f == fieldComment {
		return 3, 0
	}
	for ci, colFields := range formColumns() {
		for pi, cf := range colFields {
			if cf == f {
				return ci, pi
			}
		}
	}
	return 0, 0 // fallback
}

// horizontalTabTarget returns the field at the same row position in the next
// (dir=+1) or previous (dir=-1) column. Wraps around columns.
func (m *Model) horizontalTabTarget(fromCol, fromPos, dir int) field {
	cols := formColumns()
	toCol := (fromCol + dir + len(cols)) % len(cols)
	targets := cols[toCol]
	if fromPos >= len(targets) {
		fromPos = len(targets) - 1
	}
	return targets[fromPos]
}

// nextRowField moves focus down (↓). Each column wraps through its footer:
// col 0 bottom → Comment → col 0 top. col 1 → Keep. col 2 → Retain.
func (m *Model) nextRowField() {
	m.onFieldExit()

	if m.keepFocused {
		// In checkbox row: Down wraps to top of corresponding column.
		s := m.keepSubFocus
		m.keepFocused = false
		m.keepSubFocus = 0
		switch s {
		case 0:
			m.focus = formMiddle[0]
		case 1:
			m.focus = formRight[0]
		}
		m.fields[m.focus].Focus()
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == fieldComment {
		m.focus = formLeft[0]
		m.fields[m.focus].Focus()
		return
	}

	col, pos := m.fieldColumnPos(m.focus)
	if col < 0 || col > 2 {
		m.focus = (m.focus + 1) % fieldCount
	} else {
		colFields := formColumns()[col]
		if pos+1 < len(colFields) {
			m.focus = colFields[pos+1]
		} else {
			switch col {
			case 0:
				m.focus = fieldComment
			case 1:
				m.keepFocused = true
				m.keepSubFocus = 0
				return
			case 2:
				m.keepFocused = true
				m.keepSubFocus = 1
				return
			}
		}
	}
	for m.isFieldHidden(m.focus) {
		c, p := m.fieldColumnPos(m.focus)
		if c < 0 || c > 2 {
			m.focus = (m.focus + 1) % fieldCount
		} else {
			cf := formColumns()[c]
			if p+1 < len(cf) {
				m.focus = cf[p+1]
			} else {
				// Bottom of column with hidden fields → route to footer.
				switch c {
				case 0:
					m.focus = fieldComment
				case 1:
					m.keepFocused = true
					m.keepSubFocus = 0
					return
				case 2:
					m.keepFocused = true
					m.keepSubFocus = 1
					return
				}
			}
		}
	}
	m.fields[m.focus].Focus()
}

// prevRowField moves focus up (↑). Each column wraps through its footer:
// Comment → col 0 bottom. Keep → col 1 bottom. Retain → col 2 bottom.
func (m *Model) prevRowField() {
	m.onFieldExit()

	if m.keepFocused {
		// In checkbox row: Up wraps to bottom of corresponding column.
		s := m.keepSubFocus
		m.keepFocused = false
		m.keepSubFocus = 0
		switch s {
		case 0:
			m.focus = formMiddle[len(formMiddle)-1]
		case 1:
			m.focus = formRight[len(formRight)-1]
		}
		m.fields[m.focus].Focus()
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == fieldComment {
		// Go to the last visible field in the left column.
		m.focus = formLeft[len(formLeft)-1]
		for m.isFieldHidden(m.focus) {
			col, pos := m.fieldColumnPos(m.focus)
			if pos > 0 {
				m.focus = formColumns()[col][pos-1]
			} else {
				break
			}
		}
		m.fields[m.focus].Focus()
		return
	}

	col, pos := m.fieldColumnPos(m.focus)
	if col < 0 || col > 2 {
		m.focus--
		if m.focus < 0 {
			m.focus = fieldComment
		}
	} else {
		if pos > 0 {
			m.focus = formColumns()[col][pos-1]
		} else {
			// Top of column → go to footer.
			switch col {
			case 0:
				m.focus = fieldComment
			case 1:
				m.keepFocused = true
				m.keepSubFocus = 0
				return
			case 2:
				m.keepFocused = true
				m.keepSubFocus = 1
				return
			}
		}
	}
	for m.isFieldHidden(m.focus) {
		c, p := m.fieldColumnPos(m.focus)
		if c < 0 || c > 2 {
			m.focus--
			if m.focus < 0 {
				m.focus = fieldComment
			}
		} else {
			if p > 0 {
				m.focus = formColumns()[c][p-1]
			} else {
				switch c {
				case 0:
					m.focus = fieldComment
				case 1:
					m.keepFocused = true
					m.keepSubFocus = 0
					return
				case 2:
					m.keepFocused = true
					m.keepSubFocus = 1
					return
				}
			}
		}
	}
	m.fields[m.focus].Focus()
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
	curMode := strings.TrimSpace(m.fields[fieldMode].Value())
	if curMode == "" {
		if low >= 50 {
			m.fields[fieldMode].SetValue("FM")
		} else {
			m.fields[fieldMode].SetValue("SSB")
		}
	}

	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	switch mode {
	case "SSB":
		// Only auto-fill sideband when neither LSB nor USB is already
		// set (e.g. from a DXC spot that carried USB/LSB explicitly).
		curSub := strings.ToUpper(strings.TrimSpace(m.fields[fieldSubmode].Value()))
		if curSub != "LSB" && curSub != "USB" {
			if freq < 10.0 {
				m.fields[fieldSubmode].SetValue("LSB")
			} else {
				m.fields[fieldSubmode].SetValue("USB")
			}
		}
	default:
		// Clear any stale submode from a previous QSO — only SSB has
		// an auto-filled submode here. Other modes (CW, FT8, RTTY…)
		// either have no submode or the user sets it separately.
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
	if m.keepFocused {
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
			m.lookup.qrzLookupDone = false
			m.lookup.qrzLookupCall = ""
			m.lookup.wlLookupDone = false
			m.lookup.wlLookupCall = ""
			m.dupeConfirmed = false
			m.gridSource = gridSourceNone
			m.screen = screenQSO
			m.clearFilteredTable()
			m.invalidatePartnerMapCache()
			m.rc.pathCall = ""
			m.rc.pathGrid = ""
			m.rc.pathSig = ""
			m.rc.logStatsSig = ""
			// Clear QRZ-populated fields so old callsign data does not bleed
			// into the new callsign before the next lookup completes.
			m.fields[fieldName].SetValue("")
			m.fields[fieldQTH].SetValue("")
			m.fields[fieldGrid].SetValue("")
			m.fields[fieldCountry].SetValue("")
			m.updateSCP()
		}

	case fieldBand:
		// Only clear WL data when the band actually changes to a different
		// band (not on transient keystrokes that leave the same value).
		if m.fields[f].Value() != prevVal {
			m.applyFreqDefaults()
			newBand := qso.NormalizeBand(m.fields[f].Value())
			oldBand := qso.NormalizeBand(prevVal)
			if newBand != oldBand {
				if m.lookup.wlPrivateData != nil || m.lookup.wlLookupDone {
					m.lookup.wlPrivateData = nil
					m.lookup.wlLookupDone = false
				}
			}
		}

	case fieldMode:
		if m.fields[f].Value() != prevVal {
			newMode := qso.NormalizeRigMode(m.fields[f].Value())
			oldMode := qso.NormalizeRigMode(prevVal)
			if newMode != oldMode {
				if m.lookup.wlPrivateData != nil || m.lookup.wlLookupDone {
					m.lookup.wlPrivateData = nil
					m.lookup.wlLookupDone = false
				}
			}
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
// commitAndLookup finalizes the callsign field, triggers lookups and dupe
// check. Used by Insert, arrow-down field exit, and first Enter press.
// Returns a lookup command (or nil).  Callers that discard the command
// (e.g. arrow-down in onFieldExit) must set m.lookup.qrzNeed=true so that
// handlePendingRequests dispatches the lookup on the next tick.
func (m *Model) commitAndLookup() tea.Cmd {
	call := m.commitCall()
	if call == "" {
		// Not a valid callsign — but still hide SCP suggestions.
		m.scpMatches = nil
		m.scpCacheKey = ""
		return nil
	}
	m.scpMatches = nil
	m.scpCacheKey = ""
	m.dxccAutoFill()
	m.prefillPreviousContestExchange(call)
	m.prefillContestExchange()
	m.lookup.qrzCall = call
	m.lookup.wlCall = call
	m.autoFillRST()
	m.autoFillSSBSubmode()
	m.checkDupe()
	m.rc.pathSig = "" // invalidate cache to show DUPE badge
	m.rc.logStatsSig = ""
	return m.lookupCallCmd(call)
}

// buildSpotComment constructs the pre-filled spot comment.
//
// Format follows real-world DX cluster conventions:
//
//	[refs...] MODE
//
// Examples:
//
//	SSB                          (mode only)
//	POTA SP-0123 SSB             (refs + mode)
//	WWFF SPFF-0008 CW            (refs + mode)
//
// Mode is always included.
// Zulu time is appended by the cluster, not part of the comment.
func (m *Model) buildSpotComment() string {
	var parts []string

	// 0. Contest — when active, prepend the ADIF Contest-ID (or name).
	if m.App.Logbook.ActiveContest != "" {
		if ct, ok := m.App.Config.Contests[m.App.Logbook.ActiveContest]; ok {
			if ct.ContestID != "" {
				parts = append(parts, ct.ContestID)
			} else if ct.Name != "" {
				parts = append(parts, ct.Name)
			}
		}
	}

	// 1. References (SOTA, POTA, WWFF, IOTA, SIG).
	refs := map[string]string{
		"SOTA": strings.TrimSpace(m.fields[fieldSOTA].Value()),
		"POTA": strings.TrimSpace(m.fields[fieldPOTA].Value()),
		"WWFF": strings.TrimSpace(m.fields[fieldWWFF].Value()),
		"IOTA": strings.TrimSpace(m.fields[fieldIOTA].Value()),
		"SIG":  strings.TrimSpace(m.fields[fieldSIG].Value()),
	}
	for _, key := range []string{"SOTA", "POTA", "WWFF", "IOTA", "SIG"} {
		val := refs[key]
		if val == "" {
			continue
		}
		if key == "SIG" {
			parts = append(parts, "SIG "+val)
		} else {
			parts = append(parts, key+" "+val)
		}
	}

	// 2. Mode — always included at the end.
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	if mode == "" {
		submode := strings.ToUpper(strings.TrimSpace(m.fields[fieldSubmode].Value()))
		if submode != "" {
			mode = submode
		}
	}
	if mode != "" {
		parts = append(parts, mode)
	}

	return strings.Join(parts, " ")
}

// openSpotDialog validates preconditions and opens the spot dialog.
func (m *Model) openSpotDialog() tea.Cmd {
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	if call == "" {
		m.toasts.Warn("Enter a callsign to spot")
		return nil
	}
	freqStr := strings.TrimSpace(m.fields[fieldFreq].Value())
	if freqStr == "" {
		m.toasts.Warn("Enter a frequency to spot")
		return nil
	}
	var freqMhz float64
	fmt.Sscanf(freqStr, "%f", &freqMhz)
	if freqMhz <= 0 {
		m.toasts.Warn("Enter a valid frequency to spot")
		return nil
	}
	freqKhz := freqMhz * 1000

	comment := m.buildSpotComment()
	m.spotDialog = &SpotDialog{}
	*m.spotDialog = NewSpotDialog(call, freqKhz, comment)
	return nil
}

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
			m.lookup.qrzLastCall = ""
			break
		}
		m.lookup.qrzLastCall = strings.ToUpper(cur)
		m.lookup.pendingLookupCmd = m.commitAndLookup()
		// Also flag for handlePendingRequests as fallback (tick loop).

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
// combination already exists in the database. Result is cached by
// (call,band,mode,date) to avoid re-querying on every field navigation.
func (m *Model) checkDupe() {
	m.dupe = false
	if m.App == nil || m.App.DB == nil {
		applog.Debug("dupe: checkDupe bail — no App/DB")
		return
	}
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	band := qso.NormalizeBand(m.fields[fieldBand].Value())
	// Normalize via rig mode table so spot-derived modes ("USB"/"LSB")
	// match QSOs saved from the rig ("SSB"). NormalizeRigMode maps
	// USB→SSB, LSB→SSB, and passes through already-normalized values.
	mode := qso.NormalizeRigMode(m.fields[fieldMode].Value())
	date := qso.StripNonDigits(m.fields[fieldDate].Value())
	if call == "" || band == "" || mode == "" || date == "" {
		applog.Debug("dupe: checkDupe bail — missing field",
			"call", call, "band", band, "mode", mode, "date", date)
		return
	}
	// Cache key — date changes once per day, so cache is highly effective.
	key := call + "|" + band + "|" + mode + "|" + date
	if m.dupeCacheKey == key {
		m.dupe = m.dupeCacheResult
		return
	}
	m.dupeCacheKey = key

	isDupe, existing := store.IsDuplicateQSO(m.App.DB, call, band, mode, date)
	if !isDupe || existing == nil {
		m.dupeCacheResult = false
		applog.Debug("dupe: checkDupe result", "key", key, "dupe", false)
		return
	}
	// If any reference field differs, it's not a dupe (e.g. different summit).
	formSOTA := strings.TrimSpace(m.fields[fieldSOTA].Value())
	formPOTA := strings.TrimSpace(m.fields[fieldPOTA].Value())
	formWWFF := strings.TrimSpace(m.fields[fieldWWFF].Value())
	formIOTA := strings.TrimSpace(m.fields[fieldIOTA].Value())
	if formSOTA != existing.SOTA || formPOTA != existing.POTA ||
		formWWFF != existing.WWFF || formIOTA != existing.IOTA {
		m.dupeCacheResult = false
		return
	}
	m.dupe = true
	m.dupeCacheResult = true
}

// clearForm resets the entire QSO form for a new QSO: clears fields (with
// retention of rig values and comment), partner lookup data, navigation state,
// and the recent-QSO filter table.  Partner data is preserved when the user is
// actively viewing the Partner/Image screen.
func (m *Model) clearForm() {
	m.dupe = false
	m.dupeCacheKey = ""
	if !m.retainForm {
		m.resetQSOFields()
	}
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
	if m.keepComment {
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
	m.keepFocused = false
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
		m.lookup.qrzLookupDone = false
		m.lookup.qrzLookupCall = ""
		m.lookup.wlPrivateData = nil
		m.lookup.wlLookupDone = false
		m.lookup.wlLookupCall = ""
		m.lookup.wlDispatchTime = time.Time{}
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
