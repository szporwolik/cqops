package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
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

func remoteQSODoc(id int64) map[string]any {
	return map[string]any{
		"id": id, "station_id": 1, "qso_date": "2026-05-01 12:30:00",
		"mode": "SSB", "submode": nil, "freq": "14250000", "freq_rx": nil,
		"call": "SP9MOA", "band": "20m", "band_rx": "",
		"rst_sent": "55", "rst_rcvd": "57", "gridsquare": "JO90", "name": "Remote",
		"comment": "remote edit", "notes": "n", "qth": "Krakow", "tx_pwr": "25",
		"prop_mode": "", "sat_name": "", "sat_mode": "",
		"sota_ref": "SP/TQ-001", "pota_ref": "", "wwff_ref": "", "iota": "",
		"sig": "", "sig_info": "", "darc_dok": "", "state": "", "cnty": "",
		"cqz": 15, "ituz": 28, "qsl_via": "", "srx": nil, "stx": nil,
		"srx_string": "", "stx_string": "",
	}
}

// TestFillEditForm_ClearsPreviousNumericValues reproduces the stale-field
// bug: editing a contact with freq/freq_rx/distance/bearing/serial values,
// then one with those fields empty, must clear them — otherwise saving the
// second contact persists the first contact's numbers.
func TestFillEditForm_ClearsPreviousNumericValues(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")

	populated := &qso.QSO{Call: "A1AA", Band: "20m", Mode: "SSB", QSODate: "20260601",
		TimeOn: "120000", Freq: 14.2500, FreqRx: 14.2505, Distance: 123.4, Bearing: 90, STX: 5, SRX: 7}
	le.editing = populated
	le.fillEditForm(populated)

	empty := &qso.QSO{Call: "B2BB", Band: "40m", Mode: "FT8", QSODate: "20260602", TimeOn: "130000"}
	le.editing = empty
	le.fillEditForm(empty)

	for _, f := range []qsoEditField{qefFreq, qefFreqRx, qefDistance, qefBearing, qefSTX, qefSRX} {
		if v := le.fields[f].Value(); v != "" {
			t.Errorf("field %d should be cleared for the empty contact, got %q", f, v)
		}
	}
	if got := le.fields[qefCall].Value(); got != "B2BB" {
		t.Errorf("call = %q, want B2BB", got)
	}

	// The read-back form must not resurrect the previous values either.
	round := le.readEditForm()
	if round.Freq != 0 || round.FreqRx != 0 || round.Distance != 0 || round.Bearing != 0 || round.STX != 0 || round.SRX != 0 {
		t.Errorf("readEditForm carried stale numeric values: %+v", round)
	}
}

// TestFetchRemoteCopyAndRefresh verifies the fetch-on-enter flow: the remote
// copy is fetched, merged into the local row and reflected in the edit form.
func TestFetchRemoteCopyAndRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso/77" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(77)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "local", WavelogID: 77}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)

	cmd := le.fetchRemoteCopy(77, id)
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.wlFetchQSO == nil || em.wlFetchErr != "" {
		t.Fatalf("fetch failed: qso=%v err=%q", em.wlFetchQSO, em.wlFetchErr)
	}
	// The response must carry the issuing editor's identity.
	if em.wlFetchGen != le.gen || em.wlFetchDB != le.db {
		t.Fatalf("fetch result not bound to the issuing editor: gen=%d db-set=%v; want gen=%d",
			em.wlFetchGen, em.wlFetchDB != nil, le.gen)
	}

	applied, err := le.ApplyRemoteRefresh(em.wlFetchQSO, remoteRefreshRequest{
		gen: em.wlFetchGen, db: em.wlFetchDB, localID: em.wlFetchQSOID, rev: em.wlFetchRev,
	})
	if err != nil {
		t.Fatalf("ApplyRemoteRefresh: %v", err)
	}
	if !applied {
		t.Fatal("ApplyRemoteRefresh should apply a fresh fetch with no edits")
	}

	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "remote edit" {
		t.Errorf("Comment = %q, want remote edit", stored.Comment)
	}
	if stored.RSTSent != "55" || stored.RSTRcvd != "57" {
		t.Errorf("RST = %q/%q, want 55/57", stored.RSTSent, stored.RSTRcvd)
	}
	if stored.QSODate != "20260501" || stored.TimeOn != "123000" {
		t.Errorf("date/time = %s %s, want 20260501 123000", stored.QSODate, stored.TimeOn)
	}
	if stored.Freq != 14.25 {
		t.Errorf("Freq = %v, want 14.25", stored.Freq)
	}
	if stored.Name != "Remote" || stored.QTH != "Krakow" || stored.TXPower != "25" {
		t.Errorf("merged fields wrong: %+v", stored)
	}
	if stored.WavelogID != 77 {
		t.Errorf("WavelogID = %d, want 77", stored.WavelogID)
	}
	// The edit form must show the refreshed values.
	if le.fields[qefComment].Value() != "remote edit" {
		t.Errorf("form comment = %q, want remote edit", le.fields[qefComment].Value())
	}
	// The read-only WL Id field shows the remote id.
	if le.fields[qefWLStatus].Value() != "77" {
		t.Errorf("form WL Id = %q, want 77", le.fields[qefWLStatus].Value())
	}
}

// TestFetchRemoteCopyStaleResultIgnored verifies a late refresh result does
// not clobber the form after the user left the edit mode.
func TestFetchRemoteCopyStaleResultIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(77)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")
	le.editing = &qso.QSO{ID: 1, WavelogID: 77}
	le.mode = edModeList // user already left the edit form

	applied, err := le.ApplyRemoteRefresh(&wavelog.QSOData{ID: 77},
		remoteRefreshRequest{localID: 1, rev: 0})
	if err != nil {
		t.Fatalf("ApplyRemoteRefresh should be a no-op, got %v", err)
	}
	if applied {
		t.Error("ApplyRemoteRefresh should not apply after the user left edit mode")
	}
}

// TestApplyRemoteRefresh_KeepsUnsavedEdits reproduces the reported bug: the
// operator types while the remote GET is pending. The late result carries
// the pre-edit revision and must NOT overwrite the typed values.
func TestApplyRemoteRefresh_KeepsUnsavedEdits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(77)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "local", WavelogID: 77}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.editRev = 0

	// Fetch begins at revision 0.
	cmd := le.fetchRemoteCopy(77, id)
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok || em.wlFetchQSO == nil {
		t.Fatalf("fetch failed: %#v", msg)
	}

	// The operator types while the GET is in flight.
	le.fields[qefComment].SetValue("typed while fetching")
	le.editRev++

	applied, err := le.ApplyRemoteRefresh(em.wlFetchQSO, remoteRefreshRequest{
		gen: em.wlFetchGen, db: em.wlFetchDB, localID: em.wlFetchQSOID, rev: em.wlFetchRev,
	})
	if err != nil {
		t.Fatalf("ApplyRemoteRefresh: %v", err)
	}
	if applied {
		t.Fatal("stale refresh must not apply after the operator typed")
	}

	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "local" {
		t.Errorf("DB comment = %q, want local (typed values and row must survive)", stored.Comment)
	}
	if le.fields[qefComment].Value() != "typed while fetching" {
		t.Errorf("form comment = %q, want the typed value kept", le.fields[qefComment].Value())
	}
}

// TestApplyRemoteRefresh_KeepsPendingLocalChanges verifies a remote refresh
// never overwrites a row whose local changes were saved but not synced to
// Wavelog (durable pending-sync state).
func TestApplyRemoteRefresh_KeepsPendingLocalChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(77)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "offline change", WavelogID: 77}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	if err := store.SetWavelogDirty(le.db, id, true); err != nil {
		t.Fatalf("SetWavelogDirty: %v", err)
	}

	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)

	applied, err := le.ApplyRemoteRefresh(&wavelog.QSOData{ID: 77},
		remoteRefreshRequest{localID: id, rev: le.editRev})
	if err != nil {
		t.Fatalf("ApplyRemoteRefresh: %v", err)
	}
	if applied {
		t.Fatal("refresh must not apply over pending unsynced local changes")
	}

	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "offline change" {
		t.Errorf("DB comment = %q, want offline change (must survive the refresh)", stored.Comment)
	}
}

// TestEnterSkipsRemoteRefreshForPendingSync verifies opening a synced QSO
// with durable pending changes never triggers the remote GET — the operator
// edits the local copy and gets a notice instead.
func TestEnterSkipsRemoteRefreshForPendingSync(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Errorf("pending-sync row must not trigger a remote GET (got %s %s)", r.Method, r.URL.Path)
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", Comment: "offline change", WavelogID: 77}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	if err := store.SetWavelogDirty(le.db, id, true); err != nil {
		t.Fatalf("SetWavelogDirty: %v", err)
	}
	le.qsos = []qso.QSO{*q}
	le.mode = edModeList

	upd, cmd := le.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Fatalf("mode = %v, want edModeEdit", le.mode)
	}
	if cmd == nil {
		t.Fatal("expected a toast command for the pending-sync notice")
	}
	msg := execCmd(cmd)
	em, ok := msg.(editorMsg)
	if !ok || em.toastWarn == "" {
		t.Fatalf("expected a pending-sync toast, got %#v", msg)
	}
	if requests != 0 {
		t.Errorf("Wavelog received %d request(s) for a pending-sync row", requests)
	}
}

// TestEditSavePatchesWavelog verifies saving a synced QSO also PATCHes the
// Wavelog copy with the v2 conventions (date-only qso_date + HH:MM:SS time_on
// together, string-encoded Hz frequency).
func TestEditSavePatchesWavelog(t *testing.T) {
	var patchedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/qso/42" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&patchedBody); err != nil {
			t.Errorf("decode patch body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", Freq: 14.25, RSTSent: "59", RSTRcvd: "59", Comment: "edited", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("edited twice")

	msg := execCmd(le.doSave())
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.err != nil {
		t.Fatalf("save failed: %v", em.err)
	}
	if !em.wlSyncOK {
		t.Fatalf("wlSyncOK = false, want true (err=%q gone=%v)", em.wlSyncErr, em.wlSyncGone)
	}

	if patchedBody == nil {
		t.Fatal("no PATCH received by the mock server")
	}
	if patchedBody["qso_date"] != "2024-05-01" || patchedBody["time_on"] != "12:00:00" {
		t.Errorf("date/time = %v/%v, want 2024-05-01 + 12:00:00", patchedBody["qso_date"], patchedBody["time_on"])
	}
	if patchedBody["comment"] != "edited twice" {
		t.Errorf("comment = %v, want edited twice", patchedBody["comment"])
	}
	if patchedBody["freq"] != "14250000" {
		t.Errorf("freq = %v, want 14250000 (string Hz)", patchedBody["freq"])
	}
	if patchedBody["mode"] != "SSB" || patchedBody["band"] != "20m" || patchedBody["call"] != "SP9MOA" {
		t.Errorf("identity fields wrong: %v", patchedBody)
	}
	if _, ok := patchedBody["station_profile_id"]; ok {
		t.Error("PATCH must not send station_profile_id")
	}

	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "edited twice" {
		t.Errorf("local comment = %q, want edited twice", stored.Comment)
	}
	if stored.WavelogID != 42 {
		t.Errorf("WavelogID = %d, want 42", stored.WavelogID)
	}
	if stored.WavelogDirty {
		t.Error("WavelogDirty should be cleared after a successful PATCH")
	}
}

// TestEditSavePatchGone verifies a save whose remote copy was deleted clears
// the local remote id and reports the state honestly.
func TestEditSavePatchGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "not_found", "message": "QSO not found"},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)

	msg := execCmd(le.doSave())
	em := msg.(editorMsg)
	if em.err != nil {
		t.Fatalf("save failed: %v", em.err)
	}
	if !em.wlSyncGone || em.wlSyncOK {
		t.Errorf("wlSyncGone=%v wlSyncOK=%v, want gone", em.wlSyncGone, em.wlSyncOK)
	}
	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogID != 0 {
		t.Errorf("WavelogID = %d, want 0 after remote deletion", stored.WavelogID)
	}
}

