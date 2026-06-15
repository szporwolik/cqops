package wsjtx

import (
	"fmt"
	"net"
	"sync"

	wsjtx "github.com/k0swe/wsjtx-go/v4"
	"github.com/szporwolik/cqops/internal/applog"
)

type Event struct {
	Msg     interface{}
	RawADIF string
}

type Listener struct {
	mu         sync.Mutex
	server     *wsjtx.Server
	active     bool
	generation uint64 // incremented on each Start; used to reject stale callbacks
	Events     chan Event
	stop       chan struct{}
	wg         sync.WaitGroup
	OnADIF     func(string)
	OnStatus   func(string, string, uint64, string, string, string)
}

func NewListener() *Listener {
	return &Listener{
		Events: make(chan Event, 2048),
	}
}

// Start creates a new server and begins listening for WSJT-X UDP messages.
// It is safe to call multiple times — a previous listener (if any) is stopped
// first so that config changes (host/port) take effect.
//
// Known limitation: the underlying wsjtx-go library does not expose a way to
// close the UDP socket. The goroutine that calls ListenToWsjtx will persist
// until the process exits. On restart, the old UDP goroutine leaks, but the
// event-processing goroutine (eventLoop) is properly cleaned up.
func (l *Listener) Start(host string, port int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure any previous listener is fully stopped before creating a new one.
	l.stopLocked()

	// Bump generation so any callbacks still in flight from a previous
	// listener will be rejected when they call isCurrentLocked.
	l.generation++
	gen := l.generation

	ip := net.ParseIP(host)
	if ip == nil {
		ip = net.ParseIP("127.0.0.1")
	}

	srv, err := wsjtx.MakeServerGiven(ip, uint(port))
	if err != nil {
		applog.Error("WSJT-X: server create failed", "host", host, "port", port, "error", err.Error())
		return fmt.Errorf("server: %w", err)
	}

	l.server = &srv
	l.active = true
	l.stop = make(chan struct{})

	msgCh := make(chan interface{}, 128)
	errCh := make(chan error, 16)

	// This goroutine blocks on UDP read and cannot be interrupted from
	// outside (the library exposes no Shutdown/Close for the socket).
	// It will be cleaned up when the process exits.
	go func() {
		l.server.ListenToWsjtx(msgCh, errCh)
	}()

	applog.Info("WSJT-X listener started", "host", host, "port", port)

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.eventLoop(gen, msgCh, errCh)
	}()
	return nil
}

// Stop signals the event-processing goroutine to exit and waits for it.
// It is safe to call multiple times (idempotent).
//
// Note: the low-level UDP-listen goroutine (ListenToWsjtx) cannot be
// stopped — the library provides no socket-close mechanism. It will
// exit when the process terminates.
func (l *Listener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopLocked()
}

// stopLocked performs the actual stop logic. Caller must hold l.mu.
func (l *Listener) stopLocked() {
	if !l.active {
		return
	}
	close(l.stop)
	l.wg.Wait()
	l.active = false
	l.server = nil
	applog.Info("WSJT-X listener stopped")
}

// IsActive returns true if the listener is currently active. Safe for concurrent use.
func (l *Listener) IsActive() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.active
}

// eventLoop reads WSJT-X messages from the UDP goroutine and dispatches
// callbacks. The gen parameter is the listener generation captured at Start
// time — callbacks are only invoked if the generation is still current and
// the listener is active.
//
// Callbacks run inline (not in separate goroutines) because they are trivial
// field assignments under a lock — spawning a goroutine per message would
// create massive scheduler overhead at typical FT8 message rates (50+/cycle).
func (l *Listener) eventLoop(gen uint64, msgCh chan interface{}, errCh chan error) {
	for {
		select {
		case <-l.stop:
			return
		case msg, ok := <-msgCh:
			if !ok || msg == nil {
				continue
			}
			switch m := msg.(type) {
			case wsjtx.HeartbeatMessage:
				// No callback needed — heartbeat is informational.
			case wsjtx.StatusMessage:
				if onStatus := l.snapshotOnStatus(gen); onStatus != nil {
					onStatus(m.DxCall, m.DxGrid, m.DialFrequency, m.Mode, m.SubMode, m.Report)
				}
			case wsjtx.DecodeMessage:
				// Decode messages carry no callsign/freq data — just mark activity.
				// No callback needed; the status message handles field updates.
			case wsjtx.LoggedAdifMessage:
				applog.InfoDetail("WSJT-X: logged ADIF", m.Adif)
				if onADIF := l.snapshotOnADIF(gen); onADIF != nil {
					onADIF(m.Adif)
				}
			case wsjtx.QsoLoggedMessage:
				applog.InfoDetail("WSJT-X: QSO logged",
					fmt.Sprintf("dx=%s dxGrid=%s freq=%d mode=%s", m.DxCall, m.DxGrid, m.TxFrequency, m.Mode))
			case wsjtx.CloseMessage:
				applog.Info("WSJT-X: close")
			}
			select {
			case l.Events <- Event{Msg: msg}:
			default:
			}
		case e, ok := <-errCh:
			if !ok {
				return
			}
			if e != nil {
				applog.Error("WSJT-X: error", "error", e.Error())
			}
		}
	}
}

// snapshotOnADIF returns the OnADIF callback if the listener is still active
// and the generation matches. Returns nil otherwise — stale callbacks are
// silently dropped. Must be called without holding l.mu.
func (l *Listener) snapshotOnADIF(gen uint64) func(string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.active || l.generation != gen {
		return nil
	}
	return l.OnADIF
}

// snapshotOnStatus returns the OnStatus callback under the same generation guard.
func (l *Listener) snapshotOnStatus(gen uint64) func(string, string, uint64, string, string, string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.active || l.generation != gen {
		return nil
	}
	return l.OnStatus
}
