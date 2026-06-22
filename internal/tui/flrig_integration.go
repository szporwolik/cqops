package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/rig/flrig"
)

// =============================================================================
// flrig integration — rig frequency/mode polling and connection management.
// =============================================================================

// refreshFlrigClient reinitializes the flrig HTTP client from current config.
// Clears cached mode table so it will be re-fetched on first successful status.
func (m *Model) refreshFlrigClient() {
	if m.App == nil || m.App.Logbook == nil {
		return
	}
	rigName := m.App.Logbook.Station.RigName
	// If no rig is active, pick the first available rig.
	if rigName == "" {
		for _, id := range config.SortedRigIDs(m.App.Config) {
			rigName = id
			break
		}
	}
	if rp, ok := m.App.Config.Rigs[rigName]; ok && rp.RadioBackend == "flrig" {
		host, port := rp.FlrigHost, rp.FlrigPort
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "12345"
		}
		url := "http://" + host + ":" + port
		applog.InfoDetail("flrig: connecting", fmt.Sprintf("rig=%s host=%s port=%s url=%s", rigName, host, port, url))
		m.rig.client = flrig.New(url, flrigDefaultTimeout)
		// Trigger an immediate poll so the user sees flrig connect within ~1s
		// instead of waiting up to 5 seconds for the next poll cycle.
		m.rig.skipTicks = 4 // one tick before pollInterval=5 triggers
	} else {
		if !ok {
			applog.Debug("flrig: rig not found in config", "rigName", rigName)
		} else {
			applog.Debug("flrig: disabled for rig", "rigName", rigName)
		}
		m.rig.client = nil
	}
	// Clear cached modes and name — a new client means the rig (or flrig instance)
	// may have changed its mode table or model.
	m.rig.modes = nil
	m.rig.name = ""
}

type flrigResultMsg struct {
	connected bool
	freq      float64
	freqRx    float64 // VFO B frequency (0 if not available)
	split     bool    // true when radio is in split mode
	mode      string
	band      string
	power     float64
	err       string
}

// flrigStatusCmd returns a tea.Cmd that fetches current rig status from flrig.
func (m *Model) flrigStatusCmd() tea.Cmd {
	if m.rig.client == nil {
		return nil
	}
	client := m.rig.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), flrigStatusTimeout)
		defer cancel()
		s, err := client.Status(ctx)
		if err != nil {
			return flrigResultMsg{err: err.Error()}
		}
		return flrigResultMsg{connected: s.Connected, freq: s.FrequencyMHz, freqRx: s.FrequencyRxMHz, split: s.Split, mode: s.Mode, band: s.Band, power: s.Power}
	}
}

// pollFlrig periodically polls flrig for rig status every 5 seconds.
func (m *Model) pollFlrig() tea.Cmd {
	const pollInterval = 5
	m.rig.skipTicks++
	if m.rig.skipTicks < pollInterval {
		return nil
	}
	m.rig.skipTicks = 0
	if m.rig.client == nil {
		m.rig.connected = false
		// Auto-reconnect: if flrig is enabled in config but the client
		// was never created (e.g. flrig started after CQOps), try now.
		// Limit to once every 30 ticks (30 seconds) to avoid log spam.
		if m.tickCount%30 == 0 {
			m.refreshFlrigClient()
		}
		return nil
	}
	if m.rig.polling {
		return nil
	}
	m.rig.polling = true
	m.rig.blink = !m.rig.blink
	return m.flrigStatusCmd()
}

// applyFlrigResult applies a flrig status result to the model state and QSO form.
// Returns an optional tea.Cmd for async mode/name fetching (safe, not raw goroutines).
func (m *Model) applyFlrigResult(r flrigResultMsg) tea.Cmd {
	m.rig.polling = false
	if r.err != "" || !r.connected {
		if m.rig.connected {
			m.rc.status = ""
		}
		m.rig.connected = false
		// Clear modes and name on disconnect so they are re-fetched when flrig comes back.
		m.rig.modes = nil
		m.rig.name = ""
		return nil
	}
	if !m.rig.connected {
		m.rc.status = ""
	}
	m.rig.connected = true
	// Fetch mode table and rig name whenever we (re)connect — rig may have changed.
	var cmds []tea.Cmd
	if len(m.rig.modes) == 0 {
		cmds = append(cmds, m.fetchFlrigModesCmd())
	}
	if m.rig.name == "" {
		cmds = append(cmds, m.fetchFlrigNameCmd())
	}
	m.rig.freq = r.freq
	// Split operation: when radio is in split mode (VFO A = RX, VFO B = TX),
	// set Freq to TX (VFO B) and Freq RX to RX (VFO A).
	split := r.split && r.freqRx > 0 && r.freqRx != r.freq && !m.wsjtx.online
	if !m.wsjtx.online {
		if split {
			m.rig.freq = r.freqRx // track TX frequency
			m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", r.freqRx))
			m.fields[fieldFreqRx].SetValue(fmt.Sprintf("%.6f", r.freq))
		} else {
			m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", r.freq))
		}
	}
	if r.mode != "" && !m.wsjtx.online {
		m.fields[fieldMode].SetValue(r.mode)
	}
	if r.band != "" {
		m.fields[fieldBand].SetValue(r.band)
	}
	if !m.wsjtx.online {
		m.autoFillSSBSubmode()
	}
	if r.power > 0 {
		m.fields[fieldTXPower].SetValue(fmt.Sprintf("%.0f", r.power))
	}
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// fmodesMsg carries the result of an async flrig mode fetch.
type fmodesMsg struct {
	modes []string
	err   string
}

// fetchFlrigModesCmd returns a tea.Cmd that fetches the flrig mode table.
// Retries once on transient failure, then gives up until the next reconnect.
func (m *Model) fetchFlrigModesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.rig.client == nil {
			return fmodesMsg{}
		}
		for attempt := 0; attempt < 2; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			modes, err := m.rig.client.GetModes(ctx)
			cancel()
			if err == nil {
				applog.Info("flrig: modes fetched", "count", len(modes), "modes", modes)
				return fmodesMsg{modes: modes}
			}
			applog.Warn("flrig: get_modes failed", "attempt", attempt+1, "error", err)
			time.Sleep(500 * time.Millisecond)
		}
		return fmodesMsg{err: "get_modes failed"}
	}
}

// fnameMsg carries the result of an async flrig name fetch.
type fnameMsg struct {
	name string
	err  string
}

// fetchFlrigNameCmd returns a tea.Cmd that fetches the flrig rig model name.
func (m *Model) fetchFlrigNameCmd() tea.Cmd {
	return func() tea.Msg {
		if m.rig.client == nil {
			return fnameMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		name, err := m.rig.client.GetName(ctx)
		if err != nil {
			applog.Warn("flrig: get_name failed", "error", err)
			return fnameMsg{err: err.Error()}
		}
		applog.Info("flrig: rig name fetched", "name", name)
		return fnameMsg{name: name}
	}
}
