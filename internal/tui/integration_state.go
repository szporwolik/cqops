package tui

import (
	"sync"
	"time"
)

// rigState holds polled data, connection state, and the rig backend client.
type rigState struct {
	connected bool
	freq      float64 // display frequency (VFO B/TX when split, VFO A otherwise)
	freqRx    float64 // non-display RX frequency (VFO A when split, 0 otherwise)
	wasSplit  bool    // true if split was active on previous poll
	blink     bool
	skipTicks int
	slowTick  int // counter for slow-poll cycle (power, preamp, etc.)
	polling   bool
	client    RigClient // rig backend client (nil when disabled or not configured)
	modes     []string  // mode table from rig backend (indexed)
	name      string    // rig model name from rig backend (e.g. "FT-DX10")
}

// rotorState holds polled rotor data and the rotor backend client.
type rotorState struct {
	connected bool
	azimuth   float64
	elevation float64
	targetAz  float64     // commanded azimuth (0 when not moving)
	targetEl  float64     // commanded elevation (0 when not moving)
	name      string      // rotor model name (e.g. "YAESU G-800DXA")
	client    RotorClient // rotor backend client (nil when disabled)
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
