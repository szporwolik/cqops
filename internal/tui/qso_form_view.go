package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// viewForm renders the QSO entry form in a three-column layout.
// width is the exact available space inside the border.
func (m *Model) viewForm(width int) string {
	bodyW := width
	if bodyW < 20 {
		bodyW = 20
	}
	dim := DimStyle
	choiceFields := map[field]bool{fieldBand: true, fieldMode: true, fieldSubmode: true}

	leftFields := []field{fieldDate, fieldTime, fieldCall, fieldFreq, fieldBand, fieldMode, fieldSubmode}
	middleFields := []field{fieldRSTSent, fieldRSTRcvd, fieldName, fieldQTH, fieldGrid, fieldCountry}
	rightFields := []field{fieldTXPower, fieldFreqRx, fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA}

	colW := (bodyW - 4) / 3 // 4 = two 2-char gaps between three columns
	if colW < 20 {
		colW = bodyW // fallback to single column on very narrow terminals
	}

	renderField := func(f field, w int) string {
		label := fieldNames[f]
		raw := strings.TrimSpace(m.fields[f].Value())
		isFocused := int(f) == int(m.focus) && !m.retainFocused
		ti := m.fields[f]

		choiceIcon := ""
		if choiceFields[f] {
			choiceIcon = dim.Render("\u25bc ")
		}

		// Width available for the textinput value.
		prefixW := 2  // "> " or "  "
		lblW := 13    // FormLabel.Width(13)
		choiceW := lipgloss.Width(choiceIcon)
		gapW := 1
		valW := w - prefixW - lblW - choiceW - gapW
		if valW > 12 {
			valW -= 3 // safety margin; more aggressive for narrow columns
		}
		if valW < 4 {
			valW = 4
		}
		ti.SetWidth(valW)
		if isFocused {
			ti.SetCursor(ti.Position()) // recalculate overflow for editing
			m.fields[f] = ti
		}

		// Focused: Bubbles handles width/scrolling natively.
		// Unfocused: truncate raw text — Bubbles' offsetRight isn't reliable
		// when cursor is at 0 and text exceeds width.
		var v string
		if raw == "" && !isFocused {
			v = SubtleStyle.Render("\u2014")
		} else if isFocused {
			v = ti.View()
		} else if f == fieldCall {
			v = S.Info.Render(truncateText(raw, valW))
		} else {
			v = ValueStyle.Render(truncateText(raw, valW))
		}
		val := choiceIcon + v

		// Label with focus indicator.
		prefix := "  "
		lblStyled := S.FormLabel.Align(lipgloss.Left).Render(label)
		if isFocused {
			prefix = CursorStyle.Render("> ")
			lblStyled = S.FormLabel.Copy().
				Foreground(lipgloss.Color("212")).
				Align(lipgloss.Left).
				Render(label)
		}
		lblPart := lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface).Render(prefix) + lblStyled
		if isFocused {
			lblPart = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(P.Surface).Render(prefix) + lblStyled
		}
		gap := lipgloss.NewStyle().Width(gapW).Background(P.Surface).Render(" ")
		return lipgloss.NewStyle().Width(w).MaxWidth(w).Background(P.Surface).Render(
			lipgloss.JoinHorizontal(lipgloss.Center, lblPart, gap, val),
		)
	}

	var b strings.Builder

	rows := len(leftFields)
	if len(middleFields) > rows {
		rows = len(middleFields)
	}
	if len(rightFields) > rows {
		rows = len(rightFields)
	}
	for i := 0; i < rows; i++ {
		var cols []string
		if i < len(leftFields) {
			cols = append(cols, renderField(leftFields[i], colW))
		}
		if i < len(middleFields) {
			cols = append(cols, renderField(middleFields[i], colW))
		}
		if i < len(rightFields) {
			cols = append(cols, renderField(rightFields[i], colW))
		}
		if colW >= 20 {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols...))
		} else {
			b.WriteString(lipgloss.JoinVertical(lipgloss.Left, cols...))
		}
		b.WriteString("\n")
	}

	// Comment row spans columns 1+2; Retain checkbox in column 3
	commentW := colW*2 + 2
	if commentW < 20 {
		commentW = bodyW
	}
	commentLine := renderField(fieldComment, commentW)

	retainBox := m.renderRetainCheckbox(colW)
	if colW >= 20 {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, commentLine, retainBox))
	} else {
		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, commentLine, retainBox))
	}
	b.WriteString("\n")

	return b.String()
}

