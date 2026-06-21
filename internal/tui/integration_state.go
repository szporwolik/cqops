package tui

import (
	"sync"
	"time"
)

// rigState holds flrig polled data, connection state, and the flrig client.
type rigState struct {
	connected bool
	freq      float64
	blink     bool
	skipTicks int
	polling   bool
	client    FlrigClient // flrig HTTP client (nil when disabled or not configured)
	modes     []string    // mode table from flrig (indexed)
	name      string      // rig model name from flrig (e.g. "FT-DX10")
}

// wsjtxState holds WSJT-X integration connection and status state.
type wsjtxState struct {
	online   bool
	tx       bool   // true when WSJT-X is transmitting (from StatusMessage)
	txMsg    string // last TX message from WSJT-X (e.g. "CQ SP9XXX JO90")
	lastSeen time.Time
	status   string // current mode/submode from WSJT-X
}

// adifQueue holds the WSJT-X ADIF queue with its mutex.
// Written by UDP callbacks from a background goroutine.
// Read by the update loop under lock.
type adifQueue struct {
	mu     sync.Mutex
	adifs  []string
	status statusPending
}
