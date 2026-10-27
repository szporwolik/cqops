package tui

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ctybig"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Periodic DXCC backfill — runs off the update loop
// =============================================================================

// dxccBackfillTestModel is a model with a Big CTY table and the quiet-but-online
// tick state the backfill needs: the periodic health/reference-data work is
// already up to date, so the only command a tick starts is the backfill pass.
func dxccBackfillTestModel(t *testing.T) *Model {
	t.Helper()
	m := newLifecycleTestModel(t)
	bdb, err := ctybig.ParseCSV(strings.NewReader(
		"SP,Poland,269,EU,15,28,52.00,-20.00,-1.0,HF SN SO SP SQ SR 3Z;\n" +
			"DL,Germany,230,EU,14,28,51.00,-10.00,-1.0,DA DB DC DD DF DG DH DJ DK DL DM DO DP DQ DR;\n"))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	m.App.BigCTY = bdb
	m.inetOnline = true
	m.inetConfirmed = true
	m.lastDataCheck = time.Now()
	dashboardDataDirty = false
	t.Cleanup(func() { dashboardDataDirty = true })
	return m
}

// insertContactWithoutEntity logs a contact the way an import from before Big
// CTY was available would leave it: no DXCC entity.
func insertContactWithoutEntity(t *testing.T, db *sql.DB, call string) {
	t.Helper()
	if _, err := store.InsertQSO(db, &qso.QSO{
		Call: call, Band: "20m", Mode: "CW", GridSquare: "JO90AA",
		QSODate: "20261027", TimeOn: "120000", RSTSent: "59", RSTRcvd: "59",
	}); err != nil {
		t.Fatalf("InsertQSO(%s): %v", call, err)
	}
}

