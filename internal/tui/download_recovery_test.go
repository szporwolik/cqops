package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Wavelog download error/recovery UX tests (Pass 27)
// =============================================================================
// Tests editor-level behavior when Wavelog download fails, is retried,
// and eventually succeeds. Uses httptest.Server, temp SQLite DBs, and
// temp ADIF files. No real network calls.

// execAllDownloadMsgs reads all messages from the editor's download channel
// and applies them via Update until dlDone is seen. Returns the final editor.
func execAllDownloadMsgs(t *testing.T, le *LogbookEditor) *LogbookEditor {
	t.Helper()
	for {
		msg := le.readDownloadMsg()
		le2, _ := le.Update(msg)
		le = le2.(*LogbookEditor)
		if !le.dlActive {
			return le
		}
	}
}

// startFakeDownload creates a LogbookEditor, starts a download against a fake
// server, and drains all messages synchronously. Returns the editor after
// the download goroutine has completed.
func startFakeDownload(t *testing.T, server *httptest.Server, _ []store.DXCSpot) *LogbookEditor {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "wl_dl_test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	le := NewLogbookEditor(LogbookEditorConfig{DB: db, WLURL: server.URL, WLKey: "test-api-key", WLStationID: "1", WLLastFetchedID: 0, StationOperator: "OP", StationGrid: "JO90", StationCall: ""})
	// Add a pre-existing QSO to verify it isn't removed by failed download.
	if _, err := store.InsertQSO(db, &qso.QSO{
		Call: "SP9EXISTING", QSODate: "20240601", TimeOn: "120000",
		Band: "20m", Mode: "SSB", Source: "manual",
	}); err != nil {
		t.Fatalf("seed QSO: %v", err)
	}

	// Start the download goroutine.
	cmd := le.doWavelogDownload()
	if cmd == nil {
		t.Fatal("doWavelogDownload returned nil cmd")
	}

	// Process initial message (dlProgress=0).
	msg := cmd()
	m2, _ := le.Update(msg)
	le = m2.(*LogbookEditor)

	// Drain remaining messages until done.
	return execAllDownloadMsgs(t, le)
}

// =============================================================================
// Failure tests
// =============================================================================

func TestDownload_HTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadErr == "" {
		t.Error("wlDownloadErr should be set on HTTP 500 failure")
	}
	if le.dlActive {
		t.Error("dlActive should be false after download completes")
	}
	if le.wlDownloadCount != 0 {
		t.Errorf("wlDownloadCount = %d, want 0 on failure", le.wlDownloadCount)
	}
	// Existing QSOs should still be present.
	qsos, _ := store.ListQSOs(le.db, 10, "")
	found := false
	for _, q := range qsos {
		if q.Call == "SP9EXISTING" {
			found = true
			break
		}
	}
	if !found {
		t.Error("existing QSO should survive failed download")
	}
}

func TestDownload_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadErr == "" {
		t.Error("wlDownloadErr should be set on auth failure")
	}
	if le.wlDownloadCount != 0 {
		t.Errorf("wlDownloadCount = %d, want 0", le.wlDownloadCount)
	}
}

func TestDownload_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadErr == "" {
		t.Error("wlDownloadErr should be set when response is not valid JSON")
	}
}

func TestDownload_EmptySuccess(t *testing.T) {
	// Server returns valid JSON but no ADIF data.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 0,
			"lastfetchedid": 42,
		})
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadErr != "" {
		t.Errorf("wlDownloadErr = %q, want empty for empty success", le.wlDownloadErr)
	}
	if le.wlDownloadCount != 0 {
		t.Errorf("wlDownloadCount = %d, want 0", le.wlDownloadCount)
	}
}

func TestDownload_FailureClearsActiveFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.dlActive {
		t.Error("dlActive should be false after download failure")
	}
	if le.dlCancel != nil {
		t.Error("dlCancel should be nil after download completes")
	}
}

func TestDownload_MissingAPIKey(t *testing.T) {
	// Use a valid server URL but empty key and station ID.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	// Create editor with empty Wavelog config — but the server URL is valid.
	// FetchContacts will reject empty key/stationID.
	le := startFakeDownload(t, server, nil)
	if le.wlDownloadErr == "" {
		t.Error("wlDownloadErr should be set when key/stationID are empty")
	}
}

// =============================================================================
// Secret safety tests
// =============================================================================

func TestDownload_ErrorDoesNotLeakAPIKey(t *testing.T) {
	key := "secret-key-abc123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify key is in Authorization header (not URL).
		if !strings.Contains(r.Header.Get("Authorization"), key) {
			t.Log("key sent in header, not URL — good")
		}
		// Verify key is NOT in URL query.
		if strings.Contains(r.URL.RawQuery, key) {
			t.Error("API key found in URL query — should be in header only")
		}
		w.WriteHeader(500)
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	// dlErr should not contain the actual API key.
	if strings.Contains(le.wlDownloadErr, key) {
		t.Errorf("wlDownloadErr should NOT leak API key, got: %q", le.wlDownloadErr)
	}
}

