package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Test helpers
// =============================================================================

func newTempDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := InitDB(path)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func mustInsertQSO(t *testing.T, db *sql.DB, q *qso.QSO) int64 {
	t.Helper()
	id, err := InsertQSO(db, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	q.ID = id
	return id
}

// validQSO returns a minimal valid QSO for insert tests.
func validQSO() *qso.QSO {
	return &qso.QSO{
		Call:    "SP9MOA",
		QSODate: "20240501",
		TimeOn:  "120000",
		Band:    "20m",
		Mode:    "SSB",
		Source:  "manual",
	}
}

// =============================================================================
// InitDB / Migrate tests
// =============================================================================

func TestInitDB_CreatesSchema(t *testing.T) {
	db := newTempDB(t)

	// Verify the qsos table exists by querying it.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM qsos").Scan(&count); err != nil {
		t.Fatalf("qsos table query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("qsos table should be empty after InitDB, got %d rows", count)
	}
}

func TestInitDB_Idempotent(t *testing.T) {
	// Use a single explicit path so we can re-open it.
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("first InitDB: %v", err)
	}
	id := mustInsertQSO(t, db, validQSO())
	db.Close()

	// Re-open the same file — migration should be idempotent.
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("second InitDB: %v", err)
	}
	defer db2.Close()

	// Re-running migration should not destroy existing data.
	q, err := GetQSOByID(db2, id)
	if err != nil {
		t.Fatalf("GetQSOByID after re-init: %v", err)
	}
	if q.Call != "SP9MOA" {
		t.Errorf("call = %q; want SP9MOA", q.Call)
	}
}

// =============================================================================
// InsertQSO / GetQSOByID / ListQSOs tests
// =============================================================================

func TestInsertAndGetQSO(t *testing.T) {
	db := newTempDB(t)

	q := validQSO()
	q.Freq = 14.200
	q.RSTSent = "59"
	q.RSTRcvd = "59"
	q.Name = "Szymon"
	q.Operator = "OP"
	q.StationCallsign = "SP9MOA"
	q.MyGridSquare = "KO00ca"

	id := mustInsertQSO(t, db, q)
	if id <= 0 {
		t.Errorf("InsertQSO returned invalid ID: %d", id)
	}

	loaded, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	if loaded.Call != "SP9MOA" {
		t.Errorf("call = %q", loaded.Call)
	}
	if loaded.Band != "20m" {
		t.Errorf("band = %q", loaded.Band)
	}
	if loaded.Mode != "SSB" {
		t.Errorf("mode = %q", loaded.Mode)
	}
	if loaded.Freq != 14.200 {
		t.Errorf("freq = %f", loaded.Freq)
	}
	if loaded.RSTSent != "59" {
		t.Errorf("rst_sent = %q", loaded.RSTSent)
	}
	if loaded.RSTRcvd != "59" {
		t.Errorf("rst_rcvd = %q", loaded.RSTRcvd)
	}
	if loaded.Name != "Szymon" {
		t.Errorf("name = %q", loaded.Name)
	}
	if loaded.Operator != "OP" {
		t.Errorf("operator = %q", loaded.Operator)
	}
	if loaded.StationCallsign != "SP9MOA" {
		t.Errorf("station_callsign = %q", loaded.StationCallsign)
	}
	if loaded.MyGridSquare != "KO00ca" {
		t.Errorf("my_gridsquare = %q", loaded.MyGridSquare)
	}
	if loaded.Source != "manual" {
		t.Errorf("source = %q", loaded.Source)
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}

func TestInsertQSO_DefaultsSource(t *testing.T) {
	db := newTempDB(t)
	q := validQSO()
	q.Source = "" // empty → should default to "manual"
	id := mustInsertQSO(t, db, q)

	loaded, _ := GetQSOByID(db, id)
	if loaded.Source != "manual" {
		t.Errorf("source = %q; want manual (default)", loaded.Source)
	}
}

func TestListQSOs_ReturnsInserted(t *testing.T) {
	db := newTempDB(t)

	mustInsertQSO(t, db, &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"})
	mustInsertQSO(t, db, &qso.QSO{Call: "B2B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW"})

	qsos, err := ListQSOs(db, 10)
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) != 2 {
		t.Fatalf("expected 2 QSOs, got %d", len(qsos))
	}

	// Most recent first (ORDER BY id DESC).
	if qsos[0].Call != "B2B" {
		t.Errorf("first QSO call = %q; want B2B (most recent)", qsos[0].Call)
	}
	if qsos[1].Call != "A1A" {
		t.Errorf("second QSO call = %q; want A1A", qsos[1].Call)
	}
}

func TestListAllQSOs(t *testing.T) {
	db := newTempDB(t)

	for i := 0; i < 5; i++ {
		mustInsertQSO(t, db, validQSO())
	}

	qsos, err := ListAllQSOs(db)
	if err != nil {
		t.Fatalf("ListAllQSOs: %v", err)
	}
	if len(qsos) != 5 {
		t.Errorf("expected 5 QSOs, got %d", len(qsos))
	}
}

func TestGetQSOByID_NotFound(t *testing.T) {
	db := newTempDB(t)
	_, err := GetQSOByID(db, 99999)
	if err == nil {
		t.Error("GetQSOByID should return error for non-existent ID")
	}
}

// =============================================================================
// UpdateQSO tests
// =============================================================================

func TestUpdateQSO_PersistsChanges(t *testing.T) {
	db := newTempDB(t)

	id := mustInsertQSO(t, db, validQSO())

	loaded, _ := GetQSOByID(db, id)
	loaded.Call = "SP9XYZ"
	loaded.Band = "40m"
	loaded.RSTSent = "599"
	loaded.Comment = "Updated test"

	if err := UpdateQSO(db, loaded); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}

	reloaded, _ := GetQSOByID(db, id)
	if reloaded.Call != "SP9XYZ" {
		t.Errorf("call = %q; want SP9XYZ", reloaded.Call)
	}
	if reloaded.Band != "40m" {
		t.Errorf("band = %q; want 40m", reloaded.Band)
	}
	if reloaded.RSTSent != "599" {
		t.Errorf("rst_sent = %q; want 599", reloaded.RSTSent)
	}
	if reloaded.Comment != "Updated test" {
		t.Errorf("comment = %q", reloaded.Comment)
	}
	if reloaded.ID != id {
		t.Errorf("ID changed from %d to %d", id, reloaded.ID)
	}
}

