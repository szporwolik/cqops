package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// Pre-allocated QSO form layout data — avoids per-frame allocations.
var (
	formLeft      = []field{fieldDate, fieldTime, fieldCall, fieldRSTSent, fieldRSTRcvd, fieldFreq, fieldBand, fieldExchSent, fieldExchRcvd}
	formMiddle    = []field{fieldMode, fieldSubmode, fieldName, fieldQTH, fieldGrid, fieldCountry, fieldSIG}
	formRight     = []field{fieldTXPower, fieldFreqRx, fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA, fieldSIGInfo}
	allFields     = buildAllFields()
	choiceIconStr = DimStyle.Render("\u25bc ")
	choiceIconW   = lipgloss.Width(choiceIconStr)

	// Pre-allocated badge styles — invariant, created once at init.
	dupeBadgeStyle = lipgloss.NewStyle().Foreground(P.Text).Background(P.Error).Bold(true).Padding(0, 1)
	newBadgeStyle  = lipgloss.NewStyle().Foreground(P.Text).Background(P.Success).Bold(true).Padding(0, 1)
)

func buildAllFields() []field {
	fs := make([]field, 0, fieldCount)
	for f := field(0); f < fieldCount; f++ {
		fs = append(fs, f)
	}
	return fs
}

// isChoiceField returns true for fields that have a cycle ▼ icon.
func isChoiceField(f field) bool { return f == fieldBand || f == fieldMode || f == fieldSubmode }

// isFieldHidden returns true when the given field should not be visible.
// Exchange fields are hidden when no contest is active.
func (m *Model) isFieldHidden(f field) bool {
	if (f == fieldExchSent || f == fieldExchRcvd) && m.App.Logbook.ActiveContest == "" {
		return true
	}
	return false
}

