package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// backfillLog builds a logbook of n contacts that all lack a DXCC entity — the
// state the backfill exists for (legacy imports, ADIF loads, downloads from
// before Big CTY was available).
func backfillLog(t *testing.T, n int) *sql.DB {
	t.Helper()
	return backfillLogTB(t, n)
}

func backfillLogTB(tb testing.TB, n int) *sql.DB {
	tb.Helper()
	db, err := InitDB(filepath.Join(tb.TempDir(), "backfill.db"))
	if err != nil {
		tb.Fatalf("InitDB: %v", err)
	}
	tb.Cleanup(func() { db.Close() })
	tx, err := db.Begin()
	if err != nil {
		tb.Fatalf("begin: %v", err)
	}
	bands := []string{"160m", "80m", "40m", "20m", "15m", "10m", "6m"}
	modes := []string{"SSB", "CW", "FT8", "FT4", "RTTY"}
	grids := []string{"JO90", "KO00", "JN18", "IO91", "FN31", "DM03"}
	for i := 0; i < n; i++ {
		call := fmt.Sprintf("SP%dABC", i%10)
		q := &qso.QSO{
			Call:       call,
			QSODate:    fmt.Sprintf("2024%02d%02d", (i%12)+1, (i%28)+1),
			TimeOn:     fmt.Sprintf("%02d%02d00", i%24, i%60),
			Band:       bands[i%len(bands)],
			Mode:       modes[i%len(modes)],
			GridSquare: grids[i%len(grids)] + "aa",
			RSTSent:    "59",
			RSTRcvd:    "59",
		}
		if _, err := insertQSO(tx, q); err != nil {
			tx.Rollback()
			tb.Fatalf("seed insert: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		tb.Fatalf("commit: %v", err)
	}
	// The raw insert above skips the index maintenance, so build both the
	// aggregates and the coverage counters from the data once.
	if err := RebuildWorkedIndex(db); err != nil {
		tb.Fatalf("rebuild worked index: %v", err)
	}
	return db
}

// missingDXCCIDs picks up to n contacts without an entity.
func missingDXCCIDs(t testing.TB, db *sql.DB, n int) map[int64]string {
	t.Helper()
	rows, err := db.Query(`SELECT id FROM qsos WHERE COALESCE(dxcc,'') = '' ORDER BY id LIMIT ?`, n)
	if err != nil {
		t.Fatalf("select ids: %v", err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		out[id] = "269"
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ids rows: %v", err)
	}
	return out
}

// TestBackfillDXCCLeavesTheOtherAggregatesAlone is the structural guard for the
// cost of a backfill pass: the pass writes only the dxcc column, so only the
// DXCC aggregates can change. Removing and re-adding the whole contribution —
// what an ordinary edit does — also decrements the base-call and four-character
// grid rows, and the removal half re-derives their first/last timestamps from
// the entire log; the grid one cannot use an index at all
// (UPPER(SUBSTR(gridsquare,1,4)) is a function on the column). That walk cost
// 395 ms for 50 rows in a 10k log and ~6 s in a 100k one, inside the write
// transaction a save then had to wait for.
func TestBackfillDXCCLeavesTheOtherAggregatesAlone(t *testing.T) {
	db := backfillLog(t, 200)
	indexWriteLog(t, db)
	ids := missingDXCCIDs(t, db, 50)
	clearIndexWrites(t, db)

	n, err := BackfillDXCC(db, ids)
	if err != nil {
		t.Fatalf("BackfillDXCC: %v", err)
	}
	if n != len(ids) {
		t.Fatalf("backfilled %d rows, want %d", n, len(ids))
	}

	var dxccWrites int
	for _, w := range indexWrites(t, db) {
		switch {
		case strings.HasPrefix(w, "worked_call"), strings.HasPrefix(w, "worked_grid"):
			t.Errorf("the backfill rewrote %q — it must only move the DXCC aggregates", w)
		case strings.HasPrefix(w, "worked_dxcc"):
			dxccWrites++
		}
	}
	if dxccWrites == 0 {
		t.Error("no worked_dxcc rows were written, so the backfill did not reach the index")
	}
}

// TestBackfillDXCCMatchesARebuild pins the result, not just the route: whatever
// the backfill writes has to be exactly what a full rebuild of the same data
// would produce — every aggregate row, its counts, first/last stamps and source
// mask, plus the coverage counters.
func TestBackfillDXCCMatchesARebuild(t *testing.T) {
	db := backfillLog(t, 120)

	// A contact that already had an entity must be left alone, and the mix of
	// bands/modes/grids means several aggregate rows get a new member.
	if _, err := InsertQSO(db, &qso.QSO{
		Call: "SP9MOA", Band: "20m", Mode: "SSB", GridSquare: "JO90AA",
		DXCC: "269", QSODate: "20260501", TimeOn: "120000", RSTSent: "59", RSTRcvd: "59",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	ids := missingDXCCIDs(t, db, 10_000) // every row, so a rebuild sees the same set
	// Two entities, so the aggregate gets rows that did not exist yet.
	i := 0
	for id := range ids {
		if i%2 == 0 {
			ids[id] = "269"
		} else {
			ids[id] = "230"
		}
		i++
	}
	if _, err := BackfillDXCC(db, ids); err != nil {
		t.Fatalf("BackfillDXCC: %v", err)
	}
	backfilled := snapshotAggregates(t, db)

	if err := RebuildWorkedIndex(db); err != nil {
		t.Fatalf("RebuildWorkedIndex: %v", err)
	}
	rebuilt := snapshotAggregates(t, db)

	// The rebuild stamps updated_utc with its own clock; everything else must
	// match exactly.
	if len(backfilled) != len(rebuilt) {
		t.Fatalf("backfill produced %d aggregate rows, a rebuild produces %d", len(backfilled), len(rebuilt))
	}
	for key, want := range rebuilt {
		got, ok := backfilled[key]
		if !ok {
			t.Errorf("row %q exists after a rebuild but not after the backfill", key)
			continue
		}
		if got != want {
			t.Errorf("row %q\n backfill: %s\n  rebuild: %s", key, got, want)
		}
	}
}

// snapshotAggregates flattens every worked-index row (without updated_utc,
// which is a write timestamp) into a comparable map.
func snapshotAggregates(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, q := range []struct{ table, cols, key string }{
		{"worked_call", "base_call", "base_call"},
		{"worked_grid", "grid4", "grid4"},
		{"worked_dxcc", "dxcc || '/' || band || '/' || mode", "dxcc,band,mode"},
	} {
		rows, err := db.Query(fmt.Sprintf(
			`SELECT %s, qso_count, confirmed_count, first_utc, last_utc%s FROM %s`,
			q.cols, map[bool]string{true: ", source_mask", false: ""}[q.table == "worked_dxcc"], q.table))
		if err != nil {
			t.Fatalf("read %s: %v", q.table, err)
		}
		for rows.Next() {
			var key, count, confirmed, first, last string
			var mask string
			dest := []any{&key, &count, &confirmed, &first, &last}
			if q.table == "worked_dxcc" {
				dest = append(dest, &mask)
			}
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				t.Fatalf("scan %s: %v", q.table, err)
			}
			out[q.table+" "+key] = fmt.Sprintf("count=%s confirmed=%s first=%s last=%s mask=%s",
				count, confirmed, first, last, mask)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatalf("%s rows: %v", q.table, err)
		}
		rows.Close()
	}
	cov, err := Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	out["coverage"] = fmt.Sprintf("%+v", cov)
	return out
}

// TestBackfillDXCCCoversTheMissingCounter: resolving an entity has to move the
// "rows without DXCC" counter with it, or the panel keeps reporting an
// incomplete log (and every new callsign stays UNKNOWN instead of NEW).
func TestBackfillDXCCCoversTheMissingCounter(t *testing.T) {
	db := backfillLog(t, 40)
	before, err := Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if before.UnresolvedDXCC != 40 {
		t.Fatalf("UnresolvedDXCC = %d before the backfill, want 40", before.UnresolvedDXCC)
	}
	ids := missingDXCCIDs(t, db, 15)
	if _, err := BackfillDXCC(db, ids); err != nil {
		t.Fatalf("BackfillDXCC: %v", err)
	}
	after, err := Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if want := before.UnresolvedDXCC - len(ids); after.UnresolvedDXCC != want {
		t.Errorf("UnresolvedDXCC = %d after backfilling %d rows, want %d",
			after.UnresolvedDXCC, len(ids), want)
	}
	if after.QSOCount != before.QSOCount || after.MissingBand != before.MissingBand || after.MissingMode != before.MissingMode {
		t.Errorf("the backfill moved a counter it does not change: before %+v, after %+v", before, after)
	}
}

// TestBackfillDXCCLetsAConcurrentSaveThrough: the pass holds the write lock, so
// a save that lands meanwhile must still complete and must not lose the
// backfill's work. With the whole contribution being removed and re-added this
// save waited 433 ms in a 10k log and ~6 s in a 100k one — longer than the 5 s
// busy timeout, i.e. a real "database is locked" for the operator.
func TestBackfillDXCCLetsAConcurrentSaveThrough(t *testing.T) {
	db := backfillLog(t, 2_000)
	ids := missingDXCCIDs(t, db, 50)

	done := make(chan error, 1)
	go func() {
		_, err := BackfillDXCC(db, ids)
		done <- err
	}()

	if _, err := InsertQSO(db, &qso.QSO{
		Call: "DL1AA", Band: "20m", Mode: "CW", GridSquare: "JO90AA",
		DXCC: "230", QSODate: "20261027", TimeOn: "120000", RSTSent: "59", RSTRcvd: "59",
	}); err != nil {
		t.Fatalf("concurrent save failed: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("backfill failed: %v", err)
	}

	// Both sides landed in the aggregates: the save's entity counts, and every
	// backfilled row is no longer missing.
	saved, err := WorkedDXCC(db, "230", "20m", "CW")
	if err != nil {
		t.Fatalf("WorkedDXCC: %v", err)
	}
	if saved.Count != 1 {
		t.Errorf("the concurrent save's DXCC aggregate has count %d, want 1", saved.Count)
	}
	var missing int
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos WHERE COALESCE(dxcc,'') = ''`).Scan(&missing); err != nil {
		t.Fatalf("count missing: %v", err)
	}
	if want := 2_000 - len(ids); missing != want {
		t.Errorf("%d rows still without an entity, want %d", missing, want)
	}
	cov, err := Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.UnresolvedDXCC != missing {
		t.Errorf("UnresolvedDXCC = %d, want %d (the counter drifted from the log)", cov.UnresolvedDXCC, missing)
	}
}

// benchmarkBackfill measures one 50-row pass on a log of the given size. The
// batch is un-resolved again outside the timer so every iteration does the same
// work.
func benchmarkBackfill(b *testing.B, rows int) {
	db := backfillLogTB(b, rows)
	ids := missingDXCCIDs(b, db, 50)
	unset := func() {
		for id := range ids {
			if _, err := db.Exec(`UPDATE qsos SET dxcc = '', dxcc_source = '' WHERE id = ?`, id); err != nil {
				b.Fatalf("reset row: %v", err)
			}
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		unset()
		b.StartTimer()
		if _, err := BackfillDXCC(db, ids); err != nil {
			b.Fatalf("BackfillDXCC: %v", err)
		}
	}
}

func BenchmarkBackfillDXCCBatch_10k(b *testing.B)  { benchmarkBackfill(b, 10_000) }
func BenchmarkBackfillDXCCBatch_100k(b *testing.B) { benchmarkBackfill(b, 100_000) }
