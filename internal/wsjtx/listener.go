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
	server   *wsjtx.Server
	active   bool
	Events   chan Event
	stop     chan struct{}
	wg       sync.WaitGroup
	OnADIF   func(string)
	OnStatus func(string, string, uint64, string, string, string)
}

func NewListener() *Listener {
	return &Listener{
		Events: make(chan Event, 2048),
	}
}

func (l *Listener) Start(host string, port int) error {
	if l.active {
		return nil
	}

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
	// outside (the library exposes no Shutdown/Close method).  It will be
	// cleaned up when the process exits.
	go func() {
		l.server.ListenToWsjtx(msgCh, errCh)
	}()

	applog.Info("WSJT-X listener started", "host", host, "port", port)

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.eventLoop(msgCh, errCh)
	}()
	return nil
}

func (l *Listener) Stop() {
	if !l.active {
		return
	}
	close(l.stop)
	l.wg.Wait()
	l.active = false
	l.server = nil
	applog.Info("WSJT-X listener stopped")
}

func (l *Listener) IsActive() bool {
	return l.active
}

func (l *Listener) eventLoop(msgCh chan interface{}, errCh chan error) {
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
				applog.Debug("WSJT-X: heartbeat", "id", m.Id, "version", m.Version)
			case wsjtx.StatusMessage:
				if l.OnStatus != nil {
					go l.OnStatus(m.DxCall, m.DxGrid, m.DialFrequency, m.Mode, m.SubMode, m.Report)
				}
			case wsjtx.DecodeMessage:
				if l.OnStatus != nil {
					go l.OnStatus("", "", 0, "", "", "")
				}
			case wsjtx.LoggedAdifMessage:
				applog.InfoDetail("WSJT-X: logged ADIF", m.Adif)
				if l.OnADIF != nil {
					go l.OnADIF(m.Adif)
				} else {
					applog.Warn("WSJT-X: OnADIF callback is nil, ADIF not auto-logged")
				}
			case wsjtx.QsoLoggedMessage:
				applog.InfoDetail("WSJT-X: QSO logged",
					fmt.Sprintf("dx=%s dxGrid=%s freq=%d mode=%s", m.DxCall, m.DxGrid, m.TxFrequency, m.Mode))
			case wsjtx.CloseMessage:
				applog.Info("WSJT-X: close")
			default:
				applog.Debug("WSJT-X: unknown msg", "type", fmt.Sprintf("%T", msg))
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
