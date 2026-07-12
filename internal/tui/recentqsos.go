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

	// Contest mode swaps SOTA/POTA/WWFF/IOTA/SIG for ExchSent/ExchRcvd
	// at the wide tiers. Cache is invalidated when this changes.
	contest bool

	// Multi-operator mode swaps Grid for Operator at all tiers. Used
	// for club logbooks where multiple ops log under one callsign.
	multiOp bool
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

// SetContest enables or disables contest column mode. When active, the
// SOTA/POTA/WWFF/IOTA/SIG columns are replaced with ExchSent/ExchRcvd.
func (r *RecentQSOs) SetContest(v bool) {
	if r.contest == v {
		return
	}
	r.contest = v
	r.cachedView = ""
	r.filteredCachedView = ""
}

// SetMultiOp enables or disables multi-operator mode. When active, the
// Operator column replaces Grid — useful for club logbooks where
// multiple operators log under the same callsign.
func (r *RecentQSOs) SetMultiOp(v bool) {
	if r.multiOp == v {
		return
	}
	r.multiOp = v
	r.cachedView = ""
	r.filteredCachedView = ""
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
	tiers := qsoColTiers
	if r.contest {
		tiers = qsoColTiersContest
	}
	var names []string
	for i := len(tiers) - 1; i >= 0; i-- {
		if bodyW >= tiers[i].maxW {
			names = tiers[i].names
			break
		}
	}
	if names == nil {
		// Terminal too narrow for any tier — whittle from the narrowest set.
		names = append([]string{}, tiers[0].names...)
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

	// Multi-operator mode: swap Grid for Operator (useful for club
	// logbooks where callsign alone doesn't identify the op).
	if r.multiOp {
		for i, n := range names {
			if n == "Grid" {
				names[i] = "Operator"
				break
			}
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

	// Per-column width caps prevent short fields from blowing up on
	// ultra-wide screens. Caps are generous enough to never truncate
	// real data — they only cap the extra space distributed beyond
	// minWidth. Columns not listed here (e.g. future additions)
	// absorb leftover via the final dump to the last column.
	caps := map[string]int{
		"Date":    10, // YYYY-MM-DD
		"Time":    8,  // HH:MM:SS
		"Call":    12, // longest ~10 (with portable suffix)
		"Band":    7,  // "1.25cm" is 6
		"Freq":    10, // "1296.0000" is 9
		"Mode":    6,  // "RTTY" is 4
		"Sub":     5,  // "LSB"/"USB" is 3
		"RSTs":    5,  // "599" is 3
		"RSTr":    5,
		"DXCC":    20, // longest country name ~32, but 20 is practical
		"Name":    30, // generous for operator/org names
		"Grid":    8,  // "AA00aa" max 6
		"QTH":     25, // city/region names rarely exceed this
		"Comment": 30, // generous for short notes
		"Dist":    6,  // distance in km, 5 digits max
		"Pwr":     5,  // "1500" is 4
		"Src":     5,  // "WSJT" or "Man" max 4
		"WL":      4,  // "Y"/"N"/"—" max 1
		"SOTA":    14, // "XX-nnnn" ~7-8; higher cap lets it breathe on huge screens
		"POTA":    14, // same pattern
		"WWFF":    16, // "XXFF-nnnn" ~9; higher cap absorbs leftover on wide displays
		"IOTA":    10, // "XX-nnn" is 6
		"SIG":     8,  // "SOTA"/"POTA" max 4
		"Snt":     10, // "599 0001" is 8
		"Rcv":     10,
		"Op":      12, // callsign ~6-8
	}

	if extra > 0 && len(cols) > 0 {
		// Pass 1: proportional distribution to text-heavy columns.
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
			}
			if share > 0 {
				share = applyCap(cols[i].Width, share, cols[i].Title, caps)
			}
			cols[i].Width += share
			distributed += share
		}

		// Pass 2+: iteratively redistribute leftover among columns
		// still below their caps. Stop when nothing moves or no
		// columns can accept more.
		leftover := extra - distributed
		for leftover > 0 {
			moved := 0
			eligible := 0
			for i := range cols {
				if cap, ok := caps[cols[i].Title]; !ok || cols[i].Width < cap {
					eligible++
				}
			}
			if eligible == 0 {
				// All columns at cap — dump remainder on last column.
				cols[len(cols)-1].Width += leftover
				break
			}
			perCol := leftover / eligible
			if perCol == 0 {
				perCol = 1
			}
			for i := range cols {
				if cap, ok := caps[cols[i].Title]; ok && cols[i].Width >= cap {
					continue
				}
				add := perCol
				if add > leftover {
					add = leftover
				}
				add = applyCap(cols[i].Width, add, cols[i].Title, caps)
				cols[i].Width += add
				leftover -= add
				moved += add
				if leftover <= 0 {
					break
				}
			}
			if moved == 0 {
				// Safety: nothing moved, dump and exit.
				cols[len(cols)-1].Width += leftover
				break
			}
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
	{190, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Source", "WL"}},
	{240, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Source", "WL", "SOTA", "POTA", "IOTA"}},
	{280, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Source", "WL", "SOTA", "POTA", "WWFF", "IOTA", "SIG"}},
}

// qsoColTiersContest mirrors qsoColTiers but swaps SOTA/POTA/WWFF/IOTA/SIG
// for ExchSent/ExchRcvd at the wide tiers. Narrow and medium tiers are
// identical — contest exchanges only appear once there is enough room
// (same thresholds as SOTA/POTA/IOTA would in non-contest mode).
var qsoColTiersContest = []struct {
	minW  int
	maxW  int
	names []string
}{
	{0, 0, []string{"Date", "Time", "Call", "Mode", "RSTs", "RSTr"}},
	{55, 0, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr"}},
	{70, 0, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "DXCC"}},
	{90, 0, []string{"Date", "Time", "Call", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name"}},
	{115, 0, []string{"Date", "Time", "Call", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist"}},
	{150, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power"}},
	{165, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Operator", "WL"}},
	{190, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Source", "WL"}},
	// Wide tier: ExchSent/ExchRcvd replace SOTA/POTA/IOTA.
	{220, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Source", "WL", "ExchSent", "ExchRcvd"}},
	// Super-wide tier: same as wide but triggers later (more room for exchange to breathe).
	{280, 0, []string{"Date", "Time", "Call", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Source", "WL", "ExchSent", "ExchRcvd"}},
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
	for i := range qsoColTiersContest {
		t := &qsoColTiersContest[i]
		total := 0
		for _, n := range t.names {
			total += qsoAllCols[n].minWidth
		}
		total += len(t.names) - 1
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
	"SOTA":     {"SOTA", 8, func(q *qso.QSO) string { return q.SOTARef }},
	"POTA":     {"POTA", 8, func(q *qso.QSO) string { return q.POTARef }},
	"WWFF":     {"WWFF", 8, func(q *qso.QSO) string { return q.WWFFRef }},
	"IOTA":     {"IOTA", 6, func(q *qso.QSO) string { return q.IOTA }},
	"SIG":      {"SIG", 5, func(q *qso.QSO) string { return q.SIG }},
	"ExchSent": {"Snt", 6, func(q *qso.QSO) string { return q.ExchSent }},
	"ExchRcvd": {"Rcv", 6, func(q *qso.QSO) string { return q.ExchRcvd }},
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

// applyCap clamps share so that width+share does not exceed the column's
// cap. Returns the (possibly reduced) share. If the column is not in the
// caps map, share is returned unchanged.
func applyCap(currentWidth, share int, title string, caps map[string]int) int {
	cap, ok := caps[title]
	if !ok {
		return share
	}
	if currentWidth+share > cap {
		share = cap - currentWidth
		if share < 0 {
			share = 0
		}
	}
	return share
}