// missingEntities counts the contacts that still have no DXCC entity.
func missingEntities(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos WHERE COALESCE(dxcc,'') = ''`).Scan(&n); err != nil {
		t.Fatalf("count missing dxcc: %v", err)
	}
	return n
}

// TestPeriodicDXCCBackfillRunsOffTheUpdateLoop: the pass is a write transaction
// over a batch of contacts (395 ms for 50 rows in a 10k log, ~6 s in a 100k one
// before the index work was narrowed to the DXCC aggregates). Running it inside
// handleTick meant every keystroke queued behind it.
//
// The tick must start the pass and return; the rows change only when the result
// is delivered; a second pass cannot start meanwhile; and installing the result
// refreshes what the pass made stale.
func TestPeriodicDXCCBackfillRunsOffTheUpdateLoop(t *testing.T) {
	m := dxccBackfillTestModel(t)
	insertContactWithoutEntity(t, m.App.DB, "SP9MOA")
	insertContactWithoutEntity(t, m.App.DB, "DL1AA")

	m.tickCount = 29 // the next tick is the periodic one
	_, cmd := m.Update(tickMsg(time.Now()))

	if !m.backfill.inFlight {
		t.Fatal("the periodic tick did not start a backfill pass")
	}
	if n := missingEntities(t, m.App.DB); n != 2 {
		t.Fatalf("%d contacts already changed by the end of the tick — the pass ran inline", 2-n)
	}
	if c := m.dxccBackfillCmd(dxccBackfillBatch); c != nil {
		t.Error("a second pass started while one was still in flight")
	}

	msg := awaitTickMessage[dxccBackfillMsg](t, cmd, "DXCC backfill pass")
	if msg.err != nil {
		t.Fatalf("backfill: %v", msg.err)
	}
	if msg.count != 2 {
		t.Errorf("the pass resolved %d contacts, want 2", msg.count)
	}
	if msg.db != m.App.DB || msg.logbook != m.App.LogbookName {
		t.Errorf("the result names %q/%p, want %q/%p", msg.logbook, msg.db, m.App.LogbookName, m.App.DB)
	}

	// Prime the caches the pass must invalidate.
	m.rc.workedSummarySig = "stale"
	m.rc.logStatsSig = "stale"
	dashboardDataDirty = false
	_, queued := m.Update(msg)

	if m.backfill.inFlight {
		t.Error("the finished pass is still marked as running")
	}
	if n := missingEntities(t, m.App.DB); n != 0 {
		t.Errorf("%d contacts still without an entity after the pass", n)
	}
	info, err := store.WorkedDXCC(m.App.DB, "269", "", "")
	if err != nil {
		t.Fatalf("WorkedDXCC: %v", err)
	}
	if info.Count != 1 {
		t.Errorf("the worked index counts %d Polish contacts, want 1", info.Count)
	}
	if !dashboardDataDirty {
		t.Error("the dashboard was not marked dirty after a backfill")
	}
	if m.rc.workedSummarySig != "" || m.rc.logStatsSig != "" {
		t.Error("the worked/statistics panels kept their pre-backfill signature")
	}
	// The update loop turns the refresh request into the recent-QSOs reload, so
	// on a QSO screen the flag is already consumed by the time Update returns.
	if queued == nil {
		t.Error("the QSO list was not reloaded after the backfill")
	}
}

// TestDXCCBackfillResultIsDroppedAfterLogbookSwitch: the pass finished against a
// database the operator has already replaced. Its rows belong to the old
// logbook, so the result must not clear the new logbook's caches or mark its
// dashboard dirty — but the pass itself completed on the database it started on.
func TestDXCCBackfillResultIsDroppedAfterLogbookSwitch(t *testing.T) {
	m := dxccBackfillTestModel(t)
	insertContactWithoutEntity(t, m.App.DB, "SP9MOA")
	oldDB := m.App.DB

	m.tickCount = 29
	_, cmd := m.Update(tickMsg(time.Now()))
	msg := awaitTickMessage[dxccBackfillMsg](t, cmd, "DXCC backfill pass")
	// The pass did its work on the database it started on.
	if n := missingEntities(t, oldDB); n != 0 {
		t.Fatalf("%d contacts in the running pass's logbook still lack an entity", n)
	}

	m.App.Config.Logbooks["b"] = config.Logbook{
		Station:      config.Station{Callsign: "SP9MOA/P", Grid: "JO90", RigName: "default"},
		Wavelog:      &config.WavelogConfig{},
		DatabasePath: filepath.Join(t.TempDir(), "second.db"),
	}
	if err := m.App.SwitchLogbook("b"); err != nil {
		t.Fatalf("SwitchLogbook: %v", err)
	}

	m.rc.workedSummarySig = "current logbook"
	m.rc.logStatsSig = "current logbook"
	m.needRefresh = false
	dashboardDataDirty = false
	m.Update(msg)

	if m.backfill.inFlight {
		t.Error("the dropped result left the backfill marked as running")
	}
	if m.rc.workedSummarySig != "current logbook" || m.rc.logStatsSig != "current logbook" {
		t.Error("a pass from the previous logbook cleared the current logbook's caches")
	}
	if m.needRefresh {
		t.Error("a pass from the previous logbook asked the current logbook's list to refresh")
	}
	if dashboardDataDirty {
		t.Error("a pass from the previous logbook marked the dashboard dirty")
	}
}

// TestDXCCBackfillErrorDoesNotWedgeThePasses: a pass that fails (here: the
// database went away) must clear its running marker, or the backfill never
// runs again for the rest of the session.
func TestDXCCBackfillErrorDoesNotWedgeThePasses(t *testing.T) {
	m := dxccBackfillTestModel(t)
	insertContactWithoutEntity(t, m.App.DB, "SP9MOA")
	working := m.App.DB

	broken, err := store.InitDB(filepath.Join(t.TempDir(), "broken.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := broken.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	m.App.DB = broken

	m.tickCount = 29
	_, cmd := m.Update(tickMsg(time.Now()))
	if !m.backfill.inFlight {
		t.Fatal("the tick did not start a backfill pass")
	}
	msg := awaitTickMessage[dxccBackfillMsg](t, cmd, "DXCC backfill pass")
	if msg.err == nil {
		t.Fatal("the pass reported success against a closed database")
	}
	m.Update(msg)
	if m.backfill.inFlight {
		t.Error("a failed pass left the backfill marked as running")
	}

	// The database comes back: the next periodic tick must start a new pass.
	m.App.DB = working
	m.tickCount = 29
	_, cmd = m.Update(tickMsg(time.Now()))
	if !m.backfill.inFlight {
		t.Fatal("the failed pass blocked every later pass")
	}
	msg = awaitTickMessage[dxccBackfillMsg](t, cmd, "DXCC backfill pass")
	if msg.err != nil {
		t.Fatalf("backfill after recovery: %v", msg.err)
	}
	m.Update(msg)
	if n := missingEntities(t, working); n != 0 {
		t.Errorf("%d contacts still without an entity after recovery", n)
	}
}