// viewForm renders the QSO entry form in a three-column layout.
// Columns are capped at maxColW so they don't spread absurdly on wide screens;
// the three-column block is left-aligned with a tight border.
func (m *Model) viewForm(width int) string {
	bodyW := width
	if bodyW < 20 {
		bodyW = 20
	}

	// Build a cache signature from all inputs that affect form output.
	// Exclude clock fields (date/time) when auto-updating. The 1-second TTL
	// is enforced via formSec comparison, not the cache key, so the cache
	// actually hits for ~59 out of 60 frames.
	var sigB strings.Builder
	fmt.Fprintf(&sigB, "%d|%d|%s|", width, m.focus, m.App.Logbook.ActiveContest)
	if m.keepFocused {
		sigB.WriteString("rf|")
		fmt.Fprintf(&sigB, "ks%d|", m.keepSubFocus)
	} else {
		fmt.Fprintf(&sigB, "cp%d|", m.fields[m.focus].Position())
	}
	if m.keepComment {
		sigB.WriteString("rc|")
	}
	if m.retainForm {
		sigB.WriteString("rt|")
	}
	if m.dateTimeAuto {
		sigB.WriteString("dta|")
	}
	for _, f := range allFields {
		if m.dateTimeAuto && (f == fieldDate || f == fieldTime) {
			continue
		}
		sigB.WriteString(m.fields[f].Value())
		sigB.WriteByte('|')
	}
	if !m.dateTimeAuto {
		sigB.WriteString(m.fields[fieldDate].Value())
		sigB.WriteByte('|')
		sigB.WriteString(m.fields[fieldTime].Value())
		sigB.WriteByte('|')
	}
	sig := sigB.String()
	// Cache hit: signature match AND we're in the same minute.
	// Minute-based TTL synchronizes with the system clock crossing :00
	// so the display is always in sync with real time.
	if m.rc.formSig == sig && m.rc.formView != "" && m.rc.formSec == time.Now().Minute() {
		return m.rc.formView
	}

	const maxColW = 41
	colW := bodyW / 3
	if colW > maxColW {
		colW = maxColW
	}
	if colW < 20 {
		colW = bodyW
	}

	// Cache column & comment styles per width.
	var colStyle, commentStyle lipgloss.Style
	if m.rc.formColW == colW {
		colStyle = m.rc.formColStyle
		commentStyle = m.rc.formCommentStyle
	} else {
		colStyle = lipgloss.NewStyle().Width(colW).MaxWidth(colW).Align(lipgloss.Left).Inline(true)
		commentW := colW * 2
		if commentW < 20 {
			commentW = bodyW
		}
		commentStyle = lipgloss.NewStyle().Width(commentW).MaxWidth(commentW).Align(lipgloss.Left).Inline(true)
		m.rc.formColW = colW
		m.rc.formColStyle = colStyle
		m.rc.formCommentStyle = commentStyle
	}

	// labelW is the fixed space: 2-char prefix + 11-char label.
	const labelW = 2 + 11

	// renderLine returns the raw field line (label + value) without column-width
	// wrapping. Textinput width is set locally for rendering but is not persisted
	// back to the model — width sync happens in Update() on WindowSizeMsg.
	// This avoids mutating model state during View() and busting the form cache.
	renderLine := func(f field, availW int) string {
		label := fieldNames[f]
		raw := strings.TrimSpace(m.fields[f].Value())
		isFocused := int(f) == int(m.focus) && !m.keepFocused
		ti := m.fields[f]

		choiceIcon := ""
		if isChoiceField(f) {
			choiceIcon = choiceIconStr
		}

		vw := availW - labelW - 1 - choiceIconW
		if vw < 3 {
			vw = 3
		}
		if vw > 40 {
			vw = 40
		}
		// Set width locally for correct View() output; don't persist back.
		if isFocused && lipgloss.Width(raw) > vw {
			vw = vw - 1
		}
		ti.SetWidth(vw)

		var v string
		if raw == "" && !isFocused {
			v = DimStyle.Render("\u2014")
		} else if isFocused {
			v = ti.View()
		} else if f == fieldCall {
			v = S.Info.Render(truncateText(raw, vw))
		} else if f == fieldFreq && raw != "" && !qso.IsInHamBand(parseFrequency(raw), m.App.Logbook.Station.IARURegion) {
			v = S.Error.Render(truncateText(raw, vw))
		} else {
			v = ValueStyle.Render(truncateText(raw, vw))
		}
		val := choiceIcon + v

		prefix := "  "
		lblStyled := S.FormLabel.Align(lipgloss.Left).Render(label)
		var lblPart string
		if isFocused {
			prefix = "> "
			lblStyled = fieldFocusedLabel.Align(lipgloss.Left).Render(label)
			lblPart = fieldFocusedPrefix.Render(prefix) + lblStyled
		} else {
			lblPart = fieldUnfocusedPrefix.Render(prefix) + lblStyled
		}
		return "    " + lipgloss.JoinHorizontal(lipgloss.Center, lblPart, " ", val)
	}

	// Count visible fields in each column so the form shrinks when exchange
	// fields are hidden (non-contest mode).
	visibleRows := len(formLeft)
	for _, f := range formLeft {
		if m.isFieldHidden(f) {
			visibleRows--
		}
	}
	if len(formMiddle) > visibleRows {
		visibleRows = len(formMiddle)
	}
	if len(formRight) > visibleRows {
		visibleRows = len(formRight)
	}

	var b strings.Builder

	for i := 0; i < visibleRows; i++ {
		var cols []string
		if i < len(formLeft) {
			f := formLeft[i]
			if m.isFieldHidden(f) {
				cols = append(cols, colStyle.Render(""))
			} else {
				cols = append(cols, colStyle.Render(renderLine(f, colW)))
			}
		} else {
			cols = append(cols, colStyle.Render(""))
		}
		if i < len(formMiddle) {
			cols = append(cols, colStyle.Render(renderLine(formMiddle[i], colW)))
		} else {
			cols = append(cols, colStyle.Render(""))
		}
		if i < len(formRight) {
			cols = append(cols, colStyle.Render(renderLine(formRight[i], colW)))
		} else {
			cols = append(cols, colStyle.Render(""))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
		if colW < 20 {
			row = lipgloss.JoinVertical(lipgloss.Left, cols...)
		}
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Comment row: Comment in left, Keep in middle, Retain in right — all one line.
	commentLine := colStyle.Render(renderLine(fieldComment, colW))
	keepBox := colStyle.Render(m.renderKeepCheckbox(colW))
	retainBox := colStyle.Render(m.renderRetainFormCheckbox(colW))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, commentLine, keepBox, retainBox))

	result := b.String()
	m.rc.formSig = sig
	m.rc.formView = result
	m.rc.formSec = time.Now().Minute()

	return result
}

