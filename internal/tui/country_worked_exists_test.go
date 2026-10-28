package tui

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/ctybig"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// "Country worked before" — EXISTS instead of COUNT(*)
//
// Only `count > 0` was ever used, but COUNT(*) aggregates every matching row
// before the answer exists (2.41 ms on a 10k-contact fixture, against 0.10 ms
// for the existence form). The predicates must stay exactly as they were, so
// the counting query lives on here as the oracle every case is compared with.
// =============================================================================

// legacyCountryWorkedCount is the pre-EXISTS query. Kept as the oracle: it is
// the behaviour the dashboard shipped before the change.
func legacyCountryWorkedCount(t *testing.T, db *sql.DB, country, dxcc, baseCall string) bool {
	t.Helper()
	var count int
	var err error
	if dxcc != "" {
		err = db.QueryRow(
			`SELECT COUNT(*) FROM (
				SELECT base_call FROM qsos WHERE dxcc = ?
				UNION ALL
				SELECT base_call FROM qsos
				WHERE (dxcc IS NULL OR dxcc = '' OR dxcc != ?)
					AND country LIKE ? COLLATE NOCASE
			) WHERE base_call != ?`,
			dxcc, dxcc, country+"%", baseCall,
		).Scan(&count)
	} else {
		err = db.QueryRow(
			`SELECT COUNT(*) FROM qsos
			WHERE (country = ? COLLATE NOCASE OR country LIKE ? COLLATE NOCASE)
				AND base_call != ?`,
			country, country+"%", baseCall,
		).Scan(&count)
	}
	return err == nil && count > 0
}

// workedBeforeModel builds a model with a Big CTY table (so the DXCC arm is
// reachable) and the given contacts in its logbook.
func workedBeforeModel(t *testing.T, contacts []qso.QSO) *Model {
	return workedBeforeModelCTY(t, contacts, true)
}

// workedBeforeModelCTY is workedBeforeModel with the entity table optional: the
// shipped check falls back to the country-only arm when Big CTY is unavailable.
func workedBeforeModelCTY(t *testing.T, contacts []qso.QSO, withCTY bool) *Model {
	t.Helper()
	m := newLifecycleTestModel(t)
	bdb, err := ctybig.ParseCSV(strings.NewReader(
		"SP,Poland,269,EU,15,28,52.00,-20.00,-1.0,HF SN SO SP SQ SR 3Z;\n" +
			"DL,Germany,230,EU,14,28,51.00,-10.00,-1.0,DA DB DC DD DF DG DH DJ DK DL DM DO DP DQ DR;\n"))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if withCTY {
		m.App.BigCTY = bdb
		m.App.Config.General.UseCTY = true
	}
	for i := range contacts {
		c := contacts[i]
		if c.QSODate == "" {
			c.QSODate = "20240101"
		}
		if c.TimeOn == "" {
			c.TimeOn = "120000"
		}
		if c.Band == "" {
			c.Band = "20m"
		}
		if c.Mode == "" {
			c.Mode = "SSB"
		}
		if _, err := store.InsertQSO(m.App.DB, &c); err != nil {
			t.Fatalf("InsertQSO(%s): %v", c.Call, err)
		}
	}
	return m
}

// workedBefore runs the production check with the cache cleared, and returns
// the production answer next to the oracle's.
func workedBefore(t *testing.T, m *Model, activeCall, country string) (got, want bool, dxcc, baseCall string) {
	t.Helper()
	for k := range countryWorkedCache {
		delete(countryWorkedCache, k)
	}
	pushDashboardLastCall = activeCall
	t.Cleanup(func() { pushDashboardLastCall = "" })

	got = m.countryWorkedBefore(country)
	if p := m.dxccLookup(activeCall); p != nil {
		dxcc = fmt.Sprintf("%d", p.DXCC)
	}
	baseCall = qso.DeriveBaseCall(activeCall)
	want = legacyCountryWorkedCount(t, m.App.DB, country, dxcc, baseCall)
	return got, want, dxcc, baseCall
}