// =============================================================================
// DeleteQSO / PurgeQSOs tests
// =============================================================================

func TestDeleteQSO_RemovesRecord(t *testing.T) {
	db := newTempDB(t)

	id1 := mustInsertQSO(t, db, &qso.QSO{Call: "A1A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"})
	id2 := mustInsertQSO(t, db, &qso.QSO{Call: "B2B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW"})

	if err := DeleteQSO(db, id1); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}

	// Deleted QSO should not be found.
	if _, err := GetQSOByID(db, id1); err == nil {
		t.Error("GetQSOByID should return error for deleted QSO")
	}

	// Second QSO should still exist.
	q2, err := GetQSOByID(db, id2)
	if err != nil {
		t.Fatalf("GetQSOByID for non-deleted QSO: %v", err)
	}
	if q2.Call != "B2B" {
		t.Errorf("remaining QSO call = %q", q2.Call)
	}
}

func TestPurgeQSOs_RemovesAll(t *testing.T) {
	db := newTempDB(t)

	mustInsertQSO(t, db, validQSO())
	mustInsertQSO(t, db, validQSO())

	if err := PurgeQSOs(db); err != nil {
		t.Fatalf("PurgeQSOs: %v", err)
	}

	qsos, _ := ListAllQSOs(db)
	if len(qsos) != 0 {
		t.Errorf("expected 0 QSOs after purge, got %d", len(qsos))
	}
}

// =============================================================================
// UpdateWavelogStatus tests
// =============================================================================

func TestUpdateWavelogStatus(t *testing.T) {
	db := newTempDB(t)

	id := mustInsertQSO(t, db, validQSO())

	// Initially empty/default.
	q, _ := GetQSOByID(db, id)
	if q.WavelogUploaded != "" {
		t.Errorf("initial wavelog_uploaded = %q; want empty", q.WavelogUploaded)
	}

	// Set to "yes".
	if err := UpdateWavelogStatus(db, id, "yes"); err != nil {
		t.Fatalf("UpdateWavelogStatus yes: %v", err)
	}
	q, _ = GetQSOByID(db, id)
	if q.WavelogUploaded != "yes" {
		t.Errorf("wavelog_uploaded = %q; want yes", q.WavelogUploaded)
	}

	// Set to "no".
	if err := UpdateWavelogStatus(db, id, "no"); err != nil {
		t.Fatalf("UpdateWavelogStatus no: %v", err)
	}
	q, _ = GetQSOByID(db, id)
	if q.WavelogUploaded != "no" {
		t.Errorf("wavelog_uploaded = %q; want no", q.WavelogUploaded)
	}
}

