package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

	if err := le.ApplyRemoteRefresh(em.wlFetchQSO); err != nil {
		t.Fatalf("ApplyRemoteRefresh: %v", err)
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

	if err := le.ApplyRemoteRefresh(&wavelog.QSOData{ID: 77}); err != nil {
		t.Fatalf("ApplyRemoteRefresh should be a no-op, got %v", err)
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
