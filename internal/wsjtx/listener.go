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

type Event struct {
	Msg     interface{}
	RawADIF string
}

type Listener struct {
	mu         sync.Mutex
	server     *wsjtx.Server
	active     bool
	generation uint64     // incremented on each Start; used to reject stale callbacks
	Events     chan Event // exported for external consumers; not drained by CQOps internally
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
	// If the library's internal struct changes, the recover prevents
	// a crash — the socket just won't be closed immediately.
	if l.server != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					applog.Warn("WSJT-X: unsafe conn close panicked — library may have changed", "panic", r)
					applog.Debug("WSJT-X: socket close fallback — port may be held until OS timeout; sleeping 500ms to help release")
					time.Sleep(500 * time.Millisecond)
				}
			}()
			rv := reflect.ValueOf(l.server).Elem()
			if connField := rv.FieldByName("conn"); connField.IsValid() && connField.Kind() == reflect.Ptr && !connField.IsNil() {
				connPtr := (**net.UDPConn)(unsafe.Pointer(connField.UnsafeAddr()))
				(*connPtr).Close()
			}
		}()
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
