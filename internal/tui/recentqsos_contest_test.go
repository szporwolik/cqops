package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

func contestQSO() []qso.QSO {
	return []qso.QSO{{
		ID: 1, Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "CW", RSTSent: "599", RSTRcvd: "599",
		ExchSent: "001", ExchRcvd: "123",
	}}
}

// TestRecentQSOsContestShowsExchange: with a contest active the recent-QSO
// table shows the sent/received exchange at ordinary widths. It used to need
// a 220-column terminal, and narrow screens showed no exchange at all.
func TestRecentQSOsContestShowsExchange(t *testing.T) {
	for _, w := range []int{80, 120, 160, 200} {
		r := NewRecentQSOs(contestQSO())
		r.SetSize(w, 12)
		r.SetContest(true)
		view := r.View()
		for _, want := range []string{"Snt", "Rcv", "001", "123"} {
			if !strings.Contains(view, want) {
				t.Errorf("width %d: %q missing from the contest table:\n%s", w, want, view)
			}
		}
	}
}

// TestRecentQSOsNonContestHidesExchange keeps the normal table unchanged.
func TestRecentQSOsNonContestHidesExchange(t *testing.T) {
	r := NewRecentQSOs(contestQSO())
	r.SetSize(160, 12)
	view := r.View()
	if strings.Contains(view, "Snt") || strings.Contains(view, "Rcv") {
		t.Errorf("exchange headers shown without a contest:\n%s", view)
	}
}

// TestRecentQSOTiersContestAreOrderedByWidth: the picker takes the last tier
// that fits, so the contest tiers must grow monotonically — otherwise a
// richer tier can never be selected.
func TestRecentQSOTiersContestAreOrderedByWidth(t *testing.T) {
	prev := 0
	for i, tier := range qsoColTiersContest {
		if tier.maxW <= prev {
			t.Fatalf("contest tier %d total %d is not wider than the previous %d", i, tier.maxW, prev)
		}
		prev = tier.maxW
	}
	// Every tier must keep the columns the table is read by.
	for i, tier := range qsoColTiersContest {
		for _, want := range []string{"Date", "Time", "Call", "Mode", "RSTs", "RSTr"} {
			found := false
			for _, n := range tier.names {
				if n == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("contest tier %d is missing %s: %v", i, want, tier.names)
			}
		}
	}
}
