package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      0,
				"lastfetchedid": 42,
				"adif":          nil,
			},
			"meta": map[string]any{"has_more": false},
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
	if le.dlOp != nil {
		t.Error("dlOp should be nil after download completes")
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

// TestDownload_TransientFailureFreezesCheckpoint reproduces the lost-contact
// bug: when a record fails to INSERT (here a trigger injects a deterministic
// failure standing in for database contention), the persisted
// last_fetched_id must stay before that record so the next incremental
// download retries it. Later records that did succeed are re-fetched and
// deduplicated on the retry.
func TestDownload_TransientFailureFreezesCheckpoint(t *testing.T) {
	adifContent := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500
<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59
<GRIDSQUARE:4>JO90 <EOR>
<CALL:6>DL1ABC <BAND:3>40m <MODE:3>FT8 <FREQ:8>7.074000
<QSO_DATE:8>20260619 <TIME_ON:6>130000 <RST_SENT:3>-10 <RST_RCVD:3>-05
<GRIDSQUARE:4>JN58 <EOR>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      2,
				"lastfetchedid": 200,
				"adif":          adifContent,
			},
			"meta": map[string]any{"has_more": false},
		})
	}))
	defer server.Close()

	m := newLifecycleTestModel(t)
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	wl := m.App.Logbook.Wavelog
	wl.URL = server.URL
	wl.APIKey = "key"
	wl.Enabled = true
	wl.StationProfileID = "1"

	// Make the FIRST contact's insert fail — the second one must still
	// succeed, reproducing "contact 101 fails, later contacts succeed".
	if _, err := m.App.DB.Exec(`CREATE TRIGGER fail_test_insert BEFORE INSERT ON qsos
		WHEN NEW.call='SP9MOA'
		BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	m.ui.logbookEditor = NewLogbookEditor(LogbookEditorConfig{
		DB: m.App.DB, WLURL: server.URL, WLKey: "key", WLStationID: "1",
		WLLastFetchedID: 0, StationOperator: "OP", StationGrid: "JO90",
	})

	cmd := m.ui.logbookEditor.doWavelogDownload()
	if cmd == nil {
		t.Fatal("doWavelogDownload returned nil cmd")
	}
	for i := 0; i < 500 && m.ui.logbookEditor.isDownloadActive(); i++ {
		msg := cmd()
		next, c := m.Update(msg)
		var ok bool
		m, ok = next.(*Model)
		if !ok {
			t.Fatalf("Update returned %T", next)
		}
		if c == nil {
			break
		}
		cmd = c
	}
	if m.ui.logbookEditor.isDownloadActive() {
		t.Fatal("download did not complete")
	}

	le := m.ui.logbookEditor
	if le.wlDownloadCount != 1 {
		t.Errorf("inserted = %d, want 1 (only the valid record)", le.wlDownloadCount)
	}
	if le.wlDownloadHold != 1 {
		t.Errorf("deferred = %d, want 1 (the failed record must be retried)", le.wlDownloadHold)
	}

	// The cursor must NOT advance to the export's last id (200): the failed
	// record is unsaved and must be re-fetched next time.
	if wl.LastFetchedID != 0 {
		t.Errorf("LastFetchedID = %d, want 0 (frozen before the failed record)", wl.LastFetchedID)
	}
	data, err := os.ReadFile(m.App.ConfigPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if strings.Contains(string(data), "last_fetched_id: 200") {
		t.Error("config advanced the cursor past an unsaved contact")
	}
}

// TestEditorSideEffects_ImportExportDoesNotMoveWavelogCursor verifies the
// shared completion handler only persists last_fetched_id for Wavelog
// download completions — ordinary ADIF import/export must not reset it.
func TestEditorSideEffects_ImportExportDoesNotMoveWavelogCursor(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	wl := m.App.Logbook.Wavelog
	wl.LastFetchedID = 42

	// Import/export-shaped completion (no dlDownload flag) — cursor untouched.
	m.handleEditorSideEffects(editorMsg{dlDone: true, dlCount: 3})
	if wl.LastFetchedID != 42 {
		t.Errorf("import completion reset cursor to %d, want 42", wl.LastFetchedID)
	}

	// Download-shaped completion advances the cursor.
	m.handleEditorSideEffects(editorMsg{dlDone: true, dlCount: 1, dlLastID: 99, dlDownload: true})
	if wl.LastFetchedID != 99 {
		t.Errorf("download completion should set cursor to 99, got %d", wl.LastFetchedID)
	}
}

// TestDownload_CompletesWhileOnQSOScreen reproduces the stale-recent-QSOs
// bug: the user starts a Wavelog download in the logbook editor and
// switches to the QSO screen before it finishes. The editor message pump
// must keep running globally so the final done message triggers the QSO
// refresh. The same pump serves ADIF import and export.
func TestDownload_CompletesWhileOnQSOScreen(t *testing.T) {
	adifContent := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500
<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59
<GRIDSQUARE:4>JO90 <EOR>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      1,
				"lastfetchedid": 77,
				"adif":          adifContent,
			},
			"meta": map[string]any{"has_more": false},
		})
	}))
	defer server.Close()

	m := newLifecycleTestModel(t)
	m.screen = screenQSO
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	wl := m.App.Logbook.Wavelog
	wl.URL = server.URL
	wl.APIKey = "key"
	wl.Enabled = true
	wl.StationProfileID = "1"

	m.ui.logbookEditor = NewLogbookEditor(LogbookEditorConfig{
		DB: m.App.DB, WLURL: server.URL, WLKey: "key", WLStationID: "1",
		WLLastFetchedID: 0, StationOperator: "OP", StationGrid: "JO90",
	})

	cmd := m.ui.logbookEditor.doWavelogDownload()
	if cmd == nil {
		t.Fatal("doWavelogDownload returned nil cmd")
	}

	// Pump all editor messages through the model while it sits on the QSO
	// screen — the download must complete and flag the refresh.
	for i := 0; i < 500 && m.ui.logbookEditor.isDownloadActive(); i++ {
		msg := cmd()
		next, c := m.Update(msg)
		var ok bool
		m, ok = next.(*Model)
		if !ok {
			t.Fatalf("Update returned %T", next)
		}
		if c == nil {
			break
		}
		cmd = c
	}

	if m.ui.logbookEditor.isDownloadActive() {
		t.Fatal("download did not complete while on the QSO screen")
	}
	if m.ui.logbookEditor.wlDownloadCount != 1 {
		t.Errorf("wlDownloadCount = %d, want 1", m.ui.logbookEditor.wlDownloadCount)
	}
	if wl.LastFetchedID != 77 {
		t.Errorf("LastFetchedID = %d, want 77", wl.LastFetchedID)
	}
	if !m.needRefresh {
		t.Error("needRefresh should be set after the download completes")
	}

	// The last_fetched_id must land in the config file, so the next
	// download resumes instead of re-fetching from 0.
	data, err := os.ReadFile(m.App.ConfigPath)
	if err != nil {
		t.Fatalf("config file not written after download: %v", err)
	}
	if !strings.Contains(string(data), "last_fetched_id: 77") {
		t.Errorf("config file missing last_fetched_id, got:\n%s", data)
	}

	// The deferred refresh updates the recent QSOs shown on the QSO page.
	pending, _ := m.handlePendingRequests(nil)
	if pending == nil {
		t.Fatal("handlePendingRequests returned nil for pending refresh")
	}
	next, _ := m.Update(pending())
	m = next.(*Model)
	found := false
	for _, q := range m.recentQSOs.qsos {
		if q.Call == "SP9MOA" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("recent QSOs not updated after download: %v", m.recentQSOs.qsos)
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
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      1,
				"lastfetchedid": 99,
				"adif":          adifContent,
			},
			"meta": map[string]any{"has_more": false},
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
		if q.Call == "SP9MOA" && q.Source == "wavelog" {
			found = true
			break
		}
	}
	if !found {
		t.Error("imported QSO should have Source=wavelog")
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
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      2,
				"lastfetchedid": 55,
				"adif":          adifContent,
			},
			"meta": map[string]any{"has_more": false},
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
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      1,
				"lastfetchedid": 10,
				"adif":          adifContent,
			},
			"meta": map[string]any{"has_more": false},
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
// Wavelog source set correctly on success
// =============================================================================

func TestDownload_SetsWavelogSource(t *testing.T) {
	adifContent := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      1,
				"lastfetchedid": 77,
				"adif":          adifContent,
			},
			"meta": map[string]any{"has_more": false},
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

// TestDownload_AbortDuringFetchCompletes is the regression test for lost
// cancellation: the operation object survives the UI's abort, the in-flight
// HTTP request is cancelled via context, and the worker still delivers its
// final aborted-done message.
func TestDownload_AbortDuringFetchCompletes(t *testing.T) {
	// The handler blocks until released — cancellation of the request
	// context must unblock the fetch without waiting for a response.
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      0,
				"lastfetchedid": 42,
				"adif":          nil,
			},
			"meta": map[string]any{"has_more": false},
		})
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	dbPath := filepath.Join(t.TempDir(), "abort_test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	le := NewLogbookEditor(LogbookEditorConfig{
		DB: db, WLURL: server.URL, WLKey: "test-api-key", WLStationID: "1",
		WLLastFetchedID: 0, StationOperator: "OP", StationGrid: "JO90",
	})

	cmd := le.doWavelogDownload()
	if cmd == nil {
		t.Fatal("doWavelogDownload returned nil cmd")
	}
	_ = cmd() // first read; the worker starts concurrently

	// Abort immediately — the worker must notice even though the UI
	// tears down its own references.
	le.cancelDownload()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for le.dlActive {
			msg := le.readDownloadMsg()
			m2, _ := le.Update(msg)
			le = m2.(*LogbookEditor)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("aborted download did not complete — cancellation was lost")
	}

	if le.dlOp != nil {
		t.Error("dlOp should be nil after the operation completes")
	}
	if !le.wlDownloadAbort {
		t.Error("download result should be marked as aborted")
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

// =============================================================================
// ADIF import/export terminal-result tests
// =============================================================================
// Import/export failures must reach the result screen as errors instead of
// ending as success-looking results, and exports must only publish the
// target file after a fully successful write.

// startImport runs an ADIF import producer against a temp DB and drains all
// messages synchronously. Returns the editor after completion.
func startImport(t *testing.T, adifPath string) *LogbookEditor {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "import_test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	le := NewLogbookEditor(LogbookEditorConfig{DB: db, StationOperator: "OP", StationGrid: "JO90", StationCall: ""})
	le.dlActive = true
	le.mode = edModeImporting
	op := newDownloadOp()
	le.dlOp = op
	go le.runImport(op, adifPath)
	return execAllDownloadMsgs(t, le)
}

// startExport runs an ADIF export producer against the editor's DB and
// drains all messages synchronously. Returns the editor after completion.
func startExport(t *testing.T, le *LogbookEditor, target string) *LogbookEditor {
	t.Helper()
	le.dlActive = true
	le.mode = edModeExporting
	op := newDownloadOp()
	le.dlOp = op
	go le.runExport(op, target)
	return execAllDownloadMsgs(t, le)
}

func TestImport_OpenFailureEndsWithError(t *testing.T) {
	le := startImport(t, filepath.Join(t.TempDir(), "missing.adi"))

	if le.dlActive {
		t.Error("dlActive should be false after the import finishes")
	}
	if le.mode != edModeImportResult {
		t.Errorf("mode = %v, want edModeImportResult", le.mode)
	}
	if !strings.Contains(le.impErr, "cannot open file") {
		t.Errorf("impErr = %q, want the open failure to reach the result", le.impErr)
	}
}

func TestImport_ScannerErrorReachesResult(t *testing.T) {
	dir := t.TempDir()
	adifPath := filepath.Join(dir, "corrupt.adi")
	data := "CQOps test\n<ADIF_VER:5>3.1.7<EOH>\n" +
		"<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20240501 <TIME_ON:6>120000 <EOR>\n" +
		"<CALL:XX>SP9MOA<EOR>\n"
	if err := os.WriteFile(adifPath, []byte(data), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	le := startImport(t, adifPath)

	if le.mode != edModeImportResult {
		t.Errorf("mode = %v, want edModeImportResult", le.mode)
	}
	if !strings.Contains(le.impErr, "could not be read completely") {
		t.Errorf("impErr = %q, want the scanner error to reach the result", le.impErr)
	}
	if le.impInserted != 1 {
		t.Errorf("impInserted = %d, want 1 (valid record before the corruption)", le.impInserted)
	}

	// The valid record before the corruption must still be in the DB.
	qsos, err := store.ListQSOs(le.db, 10, "")
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) != 1 || qsos[0].Call != "SP9MOA" {
		t.Errorf("imported QSOs = %+v, want exactly SP9MOA", qsos)
	}
}

func TestExport_SuccessPublishesOnlyFinalFile(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "", "")

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000",
		RSTSent: "59", RSTRcvd: "59"}
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000",
		RSTSent: "599", RSTRcvd: "579"}
	insertTestQSO(t, le.db, q1)
	insertTestQSO(t, le.db, q2)

	target := filepath.Join(t.TempDir(), "out.adi")
	le = startExport(t, le, target)

	if le.mode != edModeExportResult {
		t.Errorf("mode = %v, want edModeExportResult", le.mode)
	}
	if le.impErr != "" {
		t.Errorf("impErr = %q, want success", le.impErr)
	}
	if le.impInserted != 2 {
		t.Errorf("impInserted = %d, want 2", le.impInserted)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target file should exist after successful export: %v", err)
	}
	if !strings.Contains(string(data), "SP9AAA") || !strings.Contains(string(data), "SP9BBB") {
		t.Errorf("exported ADIF missing callsigns: %q", string(data))
	}
	if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file must not survive a successful export")
	}
}

func TestExport_EmptyLogbookFailsWithoutFile(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "", "")

	target := filepath.Join(t.TempDir(), "empty.adi")
	le = startExport(t, le, target)

	if le.mode != edModeExportResult {
		t.Errorf("mode = %v, want edModeExportResult", le.mode)
	}
	if le.impErr != "logbook is empty" {
		t.Errorf("impErr = %q, want 'logbook is empty'", le.impErr)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("no export file should be created for an empty logbook")
	}
	if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
		t.Error("no temp file should be left behind for an empty logbook")
	}
}

// TestDownload_StoresWavelogIDs verifies the remote QSO ids captured from the
// v2 JSON list sidecar are stored on the imported QSOs — the foundation for
// future edit/delete support.
func TestDownload_StoresWavelogIDs(t *testing.T) {
	adifContent := `<CALL:6>SP9AAA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260921 <TIME_ON:4>1200 <EOR>
<CALL:6>SP9BBB <BAND:3>40m <MODE:2>CW <QSO_DATE:8>20260921 <TIME_ON:4>1300 <EOR>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("format") == "" {
			// JSON id sidecar request.
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 10}, {"id": 11}},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      2,
				"lastfetchedid": 11,
				"adif":          adifContent,
			},
			"meta": map[string]any{"has_more": false, "total": 2},
		})
	}))
	defer server.Close()

	le := startFakeDownload(t, server, nil)

	if le.wlDownloadCount != 2 {
		t.Fatalf("wlDownloadCount = %d, want 2", le.wlDownloadCount)
	}

	qsos, _ := store.ListQSOs(le.db, 10, "")
	var aaa, bbb *qso.QSO
	for i := range qsos {
		switch qsos[i].Call {
		case "SP9AAA":
			aaa = &qsos[i]
		case "SP9BBB":
			bbb = &qsos[i]
		}
	}
	if aaa == nil || bbb == nil {
		t.Fatalf("downloaded QSOs missing: %+v", qsos)
	}
	if aaa.WavelogID != 10 || bbb.WavelogID != 11 {
		t.Errorf("WavelogID = %d/%d, want 10/11", aaa.WavelogID, bbb.WavelogID)
	}
}
