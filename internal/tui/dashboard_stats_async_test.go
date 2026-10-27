package tui

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
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

// =============================================================================
// Whole cycle through Model.Update: tick → command → message → tick
// =============================================================================

// dashboardCycleModel returns a model whose ticks behave like a running
// installation: the dashboard server is enabled and connected (a disabled one
// would be torn down by the tick itself), the machine is online with its
// reference data already up to date, and no stats refresh is pending — so the
// only command a tick starts is the computation under test.
func dashboardCycleModel(t *testing.T) *Model {
	t.Helper()
	m := dashboardStatsTestModel(t)
	m.App.Config.Integrations.HTTPServer.Enabled = true
	m.inetOnline = true
	m.inetConfirmed = true
	m.lastDataCheck = time.Now()
	// Production starts dirty (the panel has never been filled) and the
	// periodic rate refresh is a minute out. Both are package state, so they
	// are restored for the tests that follow.
	dashboardDataDirty = true
	lastDashboardPushTick = m.tickCount - 10
	lastDashboardStatsPush = time.Now()
	t.Cleanup(func() {
		dashboardDataDirty = true
		lastDashboardPushTick = 0
		lastDashboardStatsPush = time.Time{}
	})
	return m
}

// insertContact logs one contact that the stats window counts.
func insertContact(t *testing.T, m *Model, call string) {
	t.Helper()
	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call:    call,
		QSODate: time.Now().UTC().Format("20060102"),
		TimeOn:  "120000",
		Band:    "20m",
		Mode:    "CW",
	}); err != nil {
		t.Fatalf("InsertQSO(%s): %v", call, err)
	}
}

// panelStats reads what the dashboard website would show.
func panelStats(m *Model) dashboard.Stats {
	return m.http.client.State().Snapshot().Stats
}

// tick drives ONE real tick through Model.Update. When the tick starts a stats
// computation the command is run and its message returned, so the caller
// decides when that result reaches the panel — the ordering a real runtime
// cannot control, and the reason a stale result can overwrite a fresh one.
//
// The boolean reports whether a computation was started at all.
func tick(t *testing.T, m *Model) (dashboardStatsMsg, bool) {
	t.Helper()
	// pushDashboardState is throttled to every 2 ticks (~2 s of wall clock);
	// a test tick is one logical second, so the throttle is cleared to keep
	// the cycle deterministic instead of skipping every second push.
	lastDashboardPushTick = m.tickCount - 10
	wasInFlight := m.http.statsInFlight
	_, cmd := m.Update(tickMsg(time.Now()))
	if wasInFlight || !m.http.statsInFlight {
		return dashboardStatsMsg{}, false
	}
	return awaitStatsMsg(t, cmd), true
}

// awaitStatsMsg runs the commands a tick issued and returns the stats result.
// The tick also arms the tick timer (and, in a busier model, read-only view
// fetches); their messages are discarded because they do not touch the stats
// panel.
func awaitStatsMsg(t *testing.T, cmd tea.Cmd) dashboardStatsMsg {
	t.Helper()
	msgs := make(chan tea.Msg, 32)
	go pumpCommands(cmd, msgs)
	deadline := time.After(30 * time.Second)
	for {
		select {
		case msg := <-msgs:
			if s, ok := msg.(dashboardStatsMsg); ok {
				return s
			}
		case <-deadline:
			t.Fatal("the tick marked the stats in flight but no computation finished")
		}
	}
}

// pumpCommands executes a command tree the way the runtime does: a batch is
// expanded and each leaf runs on its own goroutine.
func pumpCommands(cmd tea.Cmd, out chan<- tea.Msg) {
	if cmd == nil {
		return
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		select {
		case out <- msg:
		default: // the test already has what it needs
		}
		return
	}
	for _, sub := range batch {
		go pumpCommands(sub, out)
	}
}

// deliverStats hands a finished computation back to the update loop, exactly as
// the Bubble Tea runtime does when a command returns.
func deliverStats(t *testing.T, m *Model, msg dashboardStatsMsg) {
	t.Helper()
	m.Update(msg)
}

// TestDashboardStatsSecondRunAfterAChangeDuringComputation covers the cycle a
// frozen panel hides: while the first computation is still on its way, the
// operator logs a contact. That contact is not in the result that arrives, so
// the refresh it triggered has to survive the install and run again — through
// real ticks and real messages, not by calling the install directly.
func TestDashboardStatsSecondRunAfterAChangeDuringComputation(t *testing.T) {
	m := dashboardCycleModel(t)
	insertContact(t, m, "SP9MOA")

	first, started := tick(t, m)
	if !started {
		t.Fatal("the first tick did not start a stats computation")
	}
	if first.err != nil {
		t.Fatalf("GetDashboardStats: %v", first.err)
	}
	if first.stats.QSOsToday != 1 {
		t.Fatalf("first computation counted %d contacts, want 1", first.stats.QSOsToday)
	}

	// The operator logs a contact while the first result is in flight.
	insertContact(t, m, "DL1AA")
	m.invalidateDashboardFlags() // what a save does: every panel is stale now

	// The tick that sees the change must neither start a competing
	// computation nor drop the pending refresh.
	if _, started := tick(t, m); started {
		t.Error("a second computation started while the first was still in flight")
	}
	if !m.http.statsDue {
		t.Fatal("the pending refresh was dropped while a computation was in flight")
	}

	// The stale result arrives and is installed. The panel is one contact
	// behind — and the pending refresh has to survive the install.
	deliverStats(t, m, first)
	if got := panelStats(m).QSOsToday; got != 1 {
		t.Fatalf("panel shows %d contacts after the completed run, want 1", got)
	}
	if !m.http.statsDue {
		t.Error("installing the result cleared the pending refresh")
	}

	second, started := tick(t, m)
	if !started {
		t.Fatal("the change during the computation never started a second run")
	}
	if second.err != nil {
		t.Fatalf("GetDashboardStats: %v", second.err)
	}
	if second.stats.QSOsToday != 2 {
		t.Errorf("second computation counted %d contacts, want 2", second.stats.QSOsToday)
	}
	deliverStats(t, m, second)
	if got := panelStats(m).QSOsToday; got != 2 {
		t.Errorf("panel shows %d contacts, want 2 (the second run never reached it)", got)
	}
	if m.http.statsDue {
		t.Error("the refresh stayed pending after a computation was installed")
	}
}

