package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// dxcTimeWindows is the list of time filter options in minutes (0 = all).
var dxcTimeWindows = []int{0, 60, 30, 15, 10, 5}

// dxcColWidths maps DXC column keys to minimum widths.
var dxcColWidths = map[string]int{
	"Time": 8, "Freq": 8, "Band": 5, "Mode": 4, "DX Cont": 3, "DXCC": 10, "DX Call": 8, "Spot Cont": 3, "Spotter": 7, "Comment": 8,
}

// dxcMaxWidths maps column keys to maximum widths for width distribution.
// Pre-allocated to avoid per-table-rebuild allocation.
var dxcMaxWidths = map[string]int{
	"DX Call": 11,
	"Spotter": 9,
	"DXCC":    12,
}

// dxcColOrder is the fixed display order (keys, not titles).
var dxcColOrder = []string{"Time", "Freq", "Band", "Mode", "DX Cont", "DX Call", "Spot Cont", "Spotter", "Comment"}

// dxcColTitle returns the header title for a column, or "" for no header.
func dxcColTitle(key string) string {
	switch key {
	case "DX Cont", "Spot Cont":
		return ""
	}
	return key
}

// dxcColValue returns the display value for a DXC column and spot.
func dxcColValue(col string, s *store.DXCSpot) string {
	switch col {
	case "Time":
		return formatDXCSpotTime(s.ReceivedAt)
	case "Freq":
		if s.Frequency <= 0 {
			return "\u2014"
		}
		mhz := s.Frequency / 1000
		if mhz < 100 {
			return strconv.FormatFloat(mhz, 'f', 4, 64)
		}
		return strconv.FormatFloat(mhz, 'f', 3, 64)
	case "Band":
		return s.Band
	case "Mode":
		return s.Mode
	case "DX Call":
		return s.DXCall
	case "DX Cont":
		if s.DXCont == "" {
			return "\u2014"
		}
		return s.DXCont
	case "DXCC":
		if s.DXCC == "" {
			return "\u2014"
		}
		return s.DXCC
	case "Spotter":
		return s.Spotter
	case "Spot Cont":
		if s.SpotCont == "" {
			return "\u2014"
		}
		return s.SpotCont
	case "Comment":
		return s.Comment
	}
	return ""
}

