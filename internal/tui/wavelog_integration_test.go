package tui

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
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

// wavelogVersionHandler returns a handler for GET /api/v2/status.
func wavelogVersionHandler(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/status" {
			http.NotFound(w, r)
			return
		}
		resp := map[string]any{
			"data": map[string]string{"name": "Wavelog API", "status": status},
			"meta": map[string]string{"resource": "status", "method": "GET"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// wavelogTokenHandler returns a handler for GET /api/v2/token.
func wavelogTokenHandler(scopes []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/token" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "unauthorized", "message": "Missing API token"},
			})
			return
		}
		if scopes == nil {
			scopes = []string{"qso:read", "qso:write", "station:read", "lookup:read"}
		}
		resp := map[string]any{
			"data": map[string]any{
				"id": 1, "name": "Test", "owner": "SP9MOA", "user_id": 1,
				"scopes": scopes, "expires_at": "2099-01-01 00:00:00",
			},
			"meta": map[string]string{"resource": "token", "method": "GET"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// wavelogStationHandler returns a handler for GET /api/v2/station.
func wavelogStationHandler(stations []map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/station" {
			json.NewEncoder(w).Encode(map[string]any{
				"data": stations,
				"meta": map[string]string{"resource": "station", "method": "GET"},
			})
			return
		}
		if len(r.URL.Path) > len("/api/v2/station/") && r.URL.Path[:len("/api/v2/station/")] == "/api/v2/station/" {
			id := r.URL.Path[len("/api/v2/station/"):]
			for _, s := range stations {
				if fmt.Sprint(s["id"]) == id {
					json.NewEncoder(w).Encode(map[string]any{
						"data": s,
						"meta": map[string]string{"resource": "station", "method": "GET"},
					})
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "not_found", "message": "Station not found"},
			})
			return
		}
		http.NotFound(w, r)
	}
}

