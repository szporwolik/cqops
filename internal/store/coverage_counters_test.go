package store

import (
	"database/sql"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// coverageMatchesScan is the equivalence check the counters have to pass: the
// maintained numbers must agree with a fresh pass over qsos after every kind of
// write. A drift here would turn a NEW into an UNKNOWN on the partner panel
// without any query failing.
func coverageMatchesScan(t *testing.T, db *sql.DB, step string) {
	t.Helper()
	want, err := scanCoverage(db)
	if err != nil {
		t.Fatalf("%s: scan: %v", step, err)
	}
	got, ok, err := readCoverageCounters(db)
	if err != nil {
		t.Fatalf("%s: counters: %v", step, err)
	}
	if !ok {
		t.Fatalf("%s: coverage counters missing — Coverage would scan on every lookup", step)
	}
	if got != want {
		t.Fatalf("%s: counters %+v, scan %+v", step, got, want)
	}
}

// TestCoverageCountersTrackWrites walks the write paths that change the
// completeness of a log and checks the counters against the scan each time.
func TestCoverageCountersTrackWrites(t *testing.T) {
	db := newTempDB(t)
	coverageMatchesScan(t, db, "empty log")

	complete := &qso.QSO{
		Call: "SP9MOA", Band: "20m", Mode: "FT8", DXCC: "269",
		QSODate: "20260501", TimeOn: "120000", GridSquare: "JO90AA",
	}
	completeID := mustInsertQSO(t, db, complete)
	coverageMatchesScan(t, db, "insert a complete contact")

	partial := &qso.QSO{
		Call: "XX1AA", Band: "20m", Mode: "FT8",
		QSODate: "20260502", TimeOn: "120000", GridSquare: "AA00",
	}
	mustInsertQSO(t, db, partial)
	coverageMatchesScan(t, db, "insert a contact with no DXCC")

	cov, err := Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.UnresolvedDXCC != 1 || cov.DXCCComplete() {
		t.Fatalf("coverage after the partial contact: %+v, want one unresolved entity", cov)
	}

	// Filling the last missing entity must restore the verdict immediately —
	// this is the acceptance case: "worked status can be trusted again".
	partial.DXCC = "291"
	if err := UpdateQSO(db, partial); err != nil {
		t.Fatalf("UpdateQSO (fill dxcc): %v", err)
	}
	coverageMatchesScan(t, db, "fill the missing DXCC")
	if cov, err = Coverage(db); err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.UnresolvedDXCC != 0 || !cov.DXCCComplete() {
		t.Errorf("coverage after filling DXCC: %+v, want complete", cov)
	}

	// Taking the band away has to show up as missing band data.
	partial.Band = ""
	if err := UpdateQSO(db, partial); err != nil {
		t.Fatalf("UpdateQSO (clear band): %v", err)
	}
	coverageMatchesScan(t, db, "clear the band")
	if cov, err = Coverage(db); err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.MissingBand != 1 || cov.BandComplete() {
		t.Errorf("coverage after clearing the band: %+v, want one missing band", cov)
	}

	// Deletion returns the log to the previous state, and never goes negative.
	if err := DeleteQSO(db, completeID); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}
	coverageMatchesScan(t, db, "delete a contact")
	if err := DeleteQSO(db, partial.ID); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}
	coverageMatchesScan(t, db, "delete the last contact")
	if cov, err = Coverage(db); err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.QSOCount != 0 || cov.UnresolvedDXCC != 0 || cov.MissingBand != 0 || cov.MissingMode != 0 {
		t.Errorf("coverage of an emptied log = %+v, want all zero", cov)
	}
}

// TestCoverageFallsBackToTheScan: an installation whose index was built before
// the counters existed must still get correct answers, just slower, and the
// next rebuild has to seed them.
func TestCoverageFallsBackToTheScan(t *testing.T) {
	db := newTempDB(t)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "XX1AA", Band: "", Mode: "FT8",
		QSODate: "20260502", TimeOn: "120000",
	})
	if _, err := db.Exec(`DELETE FROM worked_index_meta WHERE key LIKE 'coverage\_%' ESCAPE '\'`); err != nil {
		t.Fatalf("drop counters: %v", err)
	}
	if _, ok, err := readCoverageCounters(db); ok || err != nil {
		t.Fatalf("counters still readable after being dropped (ok=%v err=%v)", ok, err)
	}
	cov, err := Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.QSOCount != 1 || cov.UnresolvedDXCC != 1 || cov.MissingBand != 1 {
		t.Errorf("fallback coverage = %+v, want 1 QSO, 1 unresolved, 1 missing band", cov)
	}

	if err := RebuildWorkedIndex(db); err != nil {
		t.Fatalf("RebuildWorkedIndex: %v", err)
	}
	coverageMatchesScan(t, db, "after a rebuild")
}
