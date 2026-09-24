package wsjtx

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	wsjtx "github.com/k0swe/wsjtx-go/v4"
	"github.com/szporwolik/cqops/internal/applog"
)

type Listener struct {
	mu         sync.Mutex
	server     *wsjtx.Server
	active     bool
	generation uint64 // incremented on each Start; used to reject stale callbacks
	stop       chan struct{}
	msgCh      chan interface{} // reader→loop message channel (detached on stop)
	errCh      chan error       // reader→loop error channel (detached on stop)
	loopWG     sync.WaitGroup   // event-processing goroutine
	readerWG   sync.WaitGroup   // UDP reader goroutine (exits when the socket closes)
	OnADIF     func(string)
	OnStatus   func(string, string, uint64, string, string, string, string, bool) // call, grid, freqHz, mode, submode, report, txMessage, transmitting
}

func NewListener() *Listener {
	return &Listener{}
}

// Start creates a new server and begins listening for WSJT-X UDP messages.
// It is safe to call multiple times — a previous listener (if any) is stopped
// first (including closing the UDP socket) so that config changes (host/port)
// take effect.
func (l *Listener) Start(host string, port int) error {
	// Stop any previous listener. Detach under the mutex, but tear down
	// outside it — waiting while holding l.mu would deadlock against the
	// event loop's callback snapshots (see Stop).
	l.mu.Lock()
	oldSrv, oldStop, oldMsgCh, oldErrCh := l.stopLocked()
	l.mu.Unlock()
	l.finishStop(oldSrv, oldStop, oldMsgCh, oldErrCh)

	l.mu.Lock()
	defer l.mu.Unlock()

	// Bump generation so any callbacks still in flight from a previous
	// listener will be rejected when they call isCurrentLocked.
	l.generation++
	gen := l.generation

	ip := net.ParseIP(host)
	if ip == nil {
		ip = net.ParseIP("127.0.0.1")
	}

	newSrv, err := wsjtx.MakeServerGiven(ip, uint(port))
	if err != nil {
		applog.Error("WSJT-X: server create failed", "host", host, "port", port, "error", err.Error())
		return fmt.Errorf("server: %w", err)
	}

	l.server = &newSrv
	l.active = true
	stop := make(chan struct{})
	l.stop = stop

	msgCh := make(chan interface{}, 128)
	errCh := make(chan error, 16)
	l.msgCh = msgCh
	l.errCh = errCh

	// The UDP reader blocks on the socket; it exits (closing both channels)
	// when the socket is closed during shutdown.
	l.readerWG.Add(1)
	go func() {
		defer l.readerWG.Done()
		newSrv.ListenToWsjtx(msgCh, errCh)
	}()

	applog.Info("WSJT-X listener started", "host", host, "port", port)

	l.loopWG.Add(1)
	go func() {
		defer l.loopWG.Done()
		l.eventLoop(gen, stop, msgCh, errCh)
	}()
	return nil
}

// Stop signals the event-processing goroutine to exit and waits for it.
// It also shuts down the underlying UDP socket via the library's unexported
// conn field, allowing the port to be reused on the next Start.
// It is safe to call multiple times (idempotent).
//
// The mutex is released before the socket is closed and before waiting for
// the goroutines: the event loop takes l.mu in its callback snapshots, so
// waiting while holding the mutex would deadlock shutdown.
func (l *Listener) Stop() {
	l.mu.Lock()
	srv, stop, msgCh, errCh := l.stopLocked()
	l.mu.Unlock()
	l.finishStop(srv, stop, msgCh, errCh)
}

