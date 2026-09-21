package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// TestEnsureWavelogRadio verifies the CQOps radio is resolved (found or
// created) and its id stored in the lookup state.
func TestEnsureWavelogRadio(t *testing.T) {
	postCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		case http.MethodPost:
			postCount++
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": 9, "radio": "CQOps", "frequency": 0, "mode": nil},
			})
		}
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.Enabled = true
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"

	msg := execCmd(m.ensureWavelogRadioCmd())
	em, ok := msg.(wlRadioEnsureMsg)
	if !ok || em.radio == nil {
		t.Fatalf("expected wlRadioEnsureMsg with radio, got %T %+v", msg, em)
	}
	if !m.handleWavelogRadioMsg(msg) {
		t.Fatal("handleWavelogRadioMsg should consume the message")
	}
	if m.lookup.wlRadioID != 9 {
		t.Errorf("wlRadioID = %d, want 9", m.lookup.wlRadioID)
	}
	if postCount != 1 {
		t.Errorf("POST create count = %d, want 1", postCount)
	}
}

// TestRadioPushRecreatesOnNotFound verifies the push recreates the radio
// when the server reports not_found and retries once.
func TestRadioPushRecreatesOnNotFound(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		case http.MethodPost:
			posts++
			if posts == 1 {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{"code": "not_found", "message": "Radio not found"},
				})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": 12, "radio": "CQOps", "frequency": 14250000, "mode": "SSB"},
			})
		}
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.Enabled = true
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	m.lookup.wlRadioID = 0

	m.fields[fieldFreq].SetValue("14.250")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldTXPower].SetValue("25")

	msg := execCmd(m.pushWavelogRadioCmd())
	pm, ok := msg.(wlRadioPushMsg)
	if !ok {
		t.Fatalf("expected wlRadioPushMsg, got %T", msg)
	}
	if pm.err != nil {
		t.Fatalf("push after recreate failed: %v", pm.err)
	}
	if !pm.retried {
		t.Error("expected retried=true after recreate")
	}
	if !m.handleWavelogRadioMsg(msg) {
		t.Fatal("handleWavelogRadioMsg should consume the message")
	}
	if m.lookup.wlRadioID != 12 {
		t.Errorf("wlRadioID = %d, want 12 (recreated)", m.lookup.wlRadioID)
	}
	if posts != 3 {
		t.Errorf("POST count = %d, want 3 (failed push + ensure create + retry push)", posts)
	}
}

// TestRadioStateFromForm verifies the form fields map to the radio state.
func TestRadioStateFromForm(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.fields[fieldFreq].SetValue("14.250")
	m.fields[fieldFreqRx].SetValue("14.255")
	m.fields[fieldMode].SetValue("USB")
	m.fields[fieldTXPower].SetValue("50")

	st, ok := m.radioStateFromForm()
	if !ok {
		t.Fatal("radioStateFromForm should report ok")
	}
	if st.FrequencyHz != 14250000 {
		t.Errorf("FrequencyHz = %d, want 14250000", st.FrequencyHz)
	}
	if st.FrequencyRxHz != 14255000 {
		t.Errorf("FrequencyRxHz = %d, want 14255000", st.FrequencyRxHz)
	}
	if st.Mode != "SSB" {
		t.Errorf("Mode = %q, want SSB (USB normalised)", st.Mode)
	}
	if st.PowerWatts != 50 {
		t.Errorf("PowerWatts = %d, want 50", st.PowerWatts)
	}

	// Without frequency/mode there is nothing to push.
	m2 := newLifecycleTestModel(t)
	if _, ok := m2.radioStateFromForm(); ok {
		t.Error("empty form should report !ok")
	}
}

// TestTickDispatchesRadioPush verifies the 15-second cadence dispatches a
// radio push from the tick handler.
func TestTickDispatchesRadioPush(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": 9, "radio": "CQOps", "frequency": 14250000, "mode": "SSB"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.Enabled = true
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	m.lookup.wlOnline = true
	m.lookup.wlRadioID = 9
	m.lookup.lastRadioPush = time.Now().Add(-time.Minute)

	m.fields[fieldFreq].SetValue("14.250")
	m.fields[fieldMode].SetValue("SSB")

	cmd := m.handleTick(nil)
	if cmd == nil {
		t.Fatal("tick must dispatch the radio push")
	}
	msg := execCmd(cmd)
	switch mm := msg.(type) {
	case wlRadioPushMsg:
		if mm.err != nil {
			t.Fatalf("push failed: %v", mm.err)
		}
	case tea.BatchMsg:
		found := false
		for _, sub := range mm {
			if pm, ok := sub().(wlRadioPushMsg); ok {
				found = true
				if pm.err != nil {
					t.Fatalf("push failed: %v", pm.err)
				}
			}
		}
		if !found {
			t.Fatalf("no radio push in tick batch: %T", msg)
		}
	default:
		t.Fatalf("unexpected tick cmd result: %T", msg)
	}

	// A second tick inside the 15 s window must not push again (other
	// periodic commands may still be returned).
	cmd2 := m.handleTick(nil)
	if cmd2 != nil {
		switch mm := execCmd(cmd2).(type) {
		case wlRadioPushMsg:
			t.Error("second tick within the interval dispatched a radio push")
		case tea.BatchMsg:
			for _, sub := range mm {
				if _, ok := sub().(wlRadioPushMsg); ok {
					t.Error("second tick within the interval dispatched a radio push")
				}
			}
		}
	}
}

// TestWlStatusDispatchesRadioEnsure verifies an online Wavelog status with no
// radio id yet dispatches the ensure command.
func TestWlStatusDispatchesRadioEnsure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 5, "radio": "CQOps", "frequency": 7100000, "mode": "CW"}},
		})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.Enabled = true
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	m.lookup.wlRadioID = 0

	handled, cmd := m.handleAsyncMessages(wlStatusMsg{online: true, stationName: "X"})
	if !handled {
		t.Fatal("wlStatusMsg should be handled")
	}
	if cmd == nil {
		t.Fatal("online status with no radio must dispatch the ensure command")
	}
	msg := execCmd(cmd)
	if em, ok := msg.(wlRadioEnsureMsg); !ok || em.radio == nil {
		t.Fatalf("expected wlRadioEnsureMsg, got %T", msg)
	}
	if !m.handleWavelogRadioMsg(msg) {
		t.Fatal("handleWavelogRadioMsg should consume the message")
	}
	if m.lookup.wlRadioID != 5 {
		t.Errorf("wlRadioID = %d, want 5", m.lookup.wlRadioID)
	}
}
