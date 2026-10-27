package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// The column tiers (qsoColTiers, editorColTiers and their contest variants) are
// shared package-level definitions. Multi-operator mode swaps Grid for Operator,
// and doing that in place rewrote the pattern for every later view — including
// the same view after the mode was turned off again.

// recentQSOsColumns renders the recent-QSOs table at width w. A fresh view
// every time, so the table's own render cache cannot hide a change.
func recentQSOsColumns(w int, multiOp, contest bool) string {
	view := NewRecentQSOs([]qso.QSO{{
		Call: "SP9MOA", Band: "20m", Mode: "SSB", GridSquare: "JO90AA",
		QSODate: "20261027", TimeOn: "120000", RSTSent: "59", RSTRcvd: "59",
	}})
	view.SetSize(w, 12)
	view.SetContest(contest)
	view.SetMultiOp(multiOp)
	return view.View()
}

// TestRecentQSOsMultiOpRoundTripKeepsTheColumns: single → multi → single must
// come back to exactly the table the single view had. Switching the mode off
// used to leave the Operator column in place, because the swap had rewritten
// the shared tier.
func TestRecentQSOsMultiOpRoundTripKeepsTheColumns(t *testing.T) {
	plainWidths := make([]int, 0, len(qsoColTiers))
	for i := range qsoColTiers {
		if hasColumn(qsoColTiers[i].names, "Grid") {
			plainWidths = append(plainWidths, qsoColTiers[i].maxW+10)
		}
	}
	contestWidths := make([]int, 0, len(qsoColTiersContest))
	for i := range qsoColTiersContest {
		if hasColumn(qsoColTiersContest[i].names, "Grid") {
			contestWidths = append(contestWidths, qsoColTiersContest[i].maxW+10)
		}
	}
	for _, tc := range []struct {
		name    string
		contest bool
		widths  []int
	}{
		{"ordinary", false, plainWidths},
		{"contest", true, contestWidths},
	} {
		if len(tc.widths) == 0 {
			t.Fatalf("%s: no tier carries a Grid column, the test proves nothing", tc.name)
		}
		for _, w := range tc.widths {
			before := recentQSOsColumns(w, false, tc.contest)
			club := recentQSOsColumns(w, true, tc.contest)
			if before == club {
				t.Fatalf("%s width %d: multi-operator mode changed nothing, so the test proves nothing", tc.name, w)
			}
			after := recentQSOsColumns(w, false, tc.contest)
			if after != before {
				t.Errorf("%s width %d: the single-operator table changed after a multi-operator table was rendered\nbefore:\n%s\nafter:\n%s",
					tc.name, w, before, after)
			}
		}
	}
}

// TestRecentQSOsClubViewDoesNotAffectASecondView is the two-view case: a club
// logbook and an ordinary one open against the same definitions. Rendering the
// club table must not restyle the ordinary one.
func TestRecentQSOsClubViewDoesNotAffectASecondView(t *testing.T) {
	qsos := []qso.QSO{{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20261027", TimeOn: "120000"}}

	plain := NewRecentQSOs(qsos)
	plain.SetSize(160, 12)
	first := plain.View()

	club := NewRecentQSOs(qsos)
	club.SetSize(160, 12)
	club.SetMultiOp(true)
	clubView := club.View()
	if clubView == first {
		t.Fatal("precondition: the club table rendered identically to the ordinary one")
	}

	// Force the ordinary view to render from scratch: its cache would
	// otherwise return the string computed before the club view existed.
	plain.SetQSOS(qsos)
	if second := plain.View(); second != first {
		t.Errorf("an ordinary table changed after a club table was rendered\nbefore:\n%s\nafter:\n%s", first, second)
	}
}

// editorColumns builds the editor table at width w and returns the columns it
// picked.
func editorColumns(w int, multiOp, contest bool) []string {
	le := &LogbookEditor{width: w, height: 40}
	if contest {
		le.SetContestMode(true)
	}
	if multiOp {
		le.SetMultiOp(true)
	}
	le.buildTable()
	return append([]string(nil), le.formattedNames...)
}

// assertEditorMultiOpRoundTrip: at the exact width a tier needs, a club table
// must not change the columns an ordinary table picks — and the ordinary table
// must keep the reference columns (QTH, Comment, Contest) that the richer tiers
// exist for. Operator is wider than Grid, so an in-place swap pushed the tier
// over its own width and the next build fell back to a poorer tier.
func assertEditorMultiOpRoundTrip(t *testing.T, w int, contest, swaps bool) {
	t.Helper()
	before := editorColumns(w, false, contest)
	club := editorColumns(w, true, contest)
	if swaps {
		if !hasColumn(club, "Operator") || hasColumn(club, "Grid") {
			t.Fatalf("width %d (contest=%v): the club table did not swap Grid for Operator: %v", w, contest, club)
		}
	}
	after := editorColumns(w, false, contest)
	if strings.Join(after, " ") != strings.Join(before, " ") {
		t.Errorf("width %d (contest=%v): the ordinary table changed after a club table was rendered\nbefore: %v\nafter:  %v",
			w, contest, before, after)
	}
}

func TestEditorMultiOpRoundTripKeepsTheColumns(t *testing.T) {
	for _, tier := range editorColTiers {
		if !hasColumn(tier.names, "Grid") {
			continue
		}
		// buildTable works on width-4; +4 lands exactly on this tier's total.
		assertEditorMultiOpRoundTrip(t, editorTierTotal(tier.names)+4, false,
			!hasColumn(tier.names, "Operator"))
	}
	for _, tier := range editorColTiersContest {
		if !hasColumn(tier.names, "Grid") {
			continue
		}
		assertEditorMultiOpRoundTrip(t, editorTierTotal(tier.names)+4, true,
			!hasColumn(tier.names, "Operator"))
	}
}

// TestEditorClubColumnsStillFitTheirTier is the reported symptom on its own: at
// a boundary width, switching to multi-operator and back must leave the ordinary
// table with QTH, Comment and Contest instead of dropping to a poorer tier.
func TestEditorClubColumnsStillFitTheirTier(t *testing.T) {
	var widths []int
	for _, tier := range editorColTiers {
		if hasColumn(tier.names, "Grid") {
			widths = append(widths, editorTierTotal(tier.names)+4)
		}
	}
	if len(widths) == 0 {
		t.Fatal("no tier carries a Grid column, the test proves nothing")
	}
	before := make([][]string, len(widths))
	for i, w := range widths {
		before[i] = editorColumns(w, false, false)
	}
	for _, w := range widths {
		editorColumns(w, true, false)
	}
	for i, w := range widths {
		after := editorColumns(w, false, false)
		if strings.Join(after, " ") != strings.Join(before[i], " ") {
			t.Errorf("width %d: the ordinary table lost columns after a club table was rendered\nbefore: %v\nafter:  %v",
				w, before[i], after)
		}
	}
}
