package tui

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Editor upload tests — pure logic, no real HTTP/network
// =============================================================================

// newTestEditor creates a minimal LogbookEditor for upload logic tests.
// The DB is nil because doBatchUpload/doUploadToWavelog pure-logic paths
// don't touch the database directly (only the returned command closures do).
func newTestEditor(wlURL, wlKey, wlStationID, logOp, logGrid string) *LogbookEditor {
	return NewLogbookEditor(LogbookEditorConfig{DB: nil, WLURL: wlURL, WLKey: wlKey, WLStationID: wlStationID, WLLastFetchedID: 0, StationOperator: logOp, StationGrid: logGrid, StationCall: ""})
}

// execCmd executes a tea.Cmd and returns the message. Returns nil if cmd is nil.
func execCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// runUploadPrep drives the async upload-preparation flow: runs the prep
// command, feeds the uploadPrepMsg through Update, and returns the updated
// editor plus the follow-up command (nil on the mismatch dialog path).
func runUploadPrep(t *testing.T, le *LogbookEditor) (*LogbookEditor, tea.Cmd) {
	t.Helper()
	cmd := le.doBatchUpload()
	if cmd == nil {
		t.Fatal("doBatchUpload returned nil command")
	}
	msg := execCmd(cmd)
	prep, ok := msg.(uploadPrepMsg)
	if !ok {
		t.Fatalf("expected uploadPrepMsg, got %T", msg)
	}
	updated, next := le.Update(prep)
	return updated.(*LogbookEditor), next
}

// prepAndRunUpload runs the full preparation flow and executes the resulting
// upload command, returning the final editorMsg.
func prepAndRunUpload(t *testing.T, le *LogbookEditor) editorMsg {
	t.Helper()
	_, next := runUploadPrep(t, le)
	if next == nil {
		t.Fatal("prep produced no follow-up command")
	}
	msg := execCmd(next)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	return em
}

// =============================================================================
// doUploadToWavelog tests
// =============================================================================

func TestDoUploadToWavelog_MissingConfig(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.editing = &qso.QSO{ID: 1, Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501"}

	cmd := le.doUploadToWavelog()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.wlOK {
		t.Error("wlOK should be false when Wavelog not configured")
	}
	if em.err == nil {
		t.Error("error should be non-nil when Wavelog not configured")
	}
}

func TestDoUploadToWavelog_MissingRequiredFields(t *testing.T) {
	le := newTestEditor("https://log.example.com", "key123", "1", "", "")
	// QSO with no band, mode, or date.
	le.editing = &qso.QSO{ID: 2, Call: "SP9MOA"}
	// Need to populate the form fields so readEditForm works.
	le.fields[qefCall].SetValue("SP9MOA")

	cmd := le.doUploadToWavelog()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.wlOK {
		t.Error("wlOK should be false when required fields are missing")
	}
	if em.wlQSOID != 2 {
		t.Errorf("wlQSOID = %d; want 2", em.wlQSOID)
	}
}

// =============================================================================
// doBatchUpload tests
// =============================================================================

func TestDoBatchUpload_AllAlreadySent(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "A", WavelogID: 1},
		{ID: 2, Call: "B", WavelogID: 1},
	}

	em := prepAndRunUpload(t, le)
	if !em.wlOK {
		t.Error("wlOK should be true when all QSOs already sent")
	}
	if em.wlCall != "all sent" {
		t.Errorf("wlCall = %q; want 'all sent'", em.wlCall)
	}
}

func TestDoBatchUpload_EmptyQSOList(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.qsos = nil

	em := prepAndRunUpload(t, le)
	if !em.wlOK {
		t.Error("wlOK should be true for empty QSO list")
	}
	if em.wlCall != "all sent" {
		t.Errorf("wlCall = %q; want 'all sent'", em.wlCall)
	}
}

// TestUploadPrepResultRejectedByReplacedEditor verifies the cross-logbook
// guard: preparation started in editor A must never be applied by editor B
// (e.g. after a logbook switch). The result is discarded without any upload
// and without touching B's state.
func TestUploadPrepResultRejectedByReplacedEditor(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Errorf("replaced-editor result must not upload (got %s %s)", r.Method, r.URL.Path)
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	// Editor A prepares a batch against its own database.
	leA := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "OPA", "JO90")
	q := &qso.QSO{Call: "SP9AAA", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"}
	id := insertTestQSO(t, leA.db, q)
	q.ID = id

	cmd := leA.doBatchUpload()
	if cmd == nil {
		t.Fatal("doBatchUpload returned nil command")
	}
	prep, ok := execCmd(cmd).(uploadPrepMsg)
	if !ok {
		t.Fatalf("expected uploadPrepMsg, got %T", execCmd(cmd))
	}
	if len(prep.unsent) != 1 || prep.err != nil {
		t.Fatalf("prep = unsent:%d err:%v, want 1 unsent row", len(prep.unsent), prep.err)
	}

	// Editor B: a new instance (different generation) with its own database.
	leB := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "OPB", "JO90")
	if leA.gen == leB.gen {
		t.Fatal("editor generations must differ")
	}

	updated, next := leB.Update(prep)
	leB = updated.(*LogbookEditor)
	if next != nil {
		t.Fatal("replaced editor must not produce a follow-up upload command")
	}
	if leB.wlSkipped != 0 || leB.mode != edModeList {
		t.Errorf("replaced editor state changed: skipped=%d mode=%v", leB.wlSkipped, leB.mode)
	}
	if requests != 0 {
		t.Errorf("Wavelog received %d request(s)", requests)
	}
}

