// Package dxc provides a DX Cluster telnet client that connects to
// dxspider.co.uk:7300 (or a user-configured host/port), authenticates
// with the station callsign, and streams parsed spots.
package dxc

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

// Spot holds a single parsed DX Cluster spot.
type Spot struct {
	DXCall     string  // spotted callsign
	Frequency  float64 // kHz
	Comment    string  // free-text comment
	Spotter    string  // who spotted
	ReceivedAt time.Time
}

// connState is one connection generation's private state. All goroutines of
// a generation receive their own connState pointer and only touch their own
// fields, so no lock is needed between them — except pendingRsp, which both
// the reader and SendSpot access (guarded by rspMu), and loginSent, which is
// CAS-protected so the login is sent exactly once.
type connState struct {
	conn       net.Conn
	stopCh     chan struct{}
	loginSent  atomic.Bool
	rspMu      sync.Mutex
	pendingRsp chan string
}

// Client is a DX Cluster telnet connection with auto-reconnect. After the
// first successful connection, the client owns reconnection: it redials with
// exponential backoff until Stop is called. Callers must Stop (which joins
// the client's goroutines) before discarding a client.
//
// Shared state is synchronized through mu; the spot/status channels are
// created once and remain stable across reconnects.
type Client struct {
	host  string
	port  string
	login string

	mu            sync.Mutex
	cur           *connState // current connection generation; nil when idle
	connecting    bool       // a dial attempt is in progress
	stopped       bool       // true after Stop — no further reconnects
	connectedOnce bool       // true after the first successful connection
	reconnecting  bool       // the reconnect loop is running

	spotsCh  chan Spot     // created once in NewClient — stable across reconnects
	statusCh chan bool     // created once in NewClient
	stopAll  chan struct{} // closed once by Stop; wakes the reconnect loop

	loopWG sync.WaitGroup // readLoop + reconnectLoop goroutines
}

// NewClient creates a DX Cluster client. It does not connect until Start is called.
func NewClient(host, port, login string) *Client {
	if host == "" {
		host = "dxspots.com"
	}
	if port == "" {
		port = "7300"
	}
	return &Client{
		host:     host,
		port:     port,
		login:    login,
		spotsCh:  make(chan Spot, 256),
		statusCh: make(chan bool, 1),
		stopAll:  make(chan struct{}),
	}
}

// ConnectedOnce reports whether the client has ever connected successfully.
// The TUI uses this to decide who owns reconnection: before the first
// connection the TUI retries Start; afterwards the client reconnects itself.
func (c *Client) ConnectedOnce() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectedOnce
}

// Start dials the cluster and begins reading spots in a background
// goroutine. Safe to call after a failed Start (retry) or from the internal
// reconnect loop; concurrent dial attempts are refused.
func (c *Client) Start() error {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return fmt.Errorf("dxc: client stopped")
	}
	if c.connecting {
		c.mu.Unlock()
		return fmt.Errorf("dxc: already connecting")
	}
	c.connecting = true
	c.mu.Unlock()

	addr := net.JoinHostPort(c.host, c.port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		c.mu.Lock()
		c.connecting = false
		c.mu.Unlock()
		return fmt.Errorf("dxc dial %s: %w", addr, err)
	}

	st := &connState{conn: conn, stopCh: make(chan struct{})}

	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		conn.Close()
		return fmt.Errorf("dxc: client stopped")
	}
	c.cur = st
	c.connecting = false
	if !c.connectedOnce {
		c.connectedOnce = true
	}
	c.mu.Unlock()

	select {
	case c.statusCh <- true:
	default:
	}
	applog.Info("DXC: connected", "host", addr)

	// Give the cluster 2s to send a login prompt. If none arrives, send
	// login anyway (DX Spider-style immediate login).
	go func() {
		select {
		case <-time.After(2 * time.Second):
			if st.loginSent.CompareAndSwap(false, true) {
				applog.Debug("DXC: no login prompt detected, sending callsign")
				writeLine(st, "%s\r\n", strings.ToUpper(c.login))
			}
		case <-st.stopCh:
		}
	}()

	// Request recent spots after login so the table isn't empty on startup.
	go func() {
		select {
		case <-time.After(2 * time.Second):
			c.requestRecent(st, 50)
		case <-st.stopCh:
		}
	}()

	c.loopWG.Add(1)
	go func() {
		defer c.loopWG.Done()
		c.readLoop(st)
	}()
	return nil
}

