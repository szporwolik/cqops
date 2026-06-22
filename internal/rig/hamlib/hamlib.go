package hamlib

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig"
)

// Client is a TCP client for hamlib rigctld.  The underlying TCP connection
// is persistent — opened on first use and reused across Status/SetFrequency/
// SetMode/GetName calls.  A dropped connection is re-established transparently.
// All command execution is serialised via cmdMu to prevent interleaving when
// Bubble Tea runs Status/GetName/Power commands concurrently.
type Client struct {
	addr    string
	timeout time.Duration

	mu        sync.Mutex // protects conn/r establishment and teardown
	cmdMu     sync.Mutex // serialises all command execution on the connection
	conn      net.Conn
	r         *bufio.Reader
	maxPower  float64 // rig max power in watts (default 100)
	vfoMode   bool    // true if rigctld was started with --vfo
	vfoProbed bool    // true after first probe attempt (pass or fail)

	modesOnce  sync.Once
	modesCache []string // populated on first GetModes call
}

// New returns a new hamlib client.
func New(host, port string, timeout time.Duration) *Client {
	return &Client{
		addr:     net.JoinHostPort(host, port),
		timeout:  timeout,
		maxPower: 100,
	}
}

// VfoMode reports whether rigctld is running with --vfo, which allows
// querying specific VFOs (e.g. 'f VFOB') without switching.
func (c *Client) VfoMode() bool { return c.vfoMode }

// SetVfoMode forces the VFO mode flag — for testing only.
func (c *Client) SetVfoMode(v bool) { c.vfoMode = v; c.vfoProbed = true }

// SetMaxPower configures the maximum RF power used to convert the
// normalised RFPOWER level (0–1) into approximate watts.
func (c *Client) SetMaxPower(watts float64) {
	if watts > 0 {
		c.maxPower = watts
	}
}

// Close releases the persistent connection (if any).
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		conn := c.conn
		c.conn = nil
		c.r = nil
		return conn.Close()
	}
	return nil
}

// getConn returns the current persistent connection, or dials a new one.
func (c *Client) getConn(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn, c.r, nil
	}

	applog.Debug("hamlib: dialing", "addr", c.addr)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		applog.Debug("hamlib: dial failed", "addr", c.addr, "error", err)
		return nil, nil, fmt.Errorf("hamlib dial %s: %w", c.addr, err)
	}
	applog.Debug("hamlib: connected", "addr", c.addr)
	c.conn = conn
	c.r = bufio.NewReader(conn)
	return c.conn, c.r, nil
}

// dropConn closes the persistent connection after an error.
func (c *Client) dropConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		applog.Debug("hamlib: closing connection", "addr", c.addr)
		c.conn.Close()
		c.conn = nil
		c.r = nil
	}
}

