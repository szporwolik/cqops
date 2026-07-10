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
	vfoMode   bool    // true if rigctld accepts VFO-prefixed commands
	vfoProbed bool    // true after first probe attempt (pass or fail)

	powerVfoOK bool // true if "l VFOA RFPOWER" works on this backend
	vfoNameOK  bool // true if "v" VFO name query works on this backend
	modesOnce  sync.Once
	modesCache []string // populated on first GetModes call
}

// New returns a new hamlib client.
func New(host, port string, timeout time.Duration) *Client {
	return &Client{
		addr:       net.JoinHostPort(host, port),
		timeout:    timeout,
		maxPower:   100,
		powerVfoOK: true, // try VFO form first; cleared on first failure
		vfoNameOK:  true, // try "v" query first; cleared on first failure
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

	// Read current VFO name.  Some backends (Icom) don't support this
	// — treat as non-fatal so we can still poll freq/mode/split.
	// Once the query fails, stop retrying (like powerVfoOK for power).
	var vfo string
	if c.vfoNameOK {
		var err error
		vfo, err = c.cmd(r, conn, "v")
		if err != nil {
			applog.Debug("hamlib: vfo name query failed, skipping future attempts", "error", err)
			c.vfoNameOK = false
			vfo = ""
		} else {
			vfo = strings.TrimSpace(vfo)
		}
	}

	// Split status.  With --vfo, single-char 's' returns nothing — must
	// use VFO-prefixed 's VFOA'.  Without --vfo, plain 's' works.
	// Non-fatal: some backends (Xiegu) don't support this query.
	// Do NOT drop the connection on failure — if the connection is truly
	// broken the subsequent frequency query will detect it.
	var split bool
	if c.vfoMode {
		if raw, sErr := c.cmdDrain(r, conn, "s VFOA"); sErr == nil {
			split = strings.TrimSpace(raw) == "1"
		} else {
			applog.Debug("hamlib: split query failed, assuming off", "error", sErr)
		}
	} else {
		if raw, sErr := c.cmd(r, conn, "s"); sErr == nil {
			split = strings.TrimSpace(raw) == "1"
		} else {
			applog.Debug("hamlib: split query failed, assuming off", "error", sErr)
		}
	}

	// Main VFO frequency.  With --vfo, 'f VFOA' targets the VFO; without
	// --vfo, plain 'f' returns the active VFO frequency.  Fatal — without
	// frequency there is nothing useful to show.
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
	freq := parseFloat(freqHz)
	// A valid ham-band frequency is never <= 100 kHz.  Zero or tiny values
	// mean the response was stale data (RPRT, mode string, etc.) leaking
	// from a previous timed-out command.  Drop the connection so the next
	// poll starts with a clean TCP buffer.
	if freq <= 100000 {
		applog.Debug("hamlib: frequency returned invalid value, dropping connection", "raw", freqHz, "freq", freq)
		c.dropConn()
		return rig.RigStatus{}, fmt.Errorf("hamlib: frequency invalid: %q (%.0f Hz)", freqHz, freq)
	}
	freqMHz := freq / 1e6

	// Mode.  With --vfo, 'm' needs the VFO prefix; without, plain 'm'.
	// Non-fatal: same rationale as split.
	var mode string
	if c.vfoMode {
		if raw, mErr := c.cmdDrain(r, conn, "m VFOA"); mErr == nil {
			mode = strings.TrimSpace(raw)
		} else {
			applog.Debug("hamlib: mode query failed, assuming empty", "error", mErr)
		}
	} else {
		if raw, mErr := c.cmd(r, conn, "m"); mErr == nil {
			mode = strings.TrimSpace(raw)
		} else {
			applog.Debug("hamlib: mode query failed, assuming empty", "error", mErr)
		}
	}

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

// probeVfoMode detects whether rigctld is running with --vfo (VFO-aware
// command protocol).  With --vfo, plain single-char commands (f, m, s, l)
// return nothing; a VFO argument is required.  Without --vfo, plain
// commands work and VFO arguments are silently ignored.
//
// Strategy: try "f VFOA" first — on VFO-aware backends this returns a
// single frequency line.  On non-VFO backends the 'f' character returns
// the frequency but the ' VFOA' suffix triggers RPRT -1 on subsequent
// lines.  We detect this to avoid false positives.  Falls back to plain
// "f" when VFOA form is rejected.
// Fails silently; VFO mode stays false if the probe is inconclusive.
func (c *Client) probeVfoMode(ctx context.Context) {
	send := func(cmd string) (float64, bool) {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		var d net.Dialer
		conn, err := d.DialContext(probeCtx, "tcp", c.addr)
		if err != nil {
			return 0, false
		}
		defer conn.Close()

		conn.SetDeadline(time.Now().Add(2 * time.Second))
		fmt.Fprintf(conn, "%s\r\n", cmd)
		r := bufio.NewReader(conn)
		resp, err := r.ReadString('\n')
		if err != nil {
			return 0, false
		}
		resp = strings.TrimSpace(resp)
		if strings.HasPrefix(resp, "RPRT ") {
			return 0, false
		}
		freq := parseFloat(resp)
		if freq <= 0 {
			return 0, false
		}

		// On non-VFO backends "f VFOA" processes 'f' first (returns
		// freq), then the ' VFOA' suffix triggers RPRT -1 on later
		// lines.  Drain only the character-mode repeat line (60ms),
		// then peek at the next line: RPRT -1 means the suffix was
		// rejected → we are on a non-VFO backend.
		if cmd == "f VFOA" {
			// Drain at most one line of character-mode repeat.
			conn.SetDeadline(time.Now().Add(60 * time.Millisecond))
			r.ReadString('\n') // ignore result — it's the repeat or first RPRT
			// Now peek: if the next line is RPRT -, the VFOA suffix
			// was rejected character-by-character → non-VFO backend.
			conn.SetDeadline(time.Now().Add(150 * time.Millisecond))
			if extra, err := r.ReadString('\n'); err == nil {
				extra = strings.TrimSpace(extra)
				if strings.HasPrefix(extra, "RPRT -") {
					applog.Debug("hamlib: vfo probe: f VFOA suffix rejected by backend", "extra", extra)
					return 0, false
				}
			}
		}

		return freq, true
	}

	// Try "f VFOA" first — works instantly on VFO-aware backends.
	if freq, ok := send("f VFOA"); ok && freq > 0 {
		c.vfoMode = true
		applog.Debug("hamlib: vfo mode detected (f VFOA ok)", "freq", freq)
		return
	}

	// "f VFOA" failed — try plain "f".  Works on non-VFO backends.
	if freq, ok := send("f"); ok && freq > 0 {
		applog.Debug("hamlib: vfo mode not detected (plain f returned frequency)", "freq", freq)
		return
	}

	applog.Debug("hamlib: vfo mode probe inconclusive — assuming vfo off")
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
	if c.vfoMode && c.powerVfoOK {
		raw, err = c.cmdDrain(r, conn, "l VFOA RFPOWER")
		if err != nil {
			// VFO-prefixed form rejected — fall back to plain form
			// and remember so we skip it on future calls.
			applog.Debug("hamlib: power VFO form failed, retrying plain", "error", err)
			c.powerVfoOK = false
			raw, err = c.cmdDrain(r, conn, "l RFPOWER")
		}
	} else if c.vfoMode && !c.powerVfoOK {
		// VFO form already known to fail — use plain form directly.
		raw, err = c.cmdDrain(r, conn, "l RFPOWER")
	} else {
		raw, err = c.cmdDrain(r, conn, "l RFPOWER")
	}
	if err != nil {
		// Power is non-critical — don't drop the shared connection.
		// Many backends don't support l VFOA RFPOWER or even l RFPOWER
		// in VFO mode; the rig still works fine without power data.
		c.discardReader()
		applog.Debug("hamlib: power failed", "error", err)
		return 0, nil
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

	// Drain the character-mode repeat / multi-line extras (e.g. 's' with
	// --vfo returns split+VFO name) so nothing leaks into the next read.
	// Short deadline — the repeat arrives immediately on the same conn.
	// MUST drain BEFORE checking RPRT: an RPRT error line also gets
	// repeated, and skipping the drain poisons subsequent commands.
	if c.timeout > 0 {
		conn.SetDeadline(time.Now().Add(60 * time.Millisecond))
	}
	for {
		if _, err := r.ReadString('\n'); err != nil {
			break
		}
	}

	if strings.HasPrefix(resp, "RPRT ") && !strings.HasPrefix(resp, "RPRT 0") {
		return "", fmt.Errorf("hamlib error: %s", resp)
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