// Stop closes the connection, disables reconnection, and waits for the
// client's goroutines to exit. Idempotent.
func (c *Client) Stop() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.stopped = true
	close(c.stopAll)
	st := c.cur
	c.cur = nil
	c.mu.Unlock()

	if st != nil {
		st.conn.Close()
		close(st.stopCh)
	}
	c.loopWG.Wait()
}

// reconnectLoop attempts to reconnect with exponential backoff:
// 2s, 4s, 8s, ... up to 60s. Exits when Stop is called. Runs at most once —
// started by readLoop when the connection drops.
func (c *Client) reconnectLoop() {
	defer func() {
		c.mu.Lock()
		c.reconnecting = false
		c.mu.Unlock()
	}()

	delay := 2 * time.Second
	const maxDelay = 60 * time.Second

	for {
		c.mu.Lock()
		stopped := c.stopped
		c.mu.Unlock()
		if stopped {
			return
		}

		applog.Info("DXC: reconnecting", "delay", delay.Round(time.Second))

		select {
		case <-time.After(delay):
		case <-c.stopAll:
			return
		}

		c.mu.Lock()
		stopped = c.stopped
		c.mu.Unlock()
		if stopped {
			return
		}

		if err := c.Start(); err != nil {
			if strings.Contains(err.Error(), "stopped") {
				return
			}
			applog.Warn("DXC: reconnect failed", "error", err.Error(), "next", delay*2)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}
		applog.Info("DXC: reconnected")
		return
	}
}

// Spots returns the channel on which parsed spots are delivered.
func (c *Client) Spots() <-chan Spot {
	return c.spotsCh
}

// Status returns a channel that receives true on connect and false on disconnect.
func (c *Client) Status() <-chan bool {
	return c.statusCh
}

// RequestRecent asks the cluster for the last n spots via SH/FDX.
func (c *Client) RequestRecent(n int) {
	c.mu.Lock()
	st := c.cur
	c.mu.Unlock()
	if st == nil {
		return
	}
	c.requestRecent(st, n)
}

func (c *Client) requestRecent(st *connState, n int) {
	writeLine(st, "SH/FDX %d\r\n", n)
	applog.Debug("DXC: requested recent spots via SH/FDX", "count", n)
}

// writeLine writes a line to the given connection generation, logging it at
// DEBUG level.
func writeLine(st *connState, format string, args ...interface{}) {
	if st == nil || st.conn == nil {
		return
	}
	line := fmt.Sprintf(format, args...)
	applog.Debug("DXC: tx", "line", strings.TrimRight(line, "\r\n"))
	fmt.Fprint(st.conn, line)
}

// SendSpot sends a DX spot to the cluster and returns any response
// from the cluster (e.g. error message). Returns empty string on
// timeout or if the cluster doesn't send a response.
// Format: DX [freq_kHz] [call] [comment]
func (c *Client) SendSpot(freqKhz float64, call, comment string) (string, error) {
	c.mu.Lock()
	st := c.cur
	c.mu.Unlock()
	if st == nil {
		return "", fmt.Errorf("dxc: not connected")
	}
	line := fmt.Sprintf("DX %.1f %s %s\r\n", freqKhz, strings.ToUpper(call), comment)

	// Set up a response channel before writing so readLoop can capture the reply.
	rspCh := make(chan string, 1)
	st.rspMu.Lock()
	st.pendingRsp = rspCh
	st.rspMu.Unlock()

	writeLine(st, "%s", line)
	applog.Info("DXC: spot sent", "call", call, "freq", freqKhz, "comment", comment)

	// Wait up to 1.5s for a response (error message or confirmation).
	select {
	case rsp := <-rspCh:
		st.rspMu.Lock()
		st.pendingRsp = nil
		st.rspMu.Unlock()
		if rsp != "" {
			applog.Warn("DXC: cluster response", "response", rsp)
		}
		return rsp, nil
	case <-time.After(1500 * time.Millisecond):
		st.rspMu.Lock()
		st.pendingRsp = nil
		st.rspMu.Unlock()
		return "", nil
	}
}

