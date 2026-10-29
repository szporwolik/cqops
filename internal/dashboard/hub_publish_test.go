package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// Publishing with no subscriber
//
// Publish used to serialize every event before looking at the subscriber list,
// so an enabled dashboard server with no browser open paid JSON encoding on
// every push: 3.6 ms and 0.82 MiB of allocations for a 5,000-contact event.
// The event is now dropped before encoding. What must not change: the state the
// next browser reads, and the event IDs.
// =============================================================================

// bigPayload builds a payload comparable to a full Today push.
func bigPayload(n int) []QSOView {
	views := make([]QSOView, 0, n)
	for i := 0; i < n; i++ {
		views = append(views, QSOView{
			ID:        fmt.Sprintf("%d", i),
			TimeUTC:   time.Date(2026, 10, 29, 12, 0, i%60, 0, time.UTC),
			Call:      fmt.Sprintf("SP%dABC", i),
			Band:      "20m",
			Mode:      "FT8",
			Submode:   "FT8",
			Frequency: "14.074 MHz",
			RSTSent:   "-10",
			RSTRcvd:   "-12",
			Grid:      "JO90WE",
			MyGrid:    "KO00AA",
			Country:   "Poland",
			Operator:  "SP9MOA",
		})
	}
	return views
}

// marshalCounter is a payload that counts its own JSON encoding, so a test can
// assert that Publish really skipped the serializer rather than merely getting
// faster.
type marshalCounter struct {
	n     *int32
	label string
}

func (c marshalCounter) MarshalJSON() ([]byte, error) {
	atomic.AddInt32(c.n, 1)
	return []byte(`"` + c.label + `"`), nil
}

// TestPublishWithoutSubscribersSkipsSerialization is the point of the change:
// with no subscriber the payload must not be encoded at all.
func TestPublishWithoutSubscribersSkipsSerialization(t *testing.T) {
	hub := NewHub()
	var encoded int32

	for i := 0; i < 10; i++ {
		hub.Publish(EventToday, marshalCounter{n: &encoded, label: "today"})
	}
	if encoded != 0 {
		t.Errorf("the payload was serialized %d times with no subscriber — that is the ~3.6 ms and 0.82 MiB per push this removes", encoded)
	}

	// With a subscriber the event has to be encoded, exactly once, and shared.
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)
	hub.Publish(EventToday, marshalCounter{n: &encoded, label: "today"})
	if encoded != 1 {
		t.Errorf("encoded %d times for one subscriber, want 1", encoded)
	}
	select {
	case ev := <-ch:
		if string(ev.sseData) == "" {
			t.Error("the delivered event carries no serialized payload")
		}
	default:
		t.Fatal("subscriber received nothing")
	}
}

// TestPublishWithoutSubscribersKeepsTheState: publishing to nobody must still
// leave the snapshot the next browser reads complete.
func TestPublishWithoutSubscribersKeepsTheState(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)
	mux := NewMux(context.Background(), state, hub)

	state.SetStation(StationInfo{Callsign: "SP9MOA", Locator: "JO90"})
	state.SetToday(bigPayload(50))
	state.SetStats(Stats{QSOsToday: 50, DXCC: 7, Rate5m: 3})
	state.SetOperator(OperatorInfo{Callsign: "SP9MOA", Name: "Szymon"})

	if hub.Clients() != 0 {
		t.Fatalf("%d subscribers before anyone connected", hub.Clients())
	}

	// The browser that opens now must see everything above.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/snapshot", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/snapshot status = %d", rec.Code)
	}
	var snap Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("snapshot is not valid JSON: %v", err)
	}
	if snap.Station.Callsign != "SP9MOA" {
		t.Errorf("station = %q, want SP9MOA", snap.Station.Callsign)
	}
	if len(snap.Today) != 50 {
		t.Errorf("today holds %d contacts, want 50", len(snap.Today))
	}
	if snap.Stats.QSOsToday != 50 || snap.Stats.DXCC != 7 {
		t.Errorf("stats = %+v, want 50 contacts / 7 entities", snap.Stats)
	}
	if snap.Operator.Callsign != "SP9MOA" {
		t.Errorf("operator = %q, want SP9MOA", snap.Operator.Callsign)
	}
}