// TestEditSavePatchFails verifies local save succeeds and the remote id is
// kept when the Wavelog PATCH fails for transient reasons.
func TestEditSavePatchFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "kept", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)

	msg := execCmd(le.doSave())
	em := msg.(editorMsg)
	if em.err != nil {
		t.Fatalf("local save must not fail: %v", em.err)
	}
	if em.wlSyncErr == "" || em.wlSyncOK || em.wlSyncGone {
		t.Errorf("wlSyncErr=%q wlSyncOK=%v wlSyncGone=%v, want sync error", em.wlSyncErr, em.wlSyncOK, em.wlSyncGone)
	}
	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "kept" {
		t.Errorf("local comment = %q, want kept", stored.Comment)
	}
	if stored.WavelogID != 42 {
		t.Errorf("WavelogID = %d, want 42 (kept after failed PATCH)", stored.WavelogID)
	}
	if !stored.WavelogDirty {
		t.Error("WavelogDirty should be set after a failed PATCH (local row diverged)")
	}
}

// TestEditSaveOfflineDoesNotContactWavelog verifies --offline mode never
// contacts Wavelog on save and reports the change as pending sync.
func TestEditSaveOfflineDoesNotContactWavelog(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Errorf("offline save must not contact Wavelog (got %s %s)", r.Method, r.URL.Path)
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")
	le.Offline = true

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "offline", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("offline edit")

	msg := execCmd(le.doSave())
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.err != nil {
		t.Fatalf("local save failed: %v", em.err)
	}
	if !em.wlSyncPending {
		t.Error("wlSyncPending should be set for a synced QSO saved offline")
	}
	if em.wlSyncOK || em.wlSyncGone || em.wlSyncErr != "" {
		t.Errorf("no sync result expected offline: ok=%v gone=%v err=%q", em.wlSyncOK, em.wlSyncGone, em.wlSyncErr)
	}
	if requests != 0 {
		t.Errorf("Wavelog received %d request(s) in offline mode", requests)
	}

	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "offline edit" {
		t.Errorf("local comment = %q, want offline edit", stored.Comment)
	}
	if stored.WavelogID != 42 {
		t.Errorf("WavelogID = %d, want 42 (kept for future sync)", stored.WavelogID)
	}
	if !stored.WavelogDirty {
		t.Error("WavelogDirty should be set for an offline save (sync deferred)")
	}
}

// TestEditSaveClearsWavelogFields verifies clearing a field in the editor
// clears the remote copy too: the PATCH carries explicit nulls for the
// emptied fields instead of silently leaving the remote value in place.
func TestEditSaveClearsWavelogFields(t *testing.T) {
	var patchedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/qso/42" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&patchedBody); err != nil {
			t.Errorf("decode patch body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", Freq: 14.25, FreqRx: 14.2, RSTSent: "59", RSTRcvd: "59",
		GridSquare: "JO90", Comment: "old comment", IOTA: "EU-001", SOTARef: "SP/BB-001", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)
	// The operator empties comment, locator, receive frequency and refs.
	le.fields[qefComment].SetValue("")
	le.fields[qefGrid].SetValue("")
	le.fields[qefFreqRx].SetValue("")
	le.fields[qefIOTA].SetValue("")
	le.fields[qefSOTA].SetValue("")

	msg := execCmd(le.doSave())
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.err != nil {
		t.Fatalf("save failed: %v", em.err)
	}
	if !em.wlSyncOK {
		t.Fatalf("wlSyncOK = false, want true (err=%q)", em.wlSyncErr)
	}

	if patchedBody == nil {
		t.Fatal("no PATCH received by the mock server")
	}
	for _, key := range []string{"comment", "gridsquare", "freq_rx", "iota", "sota_ref"} {
		v, present := patchedBody[key]
		if !present || v != nil {
			t.Errorf("%s = %v (present=%v), want explicit null", key, v, present)
		}
	}
	// Fields that were not cleared keep their values.
	if patchedBody["rst_sent"] != "59" || patchedBody["freq"] != "14250000" {
		t.Errorf("untouched fields wrong: rst_sent=%v freq=%v", patchedBody["rst_sent"], patchedBody["freq"])
	}

	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "" || stored.GridSquare != "" || stored.FreqRx != 0 || stored.IOTA != "" || stored.SOTARef != "" {
		t.Errorf("cleared fields not persisted locally: comment=%q grid=%q freq_rx=%v iota=%q sota=%q",
			stored.Comment, stored.GridSquare, stored.FreqRx, stored.IOTA, stored.SOTARef)
	}
	if stored.WavelogID != 42 {
		t.Errorf("WavelogID = %d, want 42", stored.WavelogID)
	}
}

// TestEditDeleteRemovesWavelog verifies deleting a synced QSO also DELETEs the
// Wavelog copy and reports success.
func TestEditDeleteRemovesWavelog(t *testing.T) {
	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v2/qso/42" {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.qsos = []qso.QSO{*q}
	le.buildTable()
	le.mode = edModeConfirmDelete
	le.ensureDialog("Delete QSO", "x", Option{})

	msg := execCmd(le.doConfirm())
	em, ok := msg.(editorMsg)
	if !ok {
		t.Fatalf("expected editorMsg, got %T", msg)
	}
	if em.err != nil {
		t.Fatalf("local delete failed: %v", em.err)
	}
	if !em.delSyncOK || em.delSyncErr != "" {
		t.Errorf("delSyncOK=%v delSyncErr=%q, want remote delete ok", em.delSyncOK, em.delSyncErr)
	}
	if !deleted {
		t.Error("mock server never received the remote DELETE")
	}
	if _, err := store.GetQSOByID(le.db, id); err == nil {
		t.Error("local row should be gone")
	}
}

// TestEditDeleteRemoteGone verifies a 404 during remote delete is treated as
// success (the remote copy was already removed).
func TestEditDeleteRemoteGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "not_found", "message": "QSO not found"},
		})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.qsos = []qso.QSO{*q}
	le.buildTable()
	le.mode = edModeConfirmDelete
	le.ensureDialog("Delete QSO", "x", Option{})

	em := execCmd(le.doConfirm()).(editorMsg)
	if em.err != nil {
		t.Fatalf("local delete failed: %v", em.err)
	}
	if !em.delSyncOK || em.delSyncErr != "" {
		t.Errorf("delSyncOK=%v delSyncErr=%q, want remote delete ok (404 = already gone)", em.delSyncOK, em.delSyncErr)
	}
	if _, err := store.GetQSOByID(le.db, id); err == nil {
		t.Error("local row should be gone")
	}
}

// TestEditDeleteRemoteFails verifies a failed remote delete does not break the
// local delete and is reported as a warning.
func TestEditDeleteRemoteFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.qsos = []qso.QSO{*q}
	le.buildTable()
	le.mode = edModeConfirmDelete
	le.ensureDialog("Delete QSO", "x", Option{})

	em := execCmd(le.doConfirm()).(editorMsg)
	if em.err != nil {
		t.Fatalf("local delete failed: %v", em.err)
	}
	if em.delSyncErr == "" || em.delSyncOK {
		t.Errorf("delSyncErr=%q delSyncOK=%v, want remote failure reported", em.delSyncErr, em.delSyncOK)
	}
	if _, err := store.GetQSOByID(le.db, id); err == nil {
		t.Error("local row should be gone")
	}
}

// TestConfirmMessagesMentionWavelog verifies the confirm dialogs tell the user
// about the Wavelog side effects, and stay silent for unsynced QSOs.
func TestConfirmMessagesMentionWavelog(t *testing.T) {
	le := newTestEditorWithDB(t, "https://log.example.com", "wl2_test", "1", "Szymon", "KO00ca")

	synced := &qso.QSO{Call: "SP9MOA", QSODate: "20240501", WavelogID: 42}
	unsynced := &qso.QSO{Call: "SP9MOA", QSODate: "20240501"}

	if msg := le.deleteConfirmMessage(synced); msg == "" || !strings.Contains(msg, "also be deleted from Wavelog") {
		t.Errorf("delete message for synced QSO = %q", msg)
	}
	if msg := le.saveConfirmMessage(synced); msg == "" || !strings.Contains(msg, "also be updated on Wavelog") {
		t.Errorf("save message for synced QSO = %q", msg)
	}
	if msg := le.deleteConfirmMessage(unsynced); strings.Contains(msg, "Wavelog") {
		t.Errorf("delete message for unsynced QSO should not mention Wavelog: %q", msg)
	}
	if msg := le.saveConfirmMessage(unsynced); strings.Contains(msg, "Wavelog") {
		t.Errorf("save message for unsynced QSO should not mention Wavelog: %q", msg)
	}

	le.Offline = true
	if msg := le.deleteConfirmMessage(synced); msg == "" || !strings.Contains(msg, "locally only") {
		t.Errorf("offline delete message = %q, want local-only note", msg)
	}
	if msg := le.saveConfirmMessage(synced); msg == "" || !strings.Contains(msg, "locally only") {
		t.Errorf("offline save message = %q, want local-only note", msg)
	}
}

