package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// dxcTimeWindows is the list of time filter options in minutes (0 = all).
var dxcTimeWindows = []int{0, 60, 30, 15, 10, 5}

// dxcColWidths maps DXC column titles to minimum widths (matching editorColWidths).
var dxcColWidths = map[string]int{
	"Time": 8, "Freq": 8, "Band": 5, "Mode": 4, "DX Call": 10, "Spotter": 10, "Comment": 8,
}

// dxcColOrder is the fixed display order: Time, Freq, Band, Mode, DX Call, Spotter, Comment.
var dxcColOrder = []string{"Time", "Freq", "Band", "Mode", "DX Call", "Spotter", "Comment"}

// dxcColValue returns the display value for a DXC column and spot.
func dxcColValue(col string, s *store.DXCSpot) string {
	switch col {
	case "Time":
		return time.Unix(s.ReceivedAt, 0).UTC().Format("15:04:05")
	case "Freq":
		return fmt.Sprintf("%.1f", s.Frequency)
	case "Band":
		return s.Band
	case "Mode":
		return s.Mode
	case "DX Call":
		return s.DXCall
	case "Spotter":
		return s.Spotter
	case "Comment":
		return s.Comment
	}
	return ""
}

// buildDXCTable constructs the bubbles/table for DXC spots.
func (m *Model) buildDXCTable() {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	tableH := h - 6
	if tableH < 5 {
		tableH = 5
	}
	bodyW := w - 4
	if bodyW < 20 {
		bodyW = 20
	}
	if bodyW > partnerMapMaxW {
		bodyW = partnerMapMaxW
	}

	names := dxcColOrder

	var cols []table.Column
	minTotal := 0
	for _, n := range names {
		cw := dxcColWidths[n]
		minTotal += cw
		cols = append(cols, table.Column{Title: n, Width: cw})
	}
	gaps := len(cols) - 1
	extra := bodyW - gaps - minTotal
	if extra > 0 {
		dist := 0
		for i := range cols {
			var share int
			switch cols[i].Title {
			case "Comment":
				share = extra * 4 / 10
			case "Spotter":
				share = extra * 3 / 10
			case "DX Call":
				share = extra * 2 / 10
			}
			cols[i].Width += share
			dist += share
		}
		if leftover := extra - dist; leftover > 0 {
			cols[len(cols)-1].Width += leftover
		}
	}

	spots := m.dxcFilteredSpots()
	m.dxc.spotCount = len(spots)

	filtered := m.dxc.bandFilter != "" || m.dxc.modeFilter != ""
	bandHighlight := S.Info

	var rows []table.Row
	for _, s := range spots {
		s := s
		var row table.Row
		for _, n := range names {
			v := dxcColValue(n, &s)
			if v == "" {
				v = "\u2014"
			}
			if filtered && n == "Band" && v != "\u2014" && m.dxc.bandFilter != "" {
				v = bandHighlight.Render(v)
			}
			if filtered && n == "Mode" && v != "\u2014" && m.dxc.modeFilter != "" {
				v = bandHighlight.Render(v)
			}
			row = append(row, v)
		}
		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableH),
		table.WithWidth(bodyW),
	)
	sty := table.DefaultStyles()
	sty.Header = sty.Header.
		BorderForeground(P.TextDim).
		BorderBottom(true).
		Bold(false).
		Foreground(P.Text)
	if filtered {
		sty.Header = sty.Header.Foreground(P.Cursor)
	}
	t.SetStyles(sty)
	t.Focus()

	prevCall := m.dxc.selectedCall
	if prevCall != "" {
		for i, s := range spots {
			if s.DXCall == prevCall {
				t.GotoTop()
				t.MoveDown(i)
				break
			}
		}
	}

	m.dxc.table = t
	m.dxc.tableReady = true
	m.dxc.builtW = w
	m.dxc.builtH = h
	m.updateDXCSelectedCall()
}

// dxcView renders the DXC cluster spots table with filter info and spacer.
func (m *Model) dxcView() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	bodyW := w - 4
	if bodyW < 20 {
		bodyW = 20
	}
	if bodyW > partnerMapMaxW {
		bodyW = partnerMapMaxW
	}

	if m.dxc.tableReady && (m.width != m.dxc.builtW || m.height != m.dxc.builtH) {
		m.dxc.tableReady = false
	}
	if !m.dxc.tableReady {
		m.buildDXCTable()
	}

	contentH := contentHeight(h)

	timeVal := "all"
	if m.dxc.timeFilter > 0 {
		timeVal = fmt.Sprintf("%dm", m.dxc.timeFilter)
	}
	bandVal := "all"
	if m.dxc.bandFilter != "" {
		bandVal = m.dxc.bandFilter
	}
	modeVal := "all"
	if m.dxc.modeFilter != "" {
		modeVal = m.dxc.modeFilter
	}
	filterInfo := " " + DimStyle.Render("Filters:") + " " +
		DimStyle.Render("Mode") + " " + ValueStyle.Render(modeVal) +
		" " + DimStyle.Render("|") + " " +
		DimStyle.Render("Band") + " " + ValueStyle.Render(bandVal) +
		" " + DimStyle.Render("|") + " " +
		DimStyle.Render("Time") + " " + ValueStyle.Render(timeVal) +
		" " + DimStyle.Render("|") + " " +
		DimStyle.Render("Spots") + " " + ValueStyle.Render(fmt.Sprintf("%d", m.dxc.spotCount))
	spacer := lipgloss.NewStyle().Width(bodyW).Render(filterInfo)

	tablePart := lipgloss.NewStyle().
		MaxWidth(bodyW).
		Height(contentH - 1).
		Render(m.dxc.table.View())
	return lipgloss.JoinVertical(lipgloss.Left, spacer, tablePart)
}