// TestHandleEventsSnapshotAfterSilentPublishes covers the same thing through the
// SSE stream: the first frame a browser receives is the current snapshot, even
// though every event published before it was dropped.
func TestHandleEventsSnapshotAfterSilentPublishes(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)
	mux := NewMux(context.Background(), state, hub)

	for i := 0; i < 5; i++ {
		state.SetToday(bigPayload(200))
	}
	if hub.Clients() != 0 {
		t.Fatalf("the test published to %d subscribers", hub.Clients())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(rec, req)
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, "event: snapshot") {
		t.Fatalf("no snapshot frame on connect: %.200s", body)
	}
	// The snapshot frame carries the contacts published while nobody listened.
	if strings.Count(body, `"call":"SP`) < 200 {
		t.Errorf("the streamed snapshot is missing the contacts published with no subscriber (%.200s)", body)
	}
}

// TestPublishWithoutSubscribersStillCountsEvents pins the ID semantics: a
// publish consumes an ID whether or not anyone is listening, so the sequence a
// later subscriber sees never restarts or repeats.
func TestPublishWithoutSubscribersStillCountsEvents(t *testing.T) {
	hub := NewHub()

	const silent = 5 // publishes nobody hears
	for i := 0; i < silent; i++ {
		hub.Publish(EventRig, RigInfo{Connected: true})
	}

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	hub.Publish(EventRig, RigInfo{Connected: false})
	select {
	case ev := <-ch:
		if ev.ID != silent+1 {
			t.Errorf("first event a subscriber sees has ID %d, want %d (IDs must keep counting while nobody listens)",
				ev.ID, silent+1)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber received nothing")
	}

	// And the IDs keep increasing for the events that follow.
	hub.Publish(EventRig, RigInfo{Connected: true})
	select {
	case ev := <-ch:
		if ev.ID != silent+2 {
			t.Errorf("second event ID = %d, want %d", ev.ID, silent+2)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber received only one event")
	}
}

// TestPublishConcurrentSubscribeUnsubscribe exercises the lock and the cached
// subscriber slice from many goroutines at once: this test is meaningful under
// -race, where it fails on any unsynchronised access to the subscriber list or
// the cached slice.
func TestPublishConcurrentSubscribeUnsubscribe(t *testing.T) {
	hub := NewHub()
	payload := bigPayload(10)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Publishers: some with subscribers, some without, mixed.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					hub.Publish(EventToday, payload)
				}
			}
		}(i)
	}

	// Subscribers that come and go, draining what they receive.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				ch := hub.Subscribe()
				if ch == nil {
					continue
				}
				// Drain a few events, then leave.
				deadline := time.After(2 * time.Millisecond)
			drain:
				for {
					select {
					case <-ch:
					case <-deadline:
						break drain
					}
				}
				hub.Unsubscribe(ch)
			}
		}()
	}

	time.Sleep(150 * time.Millisecond)
	close(stop)
	wg.Wait()

	if got := hub.Clients(); got != 0 {
		t.Errorf("%d subscribers left behind", got)
	}
}

// BenchmarkHubPublish_NoSubscribers measures what the dashboard pays on every
// push with no browser open, and what it pays for the same event with one.
func BenchmarkHubPublish_NoSubscribers(b *testing.B) {
	for _, tc := range []struct {
		name string
		n    int
	}{
		{"small_event", 10},
		{"5000_qso_event", 5000},
	} {
		b.Run(tc.name, func(b *testing.B) {
			payload := bigPayload(tc.n)
			hub := NewHub()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hub.Publish(EventToday, payload)
			}
		})
	}
}

// BenchmarkHubPublish_OneSubscriber is the reference: the same events, encoded
// once and delivered, which is the cost that remains when a browser is open.
func BenchmarkHubPublish_OneSubscriber(b *testing.B) {
	for _, tc := range []struct {
		name string
		n    int
	}{
		{"small_event", 10},
		{"5000_qso_event", 5000},
	} {
		b.Run(tc.name, func(b *testing.B) {
			payload := bigPayload(tc.n)
			hub := NewHub()
			ch := hub.Subscribe()
			defer hub.Unsubscribe(ch)
			// Keep the buffer from filling up so the benchmark measures the
			// encode path, not the drop path.
			go func() {
				for range ch {
				}
			}()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hub.Publish(EventToday, payload)
			}
			b.StopTimer()
			close(ch)
		})
	}
}