// TestApplyRemoteRefresh_RejectsPreviousEditSession reproduces the reported
// bug: reopening the same contact reset editRev to zero, so two edit
// sessions shared the same refresh identity — the older response passed
// every check after the newer one applied and overwrote the database and
// form with stale data.
func TestApplyRemoteRefresh_RejectsPreviousEditSession(t *testing.T) {
	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		doc := remoteQSODoc(77)
		if reqCount == 1 {
			doc["comment"] = "stale copy"
		} else {
			doc["comment"] = "fresh copy"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": doc})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "local", WavelogID: 77}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	// Session 1: the contact is opened and a refresh starts.
	le.editing = q
	le.mode = edModeEdit
	le.editRev = 0
	le.editSession = 1
	le.fillEditForm(q)
	em1 := execCmd(le.fetchRemoteCopy(77, id)).(editorMsg)
	if em1.wlFetchQSO == nil {
		t.Fatalf("session-1 fetch failed: %q", em1.wlFetchErr)
	}
	if em1.wlFetchSession != 1 {
		t.Fatalf("session-1 fetch session = %d, want 1", em1.wlFetchSession)
	}

	// The operator leaves and reopens the SAME contact — a new session.
	// editRev resets to zero, but the session identity must never be reused.
	le.editing = q
	le.mode = edModeEdit
	le.editRev = 0
	le.editSession = 2
	le.fillEditForm(q)

	// The newer (session 2) refresh applies first.
	em2 := execCmd(le.fetchRemoteCopy(77, id)).(editorMsg)
	applied, err := le.ApplyRemoteRefresh(em2.wlFetchQSO, remoteRefreshRequest{
		gen: em2.wlFetchGen, db: em2.wlFetchDB, localID: em2.wlFetchQSOID,
		rev: em2.wlFetchRev, session: em2.wlFetchSession,
	})
	if err != nil || !applied {
		t.Fatalf("session-2 refresh should apply: applied=%v err=%v", applied, err)
	}

	// The older (session 1) response then arrives — identical generation,
	// database, row and revision. It must be rejected.
	applied, err = le.ApplyRemoteRefresh(em1.wlFetchQSO, remoteRefreshRequest{
		gen: em1.wlFetchGen, db: em1.wlFetchDB, localID: em1.wlFetchQSOID,
		rev: em1.wlFetchRev, session: em1.wlFetchSession,
	})
	if err != nil {
		t.Fatalf("ApplyRemoteRefresh: %v", err)
	}
	if applied {
		t.Fatal("a refresh from a previous edit session must not apply")
	}

	// The database and form must keep the newer session's copy.
	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "fresh copy" {
		t.Errorf("DB comment = %q, want fresh copy (stale session must not overwrite)", stored.Comment)
	}
	if le.fields[qefComment].Value() != "fresh copy" {
		t.Errorf("form comment = %q, want fresh copy", le.fields[qefComment].Value())
	}
}

// TestSaveCompletionFromReplacedEditorDoesNotCloseForm reproduces the
// reported bug: a delayed save completion from editor A (logbook switched)
// used to unconditionally move editor B into list mode, closing B's unsaved
// form.
func TestSaveCompletionFromReplacedEditorDoesNotCloseForm(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.editSession = 5

	// A completion from a DIFFERENT editor (generation mismatch).
	upd, _ := le.Update(editorMsg{saved: id, saveCall: "SP9MOA", gen: le.gen + 1, saveSession: 5})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("a save completion from a replaced editor must not close the form")
	}
}

// TestSaveCompletionFromSupersededSessionDoesNotCloseForm verifies that an
// earlier save's completion arriving after the contact was reopened never
// closes the newer session's form — only the session that initiated the save
// may transition back to the list.
func TestSaveCompletionFromSupersededSessionDoesNotCloseForm(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.editSession = 1

	// The save was initiated in session 1; its completion arrives after
	// the contact was reopened in session 2.
	le.editSession = 2
	upd, _ := le.Update(editorMsg{saved: id, saveCall: "SP9MOA", gen: le.gen, saveSession: 1})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("a save completion from a superseded session must not close the current form")
	}

	// The current session's own completion still closes the form.
	upd, _ = le.Update(editorMsg{saved: id, saveCall: "SP9MOA", gen: le.gen, saveSession: 2})
	le = upd.(*LogbookEditor)
	if le.mode != edModeList {
		t.Error("the initiating session's own save completion should close the form")
	}
}

// TestUploadCompletionFromReplacedEditorDoesNotCloseForm verifies upload
// completions are generation-scoped too: a delayed result from another
// editor must not switch the current editor into list mode.
func TestUploadCompletionFromReplacedEditorDoesNotCloseForm(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000"}
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)

	upd, _ := le.Update(editorMsg{wlQSOID: 1, wlCall: "SP9MOA", wlOK: true, gen: le.gen + 1})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("an upload completion from a replaced editor must not close the form")
	}

	upd, _ = le.Update(editorMsg{wlQSOID: 1, wlCall: "SP9MOA", wlOK: true, gen: le.gen})
	le = upd.(*LogbookEditor)
	if le.mode != edModeList {
		t.Error("a matching-generation upload completion should close the form")
	}
}

// TestFollowUpUsesOriginatingContextAcrossLogbookSwitch reproduces the
// reported corruption: queuing another edit in logbook A and switching to
// logbook B WITHOUT recreating the editor used to make the follow-up read
// B's row with the same local id (m.App.DB) and PATCH it to A's server (the
// retained editor's credentials) — contact corruption plus unintended data
// disclosure. The follow-up must be bound to the immutable originating
// database and endpoint, and the database lease must survive the interval
// between the two workers.
func TestFollowUpUsesOriginatingContextAcrossLogbookSwitch(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	var patched []string // "call|comment" per PATCH received
	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode patch body: %v", err)
		}
		mu.Lock()
		patched = append(patched, fmt.Sprintf("%v|%v", body["call"], body["comment"]))
		first := len(patched) == 1
		mu.Unlock()
		if first {
			close(started)
			<-release
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srvA.Close()

	dir := t.TempDir()
	dbPathA := filepath.Join(dir, "a.db")
	dbPathB := filepath.Join(dir, "b.db")

	cfg := config.DefaultConfig()
	lbA := config.Logbook{
		Station:      config.Station{Callsign: "SP9A", Grid: "JO90"},
		DatabasePath: dbPathA,
		Wavelog:      &config.WavelogConfig{Enabled: true, URL: srvA.URL, APIKey: "wl2_test", StationProfileID: "1"},
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

	// Logbook A: a synced contact (local id 1).
	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "first", WavelogID: 42}
	idA, err := store.InsertQSO(dbA, q1)
	if err != nil {
		t.Fatalf("InsertQSO A: %v", err)
	}
	q1.ID = idA

	le := NewLogbookEditor(LogbookEditorConfig{
		DB: dbA, WLURL: srvA.URL, WLKey: "wl2_test", WLStationID: "1",
		StationOperator: "OP", StationGrid: "JO90", KeepAlive: a.KeepDBAlive,
	})
	le.editing = q1
	le.mode = edModeEdit
	le.fillEditForm(q1)
	le.fields[qefComment].SetValue("edited-1")

	m := New(a, nil)
	m.ui.logbookEditor = le

	done1 := make(chan editorMsg, 1)
	go func() { done1 <- execCmd(le.doSave()).(editorMsg) }()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first PATCH never started")
	}

	// The operator edits and saves again while the PATCH is on the wire.
	le.fields[qefComment].SetValue("edited-2")
	em2 := execCmd(le.doSave()).(editorMsg)
	if !em2.wlSyncPending {
		t.Fatalf("second save: wlSyncPending=%v, want queued", em2.wlSyncPending)
	}

	// Switch to logbook B WITHOUT recreating the editor (the real flow).
	if err := a.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { a.DB.Close() })

	// B happens to contain a DIFFERENT contact under the same local id.
	qB := &qso.QSO{Call: "SP9ZZZ", Band: "40m", Mode: "CW", QSODate: "20240502",
		TimeOn: "130000", Comment: "other", WavelogID: 99}
	idB, err := store.InsertQSO(a.DB, qB)
	if err != nil {
		t.Fatalf("InsertQSO B: %v", err)
	}
	if idB != idA {
		t.Fatalf("B row id = %d, want %d (same local id as A's row)", idB, idA)
	}

	close(release)
	em1 := <-done1
	if !em1.wlSyncOK {
		t.Fatalf("first PATCH: ok=%v err=%q", em1.wlSyncOK, em1.wlSyncErr)
	}

	// The follow-up must read A's row from the ORIGINATING database and
	// PATCH the ORIGINATING endpoint — the current App.DB (B) and the
	// editor's credentials (A) were both wrong inputs on the old code.
	followUp := m.handleQSOSyncCompletion(em1)
	if followUp == nil {
		t.Fatal("expected the queued follow-up PATCH")
	}
	em3 := execCmd(followUp).(editorMsg)
	if !em3.wlSyncOK {
		t.Fatalf("follow-up PATCH: ok=%v err=%q", em3.wlSyncOK, em3.wlSyncErr)
	}
	if c := m.handleQSOSyncCompletion(em3); c != nil {
		t.Fatal("no further PATCH should be dispatched")
	}

	// The server must have received exactly A's two edits — never B's row.
	mu.Lock()
	got := append([]string(nil), patched...)
	mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("server received %d PATCHes: %v, want 2 (A's edits only)", len(got), got)
	}
	if got[0] != "SP9AAA|edited-1" || got[1] != "SP9AAA|edited-2" {
		t.Fatalf("server PATCHes = %v, want [SP9AAA|edited-1 SP9AAA|edited-2]", got)
	}

	// B's contact is untouched.
	storedB, err := store.GetQSOByID(a.DB, idB)
	if err != nil {
		t.Fatalf("GetQSOByID B: %v", err)
	}
	if storedB.Comment != "other" || storedB.Call != "SP9ZZZ" || storedB.WavelogID != 99 {
		t.Errorf("B's row was corrupted: %+v", storedB)
	}

	// A's row carries the newest edit and the chain lease kept the retired
	// database open until the chain drained.
	reopened, err := store.Open(dbPathA)
	if err != nil {
		t.Fatalf("reopen A: %v", err)
	}
	defer reopened.Close()
	storedA, err := store.GetQSOByID(reopened, idA)
	if err != nil {
		t.Fatalf("GetQSOByID A: %v", err)
	}
	if storedA.Comment != "edited-2" {
		t.Errorf("A comment = %q, want edited-2", storedA.Comment)
	}
	if storedA.WavelogDirty {
		t.Error("A's row should be clean after the follow-up ack")
	}
	// The chain lease was released at chain end: the retired db is closed.
	if err := dbA.Ping(); err == nil {
		t.Error("retired A database should be closed after the chain lease released")
	}
}

