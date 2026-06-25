package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

// RecentQSOs is a read-only view of recent QSOs. In normal mode it shows the
// most recent QSOs. When a filter call is set, it queries the full DB for
// matching QSOs and highlights the call column.
type RecentQSOs struct {
	qsos   []qso.QSO
	width  int
	height int

	// View cache — avoids rebuilding the table on every frame when nothing changed.
	cachedView   string
	cachedW      int
	cachedH      int
	cachedQSOLen int

	// Filtered view cache — same pattern as unfiltered cache but keyed on
	// filterCall + filteredQSOs length to avoid table.New() every frame.
	filteredCachedView string
	filteredCachedW    int
	filteredCachedH    int
	filteredCachedLen  int

	// Filtered mode: when filterCall is non-empty, only matching QSOs are shown.
	filterCall       string
	filteredQSOs     []qso.QSO
	filterCacheID    int64 // last QSO ID in filtered set — invalidated on new QSO
	filterSuppressed bool  // set by ClearFilter to prevent refresh re-apply race
}

// NewRecentQSOs creates a read-only recent QSOs view.
func NewRecentQSOs(qsos []qso.QSO) *RecentQSOs {
	return &RecentQSOs{qsos: qsos, width: 80, height: 10}
}

// SetQSOS updates the QSO data and invalidates the view and filter caches.
func (r *RecentQSOs) SetQSOS(qsos []qso.QSO) {
	r.qsos = qsos
	r.cachedView = ""   // force rebuild on next View()
	r.filterCacheID = 0 // invalidate filter cache
}

// SetFilterCall switches to filtered mode. Pass qsos from store.SearchQSOsByCall.
func (r *RecentQSOs) SetFilterCall(call string, qsos []qso.QSO) {
	r.filterCall = strings.ToUpper(call)
	r.filteredQSOs = qsos
	r.filteredCachedView = "" // force rebuild
	if len(qsos) > 0 {
		r.filterCacheID = qsos[0].ID // newest matching QSO ID
	}
}

// ClearFilter returns to normal (unfiltered) mode and suppresses the next
// refreshQSOS filter re-apply to avoid a race with async filter commands.
func (r *RecentQSOs) ClearFilter() {
	r.filterCall = ""
	r.filteredQSOs = nil
	r.filteredCachedView = ""
	r.filterCacheID = 0
	r.filterSuppressed = true
}

// IsFiltered returns true when the table is in filtered mode.
func (r *RecentQSOs) IsFiltered() bool { return r.filterCall != "" }

// ActiveQSOs returns the currently active QSO set (filtered or normal).
func (r *RecentQSOs) ActiveQSOs() []qso.QSO {
	if r.filterCall != "" {
		return r.filteredQSOs
	}
	return r.qsos
}

