package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
)

// dxcTuneCmd returns a tea.Cmd that tunes flrig to the highlighted spot's
// frequency and mode. Cancels any previous tune command still in flight.
func (m *Model) dxcTuneCmd() tea.Cmd {
	if !m.rig.connected || m.wsjtx.online || m.rig.client == nil {
		applog.Debug("DXC: tune skipped",
			"rigConnected", m.rig.connected,
			"wsjtxOnline", m.wsjtx.online,
			"hasClient", m.rig.client != nil,
		)
		return nil
	}
	// Cancel any previous tune command still in flight.
	if m.dxc.tuneCancel != nil {
		m.dxc.tuneCancel()
		m.dxc.tuneCancel = nil
	}
	spot, ok := m.dxcSpotAtCursor()
	if !ok {
		return nil
	}
	freqHz := int64(spot.Frequency * 1000)
	freqMHz := float64(freqHz) / 1_000_000
	call := spot.DXCall
	flrigModeIdx := m.rig.modeIndex(spotModeToFlrigMode(spot.Mode))
	flrigModeName := spot.Mode
	client := m.rig.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		m.dxc.tuneCancel = cancel
		defer func() {
			cancel()
			m.dxc.tuneCancel = nil
		}()

		if err := client.SetFrequency(ctx, freqHz); err != nil {
			if ctx.Err() != nil {
				return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("cancelled")}
			}
			applog.Warn("DXC: tune rig freq failed",
				"call", call, "freq_mhz", fmt.Sprintf("%.5f", freqMHz), "error", err,
			)
			return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("freq: %w", err)}
		}

		// Wait for rig to settle, then verify.
		time.Sleep(400 * time.Millisecond)
		if ctx.Err() != nil {
			return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("cancelled")}
		}
		status, err := client.Status(ctx)
		var verifyMsg string
		if err == nil && status.Connected {
			actualMHz := status.FrequencyMHz
			diff := actualMHz - freqMHz
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.001 {
				verifyMsg = fmt.Sprintf(" (rig at %.5f MHz)", actualMHz)
				applog.Warn("DXC: tune freq mismatch",
					"call", call,
					"requested_mhz", fmt.Sprintf("%.5f", freqMHz),
					"actual_mhz", fmt.Sprintf("%.5f", actualMHz),
				)
			}
		}

		if flrigModeIdx >= 0 {
			if ctx.Err() != nil {
				return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("cancelled")}
			}
			if err := client.SetMode(ctx, flrigModeIdx); err != nil {
				if ctx.Err() != nil {
					return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("cancelled")}
				}
				applog.Warn("DXC: tune rig mode failed",
					"call", call, "mode", flrigModeName, "idx", flrigModeIdx, "error", err,
				)
				return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("mode: %w", err)}
			}
			// Verify mode was applied.
			time.Sleep(200 * time.Millisecond)
			status2, err2 := client.Status(ctx)
			if err2 == nil && status2.Connected && status2.RawMode != "" {
				if !strings.EqualFold(status2.RawMode, flrigModeName) &&
					!strings.HasPrefix(strings.ToUpper(status2.RawMode), strings.ToUpper(flrigModeName)) {
					applog.Warn("DXC: tune mode mismatch",
						"call", call,
						"sent", flrigModeName,
						"rig_reports", status2.RawMode,
					)
				}
			}
			_ = err2
		}

		return dxcTuneResultMsg{
			call:    call,
			freqMHz: freqMHz,
			mode:    flrigModeName,
			verify:  verifyMsg,
		}
	}
}

// spotModeToFlrigMode maps a spot mode string to a canonical flrig mode name.
// Handles WSJT-X modes (FT8, FT4, JT65, JT9, MSK144) and traditional modes.
func spotModeToFlrigMode(spotMode string) string {
	switch strings.ToUpper(strings.TrimSpace(spotMode)) {
	// WSJT-X modes are data — use USB-D / DATA-U.
	case "FT8", "FT4", "JT65", "JT9", "JT65A", "JT65B", "JT65C",
		"JT9-1", "JT9-2", "JT9-5", "JT9-10", "JT9-30",
		"JT9A", "JT9B", "JT9C", "JT9D", "JT9E", "JT9F", "JT9G", "JT9H",
		"MSK144", "Q65", "FST4", "FST4W", "JS8", "FSQCALL", "WSPR":
		return "DATA-U"

	// Common digital modes — use USB-D.
	case "RTTY", "RTTYM", "PSK", "PSK31", "PSK63", "PSK125", "PSK250",
		"PSK500", "PSK1000", "MFSK", "MFSK4", "MFSK8", "MFSK16",
		"MFSK32", "MFSK64", "MFSK128", "OLIVIA", "CONTESTI", "DOMINO",
		"THOR", "HELL", "FSK", "ISCAT", "JTMS", "FSK441", "JT6M",
		"JT44", "QRA64", "SIM31", "T10", "V4", "ROS",
		"THRB", "PAX", "PAX2", "PAC", "PAC2", "PAC3", "PAC4",
		"MT63", "OFDM", "OPERA", "Q15":
		return "DATA-U"

	// CW variants — use CW-L (lower sideband CW).
	case "CW", "CW-L", "CWL", "CW-R":
		return "CW"

	case "CW-U", "CWU":
		return "CW-U"

	// Phone/voice modes.
	case "USB":
		return "USB"
	case "LSB":
		return "LSB"
	case "AM":
		return "AM"
	case "FM":
		return "FM"
	case "DIGITALVOICE", "C4FM", "DSTAR", "DMR", "FREEDV", "M17":
		return "FM"

	default:
		return "DATA-U"
	}
}
