package dashboard

import (
	"encoding/json"
	"os/exec"
	"testing"
)

// ratesStep is what the harness reports after one replayed event: the QSO-rate
// row (5m, 15m, 1h) as rendered into the stats panel.
type ratesStep struct {
	Label string   `json:"label"`
	Rates []string `json:"rates"`
}

type ratesResult struct {
	Scenario string      `json:"scenario"`
	Steps    []ratesStep `json:"steps"`
}

// TestQsoRateSurvivesContactRedraws drives the real `stats`, `today` and
// `qso_logged` handlers and checks the QSO-rate row of the stats panel.
//
// Every handler that repaints the panel from the contact list passed a null
// statistics object, because a `today` event carries contacts only. renderStats
// read the rate from that null object and wrote `5m 0 / 15m 0 / 1h 0`, so the
// panel dropped the real numbers on every logged contact and stayed at zero
// until the next minute tick — the operator saw a rate panel that never
// matched the server, which was publishing 5m 2 / 15m 8 / 1h 17 all along.
// The rate can only come from the server (the windows are UTC), so the last
// payload has to be remembered; only a logbook switch may clear it.
func TestQsoRateSurvivesContactRedraws(t *testing.T) {
	node := requireNode(t)
	appPath, harnessPath := stageDashboardScript(t)

	cases := []struct {
		scenario string
		want     [][]string // one rate triple (5m, 15m, 1h) per step
	}{
		{
			// The server's numbers survive a new contact and a logged contact,
			// and the next stats event replaces them with the newer windows.
			scenario: "survives-redraws",
			want: [][]string{
				{"2", "8", "17"},
				{"2", "8", "17"},
				{"2", "8", "17"},
				{"3", "9", "18"},
			},
		},
		{
			// The rates describe a logbook: switching must clear them instead of
			// showing the previous logbook's activity.
			scenario: "switch-resets",
			want: [][]string{
				{"2", "8", "17"},
				{"0", "0", "0"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			out, err := exec.Command(node, harnessPath, appPath, "rates:"+tc.scenario).CombinedOutput()
			if err != nil {
				t.Fatalf("node %s: %v\n%s", harnessPath, err, out)
			}
			var got ratesResult
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("parse harness output: %v\n%s", err, out)
			}
			if got.Scenario != tc.scenario {
				t.Fatalf("harness answered scenario %q, want %q", got.Scenario, tc.scenario)
			}
			if len(got.Steps) != len(tc.want) {
				t.Fatalf("the harness reported %d steps, want %d: %s", len(got.Steps), len(tc.want), out)
			}
			for i, want := range tc.want {
				step := got.Steps[i]
				if len(step.Rates) != 3 {
					t.Fatalf("after %q the rate row is %v, want three numbers", step.Label, step.Rates)
				}
				for j := range want {
					if step.Rates[j] != want[j] {
						t.Errorf("after %q the rate row is %v, want %v — the panel lost the server's rates",
							step.Label, step.Rates, want)
						break
					}
				}
			}
		})
	}
}
