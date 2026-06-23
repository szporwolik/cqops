package tui

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
	return NewLogbookEditor(nil, wlURL, wlKey, wlStationID, 0, logOp, logGrid, "")
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
	return NewLogbookEditor(db, wlURL, wlKey, wlStationID, 0, logOp, logGrid, "")
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

// =============================================================================
// Download-then-upload safety
// =============================================================================

func TestUploadSkipsDownloadedQSOs(t *testing.T) {
	// QSOs downloaded from Wavelog are marked "yes".  The batch upload
	// must skip them — only locally-created QSOs (marked "" or "no") are sent.
	// Use empty URL so the test never touches the network.
	le := newTestEditor("", "", "", "Op", "KO00")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", WavelogUploaded: "yes"},
		{ID: 2, Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", WavelogUploaded: ""},
		{ID: 3, Call: "SP9CCC", Band: "15m", Mode: "FT8", QSODate: "20240503", WavelogUploaded: "no"},
		{ID: 4, Call: "SP9DDD", Band: "10m", Mode: "SSB", QSODate: "20240504", WavelogUploaded: "yes"},
	}

	cmd := le.doBatchUpload()
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
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
	// WavelogUploaded = "yes" so a subsequent upload won't re-send them.
	le := newTestEditorWithDB(t, "", "", "", "", "")

	// Simulate what the download loop does: insert QSOs with "yes".
	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		WavelogUploaded: "yes", Source: "wavelog"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000",
		WavelogUploaded: "yes", Source: "wavelog"}

	id1 := insertTestQSO(t, le.db, q1)
	id2 := insertTestQSO(t, le.db, q2)
	q1.ID = id1
	q2.ID = id2

	// Load them as if just downloaded.
	le.qsos = []qso.QSO{*q1, *q2}

	// Batch upload should see both as already sent.
	le.mode = edModeList
	cmd := le.doBatchUpload()
	msg := execCmd(cmd)
	em := msg.(editorMsg)
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
	// Verify that doBatchUpload only selects QSOs with WavelogUploaded != "yes".
	// Use empty URL to avoid real HTTP calls.
	le := newTestEditor("", "", "", "Op", "KO00")
	le.qsos = []qso.QSO{
		{ID: 1, Call: "A", Band: "20m", Mode: "SSB", QSODate: "20240501", WavelogUploaded: "yes"},
		{ID: 2, Call: "B", Band: "20m", Mode: "SSB", QSODate: "20240502", WavelogUploaded: "no"},
		{ID: 3, Call: "C", Band: "20m", Mode: "SSB", QSODate: "20240503", WavelogUploaded: ""},
		{ID: 4, Call: "D", Band: "20m", Mode: "SSB", QSODate: "20240504", WavelogUploaded: "yes"},
	}

	cmd := le.doBatchUpload()
	msg := execCmd(cmd)
	em := msg.(editorMsg)
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok", "adif_count": 2, "adif_errors": 0, "messages": []string{""},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "SP-0001", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59", WavelogUploaded: "no"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000",
		RSTSent: "599", RSTRcvd: "579", WavelogUploaded: "no"}
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
		var status string
		if err := le.db.QueryRow("SELECT wavelog_uploaded FROM qsos WHERE id=?", id).Scan(&status); err != nil {
			t.Fatalf("query qso %d: %v", id, err)
		}
		if status != "yes" {
			t.Errorf("QSO %d wavelog_uploaded = %q, want yes", id, status)
		}
	}
}