// wavelogQSOHandler returns a handler for POST /api/v2/qso (ADIF import).
// status "ok" imports the QSO; status "abort" reports it as duplicate.
func wavelogQSOHandler(status string, messages []string, adifErrors int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/qso" && r.Method == http.MethodGet {
			// Remote-id backfill list for the standard SP9MOA test QSO.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 42, "call": "SP9MOA", "band": "20m", "mode": "SSB", "qso_date": "2026-06-14 12:00:00"},
				},
			})
			return
		}
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data := map[string]any{"parsed": 1, "imported": 1, "skipped": 0, "messages": messages}
		if status != "ok" {
			data["imported"] = 0
			data["skipped"] = 1
		}
		resp := map[string]any{
			"data": data,
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

// wavelogPrivateLookupHandler returns a handler for GET /api/v2/lookup.
func wavelogPrivateLookupHandler(data map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/lookup" || r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if data == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "not_found", "message": "not found"},
			})
			return
		}
		resp := map[string]any{
			"data": data,
			"meta": map[string]string{"resource": "lookup", "method": "GET", "detail": "full"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
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

// TestWavelogUploadAfterSwitchTargetsCapturedLogbook verifies that a single
// QSO upload command created for logbook A still stores the remote id in A
// even when the user switches to logbook B before the command runs.
func TestWavelogUploadAfterSwitchTargetsCapturedLogbook(t *testing.T) {
	srv := newWavelogTestServer(t, wavelogQSOHandler("ok", []string{""}, 0))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	oldDB := m.App.DB
	qs := qso.NewQSO()
	qs.Call = "SP9MOA"
	qs.Band = "20m"
	qs.Mode = "SSB"
	qs.QSODate = "20260614"
	qs.TimeOn = "120000"
	id, err := store.InsertQSO(oldDB, qs)
	if err != nil {
		t.Fatalf("insert QSO: %v", err)
	}
	qs.ID = id

	cmd := m.uploadQSOToWavelog(qs)
	if cmd == nil {
		t.Fatal("uploadQSOToWavelog should return a command")
	}

	// Simulate a logbook switch while the upload is in flight.
	dbB, err := store.InitDB(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("init db b: %v", err)
	}
	t.Cleanup(func() { dbB.Close() })
	m.App.DB = dbB
	m.App.LogbookName = "b"

	msg := cmd()
	result, ok := msg.(wlUploadResultMsg)
	if !ok {
		t.Fatalf("Expected wlUploadResultMsg, got %T", msg)
	}
	if !result.ok {
		t.Error("Upload should succeed with mock server")
	}
	if result.logbook != "test" {
		t.Errorf("result logbook = %q, want 'test'", result.logbook)
	}

	// Remote id must land in the original logbook…
	uploaded, err := store.GetQSOByID(oldDB, id)
	if err != nil || uploaded == nil {
		t.Fatalf("original logbook QSO missing: %v", err)
	}
	if uploaded.WavelogID == 0 {
		t.Error("original logbook should have the remote Wavelog id stored")
	}
	// …and the new logbook must stay untouched.
	newQsos, err := store.ListAllQSOs(dbB)
	if err != nil {
		t.Fatalf("list new logbook: %v", err)
	}
	if len(newQsos) != 0 {
		t.Errorf("new logbook has %d QSOs, want 0", len(newQsos))
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
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/status" {
			wavelogVersionHandler("ok")(w, r)
			return
		}
		if r.URL.Path == "/api/v2/token" {
			wavelogTokenHandler(nil)(w, r)
			return
		}
		http.NotFound(w, r)
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test-key"
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
	// A legacy v1 key must be rejected with the actionable v2-required message.
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "invalid_token", "message": "Invalid or revoked API token"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test-key"
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
	data := map[string]any{
		"callsign":              "SP9MOA",
		"name":                  "John",
		"call_worked":           true,
		"call_worked_band":      true,
		"call_worked_band_mode": false,
		"dxcc_confirmed":        true,
		"lotw_member":           "14",
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

func TestWavelogUploadNoStationProfile(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = "http://example.com"
	m.App.Logbook.Wavelog.APIKey = "test-key"
	m.App.Logbook.Wavelog.StationProfileID = "" // no station
	m.inetOnline = true

	qs := qso.NewQSO()
	qs.Call = "SP9MOA"
	qs.ID = 1

	cmd := m.uploadQSOToWavelog(qs)
	if cmd != nil {
		t.Error("uploadQSOToWavelog should return nil when station profile is empty")
	}
}

// =============================================================================
// Wavelog station-info / FetchStations mock tests
// =============================================================================

func TestWavelogStatusCheckWithStations(t *testing.T) {
	// Mock v2 status + token, then the station list.
	// checkWavelogCmd calls TestConnection first, then FetchStations.
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/status":
			wavelogVersionHandler("ok")(w, r)
		case "/api/v2/token":
			wavelogTokenHandler(nil)(w, r)
		case "/api/v2/station":
			wavelogStationHandler([]map[string]any{
				{
					"id":         1,
					"name":       "Home QTH",
					"gridsquare": "JO90",
					"callsign":   "SP9MOA",
					"active":     true,
				},
			})(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test-key"
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
		switch r.URL.Path {
		case "/api/v2/status":
			wavelogVersionHandler("ok")(w, r)
		case "/api/v2/token":
			wavelogTokenHandler(nil)(w, r)
		case "/api/v2/station":
			wavelogStationHandler([]map[string]any{})(w, r) // empty
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test-key"
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
		switch r.URL.Path {
		case "/api/v2/status":
			wavelogVersionHandler("ok")(w, r)
		case "/api/v2/token":
			wavelogTokenHandler(nil)(w, r)
		case "/api/v2/station":
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
	m.App.Logbook.Wavelog.APIKey = "wl2_test-key"
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
	q.Source = "manual"
	q.StationCallsign = "SP9MOA"
	return q
}

// getWavelogStatus reads the wavelog_id for a QSO ID and reports it as the
// legacy yes/no string so the assertion sites stay readable.
func getWavelogStatus(t *testing.T, db *sql.DB, id int64) string {
	t.Helper()
	var remoteID int64
	err := db.QueryRow(`SELECT wavelog_id FROM qsos WHERE id=?`, id).Scan(&remoteID)
	if err != nil {
		t.Fatalf("query wavelog status: %v", err)
	}
	if remoteID > 0 {
		return "yes"
	}
	return "no"
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
		t.Errorf("DB wavelog status = %q; want yes", status)
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
		t.Errorf("DB wavelog status = %q; want yes (duplicate still counts as uploaded)", status)
	}
}

func TestPostQSO_DuplicateViaError(t *testing.T) {
	// v2 signals a conflicting state with 409 + error code "conflict".
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			// Remote-id backfill after the duplicate was detected.
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 42, "call": "SP9MOA", "band": "20m", "mode": "SSB", "qso_date": "2026-06-14 12:00:00"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "conflict", "message": "Duplicate QSO detected"},
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
		t.Errorf("DB wavelog status = %q; want yes", status)
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
		t.Errorf("DB wavelog status = %q; want no (upload failed)", status)
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
		t.Errorf("DB wavelog status = %q; want no (upload failed)", status)
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
		t.Errorf("DB wavelog status = %q; want no (upload failed)", status)
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
				t.Errorf("DB wavelog status = %q; want no", status)
			}
		})
	}
}

func TestPostQSO_HTTP200MalformedBody(t *testing.T) {
	// HTTP 200 with a body that is not the v2 envelope — v2 always sends a
	// {data,meta} envelope on success, so this must be treated as a failure.
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
	if err == nil {
		t.Error("postQSO should return an error for a malformed 200 body")
	}
	if ok {
		t.Error("postQSO should return ok=false for a malformed 200 body")
	}
	if isDup {
		t.Error("postQSO should return isDup=false for a malformed 200 body")
	}
	if status := getWavelogStatus(t, m.App.DB, qID); status != "no" {
		t.Errorf("DB wavelog status = %q; want no (response unverifiable)", status)
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
		t.Errorf("DB wavelog status = %q; want no (rate limited, not on Wavelog)", status)
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
		t.Errorf("DB wavelog status = %q; want no (server down, not on Wavelog)", status)
	}
}

// =============================================================================
// Pass 7 — Request payload verification and ADIF-to-Wavelog integrated flow
// =============================================================================

func TestPostQSO_RequestPayloadVerification(t *testing.T) {
	// Verify the v2 POST body sent to Wavelog contains expected fields.
	var capturedBody map[string]any
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"parsed": 1, "imported": 1, "skipped": 0, "messages": []string{}},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))

	adifStr := "<CALL:6>SP9MOA<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260614<TIME_ON:6>120000<EOR>"
	ok, _, err := postQSO(srv.URL, "wl2_test-api-key-12345", "42", adifStr, qID, "SP9MOA", m.App.DB)
	if err != nil {
		t.Fatalf("postQSO: %v", err)
	}
	if !ok {
		t.Fatal("postQSO should succeed")
	}

	// Verify the request payload structure.
	if capturedBody["import_type"] != "adif" {
		t.Errorf("import_type = %v, want adif", capturedBody["import_type"])
	}
	if capturedBody["station_profile_id"] != float64(42) {
		t.Errorf("station_profile_id = %v, want 42", capturedBody["station_profile_id"])
	}
	if capturedBody["adif"] != adifStr {
		t.Errorf("adif payload = %v, want %q", capturedBody["adif"], adifStr)
	}
}

