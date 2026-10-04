package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

// richSourceLogbook decorates the "home" logbook with sub-options worth
// duplicating: GPS grid, an operator, Wavelog with an API key, and APRS.
func richSourceLogbook(a map[string]config.Logbook) {
	home := a["home"]
	home.Station.GPSGrid = true
	home.Station.SOTARef = "SP/OG-001"
	home.ActiveOperator = "op1"
	home.Wavelog = &config.WavelogConfig{
		Enabled:          true,
		URL:              "https://wl.example/api",
		APIKey:           "secret-key",
		StationProfileID: "42",
		SharedClub:       true,
	}
	home.APRS = &config.APRSConfig{
		Enabled:      true,
		Callsign:     "SP9MOA-7",
		RadiusKm:     50,
		SendLocation: true,
		IntervalMin:  15,
		Symbol:       "/-",
		Comment:      "CQOps test",
	}
	a["home"] = home
}

func TestLogbookChooserDuplicatePrefillsCreateForm(t *testing.T) {
	a := newChooserTestApp(t)
	a.Config.Operators = map[string]config.Operator{
		"op1": {ID: "op1", Callsign: "SP9OPR", Name: "Operator One"},
	}
	richSourceLogbook(a.Config.Logbooks)

	c := NewLogbookChooser(a, NewToastQueue())
	c.duplicateSelected()

	if c.mode != chooserCreate {
		t.Fatalf("mode = %v, want chooserCreate", c.mode)
	}
	if c.editing != "" {
		t.Errorf("editing = %q, want empty (new logbook, not an edit)", c.editing)
	}
	if got := c.station.Name.Value(); got != "Home QTH - COPY" {
		t.Errorf("name = %q, want %q", got, "Home QTH - COPY")
	}
	if got := c.station.Callsign.Value(); got != "SP9MOA" {
		t.Errorf("callsign = %q, want SP9MOA", got)
	}
	if got := c.station.Locator.Value(); got != "JO90" {
		t.Errorf("grid = %q, want JO90", got)
	}
	if !c.station.GPSGrid {
		t.Error("GPSGrid should be copied")
	}
	if got := c.station.SelectedOperatorCallsign(); got != "SP9OPR" {
		t.Errorf("operator = %q, want SP9OPR", got)
	}

	// Wavelog — enabled with the working key and station profile.
	_, _, _, _, _, _, _, wlEnabled, wlURL, wlKey, wlStationID, _, _, _, _, _, _, _, wlSharedClub := c.station.Values()
	if !wlEnabled || wlURL != "https://wl.example/api" || wlKey != "secret-key" || wlStationID != "42" || !wlSharedClub {
		t.Errorf("wavelog values not copied: enabled=%v url=%q key=%q station=%q club=%v",
			wlEnabled, wlURL, wlKey, wlStationID, wlSharedClub)
	}

	// APRS sub-options.
	aprs := c.station.APRSValues()
	if aprs == nil || !aprs.Enabled || aprs.Callsign != "SP9MOA-7" || aprs.RadiusKm != 50 || aprs.IntervalMin != 15 || aprs.Symbol != "/-" || aprs.Comment != "CQOps test" {
		t.Errorf("APRS values not copied: %+v", aprs)
	}

	// Nothing must be persisted yet — duplication is a prefill.
	if len(a.Config.Logbooks) != 2 {
		t.Errorf("config logbooks = %d, want 2 (nothing persisted before save)", len(a.Config.Logbooks))
	}
}

