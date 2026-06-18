package tui

import (
	"context"
	"fmt"
	"strings"
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
	if rp, ok := m.App.Config.Rigs[rigName]; ok && rp.FlrigEnabled {
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
	} else {
		if !ok {
			applog.Debug("flrig: rig not found in config", "rigName", rigName)
		} else {
			applog.Debug("flrig: disabled for rig", "rigName", rigName)
		}
		m.rig.client = nil
	}
}

type flrigResultMsg struct {
	connected bool
	freq      float64
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
		return flrigResultMsg{connected: s.Connected, freq: s.FrequencyMHz, mode: s.Mode, band: s.Band, power: s.Power}
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
func (m *Model) applyFlrigResult(r flrigResultMsg) {
	m.rig.polling = false
	if r.err != "" || !r.connected {
		if m.rig.connected {
			m.rc.status = ""
		}
		m.rig.connected = false
		return
	}
	if !m.rig.connected {
		m.rc.status = ""
	}
	m.rig.connected = true
	// Fetch mode table on first successful connection.
	if len(m.rig.modes) == 0 {
		go m.fetchFlrigModes()
	}
	m.rig.freq = r.freq
	if !m.wsjtx.online {
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", r.freq))
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
}

// fetchFlrigModes queries flrig for the available mode table and stores it.
func (m *Model) fetchFlrigModes() {
	if m.rig.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	modes, err := m.rig.client.GetModes(ctx)
	if err != nil {
		applog.Warn("flrig: get_modes failed", "error", err)
		return
	}
	m.rig.modes = modes
	applog.Info("flrig: modes fetched", "count", len(modes), "modes", modes)
}

// modeIndex returns the index of the desired mode in flrig's mode table.
// For CW, prefers CW-L/CWL (lower sideband, standard for HF) over CW-U/CWU.
func (r *rigState) modeIndex(want string) int {
	want = strings.ToUpper(want)
	flrigModes := r.modes

	// For CW: prefer lower-sideband CW, which is the HF standard.
	if want == "CW" {
		// 1. Exact match "CW-L" or "CWL" (lower sideband).
		for i, m := range flrigModes {
			u := strings.ToUpper(m)
			if u == "CW-L" || u == "CWL" {
				return i
			}
		}
		// 2. Exact match "CW".
		for i, m := range flrigModes {
			if strings.EqualFold(m, "CW") {
				return i
			}
		}
		// 3. Prefix "CW-" (matches "CW-L", "CW-U").
		for i, m := range flrigModes {
			if strings.HasPrefix(strings.ToUpper(m), "CW-") {
				return i
			}
		}
		// 4. Starts with "CW" and length 3 (matches "CWL", "CWU").
		for i, m := range flrigModes {
			u := strings.ToUpper(m)
			if strings.HasPrefix(u, "CW") && len(u) == 3 {
				return i
			}
		}
		return -1
	}

	// Default: exact match first, then prefix match.
	for i, m := range flrigModes {
		if strings.EqualFold(m, want) {
			return i
		}
	}
	for i, m := range flrigModes {
		if strings.HasPrefix(strings.ToUpper(m), want) {
			return i
		}
	}
	return -1
}