func TestDoBatchUpload_SkipsMissingFields(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "A", Band: "", Mode: "SSB", QSODate: "20240501"},
		{ID: 2, Call: "B", Band: "20m", Mode: "", QSODate: "20240501"},
		{ID: 3, Call: "C", Band: "20m", Mode: "SSB", QSODate: ""},
	}

	em := prepAndRunUpload(t, le)

	// All three should be skipped (missing fields) → all sent.
	if !em.wlOK {
		t.Error("wlOK should be true (all skipped → all sent)")
	}
	if le.wlSkipped != 3 {
		t.Errorf("wlSkipped = %d; want 3", le.wlSkipped)
	}
	if le.wlSkipDetail == "" {
		t.Error("wlSkipDetail should be populated when QSOs are skipped")
	}
}

func TestDoBatchUpload_SkipDetailSingle(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "SP9MOA", Band: "", Mode: "SSB", QSODate: "20240501"},
	}

	le2, _ := runUploadPrep(t, le)

	if le2.wlSkipped != 1 {
		t.Errorf("wlSkipped = %d; want 1", le2.wlSkipped)
	}
	// Single skip should mention callsign and date.
	if le2.wlSkipDetail == "" {
		t.Error("wlSkipDetail should be set for single skipped QSO")
	}
}

func TestDoBatchUpload_DetectsMismatch(t *testing.T) {
	le := newTestEditor("", "", "", "Szymon", "KO00ca")
	le.qsos = []qso.QSO{
		{
			ID:              10,
			Call:            "SP9MOA",
			Band:            "20m",
			Mode:            "SSB",
			QSODate:         "20240501",
			StationCallsign: "SP9MOA",
			Operator:        "WrongOp", // mismatch
			MyGridSquare:    "XX00xx",  // mismatch

		},
	}

	le2, next := runUploadPrep(t, le)
	// Mismatch → confirm-normalize dialog, no upload command.
	if next != nil {
		t.Errorf("expected nil command when mismatches detected, got %T", next)
	}

	if le2.mode != edModeConfirmNormalize {
		t.Errorf("mode = %v; want edModeConfirmNormalize", le2.mode)
	}
	if len(le2.mismatchQSOs) != 1 {
		t.Errorf("mismatchQSOs length = %d; want 1", len(le2.mismatchQSOs))
	}
	if len(le2.mismatchFields) < 2 {
		t.Errorf("mismatchFields should contain operator and grid, got %v", le2.mismatchFields)
	}

	// Verify fields list contains expected mismatches.
	hasOp := false
	hasGrid := false
	for _, f := range le2.mismatchFields {
		if f == "operator" {
			hasOp = true
		}
		if f == "grid" {
			hasGrid = true
		}
	}
	if !hasOp {
		t.Error("mismatchFields should contain 'operator'")
	}
	if !hasGrid {
		t.Error("mismatchFields should contain 'grid'")
	}
}

func TestDoBatchUpload_NoMismatchWhenDefaultsEmpty(t *testing.T) {
	// When logStationOp and logStationGrid are empty, no mismatches are flagged.
	le := newTestEditor("", "", "", "", "")
	le.qsos = []qso.QSO{
		{
			ID:              10,
			Call:            "SP9MOA",
			Band:            "20m",
			Mode:            "SSB",
			QSODate:         "20240501",
			StationCallsign: "",
			Operator:        "Anyone",
			MyGridSquare:    "XX00xx",
		},
	}

	_, next := runUploadPrep(t, le)
	if next == nil {
		t.Error("prep should produce an upload command when no mismatches detected")
	}
	// Don't execute the command — it would try real HTTP.
}

func TestDoBatchUpload_MixedUploadedAndUnsent(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "A", Band: "20m", Mode: "SSB", QSODate: "20240501", WavelogID: 1},
		{ID: 2, Call: "B", Band: "20m", Mode: "SSB", QSODate: "20240501"},
	}

	_, next := runUploadPrep(t, le)
	if next == nil {
		t.Error("prep should produce a command for the unsent QSO")
	}
	// Don't execute — would try real HTTP.
}

