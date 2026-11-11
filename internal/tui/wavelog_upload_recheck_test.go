package tui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Deferred recheck of an unknown upload outcome.
//
// Live case, 2026-10-30 00:53: WSJT-X auto-logged DK5WO, the write deadline
// (30 s) expired, CQOps left the row unsent and told the operator the state on
// the server was unknown — while Wavelog committed the insert a moment later.
// The contact was on the server all along; the first confirmation lookup simply
// ran while the server was still busy with the same insert.
//
// The rule now: an unknown outcome schedules a recheck. When the server
// confirms, the contact is linked and reported as confirmed. Only a recheck
// that never gets an answer leaves it unsent — and it never claims the server
// rejected it.
// =============================================================================

// fakeLateCommitWavelog models the degraded server: the create is committed
// (and only then visible in the callsign list) but the response arrives after
// the client's write deadline.
type fakeLateCommitWavelog struct {
	srv      *httptest.Server
	posts    atomic.Int32
	visible  atomic.Bool
	getFails atomic.Bool // the confirmation lookup itself cannot be answered
}

func newFakeLateCommitWavelog(t *testing.T) *fakeLateCommitWavelog {
	t.Helper()
	f := &fakeLateCommitWavelog{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/qso"):
			f.posts.Add(1)
			time.Sleep(120 * time.Millisecond) // past the 30 ms test deadline
			// The commit is invisible until the test publishes it — this is
			// the degraded server that stores the row AFTER the client's
			// confirmation lookup already ran.
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"data":{"id":%d},"meta":{}}`, fakeRemoteID)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/qso"):
			if f.getFails.Load() {
				http.Error(w, "gateway is sad", http.StatusBadGateway)
				return
			}
			if f.visible.Load() {
				fmt.Fprintf(w, `{"data":[{"id":%d,"call":"DK5WO","band":"80m","mode":"FT8",`+
					`"qso_date":"2026-10-30 00:53:00"}],"meta":{}}`, fakeRemoteID)
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

// withRecheckDelays shrinks the recheck waits for the duration of a test.
func withRecheckDelays(t *testing.T, delays ...time.Duration) {
	t.Helper()
	saved := uploadRecheckDelays
	uploadRecheckDelays = delays
	t.Cleanup(func() { uploadRecheckDelays = saved })
}

// withFastWriteDeadline sets a short write deadline so the fake's late
// response produces the timeout under test. Reads stay generous: the
// confirmation lookup must be able to answer.
func withFastWriteDeadline(t *testing.T) {
	t.Helper()
	t.Cleanup(wavelog.OverrideTimeouts(2*time.Second, 30*time.Millisecond, time.Second))
}

// uploadTestQSO inserts a QSO and returns the stored row.
func uploadTestQSO(t *testing.T, m *Model) *qso.QSO {
	t.Helper()
	q := &qso.QSO{Call: "DK5WO", QSODate: "20261030", TimeOn: "005300", Band: "80m", Mode: "FT8", Freq: 3.5735}
	id := insertTestQSO(t, m.App.DB, q)
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil || stored == nil {
		t.Fatalf("read QSO: %v", err)
	}
	return stored
}

func hasToast(m *Model, substr string) bool {
	for _, msg := range toastMessages(m.toasts) {
		if strings.Contains(msg, substr) {
			return true
		}
	}
	return false
}

// execRecheckCmd runs a command (unwrapping batches) and returns the recheck
// result it produced, executing the companion commands for their side effects.
func execRecheckCmd(t *testing.T, cmd tea.Cmd) (wlUploadRecheckMsg, bool) {
	t.Helper()
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			if sub == nil {
				continue
			}
			if r, ok := execRecheckCmd(t, sub); ok {
				return r, true
			}
		}
		return wlUploadRecheckMsg{}, false
	}
	if r, ok := msg.(wlUploadRecheckMsg); ok {
		return r, true
	}
	return wlUploadRecheckMsg{}, false
}

// ageToasts clears the toast pacing so the next notice is not suppressed. In
// production the first recheck waits 20 s, far beyond the pacing window; the
// tests shrink the delay, so the window has to be skipped explicitly.
func ageToasts(m *Model) {
	m.toasts.mu.Lock()
	m.toasts.lastPush = time.Time{}
	m.toasts.lastErr = time.Time{}
	m.toasts.lastMsg = ""
	m.toasts.mu.Unlock()
}

// runRecheckChain feeds recheck results back into the model until the sequence
// ends, and returns how many rechecks actually ran.
func runRecheckChain(t *testing.T, m *Model, cmd tea.Cmd) int {
	t.Helper()
	attempts := 0
	for cmd != nil && attempts < 10 {
		ageToasts(m)
		r, ok := execRecheckCmd(t, cmd)
		if !ok {
			break
		}
		attempts++
		_, cmd = m.Update(r)
	}
	return attempts
}

// TestUnknownUploadOutcomeIsRecheckedAndLinked reproduces the live DK5WO case:
// the write deadline expires, the server commits anyway, and the delayed
// recheck links the contact instead of leaving it reported as unsent.
func TestUnknownUploadOutcomeIsRecheckedAndLinked(t *testing.T) {
	withRecheckDelays(t, 10*time.Millisecond)
	withFastWriteDeadline(t)

	f := newFakeLateCommitWavelog(t)
	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL, wl.APIKey, wl.Enabled, wl.StationProfileID = f.srv.URL, "wl2_test", true, "1"
	m.inetOnline = true

	qs := uploadTestQSO(t, m)
	cmd := m.uploadQSOToWavelog(qs)
	if cmd == nil {
		t.Fatal("upload command is nil — the test model is not upload-enabled")
	}

	// The upload itself: the client gives up, the contact is not visible yet.
	res, ok := execCmd(cmd).(wlUploadResultMsg)
	if !ok {
		t.Fatal("upload did not produce an upload result")
	}
	if res.ok {
		t.Fatal("upload reported success although the server never answered")
	}
	if !isUploadStateUnknown(res.err) {
		t.Fatalf("err = %v, want an unknown-outcome error", res.err)
	}
	if row, _ := store.GetQSOByID(m.App.DB, qs.ID); row == nil || row.WavelogID != 0 {
		t.Fatal("row should still be unsent right after the unknown outcome")
	}

	// Dispatch through the model: the completion must queue the recheck.
	_, recheckCmd := m.Update(res)
	if recheckCmd == nil {
		t.Fatal("an unknown upload outcome queued no recheck — the contact stays unsent although it is on the server")
	}
	if hasToast(m, "failed") {
		t.Errorf("toasts = %v, want no failure claim for an unknown outcome", toastMessages(m.toasts))
	}

	// The server now shows the contact (it committed a moment ago).
	f.visible.Store(true)
	attempts := runRecheckChain(t, m, recheckCmd)
	if attempts != 1 {
		t.Errorf("rechecks ran = %d, want 1 — a confirmed contact needs no more", attempts)
	}

	row, err := store.GetQSOByID(m.App.DB, qs.ID)
	if err != nil || row == nil {
		t.Fatalf("read QSO after recheck: %v", err)
	}
	if row.WavelogID != fakeRemoteID {
		t.Errorf("WavelogID = %d, want %d — the delayed upload was never linked", row.WavelogID, fakeRemoteID)
	}
	if !hasToast(m, "confirmed on the server") {
		t.Errorf("toasts = %v, want a confirmation", toastMessages(m.toasts))
	}
	if got := f.posts.Load(); got != 1 {
		t.Errorf("%d writes reached the server, want 1 — the recheck must never re-send", got)
	}
}

// TestRecheckLeavesTheContactUnsentWhenTheServerNeverConfirms covers a contact
// that really did not land: the recheck runs out and the row stays unsent, with
// an honest message (no "failed", no claim of absence).
func TestRecheckLeavesTheContactUnsentWhenTheServerNeverConfirms(t *testing.T) {
	withRecheckDelays(t, 5*time.Millisecond, 5*time.Millisecond)
	withFastWriteDeadline(t)

	f := newFakeLateCommitWavelog(t)
	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL, wl.APIKey, wl.Enabled, wl.StationProfileID = f.srv.URL, "wl2_test", true, "1"
	m.inetOnline = true

	qs := uploadTestQSO(t, m)
	res, ok := execCmd(m.uploadQSOToWavelog(qs)).(wlUploadResultMsg)
	if !ok || !isUploadStateUnknown(res.err) {
		t.Fatalf("upload outcome = %+v, want an unknown outcome", res)
	}

	// The server answered every lookup and does not have the contact.
	f.visible.Store(false)
	_, recheckCmd := m.Update(res)
	if recheckCmd == nil {
		t.Fatal("an unknown upload outcome queued no recheck")
	}
	attempts := runRecheckChain(t, m, recheckCmd)
	if attempts != len(uploadRecheckDelays) {
		t.Errorf("rechecks ran = %d, want %d", attempts, len(uploadRecheckDelays))
	}

	row, _ := store.GetQSOByID(m.App.DB, qs.ID)
	if row == nil || row.WavelogID != 0 {
		t.Errorf("WavelogID = %v, want 0 — the contact never reached the server", row)
	}
	if !hasToast(m, "still unsent") {
		t.Errorf("toasts = %v, want an honest \"still unsent\" message", toastMessages(m.toasts))
	}
	if hasToast(m, "failed") {
		t.Errorf("toasts = %v, want no failure claim", toastMessages(m.toasts))
	}
	if got := f.posts.Load(); got != 1 {
		t.Errorf("%d writes reached the server, want 1 — a recheck must not re-send", got)
	}
}

// TestRecheckClaimsNothingWhenTheServerCannotBeAsked keeps the recheck honest:
// a lookup that fails is not proof that the contact is missing, so the message
// must not say the server does not have it.
func TestRecheckClaimsNothingWhenTheServerCannotBeAsked(t *testing.T) {
	withRecheckDelays(t, 5*time.Millisecond)
	withFastWriteDeadline(t)

	f := newFakeLateCommitWavelog(t)
	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL, wl.APIKey, wl.Enabled, wl.StationProfileID = f.srv.URL, "wl2_test", true, "1"
	m.inetOnline = true

	qs := uploadTestQSO(t, m)
	res, ok := execCmd(m.uploadQSOToWavelog(qs)).(wlUploadResultMsg)
	if !ok || !isUploadStateUnknown(res.err) {
		t.Fatalf("upload outcome = %+v, want an unknown outcome", res)
	}

	f.visible.Store(true)
	f.getFails.Store(true) // the confirmation lookup itself fails
	_, recheckCmd := m.Update(res)
	if recheckCmd == nil {
		t.Fatal("an unknown upload outcome queued no recheck")
	}
	runRecheckChain(t, m, recheckCmd)

	row, _ := store.GetQSOByID(m.App.DB, qs.ID)
	if row == nil || row.WavelogID != 0 {
		t.Errorf("WavelogID = %v, want 0 — an unanswered lookup must not link anything", row)
	}
	if !hasToast(m, "cannot ask the server") {
		t.Errorf("toasts = %v, want an \"cannot ask the server\" message", toastMessages(m.toasts))
	}
	if hasToast(m, "not seen on the server") {
		t.Errorf("toasts = %v, want no claim about absence from an unanswered lookup", toastMessages(m.toasts))
	}
}

// TestEditorUnknownUploadQueuesARecheck covers the logbook editor's own upload
// path: an unknown outcome must carry the contact into the global recheck
// instead of being reported as a failed upload.
func TestEditorUnknownUploadQueuesARecheck(t *testing.T) {
	withRecheckDelays(t, 5*time.Millisecond)
	withFastWriteDeadline(t)

	f := newFakeLateCommitWavelog(t)
	m := newLifecycleTestModel(t)
	wl := m.App.Logbook.Wavelog
	wl.URL, wl.APIKey, wl.Enabled, wl.StationProfileID = f.srv.URL, "wl2_test", true, "1"
	m.initLogbookEditor()
	le := m.ui.logbookEditor

	qs := uploadTestQSO(t, m)
	em := execCmd(le.uploadIndividual([]qso.QSO{*qs})).(editorMsg)
	if em.wlFailCount != 0 {
		t.Errorf("wlFailCount = %d, want 0 — the server never rejected the contact", em.wlFailCount)
	}
	if em.wlUnconfirmed != 1 {
		t.Errorf("wlUnconfirmed = %d, want 1", em.wlUnconfirmed)
	}
	if len(em.wlRecheck) != 1 || em.wlRecheck[0].qID != qs.ID {
		t.Fatalf("wlRecheck = %+v, want the unconfirmed contact", em.wlRecheck)
	}

	// The server publishes the contact; the global completion must queue the
	// recheck that links it.
	f.visible.Store(true)
	cmd := m.handleEditorUploadCompletion(em)
	if cmd == nil {
		t.Fatal("the editor completion queued no recheck for an unknown upload outcome")
	}
	if n := runRecheckChain(t, m, cmd); n != 1 {
		t.Errorf("rechecks ran = %d, want 1", n)
	}
	row, err := store.GetQSOByID(m.App.DB, qs.ID)
	if err != nil || row == nil {
		t.Fatalf("read QSO after recheck: %v", err)
	}
	if row.WavelogID != fakeRemoteID {
		t.Errorf("WavelogID = %d, want %d", row.WavelogID, fakeRemoteID)
	}
}

// TestRecheckSkipsAContactLinkedMeanwhile guards against double work: if the
// row already has a remote id, the recheck must stay silent.
func TestRecheckSkipsAContactLinkedMeanwhile(t *testing.T) {
	withRecheckDelays(t, 5*time.Millisecond)

	f := newFakeLateCommitWavelog(t)
	m := newLifecycleTestModel(t)
	qs := uploadTestQSO(t, m)
	f.visible.Store(true)

	// Linked by another path before the recheck ran.
	if err := store.SetWavelogID(m.App.DB, qs.ID, 777); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}

	cmd := m.queueUploadRecheck(m.App.DB, f.srv.URL, "wl2_test", "1", m.App.LogbookName,
		[]wlRecheckTarget{{qID: qs.ID, call: qs.Call, snap: *qs, rev: qs.WavelogDirtyRev}}, 0, nil)
	if cmd == nil {
		t.Fatal("recheck command is nil")
	}
	msg := execCmd(cmd).(wlUploadRecheckMsg)
	if len(msg.linked) != 0 || len(msg.left) != 0 {
		t.Errorf("linked=%d left=%d, want 0/0 — the row is already linked", len(msg.linked), len(msg.left))
	}
	row, _ := store.GetQSOByID(m.App.DB, qs.ID)
	if row == nil || row.WavelogID != 777 {
		t.Errorf("WavelogID = %v, want 777 (unchanged)", row)
	}
	if len(toastMessages(m.toasts)) != 0 {
		t.Errorf("toasts = %v, want none", toastMessages(m.toasts))
	}
}
