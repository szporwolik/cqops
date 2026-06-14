package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Wavelog HTTP mock server helpers
// =============================================================================

// newWavelogTestServer creates an httptest.Server that mimics the Wavelog API.
// The handler receives test assertions via closures.
func newWavelogTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// wavelogVersionHandler returns a handler for the /api/version endpoint.
func wavelogVersionHandler(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/version" {
			http.NotFound(w, r)
			return
		}
		resp := map[string]string{"status": status, "version": "1.0-test"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// wavelogStationInfoHandler returns a handler for /api/station_info/{key}.
func wavelogStationInfoHandler(stations []map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stations)
	}
}

// wavelogQSOHandler returns a handler for /index.php/api/qso.
func wavelogQSOHandler(status string, messages []string, adifErrors int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		resp := map[string]interface{}{
			"status":      status,
			"adif_count":  1,
			"adif_errors": adifErrors,
			"messages":    messages,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// wavelogPrivateLookupHandler returns a handler for /api/private_lookup.
func wavelogPrivateLookupHandler(data map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		if data == nil {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// =============================================================================
// Wavelog integration tests
// =============================================================================

func TestWavelogUploadDisabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = false

	qs := qso.NewQSO()
	qs.Call = "SP9MOA"
	qs.ID = 1

	cmd := m.maybeUploadToWavelog(qs)
	if cmd != nil {
		t.Error("maybeUploadToWavelog should return nil when Wavelog is disabled")
	}
}

func TestWavelogUploadEnabledNoInternet(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = "http://127.0.0.1:1" // invalid port
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "1"
	m.inetOnline = false

	qs := qso.NewQSO()
	qs.Call = "SP9MOA"
	qs.ID = 1

	cmd := m.maybeUploadToWavelog(qs)
	if cmd != nil {
		t.Error("maybeUploadToWavelog should return nil when offline")
	}
}

func TestWavelogUploadMockSuccess(t *testing.T) {
	srv := newWavelogTestServer(t, wavelogQSOHandler("ok", []string{""}, 0))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	qs := qso.NewQSO()
	qs.Call = "SP9MOA"
	qs.ID = 1

	cmd := m.maybeUploadToWavelog(qs)
	if cmd == nil {
		t.Fatal("maybeUploadToWavelog should return a command")
	}

	msg := cmd()
	result, ok := msg.(wlUploadResultMsg)
	if !ok {
		t.Fatalf("Expected wlUploadResultMsg, got %T", msg)
	}
	if !result.ok {
		t.Error("Upload should succeed with mock server")
	}
}

func TestWavelogUploadMockDuplicate(t *testing.T) {
	srv := newWavelogTestServer(t, wavelogQSOHandler("abort",
		[]string{"", "Duplicate for SP9MOA"}, 1))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	qs := qso.NewQSO()
	qs.Call = "SP9MOA"
	qs.ID = 1

	cmd := m.maybeUploadToWavelog(qs)
	if cmd == nil {
		t.Fatal("maybeUploadToWavelog should return a command")
	}

	msg := cmd()
	result, ok := msg.(wlUploadResultMsg)
	if !ok {
		t.Fatalf("Expected wlUploadResultMsg, got %T", msg)
	}
	// Duplicate uploads should still report ok=true
	if !result.ok {
		t.Error("Duplicate upload should report ok=true")
	}
}

func TestWavelogUploadMockServerError(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	qs := qso.NewQSO()
	qs.Call = "SP9MOA"
	qs.ID = 1

	cmd := m.maybeUploadToWavelog(qs)
	if cmd == nil {
		t.Fatal("maybeUploadToWavelog should return a command even for errors")
	}

	msg := cmd()
	result, ok := msg.(wlUploadResultMsg)
	if !ok {
		t.Fatalf("Expected wlUploadResultMsg, got %T", msg)
	}
	if result.ok {
		t.Error("Upload should report failure on server error")
	}
}

func TestWavelogStatusCheckSuccess(t *testing.T) {
	srv := newWavelogTestServer(t, wavelogVersionHandler("ok"))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.wlOnline = false

	cmd := m.checkWavelogCmd()
	if cmd == nil {
		t.Fatal("checkWavelogCmd should return a command")
	}

	msg := cmd()
	status, ok := msg.(wlStatusMsg)
	if !ok {
		t.Fatalf("Expected wlStatusMsg, got %T", msg)
	}
	if !status.online {
		t.Error("Status should report online with mock server")
	}
}

func TestWavelogStatusCheckFailure(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", 401)
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.wlOnline = true

	cmd := m.checkWavelogCmd()
	if cmd == nil {
		t.Fatal("checkWavelogCmd should return a command")
	}

	msg := cmd()
	status, ok := msg.(wlStatusMsg)
	if !ok {
		t.Fatalf("Expected wlStatusMsg, got %T", msg)
	}
	if status.online {
		t.Error("Status should report offline on auth error")
	}
}

func TestWavelogPrivateLookupSuccess(t *testing.T) {
	data := map[string]interface{}{
		"callsign":              "SP9MOA",
		"name":                  "John",
		"call_worked":           true,
		"call_worked_band":      true,
		"call_worked_band_mode": false,
		"dxcc_confirmed":        true,
		"lotw_member":           true,
	}
	srv := newWavelogTestServer(t, wavelogPrivateLookupHandler(data))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.inetOnline = true
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")

	cmd := m.wlLookup("SP9MOA")
	if cmd == nil {
		t.Fatal("wlLookup should return a command")
	}

	msg := cmd()
	result, ok := msg.(wlResultMsg)
	if !ok {
		t.Fatalf("Expected wlResultMsg, got %T", msg)
	}
	if result.Err != nil {
		t.Errorf("Private lookup error: %v", result.Err)
	}
	if result.Data == nil {
		t.Fatal("Private lookup returned nil data")
	}
	if !result.Data.Worked() {
		t.Error("call_worked should be true")
	}
	if !result.Data.DXCCConfirmed() {
		t.Error("dxcc_confirmed should be true")
	}
	if result.Data.Name() != "John" {
		t.Errorf("Name = %q; want John", result.Data.Name())
	}
}

func TestWavelogPrivateLookupNotFound(t *testing.T) {
	srv := newWavelogTestServer(t, wavelogPrivateLookupHandler(nil)) // nil = 404
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.inetOnline = true
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")

	cmd := m.wlLookup("ZZ0ZZ")
	if cmd == nil {
		t.Fatal("wlLookup should return a command")
	}

	msg := cmd()
	result, ok := msg.(wlResultMsg)
	if !ok {
		t.Fatalf("Expected wlResultMsg, got %T", msg)
	}
	// Error expected for 404
	if result.Err == nil {
		t.Error("Private lookup should return error for 404")
	}
}

func TestWavelogMaybeCheckWavelogDisabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = false

	cmd := m.maybeCheckWavelog()
	if cmd != nil {
		t.Error("maybeCheckWavelog should return nil when Wavelog is disabled")
	}
}

func TestWavelogUploadADIFNoStationProfile(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = "http://example.com"
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "" // no station
	m.inetOnline = true

	cmd := m.uploadADIFToWavelog("<CALL:6>SP9MOA<EOR>", 1, "SP9MOA")
	if cmd != nil {
		t.Error("uploadADIFToWavelog should return nil when station profile is empty")
	}
}

// =============================================================================
// Wavelog station-info / FetchStations mock tests
// =============================================================================

func TestWavelogStatusCheckWithStations(t *testing.T) {
	// Mock version handler first, but we need to also mock station_info
	// The checkWavelogCmd calls TestConnection first, then FetchStations
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/version":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "1.0"})
		case "/api/station_info/test-key":
			json.NewEncoder(w).Encode([]map[string]string{
				{
					"station_id":           "1",
					"station_profile_name": "Home QTH",
					"station_gridsquare":   "JO90",
					"station_callsign":     "SP9MOA",
					"station_active":       "1",
				},
			})
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "1"
	m.wlOnline = false

	cmd := m.checkWavelogCmd()
	if cmd == nil {
		t.Fatal("checkWavelogCmd should return a command")
	}

	msg := cmd()
	status, ok := msg.(wlStatusMsg)
	if !ok {
		t.Fatalf("Expected wlStatusMsg, got %T", msg)
	}
	if !status.online {
		t.Error("Status should report online with mock server + stations")
	}
	if status.stationName != "JO90 / SP9MOA" {
		t.Errorf("stationName = %q; want JO90 / SP9MOA", status.stationName)
	}
	if status.stationLabel != "Home QTH" {
		t.Errorf("stationLabel = %q; want Home QTH", status.stationLabel)
	}
}

func TestWavelogStatusCheckNoStations(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/version":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "1.0"})
		case "/api/station_info/test-key":
			json.NewEncoder(w).Encode([]map[string]string{}) // empty
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "1"
	m.wlOnline = false

	cmd := m.checkWavelogCmd()
	if cmd == nil {
		t.Fatal("checkWavelogCmd should return a command")
	}

	msg := cmd()
	status, ok := msg.(wlStatusMsg)
	if !ok {
		t.Fatalf("Expected wlStatusMsg, got %T", msg)
	}
	// Should still report online even with no stations
	if !status.online {
		t.Error("Status should report online even with empty stations list")
	}
}

func TestWavelogStatusCheckMalformedStations(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/version":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "1.0"})
		case "/api/station_info/test-key":
			// Return malformed JSON
			w.Write([]byte("not json"))
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Config.Wavelog.Enabled = true
	m.App.Config.Wavelog.URL = srv.URL
	m.App.Config.Wavelog.APIKey = "test-key"
	m.App.Config.Wavelog.StationProfileID = "1"
	m.wlOnline = false

	cmd := m.checkWavelogCmd()
	if cmd == nil {
		t.Fatal("checkWavelogCmd should return a command")
	}

	msg := cmd()
	status, ok := msg.(wlStatusMsg)
	if !ok {
		t.Fatalf("Expected wlStatusMsg, got %T", msg)
	}
	// Should still report online — stations fetch failing is not fatal
	if !status.online {
		t.Error("Status should report online even when stations fetch fails")
	}
}
