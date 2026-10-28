package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Dashboard statistics: totals vs rates
//
// The minute refresh used to recompute everything, including the seven DISTINCT
// counts over the event window (~300 ms at 100k contacts) that cannot change
// without a write. The totals are cached behind (database, logbook, event
// window, data revision) and the timer only re-counts the sliding windows and
// the elapsed time. These tests pin both halves: the cache must be reused when
// nothing changed, and it must be dropped by every change the ticket lists.
// =============================================================================

// fixedDashboardClock pins the statistics clock and returns a func that moves
// it, restoring the real clock on cleanup.
func fixedDashboardClock(t *testing.T, start time.Time) func(time.Time) {
	t.Helper()
	now := start
	dashboardNow = func() time.Time { return now }
	t.Cleanup(func() { dashboardNow = func() time.Time { return time.Now().UTC() } })
	return func(next time.Time) { now = next }
}

// logContactAtClock inserts a contact dated at the statistic clock — the date a
// real save would use while the test's clock is pinned.
func logContactAtClock(t *testing.T, m *Model, call string) int64 {
	t.Helper()
	id, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call:    call,
		QSODate: dashboardNow().Format("20060102"),
		TimeOn:  dashboardNow().Format("150405"),
		Band:    "20m",
		Mode:    "CW",
	})
	if err != nil {
		t.Fatalf("InsertQSO(%s): %v", call, err)
	}
	return id
}

// refreshStats runs one tick with the stats marked due — what the minute timer
// does — and returns the computation it started.
func refreshStats(t *testing.T, m *Model) (dashboardStatsMsg, bool) {
	t.Helper()
	m.http.statsDue = true
	return tick(t, m)
}

// TestDashboardTotalsAreReusedWhileNothingChanges is the point of the split:
// the second refresh with unchanged data must not re-read the window totals.
func TestDashboardTotalsAreReusedWhileNothingChanges(t *testing.T) {
	m := dashboardCycleModel(t)
	logContactAtClock(t, m, "SP9MOA")

	first, started := refreshStats(t, m)
	if !started {
		t.Fatal("the first refresh did not start a computation")
	}
	if !first.totalsRead {
		t.Error("the first refresh reused totals it never computed")
	}
	if first.stats.QSOsToday != 1 {
		t.Fatalf("first refresh counted %d contacts, want 1", first.stats.QSOsToday)
	}
	deliverStats(t, m, first)

	second, started := refreshStats(t, m)
	if !started {
		t.Fatal("the second refresh did not start a computation")
	}
	if second.totalsRead {
		t.Error("the idle refresh re-read the event-window totals — that is the ~300 ms per minute this split removes")
	}
	if second.stats != second.stats || second.stats.QSOsToday != 1 ||
		second.stats.Operators != first.stats.Operators ||
		second.stats.UniqueCalls != first.stats.UniqueCalls ||
		second.stats.DXCC != first.stats.DXCC ||
		second.stats.Grids != first.stats.Grids ||
		second.stats.Bands != first.stats.Bands ||
		second.stats.Modes != first.stats.Modes {
		t.Errorf("the reused totals differ from the ones they replace:\nfirst %+v\nsecond %+v",
			first.stats, second.stats)
	}
	// The rates are still recomputed from the database on every refresh, and the
	// elapsed time is derived from the cached stamp.
	if second.stats.Rate5m != first.stats.Rate5m || second.stats.Rate15m != first.stats.Rate15m ||
		second.stats.Rate60m != first.stats.Rate60m {
		t.Errorf("rates were not recomputed: first %+v, second %+v", first.stats, second.stats)
	}
	if second.stats.LastQSOAgoS < 0 {
		t.Errorf("elapsed time is negative: %d", second.stats.LastQSOAgoS)
	}
	deliverStats(t, m, second)
	if got := panelStats(m).QSOsToday; got != 1 {
		t.Errorf("panel shows %d contacts, want 1", got)
	}
}

