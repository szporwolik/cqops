package tui

import (
	"charm.land/bubbles/v2/table"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Table column definitions
// =============================================================================

// editorColTiers defines which columns to show at each available width.
// Uses the same qsoAllCols registry as RecentQSOs — column keys, widths,
// and value functions are shared. The editor adds "Contest" at the wide
// end; otherwise the tier structure mirrors RecentQSOs.
var editorColTiers = []struct {
	names []string
}{
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "RSTs", "RSTr"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "RSTs", "RSTr", "DXCC"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Contest"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Contest"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Operator", "Contest"}},
}

// editorColTiersContest mirrors editorColTiers but replaces reference
// columns with ExchSent/ExchRcvd at the wide tiers (same as RecentQSOs).
var editorColTiersContest = []struct {
	names []string
}{
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "RSTs", "RSTr"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "RSTs", "RSTr", "DXCC"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Contest"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "Contest"}},
	{[]string{"Date", "Time", "Call", "WL", "Source", "Band", "Freq", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Power", "ExchSent", "ExchRcvd", "Contest"}},
}

// =============================================================================
// Column value extraction (delegates to shared qsoAllCols)
// =============================================================================

func editorColValue(col string, q *qso.QSO) string {
	info, ok := qsoAllCols[col]
	if !ok {
		return ""
	}
	v := info.value(q)
	if v == "" {
		return "\u2014"
	}
	return v
}

// =============================================================================
// Table construction (shared logic with RecentQSOs)
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
	tableH := h - 6
	if tableH < 5 {
		tableH = 5
	}
	bodyW := w - 4
	if bodyW < 20 {
		bodyW = 20
	}

	// Reuse cached tier selection and column layout when width hasn't
	// changed and contest/multiOp flags are stable. On low-end hardware
	// this avoids recomputing column widths on every cursor move.
	tiers := editorColTiers
	if le.contest {
		tiers = editorColTiersContest
	}
	var names []string
	if le.formattedNames != nil && le.formattedWidth == bodyW {
		names = le.formattedNames
	} else {
		for i := len(tiers) - 1; i >= 0; i-- {
			total := 0
			for _, n := range tiers[i].names {
				total += qsoAllCols[n].minWidth
			}
			total += len(tiers[i].names) - 1
			if total <= bodyW {
				names = tiers[i].names
				break
			}
		}
		if len(names) == 0 && len(tiers) > 0 {
			names = tiers[0].names
		}

		// Multi-operator: swap Grid for Operator.
		if le.multiOp {
			for i, n := range names {
				if n == "Grid" {
					names[i] = "Operator"
					break
				}
			}
		}
		le.formattedNames = names
		le.formattedWidth = bodyW
		le.formattedRows = nil // invalidate row cache on tier change
	}

	// Build columns from shared qsoAllCols registry.
	var cols []table.Column
	minTotal := 0
	for _, n := range names {
		cw := qsoAllCols[n].minWidth
		minTotal += cw
		cols = append(cols, table.Column{Title: qsoAllCols[n].header, Width: cw})
	}

	// Extra width — capped per-column, iteratively redistributed (shared
	// colCaps map and applyCap from RecentQSOs).
	gaps := len(cols) - 1
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
			}
			if share > 0 {
				share = applyCap(cols[i].Width, share, cols[i].Title, colCaps)
			}
			cols[i].Width += share
			distributed += share
		}
		leftover := extra - distributed
		for leftover > 0 {
			moved := 0
			eligible := 0
			for i := range cols {
				if cap, ok := colCaps[cols[i].Title]; !ok || cols[i].Width < cap {
					eligible++
				}
			}
			if eligible == 0 {
				cols[len(cols)-1].Width += leftover
				break
			}
			perCol := leftover / eligible
			if perCol == 0 {
				perCol = 1
			}
			for i := range cols {
				if cap, ok := colCaps[cols[i].Title]; ok && cols[i].Width >= cap {
					continue
				}
				add := perCol
				if add > leftover {
					add = leftover
				}
				add = applyCap(cols[i].Width, add, cols[i].Title, colCaps)
				cols[i].Width += add
				leftover -= add
				moved += add
				if leftover <= 0 {
					break
				}
			}
			if moved == 0 {
				cols[len(cols)-1].Width += leftover
				break
			}
		}
	}

	// Build rows — reuse pre-formatted cache when available.
	var trimmedRows []table.Row
	if le.formattedRows != nil && len(le.formattedRows) == len(le.qsos) {
		trimmedRows = le.formattedRows
	} else {
		// Pre-allocate for the page.
		trimmedRows = make([]table.Row, 0, len(le.qsos))
		for _, q := range le.qsos {
			row := make(table.Row, len(names))
			for j, n := range names {
				row[j] = editorColValue(n, &q)
			}
			trimmedRows = append(trimmedRows, row)
		}
		le.formattedRows = trimmedRows
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
		Foreground(P.Text)
	t.SetStyles(s)
	t.Focus()
	le.table = t
	le.built = true
	le.builtW = w
	le.builtH = h
	le.cachedSig = "" // invalidate view cache — table was rebuilt
}