// stopLocked marks the listener stopped, invalidates the generation, and
// detaches the server, stop channel, and reader channels. Caller must hold
// l.mu. The returned resources must be torn down by the caller AFTER
// releasing l.mu (see finishStop) — doing it under the mutex would deadlock
// the event loop.
func (l *Listener) stopLocked() (*wsjtx.Server, chan struct{}, chan interface{}, chan error) {
	if !l.active {
		return nil, nil, nil, nil
	}
	l.active = false
	l.generation++ // reject callbacks that have not snapshotted yet
	srv, stop, msgCh, errCh := l.server, l.stop, l.msgCh, l.errCh
	l.server = nil
	l.stop = nil
	l.msgCh = nil
	l.errCh = nil
	return srv, stop, msgCh, errCh
}

// finishStop tears down a detached listener without holding l.mu.
func (l *Listener) finishStop(srv *wsjtx.Server, stop chan struct{}, msgCh chan interface{}, errCh chan error) {
	if srv == nil && stop == nil {
		return
	}
	socketClosed := closeServerSocket(srv)
	if stop != nil {
		close(stop)
	}
	// The event loop always exits via the stop channel — safe to join.
	l.loopWG.Wait()
	if !socketClosed {
		// The reader only exits when its socket closes. If the unsafe close
		// could not reach the conn (library changed), the reader stays
		// blocked forever — joining it (or draining) would deadlock
		// shutdown, so skip in that case.
		applog.Info("WSJT-X listener stopped")
		return
	}
	// The UDP reader exits when its socket closes — but ListenToWsjtx uses
	// BLOCKING sends into msgCh/errCh, and closing the socket cannot
	// unblock a reader already waiting to send into a full channel. With
	// the event loop gone nothing would drain them, so keep draining both
	// channels until the producer exits; only then is it safe to join it.
	if msgCh != nil || errCh != nil {
		var drainWG sync.WaitGroup
		drainWG.Add(1)
		go func() {
			defer drainWG.Done()
			drainChannels(msgCh, errCh)
		}()
		l.readerWG.Wait()
		drainWG.Wait()
	} else {
		l.readerWG.Wait()
	}
	applog.Info("WSJT-X listener stopped")
}

// drainChannels discards messages from both channels until they are closed,
// so a producer blocked on a full channel can finish its send, observe the
// closed socket, and exit.
func drainChannels(msgCh chan interface{}, errCh chan error) {
	for {
		select {
		case _, ok := <-msgCh:
			if !ok {
				msgCh = nil
			}
		case _, ok := <-errCh:
			if !ok {
				errCh = nil
			}
		}
		if msgCh == nil && errCh == nil {
			return
		}
	}
}