// renderKeepCheckbox renders the "Keep Comment" checkbox next to the Comment field.
func (m *Model) renderKeepCheckbox(_ int) string {
	mark := "[ ]"
	label := "Keep Comment"
	if m.keepComment {
		mark = "[x]"
	}
	space := " "
	if m.keepFocused && m.keepSubFocus == 0 {
		return "     " + lipgloss.JoinHorizontal(lipgloss.Center,
			CursorStyle.Render(" "+mark),
			space,
			InputStyle.Render(label),
		)
	}
	if m.keepComment {
		return "     " + lipgloss.JoinHorizontal(lipgloss.Center,
			space,
			InputStyle.Render(mark),
			space,
			DimStyle.Render(label),
		)
	}
	return "     " + lipgloss.JoinHorizontal(lipgloss.Center,
		space,
		DimStyle.Render(mark),
		space,
		DimStyle.Render(label),
	)
}

// renderRetainFormCheckbox renders the "Hold Form" checkbox below the middle column.
// When checked, the form is NOT cleared after a QSO save — useful for logging
// the same contact across multiple logbooks (e.g. private → club station).
func (m *Model) renderRetainFormCheckbox(_ int) string {
	mark := "[ ]"
	label := "Hold Form"
	if m.retainForm {
		mark = "[x]"
	}
	space := " "
	focused := m.keepFocused && m.keepSubFocus == 1
	if focused {
		return "     " + lipgloss.JoinHorizontal(lipgloss.Center,
			CursorStyle.Render(" "+mark),
			space,
			InputStyle.Render(label),
		)
	}
	if m.retainForm {
		return "     " + lipgloss.JoinHorizontal(lipgloss.Center,
			space,
			InputStyle.Render(mark),
			space,
			DimStyle.Render(label),
		)
	}
	return "     " + lipgloss.JoinHorizontal(lipgloss.Center,
		space,
		DimStyle.Render(mark),
		space,
		DimStyle.Render(label),
	)
}

// stationProfile returns the station info parts for display in the path row.
func (m *Model) stationProfile() []string {
	s := m.App.Logbook.Station
	var parts []string
	if rig := s.RigModel(m.App.Config.Rigs); rig != "" {
		part := rig
		if ant := s.RigAntenna(m.App.Config.Rigs); ant != "" {
			part += " / " + ant
		}
		parts = append(parts, part)
	}
	grid := m.effectiveGrid()
	if grid != "" {
		label := "Grid " + formatLocator(grid)
		if m.isGPSGridActive() {
			label += " (GPS)"
		}
		parts = append(parts, label)
	}
	if s.Callsign != "" && len(parts) == 0 {
		parts = append(parts, s.Callsign)
	}
	return parts
}

