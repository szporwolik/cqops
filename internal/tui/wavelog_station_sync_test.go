package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

// TestStationSyncAfterSaveAppliesOnOwnerLoop verifies the background station
// sync only FETCHES in its worker: the profile is returned as a typed
// message and the config mutation, save, active-logbook pointer and toast
// all happen in the model's global handler (owner loop).
func TestStationSyncAfterSaveAppliesOnOwnerLoop(t *testing.T) {
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

	a := newChooserTestApp(t)
	home := a.Config.Logbooks["home"]
	a.Logbook = &home
	m := New(a, nil)

	// The saved logbook has stale station values.
	lb := a.Config.Logbooks["home"]
	lb.Station = config.Station{Callsign: "OLD", Grid: "AA00aa"}
	a.Config.Logbooks["home"] = lb

	c := NewLogbookChooser(a, NewToastQueue())
	cmd := c.syncStationAfterSaveCmd("home", &config.WavelogConfig{
		Enabled: true, URL: srv.URL, APIKey: "wl2_test", StationProfileID: "1",
	})
	if cmd == nil {
		t.Fatal("syncStationAfterSaveCmd returned nil")
	}
	sd, ok := execCmd(cmd).(stationSyncDoneMsg)
	if !ok {
		t.Fatalf("expected stationSyncDoneMsg, got %T", execCmd(cmd))
	}
	if sd.err != nil || sd.st == nil {
		t.Fatalf("sync fetch failed: st=%v err=%v", sd.st, sd.err)
	}
	if sd.lbID != "home" {
		t.Errorf("lbID = %q, want home", sd.lbID)
	}
	if sd.gen != logbookSyncGen.Load() {
		t.Errorf("gen = %d, want %d", sd.gen, logbookSyncGen.Load())
	}

	// Owner loop: the model's global handler installs the result.
	followUp := m.handleStationSyncDone(sd)
	if followUp == nil {
		t.Fatal("expected the logbook-switched follow-up command")
	}

	lb = a.Config.Logbooks["home"]
	if lb.Station.Grid != "KO00CA" || lb.Station.DXCC != 269 ||
		lb.Station.CQZone != 15 || lb.Station.ITUZone != 28 || lb.Station.SOTARef != "SP/TA-001" {
		t.Errorf("station not synced on owner loop: %+v", lb.Station)
	}

	// The synced config was persisted.
	data, err := os.ReadFile(a.ConfigPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !strings.Contains(string(data), "KO00CA") {
		t.Error("saved config does not contain the synced grid")
	}

	// The follow-up keeps the logbook-switch flow alive.
	next := execCmd(followUp)
	if _, ok := next.(logbookSwitchedMsg); !ok {
		t.Errorf("follow-up = %T, want logbookSwitchedMsg", next)
	}
}

// TestStationSyncDoneHandledAwayFromChooser reproduces the dropped-result
// bug: the user saves a logbook with Wavelog enabled and leaves the chooser
// before the station request finishes. The completion must still be applied
// globally, and the switch bookkeeping must still follow.
func TestStationSyncDoneHandledAwayFromChooser(t *testing.T) {
	a := newChooserTestApp(t)
	home := a.Config.Logbooks["home"]
	a.Logbook = &home
	m := New(a, nil)
	m.screen = screenQSO // the chooser has already been left

	lb := a.Config.Logbooks["home"]
	lb.Station = config.Station{Callsign: "OLD", Grid: "AA00aa"}
	a.Config.Logbooks["home"] = lb

	// The save bumped the generation; the result arrives afterwards.
	gen := logbookSyncGen.Add(1)
	upd, cmd := m.Update(stationSyncDoneMsg{
		lbID: "home", gen: gen,
		st: &wavelog.Station{Callsign: "SP9SPM", Gridsquare: "KO00CA", DXCC: 269, CQ: 15, ITU: 28},
	})
	m = upd.(*Model)
	if cmd == nil {
		t.Fatal("expected the logbook-switched follow-up command")
	}
	next := execCmd(cmd)
	if _, ok := next.(logbookSwitchedMsg); !ok {
		t.Fatalf("follow-up = %T, want logbookSwitchedMsg", next)
	}

	lb = a.Config.Logbooks["home"]
	if lb.Station.Grid != "KO00CA" || lb.Station.DXCC != 269 {
		t.Errorf("station not synced away from the chooser: %+v", lb.Station)
	}
}

// TestStationSyncDoneStaleGenerationDiscarded verifies the config-revision
// guard: a sync result from a superseded save must not overwrite newer
// station data nor re-trigger switch bookkeeping.
func TestStationSyncDoneStaleGenerationDiscarded(t *testing.T) {
	a := newChooserTestApp(t)
	home := a.Config.Logbooks["home"]
	a.Logbook = &home
	m := New(a, nil)

	lb := a.Config.Logbooks["home"]
	lb.Station = config.Station{Callsign: "NEW", Grid: "JO90"}
	a.Config.Logbooks["home"] = lb

	// Two saves happen; the result belongs to the first one.
	gen := logbookSyncGen.Add(1) // first save
	logbookSyncGen.Add(1)        // second save supersedes it

	upd, cmd := m.Update(stationSyncDoneMsg{
		lbID: "home", gen: gen,
		st: &wavelog.Station{Callsign: "SP9SPM", Gridsquare: "KO00CA"},
	})
	m = upd.(*Model)
	if cmd != nil {
		t.Error("a stale sync result must not produce the switch follow-up")
	}
	lb = a.Config.Logbooks["home"]
	if lb.Station.Grid != "JO90" || lb.Station.Callsign != "NEW" {
		t.Errorf("stale station data overwrote newer configuration: %+v", lb.Station)
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
