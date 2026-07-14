// Package dashboard provides a read-model for the CQOps Live browser dashboard.
// It holds a thread-safe Snapshot updated by the TUI and served to HTTP clients
// via Server-Sent Events and REST endpoints.
//
// The dashboard is a passive observer — it never queries the database or calls
// into the TUI directly. All data flows from the TUI into the State, and from
// the State out to HTTP handlers.
package dashboard

import "time"

// =============================================================================
// Event model
// =============================================================================

// EventType identifies the kind of SSE event.
type EventType string

const (
	EventSnapshot   EventType = "snapshot"
	EventActiveQSO  EventType = "active_qso"
	EventQSOLogged  EventType = "qso_logged"
	EventRecentQSOs EventType = "recent_qsos"
	EventStats      EventType = "stats"
	EventStation    EventType = "station"
	EventOperator   EventType = "operator"
	EventLogbook    EventType = "logbook"
	EventRig        EventType = "rig"
	EventWSJTX      EventType = "wsjtx"
	EventSolar      EventType = "solar"
	EventDXC        EventType = "dxc"
	EventPSK        EventType = "psk"
	EventPartner    EventType = "partner"
	EventToday      EventType = "today"
	EventAPRS       EventType = "aprs"
	EventDisplay    EventType = "display"
	EventHeartbeat  EventType = "heartbeat"
)

// Event is a single SSE event with a monotonic ID and typed payload.
type Event struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}