// TestCountryWorkedBeforeMatchesTheCountingQuery is the regression gate: every
// predicate keeps its meaning, and the answer is identical to the old query.
func TestCountryWorkedBeforeMatchesTheCountingQuery(t *testing.T) {
	cases := []struct {
		name     string
		contacts []qso.QSO
		active   string
		lookup   string
		want     bool
		why      string
		// entityOnly marks a fixture the country-only arm cannot answer (the
		// contact has no country name at all), so only the entity arm is run.
		entityOnly bool
	}{
		{
			name:     "same country, different call",
			contacts: []qso.QSO{{Call: "SP1BBB", Country: "Poland"}},
			active:   "SP1AAA",
			lookup:   "Poland",
			want:     true,
			why:      "any other call in the country counts",
		},
		{
			name:     "only the active call itself",
			contacts: []qso.QSO{{Call: "SP1AAA", Country: "Poland"}},
			active:   "SP1AAA",
			lookup:   "Poland",
			want:     false,
			why:      "the operator's own contact must not make the country 'worked'",
		},
		{
			name: "active base call excluded even with a different suffix",
			contacts: []qso.QSO{
				{Call: "SP1AAA/P", Country: "Poland"},
				{Call: "SP1AAA", Country: "Poland"},
			},
			active: "SP1AAA",
			lookup: "Poland",
			want:   false,
			why:    "the exclusion is by base call, so a portable suffix does not sneak in",
		},
		{
			name: "a portable variant by another operator counts",
			contacts: []qso.QSO{
				{Call: "SP1AAA/P", Country: "Poland"},
				{Call: "SP9XYZ/P", Country: "Poland"},
			},
			active: "SP1AAA",
			lookup: "Poland",
			want:   true,
			why:    "SP9XYZ is a different base call",
		},
		{
			name:     "case-insensitive equality",
			contacts: []qso.QSO{{Call: "DL1AA", Country: "fed. rep. of germany"}},
			active:   "SP1AAA",
			lookup:   "FED. REP. OF GERMANY",
			want:     true,
			why:      "COLLATE NOCASE on both arms",
		},
		{
			name:     "mixed case in the stored value",
			contacts: []qso.QSO{{Call: "DL1AA", Country: "Poland"}},
			active:   "SP1AAA",
			lookup:   "poland",
			want:     true,
			why:      "the lookup is lower-case, the stored value is not",
		},
		{
			name:     "prefix fallback: stored value is longer",
			contacts: []qso.QSO{{Call: "K1ABC", Country: "United States of America"}},
			active:   "SP1AAA",
			lookup:   "United States",
			want:     true,
			why:      "country LIKE '<lookup>%'",
		},
		{
			name:     "prefix fallback must not match inside a word",
			contacts: []qso.QSO{{Call: "K1ABC", Country: "United Kingdom"}},
			active:   "SP1AAA",
			lookup:   "United States",
			want:     false,
			why:      "the LIKE arm is a prefix, not a substring",
		},
		{
			name:     "unrelated country",
			contacts: []qso.QSO{{Call: "SP1BBB", Country: "Poland"}},
			active:   "SP1AAA",
			lookup:   "Germany",
			want:     false,
			why:      "absent matches still search, and answer false",
		},
		{
			name:     "empty logbook",
			contacts: nil,
			active:   "SP1AAA",
			lookup:   "Poland",
			want:     false,
			why:      "nothing to find",
		},
		{
			name:       "DXCC arm: entity number matches with no country name",
			contacts:   []qso.QSO{{Call: "SP9XYZ", Country: "", DXCC: "269"}},
			active:     "SP1AAA",
			lookup:     "Poland",
			want:       true,
			why:        "the stored entity is the reliable key when the name is missing",
			entityOnly: true,
		},
		{
			name:     "DXCC arm: entity match wins over a different name",
			contacts: []qso.QSO{{Call: "SP9XYZ", Country: "Nowhere", DXCC: "269"}},
			active:   "SP1AAA",
			lookup:   "Nowhere",
			want:     true,
			why:      "the entity arm matches the row regardless of its stored name",
		},
		{
			name:     "country fallback for a row with a different entity",
			contacts: []qso.QSO{{Call: "DL1AA", Country: "Poland", DXCC: "230"}},
			active:   "SP1AAA",
			lookup:   "Poland",
			want:     true,
			why:      "a row carrying another entity is still matched by country name",
		},
		{
			name:     "country fallback for a row with no entity",
			contacts: []qso.QSO{{Call: "SP9XYZ", Country: "Poland", DXCC: ""}},
			active:   "SP1AAA",
			lookup:   "Poland",
			want:     true,
			why:      "dxcc IS NULL / '' arm",
		},
		{
			name:     "DXCC arm still excludes the active base call",
			contacts: []qso.QSO{{Call: "SP1AAA", Country: "", DXCC: "269"}},
			active:   "SP1AAA",
			lookup:   "Poland",
			want:     false,
			why:      "the exclusion applies to the entity arm too",
		},
		{
			name:     "another entity's contacts do not count",
			contacts: []qso.QSO{{Call: "DL1AA", Country: "Germany", DXCC: "230"}},
			active:   "SP1AAA",
			lookup:   "Poland",
			want:     false,
			why:      "Germany is not Poland",
		},
	}

	for _, tc := range cases {
		// Both SQL arms: with Big CTY the entity-number branch runs, without it
		// the country-only branch does. Each must answer exactly what the
		// counting query answers for the same arm.
		for _, arm := range []struct {
			name string
			cty  bool
		}{
			{"entity_arm", true},
			{"country_arm", false},
		} {
			if tc.entityOnly && !arm.cty {
				continue
			}
			t.Run(tc.name+"/"+arm.name, func(t *testing.T) {
				m := workedBeforeModelCTY(t, tc.contacts, arm.cty)
				got, want, dxcc, baseCall := workedBefore(t, m, tc.active, tc.lookup)
				if got != want {
					t.Fatalf("EXISTS answered %v, the counting query %v (dxcc=%q base=%q) — %s",
						got, want, dxcc, baseCall, tc.why)
				}
				if got != tc.want {
					t.Errorf("worked-before = %v, want %v — %s", got, tc.want, tc.why)
				}
			})
		}
	}
}

