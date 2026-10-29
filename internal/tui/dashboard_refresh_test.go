package tui

import (
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// resetDashboardFingerprints clears the package-level panel caches (they are
// globals shared by the whole model) so a test starts from "nothing pushed".
func resetDashboardFingerprints(t *testing.T) {
	t.Helper()
	lastTodayFP = panelFingerprint{}
	lastRecentFP = panelFingerprint{}
	t.Cleanup(func() {
		lastTodayFP = panelFingerprint{}
		lastRecentFP = panelFingerprint{}
	})
}

// TestDashboardTodayRepaintsAfterFieldOnlyChange is the regression test for
// the "same QSO has a country in Recent but none in Top QSOs" dashboard bug.
//
// WSJT-X auto-logs a QSO without a country, the dashboard pushes the Today
// list (Top QSOs + map + stats are built from it), and only then does the
// callbook/CTY enrichment fill the country via store.UpdateQSOEnrichment —
// which changes no row ids. The Today panel used to fingerprint rows by id
// alone, skip the push, and keep the countryless row forever, so a browser
// reload could not help: the server snapshot itself was stale. Recent was
// force-pushed after enrichment, hence the mismatch the operator saw.
func TestDashboardTodayRepaintsAfterFieldOnlyChange(t *testing.T) {
	resetDashboardFingerprints(t)
	m := newLifecycleTestModel(t)
	// Include every QSO date in the today query.
	m.App.Config.Integrations.HTTPServer.EventStart = "2026-01-01"

	id, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "DJ7NT", QSODate: "20260315", TimeOn: "120000",
		Band: "20m", Mode: "FT8", GridSquare: "JO31",
		RSTSent: "+05", RSTRcvd: "-11",
	})
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	ds := dashboard.NewState(dashboard.NewHub())
	m.pushDashboardToday(ds)
	snap := ds.Snapshot()
	if len(snap.Today) != 1 {
		t.Fatalf("today panel has %d rows, want 1", len(snap.Today))
	}
	if got := snap.Today[0].Country; got != "" {
		t.Fatalf("precondition: today country = %q, want empty", got)
	}

	// Enrichment writes the country; the row id is unchanged.
	if err := store.UpdateQSOEnrichment(m.App.DB, id, store.EnrichmentData{
		Country: "Fed. Rep. of Germany",
	}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}

	m.pushDashboardToday(ds)
	snap = ds.Snapshot()
	if got := snap.Today[0].Country; got != "Fed. Rep. of Germany" {
		t.Errorf("today panel kept stale country %q after enrichment — Top QSOs would show no country while Recent has it", got)
	}
}

// TestDashboardTodaySkipsUnchangedContent guards the other direction: the
// content fingerprint must still suppress redundant pushes when nothing
// changed (this is what keeps the periodic push cheap).
func TestDashboardTodaySkipsUnchangedContent(t *testing.T) {
	resetDashboardFingerprints(t)
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.HTTPServer.EventStart = "2026-01-01"

	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "SP9MOA", QSODate: "20260315", TimeOn: "120000",
		Band: "40m", Mode: "SSB", GridSquare: "JO90",
	}); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	ds := dashboard.NewState(dashboard.NewHub())
	m.pushDashboardToday(ds)

	// Replace the panel with a marker; an unchanged re-push must not repaint.
	ds.SetToday([]dashboard.QSOView{{ID: "marker", Call: "MARKER"}})
	m.pushDashboardToday(ds)

	got := ds.Snapshot().Today
	if len(got) != 1 || got[0].Call != "MARKER" {
		t.Errorf("unchanged data still pushed (panel = %+v), fingerprint is not suppressing redundant work", got)
	}
}