// TestSaveSerializationSurvivesEditorRecreation reproduces the reported bug:
// the serialization maps lived on the editor instance, so pressing F8 (which
// creates a NEW editor) while a PATCH was still on the wire let the next
// save dispatch independently — the delayed "first" then overwrote "second"
// on the server (probe: remote="first", local="second", dirty=false). The
// coordination must live on the model, keyed by persistent logbook/contact
// identity and remote source, so queued revisions survive editor recreation.
func TestSaveSerializationSurvivesEditorRecreation(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var mu sync.Mutex
	var applied []string
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Comment string `json:"comment"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		requestCount++
		i := requestCount
		mu.Unlock()
		if i == 1 {
			close(firstStarted)
			<-releaseFirst // the first PATCH stalls
		}
		mu.Lock()
		applied = append(applied, body.Comment)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id, err := store.InsertQSO(m.App.DB, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	q.ID = id

	// Editor instance #1 (F8): save "first" — its PATCH stalls.
	m.initLogbookEditor()
	le1 := m.ui.logbookEditor
	le1.editing = q
	le1.mode = edModeEdit
	le1.fillEditForm(q)
	le1.fields[qefComment].SetValue("first")

	done1 := make(chan editorMsg, 1)
	go func() { done1 <- execCmd(le1.doSave()).(editorMsg) }()
	select {
	case <-firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("first PATCH never started")
	}

	// F8 while the PATCH is still on the wire: a NEW editor instance.
	m.initLogbookEditor()
	le2 := m.ui.logbookEditor
	if le2 == le1 {
		t.Fatal("F8 must create a new editor instance")
	}
	le2.editing = q
	le2.mode = edModeEdit
	le2.fillEditForm(q)
	le2.fields[qefComment].SetValue("second")

	em2 := execCmd(le2.doSave()).(editorMsg)
	if em2.err != nil {
		t.Fatalf("second local save failed: %v", em2.err)
	}
	if !em2.wlSyncPending {
		t.Fatal("second save across editor recreation must be queued while the PATCH is in flight")
	}

	close(releaseFirst)
	em1 := <-done1
	if !em1.wlSyncOK {
		t.Fatalf("first save: ok=%v err=%q, want ok", em1.wlSyncOK, em1.wlSyncErr)
	}
	followUp := m.handleQSOSyncCompletion(em1)
	if followUp == nil {
		t.Fatal("the queued revision must survive editor recreation")
	}
	em3 := execCmd(followUp).(editorMsg)
	if !em3.wlSyncOK {
		t.Fatalf("follow-up save: ok=%v err=%q, want ok", em3.wlSyncOK, em3.wlSyncErr)
	}
	if m.handleQSOSyncCompletion(em3) != nil {
		t.Fatal("no further PATCH should be dispatched")
	}

	mu.Lock()
	gotOrder := append([]string(nil), applied...)
	mu.Unlock()
	if len(gotOrder) != 2 || gotOrder[0] != "first" || gotOrder[1] != "second" {
		t.Fatalf("server applied requests in order %v, want [first second]", gotOrder)
	}

	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "second" {
		t.Errorf("local comment = %q, want second", stored.Comment)
	}
	if stored.WavelogDirty {
		t.Error("pending flag should clear only after the newest revision was acknowledged")
	}
}

// TestPatchCompletionSurvivesPendingLookupEarlyReturn reproduces the reported
// stuck-queue bug: handlePendingRequests can early-return after dispatching
// an unrelated pending lookup, consuming the incoming PATCH completion before
// the serialization handler ran — the slot stayed occupied and every later
// save kept queueing with no worker to drain it. Completions must be
// processed BEFORE unrelated early-return paths.
func TestPatchCompletionSurvivesPendingLookupEarlyReturn(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var mu sync.Mutex
	var applied []string
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Comment string `json:"comment"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		requestCount++
		i := requestCount
		mu.Unlock()
		if i == 1 {
			close(firstStarted)
			<-releaseFirst // the first PATCH stalls
		}
		mu.Lock()
		applied = append(applied, body.Comment)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id, err := store.InsertQSO(m.App.DB, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	q.ID = id

	m.initLogbookEditor()
	le := m.ui.logbookEditor
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("first")

	done1 := make(chan editorMsg, 1)
	go func() { done1 <- execCmd(le.doSave()).(editorMsg) }()
	select {
	case <-firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("first PATCH never started")
	}
	le.fields[qefComment].SetValue("second")
	em2 := execCmd(le.doSave()).(editorMsg)
	if !em2.wlSyncPending {
		t.Fatal("second save must be queued while the PATCH is in flight")
	}

	close(releaseFirst)
	em1 := <-done1
	if !em1.wlSyncOK {
		t.Fatalf("first save: ok=%v err=%q, want ok", em1.wlSyncOK, em1.wlSyncErr)
	}

	// A pending DXC lookup makes handlePendingRequests early-return on the
	// very update that delivers the completion.
	m.dxc.need = true
	m.dxc.call = "SP9MOA"

	upd, out := m.Update(em1)
	m = upd.(*Model)

	// The completion must have been processed before the pending-lookup
	// early return: the queue slot is drained and the follow-up dispatched.
	key := contactSyncKey{logbook: "test", localID: id, url: srv.URL}
	if m.sync.queued[key] {
		t.Fatal("completion was consumed by the pending-lookup early return — the queue is stuck")
	}
	if m.sync.inFlight[key] == nil {
		t.Fatal("the queued follow-up was not dispatched")
	}

	// Walk the returned batch and run the follow-up worker.
	batch, ok := execCmd(out).(tea.BatchMsg)
	if !ok {
		t.Fatalf("Update returned %T, want tea.BatchMsg", out)
	}
	var followUp editorMsg
	for _, sub := range batch {
		if em, ok := sub().(editorMsg); ok && em.wlSyncFollowUp {
			followUp = em
		}
	}
	if followUp.saved == 0 {
		t.Fatal("the follow-up PATCH was not among the dispatched commands")
	}
	if !followUp.wlSyncOK {
		t.Fatalf("follow-up PATCH: ok=%v err=%q, want ok", followUp.wlSyncOK, followUp.wlSyncErr)
	}
	if m.handleQSOSyncCompletion(followUp) != nil {
		t.Fatal("no further PATCH should be dispatched")
	}

	mu.Lock()
	gotOrder := append([]string(nil), applied...)
	mu.Unlock()
	if len(gotOrder) != 2 || gotOrder[0] != "first" || gotOrder[1] != "second" {
		t.Fatalf("server applied requests in order %v, want [first second]", gotOrder)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "second" {
		t.Errorf("local comment = %q, want second", stored.Comment)
	}
	if stored.WavelogDirty {
		t.Error("pending flag should clear after the newest revision was acknowledged")
	}
}

// TestForeignPurgeCompletionPersistsAgainstOriginatingLogbook reproduces the
// reported cursor corruption: purging logbook A and switching to B before the
// completion arrives used to reset B's Wavelog download cursor (99 → 0)
// through the visible editor, while A's own cursor was never handled. The
// completion must apply to the ORIGINATING logbook only — A's cursor resets,
// B's stays untouched, and B's editor is not reloaded.
func TestForeignPurgeCompletionPersistsAgainstOriginatingLogbook(t *testing.T) {
	dir := t.TempDir()
	dbPathA := filepath.Join(dir, "a.db")
	dbPathB := filepath.Join(dir, "b.db")
	if _, err := store.InitDB(dbPathA); err != nil {
		t.Fatalf("init db A: %v", err)
	}
	dbB, err := store.InitDB(dbPathB)
	if err != nil {
		t.Fatalf("init db B: %v", err)
	}
	defer dbB.Close()

	cfg := config.DefaultConfig()
	lbA := config.Logbook{
		Station:      config.Station{Callsign: "SP9A", Grid: "JO90"},
		DatabasePath: dbPathA,
		Wavelog:      &config.WavelogConfig{Enabled: true, URL: "https://a.example", APIKey: "wl2_test", StationProfileID: "1", LastFetchedID: 0},
	}
	lbB := config.Logbook{
		Station:      config.Station{Callsign: "SP9B", Grid: "JO91"},
		DatabasePath: dbPathB,
		Wavelog:      &config.WavelogConfig{Enabled: true, URL: "https://b.example", APIKey: "wl2_test", StationProfileID: "1", LastFetchedID: 99},
	}
	cfg.Logbooks = map[string]config.Logbook{"a": lbA, "b": lbB}
	cfg.State.ActiveLogbook = "a"
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	dbA, err := store.Open(dbPathA)
	if err != nil {
		t.Fatalf("open db A: %v", err)
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

	m := New(a, nil)
	m.screen = screenLogbookEditor
	m.initLogbookEditor()
	genA := m.ui.logbookEditor.gen

	// Switch to B and open its editor before A's purge completion arrives.
	if err := a.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { a.DB.Close() })
	m.initLogbookEditor()
	if got := m.ui.logbookEditor.wlLastFetchedID; got != 99 {
		t.Fatalf("B editor cursor = %d, want 99", got)
	}

	upd, _ := m.Update(editorMsg{purged: true, gen: genA, lbID: "a"})
	m = upd.(*Model)

	// A's cursor was reset against the ORIGINATING logbook…
	if got := m.App.Config.Logbooks["a"].Wavelog.LastFetchedID; got != 0 {
		t.Errorf("A cursor = %d, want 0", got)
	}
	// …and B is untouched everywhere: config, active pointer, editor.
	if got := m.App.Config.Logbooks["b"].Wavelog.LastFetchedID; got != 99 {
		t.Errorf("B config cursor = %d, want 99 (foreign purge must not reset it)", got)
	}
	if got := m.App.Logbook.Wavelog.LastFetchedID; got != 99 {
		t.Errorf("active Wavelog cursor = %d, want 99", got)
	}
	if got := m.ui.logbookEditor.wlLastFetchedID; got != 99 {
		t.Errorf("B editor cursor = %d, want 99", got)
	}
	if m.ui.logbookEditor.needsReload {
		t.Error("foreign purge completion must not reload B's editor")
	}

	// The reset persisted to disk for A, not B.
	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if got := reloaded.Logbooks["a"].Wavelog.LastFetchedID; got != 0 {
		t.Errorf("persisted A cursor = %d, want 0", got)
	}
	if got := reloaded.Logbooks["b"].Wavelog.LastFetchedID; got != 99 {
		t.Errorf("persisted B cursor = %d, want 99", got)
	}
}

// TestSaveCompletionPreservesNewerUnsavedEdits reproduces the reported bug:
// saving a contact and continuing to type (same session) used to close the
// form when the save completion arrived, abandoning the newer input. The
// completion must close only when BOTH the session and the form revision
// still match.
func TestSaveCompletionPreservesNewerUnsavedEdits(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.editSession = 3

	// The save was captured at revision 7; the operator kept typing, so
	// the form is now at revision 8.
	le.editRev = 8
	upd, _ := le.Update(editorMsg{saved: id, saveCall: "SP9MOA", gen: le.gen, saveSession: 3, saveRev: 7})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("a save completion with a newer unsaved revision must not close the form")
	}

	// When session AND revision match, the completion still closes.
	le.editRev = 7
	upd, _ = le.Update(editorMsg{saved: id, saveCall: "SP9MOA", gen: le.gen, saveSession: 3, saveRev: 7})
	le = upd.(*LogbookEditor)
	if le.mode != edModeList {
		t.Error("a completion matching session and revision should close the form")
	}
}

// TestDoSaveStampsFormRevision verifies doSave captures the form revision so
// completions can tell whether the operator typed after the save.
func TestDoSaveStampsFormRevision(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.editRev = 7

	em := execCmd(le.doSave()).(editorMsg)
	if em.saveRev != 7 {
		t.Errorf("saveRev = %d, want 7", em.saveRev)
	}
}

// TestDeleteCompletionDoesNotCloseUnrelatedContactForm reproduces the
// reported bug: deleting contact A and opening contact B while A's remote
// delete is pending used to switch the editor into list mode when A's
// completion arrived, discarding B's unsaved form. The completion must be
// bound to the session it was initiated in and only refresh the list behind
// the open form.
func TestDeleteCompletionDoesNotCloseUnrelatedContactForm(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000"}
	idA := insertTestQSO(t, le.db, q)
	q.ID = idA
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.editSession = 2 // contact B is open with unsaved edits

	// A's delete was initiated in session 1; its completion arrives now.
	upd, _ := le.Update(editorMsg{deleted: idA, delCall: "SP9AAA", gen: le.gen, opSession: 1, opSessionSet: true})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("a delete completion from a superseded session must not close the current form")
	}
	if !le.needsReload {
		t.Error("the list behind the open form should still be refreshed")
	}

	// A completion from the same session still returns to the list.
	le.needsReload = false
	le.editSession = 1
	upd, _ = le.Update(editorMsg{deleted: idA, delCall: "SP9AAA", gen: le.gen, opSession: 1, opSessionSet: true})
	le = upd.(*LogbookEditor)
	if le.mode != edModeList {
		t.Error("the initiating session's own delete completion should close to the list")
	}
}

