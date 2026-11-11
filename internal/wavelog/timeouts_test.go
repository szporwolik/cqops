package wavelog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Deadlines and timeout typing.
//
// A read and a write get different deadlines, and a request that runs out of
// time is a distinct error type — it says the outcome is unknown, not that the
// call failed. Both matter: on 2026-10-29 a degraded Wavelog held a QSO insert
// past the 10 s deadline, committed it anyway, and CQOps reported "upload
// failed" for a contact that was already on the server.
// =============================================================================

// TestClientForMethodPicksTheDeadline pins which client a request gets.
func TestClientForMethodPicksTheDeadline(t *testing.T) {
	defer OverrideTimeouts(10*time.Second, 30*time.Second, 5*time.Second)()

	if got := httpClient.Timeout; got != 10*time.Second {
		t.Errorf("read deadline = %s, want 10s", got)
	}
	if got := writeClient.Timeout; got != 30*time.Second {
		t.Errorf("write deadline = %s, want 30s", got)
	}
	if got := radioClient.Timeout; got != 5*time.Second {
		t.Errorf("radio deadline = %s, want 5s", got)
	}

	writes := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range writes {
		if got := clientForMethod(method); got != writeClient {
			t.Errorf("%s uses the read client; a write may be committed after the client gives up", method)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		if got := clientForMethod(method); got != httpClient {
			t.Errorf("%s uses the write client; a lookup must not wait 30 s", method)
		}
	}
}

// TestWriteDeadlineCoversAMeasuredServer pins the production write deadline
// against the slowest write ever measured on a live deployment: 60.2 s per
// accepted write on qsl.sp9moa.pl (2026-10-30), while reads answered in
// 0.05–0.5 s. At 30 s every one of those writes was abandoned just before the
// answer arrived, so the contact stayed unsent locally although it was on the
// server. The margin must not be closed without a new measurement.
func TestWriteDeadlineCoversAMeasuredServer(t *testing.T) {
	const measuredSlowWrite = 60500 * time.Millisecond
	if WriteDeadline <= measuredSlowWrite {
		t.Errorf("WriteDeadline = %s, want more than the measured %s", WriteDeadline, measuredSlowWrite)
	}
	if got := writeClient.Timeout; got != WriteDeadline {
		t.Errorf("write client deadline = %s, want %s", got, WriteDeadline)
	}
}

// TestSlowWriteSucceedsOnTheWriteClient proves the deadline choice is real
// behaviour and not just a constant: a write that takes far longer than the
// read deadline still succeeds, because writes get their own client.
func TestSlowWriteSucceedsOnTheWriteClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":99},"meta":{}}`)
	}))
	defer srv.Close()
	defer OverrideTimeouts(40*time.Millisecond, 2*time.Second, 40*time.Millisecond)()

	id, dup, err := CreateQSO(srv.URL, "wl2_test", CreateQSOInput{StationProfileID: 1, Call: "DL1AA"})
	if err != nil {
		t.Fatalf("a slow write must succeed on the write deadline, got %v", err)
	}
	if dup || id != 99 {
		t.Errorf("id=%d dup=%v, want 99/false", id, dup)
	}
}

// TestOverrideTimeoutsRestores closes the seam itself: a leaked short deadline
// would make every later test in the package flaky.
func TestOverrideTimeoutsRestores(t *testing.T) {
	before := httpClient
	restore := OverrideTimeouts(time.Millisecond, time.Millisecond, time.Millisecond)
	if httpClient == before {
		t.Fatal("OverrideTimeouts did not replace the read client")
	}
	restore()
	if httpClient != before {
		t.Fatal("restore did not put the read client back")
	}
}

// TestCreateQSOTimeoutIsTyped drives the real client against a server that
// answers after the deadline: the error must be a TimeoutError, not a generic
// transport failure, because callers branch on exactly that.
func TestCreateQSOTimeoutIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":1},"meta":{}}`)
	}))
	defer srv.Close()
	defer OverrideTimeouts(40*time.Millisecond, 40*time.Millisecond, 40*time.Millisecond)()

	_, _, err := CreateQSO(srv.URL, "wl2_test", CreateQSOInput{StationProfileID: 1, Call: "DL1AA", Band: "80m", Mode: "FT8"})
	if err == nil {
		t.Fatal("a server that answers after the deadline must not look like a success")
	}
	if !IsTimeout(err) {
		t.Fatalf("error %v (%T) is not a TimeoutError — callers cannot tell an unknown outcome from a failure", err, err)
	}
	if got := err.Error(); got != "no answer from Wavelog within 40ms" {
		t.Errorf("message = %q, want the elapsed deadline", got)
	}
	if IsServerError(err) {
		t.Error("a timeout is not a server error")
	}
}

// TestServerErrorIsTyped: a 5xx is also an unknown outcome for a write, because
// a proxy can answer it for a request that reached the database.
func TestServerErrorIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"error":{"code":"internal_error","message":"upstream died"}}`)
	}))
	defer srv.Close()

	_, _, err := CreateQSO(srv.URL, "wl2_test", CreateQSOInput{StationProfileID: 1, Call: "DL1AA", Band: "80m", Mode: "FT8"})
	if err == nil {
		t.Fatal("a 502 must surface as an error")
	}
	if !IsServerError(err) {
		t.Fatalf("error %v (%T) is not typed as a server error", err, err)
	}
	if IsTimeout(err) {
		t.Error("a 5xx is not a timeout")
	}
	if got := err.Error(); !strings.Contains(got, "upstream died") {
		t.Errorf("message = %q, want the server's own text to survive the typing", got)
	}
}

// TestReadErrorsStayUntyped: a 404 must not be mistaken for an unknown write
// outcome, or a create against a wrong URL would be "reconciled" instead of
// reporting the bad address.
func TestReadErrorsStayUntyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := FindQSOID(srv.URL, "wl2_test", "1", "DL1AA", "2026-10-29", "230115", "80m", "FT8")
	if err == nil {
		t.Fatal("a 404 must surface as an error")
	}
	if IsTimeout(err) || IsServerError(err) {
		t.Errorf("a 404 is a plain failure, got %T", err)
	}
}
