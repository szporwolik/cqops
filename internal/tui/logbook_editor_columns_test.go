package tui

import (
	"testing"
)

// editorTierTotal is the width a tier occupies once rendered: the sum of its
// column minimums plus the per-cell padding bubbles/table adds. buildTable works
// on width-4, so a terminal width of editorTierTotal+4 lands exactly on it.
func editorTierTotal(names []string) int {
	total := 0
	for _, n := range names {
		total += qsoAllCols[n].minWidth
	}
	return tierRenderWidth(total, len(names))
}

func hasColumn(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// TestEditorContestColumnsShowExchange pins the contest column priority: with
// a contest active the sent/received exchange must be visible on ordinary
// terminal widths, not only at the widest tier (it used to need ~160 columns).
func TestEditorContestColumnsShowExchange(t *testing.T) {
	// The narrowest tier that carries the exchange, plus roomier terminals.
	narrowest := 0
	for _, tier := range editorColTiersContest {
		if hasColumn(tier.names, "ExchSent") {
			narrowest = editorTierTotal(tier.names) + 4
			break
		}
	}
	if narrowest == 0 {
		t.Fatal("no contest tier carries the exchange")
	}
	widths := []int{narrowest, narrowest + 4, 120, 160, 200}
	for _, w := range widths {
		le := &LogbookEditor{width: w, height: 40}
		le.SetContestMode(true)
		le.buildTable()
		names := le.formattedNames
		if !hasColumn(names, "ExchSent") || !hasColumn(names, "ExchRcvd") {
			t.Errorf("width %d: exchange columns missing (got %v)", w, names)
		}
		// The exchange must come before the reference-style columns.
		sent := -1
		for i, n := range names {
			if n == "ExchSent" {
				sent = i
			}
		}
		for i, n := range names {
			if (n == "Comment" || n == "QTH" || n == "Name") && i < sent {
				t.Errorf("width %d: %s placed before the exchange (%v)", w, n, names)
			}
		}
	}
}

// TestEditorNonContestColumnsHaveNoExchange keeps the ordinary logbook view
// unchanged: exchange columns are a contest-mode feature.
func TestEditorNonContestColumnsHaveNoExchange(t *testing.T) {
	for _, w := range []int{120, 200} {
		le := &LogbookEditor{width: w, height: 40}
		le.buildTable()
		if hasColumn(le.formattedNames, "ExchSent") || hasColumn(le.formattedNames, "ExchRcvd") {
			t.Errorf("width %d: exchange shown without a contest: %v", w, le.formattedNames)
		}
	}
}

// TestContestTiersAreOrderedByWidth: buildTable picks the last tier whose
// total fits, so the contest tiers must grow monotonically — otherwise a
// richer tier can never be selected.
func TestContestTiersAreOrderedByWidth(t *testing.T) {
	prev := 0
	for i, tier := range editorColTiersContest {
		total := editorTierTotal(tier.names)
		if total <= prev {
			t.Fatalf("tier %d total %d is not wider than the previous %d", i, total, prev)
		}
		prev = total
	}
	// And the very narrow fallback keeps a usable core.
	for _, want := range []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr"} {
		if !hasColumn(editorColTiersContest[0].names, want) {
			t.Errorf("narrowest contest tier is missing %s", want)
		}
	}
}

// TestEditorContestMultiOpDoesNotDuplicateOperator: the widest contest tiers
// already carry Operator, so the multi-op Grid→Operator swap must be skipped
// there instead of producing two Operator columns.
func TestEditorContestMultiOpDoesNotDuplicateOperator(t *testing.T) {
	le := &LogbookEditor{width: 220, height: 40}
	le.SetContestMode(true)
	le.SetMultiOp(true)
	le.buildTable()
	count := 0
	for _, n := range le.formattedNames {
		if n == "Operator" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Operator columns = %d, want 1: %v", count, le.formattedNames)
	}
}

// TestEditorContestModeInvalidatesTableCache: switching the mode must force a
// rebuild, otherwise the columns stay those of the previous mode.
func TestEditorContestModeInvalidatesTableCache(t *testing.T) {
	le := &LogbookEditor{width: 120, height: 40}
	le.buildTable()
	if hasColumn(le.formattedNames, "ExchSent") {
		t.Fatal("precondition: non-contest table already has exchange")
	}
	le.SetContestMode(true)
	if le.formattedNames != nil || le.formattedRows != nil {
		t.Fatal("SetContestMode left the formatted column cache in place")
	}
	le.buildTable()
	if !hasColumn(le.formattedNames, "ExchSent") {
		t.Fatalf("cache was not rebuilt with contest columns: %v", le.formattedNames)
	}
}
