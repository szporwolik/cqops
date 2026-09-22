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

// TestWavelogDirtyRoundtrip verifies the durable pending-sync flag:
// set, read and clear through the dedicated accessors.
func TestWavelogDirtyRoundtrip(t *testing.T) {
	db := newTempDB(t)

	id := mustInsertQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501",
		TimeOn: "120000", Band: "20m", Mode: "SSB", WavelogID: 77})

	dirty, err := QSOHasPendingSync(db, id)
	if err != nil {
		t.Fatalf("QSOHasPendingSync: %v", err)
	}
	if dirty {
		t.Error("fresh QSO should not have pending sync")
	}

	if err := SetWavelogDirty(db, id, true); err != nil {
		t.Fatalf("SetWavelogDirty(true): %v", err)
	}
	dirty, err = QSOHasPendingSync(db, id)
	if err != nil || !dirty {
		t.Errorf("pending sync after set = %v (err=%v), want true", dirty, err)
	}

	if err := SetWavelogDirty(db, id, false); err != nil {
		t.Fatalf("SetWavelogDirty(false): %v", err)
	}
	dirty, err = QSOHasPendingSync(db, id)
	if err != nil || dirty {
		t.Errorf("pending sync after clear = %v (err=%v), want false", dirty, err)
	}
}

