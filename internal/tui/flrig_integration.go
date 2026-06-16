package tui

import (
	"context"
	"fmt"

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
		m.flrigClient = flrig.New(url, flrigDefaultTimeout)
	} else {
		if !ok {
			applog.Debug("flrig: rig not found in config", "rigName", rigName)
		} else {
			applog.Debug("flrig: disabled for rig", "rigName", rigName)
		}
		m.flrigClient = nil
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
	if m.flrigClient == nil {
		return nil
	}
	client := m.flrigClient
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
	m.rigSkipTicks++
	if m.rigSkipTicks < pollInterval {
		return nil
	}
	m.rigSkipTicks = 0
	if m.flrigClient == nil {
		m.rigConnected = false
		return nil
	}
	if m.rigPolling {
		return nil
	}
	m.rigPolling = true
	m.rigBlink = !m.rigBlink
	return m.flrigStatusCmd()
}

// applyFlrigResult applies a flrig status result to the model state and QSO form.
func (m *Model) applyFlrigResult(r flrigResultMsg) {
	m.rigPolling = false
	if r.err != "" || !r.connected {
		m.rigConnected = false
		m.cachedStatus = ""
		return
	}
	m.rigConnected = true
	m.rigFreq = r.freq
	m.cachedStatus = ""
	if !m.wsjtxOnline {
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", r.freq))
	}
	if r.mode != "" && !m.wsjtxOnline {
		m.fields[fieldMode].SetValue(r.mode)
	}
	if r.band != "" {
		m.fields[fieldBand].SetValue(r.band)
	}
	if !m.wsjtxOnline {
		m.autoFillSSBSubmode()
	}
	if r.power > 0 {
		m.fields[fieldTXPower].SetValue(fmt.Sprintf("%.0f", r.power))
	}
}