func TestUploadBatch_MockServerDuplicate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "abort", "adif_count": 1, "adif_errors": 1,
			"messages": []string{"", "Duplicate for SP9AAA"},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "SP-0001", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59", WavelogUploaded: "no"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	unsent := []qso.QSO{*q1}
	cmd := le.uploadBatch(unsent)
	msg := execCmd(cmd)
	em := msg.(editorMsg)
	if !em.wlOK {
		t.Errorf("duplicate should be treated as OK, got err=%v", em.err)
	}

	// After duplicate batch, wavelog_uploaded should NOT yet be "yes"
	// because AllDuplicates checks on the result object, not on error string.
	// The error path for "duplicate" falls through to uploadIndividual.
	// Let's verify the final state.
	var status string
	if err := le.db.QueryRow("SELECT wavelog_uploaded FROM qsos WHERE id=?", id1).Scan(&status); err != nil {
		t.Fatalf("query qso %d: %v", id1, err)
	}
	// The result is "abort" with AllDuplicates=true after JSON parse.
	// Verify status was updated.
	if status != "yes" {
		t.Errorf("QSO wavelog_uploaded = %q, want yes (duplicate = present on Wavelog)", status)
	}
}

func TestUploadBatch_MockServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "SP-0001", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59", WavelogUploaded: "no"}
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
	var status string
	if err := le.db.QueryRow("SELECT wavelog_uploaded FROM qsos WHERE id=?", id1).Scan(&status); err != nil {
		t.Fatalf("query qso %d: %v", id1, err)
	}
	if status != "no" {
		t.Errorf("QSO wavelog_uploaded = %q, want no (upload failed)", status)
	}
}

func TestUploadIndividual_MixedResults(t *testing.T) {
	// Mock server: first QSO succeeds, second fails with 500.
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok", "adif_count": 1, "adif_errors": 0, "messages": []string{""},
			})
		} else {
			http.Error(w, "server error", 500)
		}
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-key", "SP-0001", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59", WavelogUploaded: "no"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000",
		RSTSent: "599", RSTRcvd: "579", WavelogUploaded: "no"}
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

	// QSO 1 should be marked uploaded, QSO 2 should NOT.
	var s1, s2 string
	le.db.QueryRow("SELECT wavelog_uploaded FROM qsos WHERE id=?", id1).Scan(&s1)
	le.db.QueryRow("SELECT wavelog_uploaded FROM qsos WHERE id=?", id2).Scan(&s2)
	if s1 != "yes" {
		t.Errorf("QSO 1 status = %q, want yes", s1)
	}
	// QSO 2: postQSO with 500 → error → UpdateWavelogStatus(db, id, "no").
	// Actually, postQSO on 500 returns ok=false, so UpdateWavelogStatus("no") is called.
	if s2 != "no" {
		t.Errorf("QSO 2 status = %q, want no (upload failed)", s2)
	}
}

func TestUploadBatch_RequestPayloadVerification(t *testing.T) {
	var capturedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok", "adif_count": 1, "adif_errors": 0, "messages": []string{""},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "test-api-key", "42", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59", WavelogUploaded: "no"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	unsent := []qso.QSO{*q1}
	cmd := le.uploadBatch(unsent)
	execCmd(cmd)

	if capturedBody["key"] != "test-api-key" {
		t.Errorf("key = %q, want test-api-key", capturedBody["key"])
	}
	if capturedBody["station_profile_id"] != "42" {
		t.Errorf("station_profile_id = %q, want 42", capturedBody["station_profile_id"])
	}
	if capturedBody["type"] != "adif" {
		t.Errorf("type = %q, want adif", capturedBody["type"])
	}
	// Verify ADIF string contains the QSO data.
	if capturedBody["string"] == "" {
		t.Error("ADIF string should not be empty")
	}
	if !strings.Contains(capturedBody["string"], "SP9AAA") {
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

	le := newTestEditorWithDB(t, srv.URL, "wrong-key", "SP-0001", "Op", "JO90")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59", WavelogUploaded: "no"}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	unsent := []qso.QSO{*q1}
	cmd := le.uploadBatch(unsent)
	msg := execCmd(cmd)
	em := msg.(editorMsg)
	if em.wlOK {
		t.Error("batch upload should fail on 401")
	}

	var status string
	le.db.QueryRow("SELECT wavelog_uploaded FROM qsos WHERE id=?", id1).Scan(&status)
	if status != "no" {
		t.Errorf("QSO status = %q, want no (upload failed)", status)
	}
}
