package tui

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// TestWSJTXAutoLogStoresWavelogID verifies the WSJT-X auto-log → enrichment →
// upload pipeline stores the remote Wavelog id locally.
func TestWSJTXAutoLogStoresWavelogID(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 55, "call": "SP9MOA"},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:9>14.074550 <MODE:3>FT8 " +
		"<QSO_DATE:8>20260921 <TIME_ON:6>120000 <RST_SENT:3>-10 <RST_RCVD:3>-05 <GRIDSQUARE:6>JO90aa <EOR>"

	cmd, retry := m.logQSOFromADIF(adif)
	if retry {
		t.Fatal("logQSOFromADIF requested retry")
	}
	if cmd == nil {
		t.Fatal("expected upload command")
	}

	// Run the returned Batch (refresh + enrich/upload) and pump every
	// resulting message through Update.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	for _, sub := range batch {
		subMsg := sub()
		if subMsg == nil {
			continue
		}
		next, _ := m.Update(subMsg)
		m = next.(*Model)
	}

	qsos, err := store.ListQSOs(m.App.DB, 5, "")
	if err != nil || len(qsos) == 0 {
		t.Fatalf("no QSO logged: %v", err)
	}
	q := qsos[0]
	if q.WavelogID != 55 {
		t.Errorf("WavelogID = %d, want 55", q.WavelogID)
	}
	if q.WavelogID != 55 {
		t.Errorf("wavelog_id = %d, want 55", q.WavelogID)
	}
}

// TestWSJTXAutoLog_SwitchKeepsOriginalLogbookTarget verifies the captured
// operation context: when the user switches logbooks while the enrich/upload
// command is in flight, the write and upload still apply to the logbook the
// QSO was logged into, and the result carries that logbook's identity.
func TestWSJTXAutoLog_SwitchKeepsOriginalLogbookTarget(t *testing.T) {
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 55, "call": "SP9MOA"},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:9>14.074550 <MODE:3>FT8 " +
		"<QSO_DATE:8>20260921 <TIME_ON:6>120000 <RST_SENT:3>-10 <RST_RCVD:3>-05 <GRIDSQUARE:6>JO90aa <EOR>"

	cmd, retry := m.logQSOFromADIF(adif)
	if retry {
		t.Fatal("logQSOFromADIF requested retry")
	}
	if cmd == nil {
		t.Fatal("expected upload command")
	}

	// Simulate a logbook switch while the command is in flight: the active
	// database is replaced (row ids can collide with the new logbook).
	oldDB := m.App.DB
	dbB, err := store.InitDB(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("init db b: %v", err)
	}
	t.Cleanup(func() { dbB.Close() })
	m.App.DB = dbB
	m.App.LogbookName = "b"

	// Execute the captured batch now — everything must target oldDB.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	var result wlUploadResultMsg
	found := false
	for _, sub := range batch {
		subMsg := sub()
		if r, ok := subMsg.(wlUploadResultMsg); ok {
			result = r
			found = true
			continue
		}
		if subMsg == nil {
			continue
		}
		next, _ := m.Update(subMsg)
		m = next.(*Model)
	}
	if !found {
		t.Fatal("enrich/upload command did not produce wlUploadResultMsg")
	}
	if result.logbook != "test" {
		t.Errorf("result logbook = %q, want the original 'test'", result.logbook)
	}

	// The remote id must be stored in the original logbook…
	qs, err := store.GetQSOByID(oldDB, result.qID)
	if err != nil || qs == nil {
		t.Fatalf("original logbook QSO missing: %v", err)
	}
	if qs.WavelogID != 55 {
		t.Errorf("original logbook WavelogID = %d, want 55", qs.WavelogID)
	}
	// …and the new logbook must stay untouched.
	newQsos, err := store.ListAllQSOs(dbB)
	if err != nil {
		t.Fatalf("list new logbook: %v", err)
	}
	if len(newQsos) != 0 {
		t.Errorf("new logbook has %d QSOs, want 0 — upload hit the wrong database", len(newQsos))
	}

	// The handler must drop the foreign result: no new toasts, no refresh flag.
	m.needRefresh = false
	toastsBefore := len(m.toasts.Active())
	consumed, nextCmd := m.handleAsyncMessages(result)
	if !consumed {
		t.Error("foreign wlUploadResultMsg should still be consumed")
	}
	if nextCmd != nil {
		t.Error("foreign result should not trigger a refresh command")
	}
	if m.needRefresh {
		t.Error("foreign result should not set needRefresh")
	}
	if len(m.toasts.Active()) != toastsBefore {
		t.Error("foreign result should not toast")
	}
}