// TestPostQSOSingle_StoresRemoteID verifies the single-QSO JSON create path
// stores the remote id returned by Wavelog.
func TestPostQSOSingle_StoresRemoteID(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["import_type"] != nil {
			t.Errorf("JSON create must not set import_type, got %v", body["import_type"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 42, "call": "SP9MOA"},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))
	qs, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	ok, isDup, remoteID, _, err := postQSOSingle(srv.URL, "wl2_test", "1", qs, m.App.DB, -1)
	if err != nil {
		t.Fatalf("postQSOSingle: %v", err)
	}
	if !ok || isDup || remoteID != 42 {
		t.Errorf("ok=%v isDup=%v remoteID=%d, want ok=true isDup=false remoteID=42", ok, isDup, remoteID)
	}

	var storedID int64
	if err := m.App.DB.QueryRow(`SELECT wavelog_id FROM qsos WHERE id=?`, qID).
		Scan(&storedID); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if storedID != 42 {
		t.Errorf("wavelog_id = %d, want 42", storedID)
	}
}

// TestPostQSOSingle_FallsBackToADIF verifies that a rejected JSON create falls
// back to the ADIF import path, preserving upload behavior.
func TestPostQSOSingle_FallsBackToADIF(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, isADIF := body["adif"]; !isADIF {
			// Reject the JSON create with a validation error.
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "validation_error", "message": "bad call"},
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"parsed": 1, "imported": 1, "skipped": 0, "messages": []string{}},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))
	qs, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	ok, isDup, _, _, err := postQSOSingle(srv.URL, "wl2_test", "1", qs, m.App.DB, -1)
	if err != nil {
		t.Fatalf("postQSOSingle fallback: %v", err)
	}
	if !ok || isDup {
		t.Errorf("ok=%v isDup=%v, want ok=true isDup=false", ok, isDup)
	}

	var remoteID int64
	if err := m.App.DB.QueryRow(`SELECT wavelog_id FROM qsos WHERE id=?`, qID).
		Scan(&remoteID); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if remoteID != 0 {
		t.Errorf("wavelog_id = %d, want 0 (ADIF fallback returns no id)", remoteID)
	}
}

// TestWavelogStatusCheckV1Key verifies that a legacy v1 key is detected
// without any network call and reported with the migration message.
func TestWavelogStatusCheckV1Key(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = "http://127.0.0.1:1" // unreachable — must not be called
	m.App.Logbook.Wavelog.APIKey = "wl123_not_v2"
	m.App.Logbook.Wavelog.StationProfileID = "1"

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
		t.Error("v1 key must report offline")
	}
	if status.err != wavelog.V1KeyRequiredMsg {
		t.Errorf("err = %q, want the v2 migration message", status.err)
	}
}

// TestWavelogStatusErrToast verifies the status error is surfaced once as a
// warning toast and not repeated while it stays unchanged.
func TestWavelogStatusErrToast(t *testing.T) {
	m := newLifecycleTestModel(t)

	next, _ := m.Update(wlStatusMsg{online: false, err: wavelog.V1KeyRequiredMsg})
	m = next.(*Model)
	if m.lookup.wlStatusErr != wavelog.V1KeyRequiredMsg {
		t.Fatalf("wlStatusErr = %q", m.lookup.wlStatusErr)
	}
	toasts := m.toasts.Active()
	found := false
	for _, toast := range toasts {
		if toast.Level == ToastWarning && strings.Contains(toast.Message, "wl2_") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected migration warning toast, got %+v", toasts)
	}
	n1 := len(toasts)

	// Repeating the same error must not re-toast.
	next, _ = m.Update(wlStatusMsg{online: false, err: wavelog.V1KeyRequiredMsg})
	m = next.(*Model)
	if n2 := len(m.toasts.Active()); n2 != n1 {
		t.Errorf("toast count grew from %d to %d — repeated warning", n1, n2)
	}
}

// TestPostQSOSingle_AuthFailureNoFallback verifies an auth failure surfaces
// the migration guidance directly and does not attempt the ADIF fallback.
func TestPostQSOSingle_AuthFailureNoFallback(t *testing.T) {
	requests := 0
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "invalid_token", "message": "Invalid or revoked API token"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	qID := insertTestQSO(t, m.App.DB, makeTestQSO("SP9MOA"))
	qs, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	ok, _, _, _, err := postQSOSingle(srv.URL, "wl2_bad", "1", qs, m.App.DB, -1)
	if ok {
		t.Error("upload should fail on invalid token")
	}
	if err == nil || !strings.Contains(err.Error(), "wl2_") {
		t.Errorf("err = %v, want v2 migration guidance", err)
	}
	if requests != 1 {
		t.Errorf("requests = %d, want 1 (no ADIF fallback on auth errors)", requests)
	}
}

