package tui

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/store"
)

// benchContestDB builds a logbook whose contest holds n contacts, one every 30
// seconds, so the window counters and the peak-rate scan have realistic
// overlap. Rows go in through raw SQL: the benchmark exercises the reading
// side, and the store's seeding helpers would spend most of the time on the
// worked index.
func benchContestDB(b *testing.B, n int) *sql.DB {
	b.Helper()
	path := filepath.Join(b.TempDir(), "contest.db")
	db, err := store.InitDB(path)
	if err != nil {
		b.Fatalf("InitDB: %v", err)
	}
	b.Cleanup(func() { db.Close() })

	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin: %v", err)
	}
	start := time.Now().UTC().Add(-time.Duration(n*30) * time.Second)
	for i := 0; i < n; i++ {
		at := start.Add(time.Duration(i*30) * time.Second)
		if _, err := tx.Exec(
			`INSERT INTO qsos (call, base_call, qso_date, time_on, band, mode, contest_id, created_at, updated_at)
			 VALUES (?, ?, ?, ?, '20m', 'SSB', 'BENCH', ?, ?)`,
			fmt.Sprintf("SP%dABC", i%50), fmt.Sprintf("SP%dABC", i%50),
			at.Format("20060102"), at.Format("150405"),
			at.Format(time.RFC3339), at.Format(time.RFC3339),
		); err != nil {
			tx.Rollback()
			b.Fatalf("seed: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}
	return db
}

func benchContestStats(b *testing.B, n int, cold bool) {
	db := benchContestDB(b, n)
	cs := &contestStats{}
	if !cold {
		cs.computeIfStale(db, "BENCH", "Benchmark", 1) // warm the history cache
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cs.computedAt = time.Time{} // force the throttle open
		if cold {
			cs.loaded = false // force a full read of the contest history
		}
		cs.computeIfStale(db, "BENCH", "Benchmark", 1)
	}
}

// BenchmarkContestStatsCold measures a full refresh — the history is re-read
// and every derived value recomputed. That is what a QSO change costs.
func BenchmarkContestStatsCold_10k(b *testing.B) { benchContestStats(b, 10_000, true) }
func BenchmarkContestStatsCold_50k(b *testing.B) { benchContestStats(b, 50_000, true) }

// BenchmarkContestStatsTick measures an idle refresh, which is what the panel
// does every five seconds: the cached timestamps are re-scanned for the sliding
// windows and the minute chart, and nothing touches the database.
func BenchmarkContestStatsTick_10k(b *testing.B) { benchContestStats(b, 10_000, false) }
func BenchmarkContestStatsTick_50k(b *testing.B) { benchContestStats(b, 50_000, false) }
