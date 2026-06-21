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
	host       string
	port       string
	login      string
	conn       net.Conn
	spotsCh    chan Spot
	stopCh     chan struct{}
	pendingRsp chan string // set before sending a command, read by readLoop
	loginSent  bool        // true after login prompt responded to (avoid loops)
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

	// Send login. Uppercase — some clusters reject lowercase.
	c.writeLine("%s\r\n", strings.ToUpper(c.login))

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
		c.writeLine("SH/FDX 50\r\n")
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

// writeLine writes a line to the cluster, logging it at DEBUG level.
func (c *Client) writeLine(format string, args ...interface{}) {
	if c.conn == nil {
		return
	}
	line := fmt.Sprintf(format, args...)
	applog.Debug("DXC: tx", "line", strings.TrimRight(line, "\r\n"))
	fmt.Fprint(c.conn, line)
}

// SendSpot sends a DX spot to the cluster and returns any response
// from the cluster (e.g. error message). Returns empty string on
// timeout or if the cluster doesn't send a response.
// Format: DX [freq_kHz] [call] [comment]
func (c *Client) SendSpot(freqKhz float64, call, comment string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("dxc: not connected")
	}
	line := fmt.Sprintf("DX %.1f %s %s\r\n", freqKhz, strings.ToUpper(call), comment)

	// Set up a response channel before writing so readLoop can capture the reply.
	rspCh := make(chan string, 1)
	c.pendingRsp = rspCh

	c.writeLine("%s", line)
	applog.Info("DXC: spot sent", "call", call, "freq", freqKhz, "comment", comment)

	// Wait up to 1.5s for a response (error message or confirmation).
	select {
	case rsp := <-rspCh:
		c.pendingRsp = nil
		if rsp != "" {
			applog.Warn("DXC: cluster response", "response", rsp)
		}
		return rsp, nil
	case <-time.After(1500 * time.Millisecond):
		c.pendingRsp = nil
		return "", nil
	}
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

		// DEBUG: log every line from the cluster for troubleshooting.
		applog.Debug("DXC: raw", "line", line)

		spot, ok := parseSpot(line)
		if !ok {
			// Log cluster error messages (*** followed by a space — real errors,
			// not decorative ****** banners).
			if strings.HasPrefix(line, "*** ") {
				applog.Warn("DXC: cluster message", "line", line)
			}

			// Detect login prompts — some clusters (CC Cluster, AR-Cluster) ask
			// for the callsign after the MOTD.  Skip lines that are error
			// responses ("login: xyz command error") to avoid infinite loops.
			lower := strings.ToLower(line)
			isLoginError := strings.HasPrefix(lower, "login:") && strings.Contains(lower, "error")
			if !isLoginError && !c.loginSent {
				if strings.Contains(lower, "enter your call") ||
					strings.Contains(lower, "enter your callsign") ||
					strings.HasPrefix(lower, "login:") {
					applog.Info("DXC: login prompt detected, sending callsign", "line", line)
					c.writeLine("%s\r\n", strings.ToUpper(c.login))
					c.loginSent = true
				}
			}

			// If there's a pending response reader (e.g. after SendSpot),
			// forward any non-spot, non-prompt line that could be a response.
			// Prompts look like "CALL de CLUSTER date time XXX >"
			isPrompt := strings.HasSuffix(strings.TrimSpace(line), ">") &&
				strings.Contains(line, "de")
			if c.pendingRsp != nil &&
				!strings.HasPrefix(line, "DX de") &&
				!isPrompt {
				select {
				case c.pendingRsp <- line:
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