// =============================================================================
// NormalizeStationFields tests
// =============================================================================

func TestNormalizeStationFields_SelectedIDs(t *testing.T) {
	db := newTempDB(t)

	id1 := mustInsertQSO(t, db, &qso.QSO{
		Call: "A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		StationCallsign: "OLD_A", Operator: "OldOp1", MyGridSquare: "AA00aa",
	})
	id2 := mustInsertQSO(t, db, &qso.QSO{
		Call: "B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW",
		StationCallsign: "OLD_B", Operator: "OldOp2", MyGridSquare: "BB00bb",
	})
	id3 := mustInsertQSO(t, db, &qso.QSO{
		Call: "C", QSODate: "20240503", TimeOn: "140000", Band: "15m", Mode: "FT8",
		StationCallsign: "OLD_C", Operator: "OldOp3", MyGridSquare: "CC00cc",
	})

	// Normalize only QSOs 1 and 3.
	if err := NormalizeStationFields(db, []int64{id1, id3}, "NEW_CALL", "NewOp", "NEW00"); err != nil {
		t.Fatalf("NormalizeStationFields: %v", err)
	}

	// QSO 1 should be normalized.
	q1, _ := GetQSOByID(db, id1)
	if q1.StationCallsign != "NEW_CALL" || q1.Operator != "NewOp" || q1.MyGridSquare != "NEW00" {
		t.Errorf("q1 not normalized: call=%q op=%q grid=%q", q1.StationCallsign, q1.Operator, q1.MyGridSquare)
	}

	// QSO 2 should be unchanged.
	q2, _ := GetQSOByID(db, id2)
	if q2.StationCallsign != "OLD_B" || q2.Operator != "OldOp2" || q2.MyGridSquare != "BB00bb" {
		t.Errorf("q2 was changed: call=%q op=%q grid=%q", q2.StationCallsign, q2.Operator, q2.MyGridSquare)
	}

	// QSO 3 should be normalized.
	q3, _ := GetQSOByID(db, id3)
	if q3.StationCallsign != "NEW_CALL" || q3.Operator != "NewOp" || q3.MyGridSquare != "NEW00" {
		t.Errorf("q3 not normalized: call=%q op=%q grid=%q", q3.StationCallsign, q3.Operator, q3.MyGridSquare)
	}
}

func TestNormalizeStationFields_EmptyIDs(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{
		Call: "A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB",
		StationCallsign: "OLD", Operator: "OldOp", MyGridSquare: "AA00aa",
	})

	// Normalize with empty IDs should succeed without changing anything.
	if err := NormalizeStationFields(db, nil, "NEW", "NewOp", "NEW00"); err != nil {
		t.Fatalf("NormalizeStationFields with empty IDs: %v", err)
	}

	q, _ := GetQSOByID(db, id)
	if q.StationCallsign != "OLD" {
		t.Error("QSO should be unchanged when no IDs given")
	}
}

// =============================================================================
// CountQSOs tests
// =============================================================================

func TestCountQSOs(t *testing.T) {
	db := newTempDB(t)

	mustInsertQSO(t, db, &qso.QSO{Call: "A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", Source: "wsjtx", WavelogUploaded: "yes"})
	mustInsertQSO(t, db, &qso.QSO{Call: "B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW", Source: "manual"})
	mustInsertQSO(t, db, &qso.QSO{Call: "C", QSODate: "20240503", TimeOn: "140000", Band: "15m", Mode: "FT8", Source: "wsjtx"})

	c, err := CountQSOs(db)
	if err != nil {
		t.Fatalf("CountQSOs: %v", err)
	}
	if c.Total != 3 {
		t.Errorf("total = %d; want 3", c.Total)
	}
	if c.FromWSJTX != 2 {
		t.Errorf("from_wsjtx = %d; want 2", c.FromWSJTX)
	}
	if c.ToWavelog != 1 {
		t.Errorf("to_wavelog = %d; want 1", c.ToWavelog)
	}
}

func TestCountQSOs_Empty(t *testing.T) {
	db := newTempDB(t)
	c, err := CountQSOs(db)
	if err != nil {
		t.Fatalf("CountQSOs on empty DB: %v", err)
	}
	if c.Total != 0 {
		t.Errorf("total = %d; want 0", c.Total)
	}
}
