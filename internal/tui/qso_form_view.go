package tui

import (
	"fmt"
	"strings"

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
	// The date/time fields change every second, so this invalidates at 1 Hz.
	var sigB strings.Builder
	fmt.Fprintf(&sigB, "%d|%d|%s|", width, m.focus, m.App.Logbook.ActiveContest)
	if m.retainFocused {
		sigB.WriteString("rf|")
	} else {
		// Cursor position affects View() output — must be part of the key.
		fmt.Fprintf(&sigB, "cp%d|", m.fields[m.focus].Position())
	}
	if m.retainComment {
		sigB.WriteString("rc|")
	}
	if m.dateTimeAuto {
		sigB.WriteString("dta|")
	}
	for _, f := range allFields {
		sigB.WriteString(m.fields[f].Value())
		sigB.WriteByte('|')
	}
	sig := sigB.String()
	if m.rc.formSig == sig && m.rc.formView != "" {
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
	// wrapping. Textinput width is set here because bubbles/textinput requires
	// width to be known at render time for cursor positioning.
	renderLine := func(f field, availW int) string {
		label := fieldNames[f]
		raw := strings.TrimSpace(m.fields[f].Value())
		isFocused := int(f) == int(m.focus) && !m.retainFocused
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
		ti.SetWidth(vw)
		if isFocused {
			if lipgloss.Width(raw) > vw {
				ti.SetWidth(vw - 1)
			}
			ti.SetCursor(ti.Position())
			m.fields[f] = ti
		}

		var v string
		if raw == "" && !isFocused {
			v = DimStyle.Render("\u2014")
		} else if isFocused {
			v = ti.View()
		} else if f == fieldCall {
			v = S.Info.Render(truncateText(raw, vw))
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
		return lipgloss.JoinHorizontal(lipgloss.Center, lblPart, " ", val)
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

	// Comment row spans first two columns; Retain checkbox in third.
	commentLine := commentStyle.Render(renderLine(fieldComment, colW*2))
	retainBox := colStyle.Render(m.renderRetainCheckbox(colW))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, commentLine, retainBox))

	result := b.String()
	m.rc.formSig = sig
	m.rc.formView = result

	return result
}

// renderRetainCheckbox renders the "Retain" checkbox next to the Comment field.
func (m *Model) renderRetainCheckbox(_ int) string {
	mark := "[ ]"
	label := "Retain"
	if m.retainComment {
		mark = "[x]"
	}
	space := " "
	if m.retainFocused {
		return lipgloss.JoinHorizontal(lipgloss.Center,
			CursorStyle.Render(" "+mark),
			space,
			InputStyle.Render(label),
		)
	}
	if m.retainComment {
		return lipgloss.JoinHorizontal(lipgloss.Center,
			space,
			InputStyle.Render(mark),
			space,
			DimStyle.Render(label),
		)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center,
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
		part := "Rig "
		if rp, ok := m.App.Config.Rigs[s.RigName]; ok && rp.Name != "" {
			part += rp.Name + " - "
		}
		part += rig
		if ant := s.RigAntenna(m.App.Config.Rigs); ant != "" {
			part += "/" + ant
		}
		parts = append(parts, part)
	}
	if s.Grid != "" {
		parts = append(parts, "Grid "+formatLocator(s.Grid))
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
		rotorLine += fmt.Sprintf("  Az %.0f\u00b0", m.rotor.azimuth)
		if m.rotor.targetAz != 0 && absDiff(m.rotor.azimuth, m.rotor.targetAz) >= 1 {
			arrow := "\u2192"
			if m.rotor.targetAz < m.rotor.azimuth {
				arrow = "\u2190"
			}
			rotorLine += S.Warning.Render(fmt.Sprintf(" (%s %.0f\u00b0)", arrow, m.rotor.targetAz))
		}
		rotorLine += fmt.Sprintf("  El %.0f\u00b0", m.rotor.elevation)
		if m.rotor.targetEl != 0 && absDiff(m.rotor.elevation, m.rotor.targetEl) >= 1 {
			arrow := "\u2191"
			if m.rotor.targetEl < m.rotor.elevation {
				arrow = "\u2193"
			}
			rotorLine += S.Warning.Render(fmt.Sprintf(" (%s %.0f\u00b0)", arrow, m.rotor.targetEl))
		}
	}

	// ── No callsign: station profile (or just rotor) ──
	if m.rc.pathCall == "" || strings.TrimSpace(m.fields[fieldCall].Value()) == "" {
		m.rc.pathCall = ""
		m.rc.pathSig = ""
		if rotorLine != "" {
			return pathInfoStyle.Width(width).Align(lipgloss.Right).Render(rotorLine)
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
		line := distanceLine(ownGrid, partnerGrid, m.App.Config.General.DistanceUnit)
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

	dupeStyle := lipgloss.NewStyle().Foreground(P.Text).Background(P.Error).Bold(true).Padding(0, 1)
	newStyle := lipgloss.NewStyle().Foreground(P.Text).Background(P.Success).Bold(true).Padding(0, 1)

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
	wlSig := "WL:"
	if m.lookup.wlPrivateData != nil {
		wlSig += fmt.Sprintf("wk=%v,dxcc=%v", m.lookup.wlPrivateData.Worked(), m.lookup.wlPrivateData.DXCCConfirmed())
	}
	sig := ownGrid + "|" + partnerGrid + "|" + m.App.Config.General.DistanceUnit + "|" + statsSig + "|" + wlSig + "|" + fmt.Sprint(m.dupe)
	// Include rotor state so target arrows update immediately.
	if m.rotor.connected {
		sig += fmt.Sprintf("|rotor:%.0f,%.0f,%.0f,%.0f", m.rotor.azimuth, m.rotor.elevation, m.rotor.targetAz, m.rotor.targetEl)
	}

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
