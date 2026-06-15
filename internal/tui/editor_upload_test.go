package tui

import (
	"database/sql"
	"path/filepath"
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
	return NewLogbookEditor(nil, wlURL, wlKey, wlStationID, 0, logOp, logGrid)
}

// execCmd executes a tea.Cmd and returns the message. Returns nil if cmd is nil.
func execCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
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
	le := newTestEditor("https://log.example.com", "key123", "SP-0001", "", "")
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
		{ID: 1, Call: "A", WavelogUploaded: "yes"},
		{ID: 2, Call: "B", WavelogUploaded: "yes"},
	}

	cmd := le.doBatchUpload()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
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

	cmd := le.doBatchUpload()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if !em.wlOK {
		t.Error("wlOK should be true for empty QSO list")
	}
	if em.wlCall != "all sent" {
		t.Errorf("wlCall = %q; want 'all sent'", em.wlCall)
	}
}

func TestDoBatchUpload_SkipsMissingFields(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "A", Band: "", Mode: "SSB", QSODate: "20240501", WavelogUploaded: "no"},
		{ID: 2, Call: "B", Band: "20m", Mode: "", QSODate: "20240501", WavelogUploaded: "no"},
		{ID: 3, Call: "C", Band: "20m", Mode: "SSB", QSODate: "", WavelogUploaded: "no"},
	}

	cmd := le.doBatchUpload()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}

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
		{ID: 1, Call: "SP9MOA", Band: "", Mode: "SSB", QSODate: "20240501", WavelogUploaded: "no"},
	}

	cmd := le.doBatchUpload()
	execCmd(cmd)

	if le.wlSkipped != 1 {
		t.Errorf("wlSkipped = %d; want 1", le.wlSkipped)
	}
	// Single skip should mention callsign and date.
	if le.wlSkipDetail == "" {
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
			WavelogUploaded: "no",
		},
	}

	cmd := le.doBatchUpload()
	// Should return nil (mismatch → set mode to confirm normalize, no command).
	if cmd != nil {
		t.Errorf("doBatchUpload should return nil when mismatches detected, got %T", cmd)
	}

	if le.mode != edModeConfirmNormalize {
		t.Errorf("mode = %v; want edModeConfirmNormalize", le.mode)
	}
	if len(le.mismatchQSOs) != 1 {
		t.Errorf("mismatchQSOs length = %d; want 1", len(le.mismatchQSOs))
	}
	if len(le.mismatchFields) < 2 {
		t.Errorf("mismatchFields should contain operator and grid, got %v", le.mismatchFields)
	}

	// Verify fields list contains expected mismatches.
	hasOp := false
	hasGrid := false
	for _, f := range le.mismatchFields {
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
			WavelogUploaded: "no",
		},
	}

	cmd := le.doBatchUpload()
	// Should return an uploadBatch command (not nil), since no mismatches and
	// no station defaults to compare against.
	if cmd == nil {
		t.Error("doBatchUpload should return a command when no mismatches detected")
	}
	// Don't execute the command — it would try real HTTP.
}

func TestDoBatchUpload_MixedUploadedAndUnsent(t *testing.T) {
	le := newTestEditor("", "", "", "", "")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "A", Band: "20m", Mode: "SSB", QSODate: "20240501", WavelogUploaded: "yes"},
		{ID: 2, Call: "B", Band: "20m", Mode: "SSB", QSODate: "20240501", WavelogUploaded: "no"},
	}

	cmd := le.doBatchUpload()
	// Should return a command for the one unsent QSO.
	if cmd == nil {
		t.Error("doBatchUpload should return a command for the unsent QSO")
	}
	// Don't execute — would try real HTTP.
}

func TestDoUploadToWavelog_ConfiguredAndValid(t *testing.T) {
	le := newTestEditor("https://log.example.com", "key123", "SP-0001", "", "")
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
	return NewLogbookEditor(db, wlURL, wlKey, wlStationID, 0, logOp, logGrid)
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
		WavelogUploaded: "no",
	}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	le.qsos = []qso.QSO{*q1}
	le.mismatchQSOs = []qso.QSO{*q1}

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

	if le.qsos[0].Operator != "Szymon" {
		t.Errorf("in-memory Operator = %q; want Szymon", le.qsos[0].Operator)
	}
	if le.qsos[0].MyGridSquare != "KO00ca" {
		t.Errorf("in-memory MyGridSquare = %q; want KO00ca", le.qsos[0].MyGridSquare)
	}
}

func TestDoNormalizeAndUpload_MultipleQSOs(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")

	q1 := &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		StationCallsign: "OLD1", Operator: "Old1", MyGridSquare: "AA00aa", WavelogUploaded: "no"}
	q2 := &qso.QSO{Call: "B2B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW",
		StationCallsign: "OLD2", Operator: "Old2", MyGridSquare: "BB00bb", WavelogUploaded: "no"}
	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2

	le.qsos = []qso.QSO{*q1, *q2}
	le.mismatchQSOs = []qso.QSO{*q1, *q2}

	cmd := le.doNormalizeAndUpload()
	msg := cmd()
	em := msg.(editorMsg)
	if em.normalized != 2 {
		t.Errorf("normalized = %d; want 2", em.normalized)
	}
	if le.qsos[0].Operator != "Szymon" || le.qsos[1].Operator != "Szymon" {
		t.Error("both QSOs should have Operator = Szymon")
	}
}

func TestDoNormalizeAndUpload_EmptyMismatch(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "Szymon", "KO00ca")
	q1 := &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		WavelogUploaded: "no"}
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
		StationCallsign: "OLD1", Operator: "Old1", MyGridSquare: "AA00aa", WavelogUploaded: "no"}
	q2 := &qso.QSO{Call: "B2B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW",
		StationCallsign: "OLD2", Operator: "Old2", MyGridSquare: "BB00bb", WavelogUploaded: "no"}
	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2
	le.qsos = []qso.QSO{*q1, *q2}
	le.mismatchQSOs = []qso.QSO{*q1} // only q1

	cmd := le.doNormalizeAndUpload()
	msg := cmd()
	em := msg.(editorMsg)
	if em.normalized != 1 {
		t.Errorf("normalized = %d; want 1", em.normalized)
	}
	if le.qsos[0].Operator != "Szymon" {
		t.Errorf("q1 Operator = %q; want Szymon", le.qsos[0].Operator)
	}
	if le.qsos[1].Operator != "Old2" {
		t.Errorf("q2 Operator = %q; want Old2 (unchanged)", le.qsos[1].Operator)
	}
}