// Status queries the rig for frequency, mode, VFO, split status.
// On first call it probes whether rigctld is running with --vfo and
// adapts command formats accordingly.
func (c *Client) Status(ctx context.Context) (rig.RigStatus, error) {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	defer c.discardReader()

	// Probe VFO mode on first call only.
	if !c.vfoProbed {
		c.vfoProbed = true
		c.probeVfoMode(ctx)
	}

	conn, r, err := c.getConn(ctx)
	if err != nil {
		return rig.RigStatus{}, err
	}

	// Read current VFO name.  'v' works in both modes.
	vfo, err := c.cmd(r, conn, "v")
	if err != nil {
		c.dropConn()
		return rig.RigStatus{}, fmt.Errorf("hamlib: vfo: %w", err)
	}
	vfo = strings.TrimSpace(vfo)

	// Split status.  With --vfo, single-char 's' returns nothing — must
	// use VFO-prefixed 's VFOA'.  Without --vfo, plain 's' works.
	var splitRaw string
	if c.vfoMode {
		splitRaw, err = c.cmdDrain(r, conn, "s VFOA")
	} else {
		splitRaw, err = c.cmd(r, conn, "s")
	}
	if err != nil {
		c.dropConn()
		return rig.RigStatus{}, fmt.Errorf("hamlib: split: %w", err)
	}
	split := strings.TrimSpace(splitRaw) == "1"

	// Main VFO frequency.  With --vfo, 'f VFOA' targets the VFO; without
	// --vfo, plain 'f' returns the active VFO frequency.
	var freqHz string
	if c.vfoMode {
		freqHz, err = c.cmdDrain(r, conn, "f VFOA")
	} else {
		freqHz, err = c.cmd(r, conn, "f")
	}
	if err != nil {
		c.dropConn()
		return rig.RigStatus{}, fmt.Errorf("hamlib: frequency: %w", err)
	}

	// Mode.  With --vfo, 'm' needs the VFO prefix; without, plain 'm'.
	var mode string
	if c.vfoMode {
		mode, err = c.cmdDrain(r, conn, "m VFOA")
	} else {
		mode, err = c.cmd(r, conn, "m")
	}
	if err != nil {
		c.dropConn()
		return rig.RigStatus{}, fmt.Errorf("hamlib: mode: %w", err)
	}

	freq := parseFloat(freqHz)
	freqMHz := freq / 1e6

	rs := rig.RigStatus{
		Provider:     "hamlib",
		Connected:    true,
		FrequencyHz:  int64(freq),
		FrequencyMHz: freqMHz,
		Mode:         mode,
		Split:        split,
		Band:         qso.DeriveBand(freqMHz),
	}

	// When split is active AND VFO mode is enabled, query the other VFO.
	// Without VFO mode the per-VFO argument is ignored — we'd just get
	// the same frequency again, so skip the query entirely.
	if split && vfo != "" && c.vfoMode {
		other := otherVFO(vfo)
		if other != "" {
			sf, err := c.cmdDrain(r, conn, "f "+other)
			if err != nil {
				applog.Debug("hamlib: vfoB query failed", "vfo", other, "error", err)
			} else {
				sfreq := parseFloat(sf)
				if sfreq > 0 {
					rs.FrequencyRxHz = int64(sfreq)
					rs.FrequencyRxMHz = sfreq / 1e6
				} else {
					applog.Debug("hamlib: vfoB returned zero", "vfo", other, "raw", sf)
				}
			}
		}
	}

	applog.Debug("hamlib: status",
		"freq", freqMHz,
		"freqRx", rs.FrequencyRxMHz,
		"mode", mode,
		"vfo", vfo,
		"split", split,
	)
	return rs, nil
}

// probeVfoMode queries f VFOA and f VFOB on separate temp connections.
// If both return numeric values that *differ*, --vfo is truly enabled and
// per-VFO queries work.  If they match (or either fails), the VFO argument
// is likely being ignored — typical of rigctld started without --vfo.
// Fails silently; VFO mode stays false if the probe is inconclusive.
func (c *Client) probeVfoMode(ctx context.Context) {
	readFreq := func(vfo string) (float64, bool) {
		probeCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()

		var d net.Dialer
		conn, err := d.DialContext(probeCtx, "tcp", c.addr)
		if err != nil {
			return 0, false
		}
		defer conn.Close()

		conn.SetDeadline(time.Now().Add(300 * time.Millisecond))
		fmt.Fprintf(conn, "f %s\r\n", vfo)
		r := bufio.NewReader(conn)
		resp, err := r.ReadString('\n')
		if err != nil {
			return 0, false
		}
		resp = strings.TrimSpace(resp)
		if strings.HasPrefix(resp, "RPRT ") {
			return 0, false
		}
		return parseFloat(resp), true
	}

	freqA, okA := readFreq("VFOA")
	freqB, okB := readFreq("VFOB")
	if okA && okB && freqA != freqB {
		c.vfoMode = true
		applog.Debug("hamlib: vfo mode detected", "freqA", freqA, "freqB", freqB)
	} else {
		applog.Debug("hamlib: vfo mode not detected",
			"freqA", freqA, "okA", okA,
			"freqB", freqB, "okB", okB,
		)
	}
}

// discardReader replaces the shared buffered reader with a fresh one,
// ensuring the next method call starts with a clean buffer.
func (c *Client) discardReader() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.r = bufio.NewReader(c.conn)
	}
}

