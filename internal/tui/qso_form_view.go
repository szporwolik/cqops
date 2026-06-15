package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// Pre-allocated QSO form layout data — avoids per-frame allocations.
var (
	formLeft      = []field{fieldDate, fieldTime, fieldCall, fieldFreq, fieldBand, fieldMode, fieldSubmode}
	formMiddle    = []field{fieldRSTSent, fieldRSTRcvd, fieldName, fieldQTH, fieldGrid, fieldCountry}
	formRight     = []field{fieldTXPower, fieldFreqRx, fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA}
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
	fmt.Fprintf(&sigB, "%d|%d|", width, m.focus)
	if m.retainFocused {
		sigB.WriteString("rf|")
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
	if m.cachedFormSig == sig && m.cachedFormView != "" {
		return m.cachedFormView
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
	if m.cachedFormColW == colW {
		colStyle = m.cachedFormColStyle
		commentStyle = m.cachedFormCommentStyle
	} else {
		colStyle = lipgloss.NewStyle().Width(colW).MaxWidth(colW).Align(lipgloss.Left)
		commentW := colW * 2
		if commentW < 20 {
			commentW = bodyW
		}
		commentStyle = lipgloss.NewStyle().Width(commentW).MaxWidth(commentW).Align(lipgloss.Left)
		m.cachedFormColW = colW
		m.cachedFormColStyle = colStyle
		m.cachedFormCommentStyle = commentStyle
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
			prefix = CursorStyle.Render("> ")
			lblStyled = fieldFocusedLabel.Align(lipgloss.Left).Render(label)
			lblPart = fieldFocusedPrefix.Render(prefix) + lblStyled
		} else {
			lblPart = fieldUnfocusedPrefix.Render(prefix) + lblStyled
		}
		return lipgloss.JoinHorizontal(lipgloss.Center, lblPart, " ", val)
	}

	var b strings.Builder

	rows := len(formLeft)
	if len(formMiddle) > rows {
		rows = len(formMiddle)
	}
	if len(formRight) > rows {
		rows = len(formRight)
	}
	for i := 0; i < rows; i++ {
		var cols []string
		if i < len(formLeft) {
			cols = append(cols, colStyle.Render(renderLine(formLeft[i], colW)))
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
	m.cachedFormSig = sig
	m.cachedFormView = result
	return result
}

// renderRetainCheckbox renders the "Retain" checkbox next to the Comment field.
func (m *Model) renderRetainCheckbox(colW int) string {
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
	if s.Operator != "" {
		parts = append(parts, "Op "+s.Operator)
	}
	if rig := s.RigModel(m.App.Config.Rigs); rig != "" {
		part := "Rig " + rig
		if ant := s.RigAntenna(m.App.Config.Rigs); ant != "" {
			part += "/" + ant
		}
		parts = append(parts, part)
	}
	if s.Grid != "" {
		parts = append(parts, "Grid "+formatLocator(s.Grid))
	}
	if wl := m.App.Logbook.Wavelog; wl != nil && wl.Enabled {
		name := config.LogbookDisplayName(m.App.Logbook)
		if name != "" {
			parts = append(parts, "WL "+name)
		}
	}
	if s.Callsign != "" && len(parts) == 0 {
		parts = append(parts, s.Callsign)
	}
	return parts
}

// formPathRow renders the info line between the QSO form and recent QSOs table.
// Two states: no call → station profile (right-aligned); call entered → path (left-aligned)
// or fall back to station profile if no grids.
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

	// ── No callsign: station profile ──
	// Check both the committed pathCall and the current form field —
	// the user may have backspaced the call without leaving the field.
	if m.pathCall == "" || strings.TrimSpace(m.fields[fieldCall].Value()) == "" {
		m.pathCall = ""
		m.cachedPathSig = ""
		return renderProfile(lipgloss.Right)
	}

	// ── Callsign entered: try path, fall back to profile ──
	ownGrid := formatLocator(s.Grid)
	rawGrid := strings.TrimSpace(m.fields[fieldGrid].Value())
	partnerGrid := ""
	if rawGrid != "" && qso.IsValidLocator(rawGrid) {
		partnerGrid = formatLocator(rawGrid)
	}

	if ownGrid != "" && partnerGrid != "" {
		// Load logbook stats if needed — used for "New Call!" indicator.
		call := strings.TrimSpace(m.fields[fieldCall].Value())
		band := strings.TrimSpace(m.fields[fieldBand].Value())
		mode := strings.TrimSpace(m.fields[fieldMode].Value())
		statsSig := call + "|" + band + "|" + mode
		if m.cachedLogStatsSig != statsSig && m.App.DB != nil {
			stats, err := store.GetLogbookStats(m.App.DB, call, band, mode)
			if err == nil {
				m.cachedLogStats = stats
				m.cachedLogStatsSig = statsSig
			}
		}

		// Build cache key: grids + distance + log stats + WL state.
		wlSig := "WL:"
		if m.wlPrivateData != nil {
			wlSig += fmt.Sprintf("wk=%v,dxcc=%v", m.wlPrivateData.Worked(), m.wlPrivateData.DXCCConfirmed())
		}
		sig := ownGrid + "|" + partnerGrid + "|" + m.App.Config.General.DistanceUnit + "|" + statsSig + "|" + wlSig
		if m.cachedPathSig == sig && m.cachedPathLine != "" {
			return m.cachedPathLine
		}
		line := distanceLine(ownGrid, partnerGrid, m.App.Config.General.DistanceUnit)
		if line != "" {
			line = " Path  " + line
			// Show New Call / New DXCC indicators.
			// WL-first: if WL has data it wins; otherwise fall back to local.
			var showNewCall bool
			if m.wlPrivateData != nil {
				showNewCall = !m.wlPrivateData.Worked()
			} else {
				showNewCall = !m.cachedLogStats.CallWorked
			}
			wlNewDXCC := m.wlPrivateData != nil && !m.wlPrivateData.DXCCConfirmed()

			// Build banners as plain text so truncation is ANSI-safe.
			const bannerNewCall = "New Call!"
			const bannerNewDXCC = "New DXCC!"
			var bannerPlain string
			if showNewCall {
				bannerPlain += "  " + bannerNewCall
			}
			if wlNewDXCC {
				bannerPlain += "  " + bannerNewDXCC
			}

			// Determine final display text: path line + optional banners.
			// Build plain text first, truncate if needed, then style banners.
			displayText := line
			if bannerPlain != "" {
				candidate := line + bannerPlain
				if lipgloss.Width(candidate) <= width {
					displayText = candidate
				}
				// else: banners dropped, displayText stays as line alone.
			}
			// If line alone is too wide, truncate it.
			if lipgloss.Width(displayText) > width {
				displayText = truncateText(displayText, width)
			}

			result := pathInfoStyle.Width(width).Align(lipgloss.Left).Render(displayText)
			// Re-apply banner styling after Lip Gloss width clamping.
			if showNewCall && strings.Contains(result, bannerNewCall) {
				result = strings.Replace(result, bannerNewCall, S.Success.Render(bannerNewCall), 1)
			}
			if wlNewDXCC && strings.Contains(result, bannerNewDXCC) {
				result = strings.Replace(result, bannerNewDXCC, S.Success.Render(bannerNewDXCC), 1)
			}
			m.cachedPathSig = sig
			m.cachedPathLine = result
			return result
		}
		m.cachedPathSig = ""
		m.cachedPathLine = ""
	}

	return renderProfile(lipgloss.Right)
}