func TestDoUploadToWavelog_ConfiguredAndValid(t *testing.T) {
	le := newTestEditor("https://log.example.com", "key123", "1", "", "")
	le.editing = &qso.QSO{ID: 5, Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501"}
	le.fillEditForm(le.editing)

	cmd := le.doUploadToWavelog()
	// Should return a non-nil command (the actual HTTP call closure).
	if cmd == nil {
		t.Error("doUploadToWavelog should return a command when config is valid")
	}
	// Don't execute — would try real HTTP.
}

// =============================================================================
// doNormalizeAndUpload tests — temporary SQLite DB
// =============================================================================

// newTestEditorWithDB creates a LogbookEditor backed by a temporary SQLite DB.
func newTestEditorWithDB(t *testing.T, wlURL, wlKey, wlStationID, logOp, logGrid string) *LogbookEditor {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewLogbookEditor(LogbookEditorConfig{DB: db, WLURL: wlURL, WLKey: wlKey, WLStationID: wlStationID, WLLastFetchedID: 0, StationOperator: logOp, StationGrid: logGrid, StationCall: ""})
}

// insertTestQSO inserts a QSO into the store and returns its assigned ID.
func insertTestQSO(t *testing.T, db *sql.DB, q *qso.QSO) int64 {
	t.Helper()
	id, err := store.InsertQSO(db, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	return id
}

func TestDoNormalizeAndUpload_Success(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")

	q1 := &qso.QSO{
		Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB",
		StationCallsign: "OLD_CALL", Operator: "OldOp", MyGridSquare: "OL00ld",
	}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	le.qsos = []qso.QSO{*q1}
	le.mismatchQSOs = []qso.QSO{*q1}
	le.mismatchFields = []string{"operator", "grid"} // callsign NOT flagged

	cmd := le.doNormalizeAndUpload()
	if cmd == nil {
		t.Fatal("doNormalizeAndUpload returned nil command")
	}

	msg := cmd()
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.normalized != 1 {
		t.Errorf("normalized = %d; want 1", em.normalized)
	}

	// The worker must not mutate the in-memory list — it only returns
	// immutable result data for the owner loop to apply.
	if le.qsos[0].Operator != "OldOp" {
		t.Errorf("worker mutated in-memory Operator = %q; want OldOp", le.qsos[0].Operator)
	}
	if em.normOp != "Szymon" || em.normGrid != "KO00ca" {
		t.Errorf("result data op/grid = %q/%q; want Szymon/KO00ca", em.normOp, em.normGrid)
	}

	// The owner loop applies the returned changes (validated generation).
	le.Update(em)
	if le.qsos[0].Operator != "Szymon" {
		t.Errorf("in-memory Operator after Update = %q; want Szymon", le.qsos[0].Operator)
	}
	if le.qsos[0].MyGridSquare != "KO00ca" {
		t.Errorf("in-memory MyGridSquare = %q; want KO00ca", le.qsos[0].MyGridSquare)
	}
	// The original station callsign must be preserved in memory…
	if le.qsos[0].StationCallsign != "OLD_CALL" {
		t.Errorf("in-memory StationCallsign = %q; want OLD_CALL preserved", le.qsos[0].StationCallsign)
	}
	// …and in the database.
	stored, err := store.GetQSOByID(le.db, id1)
	if err != nil || stored == nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.StationCallsign != "OLD_CALL" {
		t.Errorf("stored StationCallsign = %q; want OLD_CALL preserved", stored.StationCallsign)
	}
	if stored.Operator != "Szymon" || stored.MyGridSquare != "KO00ca" {
		t.Errorf("stored op/grid = %q/%q; want Szymon/KO00ca", stored.Operator, stored.MyGridSquare)
	}
}

func TestDoNormalizeAndUpload_MultipleQSOs(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")

	q1 := &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		StationCallsign: "OLD1", Operator: "Old1", MyGridSquare: "AA00aa"}
	q2 := &qso.QSO{Call: "B2B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW",
		StationCallsign: "OLD2", Operator: "Old2", MyGridSquare: "BB00bb"}
	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2

	le.qsos = []qso.QSO{*q1, *q2}
	le.mismatchQSOs = []qso.QSO{*q1, *q2}
	le.mismatchFields = []string{"operator", "grid"}

	cmd := le.doNormalizeAndUpload()
	msg := cmd()
	em := msg.(editorMsg)
	if em.normalized != 2 {
		t.Errorf("normalized = %d; want 2", em.normalized)
	}
	// The worker must not touch the in-memory list — the owner loop applies.
	if le.qsos[0].Operator != "Old1" || le.qsos[1].Operator != "Old2" {
		t.Errorf("worker mutated in-memory operators: %q/%q; want Old1/Old2", le.qsos[0].Operator, le.qsos[1].Operator)
	}
	le.Update(em)
	if le.qsos[0].Operator != "Szymon" || le.qsos[1].Operator != "Szymon" {
		t.Error("both QSOs should have Operator = Szymon after Update")
	}
	if le.qsos[0].StationCallsign != "OLD1" || le.qsos[1].StationCallsign != "OLD2" {
		t.Error("station callsigns should be preserved")
	}
}

func TestDoNormalizeAndUpload_EmptyMismatch(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")
	q1 := &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1
	le.qsos = []qso.QSO{*q1}
	le.mismatchQSOs = nil

	cmd := le.doNormalizeAndUpload()
	msg := cmd()
	em := msg.(editorMsg)
	if em.normalized != 0 {
		t.Errorf("normalized = %d; want 0", em.normalized)
	}
}

func TestDoNormalizeAndUpload_PartialMismatch(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")
	q1 := &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		StationCallsign: "OLD1", Operator: "Old1", MyGridSquare: "AA00aa"}
	q2 := &qso.QSO{Call: "B2B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW",
		StationCallsign: "OLD2", Operator: "Old2", MyGridSquare: "BB00bb"}
	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2
	le.qsos = []qso.QSO{*q1, *q2}
	le.mismatchQSOs = []qso.QSO{*q1} // only q1
	le.mismatchFields = []string{"operator", "grid"}

	cmd := le.doNormalizeAndUpload()
	msg := cmd()
	em := msg.(editorMsg)
	if em.normalized != 1 {
		t.Errorf("normalized = %d; want 1", em.normalized)
	}
	// The worker must not mutate the in-memory list — the owner loop applies.
	if le.qsos[0].Operator != "Old1" {
		t.Errorf("worker mutated q1 Operator = %q; want Old1", le.qsos[0].Operator)
	}
	le.Update(em)
	if le.qsos[0].Operator != "Szymon" {
		t.Errorf("q1 Operator = %q; want Szymon", le.qsos[0].Operator)
	}
	if le.qsos[1].Operator != "Old2" {
		t.Errorf("q2 Operator = %q; want Old2 (unchanged)", le.qsos[1].Operator)
	}
}