// TestSaveQSOForSyncAtomicDirtyMark verifies the edit, the dirty flag and the
// pending-sync revision change together, and that only an acknowledgement for
// the CURRENT revision clears the flag — an older revision must not.
func TestSaveQSOForSyncAtomicDirtyMark(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501",
		TimeOn: "120000", Band: "20m", Mode: "SSB", WavelogID: 77})

	q1 := &qso.QSO{ID: id, Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB", Comment: "edit one", WavelogID: 77}
	rev1, err := SaveQSOForSync(db, q1)
	if err != nil {
		t.Fatalf("SaveQSOForSync: %v", err)
	}
	if rev1 != 1 {
		t.Fatalf("rev1 = %d, want 1", rev1)
	}
	row, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if row.Comment != "edit one" || !row.WavelogDirty {
		t.Fatalf("row after save = comment %q dirty %v; want edit one, true", row.Comment, row.WavelogDirty)
	}

	q2 := &qso.QSO{ID: id, Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB", Comment: "edit two", WavelogID: 77}
	rev2, err := SaveQSOForSync(db, q2)
	if err != nil {
		t.Fatalf("SaveQSOForSync #2: %v", err)
	}
	if rev2 != 2 {
		t.Fatalf("rev2 = %d, want 2", rev2)
	}

	// An acknowledgement for the OLDER revision must not clear the flag.
	if err := ClearWavelogDirtyIfRevision(db, id, rev1); err != nil {
		t.Fatalf("ClearWavelogDirtyIfRevision(old): %v", err)
	}
	dirty, err := QSOHasPendingSync(db, id)
	if err != nil || !dirty {
		t.Errorf("pending sync after old-revision clear = %v (err=%v), want true", dirty, err)
	}

	// The CURRENT revision's acknowledgement clears it.
	if err := ClearWavelogDirtyIfRevision(db, id, rev2); err != nil {
		t.Fatalf("ClearWavelogDirtyIfRevision(current): %v", err)
	}
	dirty, err = QSOHasPendingSync(db, id)
	if err != nil || dirty {
		t.Errorf("pending sync after current-revision clear = %v (err=%v), want false", dirty, err)
	}
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

// TestUpdateQSO_RecomputesDerivedFields covers the derived-metadata policy:
// base_call always follows the call; DXCC is invalidated when the identity
// changes without a fresh value, preserved when the call is unchanged, and
// honored when the caller supplies one.
func TestUpdateQSO_RecomputesDerivedFields(t *testing.T) {
	db := newTempDB(t)

	q := validQSO()
	q.Call = "SP9ABC"
	q.DXCC = "269"
	id := mustInsertQSO(t, db, q)

	baseCallOf := func() string {
		t.Helper()
		var bc string
		if err := db.QueryRow(`SELECT base_call FROM qsos WHERE id=?`, id).Scan(&bc); err != nil {
			t.Fatalf("read base_call: %v", err)
		}
		return bc
	}
	dxccOf := func() string {
		t.Helper()
		var d string
		if err := db.QueryRow(`SELECT dxcc FROM qsos WHERE id=?`, id).Scan(&d); err != nil {
			t.Fatalf("read dxcc: %v", err)
		}
		return d
	}

	// 1. Call changed, no fresh DXCC: base_call recomputed, dxcc invalidated.
	loaded, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	loaded.Call = "W1AW"
	loaded.DXCC = ""
	if err := UpdateQSO(db, loaded); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if bc := baseCallOf(); bc != "W1AW" {
		t.Errorf("base_call = %q after call change; want W1AW", bc)
	}
	if d := dxccOf(); d != "" {
		t.Errorf("dxcc = %q after call change; want invalidated (empty)", d)
	}

	// 2. Call unchanged with no DXCC in the struct: dxcc stays empty (the
	// backfill will fill it) — nothing resurrects a stale value.
	again, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	again.DXCC = ""
	if err := UpdateQSO(db, again); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if d := dxccOf(); d != "" {
		t.Errorf("dxcc = %q after unchanged-call update; want still empty", d)
	}

	// 3. Caller-supplied DXCC is written and survives an unchanged update.
	again.DXCC = "291"
	if err := UpdateQSO(db, again); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if d := dxccOf(); d != "291" {
		t.Errorf("dxcc = %q; want 291 (caller-supplied)", d)
	}
	again, err = GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	again.DXCC = "" // edit form never carries DXCC
	if err := UpdateQSO(db, again); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if d := dxccOf(); d != "291" {
		t.Errorf("dxcc = %q after unchanged-call edit; want 291 preserved", d)
	}
}

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

	qsos, err := ListQSOs(db, 10, "")
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

// TestListAllQSOs_AcrossPages verifies keyset pagination returns every row
// past the page boundary (the old OFFSET loop would also pass, but this pins
// the new linear scan contract).
func TestListAllQSOs_AcrossPages(t *testing.T) {
	db := newTempDB(t)

	const total = 515
	for i := 0; i < total; i++ {
		mustInsertQSO(t, db, validQSO())
	}

	qsos, err := ListAllQSOs(db)
	if err != nil {
		t.Fatalf("ListAllQSOs: %v", err)
	}
	if len(qsos) != total {
		t.Errorf("expected %d QSOs, got %d", total, len(qsos))
	}
}

func TestListUnsentQSOs(t *testing.T) {
	db := newTempDB(t)

	sent := validQSO()
	sent.WavelogID = 42
	unsent := validQSO()

	mustInsertQSO(t, db, sent)
	mustInsertQSO(t, db, unsent)

	n, err := CountUnsentQSOs(db)
	if err != nil {
		t.Fatalf("CountUnsentQSOs: %v", err)
	}
	if n != 1 {
		t.Errorf("unsent count = %d, want 1", n)
	}

	rows, err := ListUnsentQSOs(db)
	if err != nil {
		t.Fatalf("ListUnsentQSOs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 unsent row, got %d", len(rows))
	}
	if rows[0].WavelogID != 0 {
		t.Errorf("unsent row wavelog_id = %d, want 0", rows[0].WavelogID)
	}

	// The list captures the pending-sync revision WITH the row data, so
	// uploaders can pair the snapshot they send with the revision it was
	// taken at — pairing old data with a newer revision falsely marks rows
	// clean after the id attach.
	if rows[0].WavelogDirtyRev != 0 {
		t.Errorf("fresh row revision = %d, want 0", rows[0].WavelogDirtyRev)
	}
	edited := rows[0]
	edited.Comment = "newer edit"
	if err := UpdateQSO(db, &edited); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	again, err := ListUnsentQSOs(db)
	if err != nil {
		t.Fatalf("ListUnsentQSOs after edit: %v", err)
	}
	if len(again) != 1 {
		t.Fatalf("expected 1 unsent row after edit, got %d", len(again))
	}
	if again[0].WavelogDirtyRev != 1 {
		t.Errorf("captured revision = %d, want 1 (the edit must bump the revision)", again[0].WavelogDirtyRev)
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
// SetWavelogID tests
// =============================================================================

func TestSetWavelogID(t *testing.T) {
	db := newTempDB(t)

	id := mustInsertQSO(t, db, validQSO())

	// Initially no remote id.
	q, _ := GetQSOByID(db, id)
	if q.WavelogID != 0 {
		t.Errorf("initial wavelog_id = %d; want 0", q.WavelogID)
	}

	if err := SetWavelogID(db, id, 42); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}
	q, _ = GetQSOByID(db, id)
	if q.WavelogID != 42 {
		t.Errorf("wavelog_id = %d; want 42", q.WavelogID)
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

	mustInsertQSO(t, db, &qso.QSO{Call: "A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", Source: "wsjtx", WavelogID: 1})
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

func TestUpdateQSOEnrichment_FillsEmpty(t *testing.T) {
	db := newTempDB(t)

	q := validQSO()
	q.Name = ""
	q.QTH = ""
	q.Country = ""
	q.GridSquare = ""
	id, err := InsertQSO(db, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	UpdateQSOEnrichment(db, id, EnrichmentData{
		Name:       "Jan",
		QTH:        "Warszawa",
		Country:    "Poland",
		GridSquare: "KO02",
	})

	stored, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Name != "Jan" {
		t.Errorf("Name = %q, want Jan", stored.Name)
	}
	if stored.QTH != "Warszawa" {
		t.Errorf("QTH = %q, want Warszawa", stored.QTH)
	}
	if stored.Country != "Poland" {
		t.Errorf("Country = %q, want Poland", stored.Country)
	}
	if stored.GridSquare != "KO02" {
		t.Errorf("GridSquare = %q, want KO02", stored.GridSquare)
	}
}

func TestUpdateQSOEnrichment_PreservesExisting(t *testing.T) {
	db := newTempDB(t)

	q := validQSO()
	q.Name = "Original"
	q.QTH = "OriginalQTH"
	id, err := InsertQSO(db, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	// Enrichment should not overwrite existing data.
	UpdateQSOEnrichment(db, id, EnrichmentData{
		Name: "Enriched",
		QTH:  "EnrichedQTH",
	})

	stored, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Name != "Original" {
		t.Errorf("Name = %q, want Original (not overwritten)", stored.Name)
	}
	if stored.QTH != "OriginalQTH" {
		t.Errorf("QTH = %q, want OriginalQTH (not overwritten)", stored.QTH)
	}
}

func TestUpdateQSOEnrichment_NoopEmpty(t *testing.T) {
	db := newTempDB(t)

	q := validQSO()
	id, err := InsertQSO(db, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	// Empty enrichment data should be a no-op.
	UpdateQSOEnrichment(db, id, EnrichmentData{})
	UpdateQSOEnrichment(db, id, EnrichmentData{Name: "", QTH: "", Country: ""})

	stored, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Name != q.Name {
		t.Errorf("Name changed unexpectedly")
	}
}

// =============================================================================
// CountQSOsForContest tests
// =============================================================================

func TestCountQSOsForContest_Filtered(t *testing.T) {
	db := newTempDB(t)

	// Insert QSOs with different contest IDs.
	mustInsertQSO(t, db, &qso.QSO{Call: "A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "c1", Source: "wsjtx", WavelogID: 1})
	mustInsertQSO(t, db, &qso.QSO{Call: "B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW", ContestID: "c1", Source: "manual"})
	mustInsertQSO(t, db, &qso.QSO{Call: "C", QSODate: "20240503", TimeOn: "140000", Band: "15m", Mode: "FT8", ContestID: "c2", Source: "wsjtx", WavelogID: 1})
	mustInsertQSO(t, db, &qso.QSO{Call: "D", QSODate: "20240504", TimeOn: "150000", Band: "10m", Mode: "SSB"})

	c, err := CountQSOsForContest(db, "c1")
	if err != nil {
		t.Fatalf("CountQSOsForContest: %v", err)
	}
	if c.Total != 2 {
		t.Errorf("total = %d; want 2", c.Total)
	}
	if c.FromWSJTX != 1 {
		t.Errorf("from_wsjtx = %d; want 1", c.FromWSJTX)
	}
	if c.ToWavelog != 1 {
		t.Errorf("to_wavelog = %d; want 1", c.ToWavelog)
	}
}

func TestCountQSOsForContest_Empty(t *testing.T) {
	db := newTempDB(t)

	c, err := CountQSOsForContest(db, "")
	if err != nil {
		t.Fatalf("CountQSOsForContest(empty): %v", err)
	}
	if c.Total != 0 {
		t.Errorf("total = %d; want 0", c.Total)
	}
}

func TestCountQSOsForContest_All(t *testing.T) {
	db := newTempDB(t)

	mustInsertQSO(t, db, &qso.QSO{Call: "A", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "c1"})
	mustInsertQSO(t, db, &qso.QSO{Call: "B", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW"})

	// Empty contestID should count all QSOs.
	c, err := CountQSOsForContest(db, "")
	if err != nil {
		t.Fatalf("CountQSOsForContest(all): %v", err)
	}
	if c.Total != 2 {
		t.Errorf("total = %d; want 2", c.Total)
	}
}

// =============================================================================
// ListQSOsPage contest filter tests
// =============================================================================

func TestListQSOsPage_ContestFilter(t *testing.T) {
	db := newTempDB(t)

	mustInsertQSO(t, db, &qso.QSO{Call: "A1", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "c1"})
	mustInsertQSO(t, db, &qso.QSO{Call: "B2", QSODate: "20240502", TimeOn: "130000", Band: "40m", Mode: "CW", ContestID: "c1"})
	mustInsertQSO(t, db, &qso.QSO{Call: "C3", QSODate: "20240503", TimeOn: "140000", Band: "15m", Mode: "FT8", ContestID: "c2"})
	mustInsertQSO(t, db, &qso.QSO{Call: "D4", QSODate: "20240504", TimeOn: "150000", Band: "10m", Mode: "SSB"})

	// Page filtered by c1 — should return 2 rows.
	qsos, err := ListQSOsPage(db, 10, 0, "c1", false)
	if err != nil {
		t.Fatalf("ListQSOsPage(c1): %v", err)
	}
	if len(qsos) != 2 {
		t.Errorf("len = %d; want 2", len(qsos))
	}
	for _, q := range qsos {
		if q.ContestID != "c1" {
			t.Errorf("unexpected ContestID %q in filtered results", q.ContestID)
		}
	}

	// Page filtered by c2 — should return 1 row.
	qsos, err = ListQSOsPage(db, 10, 0, "c2", false)
	if err != nil {
		t.Fatalf("ListQSOsPage(c2): %v", err)
	}
	if len(qsos) != 1 {
		t.Errorf("len = %d; want 1", len(qsos))
	}
	if qsos[0].Call != "C3" {
		t.Errorf("call = %q; want C3", qsos[0].Call)
	}

	// Page with empty contestID (no filter) — should return all 4.
	qsos, err = ListQSOsPage(db, 10, 0, "", false)
	if err != nil {
		t.Fatalf("ListQSOsPage(all): %v", err)
	}
	if len(qsos) != 4 {
		t.Errorf("len = %d; want 4", len(qsos))
	}
}

func TestListQSOsPage_ContestFilterPagination(t *testing.T) {
	db := newTempDB(t)

	// Insert 5 QSOs, 3 in contest c1.
	for i := 0; i < 3; i++ {
		mustInsertQSO(t, db, &qso.QSO{Call: "C" + string(rune('A'+i)), QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "c1"})
	}
	mustInsertQSO(t, db, &qso.QSO{Call: "X1", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "c2"})
	mustInsertQSO(t, db, &qso.QSO{Call: "X2", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"})

	// Page 1 of c1 — limit 2, offset 0.
	qsos, err := ListQSOsPage(db, 2, 0, "c1", false)
	if err != nil {
		t.Fatalf("ListQSOsPage(c1, limit 2): %v", err)
	}
	if len(qsos) != 2 {
		t.Errorf("len = %d; want 2", len(qsos))
	}

	// Page 2 of c1 — limit 2, offset 2. Should return 1 remaining.
	qsos, err = ListQSOsPage(db, 2, 2, "c1", false)
	if err != nil {
		t.Fatalf("ListQSOsPage(c1, offset 2): %v", err)
	}
	if len(qsos) != 1 {
		t.Errorf("len = %d; want 1", len(qsos))
	}
}