// TestUploadCompletionDoesNotCloseUnrelatedContactForm verifies the same
// session binding for upload completions: an upload finishing while another
// contact's form is open must refresh the list without closing the form.
func TestUploadCompletionDoesNotCloseUnrelatedContactForm(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000"}
	idB := insertTestQSO(t, le.db, q)
	q.ID = idB
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.editSession = 2

	upd, _ := le.Update(editorMsg{wlQSOID: idB, wlCall: "1 sent", wlOK: true, gen: le.gen, opSession: 1, opSessionSet: true})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("an upload completion from a superseded session must not close the current form")
	}
	if !le.needsReload {
		t.Error("the list behind the open form should still be refreshed")
	}

	le.needsReload = false
	le.editSession = 1
	upd, _ = le.Update(editorMsg{wlQSOID: idB, wlCall: "1 sent", wlOK: true, gen: le.gen, opSession: 1, opSessionSet: true})
	le = upd.(*LogbookEditor)
	if le.mode != edModeList {
		t.Error("the initiating session's own upload completion should close to the list")
	}
}

// TestDeleteCompletionFromSessionZeroDoesNotCloseForm reproduces the reported
// bypass: zero is the legitimate initial session of a fresh editor, so a
// delete dispatched from the list before any contact was opened carries
// session zero — treating zero as "unbound" let its completion close the
// contact opened meanwhile. Session binding must be explicit, so zero-valued
// sessions are validated too.
func TestDeleteCompletionFromSessionZeroDoesNotCloseForm(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	qA := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000"}
	idA := insertTestQSO(t, le.db, qA)
	qB := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502", TimeOn: "130000"}
	insertTestQSO(t, le.db, qB)
	qA.ID = idA
	le.editing = qA
	le.mode = edModeEdit
	le.fillEditForm(qA)

	// The delete was initiated from the fresh editor's session 0 (list
	// mode, no contact opened); contact B is now open in session 1.
	le.editSession = 1
	upd, _ := le.Update(editorMsg{deleted: idA, delCall: "SP9AAA", gen: le.gen, opSession: 0, opSessionSet: true})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("a session-zero delete completion must not close the current form")
	}
	if !le.needsReload {
		t.Error("the list behind the open form should still be refreshed")
	}

	// The same completion delivered while no contact is open (session 0)
	// still closes to the list.
	le.needsReload = false
	le.editSession = 0
	upd, _ = le.Update(editorMsg{deleted: idA, delCall: "SP9AAA", gen: le.gen, opSession: 0, opSessionSet: true})
	le = upd.(*LogbookEditor)
	if le.mode != edModeList {
		t.Error("the initiating session-zero completion should close to the list")
	}

	// An explicitely unbound (legacy) completion keeps the old behavior:
	// no session validation, closes regardless of the open contact.
	le.editSession = 1
	le.mode = edModeEdit
	le.needsReload = false
	upd, _ = le.Update(editorMsg{deleted: idA, delCall: "SP9AAA", gen: le.gen})
	le = upd.(*LogbookEditor)
	if le.mode != edModeList {
		t.Error("an unbound legacy completion should close to the list as before")
	}
}

// TestDeleteCompletionCarriesInitiatingSession verifies doConfirm stamps the
// delete completion with the session it was initiated in, so the close-form
// check can compare against the current session.
func TestDeleteCompletionCarriesInitiatingSession(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.qsos = []qso.QSO{*q}
	le.buildTable()
	le.mode = edModeConfirmDelete
	le.ensureDialog("Delete QSO", "x", Option{})
	le.editSession = 5

	em := execCmd(le.doConfirm()).(editorMsg)
	if em.err != nil {
		t.Fatalf("local delete failed: %v", em.err)
	}
	if em.opSession != 5 {
		t.Errorf("opSession = %d, want 5 (the session the delete was initiated in)", em.opSession)
	}
	if !em.opSessionSet {
		t.Error("opSessionSet should be true for a dispatch that captured a real session")
	}
}

// TestSyncedEditableFieldRoundTrips verifies the field-level synchronization
// contract: every field the edit form can change AND the remote refresh can
// overwrite must survive a full save → PATCH → GET → refresh round-trip.
// Each case edits the field to a NEW value, saves (the PATCH is applied to a
// stateful mock server), re-fetches the server copy and applies the refresh
// — a field omitted from the PATCH would be reported as synced and then
// silently restored to the server's older value (the reported CQ-zone loss).
func TestSyncedEditableFieldRoundTrips(t *testing.T) {
	cases := []struct {
		name     string
		edit     qsoEditField
		newValue string
		jsonKey  string
		want     any
	}{
		{"RSTSent", qefRSTSent, "57", "rst_sent", "57"},
		{"RSTRcvd", qefRSTRcvd, "59", "rst_rcvd", "59"},
		{"Grid", qefGrid, "JO80aa", "gridsquare", "JO80aa"},
		{"Name", qefName, "Bob", "name", "Bob"},
		{"QTH", qefQTH, "Warsaw", "qth", "Warsaw"},
		{"Comment", qefComment, "edited", "comment", "edited"},
		{"Notes", qefNotes, "round trip", "notes", "round trip"},
		{"TXPower", qefTXPower, "75", "tx_pwr", "75"},
		{"SOTA", qefSOTA, "SP/TQ-002", "sota_ref", "SP/TQ-002"},
		{"POTA", qefPOTA, "SP-0002", "pota_ref", "SP-0002"},
		{"WWFF", qefWWFF, "SPFF-0002", "wwff_ref", "SPFF-0002"},
		{"IOTA", qefIOTA, "EU-002", "iota", "EU-002"},
		{"SIG", qefSIG, "POTA", "sig", "POTA"},
		{"SIGInfo", qefSIGInfo, "info", "sig_info", "info"},
		{"CQZone", qefCQZone, "16", "cqz", float64(16)},
		{"ITUZone", qefITUZone, "29", "ituz", float64(29)},
		{"Freq", qefFreq, "21.300", "freq", "21300000"},
		{"FreqRx", qefFreqRx, "21.301", "freq_rx", "21301000"},
	}

	rowValue := func(tc struct {
		name     string
		edit     qsoEditField
		newValue string
		jsonKey  string
		want     any
	}, stored *qso.QSO) string {
		switch tc.edit {
		case qefRSTSent:
			return stored.RSTSent
		case qefRSTRcvd:
			return stored.RSTRcvd
		case qefGrid:
			return stored.GridSquare
		case qefName:
			return stored.Name
		case qefQTH:
			return stored.QTH
		case qefComment:
			return stored.Comment
		case qefNotes:
			return stored.Notes
		case qefTXPower:
			return stored.TXPower
		case qefSOTA:
			return stored.SOTARef
		case qefPOTA:
			return stored.POTARef
		case qefWWFF:
			return stored.WWFFRef
		case qefIOTA:
			return stored.IOTA
		case qefSIG:
			return stored.SIG
		case qefSIGInfo:
			return stored.SIGInfo
		case qefCQZone:
			return stored.CQZone
		case qefITUZone:
			return stored.ITUZone
		case qefFreq:
			return strconv.FormatFloat(stored.Freq, 'f', 3, 64)
		case qefFreqRx:
			return strconv.FormatFloat(stored.FreqRx, 'f', 3, 64)
		}
		return ""
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Stateful mock: the PATCH mutates the remote copy, the GET
			// returns it — exactly the production round trip.
			state := remoteQSODoc(42)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodPatch:
					var p map[string]any
					if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
						t.Errorf("decode patch body: %v", err)
					}
					for k, v := range p {
						state[k] = v
					}
				case http.MethodGet:
				default:
					http.NotFound(w, r)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"data": state})
			}))
			defer srv.Close()

			le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "OP", "JO90")
			q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
				TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
			id := insertTestQSO(t, le.db, q)
			q.ID = id
			le.editing = q
			le.mode = edModeEdit
			le.fillEditForm(q)

			le.fields[tc.edit].SetValue(tc.newValue)

			em := execCmd(le.doSave()).(editorMsg)
			if !em.wlSyncOK {
				t.Fatalf("save: ok=%v err=%q incomplete=%v", em.wlSyncOK, em.wlSyncErr, em.wlSyncIncomplete)
			}
			if got := state[tc.jsonKey]; got != tc.want {
				t.Fatalf("PATCH applied %q = %v (%T), want %v", tc.jsonKey, got, got, tc.want)
			}

			// Re-fetch the server copy and refresh — the NEW value must
			// survive; an omitted PATCH field would restore the old one.
			fetched := execCmd(le.fetchRemoteCopy(42, id)).(editorMsg)
			if fetched.wlFetchQSO == nil {
				t.Fatalf("refresh fetch failed: %q", fetched.wlFetchErr)
			}
			applied, err := le.ApplyRemoteRefresh(fetched.wlFetchQSO, remoteRefreshRequest{
				gen:     fetched.wlFetchGen,
				db:      fetched.wlFetchDB,
				localID: fetched.wlFetchQSOID,
				rev:     fetched.wlFetchRev,
			})
			if err != nil {
				t.Fatalf("ApplyRemoteRefresh: %v", err)
			}
			if !applied {
				t.Fatal("refresh should apply after a fully synced save")
			}

			stored, err := store.GetQSOByID(le.db, id)
			if err != nil {
				t.Fatalf("GetQSOByID: %v", err)
			}
			if got := rowValue(tc, stored); got != tc.newValue {
				t.Errorf("after refresh %s = %q, want %q (the server restored the old value)", tc.name, got, tc.newValue)
			}
		})
	}
}

