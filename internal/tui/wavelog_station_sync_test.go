package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/wavelog"
)

func TestApplyWavelogStation(t *testing.T) {
	st := &wavelog.Station{
		Callsign: "SP9SPM", Gridsquare: "KO00CA", DXCC: 269, CQ: 15, ITU: 28,
		SOTA: "SP/TA-001", POTA: "SP-0001", WWFF: "SPFF-0001", SIG: "SOTA", SIGInfo: "x",
	}
	s := &config.Station{}
	if !applyWavelogStation(st, s) {
		t.Fatal("applyWavelogStation should report a change")
	}
	if s.Callsign != "SP9SPM" || s.Grid != "KO00CA" || s.DXCC != 269 ||
		s.CQZone != 15 || s.ITUZone != 28 || s.SOTARef != "SP/TA-001" ||
		s.POTARef != "SP-0001" || s.WWFFRef != "SPFF-0001" || s.SIG != "SOTA" || s.SIGInfo != "x" {
		t.Errorf("station not fully mirrored: %+v", s)
	}

	// Identical values: no change reported.
	s2 := *s
	if applyWavelogStation(st, &s2) {
		t.Error("second application should report no change")
	}
}

func TestSyncLogbookStationFromWavelog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/station/1" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id": 1, "name": "Home", "callsign": "SP9SPM", "gridsquare": "KO00CA",
				"dxcc": 269, "country": "POLAND", "cq": 15, "itu": 28,
				"sota": "SP/TA-001", "pota": "", "wwff": "", "sig": "", "sig_info": "",
				"active": true,
			},
		})
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.Logbooks["test"] = config.Logbook{ID: "test", Name: "Test", Station: config.Station{
		Callsign: "OLD", Grid: "AA00aa",
	}}

	changed, err := syncLogbookStationFromWavelog(cfg, "test", srv.URL, "wl2_test", "1 — Home (SP9SPM) KO00CA")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !changed {
		t.Fatal("sync should report a change")
	}
	lb := cfg.Logbooks["test"]
	if lb.Station.Grid != "KO00CA" || lb.Station.DXCC != 269 ||
		lb.Station.CQZone != 15 || lb.Station.ITUZone != 28 || lb.Station.SOTARef != "SP/TA-001" {
		t.Errorf("station not synced: %+v", lb.Station)
	}
}

func TestStationFormHidesAdvancedByDefault(t *testing.T) {
	f := NewStationForm("", "", "")
	f.Advanced = false
	f.width = 100

	content := f.View().Content
	if strings.Contains(content, "SOTA Ref") || strings.Contains(content, "DXCC ID") {
		t.Errorf("advanced fields should be hidden:\n%s", content)
	}
	if !strings.Contains(content, "Wavelog:") {
		t.Errorf("Wavelog section missing:\n%s", content)
	}

	f.Advanced = true
	if content := f.View().Content; !strings.Contains(content, "SOTA Ref") || !strings.Contains(content, "DXCC ID") {
		t.Error("advanced fields should render when enabled")
	}
}

func TestStationFormNavigationSkipsHiddenAdvanced(t *testing.T) {
	f := NewStationForm("", "", "")
	f.Advanced = false
	f.HideOperator = true // wizard configuration
	f.HideGPSGrid = true  // wizard hides the GPS/APRS tail
	f.Name.Blur()
	f.contFocus = true

	f.NextInput()
	if !f.wlCbFocus {
		t.Error("NextInput from continent should land on the Wavelog checkbox when advanced fields are hidden")
	}

	f.NextInput() // Wavelog disabled → wraps to Name
	if !f.Name.Focused() {
		t.Error("expected wrap to Name")
	}
}

func TestWizardAppliesSyncedStationOnSave(t *testing.T) {
	w := newTestWizard(t, "SP9SPM", "JO90")
	w.station.Name.SetValue("Home")
	w.wlStation = &wavelog.Station{
		Callsign: "SP9SPM", Gridsquare: "KO00CA", DXCC: 269, CQ: 15, ITU: 28,
		SOTA: "SP/TA-001",
	}

	if err := w.saveConfig(); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	var lb config.Logbook
	for _, v := range w.App.Config.Logbooks {
		lb = v
	}
	if lb.Station.Grid != "KO00CA" || lb.Station.DXCC != 269 ||
		lb.Station.CQZone != 15 || lb.Station.ITUZone != 28 || lb.Station.SOTARef != "SP/TA-001" {
		t.Errorf("saved station should mirror the Wavelog profile: %+v", lb.Station)
	}
}