// SetSize sets the available dimensions.
func (r *RecentQSOs) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// View renders the read-only recent QSOs table. In normal mode it shows
// recent QSOs; in filtered mode it shows QSOs matching the partner call
// with the call column highlighted.
func (r *RecentQSOs) View() string {
	bodyW := r.width
	if bodyW < 20 {
		bodyW = 20
	}
	maxRows := r.height - 1
	if maxRows < 3 {
		maxRows = 3
	}

	qsos := r.ActiveQSOs()
	filtered := r.filterCall != ""

	// Filtered-mode cache — avoids table.New() every frame when the call
	// highlight is the only difference (already rendered into the cached view).
	if filtered {
		if r.filteredCachedW == bodyW && r.filteredCachedH == maxRows &&
			r.filteredCachedLen == len(qsos) && r.filteredCachedView != "" {
			return r.filteredCachedView
		}
	} else {
		if r.cachedW == bodyW && r.cachedH == maxRows && r.cachedQSOLen == len(qsos) &&
			r.cachedView != "" {
			return r.cachedView
		}
	}

	// Pre-computed tier widths — pick the widest tier that fits bodyW.
	// Fall back to whittling columns from the narrowest tier for very small terminals.
	var names []string
	for i := len(qsoColTiers) - 1; i >= 0; i-- {
		if bodyW >= qsoColTiers[i].maxW {
			names = qsoColTiers[i].names
			break
		}
	}
	if names == nil {
		// Terminal too narrow for any tier — whittle from the narrowest set.
		names = append([]string{}, qsoColTiers[0].names...)
		for len(names) > 1 {
			total := 0
			for _, n := range names {
				total += qsoAllCols[n].minWidth
			}
			total += len(names) - 1
			if total <= bodyW {
				break
			}
			names = names[:len(names)-1]
		}
	}

	var cols []table.Column
	minTotal := 0
	for _, n := range names {
		minTotal += qsoAllCols[n].minWidth
		cols = append(cols, table.Column{
			Title: qsoAllCols[n].header,
			Width: qsoAllCols[n].minWidth,
		})
	}
	gaps := len(names) - 1
	extra := bodyW - gaps - minTotal
	if extra > 0 && len(cols) > 0 {
		distributed := 0
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
			case "Notes":
				share = extra / 10
			}
			cols[i].Width += share
			distributed += share
		}
		if leftover := extra - distributed; leftover > 0 {
			cols[len(cols)-1].Width += leftover
		}
	}

	rowCount := maxRows
	if rowCount > len(qsos) {
		rowCount = len(qsos)
	}

	// Pre-allocate call highlight style for filtered mode.
	callHighlight := S.Info

	var rows []table.Row
	for i := 0; i < rowCount; i++ {
		q := qsos[i]
		var row []string
		for _, n := range names {
			c := qsoAllCols[n]
			v := c.value(&q)
			if v == "" {
				v = "\u2014"
			}
			// Highlight call column when filtered.
			if filtered && n == "Call" && v != "\u2014" {
				v = callHighlight.Render(v)
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
		table.WithFocused(false),
	)

	s := table.DefaultStyles()
	headerStyle := s.Header.
		BorderForeground(P.TextDim).
		BorderBottom(true).
		Bold(false).
		Foreground(P.Text)
	// Highlight Call header when filtered.
	if filtered {
		// We can't easily style individual headers in bubbles/table v2,
		// so we use a distinct header foreground for visual cue.
		headerStyle = headerStyle.Foreground(P.Cursor)
	}
	s.Header = headerStyle
	s.Cell = s.Cell.Foreground(P.TextMuted)
	s.Selected = lipgloss.NewStyle()
	t.SetStyles(s)

	view := t.View()

	if filtered {
		r.filteredCachedView = view
		r.filteredCachedW = bodyW
		r.filteredCachedH = maxRows
		r.filteredCachedLen = len(qsos)
	} else {
		r.cachedView = view
		r.cachedW = bodyW
		r.cachedH = maxRows
		r.cachedQSOLen = len(qsos)
	}

	return view
}

// Height returns the rendered height of the component.
func (r *RecentQSOs) Height() int {
	return lipgloss.Height(r.View())
}

// qsoColTiers defines which columns to show at each terminal width.
// Scales gracefully from narrow terminals up to 2K+ monitors.
// No "ID" column — it's internal, not relevant for operators.
// Pre-computed max widths avoid repeated inner loops on every cache miss.
var qsoColTiers = []struct {
	minW  int
	maxW  int // pre-computed: sum of minWidths + (len(names)-1)
	names []string
}{
	{0, 0, []string{"Date", "Time", "Call", "Mode", "RSTs", "RSTr"}},
	{55, 0, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr"}},
	{70, 0, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "DXCC"}},
	{90, 0, []string{"Date", "Time", "Call", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name"}},
	{115, 0, []string{"Date", "Time", "Call", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist"}},
	{150, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power"}},
	{165, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Operator", "WL"}},
	{190, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Notes", "Source", "WL"}},
	{240, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Notes", "Source", "WL", "SOTA", "POTA", "IOTA"}},
}

func init() {
	for i := range qsoColTiers {
		t := &qsoColTiers[i]
		total := 0
		for _, n := range t.names {
			total += qsoAllCols[n].minWidth
		}
		total += len(t.names) - 1 // gaps between columns
		t.maxW = total
	}
}

// qsoAllCols defines all available columns for the QSO table.
var qsoAllCols = map[string]struct {
	header   string
	minWidth int
	value    func(q *qso.QSO) string
}{
	"Date":     {"Date", 10, func(q *qso.QSO) string { return formatDate(q.QSODate) }},
	"Time":     {"Time", 8, func(q *qso.QSO) string { return formatTime(q.TimeOn) }},
	"Call":     {"Call", 10, func(q *qso.QSO) string { return q.Call }},
	"Band":     {"Band", 5, func(q *qso.QSO) string { return bandOrFreq(q) }},
	"Freq":     {"Freq", 8, func(q *qso.QSO) string { return formatFreqShort(q.Freq) }},
	"Mode":     {"Mode", 5, func(q *qso.QSO) string { return q.Mode }},
	"Sub":      {"Sub", 4, func(q *qso.QSO) string { return q.Submode }},
	"RSTs":     {"RSTs", 4, func(q *qso.QSO) string { return q.RSTSent }},
	"RSTr":     {"RSTr", 4, func(q *qso.QSO) string { return q.RSTRcvd }},
	"DXCC":     {"DXCC", 6, func(q *qso.QSO) string { return q.Country }},
	"Name":     {"Name", 8, func(q *qso.QSO) string { return q.Name }},
	"Grid":     {"Grid", 6, func(q *qso.QSO) string { return q.GridSquare }},
	"QTH":      {"QTH", 8, func(q *qso.QSO) string { return q.QTH }},
	"Comment":  {"Comment", 12, func(q *qso.QSO) string { return q.Comment }},
	"Dist":     {"Dist", 5, func(q *qso.QSO) string { return formatDistanceShort(q.Distance) }},
	"Power":    {"Pwr", 4, func(q *qso.QSO) string { return q.TXPower }},
	"Notes":    {"Notes", 12, func(q *qso.QSO) string { return q.Notes }},
	"Operator": {"Op", 8, func(q *qso.QSO) string { return q.Operator }},
	"Source": {"Src", 4, func(q *qso.QSO) string {
		switch q.Source {
		case "wsjtx":
			return "WSJT"
		case "manual":
			return "Man"
		default:
			return q.Source
		}
	}},
	"WL": {"WL", 3, func(q *qso.QSO) string {
		if q.WavelogUploaded == "yes" {
			return "Y"
		}
		if q.WavelogUploaded == "no" {
			return "N"
		}
		return "\u2014"
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
