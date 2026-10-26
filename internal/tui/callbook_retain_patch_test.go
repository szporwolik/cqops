package tui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/store"
)

// Retain form (Ctrl+H) keeps the logged callsign on screen, so a late callbook
// answer both completes the stored row AND fills the form. The PATCH command
// used to be dropped by that fall-through even though the contact had already
// been registered with the sync coordinator: nothing ever pushed it, the row
// stayed blocked for the session, and the database lease was never released.
func TestLateEnrichmentKeepsPatchWithRetainedForm(t *testing.T) {
	var patched string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v2/qso/") {
			body, _ := io.ReadAll(r.Body)
			patched = string(body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":77},"meta":{}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test"
	m.retainForm = true

	fillMinimalValidQSO(m)
	m.fields[fieldFreq].SetValue("7.155")
	session := m.lookup.formGen
	m.saveQSO()
	saved := latestQSO(t, m)
	if err := store.SetWavelogID(m.App.DB, saved.ID, 77); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}

	// Retain: the form still shows the callsign that was just logged.
	m.clearForm()
	if got := strings.TrimSpace(m.fields[fieldCall].Value()); got != saved.Call {
		t.Fatalf("form call = %q, want the retained %q", got, saved.Call)
	}

	enriched, syncCmd := m.fillCallbookData(callbookResultMsg{
		Call:    saved.Call,
		Session: session,
		Data:    &callbook.Result{Callsign: saved.Call, Name: "Late operator", Grid: "JN46kc"},
	})
	if !enriched {
		t.Fatal("late result did not reach the saved contact")
	}
	if syncCmd == nil {
		t.Fatal("PATCH command dropped on the retained-form path")
	}
	// The retained form is filled too — that is why the path falls through.
	if got := m.fields[fieldName].Value(); got != "Late operator" {
		t.Errorf("form Name = %q, want the late data", got)
	}

	key := contactSyncKey{logbook: m.App.LogbookName, localID: saved.ID, url: srv.URL}
	if m.sync.inFlight[key] == nil {
		t.Fatal("contact not registered with the sync coordinator")
	}

	// Run the command and feed the completion back the way the update loop
	// does — it must release the serialization slot.
	msg := syncCmd()
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("completion = %T, want editorMsg", msg)
	}
	if em.wlSyncErr != "" {
		t.Fatalf("PATCH failed: %s", em.wlSyncErr)
	}
	m.handleQSOSyncCompletion(em)
	if m.sync.inFlight[key] != nil {
		t.Error("completion left the contact in flight — later edits and Alt+P would skip it")
	}

	if patched == "" {
		t.Fatal("no PATCH reached the server")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(patched), &payload); err != nil {
		t.Fatalf("decode PATCH body: %v", err)
	}
	if payload["name"] != "Late operator" {
		t.Errorf("PATCH name = %v, want the enriched value", payload["name"])
	}
	if payload["gridsquare"] != "JN46kc" {
		t.Errorf("PATCH gridsquare = %v, want JN46kc", payload["gridsquare"])
	}

	// The acknowledgement cleared the pending flag for the pushed revision.
	dirty, err := store.QSOHasPendingSync(m.App.DB, saved.ID)
	if err != nil {
		t.Fatalf("QSOHasPendingSync: %v", err)
	}
	if dirty {
		t.Error("contact still pending after a successful PATCH")
	}
}