// TestDashboardRatesSlideWithoutReReadingTotals: move the clock past the short
// window and the rates follow while the totals stay cached.
func TestDashboardRatesSlideWithoutReReadingTotals(t *testing.T) {
	base := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	moveClock := fixedDashboardClock(t, base)

	m := dashboardCycleModel(t)
	logContactAtClock(t, m, "SP9MOA")

	first, started := refreshStats(t, m)
	if !started {
		t.Fatal("the first refresh did not start a computation")
	}
	if first.stats.Rate5m != 1 || first.stats.Rate15m != 1 || first.stats.Rate60m != 1 {
		t.Fatalf("rates right after the contact: %+v, want 1/1/1", first.stats)
	}
	deliverStats(t, m, first)

	// Ten minutes later the contact is out of the 5-minute window but still in
	// the 15- and 60-minute ones, and it is still in the event window.
	moveClock(base.Add(10 * time.Minute))
	second, started := refreshStats(t, m)
	if !started {
		t.Fatal("the refresh after the clock moved did not start a computation")
	}
	if second.totalsRead {
		t.Error("moving the clock re-read the event-window totals")
	}
	if second.stats.Rate5m != 0 || second.stats.Rate15m != 1 || second.stats.Rate60m != 1 {
		t.Errorf("rates after ten minutes: %+v, want 0/1/1", second.stats)
	}
	if second.stats.QSOsToday != 1 {
		t.Errorf("event-window count changed with the clock: %d, want 1", second.stats.QSOsToday)
	}
	// The elapsed seconds follow the clock without a database read.
	if got, want := second.stats.LastQSOAgoS, 600+first.stats.LastQSOAgoS; got != want {
		t.Errorf("elapsed time = %d s, want %d (cached stamp + clock)", got, want)
	}
}

// TestDashboardTotalsFollowDataChanges is the invalidation gate: each change the
// ticket lists must force a fresh read of the totals, with the new numbers.
func TestDashboardTotalsFollowDataChanges(t *testing.T) {
	base := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	fixedDashboardClock(t, base)

	m := dashboardCycleModel(t)
	firstID := logContactAtClock(t, m, "SP9MOA")

	// Prime the cache.
	first, started := refreshStats(t, m)
	if !started {
		t.Fatal("the first refresh did not start a computation")
	}
	deliverStats(t, m, first)

	steps := []struct {
		name     string
		change   func()
		wantQSos int
	}{
		{
			name: "a manually logged contact",
			change: func() {
				logContactAtClock(t, m, "DL1AA")
				m.invalidateDashboardFlags() // what a save does
			},
			wantQSos: 2,
		},
		{
			name: "an edit that changes the data revision",
			change: func() {
				// An edit goes through refreshQSOS, which bumps the revision;
				// the totals follow the revision even without the dirty flag.
				m.qsoDataChanged()
			},
			wantQSos: 2,
		},
		{
			name: "enrichment filling in a field the totals count",
			change: func() {
				if err := store.UpdateQSOEnrichment(m.App.DB, firstID, store.EnrichmentData{
					Country: "Poland", GridSquare: "JO90",
				}); err != nil {
					t.Fatalf("UpdateQSOEnrichment: %v", err)
				}
				m.qsoDataChanged() // the enrichment path refreshes the list
			},
			wantQSos: 2,
		},
		{
			name: "a delete",
			change: func() {
				if err := store.DeleteQSO(m.App.DB, firstID); err != nil {
					t.Fatalf("DeleteQSO: %v", err)
				}
				m.invalidateDashboardFlags()
			},
			wantQSos: 1,
		},
		{
			name: "an import",
			change: func() {
				for i := 0; i < 3; i++ {
					logContactAtClock(t, m, fmt.Sprintf("SP%dIMP", i))
				}
				m.invalidateDashboardFlags()
			},
			wantQSos: 4,
		},
	}

	for _, step := range steps {
		step.change()
		msg, started := refreshStats(t, m)
		if !started {
			t.Fatalf("%s: no computation started", step.name)
		}
		if !msg.totalsRead {
			t.Errorf("%s: totals were reused across a data change", step.name)
		}
		if msg.stats.QSOsToday != step.wantQSos {
			t.Errorf("%s: counted %d contacts, want %d", step.name, msg.stats.QSOsToday, step.wantQSos)
		}
		deliverStats(t, m, msg)

		// And the cache is warm again: an immediate repeat is free.
		idle, started := refreshStats(t, m)
		if !started {
			t.Fatalf("%s: the follow-up refresh did not start", step.name)
		}
		if idle.totalsRead {
			t.Errorf("%s: the refresh right after the install re-read the totals", step.name)
		}
		deliverStats(t, m, idle)
	}
}