func TestWavelogUpload_IntegratedADIFToUpload(t *testing.T) {
	// End-to-end: parse ADIF → insert to DB → trigger Wavelog upload → verify DB.
	var capturedCall, capturedBand, capturedMode, capturedDate string
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		capturedCall, _ = body["call"].(string)
		capturedBand, _ = body["band"].(string)
		capturedMode, _ = body["mode"].(string)
		capturedDate, _ = body["qso_date"].(string)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 77, "call": "SP9MOA"},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
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
		t.Errorf("wavelog status = %q, want yes", status)
	}

	// Verify the QSO was created via v2 JSON with the expected fields and
	// the remote id was stored locally.
	if capturedCall != "SP9MOA" {
		t.Errorf("captured call = %q, want SP9MOA", capturedCall)
	}
	if capturedBand != "20m" {
		t.Errorf("captured band = %q, want 20m", capturedBand)
	}
	if capturedMode != "FT8" {
		t.Errorf("captured mode = %q, want FT8", capturedMode)
	}
	if capturedDate != "2026-06-18" {
		t.Errorf("captured qso_date = %q, want 2026-06-18 (ISO)", capturedDate)
	}
	if q.WavelogID != 77 {
		t.Errorf("WavelogID = %d, want 77 (remote id stored)", q.WavelogID)
	}
}

func TestWavelogUpload_DisabledPreservesLocalQSO(t *testing.T) {
	// When Wavelog is disabled, QSO logs locally but no upload is triggered.
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog = nil // disabled
	m.App.Config.Integrations.Callbook.QRZ.Enabled = false

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
	// Verify the token is sent in the Authorization header, never in the URL.
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// API token must not appear in the URL query string.
		if r.URL.Query().Get("key") != "" {
			t.Error("API token should NOT be in URL query string")
		}
		if r.URL.Path != "/api/v2/qso" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer fake-key-for-test" {
			t.Errorf("Authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.Method == http.MethodGet {
			// Remote-id backfill after the successful upload.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 42, "call": "SP9MOA", "band": "20m", "mode": "SSB", "qso_date": "2026-06-14 12:00:00"},
				},
			})
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["import_type"] != "adif" {
			t.Errorf("import_type = %v", body["import_type"])
		}
		if body["station_profile_id"] != float64(1) {
			t.Errorf("station_profile_id = %v", body["station_profile_id"])
		}
		if _, ok := body["adif"].(string); !ok || body["adif"] == "" {
			t.Error("adif payload missing")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"parsed": 1, "imported": 1, "skipped": 0, "messages": []string{}},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
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
// stripMyGridsquare tests
// =============================================================================

func TestStripMyGridsquare(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no grid field",
			input: "<CALL:6>SP9MOA<BAND:3>20m<EOR>",
			want:  "<CALL:6>SP9MOA<BAND:3>20m<EOR>",
		},
		{
			name:  "strips MY_GRIDSQUARE",
			input: "<CALL:6>SP9MOA<MY_GRIDSQUARE:6>JO90FA<BAND:3>20m<EOR>",
			want:  "<CALL:6>SP9MOA<BAND:3>20m<EOR>",
		},
		{
			name:  "strips lowercase",
			input: "<CALL:6>SP9MOA<my_gridsquare:6>JO90FA<BAND:3>20m<EOR>",
			want:  "<CALL:6>SP9MOA<BAND:3>20m<EOR>",
		},
		{
			name:  "strips with trailing space",
			input: "<CALL:6>SP9MOA<MY_GRIDSQUARE:6 >JO90FA<BAND:3>20m<EOR>",
			want:  "<CALL:6>SP9MOA<BAND:3>20m<EOR>",
		},
		{
			name:  "preserves other MY_ fields",
			input: "<CALL:6>SP9MOA<MY_CITY:4>Krak<MY_GRIDSQUARE:6>JO90FA<BAND:3>20m<EOR>",
			want:  "<CALL:6>SP9MOA<MY_CITY:4>Krak<BAND:3>20m<EOR>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMyGridsquare(tt.input)
			if got != tt.want {
				t.Errorf("stripMyGridsquare(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
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
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      2,
				"lastfetchedid": 42,
				"adif":          adifResponse,
			},
			"meta": map[string]any{"has_more": false},
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
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      0,
				"lastfetchedid": 0,
				"adif":          nil,
			},
			"meta": map[string]any{"has_more": false},
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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodGet {
			http.Error(w, "bad request", 400)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("format") != "adif" {
			t.Errorf("format = %q, want adif", r.URL.Query().Get("format"))
		}
		if r.URL.Query().Get("since_id") != "100" {
			t.Errorf("since_id = %q, want 100", r.URL.Query().Get("since_id"))
		}
		if r.URL.Query().Get("station_id") != "42" {
			t.Errorf("station_id = %q, want 42", r.URL.Query().Get("station_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      0,
				"lastfetchedid": 100,
				"adif":          nil,
			},
			"meta": map[string]any{"has_more": false},
		})
	}))
	defer srv.Close()

	result, err := wavelog.FetchContacts(srv.URL, "test-api-key", "42", 100)
	if err != nil {
		t.Fatalf("FetchContacts: %v", err)
	}
	os.Remove(result.ADIFPath)
	if result.ExportedQSOs != 0 {
		t.Errorf("ExportedQSOs = %d, want 0", result.ExportedQSOs)
	}
	if result.LastFetchedID() != 100 {
		t.Errorf("LastFetchedID = %d, want 100 (kept when nothing new)", result.LastFetchedID())
	}
}