func TestLogbookChooserDuplicateKeyOpensPrefilledForm(t *testing.T) {
	a := newChooserTestApp(t)
	richSourceLogbook(a.Config.Logbooks)

	c := NewLogbookChooser(a, NewToastQueue())
	c = sendKey(c, tea.KeyPressMsg{Code: 'd', Text: "d"})

	if c.mode != chooserCreate {
		t.Fatalf("mode = %v, want chooserCreate after d", c.mode)
	}
	if got := c.station.Name.Value(); got != "Home QTH - COPY" {
		t.Errorf("name = %q, want %q", got, "Home QTH - COPY")
	}

	// Cancelling the form must not create anything.
	c = sendKey(c, keyEsc())
	if c.mode != chooserList {
		t.Errorf("mode = %v, want chooserList after Esc", c.mode)
	}
	if len(a.Config.Logbooks) != 2 {
		t.Errorf("config logbooks = %d, want 2 (cancelled duplicate must not persist)", len(a.Config.Logbooks))
	}
	for id := range a.Config.Logbooks {
		if id != "home" && id != "portable" {
			t.Errorf("unexpected logbook %q created by cancelled duplicate", id)
		}
	}
}

func TestLogbookChooserDuplicateSaveCreatesNewEmptyLogbook(t *testing.T) {
	// Isolate DataDir so the new logbook's database is created under a
	// temporary home, never the real user directory.
	t.Setenv("HOME", t.TempDir())

	a := newChooserTestApp(t)
	richSourceLogbook(a.Config.Logbooks)
	home := a.Config.Logbooks["home"]
	a.Logbook = &home
	t.Cleanup(func() { a.StopAPRSTimer() })
	t.Cleanup(func() { a.DB.Close() })

	c := NewLogbookChooser(a, NewToastQueue())
	c.duplicateSelected()
	c.saveForm()

	if len(a.Config.Logbooks) != 3 {
		t.Fatalf("config logbooks = %d, want 3 after saved duplicate", len(a.Config.Logbooks))
	}
	var newID string
	for id, lb := range a.Config.Logbooks {
		if id != "home" && id != "portable" {
			newID = id
			if lb.Name != "Home QTH - COPY" {
				t.Errorf("duplicate name = %q, want %q", lb.Name, "Home QTH - COPY")
			}
			if lb.Station.Callsign != "SP9MOA" || lb.Station.Grid != "JO90" || !lb.Station.GPSGrid {
				t.Errorf("duplicate station not copied: %+v", lb.Station)
			}
			if lb.Wavelog == nil || !lb.Wavelog.Enabled || lb.Wavelog.APIKey != "secret-key" ||
				lb.Wavelog.StationProfileID != "42" || lb.Wavelog.LastFetchedID != 0 {
				t.Errorf("duplicate wavelog not copied correctly: %+v", lb.Wavelog)
			}
			if lb.APRS == nil || !lb.APRS.Enabled || lb.APRS.Callsign != "SP9MOA-7" {
				t.Errorf("duplicate APRS not copied: %+v", lb.APRS)
			}
		}
	}
	if newID == "" {
		t.Fatal("no new logbook found after save")
	}
	if a.LogbookName != newID || a.Config.State.ActiveLogbook != newID {
		t.Errorf("active logbook = %q/%q, want the new duplicate %q",
			a.Config.State.ActiveLogbook, a.LogbookName, newID)
	}

	// The new database starts empty — the QSO table is NOT duplicated.
	qsos, err := store.ListQSOs(a.DB, 5, "")
	if err != nil {
		t.Fatalf("ListQSOs on new db: %v", err)
	}
	if len(qsos) != 0 {
		t.Errorf("duplicate logbook has %d QSOs, want 0", len(qsos))
	}

	// The source logbook stays untouched.
	if a.Config.Logbooks["home"].Name != "Home QTH" {
		t.Errorf("source logbook name changed: %q", a.Config.Logbooks["home"].Name)
	}
}

// TestChooserOverlayAdvertisesDuplicate pins the D entry in the ? overlay
// for the logbook chooser list mode.
func TestChooserOverlayAdvertisesDuplicate(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.screen = screenChooser
	m.ui.chooser = NewLogbookChooser(m.App, m.toasts)
	m.ui.chooser.mode = chooserList

	found := false
	for _, b := range m.ActiveBindings() {
		if b.Help().Key == "D" && b.Help().Desc == "Duplicate" {
			found = true
		}
	}
	if !found {
		t.Error("D Duplicate binding missing from the chooser overlay")
	}
}
