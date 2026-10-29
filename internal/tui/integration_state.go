package tui

import (
	"database/sql"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/store"
)

// rigState holds polled data, connection state, and the rig backend client.
type rigState struct {
	connected     bool
	freq          float64 // display frequency (VFO B/TX when split, VFO A otherwise)
	freqRx        float64 // non-display RX frequency (VFO A when split, 0 otherwise)
	wasSplit      bool    // true if split was active on previous poll
	blink         bool
	skipTicks     int
	slowTick      int // counter for slow-poll cycle (power, preamp, etc.)
	polling       bool
	pollStarted   int       // tickCount when current poll was dispatched
	pollCount     int       // total polls dispatched (for heartbeat logging)
	client        RigClient // rig backend client (nil when disabled or not configured)
	modes         []string  // mode table from rig backend (indexed)
	name          string    // rig model name from rig backend (e.g. "FT-DX10")
	vfoWarned     bool      // suppress repeated VFO-mode toasts on reconnect loops
	backendWarned bool      // suppress repeated "backend not configured" debug logs

	// Connection attempt tracking for summary-on-reconnect.
	connectAttempts int       // consecutive failed poll attempts since last disconnect
	connectStart    time.Time // first failed attempt timestamp in the current cycle
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
	online     bool
	tx         bool   // true when WSJT-X is transmitting (from StatusMessage)
	txMsg      string // last TX message from WSJT-X (e.g. "CQ SP9XXX JO90")
	lastSeen   time.Time
	status     string // current mode/submode from WSJT-X
	lastDxCall string // last DX call from WSJT-X status; empty when calling CQ
}

// adifQueue holds the WSJT-X ADIF queue with its mutex.
// Written by UDP callbacks from a background goroutine.
// Read by the update loop under lock.
type adifQueue struct {
	mu     sync.Mutex
	adifs  []string
	status statusPending
}

// maxPendingADIFs bounds the WSJT-X pending-ADIF queue. The queue is
// drained on every tick, so it only fills when the database stays busy for
// a long time (e.g. a large import); beyond the cap the oldest record is
// dropped rather than growing memory without limit.
const maxPendingADIFs = 64

// dxccBackfillState tracks the missing-DXCC backfill, which fills rows that
// escaped enrichment (legacy imports, ADIF loads, Wavelog downloads from before
// Big CTY was available) from the callsign prefix.
//
// The pass is a write transaction over a batch of contacts, so it runs off the
// update loop: at most one pass at a time, against the database it started on.
// db and logbook are what the finished pass reports back, so a result that
// outlives a logbook switch cannot repaint the new logbook's caches.
type dxccBackfillState struct {
	inFlight    bool
	fullPending bool // a fresh CTY table was installed: drain the whole log next
	db          *sql.DB
	logbook     string
}

// httpState holds the built-in HTTP server connection status.
type httpState struct {
	online      bool
	err         error
	client      *dashboard.Server
	restart     bool      // set when config changes to trigger a restart
	lastAttempt time.Time // last time we tried to start (for backoff)

	// Dashboard stats are computed off the update loop: statsDue marks that the
	// numbers are stale, statsInFlight that a computation is running (at most
	// one at a time). pushDashboardState only sets statsDue, so a caller that
	// cannot run a command — GPS, config save, logbook switch — cannot drop a
	// pending request; the next tick starts it.
	statsDue      bool
	statsInFlight bool
	// statsTotals caches the expensive half of the statistics (the DISTINCT
	// counts over the event window, ≈300 ms at 100k contacts). It is reused
	// while the window, the database and the data revision are unchanged, so the
	// minute refresh only counts the sliding rate windows.
	statsTotals statsTotalsCache
}

// statsTotalsCache holds the last computed window totals together with the
// exact inputs they describe. Any difference means the numbers belong to a
// different question and must be recomputed.
type statsTotalsCache struct {
	valid     bool
	db        *sql.DB
	logbook   string
	startDate string // event window start (today, or the configured event start)
	revision  uint64 // Model.qsoRev — bumps on save/edit/delete/import/enrichment
	totals    store.DashboardTotals
}