// TestUploadReconcilesRowEditedDuringFlight reproduces the reported
// inconsistent state: an initial upload sends a captured snapshot, and while
// the request is pending the contact is edited — the local save bypassed
// dirty tracking (remote id still zero), and the upload completion attached
// the remote id without checking, leaving the server with the old content
// and the local row falsely synced. The completion must detect the newer
// revision, keep the row durably dirty, and queue a PATCH of the latest
// revision.
func TestUploadReconcilesRowEditedDuringFlight(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var patchedComment string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			close(entered)
			<-release
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 42}})
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode patch body: %v", err)
			}
			if v, ok := body["comment"].(string); ok {
				patchedComment = v
			}
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 42}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"

	qID := insertTestQSO(t, m.App.DB, &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB",
		QSODate: "20240501", TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before"})
	qs, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	cmd := m.uploadQSOToWavelog(qs)
	if cmd == nil {
		t.Fatal("uploadQSOToWavelog returned nil")
	}
	done := make(chan wlUploadResultMsg, 1)
	go func() { done <- execCmd(cmd).(wlUploadResultMsg) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("upload POST never started")
	}

	// The operator edits the contact while the upload is on the wire.
	edited := *qs
	edited.Comment = "newer"
	if err := store.UpdateQSO(m.App.DB, &edited); err != nil {
		t.Fatalf("UpdateQSO during upload: %v", err)
	}

	close(release)
	res := <-done
	if !res.ok || res.remoteID != 42 {
		t.Fatalf("upload: ok=%v remoteID=%d err=%v, want ok with id 42", res.ok, res.remoteID, res.err)
	}
	if !res.changed {
		t.Fatal("changed must be true — the row was edited while the upload was pending")
	}

	// The row is durably dirty and a reconciliation PATCH is queued.
	stored, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID after upload: %v", err)
	}
	if stored.WavelogID != 42 {
		t.Errorf("WavelogID = %d, want 42", stored.WavelogID)
	}
	if !stored.WavelogDirty {
		t.Fatal("the row must stay dirty — the remote copy holds the old snapshot")
	}

	upd, c := m.Update(res)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the reconciliation PATCH was not queued")
	}
	batch, isBatch := execCmd(c).(tea.BatchMsg)
	if !isBatch {
		t.Fatalf("Update returned %T, want tea.BatchMsg", c)
	}
	var patchEm editorMsg
	var walk func(tea.Cmd)
	walk = func(sub tea.Cmd) {
		got := sub()
		if em, ok := got.(editorMsg); ok && em.wlSyncFollowUp {
			patchEm = em
			return
		}
		if inner, ok := got.(tea.BatchMsg); ok {
			for _, nested := range inner {
				if patchEm.saved != 0 {
					return
				}
				walk(nested)
			}
		}
	}
	for _, sub := range batch {
		walk(sub)
	}
	if patchEm.saved == 0 {
		t.Fatal("the reconciliation PATCH was not among the dispatched commands")
	}
	if !patchEm.wlSyncOK {
		t.Fatalf("reconciliation PATCH: ok=%v err=%q", patchEm.wlSyncOK, patchEm.wlSyncErr)
	}
	if patchedComment != "newer" {
		t.Errorf("PATCH comment = %q, want newer", patchedComment)
	}
	if m.handleQSOSyncCompletion(patchEm) != nil {
		t.Fatal("no further PATCH should be dispatched")
	}

	stored, err = store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID after reconciliation: %v", err)
	}
	if stored.WavelogDirty {
		t.Error("dirty should clear after the newest revision was acknowledged")
	}
	if stored.Comment != "newer" {
		t.Errorf("comment = %q, want newer", stored.Comment)
	}
}