// TestDashboardStatsLogbookSwitchDropsTheOldResult: a computation started in
// one logbook finishes after the operator switched to another. Its numbers
// describe a different database, so they must never reach the panel — and the
// new logbook must still get its own numbers.
func TestDashboardStatsLogbookSwitchDropsTheOldResult(t *testing.T) {
	m := dashboardCycleModel(t)
	insertContact(t, m, "SP9MOA")

	// A second logbook with its own database file inside the test directory.
	m.App.Config.Logbooks["b"] = config.Logbook{
		Station:      config.Station{Callsign: "SP9MOA/P", Grid: "JO90", RigName: "default"},
		Wavelog:      &config.WavelogConfig{},
		DatabasePath: filepath.Join(t.TempDir(), "second.db"),
	}

	first, started := tick(t, m)
	if !started {
		t.Fatal("the first tick did not start a stats computation")
	}
	if first.stats.QSOsToday != 1 {
		t.Fatalf("first computation counted %d contacts, want 1", first.stats.QSOsToday)
	}

	// The operator switches logbooks while that result is in flight.
	if err := m.App.SwitchLogbook("b"); err != nil {
		t.Fatalf("SwitchLogbook: %v", err)
	}
	m.handleLogbookSwitched() // the bookkeeping every switch runs

	deliverStats(t, m, first)
	if got := panelStats(m).QSOsToday; got != 0 {
		t.Errorf("panel shows %d contacts after the switch, want 0 — the old logbook's stats were installed", got)
	}

	// The refresh must publish the NEW logbook's numbers.
	for _, call := range []string{"DL1AA", "DL2BB", "DL3CC"} {
		insertContact(t, m, call)
	}
	second, started := tick(t, m)
	if !started {
		t.Fatal("the new logbook never got a stats computation")
	}
	if second.err != nil {
		t.Fatalf("GetDashboardStats: %v", second.err)
	}
	if second.stats.QSOsToday != 3 {
		t.Errorf("new logbook's computation counted %d contacts, want 3", second.stats.QSOsToday)
	}
	deliverStats(t, m, second)
	if got := panelStats(m).QSOsToday; got != 3 {
		t.Errorf("panel shows %d contacts, want 3 (the new logbook never reached it)", got)
	}
}

// TestDashboardStatsErrorDoesNotStallTheCycle: a query that fails (here: the
// database went away) must not wedge the panel. The next scheduled refresh has
// to run and publish, not sit behind a computation that never finished.
func TestDashboardStatsErrorDoesNotStallTheCycle(t *testing.T) {
	m := dashboardCycleModel(t)
	insertContact(t, m, "SP9MOA")

	// Age the periodic refresh (60 s) so the next tick asks for new numbers
	// without needing QSO data to change — no panel push touches the broken
	// database on the way.
	dashboardDataDirty = false
	lastDashboardStatsPush = time.Now().Add(-2 * dashboardStatsRefreshInterval)

	working := m.App.DB
	broken, err := store.InitDB(filepath.Join(t.TempDir(), "broken.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := broken.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	m.App.DB = broken

	first, started := tick(t, m)
	if !started {
		t.Fatal("the tick did not start a stats computation")
	}
	if first.err == nil {
		t.Fatal("the computation succeeded against a closed database")
	}
	deliverStats(t, m, first)
	if m.http.statsInFlight {
		t.Error("a failed computation left the panel marked as busy")
	}

	// The database comes back; the next refresh must compute and publish.
	m.App.DB = working
	insertContact(t, m, "DL1AA")
	lastDashboardStatsPush = time.Now().Add(-2 * dashboardStatsRefreshInterval)
	second, started := tick(t, m)
	if !started {
		t.Fatal("the failed computation blocked the next refresh")
	}
	if second.err != nil {
		t.Fatalf("GetDashboardStats: %v", second.err)
	}
	if second.stats.QSOsToday != 2 {
		t.Errorf("recovery computation counted %d contacts, want 2", second.stats.QSOsToday)
	}
	deliverStats(t, m, second)
	if got := panelStats(m).QSOsToday; got != 2 {
		t.Errorf("panel shows %d contacts after recovery, want 2", got)
	}
}