// TestMergeRemoteQSOAppliesExplicitClears reproduces the reported stale-value
// retention: remote refreshes applied zones only when positive and
// frequencies only when nonempty, so a remote CLEAR (JSON null) left the old
// local values in place — a later unrelated save sent them back and undid the
// clear. Explicit clears must apply, absent fields must leave the local
// value untouched, and populated values must still round-trip.
func TestMergeRemoteQSOAppliesExplicitClears(t *testing.T) {
	fetch := func(t *testing.T, data map[string]any) *wavelog.QSOData {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"data": data})
		}))
		defer srv.Close()
		d, err := wavelog.GetQSO(srv.URL, "wl2_test", 42)
		if err != nil {
			t.Fatalf("GetQSO: %v", err)
		}
		return d
	}

	local := &qso.QSO{CQZone: "15", ITUZone: "28", Freq: 14.2, FreqRx: 14.21, Comment: "local"}

	// Remote cleared zones and frequencies (explicit nulls).
	cleared := fetch(t, map[string]any{
		"id": 42, "station_id": 1, "qso_date": "2026-05-01 12:30:00",
		"mode": "SSB", "call": "SP9MOA", "band": "20m",
		"cqz": nil, "ituz": nil, "freq": nil, "freq_rx": nil,
		"comment": "remote",
	})
	m := mergeRemoteQSO(local, cleared)
	if m.CQZone != "" {
		t.Errorf("CQZone = %q, want cleared", m.CQZone)
	}
	if m.ITUZone != "" {
		t.Errorf("ITUZone = %q, want cleared", m.ITUZone)
	}
	if m.Freq != 0 {
		t.Errorf("Freq = %v, want cleared", m.Freq)
	}
	if m.FreqRx != 0 {
		t.Errorf("FreqRx = %v, want cleared", m.FreqRx)
	}
	if m.Comment != "remote" {
		t.Errorf("Comment = %q, want remote (string fields still apply)", m.Comment)
	}

	// Absent fields leave the local values untouched.
	absent := fetch(t, map[string]any{
		"id": 42, "station_id": 1, "qso_date": "2026-05-01 12:30:00",
		"mode": "SSB", "call": "SP9MOA", "band": "20m",
	})
	m = mergeRemoteQSO(local, absent)
	if m.CQZone != "15" || m.ITUZone != "28" || m.Freq != 14.2 || m.FreqRx != 14.21 {
		t.Errorf("absent fields must keep local values: zones=%q/%q freq=%v/%v",
			m.CQZone, m.ITUZone, m.Freq, m.FreqRx)
	}

	// Empty → populated still applies.
	populated := fetch(t, map[string]any{
		"id": 42, "station_id": 1, "qso_date": "2026-05-01 12:30:00",
		"mode": "SSB", "call": "SP9MOA", "band": "20m",
		"cqz": 16, "ituz": 29, "freq": "21300000", "freq_rx": "21301000",
	})
	m = mergeRemoteQSO(&qso.QSO{}, populated)
	if m.CQZone != "16" || m.ITUZone != "29" {
		t.Errorf("populated zones = %q/%q, want 16/29", m.CQZone, m.ITUZone)
	}
	if m.Freq != 21.3 || m.FreqRx != 21.301 {
		t.Errorf("populated freq = %v/%v, want 21.3/21.301", m.Freq, m.FreqRx)
	}
}

// TestFollowUpSyncCompletionDoesNotCloseForm verifies serialized follow-up
// PATCH results never close an open form — the form already closed at the
// original save.
func TestFollowUpSyncCompletionDoesNotCloseForm(t *testing.T) {
	le := newTestEditorWithDB(t, "", "", "", "OP", "JO90")
	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501", TimeOn: "120000"}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)

	upd, _ := le.Update(editorMsg{saved: id, gen: le.gen, wlSyncFollowUp: true})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Error("a follow-up PATCH completion must not close the form")
	}
}

// TestApplyRemoteRefresh_RejectsForeignEditorAndDatabase reproduces the
// reported bug: a refresh started in logbook A whose response lands after
// switching to logbook B must never overwrite B's contact — even when both
// rows share the same remote Wavelog id.
func TestApplyRemoteRefresh_RejectsForeignEditorAndDatabase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(77)})
	}))
	defer srv.Close()

	// Logbook A: an editor opens a synced contact and a fetch starts.
	leA := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")
	qA := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", Comment: "local A", WavelogID: 77}
	idA := insertTestQSO(t, leA.db, qA)
	qA.ID = idA
	leA.editing = qA
	leA.mode = edModeEdit
	leA.fillEditForm(qA)

	em := execCmd(leA.fetchRemoteCopy(77, idA)).(editorMsg)
	if em.wlFetchQSO == nil {
		t.Fatalf("fetch failed: %q", em.wlFetchErr)
	}

	// Logbook B: a fresh editor (new generation, new database) edits a
	// contact that happens to share remote id 77.
	leB := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")
	qB := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240502",
		TimeOn: "130000", Comment: "local B", WavelogID: 77}
	idB := insertTestQSO(t, leB.db, qB)
	qB.ID = idB
	leB.editing = qB
	leB.mode = edModeEdit
	leB.fillEditForm(qB)

	// A's delayed response arrives at B's editor — identical remote id,
	// fresh edit session. It must be rejected without touching B's row.
	applied, err := leB.ApplyRemoteRefresh(em.wlFetchQSO, remoteRefreshRequest{
		gen: em.wlFetchGen, db: em.wlFetchDB, localID: em.wlFetchQSOID, rev: em.wlFetchRev,
	})
	if err != nil {
		t.Fatalf("ApplyRemoteRefresh: %v", err)
	}
	if applied {
		t.Fatal("foreign-editor refresh must not apply")
	}
	stored, err := store.GetQSOByID(leB.db, idB)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "local B" {
		t.Errorf("B's row overwritten: Comment = %q, want local B", stored.Comment)
	}
	if leB.fields[qefComment].Value() != "local B" {
		t.Errorf("B's form overwritten: comment = %q, want local B", leB.fields[qefComment].Value())
	}
}

// TestApplyRemoteRefresh_RejectsDifferentLocalRow verifies that even within
// the same editor, a response is only applied to the row it was issued for —
// opening another contact (possibly sharing the remote id) invalidates it.
func TestApplyRemoteRefresh_RejectsDifferentLocalRow(t *testing.T) {
	le := newTestEditorWithDB(t, "", "wl2_test", "1", "Szymon", "KO00ca")

	q1 := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", Comment: "row 1", WavelogID: 77}
	id1 := insertTestQSO(t, le.db, q1)
	q1.ID = id1

	// A second local row shares the same remote id.
	q2 := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240502",
		TimeOn: "130000", Comment: "row 2", WavelogID: 77}
	id2 := insertTestQSO(t, le.db, q2)
	q2.ID = id2

	// The fetch was issued for row 1, but by the time the response
	// returns, row 2 is being edited.
	le.editing = q2
	le.mode = edModeEdit
	le.fillEditForm(q2)
	le.editRev = 0

	applied, err := le.ApplyRemoteRefresh(
		&wavelog.QSOData{ID: 77, Call: "SP9MOA", Comment: "remote edit"},
		remoteRefreshRequest{gen: le.gen, db: le.db, localID: id1, rev: 0})
	if err != nil {
		t.Fatalf("ApplyRemoteRefresh: %v", err)
	}
	if applied {
		t.Fatal("refresh issued for another local row must not apply")
	}
	stored, err := store.GetQSOByID(le.db, id2)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "row 2" {
		t.Errorf("row 2 overwritten: Comment = %q, want row 2", stored.Comment)
	}
}

// TestEditSaveMarksPendingBeforePatchCompletes reproduces the reported bug:
// while the PATCH is blocked in flight, the row must already be durably
// dirty — a crash at that point must not leave a changed row that a later
// remote refresh would treat as synced and overwrite.
func TestEditSaveMarksPendingBeforePatchCompletes(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("edited")

	done := make(chan editorMsg, 1)
	go func() { done <- execCmd(le.doSave()).(editorMsg) }()

	// Wait until the PATCH is actually in flight.
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("PATCH never started")
	}

	// Intermediate state: the edit is committed AND the row is dirty.
	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "edited" {
		t.Errorf("intermediate comment = %q, want edited", stored.Comment)
	}
	if !stored.WavelogDirty {
		t.Error("intermediate WavelogDirty = false; the row must be pending while the PATCH is in flight")
	}

	close(release)
	em := <-done
	if !em.wlSyncOK {
		t.Errorf("sync result: ok=%v err=%q, want ok", em.wlSyncOK, em.wlSyncErr)
	}

	stored, err = store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogDirty {
		t.Error("WavelogDirty should be cleared after the PATCH acknowledged")
	}
}