// TestNormalizeWorkerDoesNotMutateInMemoryList reproduces the reported race:
// doNormalizeAndUpload used to update le.qsos from its command goroutine while
// the owner loop reads that slice (table rebuilds) — the focused race probe
// reported conflicting accesses to Operator. The worker must be limited to
// database work and immutable result data; the owner loop applies the
// returned changes after validating the generation.
func TestNormalizeWorkerDoesNotMutateInMemoryList(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")

	q1 := &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		StationCallsign: "OLD1", Operator: "OldOp", MyGridSquare: "AA00aa"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1
	le.qsos = []qso.QSO{*q1}
	le.mismatchQSOs = []qso.QSO{*q1}
	le.mismatchFields = []string{"operator"}

	// Run the worker while a concurrent "owner loop" keeps reading the
	// in-memory list — under -race this flags any worker-side mutation.
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = le.qsos[0].Operator
				_ = le.qsos[0].MyGridSquare
			}
		}
	}()

	done := make(chan editorMsg, 1)
	go func() { done <- execCmd(le.doNormalizeAndUpload()).(editorMsg) }()
	em := <-done
	close(stop)
	wg.Wait()

	if em.normalized != 1 {
		t.Fatalf("normalized = %d; want 1", em.normalized)
	}
	if len(em.normIDs) != 1 || em.normIDs[0] != id1 || em.normOp != "Szymon" {
		t.Errorf("result data ids=%v op=%q; want [%d] Szymon", em.normIDs, em.normOp, id1)
	}
	if le.qsos[0].Operator != "OldOp" {
		t.Errorf("worker mutated in-memory Operator = %q; want OldOp", le.qsos[0].Operator)
	}

	// The owner loop applies the returned changes.
	le.Update(em)
	if le.qsos[0].Operator != "Szymon" {
		t.Errorf("in-memory Operator after Update = %q; want Szymon", le.qsos[0].Operator)
	}
}

// TestDoNormalizeAndUpload_CallMismatchUpdatesCallsign verifies that a
// callsign mismatch — and only a callsign mismatch — rewrites
// station_callsign, while operator/grid stay untouched.
func TestDoNormalizeAndUpload_CallMismatchUpdatesCallsign(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")
	le.logStationCall = "SP9MOA"

	q1 := &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		StationCallsign: "OLD_CALL", Operator: "KeepOp", MyGridSquare: "AA00aa"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1
	le.qsos = []qso.QSO{*q1}
	le.mismatchQSOs = []qso.QSO{*q1}
	le.mismatchFields = []string{"callsign"}

	cmd := le.doNormalizeAndUpload()
	msg := cmd()
	em := msg.(editorMsg)
	if em.normalized != 1 {
		t.Errorf("normalized = %d; want 1", em.normalized)
	}

	stored, err := store.GetQSOByID(le.db, id1)
	if err != nil || stored == nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.StationCallsign != "SP9MOA" {
		t.Errorf("stored StationCallsign = %q; want SP9MOA", stored.StationCallsign)
	}
	if stored.Operator != "KeepOp" {
		t.Errorf("stored Operator = %q; want KeepOp (unchanged)", stored.Operator)
	}
	if stored.MyGridSquare != "AA00aa" {
		t.Errorf("stored MyGridSquare = %q; want AA00aa (unchanged)", stored.MyGridSquare)
	}
}

// TestDoBatchUpload_DetectsCallMismatch verifies a station-callsign mismatch
// is flagged for confirmation when the logbook station callsign is known.
func TestDoBatchUpload_DetectsCallMismatch(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.logStationCall = "SP9MOA"
	le.qsos = []qso.QSO{
		{
			ID: 10, Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
			StationCallsign: "DIFFERENT",
		},
	}

	le2, next := runUploadPrep(t, le)
	if next != nil {
		t.Errorf("expected nil command when a callsign mismatch is flagged, got %T", next)
	}
	hasCall := false
	for _, f := range le2.mismatchFields {
		if f == "callsign" {
			hasCall = true
		}
	}
	if !hasCall {
		t.Errorf("mismatchFields should contain 'callsign', got %v", le2.mismatchFields)
	}
}

// =============================================================================
// Download-then-upload safety
// =============================================================================

