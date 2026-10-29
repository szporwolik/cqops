package dashboard

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// todayStep is what the harness reports after one replayed event: the contacts
// the dashboard kept and the window title.
type todayStep struct {
	Label string   `json:"label"`
	IDs   []string `json:"ids"`
	Title string   `json:"title"`
}

// todayResult mirrors testdata/qso_handler_harness.js for one event sequence.
type todayResult struct {
	Scenario string      `json:"scenario"`
	Steps    []todayStep `json:"steps"`
	Markers  int         `json:"markers"`
}

// TestTodayPanelAcceptsAuthoritativeLists drives the real `today` and `logbook`
// SSE handlers and checks that the contact list each event carries replaces the
// previous one — including when it is shorter because contacts were deleted, or
// empty because the logbook was emptied.
//
// The handler used to drop any shorter list for the same logbook, which kept
// deleted contacts on the map, in Top QSOs and in the stats' band/mode counts
// until the page was reloaded. Nothing needed that guard: every push is built
// on the TUI loop and delivered over one SSE stream, so an older answer cannot
// arrive after a newer one.
func TestTodayPanelAcceptsAuthoritativeLists(t *testing.T) {
	node := requireNode(t)
	appPath, harnessPath := stageDashboardScript(t)

	cases := []struct {
		scenario string
		wantIDs  []string // contacts after each step, comma separated
		wantMark int      // contact markers drawn by the last replayed events
		wantTtl  string
	}{
		// A deletion is simply a shorter list; it must replace the old one.
		{scenario: "deletion", wantIDs: []string{"3,2,1", "3,1"}, wantMark: 2},
		// An emptied logbook sends an empty list, which also has to replace.
		{scenario: "emptied", wantIDs: []string{"3,2,1", ""}, wantMark: 0},
		// Rapid logging: a newer, longer list wins, and a repeated push of the
		// same list still repaints instead of being ignored.
		{scenario: "rapid-refresh", wantIDs: []string{"3,2,1", "4,3,2,1", "4,3,2,1"}, wantMark: 4},
		// A switch clears the previous logbook's contacts immediately, and the
		// new logbook's list (empty, then one contact) is accepted as it comes.
		{scenario: "logbook-switch", wantIDs: []string{"3,2,1", "", "", "9"}, wantMark: 1, wantTtl: "CQOps - SP9XYZ"},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			out, err := exec.Command(node, harnessPath, appPath, "today:"+tc.scenario).CombinedOutput()
			if err != nil {
				t.Fatalf("node %s: %v\n%s", harnessPath, err, out)
			}
			var got todayResult
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("parse harness output: %v\n%s", err, out)
			}
			if got.Scenario != tc.scenario {
				t.Fatalf("harness answered scenario %q, want %q", got.Scenario, tc.scenario)
			}
			if len(got.Steps) != len(tc.wantIDs) {
				t.Fatalf("the harness reported %d steps, want %d: %s", len(got.Steps), len(tc.wantIDs), out)
			}
			for i, want := range tc.wantIDs {
				if ids := strings.Join(got.Steps[i].IDs, ","); ids != want {
					t.Errorf("after %q the dashboard shows [%s], want [%s]", got.Steps[i].Label, ids, want)
				}
			}
			if got.Markers != tc.wantMark {
				t.Errorf("the map drew %d contact markers for the last list, want %d", got.Markers, tc.wantMark)
			}
			if tc.wantTtl != "" && got.Steps[len(got.Steps)-1].Title != tc.wantTtl {
				t.Errorf("window title = %q, want %q", got.Steps[len(got.Steps)-1].Title, tc.wantTtl)
			}
		})
	}
}
