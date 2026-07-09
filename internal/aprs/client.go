package aprs

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/version"
)

// ErrAuthFailed is returned when the APRS-IS server rejects the login.
// This is a permanent error — retrying with the same credentials won't help.
var ErrAuthFailed = fmt.Errorf("APRS: authentication failed")

// clientVersion is resolved once at startup so we don't read the VERSION
// file or invoke the filesystem on every APRS connection attempt.
var clientVersion = version.Resolved()

// Client is the interface for APRS data sources (TCP APRS-IS or KISS serial).
type Client interface {
	Start()
	Stop()
	IsRunning() bool
	IsConnected() bool
}

// TCPClient maintains a persistent connection to an APRS-IS server,
// receives position reports, and stores them in a local cache.
type TCPClient struct {
	mu        sync.Mutex
	conn      net.Conn
	scanner   *bufio.Scanner
	server    string
	callsign  string
	passcode  string
	filter    string
	running   bool
	stopCh    chan struct{}
	doneCh    chan struct{}
	connected bool

	// Callbacks
	OnPacket func(raw string)                // called for each received APRS packet line
	OnStatus func(connected bool, err error) // called on connect/disconnect/reconnect
}

// NewTCPClient creates an APRS-IS client. Call Start() to connect.
func NewTCPClient(server, callsign, passcode, filter string) *TCPClient {
	return &TCPClient{
		server:   server,
		callsign: callsign,
		passcode: passcode,
		filter:   filter,
	}
}

// Start connects to the APRS-IS server and begins receiving packets.
// Non-blocking — connection runs in a goroutine. Use OnStatus for feedback.
func (c *TCPClient) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	c.mu.Unlock()

	go c.runLoop()
}

// Stop disconnects and shuts down the receive loop.
func (c *TCPClient) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	close(c.stopCh)
	c.disconnectLocked()
	c.mu.Unlock()

	// Wait for run loop to finish.
	select {
	case <-c.doneCh:
	case <-time.After(3 * time.Second):
	}
	applog.Info("APRS: client stopped")
}

// IsRunning returns true when the client is active (may not be connected yet).
func (c *TCPClient) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// IsConnected returns true when we have an active TCP connection.
func (c *TCPClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// UpdateFilter changes the server-side filter and reconnects.
func (c *TCPClient) UpdateFilter(filter string) {
	c.mu.Lock()
	c.filter = filter
	wasRunning := c.running
	c.disconnectLocked()
	c.mu.Unlock()

	if wasRunning {
		c.Start()
	}
}

// runLoop is the main goroutine — connects, receives, reconnects.
// Exponential backoff on transient errors: 2s → 4s → 8s → … → max 60s.
// Permanent errors (ErrAuthFailed) stop the loop immediately.
func (c *TCPClient) runLoop() {
	defer close(c.doneCh)

	delay := 2 * time.Second
	const maxDelay = 60 * time.Second

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		if err := c.connect(); err != nil {
			// Permanent error — don't retry. connect() already logged the detail.
			if errors.Is(err, ErrAuthFailed) {
				c.setConnected(false, err)
				return
			}
			// Transient error — retry with backoff.
			applog.Warn("APRS: connect failed, will retry", "server", c.server, "error", err, "nextDelay", delay)
			c.setConnected(false, err)
			select {
			case <-c.stopCh:
				return
			case <-time.After(delay):
			}
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		// Connection successful — reset backoff.
		delay = 2 * time.Second
		applog.Info("APRS: connected", "server", c.server, "callsign", c.callsign)
		c.setConnected(true, nil)

		// Receive loop blocks until connection drops.
		c.receiveLoop()

		// Connection lost — reconnect with backoff.
		c.mu.Lock()
		c.disconnectLocked()
		c.mu.Unlock()
		c.setConnected(false, fmt.Errorf("connection lost"))
		applog.Debug("APRS: disconnected", "reconnect_delay", delay)

		select {
		case <-c.stopCh:
			return
		case <-time.After(delay):
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

func (c *TCPClient) setConnected(connected bool, err error) {
	c.mu.Lock()
	c.connected = connected
	c.mu.Unlock()
	if c.OnStatus != nil {
		c.OnStatus(connected, err)
	}
}

func (c *TCPClient) connect() error {
	applog.Debug("APRS: connecting", "server", c.server)
	conn, err := net.DialTimeout("tcp", c.server, DefaultTimeout)
	if err != nil {
		applog.Error("APRS: connect failed", "server", c.server, "error", err)
		return fmt.Errorf("APRS connect: %w", err)
	}

	if err := conn.SetDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		conn.Close()
		return fmt.Errorf("APRS set deadline: %w", err)
	}

	// Build login string with optional filter.
	login := fmt.Sprintf("user %s pass %s vers CQOps %s", c.callsign, c.passcode, clientVersion)
	if c.filter != "" {
		login += " filter " + c.filter
	}
	login += "\r\n"

	applog.Debug("APRS: sending login", "callsign", c.callsign, "passcodeLen", len(c.passcode), "filter", c.filter, "server", c.server)
	// Log the exact login line with passcode masked for security.
	safe := strings.Replace(login, " pass "+c.passcode, " pass ****", 1)
	applog.Debug("APRS: login string", "line", safe[:len(safe)-2] /* strip \r\n */)
	if _, err := fmt.Fprint(conn, login); err != nil {
		conn.Close()
		applog.Error("APRS: write login failed", "error", err)
		return fmt.Errorf("APRS login: %w", err)
	}

	// Read server banner and verify login.
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			applog.Error("APRS: read banner failed", "error", err)
			return fmt.Errorf("APRS banner: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		applog.Debug("APRS: banner", "line", line)

		if strings.HasPrefix(line, "# logresp ") {
			if strings.Contains(line, " verified") {
				break
			}
			conn.Close()
			applog.Error("APRS: login rejected", "line", line)
			return fmt.Errorf("%w: %s", ErrAuthFailed, line)
		}
		if !strings.HasPrefix(line, "#") {
			conn.Close()
			return fmt.Errorf("APRS unexpected response: %s", line)
		}
	}

	// Remove deadline for steady-state receive.
	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return fmt.Errorf("APRS clear deadline: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.scanner = bufio.NewScanner(conn)
	c.mu.Unlock()
	return nil
}

func (c *TCPClient) disconnectLocked() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.scanner = nil
}

// receiveLoop reads packets from the active connection. It returns when
// the scanner fails (connection lost) or the client is stopped.
func (c *TCPClient) receiveLoop() {
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.mu.Lock()
		scanner := c.scanner
		c.mu.Unlock()

		if scanner == nil {
			return
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				applog.Debug("APRS: read error", "error", err)
			}
			return
		}

		line := strings.TrimRight(scanner.Text(), "\r\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		applog.Debug("APRS: received packet", "line", line)

		if c.OnPacket != nil {
			c.OnPacket(line)
		}
	}
}

// BuildRangeFilter creates an APRS-IS range filter string.
// lat/lon in decimal degrees, radiusKm in kilometres.
// Example: "r/50.0/20.0/50" for 50km around 50°N 20°E.
func BuildRangeFilter(lat, lon float64, radiusKm int) string {
	return fmt.Sprintf("r/%.1f/%.1f/%d", lat, lon, radiusKm)
}

// SendPosition transmits an uncompressed APRS position report over the
// active connection. Returns an error if the client is not connected or
// the write fails. Safe for concurrent use with the receive loop.
//
// The packet is sent in the standard APRS-IS inject format:
//
//	CALLSIGN>APRS,TCPIP*:!DDMM.hhN/DDDMM.hhW<symbol>/...comment
func (c *TCPClient) SendPosition(callsign string, lat, lon float64, symbol, comment string) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("APRS send: not connected")
	}

	// Build uncompressed position body.
	body := formatUncompressedPosition(lat, lon, symbol, comment)
	packet := fmt.Sprintf("%s>APRS,TCPIP*:%s\r\n", strings.ToUpper(callsign), body)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("APRS send: connection lost")
	}
	if _, err := fmt.Fprint(c.conn, packet); err != nil {
		applog.Error("APRS: send position failed", "error", err)
		return fmt.Errorf("APRS send: %w", err)
	}
	applog.Info("APRS: position sent", "callsign", callsign, "lat", lat, "lon", lon)
	return nil
}

