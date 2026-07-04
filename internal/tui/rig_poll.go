package tui

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig/flrig"
	"github.com/szporwolik/cqops/internal/rig/hamlib"
	rothamlib "github.com/szporwolik/cqops/internal/rotor/hamlib"
)

// =============================================================================
// Rig control integration — frequency/mode polling and connection management.
// Supports flrig (HTTP) and hamlib rigctld (TCP) backends.
// =============================================================================

// refreshRigClient reinitializes the rig control client from current config.
// Clears cached mode table so it will be re-fetched on first successful status.
func (m *Model) refreshRigClient() {
	if m.App == nil || m.App.Logbook == nil {
		m.rig.connected = false
		return
	}
	// Close old persistent connection (hamlib keeps TCP open).
	if closer, ok := m.rig.client.(interface{ Close() error }); ok {
		closer.Close()
	}
	m.rig.client = nil
	m.rig.connected = false // reset until first successful poll
	rigName := m.App.Logbook.Station.RigName
	// If no rig is active, pick the first available rig.
	if rigName == "" {
		for _, id := range config.SortedRigIDs(m.App.Config) {
			rigName = id
			break
		}
	}
	rp, ok := m.App.Config.Rigs[rigName]
	if !ok {
		applog.Debug("rig: rig not found in config", "rigName", rigName)
		m.rig.client = nil
		m.rig.modes = nil
		m.rig.name = ""
		return
	}

	switch rp.RadioBackend {
	case "flrig":
		host, port := rp.FlrigHost, rp.FlrigPort
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "12345"
		}
		url := "http://" + host + ":" + port
		applog.InfoDetail("rig: flrig connecting", fmt.Sprintf("rig=%s host=%s port=%s url=%s", rigName, host, port, url))
		m.rig.client = flrig.New(url, rigDefaultTimeout)
		m.rig.skipTicks = 4 // one tick before pollInterval=5 triggers

	case "hamlib":
		host, port := rp.HamlibRadioHost, rp.HamlibRadioPort
		if host == "" {
			host = "127.0.0.1"
		}
		if port == "" {
			port = "4532"
		}
		applog.InfoDetail("rig: hamlib connecting", fmt.Sprintf("rig=%s host=%s port=%s", rigName, host, port))
		hc := hamlib.New(host, port, rigDefaultTimeout*time.Millisecond)
		if maxW, err := strconv.ParseFloat(rp.Power, 64); err == nil && maxW > 0 {
			hc.SetMaxPower(maxW)
		}
		m.rig.client = hc
		m.rig.skipTicks = 4

	default:
		applog.Debug("rig: backend not configured", "rigName", rigName)
		m.rig.client = nil
	}
	// Clear cached modes and name — a new client means the rig
	// may have changed its mode table or model.
	m.rig.modes = nil
	m.rig.name = ""
	m.rig.vfoWarned = false // allow fresh connect toast for new rig
}

type rigPollMsg struct {
	client    RigClient // the client that produced this result (nil = stale)
	connected bool
	freq      float64
	freqRx    float64 // VFO B frequency (0 if not available)
	split     bool    // true when radio is in split mode
	mode      string
	band      string
	power     float64
	err       string
}

// rigStatusCmd returns a tea.Cmd that fetches current rig status.
// Fast poll: frequency, mode, VFO/split (no power — that's on the slow cycle).
func (m *Model) rigStatusCmd() tea.Cmd {
	if m.rig.client == nil {
		return nil
	}
	client := m.rig.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), rigStatusTimeout)
		defer cancel()
		s, err := client.Status(ctx)
		if err != nil {
			return rigPollMsg{client: client, err: err.Error()}
		}
		return rigPollMsg{client: client, connected: s.Connected, freq: s.FrequencyMHz, freqRx: s.FrequencyRxMHz, split: s.Split, mode: s.Mode, band: s.Band, power: s.Power}
	}
}

// rigPowerCmd returns a tea.Cmd that fetches RF power (slow poll).
func (m *Model) rigPowerCmd() tea.Cmd {
	if m.rig.client == nil {
		return nil
	}
	client := m.rig.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), rigStatusTimeout)
		defer cancel()
		p, err := client.Power(ctx)
		if err != nil {
			return rigPowerMsg{client: client, err: err.Error()}
		}
		return rigPowerMsg{client: client, power: p}
	}
}

