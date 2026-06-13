package tui

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

// RecentQSOs is a read-only view of recent QSOs. It renders a bubbles/table
// that receives no keyboard events — all input stays with the QSO form.
type RecentQSOs struct {
	qsos   []qso.QSO
	width  int
	height int

	// table is rebuilt on every View() from current data; it never receives
	// Update calls so it's always read-only.
	table table.Model
}

// NewRecentQSOs creates a read-only recent QSOs view.
func NewRecentQSOs(qsos []qso.QSO) *RecentQSOs {
	return &RecentQSOs{qsos: qsos, width: 80, height: 10}
}

// SetQSOS updates the QSO data. Called when new QSOs are available.
func (r *RecentQSOs) SetQSOS(qsos []qso.QSO) {
	r.qsos = qsos
}

// SetSize sets the available dimensions.
func (r *RecentQSOs) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// View renders the read-only recent QSOs table. It never calls table.Update,
// so the table cannot consume keyboard events.
func (r *RecentQSOs) View() string {
	bodyW := r.width
	if bodyW < 20 {
		bodyW = 20
	}
	// Table header uses 1 line; remaining space is data rows.
	// Always use full available height — the table pads empty rows gracefully.
	maxRows := r.height - 1 // header only
	if maxRows < 3 {
		maxRows = 3
	}

	// Select columns based on available width
	var names []string
	for _, t := range qsoColTiers {
		if bodyW >= t.minW {
			names = t.names
		}
	}

	var cols []table.Column
	// Sum minimum widths, then expand proportionally to fill available space
	minTotal := 0
	for _, n := range names {
		c := qsoAllCols[n]
		minTotal += c.minWidth
		cols = append(cols, table.Column{Title: c.header, Width: c.minWidth})
	}
	if minTotal < bodyW && len(cols) > 0 {
		extra := bodyW - minTotal
		// Distribute extra space: 50% Comment, 20% Name, 10% QTH, 10% Call, 10% Notes
		for i := range cols {
			switch cols[i].Title {
			case "Comment":
				cols[i].Width += extra * 5 / 10
			case "Name":
				cols[i].Width += extra * 2 / 10
			case "QTH":
				cols[i].Width += extra / 10
			case "Call":
				cols[i].Width += extra / 10
			case "Notes":
				cols[i].Width += extra / 10
			}
		}
	}

	var rows []table.Row
	rowCount := maxRows
	if rowCount > len(r.qsos) {
		rowCount = len(r.qsos)
	}
	for i := 0; i < rowCount; i++ {
		q := r.qsos[i]
		var row []string
		for _, n := range names {
			c := qsoAllCols[n]
			v := c.value(&q)
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
		table.WithHeight(maxRows+1),
		table.WithWidth(bodyW),
		table.WithFocused(false), // read-only: no cursor, no selection
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(P.TextDim).
		BorderBottom(true).
		Bold(false).
		Foreground(P.TextMuted)
	s.Cell = s.Cell.Foreground(P.TextDim)
	t.SetStyles(s)

	return t.View()
}

// Height returns the rendered height of the component.
func (r *RecentQSOs) Height() int {
	return lipgloss.Height(r.View())
}

// qsoColTiers defines which columns to show at each terminal width.
// Scales gracefully from narrow terminals up to 2K+ monitors.
// No "ID" column — it's internal, not relevant for operators.
var qsoColTiers = []struct {
	minW  int
	names []string
}{
	{0, []string{"Date", "Time", "Call", "Mode", "RSTs", "RSTr"}},
	{55, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr"}},
	{70, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "DXCC"}},
	{90, []string{"Date", "Time", "Call", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name"}},
	{115, []string{"Date", "Time", "Call", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist"}},
	{150, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power"}},
	{190, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Notes", "Source"}},
	{240, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Notes", "Source", "SOTA", "POTA", "IOTA"}},
}

// qsoAllCols defines all available columns for the QSO table.
var qsoAllCols = map[string]struct {
	header   string
	minWidth int
	value    func(q *qso.QSO) string
}{
	"Date":    {"Date", 10, func(q *qso.QSO) string { return formatDate(q.QSODate) }},
	"Time":    {"Time", 8, func(q *qso.QSO) string { return formatTime(q.TimeOn) }},
	"Call":    {"Call", 10, func(q *qso.QSO) string { return q.Call }},
	"Band":    {"Band", 5, func(q *qso.QSO) string { return bandOrFreq(q) }},
	"Freq":    {"Freq", 8, func(q *qso.QSO) string { return formatFreqShort(q.Freq) }},
	"Mode":    {"Mode", 5, func(q *qso.QSO) string { return q.Mode }},
	"Sub":     {"Sub", 4, func(q *qso.QSO) string { return q.Submode }},
	"RSTs":    {"RSTs", 4, func(q *qso.QSO) string { return q.RSTSent }},
	"RSTr":    {"RSTr", 4, func(q *qso.QSO) string { return q.RSTRcvd }},
	"DXCC":    {"DXCC", 6, func(q *qso.QSO) string { return q.Country }},
	"Name":    {"Name", 8, func(q *qso.QSO) string { return q.Name }},
	"Grid":    {"Grid", 6, func(q *qso.QSO) string { return q.GridSquare }},
	"QTH":     {"QTH", 8, func(q *qso.QSO) string { return q.QTH }},
	"Comment": {"Comment", 12, func(q *qso.QSO) string { return q.Comment }},
	"Dist":    {"Dist", 5, func(q *qso.QSO) string { return formatDistanceShort(q.Distance) }},
	"Power":   {"Pwr", 4, func(q *qso.QSO) string { return q.TXPower }},
	"Notes":   {"Notes", 12, func(q *qso.QSO) string { return q.Notes }},
	"Source": {"Src", 4, func(q *qso.QSO) string {
		switch q.Source {
		case "wsjtx":
			return "FTx"
		case "manual":
			return "Man"
		default:
			return q.Source
		}
	}},
	"SOTA": {"SOTA", 8, func(q *qso.QSO) string { return q.SOTARef }},
	"POTA": {"POTA", 8, func(q *qso.QSO) string { return q.POTARef }},
	"IOTA": {"IOTA", 6, func(q *qso.QSO) string { return q.IOTA }},
}

func bandOrFreq(q *qso.QSO) string {
	b := qso.NormalizeBand(q.Band)
	if b == "" && q.Freq > 0 {
		return fmt.Sprintf("%.1f", q.Freq)
	}
	return b
}

func formatDistanceShort(d float64) string {
	if d <= 0 {
		return ""
	}
	return fmt.Sprintf("%.0f", d)
}

func formatFreqShort(f float64) string {
	if f <= 0 {
		return ""
	}
	return fmt.Sprintf("%.4f", f)
}
