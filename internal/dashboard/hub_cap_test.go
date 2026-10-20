package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHubSubscriberCapBoundsStreams pins the SSE capacity limit: the hub must
// accept up to MaxSubscribers streams and refuse the next one instead of
// allocating another channel and goroutine.
func TestHubSubscriberCapBoundsStreams(t *testing.T) {
	h := NewHub()

	chans := make([]chan Event, 0, MaxSubscribers)
	for i := 0; i < MaxSubscribers; i++ {
		ch := h.Subscribe()
		if ch == nil {
			t.Fatalf("Subscribe returned nil at %d, want a stream", i)
		}
		chans = append(chans, ch)
	}
	if got := h.Clients(); got != MaxSubscribers {
		t.Fatalf("Clients() = %d, want %d", got, MaxSubscribers)
	}

	if ch := h.Subscribe(); ch != nil {
		t.Fatalf("Subscribe past the cap returned a stream, want nil")
	}
	if got := h.Clients(); got != MaxSubscribers {
		t.Fatalf("rejected Subscribe changed the count: %d, want %d", got, MaxSubscribers)
	}

	// Freeing one slot must let the next client in again.
	h.Unsubscribe(chans[0])
	if ch := h.Subscribe(); ch == nil {
		t.Fatal("Subscribe after freeing a slot returned nil")
	}

	// Publishing still reaches the remaining subscribers without blocking.
	for _, ch := range chans[1:] {
		select {
		case <-ch:
		default:
		}
	}
}

// TestHandleEventsRejectsAtCapacity exercises the HTTP side of the cap: the
// SSE handler must answer 503 *before* it commits a 200 and an initial
// snapshot, otherwise the browser would hold a "healthy" stream that simply
// ends and reconnect in a loop.
func TestHandleEventsRejectsAtCapacity(t *testing.T) {
	h := NewHub()
	state := NewState(h)

	chans := make([]chan Event, 0, MaxSubscribers)
	for i := 0; i < MaxSubscribers; i++ {
		chans = append(chans, h.Subscribe())
	}
	defer func() {
		for _, ch := range chans {
			h.Unsubscribe(ch)
		}
	}()

	handler := handleEvents(t.Context(), state, h)
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/api/events", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}
