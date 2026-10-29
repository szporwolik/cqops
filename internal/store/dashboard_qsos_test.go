package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Dashboard projection
//
// The dashboard renders fourteen columns, and it reads up to 5000 rows on the
// TUI update path. It used to go through the general-purpose loader, which
// scans the whole record (notes, five references plus their "my" counterparts,
// station fields, Wavelog bookkeeping, timestamps) and cost 56 ms and 25.7 MiB
// of allocations per refresh. The projection is a separate query and scanner;
// these tests are what keep it identical to what the panels used to see.
// =============================================================================

// projectionOf maps a full record the way the dashboard used to read it, so a
// test can compare the two paths field by field.
func projectionOf(q qso.QSO) DashboardQSO {
	return DashboardQSO{
		ID:           q.ID,
		Call:         q.Call,
		QSODate:      q.QSODate,
		TimeOn:       q.TimeOn,
		Band:         q.Band,
		Freq:         q.Freq,
		Mode:         q.Mode,
		Submode:      q.Submode,
		RSTSent:      q.RSTSent,
		RSTRcvd:      q.RSTRcvd,
		GridSquare:   q.GridSquare,
		Country:      q.Country,
		Operator:     q.Operator,
		MyGridSquare: q.MyGridSquare,
	}
}

// seedDashboardContacts inserts a spread of contacts covering what the panels
// render: several dates, ties inside one date (the id tiebreaker), a missing
// frequency, submodes, six-character grids, a filled-in "my grid" and one
// contact whose fields are only filled by enrichment afterwards.
func seedDashboardContacts(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	rows := []*qso.QSO{
		{Call: "SP9MOA", QSODate: "20261025", TimeOn: "101500", Band: "20m", Freq: 14.0745,
			Mode: "MFSK", Submode: "FT8", RSTSent: "-10", RSTRcvd: "-12",
			GridSquare: "JO90WE", MyGridSquare: "KO00AA", Country: "Poland", Operator: "SP9MOA"},
		{Call: "DJ7NT", QSODate: "20261026", TimeOn: "120000", Band: "40m", Freq: 7.047,
			Mode: "MFSK", Submode: "FT4", RSTSent: "-05", RSTRcvd: "-07",
			GridSquare: "JO31", MyGridSquare: "KO00AA", Country: "Fed. Rep. of Germany"},
		// Same date and time as the next one: ordering must fall through to id.
		{Call: "K1ABC", QSODate: "20261027", TimeOn: "090000", Band: "15m", Freq: 21.074,
			Mode: "CW", RSTSent: "599", RSTRcvd: "579", GridSquare: "FN42"},
		{Call: "K2DEF", QSODate: "20261027", TimeOn: "090000", Band: "15m",
			Mode: "SSB", Submode: "USB", RSTSent: "59", RSTRcvd: "57", GridSquare: "FN30"},
		// No frequency at all: the panel shows "0.000 MHz" and no Hz value.
		{Call: "SP9SPM", QSODate: "20261028", TimeOn: "235900", Band: "80m",
			Mode: "SSB", RSTSent: "59", RSTRcvd: "59", GridSquare: "JO90"},
	}
	var first int64
	for i, q := range rows {
		id := mustInsertQSO(t, db, q)
		if i == 0 {
			first = id
		}
	}
	// An enriched contact whose country was filled after the save: the panels
	// repaint on this, and the projection must carry it.
	if err := UpdateQSOEnrichment(db, first, EnrichmentData{Country: "Poland", GridSquare: "JO90WE"}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}
	// K1ABC was logged without a country: a callbook lookup fills it later, and
	// the panels repaint on that even though the row id did not change.
	if err := UpdateQSOEnrichment(db, first+2, EnrichmentData{Country: "United States"}); err != nil {
		t.Fatalf("UpdateQSOEnrichment(K1ABC): %v", err)
	}
	return first
}