// TestUploadIDPersistenceFailureReportedUnresolved reproduces the reported
// false success: Wavelog accepts the contact, but the local id write fails —
// previously the completion still reported success with a nonzero remote id,
// the row stayed unsent locally, and the user saw no incomplete-sync result.
// Remote acceptance and local persistence must be separate outcomes: the
// completion reports unresolved, and a retry attaches the id once the
// database accepts the write again.
func TestUploadIDPersistenceFailureReportedUnresolved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 42}})
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 42, "call": "SP9MOA", "band": "20m", "mode": "SSB",
						"qso_date": "2024-05-01 12:00:00"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"

	qID := insertTestQSO(t, m.App.DB, &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB",
		QSODate: "20240501", TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before"})
	qs, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	// Deterministically reject the local id write.
	if _, err := m.App.DB.Exec(`CREATE TRIGGER block_wl_update BEFORE UPDATE OF wavelog_id ON qsos
		WHEN OLD.wavelog_id = 0 AND NEW.wavelog_id != 0
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	res := execCmd(m.uploadQSOToWavelog(qs)).(wlUploadResultMsg)
	if !res.ok {
		t.Fatalf("upload: ok=%v err=%v, want remote acceptance", res.ok, res.err)
	}
	if !res.unresolved {
		t.Fatal("unresolved must be true — the server accepted but the local id write failed")
	}
	if res.remoteID != 0 {
		t.Errorf("remoteID = %d, want 0 (the id was NOT persisted)", res.remoteID)
	}
	var storedID int64
	if err := m.App.DB.QueryRow(`SELECT wavelog_id FROM qsos WHERE id=?`, qID).Scan(&storedID); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if storedID != 0 {
		t.Fatalf("stored wavelog_id = %d, want 0 (write was blocked)", storedID)
	}

	// Remove the blocker — the completion must schedule an id-attach retry.
	if _, err := m.App.DB.Exec(`DROP TRIGGER block_wl_update`); err != nil {
		t.Fatalf("drop trigger: %v", err)
	}
	upd, c := m.Update(res)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the id-attach retry was not queued")
	}
	var retried wlUploadResultMsg
	var walk func(tea.Cmd)
	walk = func(sub tea.Cmd) {
		if retried.retried {
			return
		}
		if r, ok := sub().(wlUploadResultMsg); ok && r.retried {
			retried = r
			return
		}
		if inner, ok := execCmd(sub).(tea.BatchMsg); ok {
			for _, nested := range inner {
				walk(nested)
			}
		}
	}
	batch, isBatch := execCmd(c).(tea.BatchMsg)
	if isBatch {
		for _, sub := range batch {
			walk(sub)
		}
	} else {
		walk(c)
	}
	if !retried.retried {
		t.Fatal("the retry result was not among the dispatched commands")
	}
	if retried.unresolved || retried.remoteID != 42 {
		t.Fatalf("retry: unresolved=%v remoteID=%d, want id 42 persisted", retried.unresolved, retried.remoteID)
	}
	if err := m.App.DB.QueryRow(`SELECT wavelog_id FROM qsos WHERE id=?`, qID).Scan(&storedID); err != nil {
		t.Fatalf("read back after retry: %v", err)
	}
	if storedID != 42 {
		t.Errorf("stored wavelog_id = %d after retry, want 42", storedID)
	}
}

// TestUploadIDRetryPreservesSnapshotPair reproduces the reported retry
// mismatch: the server accepted an upload but the local id write failed, and
// the operator edited the contact before the id-attach retry. The retry used
// to read the CURRENT row and attach its id with the CURRENT revision, so the
// older server contents got marked clean under the newer revision — and the
// backfill looked up the id with the edited identity. The retry must preserve
// the accepted snapshot's identity AND revision, and carry the originating
// logbook.
func TestUploadIDRetryPreservesSnapshotPair(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 42}})
		case http.MethodGet:
			// The accepted contact lives under the ORIGINAL callsign; a
			// lookup with the edited identity resolves to a different id.
			rid := 99
			call := r.URL.Query().Get("callsign")
			if call == "SP9MOA" {
				rid = 42
			}
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": rid, "call": call, "band": "20m", "mode": "SSB",
						"qso_date": "2024-05-01 12:00:00"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"

	qID := insertTestQSO(t, m.App.DB, &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB",
		QSODate: "20240501", TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "old"})
	qs, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	// Deterministically reject the local id write.
	if _, err := m.App.DB.Exec(`CREATE TRIGGER block_wl_update BEFORE UPDATE OF wavelog_id ON qsos
		WHEN OLD.wavelog_id = 0 AND NEW.wavelog_id != 0
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	res := execCmd(m.uploadQSOToWavelog(qs)).(wlUploadResultMsg)
	if !res.ok || !res.unresolved || res.remoteID != 0 {
		t.Fatalf("upload: ok=%v unresolved=%v remoteID=%d, want accepted-but-unresolved", res.ok, res.unresolved, res.remoteID)
	}
	// The completion must carry the accepted snapshot pair.
	if res.snap.Call != "SP9MOA" || res.snap.ID != qID {
		t.Fatalf("snapshot = call:%q id:%d, want the accepted SP9MOA/%d", res.snap.Call, res.snap.ID, qID)
	}
	if res.uploadedRev != 0 {
		t.Fatalf("uploadedRev = %d, want 0 (the snapshot's revision)", res.uploadedRev)
	}

	// The id write can succeed again, and the operator edits the contact —
	// the revision now describes NEWER contents than the accepted snapshot.
	if _, err := m.App.DB.Exec(`DROP TRIGGER block_wl_update`); err != nil {
		t.Fatalf("drop trigger: %v", err)
	}
	row, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID before edit: %v", err)
	}
	row.Call = "SP9BBB"
	row.Comment = "newer"
	if err := store.UpdateQSO(m.App.DB, row); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}

	// The completion must schedule the id-attach retry.
	upd, c := m.Update(res)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the id-attach retry was not queued")
	}
	var retried wlUploadResultMsg
	var walk func(tea.Cmd)
	walk = func(sub tea.Cmd) {
		if retried.retried {
			return
		}
		if r, ok := sub().(wlUploadResultMsg); ok && r.retried {
			retried = r
			return
		}
		if inner, ok := execCmd(sub).(tea.BatchMsg); ok {
			for _, nested := range inner {
				walk(nested)
			}
		}
	}
	batch, isBatch := execCmd(c).(tea.BatchMsg)
	if isBatch {
		for _, sub := range batch {
			walk(sub)
		}
	} else {
		walk(c)
	}
	if !retried.retried {
		t.Fatal("the retry result was not among the dispatched commands")
	}
	if !retried.ok || retried.unresolved {
		t.Fatalf("retry: ok=%v unresolved=%v, want the id attached", retried.ok, retried.unresolved)
	}
	if retried.remoteID != 42 {
		t.Errorf("retry remoteID = %d, want 42 — the lookup must use the ACCEPTED snapshot identity", retried.remoteID)
	}
	if !retried.changed {
		t.Error("retry changed must be true — the row was edited after the snapshot was accepted")
	}
	if retried.logbook != "test" {
		t.Errorf("retry logbook = %q, want %q — the originating logbook must be carried", retried.logbook, "test")
	}
	stored, err := store.GetQSOByID(m.App.DB, qID)
	if err != nil {
		t.Fatalf("GetQSOByID after retry: %v", err)
	}
	if stored.WavelogID != 42 {
		t.Errorf("stored wavelog_id = %d, want 42 (the accepted contact's id)", stored.WavelogID)
	}
	if !stored.WavelogDirty {
		t.Fatal("row must be durably dirty — the server holds the older snapshot")
	}
}

