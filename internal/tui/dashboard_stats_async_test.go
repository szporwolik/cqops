package tui

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// dashboardStatsTestModel returns a model with an online dashboard client, so
// the stats path can be exercised end to end without a real HTTP listener.
func dashboardStatsTestModel(t *testing.T) *Model {
	t.Helper()
	m := newLifecycleTestModel(t)
	m.http.client = dashboard.New("127.0.0.1", "0")
	m.http.online = true
	t.Cleanup(func() {
		m.http.client.Stop()
		m.http.statsDue = false
		m.http.statsInFlight = false
	})
	return m
}

// TestDashboardStatsPeriodicRefresh verifies the stats panel (with its sliding
// 5m/15m/1h rate windows) refreshes on a timer even when no QSO data changed —
// otherwise the rate badges freeze at the last event-time values. The numbers
// are computed off the update loop now: the push marks them stale, the tick
// starts the command, and the message installs the result.
func TestDashboardStatsPeriodicRefresh(t *testing.T) {
	m := dashboardStatsTestModel(t)
	today := time.Now().UTC().Format("20060102")
	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "SP9MOA", QSODate: today, TimeOn: "120000", Band: "20m", Mode: "CW",
	}); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	dashboardDataDirty = false
	t.Cleanup(func() { dashboardDataDirty = true })
	lastDashboardStatsPush = time.Now().Add(-2 * time.Minute)
	lastDashboardPushTick = m.tickCount - 10
	t.Cleanup(func() {
		lastDashboardStatsPush = time.Time{}
		lastDashboardPushTick = 0
	})

	m.pushDashboardState()
	if lastDashboardStatsPush.IsZero() {
		t.Fatal("stats push stamp not set")
	}
	if !m.http.statsDue {
		t.Fatal("the periodic refresh did not mark the stats stale")
	}

	cmd := m.dashboardStatsCmd()
	if cmd == nil {
		t.Fatal("no stats command issued for a stale panel")
	}
	if !m.http.statsInFlight {
		t.Error("the computation is not marked in flight")
	}
	// At most one computation at a time: while it runs, a second request is
	// refused and the stale flag is left for the next tick.
	m.http.statsDue = true
	if second := m.dashboardStatsCmd(); second != nil {
		t.Error("a second stats computation started while one was in flight")
	}
	if !m.http.statsDue {
		t.Error("the pending refresh was dropped while a computation was in flight")
	}

	msg, ok := cmd().(dashboardStatsMsg)
	if !ok {
		t.Fatalf("stats command produced %T, want dashboardStatsMsg", cmd())
	}
	if msg.err != nil {
		t.Fatalf("GetDashboardStats: %v", msg.err)
	}
	if msg.stats.QSOsToday != 1 {
		t.Errorf("computed QSOsToday = %d, want 1", msg.stats.QSOsToday)
	}

	m.applyDashboardStats(msg)
	if m.http.statsInFlight {
		t.Error("the in-flight flag was not cleared after the result arrived")
	}
	if got := m.http.client.State().Snapshot().Stats.QSOsToday; got != 1 {
		t.Errorf("dashboard stats QSOsToday = %d, want 1 (result not installed)", got)
	}
}

// TestDashboardStatsDropForeignLogbook: a computation that finishes after the
// operator switched logbooks must never repaint the new logbook's panel, and
// the new logbook has to be marked due so it gets its own numbers.
func TestDashboardStatsDropForeignLogbook(t *testing.T) {
	m := dashboardStatsTestModel(t)
	otherDB, err := store.InitDB(filepath.Join(t.TempDir(), "other.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer otherDB.Close()

	m.http.statsInFlight = true
	m.http.statsDue = false
	m.applyDashboardStats(dashboardStatsMsg{
		db:      otherDB,
		logbook: "other",
		stats:   store.DashboardStats{QSOsToday: 99},
	})

	if m.http.statsInFlight {
		t.Error("the in-flight flag was not cleared for the discarded result")
	}
	if !m.http.statsDue {
		t.Error("the current logbook was not marked due after dropping the result")
	}
	if got := m.http.client.State().Snapshot().Stats.QSOsToday; got == 99 {
		t.Error("a previous logbook's stats were installed")
	}
}

// TestDashboardStatsErrorClearsInFlight: a failed query must not wedge the
// panel by leaving the computation marked as running forever.
func TestDashboardStatsErrorClearsInFlight(t *testing.T) {
	m := dashboardStatsTestModel(t)
	m.http.statsInFlight = true
	m.applyDashboardStats(dashboardStatsMsg{db: m.App.DB, logbook: m.App.LogbookName, err: errors.New("boom")})
	if m.http.statsInFlight {
		t.Error("a failed computation left the in-flight flag set")
	}
}