// rigPowerMsg carries the result of a slow-cycle RF power query.
type rigPowerMsg struct {
	client RigClient // the client that produced this result (nil = stale)
	power  float64
	err    string
}

// pollRig periodically polls the rig backend.
// Fast poll (every tick, ~1s):  frequency, mode, VFO, split.
// Slow poll (every 3 ticks, ~3s): RF power.
// When disconnected, back off to every 10 ticks (~10s) to avoid
// flooding rigctld with rapid connect/drop cycles.
func (m *Model) pollRig() tea.Cmd {
	pollInterval := 1 // fast-poll interval in ticks (~1 s)
	if !m.rig.connected {
		pollInterval = 10 // back off when disconnected (~10 s)
	}
	m.rig.skipTicks++
	if m.rig.skipTicks < pollInterval {
		return nil
	}
	m.rig.skipTicks = 0
	if m.rig.client == nil {
		m.rig.connected = false
		if m.tickCount%30 == 0 {
			m.refreshRigClient()
		}
		return nil
	}
	if m.rig.polling {
		return nil
	}
	m.rig.polling = true
	m.rig.blink = !m.rig.blink

	// Slow poll: add RF power every 3 fast-poll cycles.
	if m.rig.slowTick++; m.rig.slowTick >= 3 {
		m.rig.slowTick = 0
		return tea.Batch(m.rigStatusCmd(), m.rigPowerCmd())
	}
	return m.rigStatusCmd()
}