func TestParseWSJTXADIFValid(t *testing.T) {
	adif := "SP9MOA de DJ7NT\n" +
		"<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500 " +
		"<QSO_DATE:8>20260614 <TIME_ON:6>120000 " +
		"<RST_SENT:2>59 <RST_RCVD:2>59 <GRIDSQUARE:4>JO90 " +
		"<NAME:4>John <QTH:6>Krakow <COUNTRY:6>Poland <EOR>"

	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Call != "SP9MOA" {
		t.Errorf("Call = %q; want SP9MOA", qs.Call)
	}
	if qs.Band != "20m" {
		t.Errorf("Band = %q; want 20m", qs.Band)
	}
	if qs.Mode != "SSB" {
		t.Errorf("Mode = %q; want SSB", qs.Mode)
	}
	if qs.Freq != 14.2500 {
		t.Errorf("Freq = %f; want 14.2500", qs.Freq)
	}
	if qs.QSODate != "20260614" {
		t.Errorf("QSODate = %q; want 20260614", qs.QSODate)
	}
	if qs.TimeOn != "120000" {
		t.Errorf("TimeOn = %q; want 120000", qs.TimeOn)
	}
	if qs.RSTSent != "59" {
		t.Errorf("RSTSent = %q; want 59", qs.RSTSent)
	}
	if qs.RSTRcvd != "59" {
		t.Errorf("RSTRcvd = %q; want 59", qs.RSTRcvd)
	}
	if qs.GridSquare != "JO90" {
		t.Errorf("GridSquare = %q; want JO90", qs.GridSquare)
	}
	if qs.Name != "John" {
		t.Errorf("Name = %q; want John", qs.Name)
	}
	if qs.QTH != "Krakow" {
		t.Errorf("QTH = %q; want Krakow", qs.QTH)
	}
	if qs.Country != "Poland" {
		t.Errorf("Country = %q; want Poland", qs.Country)
	}
}

func TestParseWSJTXADIFEmpty(t *testing.T) {
	qs := parseWSJTXADIF("")
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil for empty input")
	}
	// Should return an empty QSO, not nil
	if qs.Call != "" {
		t.Errorf("Call should be empty for empty ADIF, got %q", qs.Call)
	}
}

func TestParseWSJTXADIFNoCall(t *testing.T) {
	adif := "<BAND:3>20m <MODE:3>SSB <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Call != "" {
		t.Errorf("Call should be empty, got %q", qs.Call)
	}
	if qs.Band != "20m" {
		t.Errorf("Band = %q; want 20m", qs.Band)
	}
}

func TestParseWSJTXADIFModeSubmode(t *testing.T) {
	adif := "<CALL:6>SP9MOA <MODE:4>FT8 <SUBMODE:0> <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	// FT8 is a standalone mode per ADIF 3.1.4.
	if qs.Mode != "FT8" {
		t.Errorf("Mode = %q; want FT8 (standalone)", qs.Mode)
	}
	if qs.Submode != "" {
		t.Errorf("Submode = %q; want empty", qs.Submode)
	}
}

func TestParseWSJTXADIFBandFromFreq(t *testing.T) {
	adif := "<CALL:6>SP9MOA <FREQ:7>14.2500 <MODE:3>SSB <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Band != "20m" {
		t.Errorf("Band = %q; want 20m (derived from 14.250 MHz)", qs.Band)
	}
}