// readLoop reads lines from the telnet connection, parses spots,
// and delivers them to the spots channel. On disconnect, it hands
// reconnection to reconnectLoop (unless the client was stopped).
func (c *Client) readLoop(st *connState) {
	defer func() {
		st.conn.Close()
		// Signal disconnect, then hand off reconnection unless stopped.
		select {
		case c.statusCh <- false:
		default:
		}
		c.mu.Lock()
		if c.cur == st {
			c.cur = nil
		}
		stopped := c.stopped
		if !stopped && !c.reconnecting {
			c.reconnecting = true
			c.loopWG.Add(1)
			go func() {
				defer c.loopWG.Done()
				c.reconnectLoop()
			}()
		}
		c.mu.Unlock()
	}()

	scanner := bufio.NewScanner(st.conn)
	// DX cluster lines can be long (comments); 4KB is enough for typical spots.
	scanner.Buffer(make([]byte, 4096), 4096)

	for scanner.Scan() {
		select {
		case <-st.stopCh:
			return
		default:
		}

		line := scanner.Text()

		// DEBUG: log every line from the cluster for troubleshooting.
		applog.Debug("DXC: raw", "line", line)

		spot, ok := parseSpot(line)
		if !ok {
			// Log cluster error messages (*** followed by a space — real errors,
			// not decorative ****** banners).
			if strings.HasPrefix(line, "*** ") {
				applog.Warn("DXC: cluster message", "line", line)
			}

			// Detect login prompts — respond once with callsign.
			// Handles CC Cluster ("Please enter your call: "),
			// AR-Cluster ("login:"), and similar.
			lower := strings.ToLower(line)
			if strings.Contains(lower, "enter your call") ||
				strings.Contains(lower, "enter your callsign") ||
				strings.HasPrefix(lower, "login:") {
				if st.loginSent.CompareAndSwap(false, true) {
					applog.Info("DXC: login prompt detected, sending callsign", "line", line)
					writeLine(st, "%s\r\n", strings.ToUpper(c.login))
				}
			}

			// If there's a pending response reader (e.g. after SendSpot),
			// forward any non-spot, non-prompt line that could be a response.
			// Prompts look like "CALL de CLUSTER date time XXX >"
			isPrompt := strings.HasSuffix(strings.TrimSpace(line), ">") &&
				strings.Contains(line, "de")
			st.rspMu.Lock()
			pr := st.pendingRsp
			st.rspMu.Unlock()
			if pr != nil &&
				!strings.HasPrefix(line, "DX de") &&
				!isPrompt {
				select {
				case pr <- line:
				default:
				}
			}
			continue
		}
		spot.ReceivedAt = time.Now().UTC()

		select {
		case c.spotsCh <- spot:
		default:
			// Channel full — drop oldest.
			select {
			case <-c.spotsCh:
			default:
			}
			c.spotsCh <- spot
		}
	}

	if err := scanner.Err(); err != nil {
		applog.Warn("DXC: read error", "error", err)
	}
}

// parseSpot attempts to parse a DX cluster spot line.
// Format: DX de SPOTTER:  FREQ   CALL   COMMENT
// Example: DX de SP9MOA:  14074.0  K1ABC  FT8 TNX
func parseSpot(line string) (Spot, bool) {
	// Must start with "DX de "
	const prefix = "DX de "
	if !strings.HasPrefix(line, prefix) {
		return Spot{}, false
	}

	rest := line[len(prefix):]

	// Split on colon: spotter : rest
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return Spot{}, false
	}
	spotter := strings.TrimSpace(rest[:colon])
	if spotter == "" {
		return Spot{}, false
	}
	rest = strings.TrimSpace(rest[colon+1:])

	// Now rest is: "FREQ   CALL   COMMENT"
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return Spot{}, false
	}

	freq, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return Spot{}, false
	}

	dxCall := strings.TrimSpace(fields[1])
	if dxCall == "" {
		return Spot{}, false
	}
	comment := ""
	if len(fields) > 2 {
		comment = strings.Join(fields[2:], " ")
	}

	return Spot{
		DXCall:    strings.ToUpper(dxCall),
		Frequency: freq,
		Comment:   comment,
		Spotter:   strings.ToUpper(spotter),
	}, true
}
