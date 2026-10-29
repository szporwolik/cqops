package dashboard

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stationCardResult mirrors what testdata/qso_handler_harness.js prints for one
// event order.
type stationCardResult struct {
	Scenario      string `json:"scenario"`
	HTML          string `json:"html"`
	RigConnected  bool   `json:"rigConnected"`
	RigDisconnect bool   `json:"rigDisconnected"`
	WSJTXOnline   bool   `json:"wsjtxOnline"`
	WSJTXOffline  bool   `json:"wsjtxOffline"`
	HasOperator   bool   `json:"hasOperator"`
	HasLogbook    bool   `json:"hasLogbook"`
	HasFrequency  bool   `json:"hasFrequency"`
}

// TestStationCardKeepsItsRowsAcrossEvents drives the real `station`, `rig`,
// `wsjtx`, `operator` and `logbook` SSE handlers in the orders the dashboard
// server produces and checks the station card still reports the live rig.
//
// A `station` event carries only the station payload. Rendering the whole card
// from that event alone left the card saying "○ Disconnected", "○ Offline",
// "Logbook —" and "Frequency —" for as long as the page lived: a station push
// arrives every couple of seconds and wiped the rows the rig/wsjtx/logbook
// events had just filled, and those events only fire on change, so nothing ever
// put them back.
func TestStationCardKeepsItsRowsAcrossEvents(t *testing.T) {
	node := requireNode(t)
	appPath, harnessPath := stageDashboardScript(t)

	for _, scenario := range []string{"events-first", "station-first"} {
		t.Run(scenario, func(t *testing.T) {
			out, err := exec.Command(node, harnessPath, appPath, "station-card:"+scenario).CombinedOutput()
			if err != nil {
				t.Fatalf("node %s: %v\n%s", harnessPath, err, out)
			}
			var got stationCardResult
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("parse harness output: %v\n%s", err, out)
			}
			if got.Scenario != scenario {
				t.Fatalf("harness answered scenario %q, want %q", got.Scenario, scenario)
			}
			if got.RigDisconnect {
				t.Errorf("the station card says the rig is disconnected: %s", got.HTML)
			}
			if !got.RigConnected {
				t.Errorf("the station card never says the rig is connected: %s", got.HTML)
			}
			if got.WSJTXOffline || !got.WSJTXOnline {
				t.Errorf("the station card does not report WSJT-X online: %s", got.HTML)
			}
			if !got.HasLogbook {
				t.Errorf("the station card lost the logbook name: %s", got.HTML)
			}
			if !got.HasFrequency {
				t.Errorf("the station card lost the rig frequency: %s", got.HTML)
			}
			if !got.HasOperator {
				t.Errorf("the station card lost the operator callsign: %s", got.HTML)
			}
		})
	}
}

// requireNode skips the test when no JavaScript engine is available.
func requireNode(t *testing.T) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; the dashboard handler tests need a JS engine")
	}
	return node
}

// stageDashboardScript writes app.js with the test hook injected and the
// browser-less harness into a temp dir, and returns the two paths.
func stageDashboardScript(t *testing.T) (appPath, harnessPath string) {
	t.Helper()
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

	harness, err := os.ReadFile("testdata/qso_handler_harness.js")
	if err != nil {
		t.Fatalf("read harness: %v", err)
	}

	dir := t.TempDir()
	appPath = filepath.Join(dir, "app.js")
	if err := os.WriteFile(appPath, []byte(src), 0o600); err != nil {
		t.Fatalf("write app.js: %v", err)
	}
	harnessPath = filepath.Join(dir, "qso_handler_harness.js")
	if err := os.WriteFile(harnessPath, harness, 0o600); err != nil {
		t.Fatalf("write harness: %v", err)
	}
	return appPath, harnessPath
}