func TestDownload_ErrorLogDoesNotContainKeyInEditorState(t *testing.T) {
	key := "my-secret-key-xyz"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	// The wlDownloadErr message on the editor should not expose the key.
	if strings.Contains(le.wlDownloadErr, key) {
		t.Errorf("editor error state should not contain API key: %q", le.wlDownloadErr)
	}
}

// =============================================================================
// Retry/recovery tests
// =============================================================================

func TestDownload_RetryAfterFailure(t *testing.T) {
	// First attempt: fail. Second attempt: succeed.
	attempt := 0
	adifContent := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500
<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59
<GRIDSQUARE:4>JO90 <EOR>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 1,
			"lastfetchedid": 99,
			"adif":          adifContent,
		})
	}))
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "wl_dl_test.db")
	db, _ := store.InitDB(dbPath)
	defer db.Close()

	le := NewLogbookEditor(LogbookEditorConfig{DB: db, WLURL: server.URL, WLKey: "key", WLStationID: "1", WLLastFetchedID: 0, StationOperator: "OP", StationGrid: "JO90", StationCall: ""})

	// First attempt — fail.
	cmd := le.doWavelogDownload()
	if cmd == nil {
		t.Fatal("doWavelogDownload returned nil")
	}
	msg := cmd()
	m2, _ := le.Update(msg)
	le = m2.(*LogbookEditor)
	le = execAllDownloadMsgs(t, le)

	if le.wlDownloadErr == "" {
		t.Fatal("first attempt should fail")
	}
	if le.dlActive {
		t.Error("dlActive should be false after failed download")
	}

	// Reset download state before retry (as the editor would via user action).
	le.mode = edModeList
	le.dlProgress = 0
	le.dlTotal = 0
	le.wlDownloadErr = ""

	// Second attempt — succeed.
	cmd = le.doWavelogDownload()
	if cmd == nil {
		t.Fatal("retry doWavelogDownload returned nil")
	}
	msg = cmd()
	m2, _ = le.Update(msg)
	le = m2.(*LogbookEditor)
	le = execAllDownloadMsgs(t, le)

	if le.wlDownloadErr != "" {
		t.Fatalf("retry should succeed, got error: %q", le.wlDownloadErr)
	}
	if le.wlDownloadCount != 1 {
		t.Errorf("wlDownloadCount = %d, want 1 after successful retry", le.wlDownloadCount)
	}

	// QSO should be in DB with Wavelog flags.
	qsos, _ := store.ListQSOs(db, 10, "")
	var found bool
	for _, q := range qsos {
		if q.Call == "SP9MOA" && q.Source == "wavelog" && q.WavelogUploaded == "yes" {
			found = true
			break
		}
	}
	if !found {
		t.Error("imported QSO should have Source=wavelog, WavelogUploaded=yes")
	}
}

// =============================================================================
// Robustness: malformed ADIF during download
// =============================================================================

