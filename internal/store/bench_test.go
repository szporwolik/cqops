package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// benchDB builds a logbook of n QSOs spread across bands, modes, grids and
// DXCC entities so the aggregate queries have realistic cardinality.
func benchDB(b *testing.B, n int) *sql.DB {
	b.Helper()
	path := filepath.Join(b.TempDir(), "bench.db")
	db, err := InitDB(path)
	if err != nil {
		b.Fatalf("InitDB: %v", err)
	}
	b.Cleanup(func() { db.Close() })

	bands := []string{"160m", "80m", "40m", "20m", "15m", "10m", "6m"}
	modes := []string{"SSB", "CW", "FT8", "FT4", "RTTY"}
	grids := []string{"JO90", "KO00", "JN18", "IO91", "FN31", "DM03"}
	countries := []string{"Poland", "Germany", "United States", "Japan", "Brazil"}
	dxccs := []string{"269", "230", "291", "339", "108"}

	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin: %v", err)
	}
	for i := 0; i < n; i++ {
		call := fmt.Sprintf("SP%dABC", i%10)
		q := &qso.QSO{
			Call:       call,
			QSODate:    fmt.Sprintf("2024%02d%02d", (i%12)+1, (i%28)+1),
			TimeOn:     fmt.Sprintf("%02d%02d00", i%24, i%60),
			Band:       bands[i%len(bands)],
			Mode:       modes[i%len(modes)],
			GridSquare: grids[i%len(grids)] + "aa",
			Country:    countries[i%len(countries)],
			DXCC:       dxccs[i%len(dxccs)],
		}
		if _, err := insertQSOTx(tx, q, qso.DeriveBaseCall(call)); err != nil {
			tx.Rollback()
			b.Fatalf("seed insert: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}
	// Benchmarks seed through the raw transaction helper, so rebuild the
	// worked index once — the measured queries read from it.
	if err := RebuildWorkedIndex(db); err != nil {
		b.Fatalf("rebuild worked index: %v", err)
	}
	return db
}

// insertQSOTx mirrors InsertQSO but reuses the caller's transaction so
// seeding a large benchmark logbook does not pay per-row commit costs.
func insertQSOTx(tx *sql.Tx, q *qso.QSO, baseCall string) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO qsos (call, base_call, qso_date, time_on, band, mode, gridsquare, country, dxcc, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		q.Call, baseCall, q.QSODate, q.TimeOn, q.Band, q.Mode, q.GridSquare, q.Country, q.DXCC)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetWorkedSummary runs on the partner-view render path. It issues six
// queries per scope (call, grid, DXCC), so its cost scales with logbook size.
func benchmarkWorkedSummary(b *testing.B, rows int) {
	db := benchDB(b, rows)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := GetWorkedSummary(db, "SP1ABC", "JO90", "269", "Poland"); err != nil {
			b.Fatalf("GetWorkedSummary: %v", err)
		}
	}
}

func BenchmarkGetWorkedSummary_10k(b *testing.B) { benchmarkWorkedSummary(b, 10_000) }
func BenchmarkGetWorkedSummary_50k(b *testing.B) { benchmarkWorkedSummary(b, 50_000) }

// GetLogbookStats runs on the QSO-form path for every new callsign.
func BenchmarkGetLogbookStats_50k(b *testing.B) {
	db := benchDB(b, 50_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := GetLogbookStats(db, "SP1ABC", "20m", "SSB"); err != nil {
			b.Fatalf("GetLogbookStats: %v", err)
		}
	}
}
