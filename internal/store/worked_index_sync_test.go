package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

// workedDXCCDB returns the aggregate count for one DXCC entity (no band/mode),
// or 0 when the aggregate row is absent.
func workedDXCCDB(t *testing.T, db *sql.DB, dxcc string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT qso_count FROM worked_dxcc WHERE dxcc=? AND band='' AND mode=''`, dxcc).Scan(&n); err != nil {
		return 0
	}
	return n
}

// TestUpdateQSOCommentOnlyKeepsDXCCIndex is the regression test for the
// "edit a comment, lose the DXCC" drift: UpdateQSO indexed the caller's
// object, which carries no DXCC when the form loaded the row without one,
// while the SQL deliberately preserves the stored dxcc. The old aggregate was
// removed and never restored, so the contact vanished from the DXCC history.
func TestUpdateQSOCommentOnlyKeepsDXCCIndex(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB", GridSquare: "JO90", DXCC: "269",
	})
	if got := workedDXCCDB(t, db, "269"); got != 1 {
		t.Fatalf("precondition: worked_dxcc(269) = %d, want 1", got)
	}

	q, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	q.DXCC = "" // what a form save sends: DXCC is not a form field
	q.Comment = "edited comment"
	q.UpdatedAt = time.Now().UTC()
	if err := UpdateQSO(db, q); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}

	stored, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID after update: %v", err)
	}
	if stored.DXCC != "269" {
		t.Fatalf("stored dxcc = %q, want 269 (must be preserved by the UPDATE)", stored.DXCC)
	}
	if got := workedDXCCDB(t, db, "269"); got != 1 {
		t.Errorf("worked_dxcc(269) = %d after a comment-only edit, want 1", got)
	}
}

// TestSaveQSOCommentOnlyKeepsDXCCIndex covers the logbook-editor save path,
// which had the same defect.
func TestSaveQSOCommentOnlyKeepsDXCCIndex(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20240502", TimeOn: "130000",
		Band: "40m", Mode: "CW", GridSquare: "JO90", DXCC: "269",
	})

	q, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	q.DXCC = ""
	q.Comment = "editor comment"
	if _, _, err := SaveQSO(db, q); err != nil {
		t.Fatalf("SaveQSO: %v", err)
	}

	if got := workedDXCCDB(t, db, "269"); got != 1 {
		t.Errorf("worked_dxcc(269) = %d after a comment-only editor save, want 1", got)
	}
	var grid int
	if err := db.QueryRow(`SELECT qso_count FROM worked_grid WHERE grid4='JO90'`).Scan(&grid); err != nil {
		t.Fatalf("query worked_grid: %v", err)
	}
	if grid != 1 {
		t.Errorf("worked_grid(JO90) = %d, want 1", grid)
	}
}

// TestUpdateQSOEnrichmentUpdatesWorkedIndex: enrichment fills dxcc and grid
// after the row is stored, and the index has to follow in the same write.
func TestUpdateQSOEnrichmentUpdatesWorkedIndex(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{
		Call: "DJ7NT", QSODate: "20240503", TimeOn: "140000",
		Band: "20m", Mode: "FT8",
	})
	if got := workedDXCCDB(t, db, "230"); got != 0 {
		t.Fatalf("precondition: worked_dxcc(230) = %d, want 0", got)
	}

	if err := UpdateQSOEnrichment(db, id, EnrichmentData{
		Country:    "Fed. Rep. of Germany",
		DXCC:       "230",
		GridSquare: "JO31",
	}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}

	if got := workedDXCCDB(t, db, "230"); got != 1 {
		t.Errorf("worked_dxcc(230) = %d after enrichment, want 1", got)
	}
	var bandMode int
	if err := db.QueryRow(`SELECT qso_count FROM worked_dxcc WHERE dxcc='230' AND band='20m' AND mode='FT8'`).Scan(&bandMode); err != nil {
		t.Fatalf("query worked_dxcc band+mode: %v", err)
	}
	if bandMode != 1 {
		t.Errorf("worked_dxcc(230,20m,FT8) = %d after enrichment, want 1", bandMode)
	}
	var grid int
	if err := db.QueryRow(`SELECT qso_count FROM worked_grid WHERE grid4='JO31'`).Scan(&grid); err != nil {
		t.Fatalf("query worked_grid: %v", err)
	}
	if grid != 1 {
		t.Errorf("worked_grid(JO31) = %d after enrichment, want 1", grid)
	}
}

// TestBackfillDXCCUpdatesWorkedIndex: the CTY backfill writes dxcc straight
// into qsos; the index must see it without waiting for a full rebuild.
func TestBackfillDXCCUpdatesWorkedIndex(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{
		Call: "F5ABC", QSODate: "20240504", TimeOn: "150000",
		Band: "15m", Mode: "SSB",
	})

	n, err := BackfillDXCC(db, map[int64]string{id: "227"})
	if err != nil {
		t.Fatalf("BackfillDXCC: %v", err)
	}
	if n != 1 {
		t.Fatalf("BackfillDXCC updated %d rows, want 1", n)
	}
	if got := workedDXCCDB(t, db, "227"); got != 1 {
		t.Errorf("worked_dxcc(227) = %d after backfill, want 1", got)
	}

	// Idempotent: the row already carries a DXCC, so a second pass changes
	// nothing and must not duplicate the aggregate.
	again, err := BackfillDXCC(db, map[int64]string{id: "999"})
	if err != nil {
		t.Fatalf("BackfillDXCC (second): %v", err)
	}
	if again != 0 {
		t.Errorf("second BackfillDXCC updated %d rows, want 0", again)
	}
	if got := workedDXCCDB(t, db, "227"); got != 1 {
		t.Errorf("worked_dxcc(227) = %d after a no-op backfill, want 1", got)
	}
	if got := workedDXCCDB(t, db, "999"); got != 0 {
		t.Errorf("worked_dxcc(999) = %d, want 0 (empty dxcc must not be overwritten)", got)
	}
}

// TestMigrateRebuildsDriftedWorkedIndexOnUpgrade: a database written by the
// buggy version carries an index that no longer matches qsos (a comment-only
// edit dropped the DXCC aggregate). Upgrading to the current schema must
// rebuild it once, so existing users are repaired without manual action.
func TestMigrateRebuildsDriftedWorkedIndexOnUpgrade(t *testing.T) {
	db := newTempDB(t)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20240505", TimeOn: "160000",
		Band: "20m", Mode: "SSB", GridSquare: "JO90", DXCC: "269",
	})

	// Simulate the drift and the older schema version.
	if _, err := db.Exec(`DELETE FROM worked_dxcc`); err != nil {
		t.Fatalf("drift worked_dxcc: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM worked_grid`); err != nil {
		t.Fatalf("drift worked_grid: %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 2`); err != nil {
		t.Fatalf("set user_version=2: %v", err)
	}
	if got := workedDXCCDB(t, db, "269"); got != 0 {
		t.Fatalf("precondition: worked_dxcc(269) = %d, want 0 (drifted)", got)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if got := workedDXCCDB(t, db, "269"); got != 1 {
		t.Errorf("worked_dxcc(269) = %d after the upgrade rebuild, want 1", got)
	}
	var grid int
	if err := db.QueryRow(`SELECT qso_count FROM worked_grid WHERE grid4='JO90'`).Scan(&grid); err != nil || grid != 1 {
		t.Errorf("worked_grid(JO90) = %d (err %v) after the upgrade rebuild, want 1", grid, err)
	}

	// The rebuild must not run on every start: a second Migrate keeps the
	// (already correct) index intact.
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if got := workedDXCCDB(t, db, "269"); got != 1 {
		t.Errorf("worked_dxcc(269) = %d after a no-op Migrate, want 1", got)
	}
}

// TestMigrateBackfillsBaseCallBeforeRebuild pins the migration ordering: an
// upgrade from an older database fills base_call, and the one-time index
// rebuild must run *after* that backfill. When the rebuild ran first, the
// contact got its base_call but was missing from worked_call until some later
// rebuild (base_call="SP9MOA", counter 0).
func TestMigrateBackfillsBaseCallBeforeRebuild(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA/P", QSODate: "20240506", TimeOn: "170000",
		Band: "20m", Mode: "SSB",
	})

	// Simulate a pre-backfill database: no base_call, drifted (empty) index.
	if _, err := db.Exec(`UPDATE qsos SET base_call='' WHERE id=?`, id); err != nil {
		t.Fatalf("clear base_call: %v", err)
	}
	for _, table := range []string{"worked_call", "worked_grid", "worked_dxcc"} {
		if _, err := db.Exec(`DELETE FROM ` + table); err != nil {
			t.Fatalf("drift %s: %v", table, err)
		}
	}
	if _, err := db.Exec(`PRAGMA user_version = 2`); err != nil {
		t.Fatalf("set user_version=2: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var baseCall string
	if err := db.QueryRow(`SELECT base_call FROM qsos WHERE id=?`, id).Scan(&baseCall); err != nil {
		t.Fatalf("read base_call: %v", err)
	}
	if baseCall != "SP9MOA" {
		t.Fatalf("base_call = %q after migration, want SP9MOA", baseCall)
	}

	var n int
	if err := db.QueryRow(`SELECT qso_count FROM worked_call WHERE base_call='SP9MOA'`).Scan(&n); err != nil {
		t.Fatalf("worked_call(SP9MOA) missing after migration: %v", err)
	}
	if n != 1 {
		t.Errorf("worked_call(SP9MOA) = %d after migration, want 1", n)
	}
}