// TestKISSServerConnection tries to open a brief TCP connection to the
// KISS server at addr and sends a KISS command frame (no operation) to
// verify the server is reachable and speaks KISS.
func TestKISSServerConnection(addr string) error {
	const fend byte = 0xC0
	const kissCmdReset byte = 0xFF

	conn, err := net.DialTimeout("tcp", addr, DefaultTimeout)
	if err != nil {
		return fmt.Errorf("KISS server: connect: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return fmt.Errorf("KISS server: set deadline: %w", err)
	}

	// Send a KISS reset command (FEND CMD FEND) as a simple probe.
	if _, err := conn.Write([]byte{fend, kissCmdReset, fend}); err != nil {
		return fmt.Errorf("KISS server: write: %w", err)
	}

	// Try to read any response — not required, just a connectivity test.
	buf := make([]byte, 64)
	conn.Read(buf)

	return nil
}

// APRS position report in the standard format:
//
//	!DDMM.hhN<sym_table>DDDMM.hhW<sym_code>/...comment
//
// The symbol TABLE goes between latitude hemisphere and longitude;
// the symbol CODE goes after longitude hemisphere.
func formatUncompressedPosition(lat, lon float64, symbol, comment string) string {
	latHemi := 'N'
	if lat < 0 {
		latHemi = 'S'
		lat = -lat
	}
	latDeg := int(lat)
	latMin := (lat - float64(latDeg)) * 60.0

	lonHemi := 'E'
	if lon < 0 {
		lonHemi = 'W'
		lon = -lon
	}
	lonDeg := int(lon)
	lonMin := (lon - float64(lonDeg)) * 60.0

	// Ensure symbol is exactly 2 chars.
	if len(symbol) == 0 {
		symbol = "/-"
	} else if len(symbol) == 1 {
		symbol = "/" + symbol
	}
	if len(symbol) > 2 {
		symbol = symbol[:2]
	}
	symTable := symbol[0]
	symCode := symbol[1]

	// Standard uncompressed APRS format:
	//   !<lat><hemi><sym_table><lon><hemi><sym_code>/...<comment>
	body := fmt.Sprintf("!%02d%05.2f%c%c%03d%05.2f%c%c",
		latDeg, math.Round(latMin*100)/100, latHemi,
		symTable,
		lonDeg, math.Round(lonMin*100)/100, lonHemi,
		symCode,
	)

	if comment != "" {
		body += "/" + comment
	}
	return body
}