func TestParseWSJTXADIFAllFields(t *testing.T) {
	adif := strings.Join([]string{
		"<CALL:6>SP9MOA",
		"<BAND:3>20m",
		"<FREQ:7>14.2500",
		"<FREQ_RX:7>14.2500",
		"<MODE:3>SSB",
		"<SUBMODE:3>USB",
		"<QSO_DATE:8>20260614",
		"<TIME_ON:6>120000",
		"<TIME_OFF:6>120500",
		"<RST_SENT:2>59",
		"<RST_RCVD:2>59",
		"<GRIDSQUARE:4>JO90",
		"<NAME:4>John",
		"<QTH:6>Krakow",
		"<COUNTRY:6>Poland",
		"<COMMENT:12>Nice contact",
		"<TX_PWR:4>100W",
		"<STATION_CALLSIGN:5>DJ7NT",
		"<OPERATOR:5>DJ7NT",
		"<MY_GRIDSQUARE:4>JO30",
		"<SOTA_REF:9>SP/TA-001",
		"<POTA_REF:7>SP-0001",
		"<WWFF_REF:9>SPFF-0001",
		"<IOTA:6>EU-001",
		"<MY_SOTA_REF:9>SP/TA-002",
		"<MY_POTA_REF:7>SP-0002",
		"<MY_WWFF_REF:9>SPFF-0002",
		"<EOR>",
	}, " ")

	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Call != "SP9MOA" {
		t.Errorf("Call = %q", qs.Call)
	}
	if qs.FreqRx != 14.2500 {
		t.Errorf("FreqRx = %f", qs.FreqRx)
	}
	if qs.Submode != "USB" {
		t.Errorf("Submode = %q", qs.Submode)
	}
	if qs.TimeOff != "120500" {
		t.Errorf("TimeOff = %q", qs.TimeOff)
	}
	if qs.Comment != "Nice contact" {
		t.Errorf("Comment = %q", qs.Comment)
	}
	if qs.TXPower != "100W" {
		t.Errorf("TXPower = %q", qs.TXPower)
	}
	if qs.StationCallsign != "DJ7NT" {
		t.Errorf("StationCallsign = %q", qs.StationCallsign)
	}
	if qs.Operator != "DJ7NT" {
		t.Errorf("Operator = %q", qs.Operator)
	}
	if qs.MyGridSquare != "JO30" {
		t.Errorf("MyGridSquare = %q", qs.MyGridSquare)
	}
	if qs.SOTARef != "SP/TA-001" {
		t.Errorf("SOTARef = %q", qs.SOTARef)
	}
	if qs.POTARef != "SP-0001" {
		t.Errorf("POTARef = %q", qs.POTARef)
	}
	if qs.IOTA != "EU-001" {
		t.Errorf("IOTA = %q", qs.IOTA)
	}
	if qs.MySOTARef != "SP/TA-002" {
		t.Errorf("MySOTARef = %q", qs.MySOTARef)
	}
}

func TestParseWSJTXADIFMalformed(t *testing.T) {
	// Malformed ADIF should not panic
	adif := "garbage data not valid adif"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil for malformed input")
	}
}

func TestParseWSJTXADIFSource(t *testing.T) {
	adif := "<CALL:6>SP9MOA <MODE:3>SSB <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	// Source should be "wsjtx" (set by ParseADIFRecord).
	if qs.Source != "wsjtx" {
		t.Errorf("Source = %q; want 'wsjtx' (from WSJT-X ADIF)", qs.Source)
	}
}

// =============================================================================
// WSJT-X TX message handling tests
// =============================================================================

func TestApplyWSJTXStatus_StoresTxMessage(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.fields = [fieldCount]textinput.Model{}
	for i := range m.fields {
		m.fields[i] = newTextinput()
	}

	m.applyWSJTXStatus("SP9ABC", "JO90", 14074000, "FT8", "", "-12", "CQ SP9XXX JO90", true)

	if !m.wsjtx.online {
		t.Error("wsjtxOnline should be true after status")
	}
	if m.wsjtx.txMsg != "CQ SP9XXX JO90" {
		t.Errorf("wsjtxTxMsg = %q; want 'CQ SP9XXX JO90'", m.wsjtx.txMsg)
	}
	if !m.wsjtx.tx {
		t.Error("wsjtxTx should be true when transmitting=true")
	}
	if m.wsjtx.lastSeen.IsZero() {
		t.Error("wsjtxLastSeen should be set")
	}
	if m.rc.status != "" {
		t.Error("cachedStatus should be invalidated (empty)")
	}
}

func TestApplyWSJTXStatus_EmptyTxMessage(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.fields = [fieldCount]textinput.Model{}
	for i := range m.fields {
		m.fields[i] = newTextinput()
	}

	m.wsjtx.online = true
	m.wsjtx.txMsg = "CQ SP9XXX JO90"
	m.applyWSJTXStatus("", "", 0, "", "", "", "", false)

	if m.wsjtx.txMsg != "" {
		t.Errorf("wsjtxTxMsg should be cleared to empty, got %q", m.wsjtx.txMsg)
	}
	if !m.wsjtx.online {
		t.Error("wsjtxOnline should remain true even with empty message")
	}
}

