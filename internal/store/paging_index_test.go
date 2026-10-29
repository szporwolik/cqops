package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// The logbook editor pages through qsos ordered by qso_date DESC, time_on
// DESC, id DESC. Contacts that share a date and a minute are the normal case
// (an imported batch, a contest run, a WSJT-X burst), so the id tiebreaker
// decides the order of most rows in a real logbook — and an index that stops
// at (qso_date, time_on) cannot serve it: SQLite appends the rowid ASC to
// every index key, so the plan adds "USE TEMP B-TREE FOR LAST TERM OF ORDER
// BY" and sorts the WHOLE filtered set before the OFFSET is skipped.

// pagingTestDB opens an empty migrated logbook.
func pagingTestDB(tb testing.TB) *sql.DB {
	tb.Helper()
	db, err := InitDB(filepath.Join(tb.TempDir(), "paging.db"))
	if err != nil {
		tb.Fatalf("InitDB: %v", err)
	}
	tb.Cleanup(func() { db.Close() })
	return db
}

// seedPagingContacts inserts n contacts through the production write path (so
// the worked index is maintained too) with the given date and time, returning
// the assigned ids in insertion order.
func seedPagingContacts(tb testing.TB, db *sql.DB, date, timeOn string, n int) []int64 {
	tb.Helper()
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		q := &qso.QSO{
			Call:    fmt.Sprintf("SP%dABC", i%7),
			QSODate: date,
			TimeOn:  timeOn,
			Band:    "20m",
			Mode:    "SSB",
		}
		id, err := InsertQSO(db, q)
		if err != nil {
			tb.Fatalf("InsertQSO: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

// queryPlan lives in worked_index_guard_test.go: it returns the EXPLAIN QUERY
// PLAN detail lines joined into one string.

// TestQSOsPageOrderNeedsNoSortStep is the regression gate for the paging
// index: the page query must walk an index in exactly the requested order.
// A plan that contains a temp B-tree means every page turn materialises and
// sorts the whole logbook first, which is what made the last page of a
// 100k-contact log take ~470 ms.
func TestQSOsPageOrderNeedsNoSortStep(t *testing.T) {
	db := pagingTestDB(t)
	seedPagingContacts(t, db, "20240101", "120000", 40)
	seedPagingContacts(t, db, "20240102", "120000", 20)

	query := `SELECT ` + qsoSelectCols + ` FROM qsos ORDER BY ` + qsoNewestFirst + ` LIMIT ? OFFSET ?`
	plan := queryPlan(t, db, query, 20, 20)

	if strings.Contains(plan, "TEMP B-TREE") {
		t.Fatalf("page query sorts before paging: %s", plan)
	}
	if !strings.Contains(plan, "idx_qsos_date_time_id") {
		t.Fatalf("page query does not walk the paging index: %s", plan)
	}
}

// TestQSOsPagingKeepsTheOrderAndTheRows walks the logbook one page at a time
// and checks the concatenated ids against the order the query promises. Rows
// that share a date and time are the interesting part: they must come back
// newest id first, every page boundary must fall in the same place, and no
// contact may be repeated or skipped.
func TestQSOsPagingKeepsTheOrderAndTheRows(t *testing.T) {
	db := pagingTestDB(t)
	var all []qso.QSO
	insert := func(date, timeOn string, n int) {
		t.Helper()
		ids := seedPagingContacts(t, db, date, timeOn, n)
		for i, id := range ids {
			all = append(all, qso.QSO{
				ID:      id,
				Call:    fmt.Sprintf("SP%dABC", i%7),
				QSODate: date,
				TimeOn:  timeOn,
			})
		}
	}
	insert("20240103", "120000", 37) // one minute, many contacts
	insert("20240103", "090000", 5)
	insert("20240102", "235900", 11)
	insert("20240101", "000000", 3)
	insert("20240103", "120000", 8) // same minute again, newer ids

	sort.Slice(all, func(a, b int) bool {
		qa, qb := all[a], all[b]
		if qa.QSODate != qb.QSODate {
			return qa.QSODate > qb.QSODate
		}
		if qa.TimeOn != qb.TimeOn {
			return qa.TimeOn > qb.TimeOn
		}
		return qa.ID > qb.ID
	})
	want := make([]int64, len(all))
	for i, q := range all {
		want[i] = q.ID
	}

	const pageSize = 20
	var got []int64
	for offset := 0; ; offset += pageSize {
		page, total, err := ListQSOsPageWithCount(db, pageSize, offset, "")
		if err != nil {
			t.Fatalf("page at offset %d: %v", offset, err)
		}
		if total != len(all) {
			t.Fatalf("total = %d, want %d", total, len(all))
		}
		if len(page) == 0 {
			break
		}
		for _, q := range page {
			got = append(got, q.ID)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("paged %d contacts, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("position %d: id %d, want %d\npaged: %v\norder: %v", i, got[i], want[i], got, want)
		}
	}
}

// TestMigrateCreatesThePagingIndex checks the two halves of the migration: a
// fresh logbook gets the three-column index, and the two-column index it
// replaces is not created.
func TestMigrateCreatesThePagingIndex(t *testing.T) {
	db := pagingTestDB(t)
	if !indexExists(t, db, "idx_qsos_date_time_id") {
		t.Error("idx_qsos_date_time_id missing on a fresh logbook")
	}
	if indexExists(t, db, "idx_qsos_date_time") {
		t.Error("idx_qsos_date_time still created; the three-column index covers it")
	}
}

// TestMigrateUpgradesAReleasedPagingIndex simulates a logbook written by the
// released schema (two-column ordering index, user_version at the current
// version). It must swap the index on the next open — a database already at
// the current schema skips the migration list, so the replacement has to live
// on the unconditional path — and stay idempotent.
func TestMigrateUpgradesAReleasedPagingIndex(t *testing.T) {
	db := pagingTestDB(t)
	// Rewind to what the released build left behind.
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_qsos_date_time_id`,
		`CREATE INDEX IF NOT EXISTS idx_qsos_date_time ON qsos(qso_date DESC, time_on DESC)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if indexExists(t, db, "idx_qsos_date_time_id") {
		t.Fatal("precondition: the three-column index should be gone")
	}

	for i := 0; i < 2; i++ { // twice: the swap must be idempotent
		if err := Migrate(db); err != nil {
			t.Fatalf("Migrate #%d: %v", i+1, err)
		}
		if !indexExists(t, db, "idx_qsos_date_time_id") {
			t.Fatalf("Migrate #%d did not create idx_qsos_date_time_id", i+1)
		}
		if indexExists(t, db, "idx_qsos_date_time") {
			t.Fatalf("Migrate #%d left the superseded idx_qsos_date_time", i+1)
		}
	}

	seedPagingContacts(t, db, "20240101", "120000", 30)
	plan := queryPlan(t, db,
		`SELECT `+qsoSelectCols+` FROM qsos ORDER BY `+qsoNewestFirst+` LIMIT ? OFFSET ?`, 10, 10)
	if strings.Contains(plan, "TEMP B-TREE") {
		t.Fatalf("upgraded logbook still sorts before paging: %s", plan)
	}
}

func indexExists(tb testing.TB, db *sql.DB, name string) bool {
	tb.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&n); err != nil {
		tb.Fatalf("index probe %s: %v", name, err)
	}
	return n > 0
}

// BenchmarkListQSOsPage_SharedMinute_100k reproduces the worst case for the
// paging order: 100k contacts that all share one date and minute, so every
// row is decided by the id tiebreaker. Run with
//
//	go test ./internal/store -run '^$' -bench BenchmarkListQSOsPage -benchtime 20x
func BenchmarkListQSOsPage_SharedMinute_100k(b *testing.B) {
	db := benchPageDB(b, 100_000)
	b.ResetTimer()
	for _, offset := range []int{0, 99_980} {
		b.Run(fmt.Sprintf("offset_%d", offset), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, _, err := ListQSOsPageWithCount(db, 20, offset, ""); err != nil {
					b.Fatalf("page: %v", err)
				}
			}
		})
	}
}

// benchPageDB seeds n contacts sharing one date and time. The bulk seeder
// leaves the nullable columns NULL, which the page query scans as text and
// numbers, so they are defaulted once afterwards.
func benchPageDB(b *testing.B, n int) *sql.DB {
	b.Helper()
	db, err := InitDB(filepath.Join(b.TempDir(), "benchpage.db"))
	if err != nil {
		b.Fatalf("InitDB: %v", err)
	}
	b.Cleanup(func() { db.Close() })

	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin: %v", err)
	}
	for i := 0; i < n; i++ {
		call := fmt.Sprintf("SP%dABC", i%10)
		q := &qso.QSO{Call: call, QSODate: "20240101", TimeOn: "120000", Band: "20m", Mode: "SSB"}
		if _, err := insertQSOTx(tx, q, qso.DeriveBaseCall(call)); err != nil {
			tx.Rollback()
			b.Fatalf("seed insert: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}
	if err := pageRowDefaults(db); err != nil {
		b.Fatalf("default nullable columns: %v", err)
	}
	return db
}

// pageRowDefaults fills the columns ListQSOsPageWithCount scans, which the
// benchmark seeder leaves NULL, in one pass over the table.
func pageRowDefaults(db *sql.DB) error {
	text := []string{
		"time_off", "submode", "rst_sent", "rst_rcvd", "gridsquare", "name", "qth",
		"country", "comment", "notes", "tx_pwr", "sota_ref", "pota_ref", "wwff_ref",
		"iota", "sig", "sig_info", "my_sota_ref", "my_pota_ref", "my_wwff_ref",
		"station_callsign", "operator", "my_gridsquare", "my_rig", "my_antenna",
		"cq_zone", "itu_zone", "dxcc", "my_cq_zone", "my_itu_zone", "my_dxcc",
		"my_sig", "my_sig_info", "contest_id", "exch_sent", "exch_rcvd",
		"stx_string", "srx_string", "contest_adif_id",
	}
	number := []string{"freq", "freq_rx", "distance", "bearing", "wavelog_id", "stx", "srx"}

	sets := make([]string, 0, len(text)+len(number))
	for _, c := range text {
		sets = append(sets, c+" = COALESCE("+c+", '')")
	}
	for _, c := range number {
		sets = append(sets, c+" = COALESCE("+c+", 0)")
	}
	_, err := db.Exec(`UPDATE qsos SET ` + strings.Join(sets, ", "))
	return err
}