// TestDashboardTotalsFollowTheEventWindow: changing the configured event start
// asks a different question and must re-read the totals.
func TestDashboardTotalsFollowTheEventWindow(t *testing.T) {
	base := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	fixedDashboardClock(t, base)

	m := dashboardCycleModel(t)
	// One contact inside today, one from last week.
	logContactAtClock(t, m, "SP9MOA")
	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "OLD1AA", QSODate: "20261022", TimeOn: "120000", Band: "20m", Mode: "CW",
	}); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	first, started := refreshStats(t, m)
	if !started {
		t.Fatal("no computation started")
	}
	if first.stats.QSOsToday != 1 {
		t.Fatalf("today-only window counted %d contacts, want 1", first.stats.QSOsToday)
	}
	deliverStats(t, m, first)

	// Widen the window to the event start: the older contact counts now.
	m.App.Config.Integrations.HTTPServer.EventStart = "2026-10-20"
	second, started := refreshStats(t, m)
	if !started {
		t.Fatal("no computation started after the window change")
	}
	if !second.totalsRead {
		t.Error("a changed event window reused the totals of the old one")
	}
	if second.stats.QSOsToday != 2 {
		t.Errorf("event window counted %d contacts, want 2", second.stats.QSOsToday)
	}
}

// TestDashboardTotalsFollowTheUTCRollover: at midnight the "today" window moves
// to the next day, so the totals belong to a different set of contacts.
func TestDashboardTotalsFollowTheUTCRollover(t *testing.T) {
	moveClock := fixedDashboardClock(t, time.Date(2026, 10, 28, 23, 58, 0, 0, time.UTC))

	m := dashboardCycleModel(t)
	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "SP9MOA", QSODate: "20261028", TimeOn: "235000", Band: "20m", Mode: "CW",
	}); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	first, started := refreshStats(t, m)
	if !started {
		t.Fatal("no computation started")
	}
	if first.stats.QSOsToday != 1 {
		t.Fatalf("before midnight: %d contacts, want 1", first.stats.QSOsToday)
	}
	deliverStats(t, m, first)

	// Midnight UTC has passed. Even though nothing was written, the window is a
	// different one — the contact now sits in yesterday.
	moveClock(time.Date(2026, 10, 29, 0, 0, 30, 0, time.UTC))
	second, started := refreshStats(t, m)
	if !started {
		t.Fatal("no computation started after the rollover")
	}
	if !second.totalsRead {
		t.Error("the rollover reused the previous day's totals")
	}
	if second.stats.QSOsToday != 0 {
		t.Errorf("after midnight: %d contacts, want 0", second.stats.QSOsToday)
	}
}

// TestDashboardStaleResultCannotRevalidateTheCache: a computation that started
// before a change must not install its totals as current — the revision travels
// with the message, so the next refresh re-reads them.
func TestDashboardStaleResultCannotRevalidateTheCache(t *testing.T) {
	m := dashboardCycleModel(t)
	logContactAtClock(t, m, "SP9MOA")

	stale, started := refreshStats(t, m)
	if !started {
		t.Fatal("no computation started")
	}

	// A contact is logged while the result is on its way.
	logContactAtClock(t, m, "DL1AA")
	m.invalidateDashboardFlags()

	// The stale result lands first.
	deliverStats(t, m, stale)
	if got := panelStats(m).QSOsToday; got != 1 {
		t.Fatalf("the completed run shows %d contacts, want 1", got)
	}

	// The follow-up refresh must not trust the cache the stale result installed.
	fresh, started := refreshStats(t, m)
	if !started {
		t.Fatal("the follow-up refresh did not start")
	}
	if !fresh.totalsRead {
		t.Error("the refresh after a stale install reused the stale totals")
	}
	if fresh.stats.QSOsToday != 2 {
		t.Errorf("counted %d contacts, want 2", fresh.stats.QSOsToday)
	}
}