// TestUploadLeaseBridgesReconciliationAcrossLogbookSwitch reproduces the
// reported failure: the upload worker released its database lease before
// returning its result, and the completion handler acquired a NEW lease for
// the reconciliation chain — switching logbooks while the upload was pending
// closed the retired database at the worker's release, and the
// reconciliation then received a closed handle ("sql: database is closed").
// One lease must cover the whole upload → reconciliation chain, transferred
// through the completion message and released only when the chain drains.
func TestUploadLeaseBridgesReconciliationAcrossLogbookSwitch(t *testing.T) {
	postStarted := make(chan struct{})
	releasePOST := make(chan struct{})
	var mu sync.Mutex
	var patchedComment string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/qso":
			close(postStarted)
			<-releasePOST
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": 42, "call": "SP9AAA"},
				"meta": map[string]string{"resource": "qso", "method": "POST"},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/qso/42":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode patch body: %v", err)
			}
			mu.Lock()
			patchedComment, _ = body["comment"].(string)
			mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	dbPathA := filepath.Join(dir, "a.db")
	dbPathB := filepath.Join(dir, "b.db")

	cfg := config.DefaultConfig()
	lbA := config.Logbook{
		Station:      config.Station{Callsign: "SP9A", Grid: "JO90"},
		DatabasePath: dbPathA,
		Wavelog:      &config.WavelogConfig{Enabled: true, URL: srv.URL, APIKey: "wl2_test", StationProfileID: "1"},
	}
	lbB := config.Logbook{Station: config.Station{Callsign: "SP9B", Grid: "JO91"}, DatabasePath: dbPathB}
	cfg.Logbooks = map[string]config.Logbook{"a": lbA, "b": lbB}
	cfg.State.ActiveLogbook = "a"
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	dbA, err := store.InitDB(dbPathA)
	if err != nil {
		t.Fatalf("init db A: %v", err)
	}
	a := &app.App{
		Config:      cfg,
		ConfigPath:  cfgPath,
		LogbookName: "a",
		Logbook:     &lbA,
		DB:          dbA,
		DBPath:      dbPathA,
	}
	t.Cleanup(a.StopAPRSTimer)

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "old"}
	id, err := store.InsertQSO(dbA, q1)
	if err != nil {
		t.Fatalf("InsertQSO A: %v", err)
	}

	m := New(a, nil)
	m.inetOnline = true

	qs, err := store.GetQSOByID(dbA, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	resCh := make(chan wlUploadResultMsg, 1)
	go func() { resCh <- execCmd(m.uploadQSOToWavelog(qs)).(wlUploadResultMsg) }()

	select {
	case <-postStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("upload POST never started")
	}

	// A newer edit lands while the upload is on the wire.
	row, err := store.GetQSOByID(dbA, id)
	if err != nil {
		t.Fatalf("GetQSOByID before edit: %v", err)
	}
	row.Comment = "newer"
	if err := store.UpdateQSO(dbA, row); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}

	// Switch logbooks while the upload is still pending — dbA is retired and
	// its close deferred until the last lease holder releases.
	if err := a.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { a.DB.Close() })

	close(releasePOST)
	res := <-resCh
	if !res.ok || res.remoteID != 42 {
		t.Fatalf("upload: ok=%v remoteID=%d, want accepted id 42", res.ok, res.remoteID)
	}
	if !res.changed {
		t.Fatal("changed must be true — the row was edited while the upload was on the wire")
	}
	if res.release == nil {
		t.Fatal("the completion must carry the transferred database lease")
	}

	// The completion dispatches the reconciliation chain — with the
	// transferred lease the retired database stays open; on the old code the
	// worker already released it, the database closed, and the PATCH failed
	// with "sql: database is closed".
	upd, c := m.Update(res)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the reconciliation PATCH was not queued")
	}
	em, ok := execCmd(c).(editorMsg)
	if !ok || !em.wlSyncFollowUp {
		t.Fatalf("expected the follow-up PATCH result, got %T", em)
	}
	if !em.wlSyncOK {
		t.Fatalf("follow-up PATCH: ok=%v err=%q", em.wlSyncOK, em.wlSyncErr)
	}
	mu.Lock()
	got := patchedComment
	mu.Unlock()
	if got != "newer" {
		t.Errorf("server PATCH comment = %q, want newer", got)
	}
	if c2 := m.handleQSOSyncCompletion(em); c2 != nil {
		t.Fatal("no further PATCH should be dispatched")
	}

	// The chain lease was released at chain end: the retired db is closed.
	if err := dbA.Ping(); err == nil {
		t.Error("retired A database should be closed after the chain lease released")
	}
	// A's row carries the newest edit and is clean after the follow-up ack.
	reopened, err := store.Open(dbPathA)
	if err != nil {
		t.Fatalf("reopen A: %v", err)
	}
	defer reopened.Close()
	stored, err := store.GetQSOByID(reopened, id)
	if err != nil {
		t.Fatalf("GetQSOByID A after chain: %v", err)
	}
	if stored.Comment != "newer" {
		t.Errorf("A comment = %q, want newer", stored.Comment)
	}
	if stored.WavelogID != 42 {
		t.Errorf("A WavelogID = %d, want 42", stored.WavelogID)
	}
	if stored.WavelogDirty {
		t.Error("A's row should be clean after the follow-up ack")
	}
}