// TestDashboardRecentRepaintsAfterFieldOnlyChange covers the Recent panel's
// half of the same bug, so both panels stay consistent.
func TestDashboardRecentRepaintsAfterFieldOnlyChange(t *testing.T) {
	resetDashboardFingerprints(t)
	m := newLifecycleTestModel(t)

	today := time.Now().UTC().Format("20060102")
	id, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "K1ABC", QSODate: today, TimeOn: "120000",
		Band: "20m", Mode: "FT8", GridSquare: "FN42",
	})
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	ds := dashboard.NewState(dashboard.NewHub())
	m.pushDashboardRecent(ds)
	if len(ds.Snapshot().Recent) == 0 {
		t.Fatal("recent panel is empty, want the inserted QSO")
	}

	if err := store.UpdateQSOEnrichment(m.App.DB, id, store.EnrichmentData{
		Country: "United States",
	}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}

	m.pushDashboardRecent(ds)
	recent := ds.Snapshot().Recent
	if recent[0].Country != "United States" {
		t.Errorf("recent panel country = %q after enrichment, want United States", recent[0].Country)
	}
}

// TestPanelFingerprintTracksRenderedFields documents which changes must
// invalidate a panel and which must not. The fingerprint reads the dashboard
// projection, so anything the panels render has to live in that projection —
// this table is what keeps the two in step.
func TestPanelFingerprintTracksRenderedFields(t *testing.T) {
	base := []store.DashboardQSO{{ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", Country: ""}}
	fp := newPanelFingerprint(base)
	if fp.equal(panelFingerprint{}) {
		t.Error("an unset fingerprint must never match (forced push spelling)")
	}
	if !fp.equal(newPanelFingerprint([]store.DashboardQSO{{ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", Country: ""}})) {
		t.Error("identical data must produce an equal fingerprint")
	}

	changes := map[string]store.DashboardQSO{
		"country":  {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", Country: "Poland"},
		"grid":     {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", GridSquare: "JO90"},
		"my grid":  {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", MyGridSquare: "KO00"},
		"mode":     {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "SSB"},
		"submode":  {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "MFSK", Submode: "FT4"},
		"rst sent": {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", RSTSent: "-10"},
		"freq":     {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", Freq: 14.0745},
		"date":     {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", QSODate: "20261028"},
		"time":     {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", TimeOn: "120000"},
		"operator": {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8",
			Operator: "SP9MOA"},
	}
	for name, changed := range changes {
		if fp.equal(newPanelFingerprint([]store.DashboardQSO{changed})) {
			t.Errorf("%s change did not invalidate the fingerprint", name)
		}
	}

	if fp.equal(newPanelFingerprint([]store.DashboardQSO{{ID: 8, Call: "DJ7NT", Band: "20m", Mode: "FT8"}})) {
		t.Error("a different row id did not invalidate the fingerprint")
	}
}

// TestDashboardStationCarriesBeaconPosition is the regression test for
// "the position sent to APRS-IS differs from the dot on the dashboard map".
//
// The beacon transmits the fixed-location-aware chain (aprsGrid: fixed
// location > GPS > station grid). The station's logging grid follows
// effectiveGrid (GPS > station grid) and deliberately ignores the fixed
// location, so with a fixed APRS position configured the two legitimately
// differ. The dashboard used to build the map marker from the logging grid,
// which put the dot at the station grid while the network saw the fixed
// point. The payload must therefore carry both: Locator/Lat/Lon for QSO
// distance math, AprsLocator/AprsPos for everything APRS on the map.
func TestDashboardStationCarriesBeaconPosition(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Logbook.APRS = &config.APRSConfig{
		Enabled:       true,
		FixedLocation: "KO00AA11AA", // intentionally different from JO90
		RadiusKm:      50,
	}

	srv := dashboard.New("127.0.0.1", "0")
	m.http.client = srv
	m.http.online = true
	m.pushDashboardFast()

	st := srv.State().Snapshot().Station

	// The logging grid keeps driving the QSO distance/bearing side.
	if st.Locator != "JO90" {
		t.Errorf("station locator = %q, want logging grid JO90", st.Locator)
	}
	// The APRS side must match what the beacon actually transmits.
	if st.AprsLocator != "KO00AA11AA" {
		t.Fatalf("aprsLocator = %q, want transmitted fixed location KO00AA11AA", st.AprsLocator)
	}
	if st.AprsPos == nil {
		t.Fatal("aprsPos is nil, the dashboard has no APRS position to draw")
	}
	wantLat, wantLon, ok := gridToLatLonOK(st.AprsLocator)
	if !ok {
		t.Fatalf("gridToLatLonOK(%q) failed", st.AprsLocator)
	}
	if st.AprsPos.Lat != wantLat || st.AprsPos.Lon != wantLon {
		t.Errorf("aprsPos = %+v, want the transmitted point %f,%f", *st.AprsPos, wantLat, wantLon)
	}
	// Guard the bug itself: the APRS dot must not sit on the logging grid.
	if st.AprsPos.Lat == st.Lat && st.AprsPos.Lon == st.Lon {
		t.Errorf("aprsPos coincides with the logging grid %f,%f; the marker would "+
			"disagree with the transmitted position", st.Lat, st.Lon)
	}
	if st.AprsRadiusKm != 50 {
		t.Errorf("aprsRadiusKm = %v, want 50", st.AprsRadiusKm)
	}

	// Disabled APRS: no APRS position is advertised at all.
	m.App.Logbook.APRS.Enabled = false
	m.pushDashboardFast()
	if st := srv.State().Snapshot().Station; st.AprsPos != nil || st.AprsLocator != "" {
		t.Errorf("disabled APRS still published aprsPos=%+v aprsLocator=%q", st.AprsPos, st.AprsLocator)
	}
}

// TestWSJTXEnrichmentRepaintsBothPanels drives the real owner-loop message
// that WSJT-X enrichment ends with, instead of calling the push helpers
// directly. The reported symptom was "the country shows in Recent QSOs but
// not in Top QSOs" — Top QSOs and the map are built from the Today list, so
// any enrichment completion that refreshes only one panel reproduces it.
func TestWSJTXEnrichmentRepaintsBothPanels(t *testing.T) {
	resetDashboardFingerprints(t)
	m := newLifecycleTestModel(t)
	// Include every QSO date in the today query.
	m.App.Config.Integrations.HTTPServer.EventStart = "2026-01-01"

	// WSJT-X auto-logs the QSO without a country (no callbook yet).
	id, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "DJ7NT", QSODate: "20260315", TimeOn: "120000",
		Band: "20m", Mode: "FT8", GridSquare: "JO31",
		RSTSent: "+05", RSTRcvd: "-11",
	})
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	srv := dashboard.New("127.0.0.1", "0")
	m.http.client = srv
	m.http.online = true
	ds := srv.State()
	m.pushDashboardToday(ds)
	m.pushDashboardRecent(ds)

	// Precondition: the row is visible in both panels without a country.
	pre := srv.State().Snapshot()
	if len(pre.Today) != 1 || len(pre.Recent) != 1 {
		t.Fatalf("precondition: today=%d recent=%d rows, want 1/1", len(pre.Today), len(pre.Recent))
	}
	if pre.Today[0].Country != "" || pre.Recent[0].Country != "" {
		t.Fatalf("precondition: country already set (today=%q recent=%q)",
			pre.Today[0].Country, pre.Recent[0].Country)
	}

	// The worker enriches the row (country only — no id change), then the
	// owner loop receives the completion message.
	if err := store.UpdateQSOEnrichment(m.App.DB, id, store.EnrichmentData{Country: "Fed. Rep. of Germany"}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}
	// The returned command only refreshes the TUI table; the dashboard push
	// happens synchronously inside the handler.
	m.Update(wsjtxEnrichDoneMsg{logbook: m.App.LogbookName})

	snap := srv.State().Snapshot()
	if len(snap.Today) != 1 || len(snap.Recent) != 1 {
		t.Fatalf("today=%d recent=%d rows, want 1/1", len(snap.Today), len(snap.Recent))
	}
	if got := snap.Today[0].Country; got != "Fed. Rep. of Germany" {
		t.Errorf("Top QSOs row country = %q, want the enriched country", got)
	}
	if got := snap.Recent[0].Country; got != "Fed. Rep. of Germany" {
		t.Errorf("Recent QSOs row country = %q, want the enriched country", got)
	}
}