func TestUploadSkipsDownloadedQSOs(t *testing.T) {
	// QSOs downloaded from Wavelog are marked "yes".  The batch upload
	// must skip them — only locally-created QSOs (marked "" or "no") are sent.
	// Use empty URL so the test never touches the network.
	le := newTestEditor("", "", "", "Op", "KO00")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", WavelogID: 1},
		{ID: 2, Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502"},
		{ID: 3, Call: "SP9CCC", Band: "15m", Mode: "FT8", QSODate: "20240503"},
		{ID: 4, Call: "SP9DDD", Band: "10m", Mode: "SSB", QSODate: "20240504", WavelogID: 1},
	}

	em := prepAndRunUpload(t, le)
	// Empty URL → upload should fail, but NOT with "all sent".
	// The filtering found unsent QSOs (2 and 3) and tried to send them.
	if em.wlCall == "all sent" {
		t.Error("should not report 'all sent' when unsent QSOs exist")
	}
	// Verify it's an upload error (empty URL), not a filtering error.
	if em.wlOK {
		t.Error("wlOK should be false — upload to empty URL must fail")
	}
}

func TestDownloadMarksAllQSOsAsUploaded(t *testing.T) {
	// After a Wavelog download, every inserted QSO must have
	// WavelogID > 0 so a subsequent upload won't re-send them.
	le := newTestEditorWithDB(t, "", "", "", "", "")

	// Simulate what the download loop does: insert QSOs with "yes".
	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		WavelogID: 1, Source: "wavelog"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000",
		WavelogID: 1, Source: "wavelog"}

	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2

	// Load them as if just downloaded.
	le.qsos = []qso.QSO{*q1, *q2}

	// Batch upload should see both as already sent.
	le.mode = edModeList
	em := prepAndRunUpload(t, le)
	if em.wlCall != "all sent" {
		t.Errorf("wlCall = %q; want 'all sent' (both QSOs already marked yes)", em.wlCall)
	}
}

// =============================================================================
// Purge tests
// =============================================================================

func TestPurge_ClearsQSOs(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "", "")

	// Insert some QSOs.
	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000"}
	insertTestQSO(t, le.db, q1)
	insertTestQSO(t, le.db, q2)

	// Verify QSOs exist.
	qsos, err := store.ListQSOs(le.db, 10, "")
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) != 2 {
		t.Fatalf("expected 2 QSOs before purge, got %d", len(qsos))
	}

	// Purge — must set dialog so doConfirm() proceeds.
	le.mode = edModeConfirmPurge
	d := NewDialog("Purge", "test")
	le.dialog = &d
	cmd := le.doConfirm()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if !em.purged {
		t.Error("purged should be true")
	}
	if em.err != nil {
		t.Errorf("unexpected error: %v", em.err)
	}

	// Verify QSOs are gone.
	qsos, err = store.ListQSOs(le.db, 10, "")
	if err != nil {
		t.Fatalf("ListQSOs after purge: %v", err)
	}
	if len(qsos) != 0 {
		t.Errorf("expected 0 QSOs after purge, got %d", len(qsos))
	}
}

func TestPurge_EmptyLogbookIsSafe(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "", "")

	// Purge an empty logbook — should succeed without error.
	le.mode = edModeConfirmPurge
	d := NewDialog("Purge", "test")
	le.dialog = &d
	cmd := le.doConfirm()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if !em.purged {
		t.Error("purged should be true even for empty logbook")
	}
	if em.err != nil {
		t.Errorf("unexpected error on empty purge: %v", em.err)
	}
}

func TestUploadBatch_FiltersUnsentQSOs(t *testing.T) {
	// Verify that doBatchUpload only selects QSOs without a remote id.
	// Use empty URL to avoid real HTTP calls.
	le := newTestEditor("", "", "", "Op", "KO00")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "A", Band: "20m", Mode: "SSB", QSODate: "20240501", WavelogID: 1},
		{ID: 2, Call: "B", Band: "20m", Mode: "SSB", QSODate: "20240502"},
		{ID: 3, Call: "C", Band: "20m", Mode: "SSB", QSODate: "20240503"},
		{ID: 4, Call: "D", Band: "20m", Mode: "SSB", QSODate: "20240504", WavelogID: 1},
	}

	cmd := le.doBatchUpload()
	msg := execCmd(cmd)
	if _, ok := msg.(uploadPrepMsg); !ok {
		t.Fatalf("expected uploadPrepMsg, got %T", msg)
	}
	_, next := le.Update(msg)
	if next == nil {
		t.Fatal("prep produced no upload command for unsent QSOs")
	}
	em := execCmd(next).(editorMsg)
	if em.wlCall == "all sent" {
		t.Error("should not report 'all sent' — QSOs 2 and 3 are unsent")
	}
}

func TestPurgeResetsWavelogLastID(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "", "")
	le.wlLastFetchedID = 12345
	le.mode = edModeConfirmPurge
	d := NewDialog("Purge", "test")
	le.dialog = &d

	cmd := le.doConfirm()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}

	if !em.purged {
		t.Error("purged should be true")
	}
	if le.wlLastFetchedID != 0 {
		t.Errorf("wlLastFetchedID = %d; want 0 after purge", le.wlLastFetchedID)
	}
}