// TestListDashboardQSOsFromDateMatchesTheFullLoader is the before/after gate:
// every field the dashboard reads, the ordering, the date filter and the limit
// must be exactly what the general-purpose loader produced.
func TestListDashboardQSOsFromDateMatchesTheFullLoader(t *testing.T) {
	db := newTempDB(t)
	seedDashboardContacts(t, db)

	for _, limit := range []int{10, 3, 1} {
		full, err := ListQSOsFromDate(db, "20261001", limit)
		if err != nil {
			t.Fatalf("ListQSOsFromDate(limit %d): %v", limit, err)
		}
		proj, err := ListDashboardQSOsFromDate(db, "20261001", limit)
		if err != nil {
			t.Fatalf("ListDashboardQSOsFromDate(limit %d): %v", limit, err)
		}
		if len(proj) != len(full) {
			t.Fatalf("limit %d: projection returned %d rows, full loader %d", limit, len(proj), len(full))
		}
		for i := range full {
			if got, want := proj[i], projectionOf(full[i]); got != want {
				t.Errorf("limit %d row %d:\n got %+v\nwant %+v", limit, i, got, want)
			}
		}
	}

	// The date filter has to cut in the same place.
	full, err := ListQSOsFromDate(db, "20261027", 50)
	if err != nil {
		t.Fatalf("ListQSOsFromDate(from 20261027): %v", err)
	}
	proj, err := ListDashboardQSOsFromDate(db, "20261027", 50)
	if err != nil {
		t.Fatalf("ListDashboardQSOsFromDate(from 20261027): %v", err)
	}
	if len(full) != len(proj) || len(proj) != 3 {
		t.Fatalf("from 20261027: full %d rows, projection %d rows; want 3", len(full), len(proj))
	}
	for i := range full {
		if proj[i].ID != full[i].ID {
			t.Errorf("from 20261027 row %d: id %d, want %d (ordering drifted)", i, proj[i].ID, full[i].ID)
		}
	}

	// Newest first, and the same-date tie broken by id (descending).
	all, err := ListDashboardQSOsFromDate(db, "20261001", 50)
	if err != nil {
		t.Fatalf("ListDashboardQSOsFromDate(all): %v", err)
	}
	wantCalls := []string{"SP9SPM", "K2DEF", "K1ABC", "DJ7NT", "SP9MOA"}
	if len(all) != len(wantCalls) {
		t.Fatalf("loaded %d contacts, want %d", len(all), len(wantCalls))
	}
	for i, want := range wantCalls {
		if all[i].Call != want {
			t.Errorf("row %d is %s, want %s (order: newest first, id desc within a minute)", i, all[i].Call, want)
		}
	}
	// Enrichment after the save is carried: K1ABC had no country when it was
	// logged, and the old loader saw the filled-in one.
	enriched := all[2]
	if enriched.Call != "K1ABC" || enriched.Country != "United States" {
		t.Errorf("enrichment missing from the projection: %+v", enriched)
	}
	// The contact without a frequency stays at zero rather than becoming NaN.
	if all[0].Freq != 0 {
		t.Errorf("row without a frequency has freq %v, want 0", all[0].Freq)
	}
}

// TestListDashboardQSOsFromDateEmptyResult keeps "no contacts" a clean empty
// result: the panels show their empty state on it, and the old loader returned
// the same thing.
func TestListDashboardQSOsFromDateEmptyResult(t *testing.T) {
	db := newTempDB(t)

	proj, err := ListDashboardQSOsFromDate(db, "20260101", 5000)
	if err != nil {
		t.Fatalf("empty database: %v", err)
	}
	if len(proj) != 0 {
		t.Errorf("empty database returned %d rows, want 0", len(proj))
	}

	seedDashboardContacts(t, db)
	proj, err = ListDashboardQSOsFromDate(db, "20991231", 5000)
	if err != nil {
		t.Fatalf("future date: %v", err)
	}
	if len(proj) != 0 {
		t.Errorf("date past every contact returned %d rows, want 0", len(proj))
	}
	// LIMIT 0 is the "show nothing" spelling used by callers that cap at zero.
	proj, err = ListDashboardQSOsFromDate(db, "20260101", 0)
	if err != nil || len(proj) != 0 {
		t.Errorf("limit 0: %d rows, err %v; want 0 rows and no error", len(proj), err)
	}
}

