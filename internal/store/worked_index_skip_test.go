package store

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// indexWriteLog installs AFTER triggers on the three aggregate tables that
// record every INSERT/UPDATE/DELETE into a side table. That is what makes "the
// index was left alone" observable without timing anything: a rowid comparison
// does not work, because SQLite hands a deleted highest rowid straight back to
// the next insert.
func indexWriteLog(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE index_writes (kind TEXT NOT NULL)`); err != nil {
		t.Fatalf("create index_writes: %v", err)
	}
	for _, table := range []string{"worked_call", "worked_grid", "worked_dxcc"} {
		for _, op := range []string{"INSERT", "UPDATE", "DELETE"} {
			stmt := fmt.Sprintf(
				`CREATE TRIGGER %s_%s AFTER %s ON %s BEGIN INSERT INTO index_writes VALUES ('%s %s'); END`,
				table, strings.ToLower(op), op, table, table, strings.ToLower(op))
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("create trigger: %v", err)
			}
		}
	}
}

// indexWrites returns the recorded aggregate writes (empty = untouched).
func indexWrites(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT kind FROM index_writes ORDER BY rowid`)
	if err != nil {
		t.Fatalf("read index_writes: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			t.Fatalf("scan index_writes: %v", err)
		}
		out = append(out, kind)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("index_writes rows: %v", err)
	}
	return out
}

