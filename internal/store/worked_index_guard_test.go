package store

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// queryPlan returns the EXPLAIN QUERY PLAN detail lines joined into one
// string, so a test can assert how SQLite resolves a statement.
func queryPlan(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN: %v", err)
	}
	defer rows.Close()
	var details []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("scan plan row: %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("plan rows: %v", err)
	}
	return strings.Join(details, " | ")
}

// TestWorkedIndexLookupShapesUseThePrimaryKey pins the four lookup shapes the
// worked panel promises (DXCC, DXCC+band, DXCC+mode, DXCC+band+mode). They are
// all served by the (dxcc, band, mode) primary key; a future schema change
// that drops or reorders that key turns these lookups into table scans — this
// test fails instead of the app getting slower on large logbooks.
func TestWorkedIndexLookupShapesUseThePrimaryKey(t *testing.T) {
	db := newTempDB(t)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "MFSK", Submode: "FT8", DXCC: "269",
	})

	shapes := []struct {
		name  string
		query string
		args  []any
	}{
		{"dxcc", `SELECT qso_count FROM worked_dxcc WHERE dxcc = ? AND band = '' AND mode = ''`, []any{"269"}},
		{"dxcc+band", `SELECT qso_count FROM worked_dxcc WHERE dxcc = ? AND band = ? AND mode = ''`, []any{"269", "20m"}},
		{"dxcc+mode", `SELECT qso_count FROM worked_dxcc WHERE dxcc = ? AND band = '' AND mode = ?`, []any{"269", "FT8"}},
		{"dxcc+band+mode", `SELECT qso_count FROM worked_dxcc WHERE dxcc = ? AND band = ? AND mode = ?`, []any{"269", "20m", "FT8"}},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			plan := queryPlan(t, db, s.query, s.args...)
			if strings.Contains(plan, "SCAN worked_dxcc") {
				t.Fatalf("query plan scans the whole index: %s", plan)
			}
			if !strings.Contains(plan, "SEARCH worked_dxcc") {
				t.Fatalf("query plan does not seek the index: %s", plan)
			}
		})
	}
}

// TestWorkedIndexFollowsTransactionRollback pins the atomicity contract: the
// normalized qsos row and its aggregates are written in ONE transaction, so an
// abandoned bulk-import chunk must leave both untouched.
func TestWorkedIndexFollowsTransactionRollback(t *testing.T) {
	db := newTempDB(t)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", GridSquare: "JO90", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269",
	})

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := InsertQSOTx(tx, &qso.QSO{
		Call: "DL1AA", GridSquare: "JN58", QSODate: "20240502", TimeOn: "130000",
		Band: "40m", Mode: "SSB", DXCC: "230",
	}); err != nil {
		t.Fatalf("InsertQSOTx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if got := workedCountOrZero(t, db, `SELECT COUNT(*) FROM qsos`); got != 1 {
		t.Errorf("qsos rows after rollback = %d, want 1", got)
	}
	if got := workedCountOrZero(t, db, `SELECT COUNT(*) FROM worked_call WHERE base_call = 'DL1AA'`); got != 0 {
		t.Errorf("worked_call(DL1AA) = %d after rollback, want 0", got)
	}
	if got := workedCountOrZero(t, db, `SELECT COUNT(*) FROM worked_dxcc WHERE dxcc = '230'`); got != 0 {
		t.Errorf("worked_dxcc(230) rows = %d after rollback, want 0", got)
	}
	if got := workedCountOrZero(t, db, `SELECT COUNT(*) FROM worked_grid WHERE grid4 = 'JN58'`); got != 0 {
		t.Errorf("worked_grid(JN58) = %d after rollback, want 0", got)
	}
	// The committed QSO keeps its full aggregate set — the rollback must not
	// have unindexed it.
	for _, query := range []string{
		`SELECT qso_count FROM worked_call WHERE base_call = 'SP9MOA'`,
		`SELECT qso_count FROM worked_grid WHERE grid4 = 'JO90'`,
		`SELECT qso_count FROM worked_dxcc WHERE dxcc = '269' AND band = '20m' AND mode = 'FT8'`,
	} {
		if got := workedCountOrZero(t, db, query); got != 1 {
			t.Errorf("%s = %d after rollback, want 1", query, got)
		}
	}
}

// TestWorkedIndexNeverGoesNegative covers the decrement edge cases: removing a
// row that was never indexed, and removing every row of a key. Counts may
// never go below zero and zeroed rows must be removed rather than left behind.
func TestWorkedIndexNeverGoesNegative(t *testing.T) {
	db := newTempDB(t)
	id := mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", GridSquare: "JO90", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269",
	})

	// Deleting an unknown id is a no-op: no aggregate may be touched.
	if err := DeleteQSO(db, 99999); err != nil {
		t.Fatalf("DeleteQSO(unknown): %v", err)
	}
	for _, query := range []string{
		`SELECT qso_count FROM worked_call WHERE base_call = 'SP9MOA'`,
		`SELECT qso_count FROM worked_grid WHERE grid4 = 'JO90'`,
		`SELECT qso_count FROM worked_dxcc WHERE dxcc = '269' AND band = '20m' AND mode = 'FT8'`,
	} {
		if got := workedCountOrZero(t, db, query); got != 1 {
			t.Errorf("%s = %d after deleting an unknown id, want 1", query, got)
		}
	}

	if err := DeleteQSO(db, id); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}
	for _, table := range []string{"worked_call", "worked_grid", "worked_dxcc"} {
		if got := workedCountOrZero(t, db,
			`SELECT COUNT(*) FROM `+table+` WHERE qso_count <= 0`); got != 0 {
			t.Errorf("%s keeps %d non-positive rows after the last delete", table, got)
		}
		if got := workedCountOrZero(t, db, `SELECT COUNT(*) FROM `+table); got != 0 {
			t.Errorf("%s rows = %d after the last delete, want 0", table, got)
		}
	}
}