// otherVFO returns the companion VFO name (e.g. VFOA ↔ VFOB).
func otherVFO(vfo string) string {
	switch strings.ToUpper(strings.TrimSpace(vfo)) {
	case "VFOA":
		return "VFOB"
	case "VFOB":
		return "VFOA"
	default:
		return ""
	}
}

// Power queries the rig for current RF power setting.
// Hamlib returns RFPOWER as a normalised 0.0–1.0 value (fraction of max
// power), not watts.  We multiply by maxPower to produce approximate watts.
// In VFO mode the VFO argument goes FIRST: "l VFOA RFPOWER".  Falls back
// to "l RFPOWER" if the VFO form is rejected (RFPOWER is typically global).
func (c *Client) Power(ctx context.Context) (float64, error) {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()

	conn, r, err := c.getConn(ctx)
	if err != nil {
		return 0, err
	}

	var raw string
	if c.vfoMode {
		raw, err = c.cmdDrain(r, conn, "l VFOA RFPOWER")
		if err != nil {
			// VFO-prefixed form rejected — fall back to plain form.
			applog.Debug("hamlib: power VFO form failed, retrying plain", "error", err)
			raw, err = c.cmdDrain(r, conn, "l RFPOWER")
		}
	} else {
		raw, err = c.cmdDrain(r, conn, "l RFPOWER")
	}
	if err != nil {
		c.dropConn()
		return 0, fmt.Errorf("hamlib: power: %w", err)
	}
	norm := parseFloat(strings.TrimSpace(raw))
	pwr := norm * c.maxPower
	c.discardReader()
	applog.Debug("hamlib: power", "norm", norm, "max", c.maxPower, "watts", pwr)
	return pwr, nil
}

// SetFrequency sets the rig frequency.  Uses the long-form \set_freq
// command which forces line-mode parsing.  In VFO mode the VFO argument
// goes after the command name: \set_freq VFOA <freq>.
func (c *Client) SetFrequency(ctx context.Context, freqHz int64) error {
	if c.vfoMode {
		return c.setCmd(ctx, fmt.Sprintf(`\set_freq VFOA %d`, freqHz))
	}
	return c.setCmd(ctx, fmt.Sprintf(`\set_freq %d`, freqHz))
}

// SetMode sets the rig mode.  \set_mode requires a mode token and a
// passband (Hz).  In VFO mode a VFO argument is required first.
func (c *Client) SetMode(ctx context.Context, mode string) error {
	if c.vfoMode {
		return c.setCmd(ctx, fmt.Sprintf(`\set_mode VFOA %s %d`, hamlibMode(mode), defaultPassband(mode)))
	}
	return c.setCmd(ctx, fmt.Sprintf(`\set_mode %s %d`, hamlibMode(mode), defaultPassband(mode)))
}

// setCmd opens a temporary connection, sends a set command, and closes.
// Multi-word commands (F 14.25, M PKTUSB) only work on clean connections
// where fgets() parses the full line before character-mode kicks in.
func (c *Client) setCmd(ctx context.Context, cmd string) error {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx2, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("hamlib dial: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))
	fmt.Fprintf(conn, "%s\r\n", cmd)
	r := bufio.NewReader(conn)
	resp, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("hamlib read: %w", err)
	}
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "RPRT ") && !strings.HasPrefix(resp, "RPRT 0") {
		return fmt.Errorf("hamlib error: %s", resp)
	}
	return nil
}

// GetModes returns the modes supported by the rig, fetched once from rigctld
// via \dump_caps on first call.  Falls back to a sensible default list if the
// rig query fails (e.g. rigctld not reachable).
func (c *Client) GetModes(ctx context.Context) ([]string, error) {
	c.modesOnce.Do(func() {
		modes, err := c.fetchRigModes(ctx)
		if err != nil || len(modes) == 0 {
			applog.Debug("hamlib: dump_caps modes failed, using defaults", "err", err)
			c.modesCache = []string{"USB", "LSB", "CW", "CW-L", "CW-U", "RTTY", "AM", "FM", "DATA-U", "DATA-L"}
			return
		}
		c.modesCache = modes
		applog.Info("hamlib: modes from rig", "count", len(modes), "modes", modes)
	})
	return c.modesCache, nil
}