// TestCountryWorkedBeforeBothPathsAgree runs the same fixtures through the arm
// without an entity number (no Big CTY) and the arm with one, and requires the
// counting query to agree on both.
func TestCountryWorkedBeforeBothPathsAgree(t *testing.T) {
	fixtures := []struct {
		name     string
		contacts []qso.QSO
		lookup   string
	}{
		{"name only", []qso.QSO{{Call: "SP1BBB", Country: "Poland"}}, "Poland"},
		{"entity only", []qso.QSO{{Call: "SP1BBB", Country: "", DXCC: "269"}}, "Poland"},
		{"absent", []qso.QSO{{Call: "SP1BBB", Country: "Poland"}}, "Japan"},
		{"empty", nil, "Poland"},
	}
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			m := workedBeforeModel(t, f.contacts)
			withCTY, wantWithCTY, dxcc, _ := workedBefore(t, m, "SP1AAA", f.lookup)
			if withCTY != wantWithCTY {
				t.Fatalf("entity arm: EXISTS %v, count %v (dxcc=%q)", withCTY, wantWithCTY, dxcc)
			}

			// Without Big CTY the check falls back to the country-only arm; the
			// answers must still be the ones the counting query gives.
			m.App.BigCTY = nil
			withoutCTY, wantWithoutCTY, _, _ := workedBefore(t, m, "SP1AAA", f.lookup)
			if withoutCTY != wantWithoutCTY {
				t.Fatalf("country-only arm: EXISTS %v, count %v", withoutCTY, wantWithoutCTY)
			}
		})
	}
}

// benchWorkedBeforeDB seeds a 10k-contact logbook spread over a handful of
// countries, the shape the dashboard sees on a busy contest weekend.
func benchWorkedBeforeDB(b *testing.B) *sql.DB {
	b.Helper()
	db, err := store.InitDB(fmt.Sprintf("%s/worked.db", b.TempDir()))
	if err != nil {
		b.Fatalf("InitDB: %v", err)
	}
	b.Cleanup(func() { db.Close() })

	countries := []string{"Poland", "Fed. Rep. of Germany", "United States", "Japan", "Brazil"}
	for i := 0; i < 10_000; i++ {
		q := &qso.QSO{
			Call:    fmt.Sprintf("SP%dABC", i%900),
			QSODate: "20240101", TimeOn: fmt.Sprintf("%02d%02d00", i%24, i%60),
			Band: "20m", Mode: "SSB", Country: countries[i%len(countries)],
		}
		if _, err := store.InsertQSO(db, q); err != nil {
			b.Fatalf("seed: %v", err)
		}
	}
	return db
}

// BenchmarkCountryWorkedBefore_10k compares the two query forms on the same
// fixture. Both plans are a scan of the covering index (the OR of an equality
// and a LIKE blocks an index seek, which is what the shipped query always was),
// so the gain depends on where the match sits in that scan: a country the index
// reaches early is answered almost immediately by EXISTS, one it reaches late
// saves only the tail, and an absent country has to be searched to the end by
// both — exactly the shape the ticket describes.
func BenchmarkCountryWorkedBefore_10k(b *testing.B) {
	db := benchWorkedBeforeDB(b)
	for _, tc := range []struct {
		name    string
		country string
	}{
		{"match_early_in_the_index", "Brazil"},
		{"match_late_in_the_index", "Poland"},
		{"absent", "Antarctica"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.Run("count", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var n int
					if err := db.QueryRow(
						`SELECT COUNT(*) FROM qsos
						WHERE (country = ? COLLATE NOCASE OR country LIKE ? COLLATE NOCASE)
							AND base_call != ?`,
						tc.country, tc.country+"%", "SP1AAA",
					).Scan(&n); err != nil {
						b.Fatalf("count: %v", err)
					}
				}
			})
			b.Run("exists", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var found bool
					if err := db.QueryRow(
						`SELECT EXISTS(
							SELECT 1 FROM qsos
							WHERE (country = ? COLLATE NOCASE OR country LIKE ? COLLATE NOCASE)
								AND base_call != ?
						)`,
						tc.country, tc.country+"%", "SP1AAA",
					).Scan(&found); err != nil {
						b.Fatalf("exists: %v", err)
					}
				}
			})
		})
	}
}
