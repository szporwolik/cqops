package dashboard

import (
	"sync"
	"time"
)

// Hub manages SSE subscribers and broadcasts typed events.
// Subscribers are buffered channels. Sends are non-blocking —
// slow clients may miss events and should recover via /api/snapshot.
type Hub struct {
	mu          sync.Mutex
	subscribers map[chan Event]struct{}
	nextID      int64
}

// NewHub creates a Hub with no subscribers.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[chan Event]struct{}),
	}
}

// Subscribe creates a new buffered channel (cap 16) and registers it.
// The caller is responsible for calling Unsubscribe when done.
func (h *Hub) Subscribe() chan Event {
	ch := make(chan Event, 16)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the hub.
// It does NOT close the channel — the subscriber owns the channel lifecycle.
// The subscriber's for-loop exits via context cancellation.
func (h *Hub) Unsubscribe(ch chan Event) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	h.mu.Unlock()
}

// Publish sends an event to all subscribers. Each event gets a
// monotonic ID. Sends are non-blocking — if a subscriber's buffer
// is full, the event is dropped for that subscriber.
func (h *Hub) Publish(typ EventType, payload any) {
	h.mu.Lock()
	h.nextID++
	ev := Event{
		ID:        h.nextID,
		Type:      string(typ),
		Timestamp: timeNow(),
		Payload:   payload,
	}
	// Snapshot subscribers under lock, then send outside.
	subs := make([]chan Event, 0, len(h.subscribers))
	for ch := range h.subscribers {
		subs = append(subs, ch)
	}
	h.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
			// Drop — subscriber buffer full.
		}
	}
}

// timeNow is a shim for testing.
var timeNow = func() time.Time { return time.Now() }
