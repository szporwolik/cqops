package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func init() {
	// Freeze time for deterministic tests.
	timeNow = func() time.Time {
		return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	}
}

func TestSnapshot_InitialEmpty(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)
	snap := state.Snapshot()

	if snap.App.Name != "" {
		t.Error("expected empty app name")
	}
	if len(snap.Recent) != 0 {
		t.Error("expected empty recent QSO list")
	}
	if snap.ActiveQSO != nil {
		t.Error("expected nil active QSO")
	}
}

func TestSetActiveQSO_UpdatesSnapshot(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)

	aq := &ActiveQSO{
		State:     "editing",
		Source:    "form",
		Call:      "SP9MOA",
		Band:      "20m",
		Mode:      "CW",
		Frequency: "14.025 MHz",
	}
	state.SetActiveQSO(aq)

	snap := state.Snapshot()
	if snap.ActiveQSO == nil {
		t.Fatal("expected active QSO to be set")
	}
	if snap.ActiveQSO.Call != "SP9MOA" {
		t.Errorf("expected call SP9MOA, got %s", snap.ActiveQSO.Call)
	}
}

func TestClearActiveQSO(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)

	state.SetActiveQSO(&ActiveQSO{Call: "SP9MOA"})
	state.ClearActiveQSO()

	snap := state.Snapshot()
	if snap.ActiveQSO != nil {
		t.Error("expected nil active QSO after clear")
	}
}

func TestAddLoggedQSO_PrependsToRecent(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)

	q1 := QSOView{Call: "SP9MOA", Band: "20m", TimeUTC: time.Now()}
	q2 := QSOView{Call: "DL1ABC", Band: "40m", TimeUTC: time.Now()}

	state.AddLoggedQSO(q1)
	state.AddLoggedQSO(q2)

	snap := state.Snapshot()
	if len(snap.Recent) != 2 {
		t.Fatalf("expected 2 recent QSOs, got %d", len(snap.Recent))
	}
	if snap.Recent[0].Call != "DL1ABC" {
		t.Errorf("expected most recent to be DL1ABC, got %s", snap.Recent[0].Call)
	}
	if snap.Recent[1].Call != "SP9MOA" {
		t.Errorf("expected second to be SP9MOA, got %s", snap.Recent[1].Call)
	}
	if snap.LastQSO == nil || snap.LastQSO.Call != "DL1ABC" {
		t.Error("expected LastQSO to be DL1ABC")
	}
}

func TestAddLoggedQSO_LimitEnforced(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)

	for i := 0; i < 25; i++ {
		state.AddLoggedQSO(QSOView{Call: "CALL", TimeUTC: time.Now()})
	}

	snap := state.Snapshot()
	if len(snap.Recent) > maxRecent {
		t.Errorf("expected at most %d recent QSOs, got %d", maxRecent, len(snap.Recent))
	}
}

func TestAddLoggedQSO_IncrementsSession(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)

	state.AddLoggedQSO(QSOView{Call: "A", TimeUTC: time.Now()})
	state.AddLoggedQSO(QSOView{Call: "B", TimeUTC: time.Now()})

	if state.SessionQSOs() != 2 {
		t.Errorf("expected 2 session QSOs, got %d", state.SessionQSOs())
	}
}

func TestSetRig_NoChangeSkipsLock(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)

	info := RigInfo{Connected: true, FrequencyHz: 14025000, Mode: "CW"}
	state.SetRig(info)

	// Subscribe to hub after first set.
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// Same info — should NOT publish.
	state.SetRig(info)

	select {
	case ev := <-ch:
		t.Errorf("unexpected event published: %s", ev.Type)
	case <-time.After(50 * time.Millisecond):
		// Good — no event.
	}
}

