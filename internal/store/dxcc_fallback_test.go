package store

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// legacyDXCCFallbackExists is the previous check, verbatim, kept as the
// reference the EXISTS form must match. LIMIT 1 there never limited the work —
// it trims the result of an aggregate that had already counted every row.
func legacyDXCCFallbackExists(t *testing.T, db *sql.DB, dxcc, countryName string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM qsos
		 WHERE (dxcc IS NULL OR dxcc = '' OR dxcc != ?) AND country LIKE ? COLLATE NOCASE LIMIT 1`,
		dxcc, strings.ToLower(countryName)+"%",
	).Scan(&n); err != nil {
		t.Fatalf("legacy check: %v", err)
	}
	return n > 0
}

// TestDXCCFallbackExistsMatchesTheCount keeps the existence check honest: the
// decision "use the country-fallback path?" must not change for any row shape,
// casing or empty argument. A difference here would silently change the DXCC
// history the partner panel shows.
func TestDXCCFallbackExistsMatchesTheCount(t *testing.T) {
	db := newTempDB(t)

	// A log with every shape the predicate distinguishes: a matching entity, a
	// different entity with the same country, an empty entity, a NULL entity,
	// and case variants of the country name.
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20260501", TimeOn: "120000",
		Band: "20m", Mode: "CW", DXCC: "269", Country: "Poland",
	})
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SQ9XYZ", QSODate: "20260501", TimeOn: "121000",
		Band: "20m", Mode: "CW", Country: "poland",
	})
	mustInsertQSO(t, db, &qso.QSO{
		Call: "DL1AA", QSODate: "20260501", TimeOn: "122000",
		Band: "40m", Mode: "SSB", DXCC: "230", Country: "Germany",
	})
	if _, err := db.Exec(
		`INSERT INTO qsos (call, base_call, qso_date, time_on, band, mode, dxcc, country, created_at, updated_at)
		 VALUES ('OE1AA', 'OE1AA', '20260501', '123000', '20m', 'CW', NULL, 'AUSTRIA', datetime('now'), datetime('now'))`,
	); err != nil {
		t.Fatalf("insert NULL dxcc row: %v", err)
	}

	args := []struct{ dxcc, country string }{
		{"269", "Poland"},  // modern row matches, legacy country-only row exists
		{"269", "poland"},  // lower-case argument
		{"269", "POLAND"},  // upper-case argument
		{"269", "Germany"}, // only a different entity in Germany
		{"230", "Germany"}, // the row with that entity is excluded, no fallback
		{"269", "Austria"}, // NULL entity matched by country
		{"", "Poland"},     // empty DXCC matches the first disjunct everywhere
		{"269", ""},        // empty country prefix matches every row
		{"999", "ZZZZ"},    // nothing matches at all
	}
	for _, a := range args {
		want := legacyDXCCFallbackExists(t, db, a.dxcc, a.country)
		got, err := dxccCountryFallbackExists(db, a.dxcc, a.country)
		if err != nil {
			t.Fatalf("dxccCountryFallbackExists(%q,%q): %v", a.dxcc, a.country, err)
		}
		if got != want {
			t.Errorf("dxcc=%q country=%q: EXISTS = %v, COUNT form = %v", a.dxcc, a.country, got, want)
		}
	}

	// An empty log has nothing to fall back to.
	empty := newTempDB(t)
	for _, a := range args {
		got, err := dxccCountryFallbackExists(empty, a.dxcc, a.country)
		if err != nil {
			t.Fatalf("empty db: %v", err)
		}
		if got {
			t.Errorf("empty log reported a fallback row for dxcc=%q country=%q", a.dxcc, a.country)
		}
	}
}

// TestScopeStatsDXCCFastMatchesLegacyWithFallbackRows: when the existence check
// fires, the fast path delegates — the two must agree field for field, which is
// what keeps GetWorkedSummary unchanged.
func TestScopeStatsDXCCFastMatchesLegacyWithFallbackRows(t *testing.T) {
	db := newTempDB(t)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20260501", TimeOn: "120000", Band: "20m", Mode: "CW",
		DXCC: "269", Country: "Poland", GridSquare: "JO90",
	})
	// Legacy row: same country, no entity number.
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SQ9XYZ", QSODate: "20260401", TimeOn: "100000", Band: "40m", Mode: "SSB",
		Country: "poland", GridSquare: "KN09",
	})

	fallback, err := dxccCountryFallbackExists(db, "269", "Poland")
	if err != nil {
		t.Fatalf("existence check: %v", err)
	}
	if !fallback {
		t.Fatal("setup: the legacy row was not detected")
	}

	fast, err := scopeStatsDXCCFast(db, "269", "Poland")
	if err != nil {
		t.Fatalf("scopeStatsDXCCFast: %v", err)
	}
	legacy, err := scopeStatsDXCC(db, "269", "Poland")
	if err != nil {
		t.Fatalf("scopeStatsDXCC: %v", err)
	}

	if fast.QSOCount != legacy.QSOCount || fast.UniqueCalls != legacy.UniqueCalls ||
		fast.UniqueBands != legacy.UniqueBands || fast.UniqueModes != legacy.UniqueModes {
		t.Errorf("counters differ: fast=%+v legacy=%+v", fast, legacy)
	}
	if (fast.FirstQSO == nil) != (legacy.FirstQSO == nil) || (fast.LastQSO == nil) != (legacy.LastQSO == nil) {
		t.Errorf("edges differ: fast=%+v/%+v legacy=%+v/%+v", fast.FirstQSO, fast.LastQSO, legacy.FirstQSO, legacy.LastQSO)
	}
	if len(fast.BandCounts) != len(legacy.BandCounts) || len(fast.ModeCounts) != len(legacy.ModeCounts) ||
		len(fast.GridCounts) != len(legacy.GridCounts) {
		t.Errorf("breakdown sizes differ: fast=%d/%d/%d legacy=%d/%d/%d",
			len(fast.BandCounts), len(fast.ModeCounts), len(fast.GridCounts),
			len(legacy.BandCounts), len(legacy.ModeCounts), len(legacy.GridCounts))
	}
}