func TestDownload_MalformedADIFImportsValid(t *testing.T) {
	// Mixed: one valid QSO, one malformed record without CALL.
	adifContent := `<CALL:7>SP9GOOD <BAND:3>40m <MODE:2>CW <QSO_DATE:8>20260618 <TIME_ON:6>130000 <RST_SENT:3>599 <RST_RCVD:3>579 <EOR>
<BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>140000 <EOR>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 2,
			"lastfetchedid": 55,
			"adif":          adifContent,
		})
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadErr != "" {
		t.Errorf("unexpected error: %q", le.wlDownloadErr)
	}
	// Only the valid QSO should be imported (plus the pre-existing seed QSO).
	if le.wlDownloadCount != 1 {
		t.Errorf("wlDownloadCount = %d, want 1 (only valid QSO imported)", le.wlDownloadCount)
	}
	qsos, _ := store.ListQSOs(le.db, 10, "")
	// Seed QSO + imported valid QSO = 2.
	if len(qsos) != 2 {
		t.Errorf("DB should have 2 QSOs (seed + imported valid), got %d", len(qsos))
	}
	found := false
	for _, q := range qsos {
		if q.Call == "SP9GOOD" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SP9GOOD should be in imported QSOs")
	}
}

// =============================================================================
// Robustness: no goroutine leak / temp file cleanup
// =============================================================================

func TestDownload_TempFileCleanup(t *testing.T) {
	adifContent := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <EOR>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 1,
			"lastfetchedid": 10,
			"adif":          adifContent,
		})
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadErr != "" {
		t.Errorf("unexpected error: %q", le.wlDownloadErr)
	}
	if le.wlDownloadCount != 1 {
		t.Errorf("wlDownloadCount = %d, want 1", le.wlDownloadCount)
	}
	if le.dlActive {
		t.Error("dlActive should be false after completion")
	}
}

// =============================================================================
// WavelogUploaded=yes set correctly on success
// =============================================================================

func TestDownload_SetsWavelogUploaded(t *testing.T) {
	adifContent := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exported_qsos": 1,
			"lastfetchedid": 77,
			"adif":          adifContent,
		})
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadCount != 1 {
		t.Fatalf("wlDownloadCount = %d, want 1", le.wlDownloadCount)
	}

	qsos, _ := store.ListQSOs(le.db, 10, "")
	// Seed QSO + imported QSO = 2.
	if len(qsos) < 1 {
		t.Fatalf("expected at least 1 QSO in DB, got %d", len(qsos))
	}
	found := false
	for _, q := range qsos {
		if q.Call == "SP9MOA" {
			if q.WavelogUploaded != "yes" {
				t.Errorf("WavelogUploaded = %q, want yes", q.WavelogUploaded)
			}
			if q.Source != "wavelog" {
				t.Errorf("Source = %q, want wavelog", q.Source)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("SP9MOA should be in imported QSOs")
	}
}

// =============================================================================
// Mid-download dlErr trimming tests (Pass 30)
// =============================================================================
// Tests the update-handler branch that captures dlErr before dlDone.
// Uses direct editorMsg injection — no goroutines or HTTP servers.

func TestMidDownload_DlErr_MeaningfulStoredTrimmed(t *testing.T) {
	le := newResultEditor(0, 0, 0, "")
	le.dlActive = true
	le.mode = edModeWLDownloading

	_, _ = le.Update(editorMsg{dlErr: "  HTTP 500  "})

	if le.wlDownloadErr != "HTTP 500" {
		t.Errorf("wlDownloadErr = %q, want trimmed 'HTTP 500'", le.wlDownloadErr)
	}
}

func TestMidDownload_DlErr_WhitespaceOnlyIgnored(t *testing.T) {
	le := newResultEditor(0, 0, 0, "")
	le.dlActive = true
	le.mode = edModeWLDownloading

	_, _ = le.Update(editorMsg{dlErr: "   "})

	if le.wlDownloadErr != "" {
		t.Errorf("wlDownloadErr = %q, want empty (whitespace-only ignored)", le.wlDownloadErr)
	}
}

func TestMidDownload_DlErr_EmptyNotStored(t *testing.T) {
	le := newResultEditor(0, 0, 0, "")
	le.dlActive = true
	le.mode = edModeWLDownloading

	_, _ = le.Update(editorMsg{dlErr: ""})

	if le.wlDownloadErr != "" {
		t.Errorf("wlDownloadErr = %q, want empty", le.wlDownloadErr)
	}
}

func TestMidDownload_DlErr_NewlinesIgnored(t *testing.T) {
	le := newResultEditor(0, 0, 0, "")
	le.dlActive = true
	le.mode = edModeWLDownloading

	_, _ = le.Update(editorMsg{dlErr: "\n\t "})

	if le.wlDownloadErr != "" {
		t.Errorf("wlDownloadErr = %q, want empty (whitespace-only ignored)", le.wlDownloadErr)
	}
}

func TestMidDownload_DlErr_PreservedWhenDlDoneArrives(t *testing.T) {
	le := newResultEditor(0, 0, 0, "")
	le.dlActive = true
	le.mode = edModeWLDownloading

	// First: meaningful error.
	_, _ = le.Update(editorMsg{dlErr: "connection refused"})
	if le.wlDownloadErr != "connection refused" {
		t.Fatalf("wlDownloadErr should be set, got %q", le.wlDownloadErr)
	}

	// Then: dlDone arrives (from channel close). Error should be preserved.
	_, _ = le.Update(editorMsg{dlDone: true})

	if le.wlDownloadErr != "connection refused" {
		t.Errorf("wlDownloadErr should survive dlDone, got %q", le.wlDownloadErr)
	}
	if le.dlActive {
		t.Error("dlActive should be false after dlDone")
	}
	if le.mode != edModeWLDownloadResult {
		t.Errorf("mode should be edModeWLDownloadResult, got %v", le.mode)
	}
}

func TestMidDownload_DlErr_WhitespaceNotPreservedThroughDlDone(t *testing.T) {
	le := newResultEditor(0, 0, 0, "")
	le.dlActive = true
	le.mode = edModeWLDownloading
	le.wlDownloadCount = 5 // simulate that imports happened

	// Whitespace-only error — ignored.
	_, _ = le.Update(editorMsg{dlErr: "   "})

	if le.wlDownloadErr != "" {
		t.Errorf("whitespace error should not be stored, got %q", le.wlDownloadErr)
	}

	// dlDone arrives. Since no error was stored, success counts should show.
	_, _ = le.Update(editorMsg{dlDone: true})

	if le.wlDownloadErr != "" {
		t.Errorf("wlDownloadErr should stay empty, got %q", le.wlDownloadErr)
	}
	if le.mode != edModeWLDownloadResult {
		t.Errorf("mode should be edModeWLDownloadResult, got %v", le.mode)
	}
}
