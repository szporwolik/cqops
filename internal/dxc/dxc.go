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

// Client is a DX Cluster telnet connection.
type Client struct {
	host    string
	port    string
	login   string
	conn    net.Conn
	spotsCh chan Spot
	stopCh  chan struct{}
}

// NewClient creates a DX Cluster client. It does not connect until Start is called.
func NewClient(host, port, login string) *Client {
	if host == "" {
		host = "dxspider.co.uk"
	}
	if port == "" {
		port = "7300"
	}
	return &Client{
		host:    host,
		port:    port,
		login:   login,
		spotsCh: make(chan Spot, 256),
		stopCh:  make(chan struct{}),
	}
}

// Start connects to the cluster and begins reading spots in a background goroutine.
// Spots are delivered on the channel returned by Spots().
func (c *Client) Start() error {
	if c.conn != nil {
		c.Stop()
	}

	addr := net.JoinHostPort(c.host, c.port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dxc dial %s: %w", addr, err)
	}

	c.conn = conn
	c.stopCh = make(chan struct{})
	c.spotsCh = make(chan Spot, 256)

	applog.Info("DXC: connected", "host", addr)

	// Send login. dxspider expects just the callsign.
	fmt.Fprintf(conn, "%s\r\n", c.login)

	// Request recent spots after login so the table isn't empty on startup.
	// SH/FDX (alias "SH/DX real") delivers historical spots in the same
	// "DX de SPOTTER:" realtime format that parseSpot already handles.
	// If the cluster doesn't support SH/FDX, the error response is just
	// ignored by parseSpot — no harm done. The 2s delay gives the cluster
	// time to finish its login handshake (MOTD, prompt, etc.).
	go func() {
		time.Sleep(2 * time.Second)
		if c.conn == nil {
			return
		}
		_, err := fmt.Fprintf(c.conn, "SH/FDX 50\r\n")
		if err != nil {
			applog.Debug("DXC: SH/FDX request failed (cluster may not support it)", "error", err)
			return
		}
		applog.Debug("DXC: requested recent spots via SH/FDX")
	}()

	go c.readLoop()
	return nil
}

// Stop closes the connection and stops the read goroutine.
func (c *Client) Stop() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

// Spots returns the channel on which parsed spots are delivered.
func (c *Client) Spots() <-chan Spot {
	return c.spotsCh
}

// Connected reports whether the client is currently connected.
func (c *Client) Connected() bool {
	return c.conn != nil
}

// SendSpot sends a DX spot to the cluster.
// Format: DX [freq_kHz] [call] [comment]
func (c *Client) SendSpot(freqKhz float64, call, comment string) error {
	if c.conn == nil {
		return fmt.Errorf("dxc: not connected")
	}
	line := fmt.Sprintf("DX %.1f %s %s\r\n", freqKhz, strings.ToUpper(call), comment)
	_, err := fmt.Fprint(c.conn, line)
	if err != nil {
		return fmt.Errorf("dxc: send spot: %w", err)
	}
	applog.Info("DXC: spot sent", "call", call, "freq", freqKhz, "comment", comment)
	return nil
}

// readLoop reads lines from the telnet connection, parses spots,
// and delivers them to the spots channel.
func (c *Client) readLoop() {
	defer func() {
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
	}()

	scanner := bufio.NewScanner(c.conn)
	// DX cluster lines can be long (comments); 4KB is enough for typical spots.
	scanner.Buffer(make([]byte, 4096), 4096)

	for scanner.Scan() {
		select {
		case <-c.stopCh:
			return
		default:
		}

		line := scanner.Text()
		spot, ok := parseSpot(line)
		if !ok {
			// Log cluster error messages (*** followed by a space — real errors,
			// not decorative ****** banners).
			if strings.HasPrefix(line, "*** ") {
				applog.Warn("DXC: cluster message", "line", line)
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
// Example: DX de SP9SPM:  14074.0  K1ABC  FT8 TNX
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