func TestWizardSetSelectedStationFillsGrid(t *testing.T) {
	w := newTestWizard(t, "SP9SPM", "")
	w.wlStations = []wavelog.StationProfile{{ID: "1", Callsign: "SP9SPM", Name: "Home", Gridsquare: "KO00CA"}}
	w.wlStationIdx = 0

	w.setSelectedStation()
	if w.station.Locator.Value() != "KO00CA" {
		t.Errorf("locator = %q, want KO00CA", w.station.Locator.Value())
	}
	if !strings.Contains(w.station.WlStationID.Value(), "KO00CA") {
		t.Errorf("station id field = %q", w.station.WlStationID.Value())
	}
}

func TestStationFormHidesIARUInWizard(t *testing.T) {
	f := NewStationForm("", "", "")
	f.HideIARU = true
	f.HideGPSGrid = true
	f.width = 100

	content := f.View().Content
	if strings.Contains(content, "IARU Region") {
		t.Errorf("IARU Region should be hidden:\n%s", content)
	}
	if !strings.Contains(content, "Continent:") {
		t.Errorf("Continent selector missing:\n%s", content)
	}

	// Navigation: grid → continent (IARU skipped), and back.
	f.Name.Blur()
	f.Locator.Focus()
	f.NextInput()
	if !f.contFocus {
		t.Error("NextInput from locator should land on continent when IARU hidden")
	}
	f.PrevInput()
	if !f.Locator.Focused() {
		t.Error("PrevInput from continent should land on locator when IARU hidden")
	}

	// Default (config menu): both selectors remain visible.
	f2 := NewStationForm("", "", "")
	f2.width = 100
	c2 := f2.View().Content
	if !strings.Contains(c2, "IARU Region") || !strings.Contains(c2, "Continent:") {
		t.Error("config menu form should show both IARU Region and Continent")
	}
}

func TestStationFormWavelogFlowButtonAboveStationID(t *testing.T) {
	f := NewStationForm("", "", "")
	f.WlEnabled = true
	f.width = 100

	content := f.View().Content
	btnIdx := strings.Index(content, "[ Update ]")
	idIdx := strings.Index(content, "Station ID:")
	if btnIdx < 0 || idIdx < 0 {
		t.Fatalf("missing button or station id:\n%s", content)
	}
	if btnIdx > idIdx {
		t.Errorf("update button should render above the Station ID field:\n%s", content)
	}

	// Tab order: key → button → station id, and back.
	f.Name.Blur()
	f.WlKey.Focus()
	f.NextInput()
	if f.wlBtnFocus != 1 {
		t.Error("NextInput from API key should land on the Update button")
	}
	f.NextInput()
	if !f.WlStationID.Focused() {
		t.Error("NextInput from button should land on the Station ID field")
	}
	f.PrevInput()
	if f.wlBtnFocus != 1 {
		t.Error("PrevInput from Station ID should land on the Update button")
	}
	f.PrevInput()
	if !f.WlKey.Focused() {
		t.Error("PrevInput from button should land on the API key field")
	}
}

