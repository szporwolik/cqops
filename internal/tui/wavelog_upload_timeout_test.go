package tui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Upload after an unknown outcome.
//
// Live incident, 2026-10-29: Wavelog degraded, held a QSO insert past the
// client's 10 s deadline and committed it anyway. CQOps reported "QSO upload
// failed", fell back to the ADIF import (a SECOND write of the same contact),
// and left wavelog_id = 0 — so the contact was on the server while the local
// row said "unsent", and the next Ctrl+W would have sent it a third time.
//
// The rule now: a timeout (or a 5xx) is an unknown outcome. Ask the server
// whether the contact is there, link it if it is, and only write again when it
// is verifiably absent.
// =============================================================================

// qsoUUID is the identity the fake server reports back for a created contact.
const fakeRemoteID = 4242

// fakeWavelogTimeoutServer answers the JSON create only after the client's
// deadline."commit" decides whether the write lands anyway (which is what the
// real server did), and postCount reports how many writes reached it.
type fakeWavelog struct {
	srv       *httptest.Server
	posts     atomic.Int32
	adifPaths atomic.Int32 // POSTs that carried import_type=adif
}

func newFakeWavelog(t *testing.T, commit bool, createStatus int) *fakeWavelog {
	t.Helper()
	f := &fakeWavelog{}
	committed := atomic.Bool{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/qso"):
			f.posts.Add(1)
			body := make([]byte, 4096)
			n, _ := r.Body.Read(body)
			if strings.Contains(string(body[:n]), `"import_type"`) {
				f.adifPaths.Add(1)
			}
			if commit {
				committed.Store(true)
			}
			time.Sleep(120 * time.Millisecond) // past the test's 40 ms deadline
			if createStatus >= 500 {
				w.WriteHeader(createStatus)
				fmt.Fprint(w, `{"error":{"code":"internal_error","message":"upstream died"}}`)
				return
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"data":{"id":%d},"meta":{}}`, fakeRemoteID)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/qso"):
			// The callsign-scoped list backfillRemoteID matches against.
			if committed.Load() {
				fmt.Fprintf(w, `{"data":[{"id":%d,"call":"DL1AA","band":"80m","mode":"FT8",`+
					`"qso_date":"2026-10-29 23:01:15"}],"meta":{}}`, fakeRemoteID)
				return
			}
			fmt.Fprint(w, `{"data":[],"meta":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func TestUploadTimeoutLinksTheContactInsteadOfSendingItTwice(t *testing.T) {
	f := newFakeWavelog(t, true /* commit anyway */, http.StatusCreated)
	defer wavelog.OverrideTimeouts(40*time.Millisecond, 40*time.Millisecond, 40*time.Millisecond)()

	m := newLifecycleTestModel(t)
	q := &qso.QSO{Call: "DL1AA", QSODate: "20261029", TimeOn: "230115", Band: "80m", Mode: "FT8", Freq: 3.5735}
	id := insertTestQSO(t, m.App.DB, q)

	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil || stored == nil {
		t.Fatalf("read QSO: %v", err)
	}
	ok, isDup, remoteID, _, uerr := postQSOSingle(f.srv.URL, "wl2_test", "1", stored, m.App.DB, stored.WavelogDirtyRev)

	if uerr != nil {
		t.Fatalf("upload reported an error (%v) although the contact is on the server", uerr)
	}
	if !ok {
		t.Fatal("upload reported failure although the contact is on the server")
	}
	if !isDup {
		t.Error("a reconciled contact must count as already present, not as a fresh create")
	}
	if remoteID != fakeRemoteID {
		t.Errorf("remote id = %d, want %d", remoteID, fakeRemoteID)
	}
	if got := f.posts.Load(); got != 1 {
		t.Errorf("%d write requests reached the server, want 1 — the ADIF fallback re-sent a contact the timeout had already stored", got)
	}
	if got := f.adifPaths.Load(); got != 0 {
		t.Errorf("%d ADIF fallbacks were sent, want 0", got)
	}
	if status := getWavelogStatus(t, m.App.DB, id); status != "yes" {
		t.Errorf("local row is still unsent (%q) — the next upload would send it again", status)
	}
}

func TestUploadTimeoutLeavesItUnsentWhenTheServerDoesNotHaveIt(t *testing.T) {
	f := newFakeWavelog(t, false /* the write never lands */, http.StatusCreated)
	defer wavelog.OverrideTimeouts(40*time.Millisecond, 40*time.Millisecond, 40*time.Millisecond)()

	m := newLifecycleTestModel(t)
	q := &qso.QSO{Call: "DL1AA", QSODate: "20261029", TimeOn: "230115", Band: "80m", Mode: "FT8", Freq: 3.5735}
	id := insertTestQSO(t, m.App.DB, q)
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil || stored == nil {
		t.Fatalf("read QSO: %v", err)
	}

	ok, _, _, _, uerr := postQSOSingle(f.srv.URL, "wl2_test", "1", stored, m.App.DB, stored.WavelogDirtyRev)
	if ok || uerr == nil {
		t.Fatal("an upload the server never confirmed must not be reported as done")
	}
	if !wavelog.IsTimeout(uerr) {
		t.Errorf("error %v is not reported as a timeout", uerr)
	}
	if !strings.Contains(uerr.Error(), "saved locally") {
		t.Errorf("message %q does not tell the operator the contact is safe", uerr.Error())
	}
	if got := f.posts.Load(); got != 1 {
		t.Errorf("%d writes reached the server, want 1 — a second write may duplicate the contact the first one may still perform", got)
	}
	if status := getWavelogStatus(t, m.App.DB, id); status != "no" {
		t.Errorf("local row claims to be uploaded (%q) although the server never confirmed it", status)
	}
}

func TestUploadServerErrorLinksTheContactInsteadOfFallingBack(t *testing.T) {
	f := newFakeWavelog(t, true, http.StatusBadGateway)
	defer wavelog.OverrideTimeouts(200*time.Millisecond, 200*time.Millisecond, 200*time.Millisecond)()

	m := newLifecycleTestModel(t)
	q := &qso.QSO{Call: "DL1AA", QSODate: "20261029", TimeOn: "230115", Band: "80m", Mode: "FT8", Freq: 3.5735}
	id := insertTestQSO(t, m.App.DB, q)
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil || stored == nil {
		t.Fatalf("read QSO: %v", err)
	}

	ok, _, remoteID, _, uerr := postQSOSingle(f.srv.URL, "wl2_test", "1", stored, m.App.DB, stored.WavelogDirtyRev)
	if uerr != nil || !ok {
		t.Fatalf("ok=%v err=%v — a 502 for a write that reached the database must be reconciled, not failed", ok, uerr)
	}
	if remoteID != fakeRemoteID {
		t.Errorf("remote id = %d, want %d", remoteID, fakeRemoteID)
	}
	if got := f.adifPaths.Load(); got != 0 {
		t.Errorf("%d ADIF fallbacks were sent after a 5xx, want 0 — the contact was already there", got)
	}
	if status := getWavelogStatus(t, m.App.DB, id); status != "yes" {
		t.Errorf("local row is still unsent (%q)", status)
	}
}

// TestUploadPlainFailureStillFallsBack keeps the old behaviour where it is
// right: a request the server rejected outright (400) means the contact is not
// there, so the ADIF import path must still run.
func TestUploadPlainFailureStillFallsBack(t *testing.T) {
	var posts, adif atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/qso"):
			posts.Add(1)
			body := make([]byte, 4096)
			n, _ := r.Body.Read(body)
			if strings.Contains(string(body[:n]), `"import_type"`) {
				adif.Add(1)
				fmt.Fprint(w, `{"data":{"imported":1},"meta":{}}`)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"code":"validation_error","message":"bad mode"}}`)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/qso"):
			fmt.Fprint(w, `{"data":[],"meta":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := newLifecycleTestModel(t)
	id := insertTestQSO(t, m.App.DB, &qso.QSO{Call: "DL1AA", QSODate: "20261029", TimeOn: "230115", Band: "80m", Mode: "FT8"})
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil || stored == nil {
		t.Fatalf("read QSO: %v", err)
	}

	ok, _, _, _, uerr := postQSOSingle(srv.URL, "wl2_test", "1", stored, m.App.DB, stored.WavelogDirtyRev)
	if uerr != nil || !ok {
		t.Fatalf("ok=%v err=%v — the ADIF fallback should have stored the contact", ok, uerr)
	}
	if posts.Load() != 2 || adif.Load() != 1 {
		t.Errorf("writes=%d adif=%d, want 2 writes with exactly 1 ADIF fallback", posts.Load(), adif.Load())
	}
}