// =============================================================================
// Pass 10 — Batch upload with httptest.Server (full HTTP integration)
// =============================================================================

func TestUploadBatch_MockServerSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			// Remote-id backfill after the batch upload.
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 42, "call": "SP9AAA", "band": "20m", "mode": "SSB", "qso_date": "2024-05-01 12:00:00"},
					{"id": 43, "call": "SP9BBB", "band": "40m", "mode": "CW", "qso_date": "2024-05-02 13:00:00"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"parsed": 2, "imported": 2, "skipped": 0, "messages": []string{}},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "1", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000",
		RSTSent: "599", RSTRcvd: "579"}
	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2

	unsent := []qso.QSO{*q1, *q2}
	cmd := le.uploadBatch(unsent)
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if !em.wlOK {
		t.Errorf("batch upload should succeed, got err=%v", em.err)
	}

	// Verify both QSOs are marked as uploaded in DB.
	for _, id := range []int64{id1, id2} {
		var remoteID int64
		if err := le.db.QueryRow("SELECT wavelog_id FROM qsos WHERE id=?", id).Scan(&remoteID); err != nil {
			t.Fatalf("query qso %d: %v", id, err)
		}
		if remoteID == 0 {
			t.Errorf("QSO %d wavelog_id = %d, want >0", id, remoteID)
		}
	}
}

func TestUploadBatch_MockServerDuplicate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			// Remote-id backfill after the duplicate batch.
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"id": 42, "call": "SP9AAA", "band": "20m", "mode": "SSB",
					"qso_date": "2024-05-01 12:00:00",
				}},
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"parsed": 1, "imported": 0, "skipped": 1, "messages": []string{"Duplicate"}},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "1", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	unsent := []qso.QSO{*q1}
	cmd := le.uploadBatch(unsent)
	msg := execCmd(cmd)
	em := msg.(editorMsg)
	if !em.wlOK {
		t.Errorf("duplicate should be treated as OK, got err=%v", em.err)
	}

	// After duplicate batch the remote id is backfilled from the mock list.
	// because AllDuplicates checks on the result object, not on error string.
	// The error path for "duplicate" falls through to uploadIndividual.
	// Let's verify the final state.
	var remoteID int64
	if err := le.db.QueryRow("SELECT wavelog_id FROM qsos WHERE id=?", id1).Scan(&remoteID); err != nil {
		t.Fatalf("query qso %d: %v", id1, err)
	}
	// The result is "abort" with AllDuplicates=true after JSON parse.
	// Verify status was updated.
	if remoteID == 0 {
		t.Errorf("QSO wavelog_id = %d, want >0 (duplicate = present on Wavelog)", remoteID)
	}
}

// TestUploadBatch_PartialChunkFailure verifies a partial batch result is
// reported honestly: the successful chunk counts as sent, the failed chunk
// as failed, and the summary never claims the whole input succeeded.
func TestUploadBatch_PartialChunkFailure(t *testing.T) {
	postCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Has("page"):
			// Reconciliation scan: nothing is already on Wavelog.
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{},
				"meta": map[string]any{"page": 1, "has_more": false},
			})
		case r.Method == http.MethodGet:
			// Id backfill for the successful first chunk.
			rows := make([]map[string]any, 0, 50)
			for i := 0; i < 50; i++ {
				rows = append(rows, map[string]any{
					"id":       1000 + i,
					"call":     fmt.Sprintf("SP9B%02d", i),
					"band":     "20m",
					"mode":     "SSB",
					"qso_date": fmt.Sprintf("2024-05-10 %02d:00:00", i),
				})
			}
			json.NewEncoder(w).Encode(map[string]any{"data": rows})
		case r.Method == http.MethodPost:
			postCount++
			if postCount == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{"parsed": 50, "imported": 50, "skipped": 0, "messages": []string{}},
					"meta": map[string]string{"resource": "qso", "method": "POST"},
				})
			} else {
				http.Error(w, "server error", http.StatusInternalServerError)
			}
		}
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	var unsent []qso.QSO
	for i := 0; i < 100; i++ {
		q := &qso.QSO{Call: fmt.Sprintf("SP9B%02d", i), Band: "20m", Mode: "SSB",
			QSODate: "20240510", TimeOn: fmt.Sprintf("%02d0000", i), RSTSent: "59", RSTRcvd: "59"}
		id := insertTestQSO(t, le.db, q)
		q.ID = id
		unsent = append(unsent, *q)
	}

	msg := execCmd(le.uploadBatch(unsent))
	em := msg.(editorMsg)
	if !em.wlOK {
		t.Fatalf("partial upload must not report total failure (err=%v)", em.err)
	}
	if em.wlSentCount != 50 || em.wlFailCount != 50 {
		t.Errorf("counts = sent:%d fail:%d, want 50/50", em.wlSentCount, em.wlFailCount)
	}
	if em.wlDupCount != 0 || em.wlUnresolvedCount != 0 {
		t.Errorf("dup/unresolved = %d/%d, want 0/0", em.wlDupCount, em.wlUnresolvedCount)
	}
	if !strings.Contains(em.wlCall, "50 sent") || !strings.Contains(em.wlCall, "50 failed") {
		t.Errorf("wlCall = %q, want explicit sent/failed tallies", em.wlCall)
	}

	// First chunk has persisted remote ids; the failed chunk stays unsent.
	for i, q := range unsent {
		stored, err := store.GetQSOByID(le.db, q.ID)
		if err != nil {
			t.Fatalf("GetQSOByID(%d): %v", q.ID, err)
		}
		if i < 50 && stored.WavelogID == 0 {
			t.Errorf("QSO %d (chunk 1) wavelog_id = 0, want >0", q.ID)
		}
		if i >= 50 && stored.WavelogID != 0 {
			t.Errorf("QSO %d (chunk 2) wavelog_id = %d, want 0 (failed)", q.ID, stored.WavelogID)
		}
	}
}

