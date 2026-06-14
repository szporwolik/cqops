package tui

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Table column definitions
// =============================================================================

// editorColTiers defines which columns to show at each available width.
var editorColTiers = []struct {
	names []string
}{
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "RSTs", "RSTr"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "RSTs", "RSTr", "DXCC"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Pwr"}},
}

// editorColWidths maps column titles to minimum widths.
var editorColWidths = map[string]int{
	"Date": 10, "Time": 8, "Call": 10, "WL": 3, "How": 4,
	"Band": 5, "Mode": 5, "Sub": 4, "RSTs": 4, "RSTr": 4,
	"DXCC": 6, "Name": 8, "Grid": 6, "QTH": 8,
	"Comment": 12, "Dist": 5, "Pwr": 4,
}

// =============================================================================
// Column value extraction
// =============================================================================

// editorColValue returns the display value for a column and QSO.
func editorColValue(col string, q *qso.QSO) string {
	switch col {
	case "Date":
		return formatDate(q.QSODate)
	case "Time":
		return formatTime(q.TimeOn)
	case "Call":
		return q.Call
	case "WL":
		if q.WavelogUploaded == "yes" {
			return "Y"
		} else if q.WavelogUploaded == "no" {
			return "N"
		}
		return ""
	case "How":
		switch q.Source {
		case "wsjtx":
			return "WSJT"
		case "manual":
			return "Man"
		default:
			return q.Source
		}
	case "Band":
		b := qso.NormalizeBand(q.Band)
		if b == "" && q.Freq > 0 {
			return fmt.Sprintf("%.1f", q.Freq)
		}
		return b
	case "Mode":
		return q.Mode
	case "Sub":
		return q.Submode
	case "RSTs":
		return q.RSTSent
	case "RSTr":
		return q.RSTRcvd
	case "DXCC":
		return q.Country
	case "Name":
		return q.Name
	case "Grid":
		return q.GridSquare
	case "QTH":
		return q.QTH
	case "Comment":
		return q.Comment
	case "Dist":
		if q.Distance > 0 {
			return fmt.Sprintf("%.0f", q.Distance)
		}
		return ""
	case "Pwr":
		return q.TXPower
	}
	return ""
}

// =============================================================================
// Table construction
// =============================================================================

func (le *LogbookEditor) buildTable() {
	w := le.width
	if w < 40 {
		w = 80
	}
	h := le.height
	if h < 10 {
		h = 24
	}
	// Dynamic viewport: fill available vertical space. Terminal height minus
	// status(1), tabs(1), help(1), border(2), spacer(1) = h-6.
	tableH := h - 7
	if tableH < 5 {
		tableH = 5
	}
	// Select the widest tier that fits within bodyW.
	bodyW := w - 4
	if bodyW < 20 {
		bodyW = 20
	}
	var names []string
	for _, t := range editorColTiers {
		total := 0
		for _, n := range t.names {
			total += editorColWidths[n]
		}
		total += len(t.names) - 1
		if total <= bodyW {
			names = t.names
		}
	}

	// Build columns from selected tier.
	var cols []table.Column
	minTotal := 0
	for _, n := range names {
		w := editorColWidths[n]
		minTotal += w
		cols = append(cols, table.Column{Title: n, Width: w})
	}
	gaps := len(cols) - 1
	extra := bodyW - gaps - minTotal
	if extra > 0 {
		dist := 0
		for i := range cols {
			var share int
			switch cols[i].Title {
			case "Comment":
				share = extra * 5 / 10
			case "Name":
				share = extra * 2 / 10
			case "QTH":
				share = extra / 10
			case "Call":
				share = extra / 10
			}
			cols[i].Width += share
			dist += share
		}
		if leftover := extra - dist; leftover > 0 {
			cols[len(cols)-1].Width += leftover
		}
	}

	// Rebuild rows with only the selected columns.
	var trimmedRows []table.Row
	for _, q := range le.qsos {
		var row table.Row
		for _, n := range names {
			row = append(row, editorColValue(n, &q))
		}
		trimmedRows = append(trimmedRows, row)
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(trimmedRows),
		table.WithFocused(true),
		table.WithHeight(tableH),
		table.WithWidth(bodyW),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderForeground(P.TextDim).
		BorderBottom(true).
		Bold(false).
		Foreground(P.Text).
		Background(P.Surface)
	// Cell: no Foreground, no Background — let the drawBorderedBox
	// wrapper supply Surface background, and let the default Selected
	// style (bold + pink foreground from bubbles) highlight the cursor row.
	t.SetStyles(s)
	t.Focus()
	le.table = t
	le.built = true
}