// fetchRigModes opens a temporary connection, sends \dump_caps, and extracts
// all unique mode tokens from "Mode list:" lines in the capabilities dump.
func (c *Client) fetchRigModes(ctx context.Context) ([]string, error) {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx2, "tcp", c.addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	fmt.Fprintf(conn, "\\dump_caps\r\n")

	seen := make(map[string]bool)
	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "RPRT 0" {
			break // end of dump
		}
		// Lines look like: "           Mode list: AM CW USB LSB ..."
		const prefix = "Mode list:"
		if idx := strings.Index(line, prefix); idx >= 0 {
			rest := strings.TrimSpace(line[idx+len(prefix):])
			for _, tok := range strings.Fields(rest) {
				seen[tok] = true
			}
		}
	}

	if len(seen) == 0 {
		return nil, fmt.Errorf("no modes found in dump_caps")
	}

	modes := make([]string, 0, len(seen))
	for m := range seen {
		modes = append(modes, m)
	}
	return modes, nil
}

// GetName returns the rig model name.
func (c *Client) GetName(ctx context.Context) (string, error) {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()

	conn, r, err := c.getConn(ctx)
	if err != nil {
		return "", err
	}
	name, err := c.cmd(r, conn, "_")
	if err != nil {
		c.dropConn()
	}
	c.discardReader()
	return name, err
}

func (c *Client) cmd(r *bufio.Reader, conn net.Conn, cmd string) (string, error) {
	if c.timeout > 0 {
		conn.SetDeadline(time.Now().Add(c.timeout))
	}
	if _, err := fmt.Fprintf(conn, "%s\r\n", cmd); err != nil {
		return "", err
	}
	resp, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("hamlib read: %w", err)
	}
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "RPRT ") && !strings.HasPrefix(resp, "RPRT 0") {
		return "", fmt.Errorf("hamlib error: %s", resp)
	}

	// Drain the \r\n repeat and any multi-line extras (e.g. 's' with
	// --vfo returns split+VFO name) so nothing leaks into the next read.
	// Short deadline — the repeat arrives immediately on the same conn.
	if c.timeout > 0 {
		conn.SetDeadline(time.Now().Add(60 * time.Millisecond))
	}
	for {
		if _, err := r.ReadString('\n'); err != nil {
			break
		}
	}
	return resp, nil
}

// cmdDrain is an alias for cmd — since cmd() itself now drains the
// character-mode repeat and extras after every read, no additional
// draining is needed for space-containing commands.
func (c *Client) cmdDrain(r *bufio.Reader, conn net.Conn, cmd string) (string, error) {
	return c.cmd(r, conn, cmd)
}

// hamlibMode converts a CQOps mode string to a hamlib mode token.
func hamlibMode(mode string) string {
	switch strings.ToUpper(mode) {
	case "USB", "LSB", "AM", "FM", "CW", "RTTY":
		return mode
	case "CW-L", "CWL":
		return "CW"
	case "CW-U", "CWU":
		return "CW"
	case "DATA-U", "DATAU":
		return "PKTUSB"
	case "DATA-L", "DATAL":
		return "PKTLSB"
	case "FT8", "FT4":
		return "PKTUSB"
	default:
		return mode
	}
}

// defaultPassband returns a sensible passband width (Hz) for \set_mode.
// Most rigs ignore the passband and apply their own per-mode default, but
// hamlib's line-mode parser requires the argument.
func defaultPassband(mode string) int {
	switch strings.ToUpper(mode) {
	case "AM":
		return 6000
	case "FM":
		return 12000
	case "CW", "CW-L", "CWL", "CW-U", "CWU", "RTTY":
		return 500
	default: // USB, LSB, DATA-U, DATA-L, PKTUSB, PKTLSB, digital
		return 2400
	}
}

func parseFloat(s string) float64 {
	var f float64
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &f); err != nil {
		applog.Debug("hamlib: parseFloat failed", "input", s, "error", err)
	}
	return f
}
