package tui

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
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
	m.App.Logbook.Wavelog = &config.WavelogConfig{Enabled: false}

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
	m.App.Logbook.Wavelog = &config.WavelogConfig{Enabled: true, URL: "http://127.0.0.1:1", APIKey: "test-key", StationProfileID: "1"}
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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.lookup.wlOnline = false

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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.lookup.wlOnline = true

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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
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
	m.App.Logbook.Wavelog.Enabled = false

	cmd := m.maybeCheckWavelog()
	if cmd != nil {
		t.Error("maybeCheckWavelog should return nil when Wavelog is disabled")
	}
}

func TestWavelogUploadADIFNoStationProfile(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = "http://example.com"
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "" // no station
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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.lookup.wlOnline = false

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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.lookup.wlOnline = false

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
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.lookup.wlOnline = false

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

// =============================================================================
// postQSO unit tests — direct testing of the canonical upload path
// =============================================================================

// makeTestQSO creates a minimal QSO for insertTestQSO.
func makeTestQSO(call string) *qso.QSO {
	q := qso.NewQSO()
	q.Call = call
	q.Band = "20m"
	q.Mode = "SSB"
	q.QSODate = "20260614"
	q.TimeOn = "120000"
	q.RSTSent = "59"
	q.RSTRcvd = "59"
	q.WavelogUploaded = "no"
	q.Source = "manual"
	q.StationCallsign = "SP9MOA"
	return q
}

// getWavelogStatus reads the wavelog_uploaded status for a QSO ID.
func getWavelogStatus(t *testing.T, db *sql.DB, id int64) string {
	t.Helper()
	var status string
	err := db.QueryRow(`SELECT wavelog_uploaded FROM qsos WHERE id=?`, id).Scan(&status)
	if err != nil {
		t.Fatalf("query wavelog status: %v", err)
	}
	return status
}

func TestPostQSO_Success(t *testing.T) {
	srv := newWavelogTestServer(t, wavelogQSOHandler("ok", []string{""}, 0))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err != nil {
		t.Errorf("postQSO returned error: %v", err)
	}
	if !ok {
		t.Error("postQSO should return ok=true for success")
	}
	if isDup {
		t.Error("postQSO should return isDup=false for new QSO")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "yes" {
		t.Errorf("DB wavelog_uploaded = %q; want yes", status)
	}
}

func TestPostQSO_DuplicateViaAllDuplicates(t *testing.T) {
	srv := newWavelogTestServer(t, wavelogQSOHandler("abort",
		[]string{"", "Duplicate for SP9MOA"}, 1))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err != nil {
		t.Errorf("postQSO returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("postQSO should return ok=true for duplicate (AllDuplicates path)")
	}
	if !isDup {
		t.Error("postQSO should return isDup=true for duplicate")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "yes" {
		t.Errorf("DB wavelog_uploaded = %q; want yes (duplicate still counts as uploaded)", status)
	}
}

func TestPostQSO_DuplicateViaError(t *testing.T) {
	// Simulate Wavelog returning HTTP 400 with a body that PostQSOWithResult
	// can't parse as structured JSON, causing it to return an error whose
	// message contains "duplicate".
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Duplicate QSO detected",
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err != nil {
		t.Errorf("postQSO returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("postQSO should return ok=true for duplicate (error-text path)")
	}
	if !isDup {
		t.Error("postQSO should return isDup=true for duplicate (error-text path)")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "yes" {
		t.Errorf("DB wavelog_uploaded = %q; want yes", status)
	}
}

func TestPostQSO_ServerError(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", 500)
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err == nil {
		t.Error("postQSO should return an error for HTTP 500")
	}
	if ok {
		t.Error("postQSO should return ok=false for server error")
	}
	if isDup {
		t.Error("postQSO should return isDup=false for server error")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "no" {
		t.Errorf("DB wavelog_uploaded = %q; want no (upload failed)", status)
	}
}

func TestPostQSO_ConnectionError(t *testing.T) {
	// Use a non-routable address that will cause a connection error.
	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO("http://127.0.0.1:1", "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err == nil {
		t.Error("postQSO should return an error for connection failure")
	}
	if ok {
		t.Error("postQSO should return ok=false for connection failure")
	}
	if isDup {
		t.Error("postQSO should return isDup=false for connection failure")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "no" {
		t.Errorf("DB wavelog_uploaded = %q; want no (upload failed)", status)
	}
}

func TestPostQSO_DuplicateNoDupeInError(t *testing.T) {
	// HTTP 400 with a non-duplicate error — should NOT be treated as duplicate.
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Invalid station callsign",
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err == nil {
		t.Error("postQSO should return an error for non-duplicate 400")
	}
	if ok {
		t.Error("postQSO should return ok=false for non-duplicate error")
	}
	if isDup {
		t.Error("postQSO should return isDup=false when error is not about duplicates")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "no" {
		t.Errorf("DB wavelog_uploaded = %q; want no (upload failed)", status)
	}
}

func TestPostQSO_EmptyParameters(t *testing.T) {
	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	tests := []struct {
		name string
		url  string
		key  string
		sid  string
		adif string
	}{
		{"empty url", "", "key", "1", "<CALL:6>SP9MOA<EOR>"},
		{"empty key", "http://example.com", "", "1", "<CALL:6>SP9MOA<EOR>"},
		{"empty station id", "http://example.com", "key", "", "<CALL:6>SP9MOA<EOR>"},
		{"empty adif", "http://example.com", "key", "1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, isDup, err := postQSO(tt.url, tt.key, tt.sid, tt.adif, qID, "SP9MOA", m.App.DB)
			if err == nil {
				t.Error("postQSO should return an error for empty parameters")
			}
			if ok {
				t.Error("postQSO should return ok=false for empty parameters")
			}
			if isDup {
				t.Error("postQSO should return isDup=false for empty parameters")
			}
			if status := getWavelogStatus(t, m.App.DB, qID); status != "no" {
				t.Errorf("DB wavelog_uploaded = %q; want no", status)
			}
		})
	}
}

func TestPostQSO_HTTP200MalformedBody(t *testing.T) {
	// HTTP 200 but body is not valid JSON — PostQSOWithResult returns nil error.
	// postQSO should treat this as success (200 = accepted by server).
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte("<html>OK</html>"))
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err != nil {
		t.Errorf("postQSO returned unexpected error for HTTP 200: %v", err)
	}
	if !ok {
		t.Error("postQSO should return ok=true for HTTP 200 (even with malformed body)")
	}
	if isDup {
		t.Error("postQSO should return isDup=false for HTTP 200 success")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "yes" {
		t.Errorf("DB wavelog_uploaded = %q; want yes (HTTP 200 = accepted)", status)
	}
}

func TestPostQSO_RateLimitNotDuplicate(t *testing.T) {
	// HTTP 429 with a message that does NOT contain "duplicate".
	// postQSO must NOT falsely treat this as a duplicate.
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Rate limit exceeded — try again in 60 seconds",
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err == nil {
		t.Error("postQSO should return an error for HTTP 429")
	}
	if ok {
		t.Error("postQSO should return ok=false for rate limit")
	}
	if isDup {
		t.Error("postQSO should NOT report isDup=true for rate limit (not a duplicate)")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "no" {
		t.Errorf("DB wavelog_uploaded = %q; want no (rate limited, not on Wavelog)", status)
	}
}

func TestPostQSO_HTMLMaintenancePage(t *testing.T) {
	// HTTP 503 with an HTML maintenance page — must NOT falsely match "duplicate".
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(503)
		w.Write([]byte(`<html><body><h1>503 Maintenance</h1><p>Service temporarily unavailable</p></body></html>`))
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, isDup, err := postQSO(srv.URL, "test-key", "1", adifStr, qID, "SP9MOA", m.App.DB)
	if err == nil {
		t.Error("postQSO should return an error for HTTP 503")
	}
	if ok {
		t.Error("postQSO should return ok=false for maintenance page")
	}
	if isDup {
		t.Error("postQSO should NOT report isDup=true for maintenance page")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "no" {
		t.Errorf("DB wavelog_uploaded = %q; want no (server down, not on Wavelog)", status)
	}
}

// =============================================================================
// Pass 7 — Request payload verification and ADIF-to-Wavelog integrated flow
// =============================================================================

func TestPostQSO_RequestPayloadVerification(t *testing.T) {
	// Verify the POST body sent to Wavelog contains expected fields.
	var capturedBody map[string]string
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok", "adif_count": 1, "adif_errors": 0, "messages": []string{""},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, _, err := postQSO(srv.URL, "test-api-key-12345", "42", adifStr, qID, "SP9MOA", m.App.DB)
	if err != nil {
		t.Fatalf("postQSO: %v", err)
	}
	if !ok {
		t.Fatal("postQSO should succeed")
	}

	// Verify the request payload structure.
	if capturedBody["key"] != "test-api-key-12345" {
		t.Errorf("key = %q, want test-api-key-12345", capturedBody["key"])
	}
	if capturedBody["station_profile_id"] != "42" {
		t.Errorf("station_profile_id = %q, want 42", capturedBody["station_profile_id"])
	}
	if capturedBody["type"] != "adif" {
		t.Errorf("type = %q, want adif", capturedBody["type"])
	}
	if capturedBody["string"] != adifStr {
		t.Errorf("string (ADIF) = %q, want %q", capturedBody["string"], adifStr)
	}
}

func TestWavelogUpload_IntegratedADIFToUpload(t *testing.T) {
	// End-to-end: parse ADIF → insert to DB → trigger Wavelog upload → verify DB.
	var capturedADIF string
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		capturedADIF = body["string"]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok", "adif_count": 1, "adif_errors": 0, "messages": []string{""},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key-123"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	// Use the ADIF from Pass 5's FT8 test.
	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:8>14.074550 <MODE:3>FT8 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:3>-10 <RST_RCVD:3>-05 <GRIDSQUARE:6>JO90aa <EOR>"

	// Full logQSOFromADIF pipeline (parse → validate → insert → upload).
	cmd, retry := m.logQSOFromADIF(adif)
	if retry {
		t.Fatal("logQSOFromADIF should not request retry")
	}

	// Execute the upload command (from maybeUploadRawADIFToWavelog).
	if cmd != nil {
		msg := cmd()
		// logQSOFromADIF may return a Batch; execute each sub-command.
		if batch, isBatch := msg.(tea.BatchMsg); isBatch {
			for _, subCmd := range batch {
				subMsg := subCmd()
				if result, ok := subMsg.(wlUploadResultMsg); ok {
					if !result.ok {
						t.Errorf("upload should succeed, got err=%v", result.err)
					}
				}
			}
		} else if result, ok := msg.(wlUploadResultMsg); ok {
			if !result.ok {
				t.Errorf("upload should succeed, got err=%v", result.err)
			}
		}
	}

	// Verify QSO was persisted locally.
	qsos, err := store.ListQSOs(m.App.DB, 1, "")
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) == 0 {
		t.Fatal("no QSO found after logQSOFromADIF")
	}
	q := qsos[0]
	if q.Call != "SP9MOA" {
		t.Errorf("Call = %q", q.Call)
	}
	if q.Source != "wsjtx" {
		t.Errorf("Source = %q, want wsjtx", q.Source)
	}

	// Verify Wavelog status was updated locally.
	if status := getWavelogStatus(t, m.App.DB, q.ID); status != "yes" {
		t.Errorf("wavelog_uploaded = %q, want yes", status)
	}

	// Verify ADIF was sent to the mock server.
	if capturedADIF == "" {
		t.Error("no ADIF was captured by the mock server")
	}
}

func TestWavelogUpload_DisabledPreservesLocalQSO(t *testing.T) {
	// When Wavelog is disabled, QSO logs locally but no upload is triggered.
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog = nil // disabled
	m.App.Config.Integrations.QRZ.Enabled = false

	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:7>14.2500 <MODE:3>SSB " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	cmd, retry := m.logQSOFromADIF(adif)
	if retry {
		t.Fatal("should not retry")
	}

	// Upload command should be nil when Wavelog disabled.
	if cmd != nil {
		// cmd may be refreshQSOS, which is fine.
		msg := cmd()
		if _, isUpload := msg.(wlUploadResultMsg); isUpload {
			t.Error("upload should not be triggered when Wavelog is disabled")
		}
	}

	// Local QSO must still be persisted.
	qsos, err := store.ListQSOs(m.App.DB, 1, "")
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) == 0 {
		t.Fatal("no QSO found — local logging should work even with Wavelog disabled")
	}
}

func TestWavelogUpload_APINotExposedInLogs(t *testing.T) {
	// Verify the test uses a fake API key and it's sent in the POST body (not URL).
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// API key must be in POST body, not in URL query string.
		if r.URL.Query().Get("key") != "" {
			t.Error("API key should NOT be in URL query string")
		}
		if r.URL.Path != "/index.php/api/qso" {
			http.NotFound(w, r)
			return
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "fake-key-for-test" {
			t.Errorf("key in body = %q", body["key"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok", "adif_count": 1, "adif_errors": 0, "messages": []string{""},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	ok, _, err := postQSO(srv.URL, "fake-key-for-test", "1",
		"<CALL:6>SP9MOA<EOR>", qID, "SP9MOA", m.App.DB)
	if err != nil {
		t.Errorf("postQSO: %v", err)
	}
	if !ok {
		t.Error("postQSO should succeed")
	}
}

// =============================================================================
// Pass 15 — Wavelog FetchContacts (download) tests with httptest.Server
// =============================================================================

func TestFetchContacts_Success(t *testing.T) {
	const adifResponse = `SP9MOA de DJ7NT
<CALL:6>SP9MOA <BAND:3>20m <FREQ:7>14.2500 <MODE:3>SSB
<QSO_DATE:8>20260614 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59
<GRIDSQUARE:4>JO90 <NAME:4>John <EOR>
<CALL:5>W1AW <BAND:3>40m <FREQ:6>7.1850 <MODE:2>CW
<QSO_DATE:8>20260615 <TIME_ON:6>130000 <RST_SENT:3>599 <RST_RCVD:3>579
<GRIDSQUARE:6>FN31pr <EOR>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 2,
			"lastfetchedid": "42",
			"message":       "OK",
			"adif":          adifResponse,
		})
	}))
	defer srv.Close()

	result, err := wavelog.FetchContacts(srv.URL, "test-key", "1", 0)
	if err != nil {
		t.Fatalf("FetchContacts: %v", err)
	}
	if result.ExportedQSOs != 2 {
		t.Errorf("ExportedQSOs = %d, want 2", result.ExportedQSOs)
	}
	if result.LastFetchedID() != 42 {
		t.Errorf("LastFetchedID = %d, want 42", result.LastFetchedID())
	}
	if result.ADIFPath == "" {
		t.Error("ADIFPath should not be empty")
	}
	// Clean up temp file.
	os.Remove(result.ADIFPath)
}

func TestFetchContacts_EmptyADIF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 0,
			"lastfetchedid": "0",
			"message":       "No QSOs",
			"adif":          "",
		})
	}))
	defer srv.Close()

	result, err := wavelog.FetchContacts(srv.URL, "test-key", "1", 0)
	if err != nil {
		t.Fatalf("FetchContacts with empty ADIF: %v", err)
	}
	if result.ExportedQSOs != 0 {
		t.Errorf("ExportedQSOs = %d, want 0", result.ExportedQSOs)
	}
	os.Remove(result.ADIFPath)
}

func TestFetchContacts_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	}))
	defer srv.Close()

	_, err := wavelog.FetchContacts(srv.URL, "test-key", "1", 0)
	if err == nil {
		t.Error("FetchContacts should return error on HTTP 500")
	}
}

func TestFetchContacts_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", 401)
	}))
	defer srv.Close()

	_, err := wavelog.FetchContacts(srv.URL, "test-key", "1", 0)
	if err == nil {
		t.Error("FetchContacts should return error on HTTP 401")
	}
}

func TestFetchContacts_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	_, err := wavelog.FetchContacts(srv.URL, "test-key", "1", 0)
	if err == nil {
		t.Error("FetchContacts should return error on invalid JSON")
	}
}

func TestFetchContacts_MissingParams(t *testing.T) {
	_, err := wavelog.FetchContacts("", "key", "1", 0)
	if err == nil {
		t.Error("FetchContacts should fail with empty URL")
	}
	_, err = wavelog.FetchContacts("https://example.com", "", "1", 0)
	if err == nil {
		t.Error("FetchContacts should fail with empty API key")
	}
	_, err = wavelog.FetchContacts("https://example.com", "key", "", 0)
	if err == nil {
		t.Error("FetchContacts should fail with empty station ID")
	}
}

func TestFetchContacts_HTMLResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(503)
		w.Write([]byte(`<html><body><h1>503 Maintenance</h1></body></html>`))
	}))
	defer srv.Close()

	_, err := wavelog.FetchContacts(srv.URL, "test-key", "1", 0)
	if err == nil {
		t.Error("FetchContacts should return error on HTML maintenance page")
	}
}

func TestFetchContacts_PayloadVerification(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 0,
			"lastfetchedid": "0",
			"message":       "",
			"adif":          "",
		})
	}))
	defer srv.Close()

	result, err := wavelog.FetchContacts(srv.URL, "test-api-key", "42", 100)
	if err != nil {
		t.Fatalf("FetchContacts: %v", err)
	}
	os.Remove(result.ADIFPath)

	if capturedBody["key"] != "test-api-key" {
		t.Errorf("key = %q, want test-api-key", capturedBody["key"])
	}
	// station_id is sent as int in JSON
	if capturedBody["fetchfromid"] != float64(100) {
		t.Errorf("fetchfromid = %v, want 100", capturedBody["fetchfromid"])
	}
}
