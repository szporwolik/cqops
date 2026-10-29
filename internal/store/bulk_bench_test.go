package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// BenchmarkInsertQSOTxBulk measures the per-contact cost of an ADIF-style bulk
// import: one transaction, InsertQSOTx per row, with the worked index and the
// coverage counters maintained inside it.
func BenchmarkInsertQSOTxBulk_2k(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer() // database creation is not the subject of this benchmark
		path := filepath.Join(b.TempDir(), fmt.Sprintf("bulk%d.db", i))
		db, err := InitDB(path)
		if err != nil {
			b.Fatalf("InitDB: %v", err)
		}
		b.StartTimer()
		benchBulkInsert(b, db, 2000)
		b.StopTimer()
		db.Close()
		b.StartTimer()
	}
}

func benchBulkInsert(b *testing.B, db *sql.DB, n int) {
	b.Helper()
	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin: %v", err)
	}
	for i := 0; i < n; i++ {
		q := &qso.QSO{
			Call: fmt.Sprintf("SP%dABC", i%50), QSODate: "20260501", TimeOn: fmt.Sprintf("%02d%02d00", i%24, i%60),
			Band: "20m", Mode: "FT8", DXCC: fmt.Sprintf("%d", 260+i%20),
			GridSquare: "JO90", Source: "adif", ContestID: "bench",
		}
		if _, err := InsertQSOTx(tx, q); err != nil {
			tx.Rollback()
			b.Fatalf("InsertQSOTx: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}
}