// clearIndexWrites forgets the setup phase of a test.
func clearIndexWrites(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM index_writes`); err != nil {
		t.Fatalf("clear index_writes: %v", err)
	}
}

// TestEditsWithoutIndexEffectSkipTheAggregates covers the cheap half of the
// rule: an edit that cannot change any aggregate — a comment, a name, a QTH,
// the notes, the power — must not unindex and reindex the contact. The removal
// half re-derives first/last/source_mask for every aggregate the contact
// belongs to, which made a comment-only save the most expensive write in a
// large log (hundreds of ms at 100k QSOs) and could stall the UI when the
// enrichment loop triggered it.
//
// Each contact is the only member of its aggregates, so the old -1/+1 pair
// removed and re-inserted every row, plus the edge recomputation on the rows
// that survived.
func TestEditsWithoutIndexEffectSkipTheAggregates(t *testing.T) {
	db := newTempDB(t)
	indexWriteLog(t, db)
	q := &qso.QSO{
		Call: "SP9MOA", Band: "20m", Mode: "FT8", GridSquare: "JO90AA", DXCC: "269",
		QSODate: "20260501", TimeOn: "123000", RSTSent: "59", RSTRcvd: "59",
		Comment: "before",
	}
	mustInsertQSO(t, db, q)
	clearIndexWrites(t, db)

	q.Comment = "after"
	q.Name = "Szymon"
	q.QTH = "Krakow"
	q.Notes = "rig notes"
	q.TXPower = "100"
	q.ITUZone = "28"
	if err := UpdateQSO(db, q); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if writes := indexWrites(t, db); len(writes) != 0 {
		t.Errorf("a comment/name/QTH edit rewrote the index: %v", writes)
	}

	// The counts have to be exactly what they were.
	var count, confirmed int
	if err := db.QueryRow(
		`SELECT qso_count, confirmed_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`,
	).Scan(&count, &confirmed); err != nil {
		t.Fatalf("read aggregate: %v", err)
	}
	if count != 1 || confirmed != 0 {
		t.Errorf("dxcc aggregate = %d/%d, want 1/0", count, confirmed)
	}

	// Enrichment that finds nothing new must be equally quiet: the row already
	// has name/QTH/grid/dxcc, so every CASE guard leaves it untouched.
	sync, err := EnrichQSO(db, q.ID, EnrichmentData{
		Name: "Szymon", QTH: "Krakow", GridSquare: "JO90AA", DXCC: "269",
	})
	if err != nil {
		t.Fatalf("EnrichQSO: %v", err)
	}
	if sync.Changed {
		t.Errorf("a no-op enrichment reported a change: %+v", sync)
	}
	if writes := indexWrites(t, db); len(writes) != 0 {
		t.Errorf("a no-op enrichment rewrote the index: %v", writes)
	}

	// The rule must not silence real work: a comment edit that also carries a
	// new grid still has to move the grid aggregate.
	q.GridSquare = "KN09AA"
	if err := UpdateQSO(db, q); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if writes := indexWrites(t, db); len(writes) == 0 {
		t.Error("a grid change did not touch the index")
	}
}

// TestEditsWithIndexEffectUpdateTheAggregates is the other half: every field
// the index aggregates over still moves the counts, the first/last edges and
// the confirmation counter when it really changes.
func TestEditsWithIndexEffectUpdateTheAggregates(t *testing.T) {
	type check struct {
		query string
		args  []any
		want  int // qso_count or confirmed_count the query returns; -1 = row must be gone
	}
	cases := []struct {
		name   string
		edit   func(q *qso.QSO)
		checks []check
	}{
		{
			name: "dxcc",
			edit: func(q *qso.QSO) { q.DXCC = "291" },
			checks: []check{
				{`SELECT qso_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`, nil, 1},
				{`SELECT qso_count FROM worked_dxcc WHERE dxcc='291' AND band='' AND mode=''`, nil, 1},
			},
		},
		{
			name: "time",
			edit: func(q *qso.QSO) { q.QSODate = "20260101"; q.TimeOn = "000100" },
			checks: []check{
				{`SELECT first_utc = '20260101000100' FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`, nil, 1},
			},
		},
		{
			name: "grid",
			edit: func(q *qso.QSO) { q.GridSquare = "KN09" },
			checks: []check{
				{`SELECT qso_count FROM worked_grid WHERE grid4='JO90'`, nil, 1},
				{`SELECT qso_count FROM worked_grid WHERE grid4='KN09'`, nil, 1},
			},
		},
		{
			name: "confirmation",
			edit: func(q *qso.QSO) { q.QSLRcvd = "Y" },
			checks: []check{
				{`SELECT confirmed_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`, nil, 1},
				{`SELECT confirmed_count FROM worked_call WHERE base_call='SP9MOA'`, nil, 1},
			},
		},
		{
			name: "band and mode",
			edit: func(q *qso.QSO) { q.Band = "40m"; q.Mode = "CW" },
			checks: []check{
				{`SELECT qso_count FROM worked_dxcc WHERE dxcc='269' AND band='20m' AND mode='FT8'`, nil, 1},
				{`SELECT qso_count FROM worked_dxcc WHERE dxcc='269' AND band='20m' AND mode=''`, nil, 1},
				{`SELECT qso_count FROM worked_dxcc WHERE dxcc='269' AND band='40m' AND mode='CW'`, nil, 1},
			},
		},
		{
			name: "call",
			edit: func(q *qso.QSO) { q.Call = "DL1AA" },
			checks: []check{
				{`SELECT qso_count FROM worked_call WHERE base_call='SP9MOA'`, nil, 1},
				{`SELECT qso_count FROM worked_call WHERE base_call='DL1AA'`, nil, 1},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTempDB(t)
			q := &qso.QSO{
				Call: "SP9MOA", Band: "20m", Mode: "FT8", GridSquare: "JO90AA", DXCC: "269",
				QSODate: "20260501", TimeOn: "123000", RSTSent: "59", RSTRcvd: "59",
			}
			mustInsertQSO(t, db, q)
			// A second contact in the same aggregates, so edge recomputation
			// and the counters have something to work with.
			mustInsertQSO(t, db, &qso.QSO{
				Call: "SP9MOA", Band: "20m", Mode: "FT8", GridSquare: "JO90AA", DXCC: "269",
				QSODate: "20260502", TimeOn: "123000", RSTSent: "59", RSTRcvd: "59",
			})

			tc.edit(q)
			if err := UpdateQSO(db, q); err != nil {
				t.Fatalf("UpdateQSO: %v", err)
			}
			for _, c := range tc.checks {
				var got int
				err := db.QueryRow(c.query, c.args...).Scan(&got)
				if c.want < 0 {
					if err == nil {
						t.Errorf("%s: expected no row, got %d", c.query, got)
					}
					continue
				}
				if err != nil {
					t.Errorf("%s: %v", c.query, err)
					continue
				}
				if got != c.want {
					t.Errorf("%s = %d, want %d", c.query, got, c.want)
				}
			}
		})
	}
}

// TestEditThatMovesEdgesKeepsCountsStable guards the arithmetic the skip must
// not disturb: moving one contact out of a two-contact aggregate leaves the
// count at one and re-derives both edges from the remaining row.
func TestEditThatMovesEdgesKeepsCountsStable(t *testing.T) {
	db := newTempDB(t)
	first := &qso.QSO{
		Call: "SP9MOA", Band: "20m", Mode: "FT8", DXCC: "269",
		QSODate: "20260101", TimeOn: "100000", RSTSent: "59", RSTRcvd: "59",
	}
	second := &qso.QSO{
		Call: "SP9MOA", Band: "20m", Mode: "FT8", DXCC: "269",
		QSODate: "20260601", TimeOn: "100000", RSTSent: "59", RSTRcvd: "59",
	}
	mustInsertQSO(t, db, first)
	mustInsertQSO(t, db, second)

	// Move the late contact to an earlier date that is still after the first.
	second.QSODate = "20260301"
	if err := UpdateQSO(db, second); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}

	var count int
	var firstUTC, lastUTC string
	if err := db.QueryRow(
		`SELECT qso_count, first_utc, last_utc FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`,
	).Scan(&count, &firstUTC, &lastUTC); err != nil {
		t.Fatalf("read aggregate: %v", err)
	}
	if count != 2 || firstUTC != "20260101100000" || lastUTC != "20260301100000" {
		t.Errorf("aggregate = count %d, %s..%s; want 2, 20260101100000..20260301100000", count, firstUTC, lastUTC)
	}
}

// TestContributionCoversEveryIndexedField pins the mapping contributionOf must
// mirror: if a field that indexQSO aggregates over is missing here, an edit of
// that field would be skipped and the index would silently drift.
func TestContributionCoversEveryIndexedField(t *testing.T) {
	base := &qso.QSO{
		Call: "SP9MOA", Band: "20m", Mode: "MFSK", Submode: "FT8", GridSquare: "JO90AA",
		DXCC: "269", QSODate: "20260501", TimeOn: "123000", Source: "manual",
	}
	changes := []struct {
		name string
		edit func(q *qso.QSO)
	}{
		{"call", func(q *qso.QSO) { q.Call = "DL1AA" }},
		{"band", func(q *qso.QSO) { q.Band = "40m" }},
		{"mode", func(q *qso.QSO) { q.Mode = "SSB"; q.Submode = "" }},
		{"submode", func(q *qso.QSO) { q.Submode = "FT4" }},
		{"grid", func(q *qso.QSO) { q.GridSquare = "KN09AA" }},
		{"grid below four characters", func(q *qso.QSO) { q.GridSquare = "JO" }},
		{"dxcc", func(q *qso.QSO) { q.DXCC = "291" }},
		{"date", func(q *qso.QSO) { q.QSODate = "20260502" }},
		{"time", func(q *qso.QSO) { q.TimeOn = "123001" }},
		{"confirmation", func(q *qso.QSO) { q.LOTWQSLRcvd = "Y" }},
		{"source", func(q *qso.QSO) { q.Source = "wsjtx" }},
	}
	for _, c := range changes {
		t.Run(c.name, func(t *testing.T) {
			edited := *base
			c.edit(&edited)
			if contributionOf(base) == contributionOf(&edited) {
				t.Errorf("%s change does not alter the index contribution — the edit would skip the index update", c.name)
			}
		})
	}

	// Fields the index does not aggregate over must not force a rewrite.
	ignored := []struct {
		name string
		edit func(q *qso.QSO)
	}{
		{"comment", func(q *qso.QSO) { q.Comment = "hello" }},
		{"name", func(q *qso.QSO) { q.Name = "Szymon" }},
		{"qth", func(q *qso.QSO) { q.QTH = "Krakow" }},
		{"notes", func(q *qso.QSO) { q.Notes = "notes" }},
		{"power", func(q *qso.QSO) { q.TXPower = "100" }},
		{"rst", func(q *qso.QSO) { q.RSTSent = "599" }},
		{"grid suffix", func(q *qso.QSO) { q.GridSquare = "JO90XX" }},
		{"zone", func(q *qso.QSO) { q.CQZone = "15" }},
	}
	for _, c := range ignored {
		t.Run("ignored_"+c.name, func(t *testing.T) {
			edited := *base
			c.edit(&edited)
			if contributionOf(base) != contributionOf(&edited) {
				t.Errorf("%s change alters the index contribution — a cosmetic edit would rewrite the aggregates", c.name)
			}
		})
	}
}
