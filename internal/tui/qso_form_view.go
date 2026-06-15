package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// Pre-allocated QSO form layout data — avoids per-frame allocations.
var (
	choiceFields  = map[field]bool{fieldBand: true, fieldMode: true, fieldSubmode: true}
	formLeft      = []field{fieldDate, fieldTime, fieldCall, fieldFreq, fieldBand, fieldMode, fieldSubmode}
	formMiddle    = []field{fieldRSTSent, fieldRSTRcvd, fieldName, fieldQTH, fieldGrid, fieldCountry}
	formRight     = []field{fieldTXPower, fieldFreqRx, fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA}
	choiceIconStr = DimStyle.Render("\u25bc ")
	choiceIconW   = lipgloss.Width(choiceIconStr)
)

// viewForm renders the QSO entry form in a three-column layout.
// Columns are capped at maxColW so they don't spread absurdly on wide screens;
// the three-column block is centered within the available width.
// width is the exact available space inside the border.
func (m *Model) viewForm(width int) string {
	bodyW := width
	if bodyW < 20 {
		bodyW = 20
	}

	// Cap column width so fields don't stretch on huge screens.
	const maxColW = 33
	colW := bodyW / 3
	if colW > maxColW {
		colW = maxColW
	}
	if colW < 20 {
		colW = bodyW // fallback to single column on very narrow terminals
	}

	// Cache column styles per width — avoids per-frame allocations.
	var colStyle lipgloss.Style
	if m.cachedFormColW == colW {
		colStyle = m.cachedFormColStyle
	} else {
		colStyle = lipgloss.NewStyle().Width(colW).MaxWidth(colW).Align(lipgloss.Left)
		m.cachedFormColW = colW
		m.cachedFormColStyle = colStyle
	}

	// No centering — form is left-aligned, border wraps content tightly.
	padStr := ""

	// labelW is the fixed space for the label part: 2-char prefix + 11-char label
	// (matches S.FormLabel / S.FormFocused Width(11)).
	labelW := 2 + 11

	// renderLine returns the raw field line (label + value) without column-width
	// wrapping. Callers apply the appropriate width style.
	renderLine := func(f field, availW int) string {
		label := fieldNames[f]
		raw := strings.TrimSpace(m.fields[f].Value())
		isFocused := int(f) == int(m.focus) && !m.retainFocused
		ti := m.fields[f]

		choiceIcon := ""
		if choiceFields[f] {
			choiceIcon = choiceIconStr
		}

		// Value width: whatever remains after label, spacer, and choice icon.
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
		b.WriteString(padStr)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Comment row spans first two columns; Retain checkbox in third column.
	commentW := colW * 2
	if commentW < 20 {
		commentW = bodyW
	}
	commentStyle := lipgloss.NewStyle().Width(commentW).MaxWidth(commentW).Align(lipgloss.Left)
	commentLine := commentStyle.Render(renderLine(fieldComment, commentW))

	retainBox := colStyle.Render(m.renderRetainCheckbox(colW))
	b.WriteString(padStr)
	if colW >= 20 {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, commentLine, retainBox))
	} else {
		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, commentLine, retainBox))
	}

	return b.String()
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
// When no callsign is entered, it shows station profile info (right-aligned).
// When a callsign is entered, it shows path distance and lookup results (left-aligned).
func (m *Model) formPathRow(width int) string {
	s := m.App.Logbook.Station
	partnerCall := strings.TrimSpace(m.fields[fieldCall].Value())

	renderParts := func(parts []string, style lipgloss.Style, align lipgloss.Position) string {
		if len(parts) == 0 {
			return ""
		}
		line := strings.Join(parts, "  \u00b7  ")
		if lipgloss.Width(line) > width {
			line = truncate(line, width)
		}
		return style.Width(width).Align(align).Render(line)
	}

	// ── No callsign: station profile, right-aligned ──
	if partnerCall == "" {
		return renderParts(m.stationProfile(), pathMutedStyle, lipgloss.Right)
	}

	// ── Callsign entered: path & lookup info ──
	ownGrid := formatLocator(s.Grid)
	partnerGrid := formatLocator(strings.TrimSpace(m.fields[fieldGrid].Value()))

	if ownGrid != "" && partnerGrid != "" {
		sig := ownGrid + "|" + partnerGrid + "|" + m.App.Config.General.DistanceUnit
		if m.cachedPathSig == sig && m.cachedPathLine != "" {
			return m.cachedPathLine
		}
		line := distanceLine(ownGrid, partnerGrid, m.App.Config.General.DistanceUnit)
		if line != "" {
			line = "Path  " + line
			if m.wlPrivateData != nil {
				sep := DimStyle.Render("  \u00b7  ")
				if !m.wlPrivateData.Worked() {
					line += sep + S.Warning.Render("New Call!")
				}
				if !m.wlPrivateData.DXCCConfirmed() {
					line += sep + S.Warning.Render("New DXCC!")
				}
			}
			if lipgloss.Width(line) > width {
				line = truncate(line, width)
			}
			result := pathInfoStyle.Width(width).Align(lipgloss.Left).Render(line)
			m.cachedPathSig = sig
			m.cachedPathLine = result
			return result
		}
		m.cachedPathSig = ""
		m.cachedPathLine = ""
		return ""
	}

	if partnerGrid != "" && ownGrid == "" {
		return pathMutedStyle.Width(width).Align(lipgloss.Left).
			Render("Set your grid in station config to enable path")
	}

	// Call entered but no path: WL info or fall back to station profile.
	var parts []string
	if m.wlPrivateData != nil {
		if !m.wlPrivateData.Worked() {
			parts = append(parts, S.Warning.Render("New Call!"))
		}
		if !m.wlPrivateData.DXCCConfirmed() {
			parts = append(parts, S.Warning.Render("New DXCC!"))
		}
	}
	if len(parts) == 0 {
		return renderParts(m.stationProfile(), pathMutedStyle, lipgloss.Left)
	}
	return renderParts(parts, pathInfoStyle, lipgloss.Left)
}