// renderRetainCheckbox renders the "Retain" checkbox next to the Comment field.
func (m *Model) renderRetainCheckbox(colW int) string {
	mark := "[ ]"
	label := "Retain"
	if m.retainComment {
		mark = "[x]"
	}
	gap := lipgloss.NewStyle().Width(1).Background(P.Surface).Render(" ")
	space := lipgloss.NewStyle().Width(1).Background(P.Surface).Render(" ")
	if m.retainFocused {
		return lipgloss.NewStyle().Width(colW).Background(P.Surface).Render(
			lipgloss.JoinHorizontal(lipgloss.Center,
				CursorStyle.Render(" "+mark),
				gap,
				inputStyle.Render(label),
			),
		)
	}
	if m.retainComment {
		return lipgloss.NewStyle().Width(colW).Background(P.Surface).Render(
			lipgloss.JoinHorizontal(lipgloss.Center,
				space,
				inputStyle.Render(mark),
				gap,
				DimStyle.Render(label),
			),
		)
	}
	return lipgloss.NewStyle().Width(colW).Background(P.Surface).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			space,
			DimStyle.Render(mark),
			gap,
			DimStyle.Render(label),
		),
	)
}

// formPathRow renders the short-path info line between the QSO form and recent QSOs table.
func (m *Model) formPathRow(width int) string {
	ownGrid := formatLocator(m.App.Logbook.Station.Grid)
	partnerGrid := formatLocator(strings.TrimSpace(m.fields[fieldGrid].Value()))

	if ownGrid != "" && partnerGrid != "" {
		line := distanceLine(ownGrid, partnerGrid, m.App.Config.General.DistanceUnit)
		if line != "" {
			line = "Path  " + line
			if m.wlPrivateData != nil {
				if !m.wlPrivateData.Worked() {
					line += "  \u00b7  " + S.Warning.Render("New Call!")
				}
				if !m.wlPrivateData.DXCCConfirmed() {
					line += "  \u00b7  " + S.Warning.Render("New DXCC!")
				}
			}
			if lipgloss.Width(line) > width {
				line = truncate(line, width)
			}
			return lipgloss.NewStyle().
				Width(width).
				Align(lipgloss.Center).
				Foreground(P.Info).
				Background(P.Surface).
				Render(line)
		}
	}

	if partnerGrid != "" && ownGrid == "" {
		return lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Center).
			Foreground(P.TextMuted).
			Background(P.Surface).
			Render("Set your grid in station config to enable path")
	}

	counts, err := store.CountQSOs(m.App.DB)
	if err != nil {
		counts = store.QSOCounts{}
	}
	var parts []string
	if counts.Total > 0 {
		parts = append(parts, fmt.Sprintf("Log %d QSOs", counts.Total))
	}
	if counts.FromWSJTX > 0 {
		parts = append(parts, fmt.Sprintf("FTx %d", counts.FromWSJTX))
	}
	if counts.ToWavelog > 0 {
		parts = append(parts, fmt.Sprintf("WL %d", counts.ToWavelog))
	}
	if m.wlPrivateData != nil {
		if !m.wlPrivateData.Worked() {
			parts = append(parts, S.Warning.Render("New Call!"))
		}
		if !m.wlPrivateData.DXCCConfirmed() {
			parts = append(parts, S.Warning.Render("New DXCC!"))
		}
	}
	line := strings.Join(parts, " \u00b7 ")
	if lipgloss.Width(line) > width {
		line = truncate(line, width)
	}
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Foreground(P.TextMuted).
		Background(P.Surface).
		Render(line)
}