// formatDXCSpotTime formats a Unix timestamp as HH:MM:SS using strconv
// instead of time.Format() to avoid allocation per spot row.
func formatDXCSpotTime(ts int64) string {
	t := time.Unix(ts, 0).UTC()
	h, m, s := t.Hour(), t.Minute(), t.Second()
	var sb strings.Builder
	if h < 10 {
		sb.WriteByte('0')
	}
	sb.WriteString(strconv.Itoa(h))
	sb.WriteByte(':')
	if m < 10 {
		sb.WriteByte('0')
	}
	sb.WriteString(strconv.Itoa(m))
	sb.WriteByte(':')
	if s < 10 {
		sb.WriteByte('0')
	}
	sb.WriteString(strconv.Itoa(s))
	return sb.String()
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

	names := make([]string, len(dxcColOrder))
	copy(names, dxcColOrder)

	// On wide screens (≥120 cols), insert DXCC column between DX Cont and DX Call.
	showDXCC := w >= 120
	if showDXCC {
		insertAt := -1
		for i, n := range names {
			if n == "DX Call" {
				insertAt = i
				break
			}
		}
		if insertAt >= 0 {
			names = append(names, "")
			copy(names[insertAt+1:], names[insertAt:])
			names[insertAt] = "DXCC"
		}
	}

	var cols []table.Column
	minTotal := 0
	for _, n := range names {
		cw := dxcColWidths[n]
		minTotal += cw
		title := dxcColTitle(n)
		// Color only the header of the actively filtered column blue, so
		// the user can see which column the filter applies to. Headers are
		// static (no per-row ANSI nesting) so this never conflicts with
		// the table's Selected-row style.
		switch {
		case m.dxc.bandFilter != "" && n == "Band":
			title = S.Info.Render(title)
		case m.dxc.modeFilter != "" && n == "Mode":
			title = S.Info.Render(title)
		case m.dxc.contFilter != "" && n == "DX Cont":
			title = S.Info.Render(title)
		}
		cols = append(cols, table.Column{Title: title, Width: cw})
	}
	gaps := len(cols) - 1
	extra := bodyW - gaps - minTotal
	if extra > 0 {
		// Compute per-column max caps. DXCC gets double width on very wide screens.
		// Use package-level map; only override DXCC for ultra-wide terminals.
		dxccExtra := 12
		if w >= 125 {
			dxccExtra = 24
		}

		// Give non-Comment columns a chance to grow to their max caps.
		for i := range cols {
			if cols[i].Title == "Comment" {
				continue
			}
			maxW := 0
			if cols[i].Title == "DXCC" {
				maxW = dxccExtra
			} else if mw, ok := dxcMaxWidths[cols[i].Title]; ok {
				maxW = mw
			}
			if maxW > 0 && cols[i].Width < maxW {
				need := maxW - cols[i].Width
				if need > extra {
					need = extra
				}
				cols[i].Width += need
				extra -= need
			}
		}
		// All remaining extra space goes to Comment.
		if extra > 0 {
			for i := range cols {
				if cols[i].Title == "Comment" {
					cols[i].Width += extra
					break
				}
			}
		}
	}

	spots := m.dxcFilteredSpots()
	m.dxc.spotCount = len(spots)

	filtered := m.dxc.bandFilter != "" || m.dxc.modeFilter != "" || m.dxc.contFilter != ""

	var rows []table.Row
	for _, s := range spots {
		s := s
		var row table.Row
		for _, n := range names {
			v := dxcColValue(n, &s)
			if v == "" {
				v = "\u2014"
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
	m.dxc.cachedFilterInfo = "" // invalidate on table rebuild (width/spot change)
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

	// Cache filter info line — only rebuild when width or filters change.
	if m.dxc.cachedFilterInfo == "" || m.dxc.cachedFilterW != bodyW {
		timeVal := "all"
		if m.dxc.timeFilter > 0 {
			timeVal = fmt.Sprintf("%dm", m.dxc.timeFilter)
		}
		bandVal := "all"
		if m.dxc.bandFilter != "" {
			bandVal = m.dxc.bandFilter
		}
		contVal := "all"
		if m.dxc.contFilter != "" {
			contVal = m.dxc.contFilter
		}
		modeVal := "all"
		if m.dxc.modeFilter != "" {
			modeVal = m.dxc.modeFilter
		}
		m.dxc.cachedFilterInfo = " " + DimStyle.Render("Filters:") + " " +
			DimStyle.Render("Cont") + " " + ValueStyle.Render(contVal) +
			" " + DimStyle.Render("|") + " " +
			DimStyle.Render("Mode") + " " + ValueStyle.Render(modeVal) +
			" " + DimStyle.Render("|") + " " +
			DimStyle.Render("Band") + " " + ValueStyle.Render(bandVal) +
			" " + DimStyle.Render("|") + " " +
			DimStyle.Render("Time") + " " + ValueStyle.Render(timeVal) +
			" " + DimStyle.Render("|") + " " +
			DimStyle.Render("Spots") + " " + ValueStyle.Render(fmt.Sprintf("%d", m.dxc.spotCount))
		m.dxc.cachedFilterW = bodyW
	}
	if m.dxc.cachedSpacerStyleW != bodyW {
		m.dxc.cachedSpacerStyle = lipgloss.NewStyle().Width(bodyW).MaxWidth(bodyW)
		m.dxc.cachedSpacerStyleW = bodyW
	}
	spacer := m.dxc.cachedSpacerStyle.Render(m.dxc.cachedFilterInfo)

	tablePart := m.dxc.cachedSpacerStyle.
		Height(contentH - 1).
		Render(m.dxc.table.View())
	return lipgloss.JoinVertical(lipgloss.Left, spacer, tablePart)
}