func TestUploadBatch_MockServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "1", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	unsent := []qso.QSO{*q1}
	cmd := le.uploadBatch(unsent)
	msg := execCmd(cmd)
	em := msg.(editorMsg)
	if em.wlOK {
		t.Error("batch upload should fail on server 500")
	}

	// Verify QSO is NOT marked as uploaded after failure.
	var remoteID int64
	if err := le.db.QueryRow("SELECT wavelog_id FROM qsos WHERE id=?", id1).Scan(&remoteID); err != nil {
		t.Fatalf("query qso %d: %v", id1, err)
	}
	if remoteID != 0 {
		t.Errorf("QSO wavelog_id = %d, want 0 (upload failed)", remoteID)
	}
}

func TestUploadIndividual_MixedResults(t *testing.T) {
	// Mock server: first QSO succeeds, second fails with 500.
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			// Remote-id backfill for the successful QSO.
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 42, "call": "SP9AAA", "band": "20m", "mode": "SSB", "qso_date": "2024-05-01 12:00:00"},
				},
			})
			return
		}
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"parsed": 1, "imported": 1, "skipped": 0, "messages": []string{}},
				"meta": map[string]string{"resource": "qso", "method": "POST"},
			})
		} else {
			http.Error(w, "server error", 500)
		}
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "1", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000",
		RSTSent: "599", RSTRcvd: "579"}
	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2

	// Use uploadIndividual directly (bypasses batch→individual fallback).
	unsent := []qso.QSO{*q1, *q2}
	cmd := le.uploadIndividual(unsent)
	msg := execCmd(cmd)
	em := msg.(editorMsg)
	if !em.wlOK {
		t.Errorf("individual upload should report OK (partial success), got err=%v", em.err)
	}
	if em.wlCall == "" {
		t.Error("wlCall should contain summary")
	}
	if em.wlSentCount != 1 || em.wlFailCount != 1 {
		t.Errorf("counts = sent:%d fail:%d, want 1/1", em.wlSentCount, em.wlFailCount)
	}
	if em.wlDupCount != 0 || em.wlUnresolvedCount != 0 {
		t.Errorf("dup/unresolved = %d/%d, want 0/0", em.wlDupCount, em.wlUnresolvedCount)
	}
	if !strings.Contains(em.wlCall, "1 sent") || !strings.Contains(em.wlCall, "1 failed") {
		t.Errorf("wlCall = %q, want explicit sent/failed tallies", em.wlCall)
	}

	// QSO 1 should be marked uploaded, QSO 2 should NOT.
	var s1, s2 int64
	le.db.QueryRow("SELECT wavelog_id FROM qsos WHERE id=?", id1).Scan(&s1)
	le.db.QueryRow("SELECT wavelog_id FROM qsos WHERE id=?", id2).Scan(&s2)
	if s1 == 0 {
		t.Errorf("QSO 1 wavelog_id = %d, want >0", s1)
	}
	// QSO 2: postQSO with 500 → error → no remote id is stored.
	if s2 != 0 {
		t.Errorf("QSO 2 wavelog_id = %d, want 0 (upload failed)", s2)
	}
}

func TestUploadBatch_RequestPayloadVerification(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"parsed": 1, "imported": 1, "skipped": 0, "messages": []string{}},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-api-key", "42", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	unsent := []qso.QSO{*q1}
	cmd := le.uploadBatch(unsent)
	execCmd(cmd)

	if capturedBody["import_type"] != "adif" {
		t.Errorf("import_type = %v, want adif", capturedBody["import_type"])
	}
	if capturedBody["station_profile_id"] != float64(42) {
		t.Errorf("station_profile_id = %v, want 42", capturedBody["station_profile_id"])
	}
	// Verify ADIF string contains the QSO data.
	adifStr, _ := capturedBody["adif"].(string)
	if adifStr == "" {
		t.Error("ADIF string should not be empty")
	}
	if !strings.Contains(adifStr, "SP9AAA") {
		t.Error("ADIF should contain callsign SP9AAA")
	}
}

func TestUploadBatch_EmptyUnsentList(t *testing.T) {
	le := newTestEditorWithDB(t, "https://example.com", "key", "1", "Op", "JO90")

	// uploadBatch with empty list should still work (returns all-sent message).
	cmd := le.uploadBatch(nil)
	msg := execCmd(cmd)
	em := msg.(editorMsg)
	// It will try to POST empty ADIF — PostQSOWithResult rejects empty adifStr.
	if em.wlOK {
		t.Error("uploadBatch with nil unsent should fail (empty ADIF rejected)")
	}
}