// formPathRow renders the info line between the QSO form and recent QSOs table.
// Two states: no call → station profile (right-aligned); call entered → path (left-aligned)
// or fall back to station profile if no grids.
// Badges (DUPE!, New Call!, New DXCC!) are shown whenever a callsign is entered,
// regardless of whether grids are available.
func (m *Model) formPathRow(width int) string {
	s := m.App.Logbook.Station

	renderProfile := func(align lipgloss.Position) string {
		parts := m.stationProfile()
		if len(parts) == 0 {
			return ""
		}
		profileLine := strings.Join(parts, "  \u00b7  ")
		if lipgloss.Width(profileLine) > width {
			profileLine = truncateText(profileLine, width)
		}
		return pathMutedStyle.Width(width).Align(align).Render(profileLine)
	}

	// ── Rotor status line (always right-aligned when connected) ──
	rotorLine := ""
	if m.rotor.connected {
		rotorLine = "Rotator"
		if m.rotor.name != "" {
			rotorLine += " " + m.rotor.name
		}
		rotorLine += "  Az " + strconv.FormatFloat(m.rotor.azimuth, 'f', 0, 64) + "\u00b0"
		if m.rotor.targetAz != 0 && absDiff(m.rotor.azimuth, m.rotor.targetAz) >= 1 {
			arrow := "\u2192"
			if m.rotor.targetAz < m.rotor.azimuth {
				arrow = "\u2190"
			}
			rotorLine += S.Warning.Render(" (" + arrow + " " + strconv.FormatFloat(m.rotor.targetAz, 'f', 0, 64) + "\u00b0)")
		}
		rotorLine += "  El " + strconv.FormatFloat(m.rotor.elevation, 'f', 0, 64) + "\u00b0"
		if m.rotor.targetEl != 0 && absDiff(m.rotor.elevation, m.rotor.targetEl) >= 1 {
			arrow := "\u2191"
			if m.rotor.targetEl < m.rotor.elevation {
				arrow = "\u2193"
			}
			rotorLine += S.Warning.Render(" (" + arrow + " " + strconv.FormatFloat(m.rotor.targetEl, 'f', 0, 64) + "\u00b0)")
		}
	}

	// ── No callsign: DXC snapshot or station profile ──
	if m.rc.pathCall == "" || strings.TrimSpace(m.fields[fieldCall].Value()) == "" {
		m.rc.pathCall = ""
		m.rc.pathSig = ""
		if rotorLine != "" {
			return pathInfoStyle.Width(width).Align(lipgloss.Right).Render(rotorLine)
		}
		// When DXC is connected and we have a frequency, show nearby spots.
		if m.dxc.online && m.App.Config.Integrations.DXC.Enabled {
			if line := m.dxcPathLine(width); line != "" {
				return line
			}
		}
		return renderProfile(lipgloss.Right)
	}

	// ── Callsign entered: load stats, compute badges, build path ──
	call := strings.TrimSpace(m.fields[fieldCall].Value())
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := strings.TrimSpace(m.fields[fieldMode].Value())
	statsSig := call + "|" + band + "|" + mode
	if m.rc.logStatsSig != statsSig && m.App.DB != nil {
		stats, err := store.GetLogbookStats(m.App.DB, call, band, mode)
		if err == nil {
			m.rc.logStats = stats
			m.rc.logStatsSig = statsSig
		}
	}

	// Compute badges — always evaluated when a call is present.
	var showNewCall bool
	if m.lookup.wlPrivateData != nil {
		showNewCall = !m.lookup.wlPrivateData.Worked()
	} else {
		showNewCall = !m.rc.logStats.CallWorked
	}
	wlNewDXCC := m.lookup.wlPrivateData != nil && !m.lookup.wlPrivateData.DXCCConfirmed()

	// Build the primary info line: path (grids) or profile fallback.
	ownGrid := formatLocator(s.Grid)
	rawGrid := strings.TrimSpace(m.fields[fieldGrid].Value())
	partnerGrid := ""
	if rawGrid != "" && qso.IsValidLocator(rawGrid) {
		partnerGrid = formatLocator(rawGrid)
	}

	var primaryLine string
	if ownGrid != "" && partnerGrid != "" {
		line := distanceLine(ownGrid, partnerGrid, m.App.Config.General.Units)
		if line != "" {
			primaryLine = " Path  " + line
		}
	}
	// When a callsign is entered but grids are unavailable, only show
	// badges (DUPE!, New Call!, New DXCC!) — do NOT fall back to the
	// station profile. The profile is shown only when no call is entered.

	// Build badge line (DUPE!, New Call!, New DXCC!).
	const bannerNewCall = "New Call!"
	const bannerNewDXCC = "New DXCC!"
	const bannerDupe = "DUPE!"

	dupeStyle := dupeBadgeStyle
	newStyle := newBadgeStyle

	var badges []string
	if m.dupe {
		badges = append(badges, dupeStyle.Render(bannerDupe))
	}
	if showNewCall {
		badges = append(badges, newStyle.Render(bannerNewCall))
	}
	if wlNewDXCC {
		badges = append(badges, newStyle.Render(bannerNewDXCC))
	}

	// Build cache key: grids + distance + stats + WL + dupe.
	var wlSigB strings.Builder
	wlSigB.WriteString("WL:")
	if m.lookup.wlPrivateData != nil {
		wlSigB.WriteString("wk=")
		wlSigB.WriteString(strconv.FormatBool(m.lookup.wlPrivateData.Worked()))
		wlSigB.WriteString(",dxcc=")
		wlSigB.WriteString(strconv.FormatBool(m.lookup.wlPrivateData.DXCCConfirmed()))
	}
	wlSig := wlSigB.String()
	var sigB strings.Builder
	sigB.WriteString(ownGrid)
	sigB.WriteByte('|')
	sigB.WriteString(partnerGrid)
	sigB.WriteByte('|')
	sigB.WriteString(m.App.Config.General.Units)
	sigB.WriteByte('|')
	sigB.WriteString(statsSig)
	sigB.WriteByte('|')
	sigB.WriteString(wlSig)
	sigB.WriteByte('|')
	sigB.WriteString(strconv.FormatBool(m.dupe))
	// Include rotor state so target arrows update immediately.
	if m.rotor.connected {
		sigB.WriteString("|rotor:")
		sigB.WriteString(strconv.FormatFloat(m.rotor.azimuth, 'f', 0, 64))
		sigB.WriteByte(',')
		sigB.WriteString(strconv.FormatFloat(m.rotor.elevation, 'f', 0, 64))
		sigB.WriteByte(',')
		sigB.WriteString(strconv.FormatFloat(m.rotor.targetAz, 'f', 0, 64))
		sigB.WriteByte(',')
		sigB.WriteString(strconv.FormatFloat(m.rotor.targetEl, 'f', 0, 64))
	}
	sig := sigB.String()

	if m.rc.pathSig == sig && m.rc.pathLine != "" {
		return m.rc.pathLine
	}

	// Assemble result: left side + right-side rotor status.
	var left string
	var result string
	if primaryLine != "" {
		left = primaryLine
	}
	if len(badges) > 0 {
		badgeLine := strings.Join(badges, " ")
		if left != "" {
			left = left + "  " + badgeLine
		} else {
			left = badgeLine
		}
	}

	if rotorLine != "" {
		if left != "" {
			// Both sides: left + spacer + rotor right-aligned.
			rotorW := lipgloss.Width(rotorLine)
			leftW := width - rotorW - 2 // 2-char gap
			if leftW < 10 {
				leftW = width
				rotorLine = ""
			}
			if rotorLine != "" {
				left = pathInfoStyle.Width(leftW).Align(lipgloss.Left).Render(left)
				right := pathInfoStyle.Render(rotorLine)
				result = lipgloss.JoinHorizontal(lipgloss.Center, left, "  ", right)
			}
		} else {
			result = pathInfoStyle.Width(width).Align(lipgloss.Right).Render(rotorLine)
		}
	} else if left != "" {
		// Path shown, no rotor — fill empty right side with station
		// profile (radio/antenna · grid) right-aligned.
		profile := strings.Join(m.stationProfile(), "  \u00b7  ")
		if profile != "" {
			profileW := lipgloss.Width(profile)
			leftW := width - profileW - 2
			if leftW >= 20 {
				leftStyled := pathInfoStyle.Width(leftW).Align(lipgloss.Left).Render(left)
				rightStyled := pathMutedStyle.Render(profile)
				result = lipgloss.JoinHorizontal(lipgloss.Center, leftStyled, "  ", rightStyled)
			} else {
				result = left
			}
		} else {
			result = left
		}
	} else {
		result = left
	}

	if result == "" {
		m.rc.pathSig = sig
		m.rc.pathLine = ""
		return ""
	}

	// Truncate if too wide.
	if lipgloss.Width(result) > width {
		// Try without badges first (preserve primary line).
		if primaryLine != "" && lipgloss.Width(primaryLine) <= width {
			result = primaryLine
			if rotorLine != "" {
				rotorW := lipgloss.Width(rotorLine)
				leftW := width - rotorW - 2
				if leftW >= 10 {
					left = pathInfoStyle.Width(leftW).Align(lipgloss.Left).Render(primaryLine)
					right := pathInfoStyle.Render(rotorLine)
					result = lipgloss.JoinHorizontal(lipgloss.Center, left, "  ", right)
				} else {
					result = primaryLine
				}
			}
		} else {
			result = truncateText(result, width)
		}
	}

	st := pathInfoStyle.Width(width).Align(lipgloss.Left).Render(result)
	m.rc.pathSig = sig
	m.rc.pathLine = st
	return st
}

