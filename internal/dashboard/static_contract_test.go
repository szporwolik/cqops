package dashboard

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestAppJSReadsOnlyFieldsThePayloadHas pins the contract between the SSE
// payload and the console log line the operator actually looks at.
//
// Reported from a live browser console: every stats event logged
// "rate: undefined", because app.js read `s.ratePerHour` — a field the payload
// never had (Stats carries rate5m/rate15m/rate60m). A log line that names a
// field which does not exist is worse than no log line: it suggests the panel
// is broken when the numbers are right there.
func TestAppJSReadsOnlyFieldsThePayloadHas(t *testing.T) {
	js, err := staticFS.ReadFile("static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	app := string(js)

	if strings.Contains(app, "ratePerHour") {
		t.Error("app.js reads stats.ratePerHour, which the payload does not carry")
	}

	// The stats log line must name the fields the payload does carry.
	raw, err := json.Marshal(Stats{})
	if err != nil {
		t.Fatalf("marshal Stats: %v", err)
	}
	payload := string(raw)
	for _, field := range []string{"rate5m", "rate15m", "rate60m", "qsosToday", "uniqueCalls"} {
		if !strings.Contains(payload, `"`+field+`"`) {
			t.Errorf("the Stats payload has no %q field — app.js cannot log it", field)
		}
		if !strings.Contains(app, field) {
			t.Errorf("app.js never mentions %q, which the payload carries", field)
		}
	}

	// And the line that started this: it must be the one logging the real names.
	line := ""
	for _, l := range strings.Split(app, "\n") {
		if strings.Contains(l, "D('sse','stats'") {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatal("the stats SSE handler no longer logs anything")
	}
	for _, field := range []string{"rate5m", "rate15m", "rate60m"} {
		if !strings.Contains(line, field) {
			t.Errorf("the stats log line does not report %s: %s", field, strings.TrimSpace(line))
		}
	}
}
