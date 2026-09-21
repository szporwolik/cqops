package hamlib

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/rotor"
)

// Client talks to a hamlib rotctld instance over TCP.
type Client struct {
	addr    string
	timeout time.Duration

	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader // framing reader for conn — same lifetime
}

// New creates a new rotor client for the given address.
func New(host, port string, timeout time.Duration) *Client {
	addr := net.JoinHostPort(host, port)
	return &Client{addr: addr, timeout: timeout}
}

// Close releases the persistent connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		conn := c.conn
		c.conn = nil
		c.reader = nil
		return conn.Close()
	}
	return nil
}

// Status queries the rotor for current azimuth and elevation.
func (c *Client) Status(ctx context.Context) (rotor.Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.getConn(ctx)
	if err != nil {
		return rotor.Status{}, err
	}

	// 'p' returns "azimuth\nelevation\n" in line mode.
	raw, err := c.cmd(conn, "p")
	if err != nil {
		c.dropConn()
		return rotor.Status{}, fmt.Errorf("rotor position: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) < 2 {
		// Some backends may return only azimuth.
		lines = append(lines, "0.0")
	}

	az := parseFloat(lines[0])
	el := parseFloat(lines[1])

	// Clamp to reasonable ranges.
	az = clamp(az, 0, 360)
	el = clamp(el, -90, 90)

	applog.Debug("rotor: status", "az", az, "el", el)
	return rotor.Status{Connected: true, Azimuth: az, Elevation: el}, nil
}

// SetPosition commands the rotor to turn to a specific azimuth/elevation.
func (c *Client) SetPosition(ctx context.Context, az, el float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.getConn(ctx)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("P %.6f %.6f", az, el)
	_, err = c.cmd(conn, cmd)
	if err != nil {
		c.dropConn()
		return fmt.Errorf("rotor set: %w", err)
	}
	return nil
}

// Stop sends the emergency-stop command to the rotor.
func (c *Client) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.getConn(ctx)
	if err != nil {
		return err
	}

	_, err = c.cmd(conn, "S")
	if err != nil {
		c.dropConn()
		return fmt.Errorf("rotor stop: %w", err)
	}
	return nil
}

// GetName returns the rotor model name from rotctld.
func (c *Client) GetName(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.getConn(ctx)
	if err != nil {
		return "", err
	}

	raw, err := c.cmd(conn, "_")
	if err != nil {
		c.dropConn()
		return "", fmt.Errorf("rotor name: %w", err)
	}
	return strings.TrimSpace(raw), nil
}

// getConn returns the persistent connection or dials a new one.
func (c *Client) getConn(ctx context.Context) (net.Conn, error) {
	if c.conn != nil {
		return c.conn, nil
	}

	applog.Debug("rotor: dialing", "addr", c.addr)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		applog.Debug("rotor: dial failed", "addr", c.addr, "error", err)
		return nil, fmt.Errorf("rotor dial %s: %w", c.addr, err)
	}
	applog.Debug("rotor: connected", "addr", c.addr)
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return c.conn, nil
}

// dropConn closes the persistent connection after an error.
func (c *Client) dropConn() {
	if c.conn != nil {
		applog.Debug("rotor: closing connection", "addr", c.addr)
		c.conn.Close()
		c.conn = nil
		c.reader = nil
	}
}

// cmd sends a command and returns the complete response payload. rotctld
// terminates every reply with an RPRT line, so the persistent reader consumes
// whole lines until that terminator — a response split across TCP reads is
// reassembled, buffered leftovers cannot leak into the next command, and an
// acknowledgement cut short reports an error instead of success.
func (c *Client) cmd(conn net.Conn, cmd string) (string, error) {
	if c.timeout > 0 {
		conn.SetDeadline(time.Now().Add(c.timeout))
	}
	if _, err := fmt.Fprintf(conn, "%s\r\n", cmd); err != nil {
		return "", err
	}

	var lines []string
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("rotor read: %w", err)
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "RPRT") {
			if line != "RPRT 0" {
				return "", fmt.Errorf("rotor error: %s", line)
			}
			return strings.Join(lines, "\n"), nil
		}
		lines = append(lines, line)
	}
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}

func clamp(v, lo, hi float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return max(lo, min(v, hi))
}