// applyRigPoll applies a rig poll result to the model state and QSO form.
// Returns an optional tea.Cmd for async mode/name fetching (safe, not raw goroutines).
func (m *Model) applyRigPoll(r rigPollMsg) tea.Cmd {
	m.rig.polling = false
	// Reject stale results from a previous client (e.g. after rig config
	// was changed or rig was disabled while a poll was in flight).
	if r.client != m.rig.client {
		return nil
	}
	if r.err != "" || !r.connected {
		if m.rig.connected {
			applog.Debug("rig: disconnected", "err", r.err)
			m.rc.status = ""
		}
		if r.err != "" && !m.rig.connected {
			applog.Debug("rig: connect failed", "err", r.err)
		}
		m.rig.connected = false
		// Clear modes and name on disconnect so they are re-fetched when rig comes back.
		m.rig.modes = nil
		m.rig.name = ""
		return nil
	}
	if !m.rig.connected {
		m.rc.status = ""
		// Connected — notify user once per session.
		if !m.rig.vfoWarned {
			m.rig.vfoWarned = true
			if _, ok := m.rig.client.(*hamlib.Client); ok {
				m.toasts.Success("Hamlib: connected")
			} else {
				m.toasts.Success("flrig: connected")
			}
		}
	}
	m.rig.connected = true
	// Fetch mode table and rig name whenever we (re)connect — rig may have changed.
	var cmds []tea.Cmd
	if len(m.rig.modes) == 0 {
		cmds = append(cmds, m.fetchRigModesCmd())
	}
	if m.rig.name == "" {
		cmds = append(cmds, m.fetchRigNameCmd())
	}
	// Split operation: when radio is in split mode (VFO A = RX, VFO B = TX),
	// main Freq field shows TX (VFO B), Freq RX field shows RX (VFO A).
	// Track both independently so tuning either VFO updates the form.
	split := r.split && r.freqRx > 0 && r.freqRx != r.freq && !m.wsjtx.online
	displayFreq := r.freq
	if split {
		displayFreq = r.freqRx
	}
	freqChanged := displayFreq != m.rig.freq
	freqRxChanged := split && r.freq != m.rig.freqRx

	m.rig.freq = displayFreq
	if split {
		m.rig.freqRx = r.freq
	} else {
		m.rig.freqRx = 0
	}

	// Detect split state transitions — force field updates when state flips
	// so frequencies don't appear frozen during A/B swaps or split on/off.
	splitBecameActive := split && !m.rig.wasSplit
	splitBecameInactive := !split && m.rig.wasSplit
	m.rig.wasSplit = split

	// Only update form fields when the value actually changed (don't overwrite user edits).
	if !m.wsjtx.online {
		if split {
			if freqChanged || splitBecameActive {
				m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", r.freqRx))
			}
			if freqRxChanged || splitBecameActive {
				m.fields[fieldFreqRx].SetValue(fmt.Sprintf("%.6f", r.freq))
			}
		} else {
			if freqChanged || splitBecameInactive {
				m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", r.freq))
			}
			if splitBecameInactive {
				m.fields[fieldFreqRx].SetValue("")
			}
		}
	}
	if r.mode != "" && !m.wsjtx.online {
		mapped := qso.NormalizeRigMode(r.mode)
		if mapped != m.fields[fieldMode].Value() {
			// Don't overwrite a real mode (e.g. FT8 from a DXC spot)
			// with a container mode like DATA-U / DATA-L / DATA-FM
			// that rigs use for generic digital operations.
			if !qso.IsValidMode(mapped) {
				cur := strings.TrimSpace(m.fields[fieldMode].Value())
				if cur != "" && qso.IsValidMode(cur) {
					// Keep the current valid mode, skip rig mode.
				} else {
					m.fields[fieldMode].SetValue(mapped)
				}
			} else {
				m.fields[fieldMode].SetValue(mapped)
			}
		}
	}
	if r.band != "" && r.band != m.fields[fieldBand].Value() {
		m.fields[fieldBand].SetValue(r.band)
	}
	if r.power > 0 {
		m.fields[fieldTXPower].SetValue(fmt.Sprintf("%.0f", r.power))
	}
	if !m.wsjtx.online {
		m.autoFillSSBSubmode()
	}
	// Push rig/mode/band changes to the dashboard immediately —
	// don't wait for the next tick.
	m.pushDashboardFast()
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// applyRigPower applies a slow-cycle RF power result to the QSO form.
func (m *Model) applyRigPower(r rigPowerMsg) {
	// Reject stale results from a previous client.
	if r.client != m.rig.client {
		return
	}
	if r.err != "" {
		return
	}
	if r.power > 0 {
		m.fields[fieldTXPower].SetValue(fmt.Sprintf("%.0f", r.power))
	}
}

// rigModesMsg carries the result of an async rig mode fetch.
type rigModesMsg struct {
	modes []string
	err   string
}

// fetchRigModesCmd returns a tea.Cmd that fetches the rig mode table.
// Retries once on transient failure, then gives up until the next reconnect.
func (m *Model) fetchRigModesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.rig.client == nil {
			return rigModesMsg{}
		}
		for attempt := 0; attempt < 2; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			modes, err := m.rig.client.GetModes(ctx)
			cancel()
			if err == nil {
				applog.Info("rig: modes fetched", "count", len(modes), "modes", modes)
				return rigModesMsg{modes: modes}
			}
			applog.Warn("rig: get_modes failed", "attempt", attempt+1, "error", err)
			time.Sleep(500 * time.Millisecond)
		}
		return rigModesMsg{err: "get_modes failed"}
	}
}

// rigNameMsg carries the result of an async rig name fetch.
type rigNameMsg struct {
	name string
	err  string
}