// TestEditSaveMissingCredentialsMarksPending verifies that a synced row whose
// Wavelog credentials are missing still gets the durable pending mark —
// previously this branch skipped dirty marking entirely.
func TestEditSaveMissingCredentialsMarksPending(t *testing.T) {
	le := newTestEditorWithDB(t, "", "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id

	le.editing = q
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("edited")

	em := execCmd(le.doSave()).(editorMsg)
	if em.err != nil {
		t.Fatalf("save failed: %v", em.err)
	}
	if em.wlSyncOK || em.wlSyncGone || em.wlSyncErr != "" || em.wlSyncPending {
		t.Errorf("no sync flags expected without credentials, got %+v", em)
	}
	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "edited" {
		t.Errorf("comment = %q, want edited", stored.Comment)
	}
	if !stored.WavelogDirty {
		t.Error("WavelogDirty should be set when credentials are missing")
	}
}

// TestOverlappingSavesSerializeRemoteUpdates reproduces the out-of-order
// PATCH bug: save A stalls at the server while save B reaches it first —
// previously B applied "second", then A completed and overwrote the server
// with "first", leaving local="second" but remote="first" with dirty=false.
// Remote updates must be serialized per contact, coalescing queued edits to
// the newest revision, and the pending flag must survive until that revision
// is acknowledged.
func TestOverlappingSavesSerializeRemoteUpdates(t *testing.T) {
	var mu sync.Mutex
	var applied []string
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Comment string `json:"comment"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		mu.Lock()
		requestCount++
		i := requestCount
		mu.Unlock()
		if i == 1 {
			close(firstStarted)
			<-releaseFirst // the first PATCH stalls
		}
		mu.Lock()
		applied = append(applied, body.Comment)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	le := NewLogbookEditor(LogbookEditorConfig{
		DB: m.App.DB, WLURL: srv.URL, WLKey: "wl2_test", WLStationID: "1",
		WLLastFetchedID: 0, StationOperator: "OP", StationGrid: "JO90",
	})
	m.ui.logbookEditor = le

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	q.ID = id
	le.editing = q
	le.fillEditForm(q)

	// Save one — its PATCH stalls at the server.
	le.fields[qefComment].SetValue("first")
	done1 := make(chan editorMsg, 1)
	go func() { done1 <- execCmd(le.doSave()).(editorMsg) }()
	<-firstStarted

	// Save two while the first PATCH is on the wire — it must be queued,
	// not sent: an out-of-order completion would end with the server
	// holding "first" while the local row says "second".
	le.fields[qefComment].SetValue("second")
	em2 := execCmd(le.doSave()).(editorMsg)
	if em2.err != nil {
		t.Fatalf("second local save failed: %v", em2.err)
	}
	if !em2.wlSyncPending {
		t.Fatal("second save must be queued while a PATCH is in flight")
	}
	if !le.sync.queued[contactSyncKey{localID: id, url: srv.URL}] {
		t.Fatal("second save must mark the contact as queued")
	}

	// The first PATCH completes; the owner loop dispatches the follow-up
	// carrying the NEWEST revision.
	close(releaseFirst)
	em1 := <-done1
	if !em1.wlSyncOK {
		t.Fatalf("first save: ok=%v err=%q, want ok", em1.wlSyncOK, em1.wlSyncErr)
	}
	followUp := m.handleQSOSyncCompletion(em1)
	if followUp == nil {
		t.Fatal("expected the queued follow-up PATCH")
	}
	em3 := execCmd(followUp).(editorMsg)
	if !em3.wlSyncOK {
		t.Fatalf("follow-up save: ok=%v err=%q, want ok", em3.wlSyncOK, em3.wlSyncErr)
	}
	if m.handleQSOSyncCompletion(em3) != nil {
		t.Fatal("no further PATCH should be dispatched")
	}

	// The server must have applied the requests in save order — never
	// "second" then "first" — and end with the newest value.
	mu.Lock()
	gotOrder := append([]string(nil), applied...)
	mu.Unlock()
	if len(gotOrder) != 2 || gotOrder[0] != "first" || gotOrder[1] != "second" {
		t.Fatalf("server applied requests in order %v, want [first second]", gotOrder)
	}

	dirty, err := store.QSOHasPendingSync(le.db, id)
	if err != nil {
		t.Fatalf("QSOHasPendingSync: %v", err)
	}
	if dirty {
		t.Error("pending flag should clear only after the newest revision was acknowledged")
	}
	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Comment != "second" {
		t.Errorf("local comment = %q, want second", stored.Comment)
	}
}

// TestEditorSaveRetainsDatabaseAcrossLogbookSwitch reproduces the reported
// bug: an editor PATCH dispatched against logbook A, with the operator
// switching to logbook B while the PATCH is on the wire, used to lose its
// database (the retired A was closed) — the local acknowledgement write then
// failed. The editor now holds a database lease for the worker's whole
// lifetime, so the ack persists and the row is durably cleared.
func TestEditorSaveRetainsDatabaseAcrossLogbookSwitch(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
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

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id, err := store.InsertQSO(dbA, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	q.ID = id

	le := NewLogbookEditor(LogbookEditorConfig{
		DB: dbA, WLURL: srv.URL, WLKey: "wl2_test", WLStationID: "1",
		StationOperator: "Szymon", StationGrid: "KO00ca",
		KeepAlive: a.KeepDBAlive,
	})
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("edited")

	done := make(chan editorMsg, 1)
	go func() { done <- execCmd(le.doSave()).(editorMsg) }()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("PATCH never started")
	}

	// Switch logbooks while the PATCH is in flight: the editor's database
	// lease must keep A open until the worker's local writes finish.
	if err := a.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { a.DB.Close() })

	close(release)
	em := <-done
	if !em.wlSyncOK {
		t.Errorf("sync result: ok=%v incomplete=%v err=%q; the lease must keep the ack write alive",
			em.wlSyncOK, em.wlSyncIncomplete, em.wlSyncErr)
	}
	if em.wlSyncIncomplete {
		t.Error("acknowledgement must persist despite the logbook switch")
	}

	// The durable ack: reopen A's database file and verify the dirty flag
	// is cleared.
	reopened, err := store.Open(dbPathA)
	if err != nil {
		t.Fatalf("reopen A: %v", err)
	}
	defer reopened.Close()
	stored, err := store.GetQSOByID(reopened, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogDirty {
		t.Error("WavelogDirty must be cleared after the ack persisted across the switch")
	}
}

// TestFailedAckPersistenceReportsIncompleteSync verifies the semantics
// required for a lost database: when the remote PATCH succeeded but the
// local acknowledgement could not be persisted, the result is reported as an
// INCOMPLETE synchronization (never full success) and the row stays durably
// pending.
func TestFailedAckPersistenceReportsIncompleteSync(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	dbPath := filepath.Join(t.TempDir(), "a.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "before", WavelogID: 42}
	id, err := store.InsertQSO(db, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	q.ID = id

	le := NewLogbookEditor(LogbookEditorConfig{
		DB: db, WLURL: srv.URL, WLKey: "wl2_test", WLStationID: "1",
		StationOperator: "Szymon", StationGrid: "KO00ca",
	})
	le.editing = q
	le.mode = edModeEdit
	le.fillEditForm(q)
	le.fields[qefComment].SetValue("edited")

	done := make(chan editorMsg, 1)
	go func() { done <- execCmd(le.doSave()).(editorMsg) }()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("PATCH never started")
	}

	// Reproduce the lost-database failure mode: the database is gone before
	// the worker's acknowledgement write runs.
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	close(release)
	em := <-done

	if em.wlSyncOK {
		t.Error("a failed acknowledgement must not be reported as wlSyncOK")
	}
	if !em.wlSyncIncomplete {
		t.Errorf("result: ok=%v incomplete=%v err=%q; want incomplete synchronization",
			em.wlSyncOK, em.wlSyncIncomplete, em.wlSyncErr)
	}

	// The row stays durably pending for a later retry.
	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer reopened.Close()
	stored, err := store.GetQSOByID(reopened, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if !stored.WavelogDirty {
		t.Error("WavelogDirty must remain set when the acknowledgement could not be persisted")
	}
}

// TestEditSaveKeepsWavelogIDAssignedDuringUpload reproduces the reported data
// loss: the form snapshot keeps WavelogID 0 while an in-flight upload
// attaches the remote id in the database; saving the form then overwrote the
// id with the stale zero (77 → 0, dirty=false), permanently unlinking the
// contact and skipping the follow-up PATCH. The save must decide
// synced/dirty from the DATABASE state atomically with the write, never
// write the stale id, and PATCH the fresh row.
func TestEditSaveKeepsWavelogIDAssignedDuringUpload(t *testing.T) {
	var patchedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/qso/77" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&patchedBody); err != nil {
			t.Errorf("decode patch body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(77)})
	}))
	defer srv.Close()

	le := newTestEditorWithDB(t, srv.URL, "wl2_test", "1", "Szymon", "KO00ca")

	q := &qso.QSO{Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "old"}
	id := insertTestQSO(t, le.db, q)
	row, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	le.editing = row
	le.fillEditForm(row) // form snapshot: WavelogID 0

	// The in-flight upload attaches the remote id while the form is open.
	if err := store.SetWavelogID(le.db, id, 77); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}

	le.fields[qefComment].SetValue("edited during upload")
	em, ok := execCmd(le.doSave()).(editorMsg)
	if !ok || em.err != nil {
		t.Fatalf("save failed: %v", em.err)
	}
	if em.saved != id {
		t.Fatalf("saved = %d, want %d", em.saved, id)
	}
	if !em.wlSyncOK {
		t.Fatalf("wlSyncOK = %v (err=%q), want PATCH of the newer edit", em.wlSyncOK, em.wlSyncErr)
	}
	if patchedBody == nil {
		t.Fatal("no PATCH received by the mock server")
	}
	if patchedBody["comment"] != "edited during upload" {
		t.Errorf("PATCH comment = %v, want edited during upload", patchedBody["comment"])
	}
	stored, err := store.GetQSOByID(le.db, id)
	if err != nil {
		t.Fatalf("GetQSOByID after save: %v", err)
	}
	if stored.WavelogID != 77 {
		t.Fatalf("WavelogID = %d, want 77 — the save must not wipe the id", stored.WavelogID)
	}
	if stored.WavelogDirty {
		t.Error("row should be clean after the PATCH acknowledged the edit")
	}
}

// TestPendingSyncRetrySyncsDirtyRows verifies the bulk pending-sync retry:
// the queue is rebuilt from the DATABASE (rows with a remote id and a
// pending local edit — the backlog of failed PATCHes and offline saves
// survives restarts), and EVERY contact is PATCHed through the shared
// per-contact coordinator.
func TestPendingSyncRetrySyncsDirtyRows(t *testing.T) {
	var mu sync.Mutex
	var patched []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode patch body: %v", err)
		}
		mu.Lock()
		patched = append(patched, fmt.Sprintf("%v|%v", body["call"], body["comment"]))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"
	m.initLogbookEditor()
	le := m.ui.logbookEditor

	q1 := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "edit one", WavelogID: 42}
	id1 := insertTestQSO(t, m.App.DB, q1)
	q2 := &qso.QSO{Call: "SP9BBB", Band: "40m", Mode: "CW", QSODate: "20240502",
		TimeOn: "130000", RSTSent: "599", RSTRcvd: "579", Comment: "edit two", WavelogID: 43}
	id2 := insertTestQSO(t, m.App.DB, q2)
	// A clean synced row must not be PATCHed.
	insertTestQSO(t, m.App.DB, &qso.QSO{Call: "SP9CCC", Band: "20m", Mode: "SSB",
		QSODate: "20240503", TimeOn: "140000", RSTSent: "59", RSTRcvd: "59", WavelogID: 44})

	if err := store.SetWavelogDirty(m.App.DB, id1, true); err != nil {
		t.Fatalf("SetWavelogDirty(1): %v", err)
	}
	if err := store.SetWavelogDirty(m.App.DB, id2, true); err != nil {
		t.Fatalf("SetWavelogDirty(2): %v", err)
	}

	listEm, ok := execCmd(le.retryPendingSync()).(editorMsg)
	if !ok || len(listEm.wlRetryIDs) != 2 {
		t.Fatalf("retry list = %#v, want 2 contact ids", listEm)
	}

	// The model routes every contact through the shared coordinator.
	upd, c := m.Update(listEm)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the retry PATCHes were not queued")
	}
	var results []editorMsg
	var walk func(cmd tea.Cmd)
	walk = func(cmd tea.Cmd) {
		msg := execCmd(cmd)
		switch v := msg.(type) {
		case tea.Cmd:
			walk(v) // a command returned another command (reconcile → PATCH worker)
		case editorMsg:
			if v.wlSyncFollowUp {
				results = append(results, v)
			}
		case tea.BatchMsg:
			for _, nested := range v {
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
	if len(results) != 2 {
		t.Fatalf("got %d PATCH completions, want 2", len(results))
	}
	for _, r := range results {
		if !r.wlSyncOK && !r.wlSyncGone {
			t.Fatalf("retry PATCH failed: ok=%v err=%q", r.wlSyncOK, r.wlSyncErr)
		}
		_ = m.handleQSOSyncCompletion(r)
	}

	mu.Lock()
	got := append([]string(nil), patched...)
	mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("server received %d PATCHes: %v, want 2", len(got), got)
	}
	for _, want := range []string{"SP9AAA|edit one", "SP9BBB|edit two"} {
		found := false
		for _, p := range got {
			if p == want {
				found = true
			}
		}
		if !found {
			t.Errorf("PATCHes = %v, want %s present", got, want)
		}
	}
	for _, id := range []int64{id1, id2} {
		stored, err := store.GetQSOByID(m.App.DB, id)
		if err != nil {
			t.Fatalf("GetQSOByID(%d): %v", id, err)
		}
		if stored.WavelogDirty {
			t.Errorf("row %d still dirty after the retry ack", id)
		}
		if stored.WavelogID == 0 {
			t.Errorf("row %d lost its remote id", id)
		}
	}
}

// TestPendingSyncRetrySerializesWithSave reproduces the reported overwrite:
// the retry used to PATCH outside the per-contact coordinator, so a save
// made while the retry was on the wire could complete first and the delayed
// retry then overwrote the server with the older content (remote=old,
// local=new, dirty=false). The retry must go through the SAME coordinator —
// the save queues behind it and its follow-up pushes the newest revision.
func TestPendingSyncRetrySerializesWithSave(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var mu sync.Mutex
	var applied []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode patch body: %v", err)
		}
		mu.Lock()
		applied = append(applied, fmt.Sprintf("%v", body["comment"]))
		first := len(applied) == 1
		mu.Unlock()
		if first {
			close(firstStarted)
			<-releaseFirst
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"
	m.initLogbookEditor()
	le := m.ui.logbookEditor

	q := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "old", WavelogID: 42}
	id := insertTestQSO(t, m.App.DB, q)
	if err := store.SetWavelogDirty(m.App.DB, id, true); err != nil {
		t.Fatalf("SetWavelogDirty: %v", err)
	}

	listEm := execCmd(le.retryPendingSync()).(editorMsg)
	if len(listEm.wlRetryIDs) != 1 {
		t.Fatalf("retry list = %v, want 1 contact", listEm.wlRetryIDs)
	}
	upd, c := m.Update(listEm)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the retry PATCH was not queued")
	}
	// With a single contact, tea.Batch returns the worker command itself
	// (queueContactReconcile dispatches synchronously during Update).
	var workerCmd tea.Cmd
	switch v := any(c).(type) {
	case tea.Cmd:
		workerCmd = v
	case tea.BatchMsg:
		if len(v) == 1 {
			workerCmd = v[0]
		}
	}
	if workerCmd == nil {
		t.Fatalf("expected the PATCH worker command, got %T", c)
	}
	retryCh := make(chan editorMsg, 1)
	go func() { retryCh <- execCmd(workerCmd).(editorMsg) }()

	select {
	case <-firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("retry PATCH never started")
	}

	// The operator saves a NEWER edit while the retry PATCH is in flight —
	// the save must queue behind the retry on the shared coordinator.
	row, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	le.editing = row
	le.fillEditForm(row)
	le.fields[qefComment].SetValue("new")
	saveEm := execCmd(le.doSave()).(editorMsg)
	if saveEm.err != nil {
		t.Fatalf("save failed: %v", saveEm.err)
	}
	if !saveEm.wlSyncPending {
		t.Fatal("the save must queue behind the in-flight retry PATCH")
	}

	close(releaseFirst)
	retryEm := <-retryCh
	if !retryEm.wlSyncOK {
		t.Fatalf("retry PATCH: ok=%v err=%q", retryEm.wlSyncOK, retryEm.wlSyncErr)
	}

	// The queued save follow-up must now push the newest revision.
	followUp := m.handleQSOSyncCompletion(retryEm)
	if followUp == nil {
		t.Fatal("the queued save follow-up was not dispatched")
	}
	followEm, ok := execCmd(followUp).(editorMsg)
	if !ok || !followEm.wlSyncOK {
		t.Fatalf("follow-up PATCH: ok=%v err=%q", followEm.wlSyncOK, followEm.wlSyncErr)
	}
	_ = m.handleQSOSyncCompletion(followEm)

	mu.Lock()
	got := append([]string(nil), applied...)
	mu.Unlock()
	if len(got) != 2 || got[0] != "old" || got[1] != "new" {
		t.Fatalf("server PATCHes = %v, want [old new] — the delayed retry must never overwrite the newer edit", got)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID after chain: %v", err)
	}
	if stored.Comment != "new" {
		t.Errorf("local comment = %q, want new", stored.Comment)
	}
	if stored.WavelogDirty {
		t.Error("row should be clean after the follow-up ack")
	}
}

// TestPendingSyncRetryIncompleteAckIsReported reproduces the reported bug:
// the server accepts the PATCH but SQLite rejects the local pending-flag
// acknowledgement. The completion is neither synced, failed, nor errored —
// the old aggregation fell through to "no pending changes" even though the
// contact is still pending. The batch must count it as unconfirmed, warn
// instead of claiming success, and keep the row dirty so Alt+P can retry it.
func TestPendingSyncRetryIncompleteAckIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": remoteQSODoc(42)})
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL = srv.URL
	wl.APIKey = "wl2_test"
	wl.Enabled = true
	wl.StationProfileID = "1"
	m.initLogbookEditor()
	le := m.ui.logbookEditor

	q := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", Comment: "pending ack", WavelogID: 42}
	id := insertTestQSO(t, m.App.DB, q)
	if err := store.SetWavelogDirty(m.App.DB, id, true); err != nil {
		t.Fatalf("SetWavelogDirty: %v", err)
	}

	// Block the pending-flag acknowledgement: the server accepted the PATCH,
	// but the local dirty clear fails (e.g. a transient SQLite write error).
	if _, err := m.App.DB.Exec(`
		CREATE TRIGGER block_pending_ack
		BEFORE UPDATE OF wavelog_dirty ON qsos
		WHEN NEW.wavelog_dirty = 0
		BEGIN
			SELECT RAISE(ABORT, 'blocked');
		END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	listEm := execCmd(le.retryPendingSync()).(editorMsg)
	if len(listEm.wlRetryIDs) != 1 {
		t.Fatalf("retry list = %v, want 1 contact", listEm.wlRetryIDs)
	}
	upd, c := m.Update(listEm)
	m = upd.(*Model)
	if c == nil {
		t.Fatal("the retry PATCH was not queued")
	}

	// Run the dispatched command(s) recursively — the worker may be the
	// command itself or nested inside a tea.BatchMsg.
	var runAll func(tea.Cmd) []editorMsg
	runAll = func(cmd tea.Cmd) []editorMsg {
		var out []editorMsg
		switch v := any(execCmd(cmd)).(type) {
		case tea.Cmd:
			out = append(out, runAll(v)...)
		case editorMsg:
			if v.wlSyncFollowUp {
				out = append(out, v)
			}
		case tea.BatchMsg:
			for _, nested := range v {
				out = append(out, runAll(nested)...)
			}
		}
		return out
	}
	completions := runAll(c)
	if len(completions) != 1 {
		t.Fatalf("got %d PATCH completions, want 1", len(completions))
	}
	workerEm := completions[0]
	if !workerEm.wlSyncIncomplete {
		t.Fatalf("PATCH completion: incomplete=%v ok=%v err=%q — the blocked ack must be incomplete",
			workerEm.wlSyncIncomplete, workerEm.wlSyncOK, workerEm.wlSyncErr)
	}
	if followUp := m.handleQSOSyncCompletion(workerEm); followUp != nil {
		t.Fatalf("drained chain returned a follow-up, got %T", followUp)
	}

	m.toasts.mu.Lock()
	toasts := append([]Toast(nil), m.toasts.items...)
	m.toasts.mu.Unlock()
	var warned bool
	for _, toast := range toasts {
		if toast.Level == ToastWarning && strings.Contains(toast.Message, "unconfirmed") {
			warned = true
		}
		if toast.Message == "Wavelog: no pending changes" {
			t.Fatal("the incomplete acknowledgement must never be reported as no pending changes")
		}
	}
	if !warned {
		t.Fatalf("expected an unconfirmed warning toast, got %#v", toasts)
	}

	// The row must stay pending — the retry entry point still sees it.
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if !stored.WavelogDirty {
		t.Fatal("row must stay dirty after the failed acknowledgement — retryability lost")
	}

	// Remove the blocker: the next Alt+P retry must be able to clear it.
	if _, err := m.App.DB.Exec(`DROP TRIGGER block_pending_ack`); err != nil {
		t.Fatalf("drop trigger: %v", err)
	}
	listEm2 := execCmd(le.retryPendingSync()).(editorMsg)
	if len(listEm2.wlRetryIDs) != 1 {
		t.Fatalf("second retry list = %v, want the still-pending contact", listEm2.wlRetryIDs)
	}
	upd, c2 := m.Update(listEm2)
	m = upd.(*Model)
	if c2 == nil {
		t.Fatal("the second retry PATCH was not queued")
	}
	completions2 := runAll(c2)
	if len(completions2) != 1 {
		t.Fatalf("second retry: got %d PATCH completions, want 1", len(completions2))
	}
	workerEm2 := completions2[0]
	if !workerEm2.wlSyncOK {
		t.Fatalf("second retry PATCH: ok=%v err=%q", workerEm2.wlSyncOK, workerEm2.wlSyncErr)
	}
	_ = m.handleQSOSyncCompletion(workerEm2)
	stored, err = store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID after retry: %v", err)
	}
	if stored.WavelogDirty {
		t.Error("row should be clean after the retried acknowledgement")
	}
}