func TestFillStationFormFromWavelog(t *testing.T) {
	f := NewStationForm("", "", "")
	f.Callsign.SetValue("USER")
	f.Locator.SetValue("JO90")
	f.SOTARef.SetValue("stale")

	fillStationFormFromWavelog(f, &wavelog.Station{
		Callsign: "SP9SPM", Gridsquare: "KO00CA", DXCC: 269, CQ: 15, ITU: 28,
		SOTA: "SP/TA-001", POTA: "SP-0001", WWFF: "SPFF-0001", SIG: "SOTA", SIGInfo: "x",
	})
	if f.Callsign.Value() != "SP9SPM" || f.Locator.Value() != "KO00CA" {
		t.Errorf("callsign/grid not mirrored: %q / %q", f.Callsign.Value(), f.Locator.Value())
	}
	if f.DXCC.Value() != "269" || f.CQZone.Value() != "15" || f.ITUZone.Value() != "28" {
		t.Errorf("zones not mirrored: %q/%q/%q", f.DXCC.Value(), f.CQZone.Value(), f.ITUZone.Value())
	}
	if f.SOTARef.Value() != "SP/TA-001" || f.POTARef.Value() != "SP-0001" ||
		f.WWFFRef.Value() != "SPFF-0001" || f.SIG.Value() != "SOTA" || f.SIGInfo.Value() != "x" {
		t.Error("reference fields not mirrored")
	}

	// An empty profile clears the optional fields but keeps callsign/grid.
	fillStationFormFromWavelog(f, &wavelog.Station{})
	if f.SOTARef.Value() != "" || f.DXCC.Value() != "" || f.ITUZone.Value() != "" {
		t.Error("empty profile should clear optional fields")
	}
	if f.Callsign.Value() != "SP9SPM" || f.Locator.Value() != "KO00CA" {
		t.Error("empty profile must not clear callsign/grid")
	}
}

func TestWizardStationDetailFillsFormAndGuardsStale(t *testing.T) {
	w := newTestWizard(t, "SP9SPM", "JO90")
	w.wlStations = []wavelog.StationProfile{{ID: "1", Callsign: "SP9SPM", Name: "Home", Gridsquare: "KO00CA"}}
	w.wlStationIdx = 0

	// A stale detail (different station) must not touch the form.
	w.Update(wlStationDetailMsg{stationID: "99", station: &wavelog.Station{Callsign: "ZZ9ZZZ", Gridsquare: "AA00AA"}})
	if w.station.Callsign.Value() != "SP9SPM" {
		t.Errorf("stale detail overwrote callsign: %q", w.station.Callsign.Value())
	}
	if w.wlStation != nil {
		t.Error("stale detail must not be cached for save")
	}

	// A matching detail fills the rest of the form and is cached for save.
	w.Update(wlStationDetailMsg{stationID: "1", station: &wavelog.Station{
		Callsign: "SP9SPM", Gridsquare: "KO00CA", DXCC: 269, CQ: 15, ITU: 28, SOTA: "SP/TA-001",
	}})
	if w.station.Locator.Value() != "KO00CA" || w.station.DXCC.Value() != "269" || w.station.SOTARef.Value() != "SP/TA-001" {
		t.Errorf("detail not applied to form: grid=%q dxcc=%q sota=%q",
			w.station.Locator.Value(), w.station.DXCC.Value(), w.station.SOTARef.Value())
	}
	if w.wlStation == nil {
		t.Fatal("matching detail should be cached for save")
	}
}

func TestLogbookChooserCycleMirrorsStationFields(t *testing.T) {
	a := newChooserTestApp(t)
	c := NewLogbookChooser(a, NewToastQueue())
	c.mode = chooserEdit
	c.wlStations = []wavelog.StationProfile{
		{ID: "7", Callsign: "SP9SPM", Name: "Home", Gridsquare: "KO00CA"},
		{ID: "8", Callsign: "SQ8ABC", Name: "Field", Gridsquare: "KN09AA"},
	}
	c.wlStationIdx = 0
	c.updateStationIDField()
	if c.station.Locator.Value() != "KO00CA" || c.station.Callsign.Value() != "SP9SPM" {
		t.Fatalf("initial mirror wrong: grid=%q call=%q", c.station.Locator.Value(), c.station.Callsign.Value())
	}
	if c.wlStationID != "7" {
		t.Fatalf("wlStationID = %q, want 7", c.wlStationID)
	}

	// Space over the Station ID field cycles to the next station and the
	// form mirrors it immediately.
	c.station.WlStationID.Focus()
	c.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if c.wlStationIdx != 1 {
		t.Fatalf("wlStationIdx = %d, want 1", c.wlStationIdx)
	}
	if c.station.Locator.Value() != "KN09AA" || c.station.Callsign.Value() != "SQ8ABC" {
		t.Errorf("cycle did not mirror: grid=%q call=%q", c.station.Locator.Value(), c.station.Callsign.Value())
	}
	if c.wlStationID != "8" {
		t.Errorf("wlStationID = %q, want 8", c.wlStationID)
	}
}