// TestUploadLeaseBridgesIDRetryAcrossLogbookSwitch covers the retry hop of
// the chain: the local id write fails, the operator switches logbooks, and
// the id-attach retry must still reach the retired (but leased) database.
// The lease transfers through the retried completion and is released when
// the chain finishes.
func TestUploadLeaseBridgesIDRetryAcrossLogbookSwitch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 42}})
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 42, "call": "SP9AAA", "band": "20m", "mode": "SSB",
						"qso_date": "2024-05-01 12:00:00"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	dbPathA := filepath.Join(dir, "a.db")
	dbPathB := filepath.Join(dir, "b.db")

	cfg := config.DefaultConfig()
	lbA := config.Logbook{
		Station:      config.Station{Callsign: "SP9A", Grid: "JO90"},
		DatabasePath: dbPathA,
		Wavelog:      &config.WavelogConfig{Enabled: true, URL: srv.URL, APIKey: "wl2_test", StationProfileID: "1"},
	}
	lbB := config.Logbook{Station: config.Station{Callsign: "SP9B", Grid: "JO91"}, DatabasePath: dbPathB}
	cfg.Logbooks = map[string]config.Logbook{"a": lbA, "b": lbB}
	cfg.State.ActiveLogbook = "a"
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	dbA, err := store.InitDB(dbPathA)
	if err != nil {
		t.Fatalf("init db A: %v", err)
	}
	a := &app.App{
		Config:      cfg,
		ConfigPath:  cfgPath,
		LogbookName: "a",
		Logbook:     &lbA,
		DB:          dbA,
		DBPath:      dbPathA,
	}
	t.Cleanup(a.StopAPRSTimer)

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "kept"}
	id, err := store.InsertQSO(dbA, q1)
	if err != nil {
		t.Fatalf("InsertQSO A: %v", err)
	}

	m := New(a, nil)
	m.inetOnline = true

	qs, err := store.GetQSOByID(dbA, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	// Deterministically reject the local id write.
	if _, err := dbA.Exec(`CREATE TRIGGER block_wl_update BEFORE UPDATE OF wavelog_id ON qsos
		WHEN OLD.wavelog_id = 0 AND NEW.wavelog_id != 0
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	res := execCmd(m.uploadQSOToWavelog(qs)).(wlUploadResultMsg)
	if !res.ok || !res.unresolved {
		t.Fatalf("upload: ok=%v unresolved=%v, want accepted-but-unresolved", res.ok, res.unresolved)
	}
	if res.release == nil {
		t.Fatal("the completion must carry the transferred database lease")
	}
	if _, err := dbA.Exec(`DROP TRIGGER block_wl_update`); err != nil {
		t.Fatalf("drop trigger: %v", err)
	}

	// Switch logbooks — the transferred lease keeps the retired database
	// open for the id-attach retry.
	if err := a.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { a.DB.Close() })

	upd, c := m.Update(res)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the id-attach retry was not queued")
	}
	var retried wlUploadResultMsg
	var walk func(tea.Cmd)
	walk = func(sub tea.Cmd) {
		if retried.retried {
			return
		}
		if r, ok := sub().(wlUploadResultMsg); ok && r.retried {
			retried = r
			return
		}
		if inner, ok := execCmd(sub).(tea.BatchMsg); ok {
			for _, nested := range inner {
				walk(nested)
			}
		}
	}
	batch, isBatch := execCmd(c).(tea.BatchMsg)
	if isBatch {
		for _, sub := range batch {
			walk(sub)
		}
	} else {
		walk(c)
	}
	if !retried.retried {
		t.Fatal("the retry result was not among the dispatched commands")
	}
	if !retried.ok || retried.unresolved || retried.remoteID != 42 {
		t.Fatalf("retry: ok=%v unresolved=%v remoteID=%d, want id 42 attached", retried.ok, retried.unresolved, retried.remoteID)
	}

	// The retried completion flows back through the handler — with no
	// follow-up it releases the transferred lease, closing the retired db.
	upd2, _ := m.Update(retried)
	m = upd2.(*Model)
	if err := dbA.Ping(); err == nil {
		t.Error("retired A database should be closed after the retry chain finished")
	}
	reopened, err := store.Open(dbPathA)
	if err != nil {
		t.Fatalf("reopen A: %v", err)
	}
	defer reopened.Close()
	stored, err := store.GetQSOByID(reopened, id)
	if err != nil {
		t.Fatalf("GetQSOByID A after retry: %v", err)
	}
	if stored.WavelogID != 42 {
		t.Errorf("A WavelogID = %d, want 42", stored.WavelogID)
	}
}
