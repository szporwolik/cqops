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
	if !m.rig.connected || m.wsjtx.tx || m.rig.client == nil {
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
	wantMode := spotModeToFlrigMode(spot.Mode)
	spotModeName := spot.Mode
	client := m.rig.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		m.dxc.tuneCancel = cancel
		defer func() {
			cancel()
			m.dxc.tuneCancel = nil
		}()

		// Reject frequencies beyond the rig's supported range.
		if maxFreq := rigMaxFreqHz(m.rig.name); freqHz > maxFreq {
			verify := fmt.Sprintf(" (rig max %.0f MHz)", float64(maxFreq)/1_000_000)
			applog.Warn("DXC: freq out of rig range",
				"call", call,
				"requested_mhz", fmt.Sprintf("%.5f", freqMHz),
				"rig_max_mhz", fmt.Sprintf("%.0f", float64(maxFreq)/1_000_000),
				"rig", m.rig.name,
			)
			return dxcTuneResultMsg{call: call, freqMHz: freqMHz, verify: verify}
		}

		if err := client.SetFrequency(ctx, freqHz); err != nil {
			if ctx.Err() != nil {
				return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("cancelled")}
			}
			applog.Warn("DXC: tune rig freq failed",
				"call", call, "freq_mhz", fmt.Sprintf("%.5f", freqMHz), "error", err,
			)
			return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("rig did not respond — try again")}
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

		// Fetch modes fresh (not from the stale rigState cache) and find the
		// best match index — same pattern as WaveLogGate's SetFreqMode.
		modes, modeErr := client.GetModes(ctx)
		if modeErr != nil {
			applog.Warn("DXC: tune get_modes failed", "error", modeErr)
		} else {
			// Update the cached modes so future calls benefit.
			m.rig.modes = modes
		}
		flrigModeName := findRigModeName(wantMode, modes)
		applog.Debug("DXC: tune mode lookup",
			"want", wantMode,
			"found", flrigModeName,
			"modes_count", len(modes),
		)

		if flrigModeName != "" {
			if ctx.Err() != nil {
				return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("cancelled")}
			}
			if err := client.SetMode(ctx, flrigModeName); err != nil {
				if ctx.Err() != nil {
					return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("cancelled")}
				}
				applog.Warn("DXC: tune rig mode failed",
					"call", call, "mode", flrigModeName, "error", err,
				)
				return dxcTuneResultMsg{call: call, freqMHz: freqMHz, err: fmt.Errorf("rig did not respond — try again")}
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
			if err2 != nil {
				applog.Debug("DXC: tune mode verify failed", "call", call, "error", err2)
			}
		}

		return dxcTuneResultMsg{
			call:    call,
			freqMHz: freqMHz,
			mode:    spotModeName,
			verify:  verifyMsg,
		}
	}
}

// findRigModeName returns the best matching flrig mode name for the given
// canonical mode. Uses fallback chains for CW, DATA, and RTTY to handle the
// wide variety of flrig mode names across different rig models.
// Returns "" if no match is found.
func findRigModeName(want string, modes []string) string {
	if len(modes) == 0 {
		return ""
	}
	upper := strings.ToUpper(want)

	// Try each candidate in priority order — first match wins.
	for _, c := range rigModeCandidates(upper) {
		for _, m := range modes {
			if strings.EqualFold(m, c) {
				return m
			}
		}
	}

	// Prefix match as last resort.
	for _, c := range rigModeCandidates(upper) {
		for _, m := range modes {
			if strings.HasPrefix(strings.ToUpper(m), c) {
				return m
			}
		}
	}

	return ""
}

// rigModeCandidates returns an ordered list of flrig mode names to try for
// a given canonical mode. The first match in the list wins. This handles the
// wide variety of mode names across Icom, Yaesu, Kenwood, and other rigs.
func rigModeCandidates(want string) []string {
	switch want {
	// ── Data / digital modes ──
	case "DATA-U":
		return []string{"DATA-U", "USB-D", "PKTUSB", "DIGU", "D-USB", "DATA"}
	case "DATA-L":
		return []string{"DATA-L", "LSB-D", "PKTLSB", "DIGL", "D-LSB"}
	case "DATA-FM":
		return []string{"DATA-FM", "PKTFM", "DIGFM"}

	// ── CW modes ──
	case "CW":
		return []string{"CW-L", "CWL", "CW", "CW-U", "CWU", "CWR", "CW-R"}
	case "CW-L":
		return []string{"CW-L", "CWL", "CW"}
	case "CW-U":
		return []string{"CW-U", "CWU", "CWR", "CW-R", "CW"}

	// ── RTTY modes ──
	case "RTTY":
		return []string{"RTTY", "RTTY-R", "RTTYR", "RTTY-U", "RTTY-L"}
	case "RTTY-R":
		return []string{"RTTY-R", "RTTYR", "RTTY"}

	// ── Phone modes ──
	case "USB":
		return []string{"USB"}
	case "LSB":
		return []string{"LSB"}
	case "AM":
		return []string{"AM", "AM-D"}
	case "FM":
		return []string{"FM", "FM-D", "WFM"}

	// ── PKT modes ──
	case "PKT-U":
		return []string{"PKT-U", "PKTUSB", "DATA-U", "USB-D", "DIGU"}
	case "PKT-L":
		return []string{"PKT-L", "PKTLSB", "DATA-L", "LSB-D", "DIGL"}
	case "PKT-FM":
		return []string{"PKT-FM", "PKTFM", "DATA-FM"}

	default:
		return []string{want}
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