// dxcPathLine returns a line showing nearby DXC spots around the current
// frequency. Displays up to N spots below and N above, with frequencies.
// N adapts to available width. Cached until frequency or spot list changes.
func (m *Model) dxcPathLine(width int) string {
	freqStr := strings.TrimSpace(m.fields[fieldFreq].Value())
	if freqStr == "" {
		return ""
	}
	freqKhz, err := strconv.ParseFloat(freqStr, 64)
	if err != nil || freqKhz <= 0 {
		return ""
	}
	// freqKhz is in MHz from the form; convert to kHz for comparison.
	curKhz := freqKhz * 1000

	// Build cache signature: frequency + spot count + width + rig identity
	// + continent + mode (these affect the smart filter).
	modeCat := spotModeCategory(strings.TrimSpace(m.fields[fieldMode].Value()))
	stationCont := m.App.Logbook.Station.Continent
	var sigB strings.Builder
	fmt.Fprintf(&sigB, "%.3f|%d|%d|%s|%s|%s|%s|%s|%s", freqKhz, m.dxc.rawGen, width,
		m.App.Logbook.Station.RigName, m.App.Logbook.Station.RigPower(m.App.Config.Rigs),
		m.App.LogbookName, m.App.Logbook.ActiveContest, stationCont, modeCat)
	sig := sigB.String()
	if m.rc.dxcPathSig == sig && m.rc.dxcPathLine != "" {
		return m.rc.dxcPathLine
	}

	// Collect spots on the current band, sorted by frequency.
	band := qso.DeriveBand(freqKhz)
	if band == "" {
		return ""
	}
	lo, hi, ok := qso.BandRange(band)
	if !ok {
		return ""
	}
	var spots []store.DXCSpot
	for _, s := range m.dxc.cachedRaw {
		if s.Frequency >= lo*1000 && s.Frequency <= hi*1000 && s.Frequency > 0 {
			spots = append(spots, s)
		}
	}
	// DB fallback: cachedRaw may be empty on startup before the first spots
	// arrive. Use the band+time-filtered query (idx_dxc_spots_band_time)
	// so SQLite does the heavy lifting — returns only recent spots on the
	// current band, already sorted by frequency.
	if len(spots) == 0 && m.App.DB != nil {
		dbSpots, err := store.QueryDXCSpotsByBand(m.App.DB, band, 900)
		if err == nil {
			spots = dbSpots
		}
	}
	// ── Smart filtering ────────────────────────────────────────────────
	// Default behaviour: show spots from the same continent, same mode
	// category (DIGI/PHONE/CW), and no older than 15 minutes. Filters
	// are applied with fallback: if filtering removes everything we
	// relax them one by one instead of showing an empty line.
	now := time.Now().UTC().Unix()
	applyFilters := func(spots []store.DXCSpot, cont, mc string, maxAgeSec int64) []store.DXCSpot {
		filtered := make([]store.DXCSpot, 0, len(spots))
		for _, s := range spots {
			if cont != "" && s.SpotCont != "" && s.SpotCont != cont {
				continue
			}
			if mc != "" && s.ModeCat != "" && s.ModeCat != mc {
				continue
			}
			if maxAgeSec > 0 && s.ReceivedAt > 0 && now-s.ReceivedAt > maxAgeSec {
				continue
			}
			filtered = append(filtered, s)
		}
		return filtered
	}
	// Try full filter (continent + mode + time).
	filtered := applyFilters(spots, stationCont, modeCat, 900)
	if len(filtered) == 0 {
		// Fallback 1: drop continent, keep mode + time.
		filtered = applyFilters(spots, "", modeCat, 900)
	}
	if len(filtered) == 0 {
		// Fallback 2: drop mode, keep time only.
		filtered = applyFilters(spots, "", "", 900)
	}
	if len(filtered) > 0 {
		spots = filtered
	}

	if len(spots) == 0 {
		m.rc.dxcPathSig = sig
		m.rc.dxcPathLine = ""
		return ""
	}

	// Sort by frequency ascending.
	sort.Slice(spots, func(i, j int) bool { return spots[i].Frequency < spots[j].Frequency })

	// Find the insertion point for curKhz.
	idx := sort.Search(len(spots), func(i int) bool { return spots[i].Frequency >= curKhz })

	// Determine how many to show based on width. Each spot takes ~18-22 chars.
	maxEach := 3
	if width < 80 {
		maxEach = 1
	} else if width < 120 {
		maxEach = 2
	}

	// Collect up to maxEach below and above, excluding spots that match
	// our frequency (within ±0.5 kHz). Matched spots go in the center.
	const matchTol = 0.5 // kHz tolerance for "on frequency"
	var matched []store.DXCSpot
	var below, above []store.DXCSpot
	for i := idx - 1; i >= 0 && len(below)+len(matched) < maxEach*2; i-- {
		s := spots[i]
		if math.Abs(s.Frequency-curKhz) <= matchTol {
			matched = append(matched, s)
		} else if len(below) < maxEach {
			below = append(below, s)
		}
	}
	for i := idx; i < len(spots) && len(above)+len(matched) < maxEach*2; i++ {
		s := spots[i]
		if math.Abs(s.Frequency-curKhz) <= matchTol {
			matched = append(matched, s)
		} else if len(above) < maxEach {
			above = append(above, s)
		}
	}
	// Reverse below so closest is last (display order: farthest…closest | center | closest…farthest).
	for i, j := 0, len(below)-1; i < j; i, j = i+1, j-1 {
		below[i], below[j] = below[j], below[i]
	}

	// Build dupe set for the current band (reuses DXCDupeSet which is
	// shared with the DXC table — single query covers both). In contest
	// mode the check spans the entire contest; outside contests we use
	// today's date. Only query when the screen is wide enough to absorb
	// the "(D)" markers gracefully.
	var dupeSet map[string]bool
	if width >= 100 && m.App.DB != nil {
		dateStr := time.Now().UTC().Format("20060102")
		if ds, err := store.DXCDupeSet(m.App.DB, dateStr, m.App.Logbook.ActiveContest); err == nil {
			dupeSet = ds
		}
	}

	// Format: "CALL FREQ  CALL FREQ … │ FREQ ││ CALLS ││ … CALL FREQ  CALL FREQ"
	// Dupes get a "D " prefix (matching the DXC table convention) so
	// they stand out even on monochrome terminals.
	formatSpot := func(s store.DXCSpot) string {
		prefix := ""
		if dupeSet != nil {
			key := qso.NormalizeCall(s.DXCall) + "|" + qso.NormalizeBand(s.Band) + "|" + qso.NormalizeRigMode(s.Mode)
			if dupeSet[key] {
				prefix = "D "
			}
		}
		return fmt.Sprintf("%s%s %s", prefix, s.DXCall, formatFreqCompact(s.Frequency))
	}
	belowParts := make([]string, len(below))
	for i, s := range below {
		belowParts[i] = formatSpot(s)
	}
	aboveParts := make([]string, len(above))
	for i, s := range above {
		aboveParts[i] = formatSpot(s)
	}

	var center string
	if len(matched) > 0 {
		// On-frequency match: show callsign + freq between double bars.
		matchParts := make([]string, len(matched))
		for i, s := range matched {
			matchParts[i] = formatSpot(s)
		}
		center = fmt.Sprintf("││ %s ││", strings.Join(matchParts, "  "))
	} else {
		center = fmt.Sprintf("│ %s │", formatFreqCompact(curKhz))
	}

	// Update cycling state for Ctrl+P. Reset index when spots change.
	if !dxcSpotsEqual(m.dxc.pathSpots, matched) {
		m.dxc.pathSpots = matched
		m.dxc.pathSpotIdx = -1
	}
	left := strings.Join(belowParts, "  ")
	right := strings.Join(aboveParts, "  ")

	// Right-align left side, left-align right side.
	leftStyled := pathMutedStyle.Align(lipgloss.Right).Render(left)
	rightStyled := pathMutedStyle.Align(lipgloss.Left).Render(right)
	dxcLine := lipgloss.JoinHorizontal(lipgloss.Center, leftStyled, " ", center, " ", rightStyled)

	// If too wide, reduce to 1 per side.
	if lipgloss.Width(dxcLine) > width && maxEach > 1 {
		maxEach = 1
		below = below[:min(len(below), 1)]
		above = above[:min(len(above), 1)]
		belowParts = make([]string, len(below))
		for i, s := range below {
			belowParts[i] = formatSpot(s)
		}
		aboveParts = make([]string, len(above))
		for i, s := range above {
			aboveParts[i] = formatSpot(s)
		}
		left = strings.Join(belowParts, "  ")
		right = strings.Join(aboveParts, "  ")
		leftStyled = pathMutedStyle.Align(lipgloss.Right).Render(left)
		rightStyled = pathMutedStyle.Align(lipgloss.Left).Render(right)
		dxcLine = lipgloss.JoinHorizontal(lipgloss.Center, leftStyled, " ", center, " ", rightStyled)
	}

	// Append rig info right-aligned when space permits.
	profileParts := m.stationProfile()
	if len(profileParts) > 0 {
		rigInfo := strings.Join(profileParts, "  ·  ")
		rigW := lipgloss.Width(rigInfo)
		dxcW := lipgloss.Width(dxcLine)
		gap := 3
		availForDXC := width - rigW - gap
		if availForDXC >= 40 && dxcW > availForDXC {
			dxcLine = truncateText(dxcLine, availForDXC)
			dxcW = lipgloss.Width(dxcLine)
		}
		if dxcW+gap+rigW <= width {
			spacer := strings.Repeat(" ", width-dxcW-rigW)
			rigStyled := pathMutedStyle.Render(rigInfo)
			dxcLine = lipgloss.JoinHorizontal(lipgloss.Center,
				dxcLine,
				spacer,
				rigStyled,
			)
		}
	}

	if lipgloss.Width(dxcLine) > width {
		dxcLine = truncateText(dxcLine, width)
	}

	m.rc.dxcPathSig = sig
	m.rc.dxcPathLine = dxcLine
	return dxcLine
}

// formatFreqCompact formats a frequency in kHz to a compact display string.
// Examples: 7123.4 → "7123.4", 14200.0 → "14200.0".
func formatFreqCompact(khz float64) string {
	return fmt.Sprintf("%.1f", khz)
}

// dxcSpotsEqual compares two DXC spot slices for equality (callsign + frequency).
func dxcSpotsEqual(a, b []store.DXCSpot) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].DXCall != b[i].DXCall || a[i].Frequency != b[i].Frequency {
			return false
		}
	}
	return true
}

// parseFrequency parses a frequency string (MHz) and returns the float value.
// Returns 0 on parse failure.
func parseFrequency(s string) float64 {
	var f float64
	fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}