// TestListDashboardQSOsFromDateToleratesNulls documents one deliberate
// difference from the general loader: a row an older import left with NULLs is
// rendered as empty instead of failing the whole panel (the general scanner
// rejects the row, which used to blank the dashboard).
func TestListDashboardQSOsFromDateToleratesNulls(t *testing.T) {
	db := newTempDB(t)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20261028", TimeOn: "120000", Band: "20m", Mode: "SSB",
	})
	if _, err := db.Exec(`UPDATE qsos SET band = NULL, submode = NULL, gridsquare = NULL,
		country = NULL, operator = NULL, my_gridsquare = NULL, freq = NULL`); err != nil {
		t.Fatalf("null out the row: %v", err)
	}

	proj, err := ListDashboardQSOsFromDate(db, "20260101", 10)
	if err != nil {
		t.Fatalf("projection failed on NULL columns: %v", err)
	}
	if len(proj) != 1 {
		t.Fatalf("loaded %d rows, want 1", len(proj))
	}
	got := proj[0]
	if got.Call != "SP9MOA" || got.Mode != "SSB" || got.QSODate != "20261028" || got.TimeOn != "120000" {
		t.Errorf("non-null columns lost: %+v", got)
	}
	if got.Band != "" || got.Submode != "" || got.GridSquare != "" || got.Country != "" ||
		got.Operator != "" || got.MyGridSquare != "" || got.Freq != 0 {
		t.Errorf("NULL columns should scan as empty/zero: %+v", got)
	}
}

// BenchmarkDashboardProjection_5k measures the read the dashboard performs on
// the update path: the whole record vs the fourteen-column projection.
func BenchmarkDashboardProjection_5k(b *testing.B) {
	db := benchDashboardDB(b, 5000)
	b.Run("full_record", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := ListQSOsFromDate(db, "20240101", 5000); err != nil {
				b.Fatalf("full load: %v", err)
			}
		}
	})
	b.Run("projection", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := ListDashboardQSOsFromDate(db, "20240101", 5000); err != nil {
				b.Fatalf("projection load: %v", err)
			}
		}
	})
}

// benchDashboardDB seeds n fully-populated contacts, so the comparison is not
// between a sparse row and a rich one.
func benchDashboardDB(b *testing.B, n int) *sql.DB {
	b.Helper()
	db, err := InitDB(filepath.Join(b.TempDir(), "benchdash.db"))
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
		q := &qso.QSO{
			Call: call, QSODate: "20240101", TimeOn: fmt.Sprintf("%02d%02d00", i%24, i%60),
			Band: "20m", Freq: 14.0745, Mode: "MFSK", Submode: "FT8",
			RSTSent: "-10", RSTRcvd: "-12", GridSquare: "JO90WE", MyGridSquare: "KO00AA",
			Name: "Bench Operator", QTH: "Benchville", Country: "Poland",
			Comment: "benchmark comment that the dashboard never renders",
			Notes:   "benchmark notes that the dashboard never renders either",
			TXPower: "100", SOTARef: "SP/SS-001", POTARef: "SP-0001", WWFFRef: "SPFF-0001",
			IOTA: "EU-001", SIG: "SIGA", SIGInfo: "SIGA-1",
			MySOTARef: "SP/SS-002", MyPOTARef: "SP-0002", MyWWFFRef: "SPFF-0002",
			StationCallsign: "SP9MOA", Operator: "SP9MOA", MyRig: "FT-891", MyAntenna: "Dipole",
			Source: "bench", CQZone: "15", ITUZone: "28", DXCC: "269",
			ContestID: "bench-contest", ExchSent: "599 001", ExchRcvd: "599 002",
			STX: 1, SRX: 2, STXString: "001", SRXString: "002",
		}
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
