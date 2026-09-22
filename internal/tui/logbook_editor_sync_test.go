package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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

// TestOlderPatchAckDoesNotClearNewerPendingEdit verifies the revision guard:
// when a second save happens while the first PATCH is still in flight, the
// first PATCH's success must not clear the pending flag that belongs to the
// second edit.
func TestOlderPatchAckDoesNotClearNewerPendingEdit(t *testing.T) {
	var mu sync.Mutex
	count := 0
	started := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
	release := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		i := count
		count++
		mu.Unlock()
		if i < 2 {
			close(started[i])
			<-release[i]
		}
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

	// First save — its PATCH blocks on release[0].
	le.fields[qefComment].SetValue("edit one")
	done1 := make(chan editorMsg, 1)
	go func() { done1 <- execCmd(le.doSave()).(editorMsg) }()
	<-started[0]

	// Second save while the first PATCH is still in flight.
	le.fields[qefComment].SetValue("edit two")
	done2 := make(chan editorMsg, 1)
	go func() { done2 <- execCmd(le.doSave()).(editorMsg) }()
	<-started[1]

	// The older PATCH succeeds first — it must NOT clear the newer pending edit.
	close(release[0])
	em1 := <-done1
	if !em1.wlSyncOK {
		t.Fatalf("first save: ok=%v err=%q, want ok", em1.wlSyncOK, em1.wlSyncErr)
	}
	dirty, err := store.QSOHasPendingSync(le.db, id)
	if err != nil {
		t.Fatalf("QSOHasPendingSync: %v", err)
	}
	if !dirty {
		t.Fatal("older PATCH acknowledgement cleared the newer pending edit")
	}

	// The newer PATCH's own acknowledgement clears it.
	close(release[1])
	em2 := <-done2
	if !em2.wlSyncOK {
		t.Fatalf("second save: ok=%v err=%q, want ok", em2.wlSyncOK, em2.wlSyncErr)
	}
	dirty, err = store.QSOHasPendingSync(le.db, id)
	if err != nil {
		t.Fatalf("QSOHasPendingSync: %v", err)
	}
	if dirty {
		t.Fatal("current PATCH acknowledgement should clear the pending flag")
	}
}