// closeServerSocket closes the library's underlying UDP socket so the port
// can be reused. wsjtx-go v4 exposes no Shutdown method, so the unexported
// conn field is reached via unsafe. Reports whether the socket was actually
// closed; a recover() keeps shutdown working if the library changes (the
// socket is then simply not closed until the OS reclaims it).
func closeServerSocket(srv *wsjtx.Server) bool {
	if srv == nil {
		return false
	}
	closed := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				applog.Warn("WSJT-X: unsafe conn close panicked — library may have changed", "panic", r)
				applog.Debug("WSJT-X: socket close fallback — port may be held until OS timeout; sleeping 500ms to help release")
				time.Sleep(500 * time.Millisecond)
			}
		}()
		rv := reflect.ValueOf(srv).Elem()
		if connField := rv.FieldByName("conn"); connField.IsValid() && connField.Kind() == reflect.Ptr && !connField.IsNil() {
			connPtr := (**net.UDPConn)(unsafe.Pointer(connField.UnsafeAddr()))
			(*connPtr).Close()
			closed = true
		}
	}()
	return closed
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
// the listener is active. The stop channel is the per-run channel captured
// at Start; the loop never reads the Listener.stop field (it is swapped
// during shutdown).
//
// Callbacks run inline (not in separate goroutines) because they are trivial
// field assignments under a lock — spawning a goroutine per message would
// create massive scheduler overhead at typical FT8 message rates (50+/cycle).
func (l *Listener) eventLoop(gen uint64, stop chan struct{}, msgCh chan interface{}, errCh chan error) {
	for {
		select {
		case <-stop:
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
				applog.Info("WSJT-X: logged ADIF", "len", len(m.Adif))
				applog.Debug("WSJT-X: logged ADIF raw", "adif", m.Adif)
				if onADIF := l.snapshotOnADIF(gen); onADIF != nil {
					onADIF(m.Adif)
				}
			case wsjtx.QsoLoggedMessage:
				applog.Info("WSJT-X: QSO logged",
					"dx", m.DxCall, "dxGrid", m.DxGrid, "freq", m.TxFrequency, "mode", m.Mode,
				)
				applog.Debug("WSJT-X: QSO logged raw fields",
					"dxCall", m.DxCall, "dxGrid", m.DxGrid,
					"txFreq", m.TxFrequency, "mode", m.Mode,
				)
				// Also save this QSO. WSJT-X may send QsoLoggedMessage
				// without a separate LoggedAdifMessage on some versions.
				// Construct a minimal ADIF record from the known fields.
				if onADIF := l.snapshotOnADIF(gen); onADIF != nil {
					adif := buildADIFFromQsoLogged(m)
					if adif != "" {
						applog.Debug("WSJT-X: built ADIF from QsoLogged", "adif", adif)
						onADIF(adif)
					}
				}
			case wsjtx.CloseMessage:
				applog.Info("WSJT-X: close")
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
				// Log parse leftovers at DEBUG so we can analyze them.
				if strings.Contains(e.Error(), "bytes left over") {
					applog.Debug("WSJT-X: parse leftover bytes", "error", e.Error())
				} else {
					applog.Error("WSJT-X: error", "error", e.Error())
				}
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

// buildADIFFromQsoLogged constructs a minimal ADIF record from a
// QsoLoggedMessage. This ensures QSOs are saved even when WSJT-X
// sends only the field-based message (without a separate LoggedAdifMessage).
func buildADIFFromQsoLogged(m wsjtx.QsoLoggedMessage) string {
	if m.DxCall == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n<adif_ver:5>3.1.0\n<programid:6>WSJT-X\n<EOH>\n")
	writeADIF(&b, "call", m.DxCall)
	writeADIF(&b, "gridsquare", m.DxGrid)
	if !m.DateTimeOn.IsZero() {
		writeADIF(&b, "qso_date", m.DateTimeOn.UTC().Format("20060102"))
		writeADIF(&b, "time_on", m.DateTimeOn.UTC().Format("150405"))
	}
	if !m.DateTimeOff.IsZero() {
		writeADIF(&b, "qso_date_off", m.DateTimeOff.UTC().Format("20060102"))
		writeADIF(&b, "time_off", m.DateTimeOff.UTC().Format("150405"))
	}
	writeADIF(&b, "mode", m.Mode)
	if m.TxFrequency > 0 {
		writeADIF(&b, "freq", fmt.Sprintf("%.6f", float64(m.TxFrequency)/1e6))
	}
	writeADIF(&b, "rst_sent", m.ReportSent)
	writeADIF(&b, "rst_rcvd", m.ReportReceived)
	writeADIF(&b, "tx_pwr", m.TxPower)
	writeADIF(&b, "comment", m.Comments)
	writeADIF(&b, "name", m.Name)
	writeADIF(&b, "station_callsign", m.MyCall)
	writeADIF(&b, "my_gridsquare", m.MyGrid)
	writeADIF(&b, "operator", m.OperatorCall)
	writeADIF(&b, "srx", m.ExchangeReceived)
	writeADIF(&b, "stx", m.ExchangeSent)
	b.WriteString("<EOR>")
	return b.String()
}

func writeADIF(b *strings.Builder, field, value string) {
	if value == "" {
		return
	}
	b.WriteString("<")
	b.WriteString(field)
	b.WriteString(":")
	b.WriteString(strconv.Itoa(len(value)))
	b.WriteString(">")
	b.WriteString(value)
	b.WriteString(" ")
}
