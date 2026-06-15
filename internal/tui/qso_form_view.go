package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/store"
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
// width is the exact available space inside the border.
func (m *Model) viewForm(width int) string {
	bodyW := width
	if bodyW < 20 {
		bodyW = 20
	}

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
			choiceIcon = choiceIconStr
		}

		// Width available for the textinput value.
		valW := w - 2 - 13 - choiceIconW - 1 - 2
		if valW > 20 {
			valW -= 1
		}
		if valW < 3 {
			valW = 3
		}
		ti.SetWidth(valW)
		if isFocused {
			if lipgloss.Width(raw) > valW {
				ti.SetWidth(valW - 1)
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
			v = S.Info.Render(truncateText(raw, valW))
		} else {
			v = ValueStyle.Render(truncateText(raw, valW))
		}
		val := choiceIcon + v

		// Label with focus indicator.
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
			cols = append(cols, renderField(formLeft[i], colW))
		}
		if i < len(formMiddle) {
			cols = append(cols, renderField(formMiddle[i], colW))
		}
		if i < len(formRight) {
			cols = append(cols, renderField(formRight[i], colW))
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

// formPathRow renders the short-path info line between the QSO form and recent QSOs table.
// Results are cached — only recomputed when grids or counts change.
func (m *Model) formPathRow(width int) string {
	ownGrid := formatLocator(m.App.Logbook.Station.Grid)
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
			result := pathInfoStyle.Width(width).Align(lipgloss.Center).Render(line)
			m.cachedPathSig = sig
			m.cachedPathLine = result
			return result
		}
	}
	m.cachedPathSig = ""
	m.cachedPathLine = ""

	if partnerGrid != "" && ownGrid == "" {
		return pathMutedStyle.Width(width).Align(lipgloss.Center).
			Render("Set your grid in station config to enable path")
	}

	// Use cached counts — refreshed on QSO save/delete.
	if !m.qsoCountsValid {
		counts, err := store.CountQSOs(m.App.DB)
		if err != nil {
			counts = store.QSOCounts{}
		}
		m.qsoCounts = counts
		m.qsoCountsValid = true
	}
	counts := m.qsoCounts
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
	return pathMutedStyle.Width(width).Align(lipgloss.Center).Render(line)
}