func TestWSJTXWatchdog_Expires(t *testing.T) {
	m := &Model{}
	m.wsjtx.online = true
	m.wsjtx.txMsg = "CQ SP9XXX JO90"
	m.wsjtx.lastSeen = time.Now().Add(-20 * time.Second)

	// Simulate watchdog check.
	if m.wsjtx.online && time.Since(m.wsjtx.lastSeen) > 15*time.Second {
		m.wsjtx.online = false
		m.wsjtx.txMsg = ""
		m.rc.status = ""
	}

	if m.wsjtx.online {
		t.Error("watchdog should set wsjtxOnline to false after 15s of inactivity")
	}
	if m.wsjtx.txMsg != "" {
		t.Error("watchdog should clear wsjtxTxMsg")
	}
	if m.rc.status != "" {
		t.Error("watchdog should invalidate cachedStatus")
	}
}

func TestWSJTXWatchdog_NotExpired(t *testing.T) {
	m := &Model{}
	m.wsjtx.online = true
	m.wsjtx.txMsg = "CQ SP9XXX JO90"
	m.wsjtx.lastSeen = time.Now()

	if m.wsjtx.online && time.Since(m.wsjtx.lastSeen) > 15*time.Second {
		m.wsjtx.online = false
	}

	if !m.wsjtx.online {
		t.Error("watchdog should NOT expire when last seen is recent")
	}
	if m.wsjtx.txMsg != "CQ SP9XXX JO90" {
		t.Error("tx message should be preserved when watchdog does not expire")
	}
}

func TestApplyWSJTXStatus_DirectedCall(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.fields = [fieldCount]textinput.Model{}
	for i := range m.fields {
		m.fields[i] = newTextinput()
	}

	m.applyWSJTXStatus("K1ABC", "FN42", 21074000, "FT8", "", "-05", "K1ABC SP9XXX -05", true)

	if m.wsjtx.txMsg != "K1ABC SP9XXX -05" {
		t.Errorf("wsjtxTxMsg = %q; want directed call message", m.wsjtx.txMsg)
	}
	// Verify the call field was updated.
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if call != "K1ABC" {
		t.Errorf("call field = %q; want K1ABC", call)
	}
}

// TestWSJTXUploadRevisionAware verifies the WSJT-X upload no longer disables
// revision checking: the QSO row and its revision are loaded together, and an
// edit saved while the upload is on the wire must leave the row durably dirty
// (queued for a follow-up PATCH) instead of being dismissed by -1 and marked
// clean although the server holds the older snapshot.
func TestWSJTXUploadRevisionAware(t *testing.T) {
	postStarted := make(chan struct{})
	release := make(chan struct{})
	srv := newWavelogTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		close(postStarted)
		<-release
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 55, "call": "SP9MOA"},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	})
	defer srv.Close()

	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = srv.URL
	m.App.Logbook.Wavelog.APIKey = "wl2_test"
	m.App.Logbook.Wavelog.StationProfileID = "1"
	m.inetOnline = true

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "FT8", QSODate: "20260921", TimeOn: "120000",
		RSTSent: "-10", RSTRcvd: "-05", Comment: "old"}
	id, err := store.InsertQSO(m.App.DB, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	cmd := m.wsjtxEnrichAndUploadCmd(id, "SP9MOA")
	if cmd == nil {
		t.Fatal("expected enrich/upload command")
	}

	resultCh := make(chan wlUploadResultMsg, 1)
	go func() { resultCh <- cmd().(wlUploadResultMsg) }()

	// The POST is in flight — a newer local edit lands now.
	<-postStarted
	row, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	row.Comment = "newer"
	if err := store.UpdateQSO(m.App.DB, row); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	close(release)

	r := <-resultCh
	if !r.ok || r.err != nil {
		t.Fatalf("upload: ok=%v err=%v", r.ok, r.err)
	}
	if r.remoteID != 55 {
		t.Errorf("remoteID = %d, want 55", r.remoteID)
	}
	if !r.changed {
		t.Fatal("changed must be true — the row was edited while the upload was on the wire")
	}
	if r.unresolved {
		t.Error("unresolved must be false — the remote id was persisted")
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID after upload: %v", err)
	}
	if stored.WavelogID != 55 {
		t.Errorf("WavelogID = %d, want 55", stored.WavelogID)
	}
	if !stored.WavelogDirty {
		t.Fatal("row must be durably dirty — the server holds the old snapshot")
	}
}
