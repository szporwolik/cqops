package aprs

import (
	"bufio"
	"errors"
	"fmt"
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

// Client maintains a persistent connection to an APRS-IS server,
// receives position reports, and stores them in a local cache.
type Client struct {
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

// NewClient creates an APRS-IS client. Call Start() to connect.
func NewClient(server, callsign, passcode, filter string) *Client {
	return &Client{
		server:   server,
		callsign: callsign,
		passcode: passcode,
		filter:   filter,
	}
}

// Start connects to the APRS-IS server and begins receiving packets.
// Non-blocking — connection runs in a goroutine. Use OnStatus for feedback.
func (c *Client) Start() {
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
func (c *Client) Stop() {
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
func (c *Client) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// IsConnected returns true when we have an active TCP connection.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// UpdateFilter changes the server-side filter and reconnects.
func (c *Client) UpdateFilter(filter string) {
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
func (c *Client) runLoop() {
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
		applog.Debug("APRS: disconnected, reconnecting in %v", delay)

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

func (c *Client) setConnected(connected bool, err error) {
	c.mu.Lock()
	c.connected = connected
	c.mu.Unlock()
	if c.OnStatus != nil {
		c.OnStatus(connected, err)
	}
}

func (c *Client) connect() error {
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

func (c *Client) disconnectLocked() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.scanner = nil
}

// receiveLoop reads packets from the active connection. It returns when
// the scanner fails (connection lost) or the client is stopped.
func (c *Client) receiveLoop() {
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
