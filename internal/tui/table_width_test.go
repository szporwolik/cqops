package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

// Width gates for the tables and the frame.
//
// bubbles/table renders every cell with Padding(0, 1), so a column whose
// content width is W occupies W+2 terminal cells. The width budget used to
// count one space per gap instead, which made a table render n+1 cells wider
// than the slot it was given (84 in 73, 133 in 118, 222 in 198) — and because
// lipgloss pads every rendered line to the widest one, one wide table made the
// whole frame wider than the terminal.
//
// The tab bar and the body filler had the same off-by-a-padding problem, so
// these gates check three levels: the table, the frame of the QSO screen, and
// the logbook editor.

// renderWidth returns the widest line of a rendered block.
func renderWidth(v string) int {
	width := 0
	for _, line := range splitRenderedLines(v) {
		if w := lipgloss.Width(line); w > width {
			width = w
		}
	}
	return width
}

func splitRenderedLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// widthTestQSOs is a set of contacts with long values in every column, so
// truncation — not an absence of data — decides the rendered width.
func widthTestQSOs(n int) []qso.QSO {
	qsos := make([]qso.QSO, 0, n)
	for i := 0; i < n; i++ {
		qsos = append(qsos, qso.QSO{
			ID: int64(i + 1), Call: "SP9MOA/P", QSODate: "20261028", TimeOn: "120000",
			Band: "20m", Freq: 14.074, Mode: "MFSK", Submode: "FT8",
			RSTSent: "599", RSTRcvd: "599", GridSquare: "JO90AA",
			Country: "Poland", Name: "Long Operator Name", QTH: "Krakow, Malopolska",
			Comment: "a rather long comment about the contact", Distance: 1234,
			TXPower: "1500", Operator: "SP9MOA", Source: "wsjtx",
			SOTARef: "SP/XX-0001", POTARef: "SP-0001", WWFFRef: "SPFF-0001",
			IOTA: "EU-032", SIG: "SOTA", ExchSent: "599 0001", ExchRcvd: "599 0002",
			ContestID: "abc", ContestADIFID: "SP-DX-CONTEST",
		})
	}
	return qsos
}

// TestRecentQSOsTableFitsItsWidth: the table must never render wider than the
// width it was given — at the tier thresholds and just above or below them,
// where a column set is added or dropped.
func TestRecentQSOsTableFitsItsWidth(t *testing.T) {
	qsos := widthTestQSOs(8)
	r := NewRecentQSOs(qsos)

	widths := []int{20, 30, 40, 50, 55, 60, 70, 73, 80, 90, 100, 115, 118, 120, 140, 150, 160, 165, 190, 198, 200, 240, 280, 320}
	for _, tiers := range [][]struct {
		minW  int
		maxW  int
		names []string
	}{qsoColTiers, qsoColTiersContest} {
		for _, tier := range tiers {
			widths = append(widths, tier.maxW-2, tier.maxW-1, tier.maxW, tier.maxW+1, tier.maxW+2)
		}
	}

	for _, contest := range []bool{false, true} {
		for _, multiOp := range []bool{false, true} {
			r.contest, r.multiOp = contest, multiOp
			for _, w := range widths {
				if w < 20 {
					continue
				}
				r.cachedView, r.filteredCachedView = "", ""
				r.width, r.height = w, 20
				if got := renderWidth(r.View()); got > w {
					t.Errorf("contest=%v multiOp=%v width %d: table renders %d cells", contest, multiOp, w, got)
				}
			}
		}
	}

	// With room for the narrowest column set the table fills the slot exactly:
	// no overflow, and no unused columns either.
	r.contest, r.multiOp = false, false
	roomy := qsoColTiers[0].maxW
	for w := roomy; w <= 260; w++ {
		r.cachedView = ""
		r.width, r.height = w, 20
		if got := renderWidth(r.View()); got != w {
			t.Fatalf("width %d: table renders %d cells, want an exact fit", w, got)
		}
	}
}

// TestLogbookEditorFrameFitsTheTerminal: the editor's own table uses the same
// tiers through its own budget, and its minimum terminal is 75 columns (below
// that the whole app refuses to start).
func TestLogbookEditorFrameFitsTheTerminal(t *testing.T) {
	le, _ := pagingTestEditor(t, 40)
	for w := 75; w <= 200; w++ {
		le.width, le.height = w, 29
		le.built = false
		le.buildTable()
		if got := renderWidth(le.View().Content); got > w {
			t.Errorf("editor at %d columns renders %d", w, got)
		}
	}
}

// TestFrameFitsTheTerminal: the widest line of a rendered QSO frame must fit
// the terminal. The status bar, the tab bar, the form and the recent table are
// all laid out for it, and lipgloss pads every line to the widest one.
func TestFrameFitsTheTerminal(t *testing.T) {
	for w := 75; w <= 240; w++ {
		for _, h := range []int{24, 40} {
			m := newLayoutTestModel(t, w, h)
			if got := renderWidth(m.View().Content); got > w {
				t.Errorf("frame at %dx%d renders %d cells wide", w, h, got)
			}
			if got := renderWidth(m.recentQSOs.View()); got > w-2 {
				t.Errorf("recent QSOs at %dx%d renders %d cells in a %d-cell slot", w, h, got, w-2)
			}
		}
	}
}
