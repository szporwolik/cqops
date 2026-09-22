package store

import (
	"fmt"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// TestListQSOsPageAfterTx_KeysetPages verifies the keyset cursor pages
// strictly after the previous page in the same order ListQSOsPage uses —
// both descending and ascending.
func TestListQSOsPageAfterTx_KeysetPages(t *testing.T) {
	db := newTempDB(t)
	for i := 1; i <= 5; i++ {
		mustInsertQSO(t, db, &qso.QSO{
			Call: fmt.Sprintf("C%d", i), QSODate: "2024050" + string(rune('0'+i)),
			TimeOn: "120000", Band: "20m", Mode: "SSB",
		})
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	// Descending: ids in date order 5,4,3,2,1.
	var got []int64
	var cursor *qso.QSO
	for {
		rows, err := ListQSOsPageAfterTx(tx, 2, "", false, cursor)
		if err != nil {
			t.Fatalf("page desc: %v", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			got = append(got, r.ID)
		}
		cursor = &rows[len(rows)-1]
		if len(rows) < 2 {
			break
		}
	}
	wantDesc := []int64{5, 4, 3, 2, 1}
	if len(got) != len(wantDesc) {
		t.Fatalf("desc pages = %v, want %v", got, wantDesc)
	}
	for i := range wantDesc {
		if got[i] != wantDesc[i] {
			t.Errorf("desc page[%d] = %d, want %d (full: %v)", i, got[i], wantDesc[i], got)
		}
	}

	// Ascending: ids 1..5.
	got = got[:0]
	cursor = nil
	for {
		rows, err := ListQSOsPageAfterTx(tx, 2, "", true, cursor)
		if err != nil {
			t.Fatalf("page asc: %v", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			got = append(got, r.ID)
		}
		cursor = &rows[len(rows)-1]
		if len(rows) < 2 {
			break
		}
	}
	wantAsc := []int64{1, 2, 3, 4, 5}
	for i := range wantAsc {
		if i >= len(got) || got[i] != wantAsc[i] {
			t.Fatalf("asc pages = %v, want %v", got, wantAsc)
		}
	}
}

// TestExportQSOsSnapshot_ConsistentDuringInsert reproduces the export race:
// a QSO inserted while the export is streaming must not duplicate or omit
// any record — the snapshot transaction freezes the view, and the keyset
// cursor never shifts.
func TestExportQSOsSnapshot_ConsistentDuringInsert(t *testing.T) {
	db := newTempDB(t)
	for i := 1; i <= 5; i++ {
		mustInsertQSO(t, db, &qso.QSO{
			Call: fmt.Sprintf("C%d", i), QSODate: "2024050" + string(rune('0'+i)),
			TimeOn: "120000", Band: "20m", Mode: "SSB",
		})
	}

	inserted := false
	seen := map[int64]int{}
	var ids []int64
	err := ExportQSOsSnapshot(db, "", false, func(q qso.QSO) error {
		if !inserted {
			inserted = true
			// Concurrent insert from a separate connection while the
			// snapshot stream is mid-flight (e.g. WSJT-X logging).
			if _, err := InsertQSO(db, &qso.QSO{
				Call: "NEW", QSODate: "20240509", TimeOn: "130000",
				Band: "20m", Mode: "SSB",
			}); err != nil {
				return fmt.Errorf("concurrent insert: %w", err)
			}
		}
		seen[q.ID]++
		ids = append(ids, q.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("ExportQSOsSnapshot: %v", err)
	}

	if len(ids) != 5 {
		t.Fatalf("exported %d rows, want 5 (%v)", len(ids), ids)
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("id %d exported %d times, want exactly once", id, n)
		}
	}
	// The snapshot must not contain the concurrently inserted row.
	var newID int64
	if err := db.QueryRow(`SELECT id FROM qsos WHERE call='NEW'`).Scan(&newID); err != nil {
		t.Fatalf("find inserted row: %v", err)
	}
	if seen[newID] != 0 {
		t.Errorf("concurrently inserted row leaked into the snapshot")
	}
	// Order must match the descending export convention.
	if ids[0] != 5 || ids[4] != 1 {
		t.Errorf("export order = %v, want [5 4 3 2 1]", ids)
	}
}