// TestPendingSyncRetryKeyOpensConfirm verifies the Alt+P entry point: the
// pending backlog is counted from the database and shown in the confirm
// dialog, and confirming dispatches the retry.
func TestPendingSyncRetryKeyOpensConfirm(t *testing.T) {
	le := newTestEditorWithDB(t, "https://log.example.com", "wl2_test", "1", "Szymon", "KO00ca")
	q := &qso.QSO{Call: "SP9AAA", Band: "20m", Mode: "SSB", QSODate: "20240501",
		TimeOn: "120000", RSTSent: "59", RSTRcvd: "59", WavelogID: 42}
	id := insertTestQSO(t, le.db, q)
	if err := store.SetWavelogDirty(le.db, id, true); err != nil {
		t.Fatalf("SetWavelogDirty: %v", err)
	}

	upd, cmd := le.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModAlt})
	le = upd.(*LogbookEditor)
	if cmd != nil {
		t.Fatal("Alt+P should only open the confirm dialog")
	}
	if le.mode != edModeConfirmWLSyncRetry {
		t.Fatalf("mode = %v, want edModeConfirmWLSyncRetry", le.mode)
	}
	if le.wlPendingCount != 1 {
		t.Fatalf("wlPendingCount = %d, want 1", le.wlPendingCount)
	}

	le.View() // materializes the dialog
	upd, cmd = le.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	le = upd.(*LogbookEditor)
	if cmd == nil {
		t.Fatal("confirming should dispatch the retry")
	}
}