// fetchRigNameCmd returns a tea.Cmd that fetches the rig model name.
func (m *Model) fetchRigNameCmd() tea.Cmd {
	return func() tea.Msg {
		if m.rig.client == nil {
			return rigNameMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		name, err := m.rig.client.GetName(ctx)
		if err != nil {
			applog.Warn("rig: get_name failed", "error", err)
			return rigNameMsg{err: err.Error()}
		}
		applog.Info("rig: rig name fetched", "name", name)
		return rigNameMsg{name: name}
	}
}

// =============================================================================
// Rotor polling — azimuth/elevation from hamlib rotctld.
// =============================================================================

// refreshRotorClient initializes or tears down the rotor client from config.
func (m *Model) refreshRotorClient() {
	if m.App == nil || m.App.Logbook == nil {
		if closer, ok := m.rotor.client.(interface{ Close() error }); ok {
			closer.Close()
		}
		m.rotor.client = nil
		m.rotor.connected = false
		m.rotor.name = ""
		m.rotor.targetAz = 0
		m.rotor.targetEl = 0
		return
	}
	// Close old connection.
	if closer, ok := m.rotor.client.(interface{ Close() error }); ok {
		closer.Close()
	}
	m.rotor.client = nil
	m.rotor.connected = false
	m.rotor.name = ""
	m.rotor.targetAz = 0
	m.rotor.targetEl = 0

	enabled, host, port := m.App.Logbook.Station.RigRotor(m.App.Config.Rigs)
	if !enabled {
		applog.Debug("rotor: disabled")
		return
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "4533"
	}
	applog.InfoDetail("rotor: hamlib connecting", fmt.Sprintf("host=%s port=%s", host, port))
	c := rothamlib.New(host, port, rigDefaultTimeout*time.Millisecond)
	m.rotor.client = c
}

// rotorPollMsg carries the result of a rotor poll.
type rotorPollMsg struct {
	client    RotorClient // the client that produced this result (nil = stale)
	connected bool
	azimuth   float64
	elevation float64
	err       string
}

// rotorStatusCmd returns a tea.Cmd that queries current rotor position.
func (m *Model) rotorStatusCmd() tea.Cmd {
	if m.rotor.client == nil {
		return nil
	}
	client := m.rotor.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), rigStatusTimeout)
		defer cancel()
		s, err := client.Status(ctx)
		if err != nil {
			return rotorPollMsg{client: client, err: err.Error()}
		}
		return rotorPollMsg{client: client, connected: s.Connected, azimuth: s.Azimuth, elevation: s.Elevation}
	}
}

// pollRotor returns a tea.Cmd that queries rotor position every 2s.
func (m *Model) pollRotor() tea.Cmd {
	if m.rotor.client == nil {
		return nil
	}
	// Slow poll (~2s) — rotor moves slowly.
	if m.tickCount%2 != 0 {
		return nil
	}
	return m.rotorStatusCmd()
}

// applyRotorPoll applies a rotor poll result to the model.
func (m *Model) applyRotorPoll(r rotorPollMsg) tea.Cmd {
	if r.client != m.rotor.client {
		return nil
	}
	if r.err != "" || !r.connected {
		if m.rotor.connected {
			applog.Debug("rotor: disconnected", "err", r.err)
			m.toasts.Warn("Rotator: disconnected")
			m.rotor.name = ""
		}
		m.rotor.connected = false
		return nil
	}
	var cmds []tea.Cmd
	if !m.rotor.connected {
		applog.Info("rotor: connected", "az", r.azimuth, "el", r.elevation)
		m.toasts.Success("Rotator: connected")
		if m.rotor.name == "" {
			cmds = append(cmds, m.fetchRotorNameCmd())
		}
	}
	m.rotor.connected = true
	m.rotor.azimuth = math.Round(r.azimuth)
	m.rotor.elevation = math.Round(r.elevation)
	if m.rotor.targetAz != 0 && absDiff(m.rotor.azimuth, m.rotor.targetAz) < 1 &&
		absDiff(m.rotor.elevation, m.rotor.targetEl) < 1 {
		m.rotor.targetAz = 0
		m.rotor.targetEl = 0
	}
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// fetchRotorNameCmd returns a tea.Cmd that fetches the rotor model name.
func (m *Model) fetchRotorNameCmd() tea.Cmd {
	if m.rotor.client == nil {
		return nil
	}
	client := m.rotor.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		name, err := client.GetName(ctx)
		if err != nil {
			applog.Debug("rotor: get_name failed", "error", err)
			return rotorNameMsg{err: err.Error()}
		}
		applog.Info("rotor: name fetched", "name", name)
		return rotorNameMsg{name: name}
	}
}

// rotorNameMsg carries the result of an async rotor name fetch.
type rotorNameMsg struct {
	name string
	err  string
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

// shutdownConnections gracefully closes rig and rotor TCP connections
// before the process exits.  Called from the quit dialog path.
func (m *Model) shutdownConnections() {
	if closer, ok := m.rig.client.(interface{ Close() error }); ok {
		applog.Debug("rig: closing connection on shutdown")
		closer.Close()
	}
	if closer, ok := m.rotor.client.(interface{ Close() error }); ok {
		applog.Debug("rotor: closing connection on shutdown")
		closer.Close()
	}
	if m.dxc.client != nil {
		applog.Debug("dxc: stopping client on shutdown")
		m.dxc.client.Stop()
	}
}
