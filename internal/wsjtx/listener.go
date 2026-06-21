package wsjtx

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"unsafe"

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
	OnStatus   func(string, string, uint64, string, string, string, string, bool) // call, grid, freqHz, mode, submode, report, txMessage, transmitting
}

func NewListener() *Listener {
	return &Listener{
		Events: make(chan Event, 2048),
	}
}

// Start creates a new server and begins listening for WSJT-X UDP messages.
// It is safe to call multiple times — a previous listener (if any) is stopped
// first (including closing the UDP socket) so that config changes (host/port)
// take effect.
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

	// The ListenToWsjtx goroutine blocks on UDP read; it exits when the
	// socket is closed (via reflection in stopLocked), receiving net.ErrClosed.
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
// It also shuts down the underlying UDP socket via the library's Shutdown
// method, allowing the port to be reused on the next Start.
// It is safe to call multiple times (idempotent).
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
	// Close the underlying UDP socket so the port can be reused.
	// wsjtx-go v4.2.1 does not expose Shutdown; we access the conn
	// via unsafe pointer to avoid the unexported-field reflect panic.
	if l.server != nil {
		rv := reflect.ValueOf(l.server).Elem()
		if connField := rv.FieldByName("conn"); connField.IsValid() && !connField.IsNil() {
			// conn is *net.UDPConn, so its address is **net.UDPConn.
			connPtr := (**net.UDPConn)(unsafe.Pointer(connField.UnsafeAddr()))
			(*connPtr).Close()
		}
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
					onStatus(m.DxCall, m.DxGrid, m.DialFrequency, m.Mode, m.SubMode, m.Report, m.TxMessage, m.Transmitting)
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
				// Suppress "use of closed network connection" — this is
				// normal during listener shutdown/cycling.
				if strings.Contains(e.Error(), "use of closed network connection") {
					continue
				}
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
func (l *Listener) snapshotOnStatus(gen uint64) func(string, string, uint64, string, string, string, string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.active || l.generation != gen {
		return nil
	}
	return l.OnStatus
}