func TestUploadBatch_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wrong-key", "1", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	unsent := []qso.QSO{*q1}
	cmd := le.uploadBatch(unsent)
	msg := execCmd(cmd)
	em := msg.(editorMsg)
	if em.wlOK {
		t.Error("batch upload should fail on 401")
	}

	var remoteID int64
	le.db.QueryRow("SELECT wavelog_id FROM qsos WHERE id=?", id1).Scan(&remoteID)
	if remoteID != 0 {
		t.Errorf("QSO wavelog_id = %d, want 0 (upload failed)", remoteID)
	}
}

// TestEditSavePreservesWavelogID verifies that saving an edited QSO keeps its
// remote id — losing it would make the QSO look unsent and invite duplicate
// uploads.
func TestEditSavePreservesWavelogID(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("edited")

	cmd := le.doSave()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.err != nil {
		t.Fatalf("save failed: %v", em.err)
	}

	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogID != 42 {
		t.Errorf("WavelogID after edit-save = %d, want 42", stored.WavelogID)
	}
}

// TestUploadBatch_ReconcilesBeforeUpload verifies the migration scenario: a
// large unsent list is reconciled against the remote list first, so QSOs that
// already exist on Wavelog learn their id locally and are never re-uploaded.
func TestUploadBatch_ReconcilesBeforeUpload(t *testing.T) {
	var postedBodies []string
	var listFetched bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Has("page"):
			listFetched = true
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 77, "call": "SP9REC", "band": "20m", "mode": "SSB", "qso_date": "2024-05-10 10:00:00"},
				},
				"meta": map[string]any{"page": 1, "has_more": false},
			})
		case r.Method == http.MethodGet:
			// FindQSOIDs backfill after the upload — return nothing.
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		case r.Method == http.MethodPost:
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if adif, ok := body["adif"].(string); ok {
				postedBodies = append(postedBodies, adif)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"parsed": 25, "imported": 25, "skipped": 0, "messages": []string{}},
				"meta": map[string]string{"resource": "qso", "method": "POST"},
			})
		}
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	rec := &qso.QSO{Call: "SP9REC", Band: "20m", Mode: "SSB", QSODate: "20240510",
		TimeOn: "100000", RSTSent: "59", RSTRcvd: "59"}
	recID := insertTestQSO(t, le.db, rec)
	rec.ID = recID

	unsent := []qso.QSO{*rec}
	for i := 0; i < 25; i++ {
		q := &qso.QSO{Call: fmt.Sprintf("SP9N%02d", i), Band: "20m", Mode: "SSB",
			QSODate: "20240511", TimeOn: fmt.Sprintf("%02d0000", i), RSTSent: "59", RSTRcvd: "59"}
		id := insertTestQSO(t, le.db, q)
		q.ID = id
		unsent = append(unsent, *q)
	}

	msg := execCmd(le.uploadBatch(unsent))
	em := msg.(editorMsg)
	if !em.wlOK {
		t.Fatalf("wlOK = false, err=%v", em.err)
	}
	if !listFetched {
		t.Error("reconciliation list was never fetched")
	}
	if !strings.Contains(em.wlCall, "already on Wavelog") {
		t.Errorf("wlCall = %q, want reconciled count", em.wlCall)
	}
	// The reconciled QSO must have learned its remote id.
	stored, err := store.GetQSOByID(le.db, recID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogID != 77 {
		t.Errorf("reconciled WavelogID = %d, want 77", stored.WavelogID)
	}
	// The reconciled QSO must NOT have been uploaded.
	for _, body := range postedBodies {
		if strings.Contains(body, "SP9REC") {
			t.Errorf("SP9REC was uploaded despite reconciliation: %q", body)
		}
	}
	if len(postedBodies) == 0 {
		t.Error("the 25 new QSOs were never uploaded")
	}
}

// TestUploadBatch_SmallBatchSkipsReconcile verifies small batches upload
// directly without the reconciliation list fetch.
func TestUploadBatch_SmallBatchSkipsReconcile(t *testing.T) {
	listFetched := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Has("page"):
			listFetched = true
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}, "meta": map[string]any{"has_more": false}})
		case r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		case r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"parsed": 3, "imported": 3, "skipped": 0, "messages": []string{}},
				"meta": map[string]string{"resource": "qso", "method": "POST"},
			})
		}
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	var unsent []qso.QSO
	for i := 0; i < 3; i++ {
		q := &qso.QSO{Call: fmt.Sprintf("SP9M%02d", i), Band: "20m", Mode: "SSB",
			QSODate: "20240511", TimeOn: fmt.Sprintf("%02d0000", i), RSTSent: "59", RSTRcvd: "59"}
		id := insertTestQSO(t, le.db, q)
		q.ID = id
		unsent = append(unsent, *q)
	}

	msg := execCmd(le.uploadBatch(unsent))
	em := msg.(editorMsg)
	if !em.wlOK {
		t.Fatalf("wlOK = false, err=%v", em.err)
	}
	if listFetched {
		t.Error("small batch should skip the reconciliation list fetch")
	}
}