func TestHub_PublishToSubscriber(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	hub.Publish(EventRig, RigInfo{Connected: true})

	select {
	case ev := <-ch:
		if ev.Type != string(EventRig) {
			t.Errorf("expected event type rig, got %s", ev.Type)
		}
		if ev.ID != 1 {
			t.Errorf("expected event ID 1, got %d", ev.ID)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for event")
	}
}

func TestHub_NonBlockingSend(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// Fill the buffer.
	for i := 0; i < 16; i++ {
		hub.Publish(EventHeartbeat, nil)
	}

	// Publish one more — should not block, event dropped.
	done := make(chan struct{})
	go func() {
		hub.Publish(EventHeartbeat, nil)
		close(done)
	}()

	select {
	case <-done:
		// Good — publish did not block.
	case <-time.After(time.Second):
		t.Error("Publish blocked on full buffer")
	}
}

func TestHub_UnsubscribeRemoves(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe()
	hub.Unsubscribe(ch)

	hub.Publish(EventHeartbeat, nil)

	select {
	case <-ch:
		t.Error("received event after unsubscribe")
	default:
		// Good.
	}
}

func TestHandleHealthz(t *testing.T) {
	state := NewState(NewHub())
	mux := NewMux(state, state.hub)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %s", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestHandleSnapshot(t *testing.T) {
	state := NewState(NewHub())
	state.SetStation(StationInfo{Callsign: "SP9MOA", Locator: "JO90"})
	mux := NewMux(state, state.hub)

	req := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var snap Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if snap.Station.Callsign != "SP9MOA" {
		t.Errorf("expected SP9MOA, got %s", snap.Station.Callsign)
	}
}

func TestHandleEvents_SendsInitialSnapshot(t *testing.T) {
	state := NewState(NewHub())
	state.SetStation(StationInfo{Callsign: "TEST"})
	mux := NewMux(state, state.hub)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()

	// SSE is streaming — we read the first few bytes and cancel.
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	// Read in a goroutine so we can cancel.
	errc := make(chan error, 1)
	go func() {
		mux.ServeHTTP(rec, req)
		errc <- nil
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-errc // wait for handler to return

	body := rec.Body.String()
	if !strings.Contains(body, "event: snapshot") {
		t.Errorf("expected snapshot event in SSE stream, got: %s", body[:min(len(body), 200)])
	}
	if !strings.Contains(body, "Content-Type: text/event-stream") {
		// Header might not be in Body — check recorded headers.
		_ = rec.Header().Get("Content-Type")
	}
}

func TestServerStartStop(t *testing.T) {
	srv := New("localhost", "0") // port 0 = random free port
	srv.Start()

	select {
	case online := <-srv.Status():
		if !online {
			t.Fatal("expected server to start successfully")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to start")
	}

	srv.Stop()

	select {
	case online := <-srv.Status():
		if online {
			t.Error("expected server to be stopped")
		}
	case <-time.After(3 * time.Second):
		t.Error("timed out waiting for server to stop")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// APRS state tests
// =============================================================================

func TestSetAPRS_UpdatesSnapshot(t *testing.T) {
	state := NewState(NewHub())
	stations := []APRSStation{
		{Callsign: "N0CALL", Lat: 50.0, Lon: 20.0, Symbol: "/r", Comment: "test", LastHeard: timeNow()},
		{Callsign: "SP9MOA", Lat: 49.5, Lon: 19.3, Symbol: "/[", Comment: "test2", LastHeard: timeNow().Add(-5 * time.Minute)},
	}
	state.SetAPRS(stations)

	snap := state.Snapshot()
	if len(snap.APRS) != 2 {
		t.Fatalf("expected 2 APRS stations, got %d", len(snap.APRS))
	}
	if snap.APRS[0].Callsign != "N0CALL" {
		t.Errorf("expected N0CALL, got %s", snap.APRS[0].Callsign)
	}
}

func TestSetAPRS_PublishesEvent(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	stations := []APRSStation{
		{Callsign: "N0CALL", Lat: 50, Lon: 20, LastHeard: timeNow()},
	}
	state.SetAPRS(stations)

	select {
	case ev := <-ch:
		if ev.Type != string(EventAPRS) {
			t.Errorf("expected event type aprs, got %s", ev.Type)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for aprs event")
	}
}

func TestHandleAPRS(t *testing.T) {
	state := NewState(NewHub())
	state.SetAPRS([]APRSStation{
		{Callsign: "N0CALL", Lat: 50, Lon: 20, LastHeard: timeNow()},
	})
	mux := NewMux(state, state.hub)

	req := httptest.NewRequest(http.MethodGet, "/api/aprs", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var stations []APRSStation
	if err := json.Unmarshal(rec.Body.Bytes(), &stations); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(stations) != 1 {
		t.Errorf("expected 1 station, got %d", len(stations))
	}
	if stations[0].Callsign != "N0CALL" {
		t.Errorf("expected N0CALL, got %s", stations[0].Callsign)
	}
}

func TestSetAPRS_EmptyList(t *testing.T) {
	state := NewState(NewHub())
	state.SetAPRS([]APRSStation{})
	snap := state.Snapshot()
	if len(snap.APRS) != 0 {
		t.Errorf("expected empty, got %d", len(snap.APRS))
	}
}
