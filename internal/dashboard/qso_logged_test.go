package dashboard

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testHookName is injected into app.js by the test to reach the closure's
// state. The dashboard script is one IIFE with nothing global by design, so the
// alternative to a hook would be a browser.
const testHook = `
window.__cqopsTest={
  qsoLogged:handleQsoLogged,
  activeGrid:function(){return activeGrid||''},
  today:function(){return todayQsos},
  setToday:function(t){todayQsos=t.slice()},
  setStation:function(s){ownStationLat=s.lat;ownStationLon=s.lon},
  setActive:function(a){switchToActive();drawActiveLine(a.grid)},
  initMap:function(cfg){initMap(cfg)},
  reset:function(){todayQsos=[];activeGrid=null;lastActiveFlags={}}
};
`

// qsoLoggedResult mirrors the JSON the harness prints for one scenario.
type qsoLoggedResult struct {
	TodayBefore      int      `json:"todayBefore"`
	BeforeActiveGrid string   `json:"beforeActiveGrid"`
	BeforeState      string   `json:"beforeState"`
	Markers          int      `json:"markers"`
	LayerGroupsClear int      `json:"layerGroupsCleared"`
	TodayAfter       int      `json:"todayAfter"`
	Calls            []string `json:"calls"`
	ActiveGridAfter  string   `json:"activeGridAfter"`
	StateAfter       string   `json:"stateAfter"`
}

// TestQsoLoggedHandlerRuns executes the real qso_logged handler with a stubbed
// DOM and Leaflet instead of matching source text, so a redraw hidden inside a
// called function — or a missed transition out of the active-contact layout —
// fails the test.
//
// One save must: leave the overview layout, clear activeGrid before drawing so
// the active path is not rendered, rebuild the map exactly once (one marker per
// today contact plus the station, and no partner dot), and put the new contact
// at the top of today's list.
func TestQsoLoggedHandlerRuns(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; the dashboard handler test needs a JS engine")
	}
	app, err := staticFS.ReadFile("static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	src := string(app)
	idx := strings.LastIndex(src, "})();")
	if idx < 0 {
		t.Fatal("app.js does not end with the IIFE close, the test hook cannot be injected")
	}
	src = src[:idx] + testHook + src[idx:]

	dir := t.TempDir()
	appPath := filepath.Join(dir, "app.js")
	if err := os.WriteFile(appPath, []byte(src), 0o600); err != nil {
		t.Fatalf("write app.js: %v", err)
	}
	harness, err := os.ReadFile("testdata/qso_handler_harness.js")
	if err != nil {
		t.Fatalf("read harness: %v", err)
	}
	harnessPath := filepath.Join(dir, "qso_handler_harness.js")
	if err := os.WriteFile(harnessPath, harness, 0o600); err != nil {
		t.Fatalf("write harness: %v", err)
	}

	out, err := exec.Command(node, harnessPath, appPath).CombinedOutput()
	if err != nil {
		t.Fatalf("node %s: %v\n%s", harnessPath, err, out)
	}
	var got map[string]qsoLoggedResult
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("parse harness output: %v\n%s", err, out)
	}

	cases := []struct {
		name         string
		expectActive bool // the scenario starts in the active-contact layout
	}{
		{"emptyLog", false},
		{"withContacts", false},
		{"fromActiveContact", true},
	}
	for _, tc := range cases {
		res, ok := got[tc.name]
		if !ok {
			t.Fatalf("%s: missing from the harness output: %s", tc.name, out)
		}
		t.Run(tc.name, func(t *testing.T) {
			if res.TodayAfter != res.TodayBefore+1 {
				t.Errorf("today holds %d contacts after the save, want %d", res.TodayAfter, res.TodayBefore+1)
			}
			if len(res.Calls) == 0 || res.Calls[0] != "DL1AA" {
				t.Errorf("the saved contact is not at the top of today's list: %v", res.Calls)
			}
			if res.ActiveGridAfter != "" {
				t.Errorf("activeGrid = %q after the save, want it cleared", res.ActiveGridAfter)
			}
			if !strings.Contains(res.StateAfter, "mode-overview") || strings.Contains(res.StateAfter, "mode-active") {
				t.Errorf("view state after the save = %q, want the overview layout", res.StateAfter)
			}
			// One rebuild draws the station plus one marker per today contact.
			// Anything more means a second rebuild (the bug this guards), or a
			// partner dot — which would mean the active path was still drawn.
			want := 1 + res.TodayAfter
			if res.Markers != want {
				t.Errorf("the save created %d markers, want %d — the map was rebuilt more than once, or the active path was still on it",
					res.Markers, want)
			}
			if res.LayerGroupsClear == 0 {
				t.Error("the map layers were never cleared, so the rebuild did not happen")
			}
			if tc.expectActive {
				if res.BeforeActiveGrid != "KO73dg" {
					t.Fatalf("setup: activeGrid before the save = %q, want KO73dg", res.BeforeActiveGrid)
				}
				if !strings.Contains(res.BeforeState, "mode-active") {
					t.Fatalf("setup: view state before the save = %q, want the active layout", res.BeforeState)
				}
			}
		})
	}
}
