package tui

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
)

// freqRegex matches a MHz frequency like "3.560" or "27.285" or "446.06875"
// within a BPL text line. Captures group 1 = frequency in MHz.
var freqRegex = regexp.MustCompile(`\b(\d{1,3}\.\d{3,5})\b`)

// rigMaxFreqHz returns the maximum supported frequency in Hz for the given
// rig model name. Returns 0 if the model is unknown (assume HF-only).
// Known models are matched case-insensitively by substring.
func rigMaxFreqHz(rigName string) int64 {
	name := strings.ToUpper(rigName)

	// HF-only rigs: 1.8–54 MHz (160m–6m).
	hfOnly := []string{
		"FT-891", "FT891", "FT-450", "FT450", "FT-950", "FT950",
		"FT-1200", "FT-2000", "FT-3000", "FTDX-1200", "FTDX1200",
		"FTDX-3000", "FTDX3000", "FTDX-10", "FTDX10", "FTDX-101",
		"FT-710", "FT710", "IC-7300", "IC7300", "IC-7610", "IC7610",
		"IC-7600", "IC7600", "IC-718", "IC718", "IC-78", "TS-590",
		"TS590", "TS-890", "TS890", "TS-990", "TS990", "K3", "K3S",
		"K4", "KX2", "KX3", "FT-817", "FT817", "FT-818", "FT818",
		"FT-857", "FT857", "FT-897", "FT897", "IC-703", "IC703",
		"IC-706", "IC706", "IC-7000", "IC7000",
	}
	for _, m := range hfOnly {
		if strings.Contains(name, m) {
			return 54_000_000 // 54 MHz (6m)
		}
	}

	// HF+VHF rigs: up to 148 MHz (2m).
	hfVhf := []string{
		"FT-991", "FT991", "FT-920", "FT920", "IC-7100", "IC7100",
		"IC-9100", "IC9100", "TS-2000", "TS2000", "FT-847", "FT847",
	}
	for _, m := range hfVhf {
		if strings.Contains(name, m) {
			return 148_000_000 // 148 MHz (2m)
		}
	}

	// HF+VHF+UHF rigs: up to 450 MHz (70cm).
	hfVhfUhf := []string{
		"FT-991A", "FT991A", "IC-705", "IC705", "IC-9700", "IC9700",
		"FT-736", "FT736", "TM-D710", "TM-V71",
	}
	for _, m := range hfVhfUhf {
		if strings.Contains(name, m) {
			return 450_000_000 // 450 MHz (70cm)
		}
	}

	// All-band DC-to-daylight rigs.
	allBand := []string{
		"IC-905", "IC905",
	}
	for _, m := range allBand {
		if strings.Contains(name, m) {
			return 10_000_000_000 // 10 GHz
		}
	}

	// Unknown rig — assume HF-only (safest default).
	return 54_000_000
}

// bplLineFreqMode extracts (frequency in Hz, flrig mode string) from a single BPL
// text line at the given cursor position. Returns 0, "" if the line is not tunable
// (e.g. header, band range, or non-frequency line).
func bplLineFreqMode(line string) (int64, string) {
	if line == "" {
		return 0, ""
	}

	// Strip leading ">" cursor marker if present.
	line = strings.TrimPrefix(line, ">")
	line = strings.TrimSpace(line)

	// Skip headers and informational lines.
	if strings.HasPrefix(line, "NOT A HAM BAND") ||
		strings.HasPrefix(line, "BROADCAST ONLY") ||
		strings.HasPrefix(line, "R3 APRS") ||
		strings.HasPrefix(line, "No ") ||
		strings.Contains(line, "check the specific") ||
		strings.Contains(line, "licence-free") {
		return 0, ""
	}

	// Skip band range headers like "160m  1.800–2.000" or "2m 144.000–146.000".
	// These contain a band name followed by a range with en-dash or regular dash.
	if strings.Contains(line, "\u2013") || strings.Contains(line, "–") {
		// Check if this is a range line (header), not a single frequency.
		// Band headers have format "Band From–To" at start of line.
		if idx := strings.Index(line, " "); idx > 0 {
			rest := line[idx:]
			if strings.Contains(rest, "\u2013") || strings.Contains(rest, "–") {
				return 0, ""
			}
		}
	}

	// Skip AVOID / beacon lines.
	if strings.Contains(line, "AVOID") || strings.Contains(line, "IBP beacon") {
		return 0, ""
	}

	// Extract frequency using regex.
	matches := freqRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return 0, ""
	}

	mhzStr := matches[1]
	mhz, err := strconv.ParseFloat(mhzStr, 64)
	if err != nil {
		return 0, ""
	}
	freqHz := int64(mhz * 1_000_000)

	// Determine mode from the line content.
	// Check for mode keywords in order of specificity.
	lineUpper := strings.ToUpper(line)
	mode := ""

	// CW modes.
	if strings.Contains(lineUpper, "CW QRP") || strings.Contains(lineUpper, "QRS SLOW CW") || strings.Contains(lineUpper, "CW COA") {
		mode = "CW"
	} else if strings.Contains(lineUpper, "SSB") || strings.Contains(lineUpper, "USB") || strings.Contains(lineUpper, "LSB") {
		mode = "USB"
	} else if strings.Contains(lineUpper, "CW/SSB") || strings.Contains(lineUpper, "SSB/CW") {
		mode = "USB"
	} else if strings.Contains(lineUpper, "FM") || strings.Contains(lineUpper, "DV CALLING") {
		mode = "FM"
	} else if strings.Contains(lineUpper, "DIG") || strings.Contains(lineUpper, "DATA") || strings.Contains(lineUpper, "APRS") || strings.Contains(lineUpper, "APR") || strings.Contains(lineUpper, "LRA") {
		mode = "DATA-U"
	} else if strings.Contains(lineUpper, "SSTV") || strings.Contains(lineUpper, "IMG") {
		mode = "USB"
	} else if strings.Contains(lineUpper, "RPT") {
		// Repeater — FM for output/repeater frequencies.
		mode = "FM"
	} else if strings.Contains(lineUpper, "SAT") {
		mode = "FM"
	} else if strings.Contains(lineUpper, "BCN") {
		mode = "CW"
	} else if strings.HasPrefix(lineUpper, "CH ") {
		// CB/PMR channels — use AM/FM/SSB based on channel tag.
		if strings.Contains(lineUpper, "SSB") {
			mode = "USB"
		} else if mhz >= 446 && mhz <= 447 {
			mode = "FM" // PMR446
		} else {
			mode = "AM" // CB defaults to AM
		}
	} else if strings.Contains(lineUpper, "LW") || strings.Contains(lineUpper, "MW") || strings.Contains(lineUpper, "SW") {
		// Broadcast — AM.
		mode = "AM"
	} else if strings.Contains(lineUpper, "CALL") || strings.Contains(lineUpper, "DX") {
		if strings.Contains(lineUpper, "FM") {
			mode = "FM"
		} else if strings.Contains(lineUpper, "CW") {
			mode = "CW"
		} else {
			mode = "USB" // default SSB for CALL/DX
		}
	} else if strings.Contains(lineUpper, "PHN") || strings.Contains(lineUpper, "PHONE") {
		mode = "USB"
	}

	if mode == "" {
		// Default to FM for channels, USB for ham bands.
		if mhz >= 26 && mhz <= 28 {
			mode = "AM" // CB
		} else if mhz >= 446 && mhz <= 447 {
			mode = "FM" // PMR446
		} else if mhz >= 462 && mhz <= 468 {
			mode = "FM" // FRS/GMRS
		} else if mhz >= 50 && mhz <= 54 {
			mode = "USB" // 6m
		} else {
			mode = "USB" // safe default for ham HF
		}
	}

	return freqHz, mode
}

// bplTuneCmd returns a tea.Cmd that tunes flrig to the frequency and mode
// of the currently highlighted BPL line. Returns nil if flrig is not connected
// or the line doesn't contain a tunable frequency.
func (m *Model) bplTuneCmd() tea.Cmd {
	if !m.rig.connected || m.wsjtx.online || m.rig.client == nil {
		return nil
	}

	// Cancel any previous tune command still in flight.
	if m.bpl.tuneCancel != nil {
		m.bpl.tuneCancel()
		m.bpl.tuneCancel = nil
	}

	// Get the current line at cursor position.
	region := 1
	if m.App != nil && m.App.Logbook != nil {
		r := m.App.Logbook.Station.IARURegion
		if r >= 1 && r <= 3 {
			region = r
		}
	}
	var lines []string
	switch m.bpl.tab {
	case bplTabHAM:
		lines = m.viewBPLHAM(region)
	case bplTabVHF:
		lines = m.viewBPLVHF(region)
	case bplTabCB:
		lines = m.viewBPLCB(region)
	case bplTabPMR:
		lines = m.viewBPLPMR(region)
	case bplTabBRC:
		lines = m.viewBPLBRC()
	}

	if m.bpl.cursor < 0 || m.bpl.cursor >= len(lines) {
		return nil
	}
	line := lines[m.bpl.cursor]
	freqHz, mode := bplLineFreqMode(line)
	if freqHz == 0 {
		return nil
	}

	freqMHz := float64(freqHz) / 1_000_000
	client := m.rig.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		m.bpl.tuneCancel = cancel
		defer func() {
			cancel()
			m.bpl.tuneCancel = nil
		}()

		// Reject frequencies beyond the rig's supported range.
		if maxFreq := rigMaxFreqHz(m.rig.name); freqHz > maxFreq {
			verify := fmt.Sprintf(" (rig max %.0f MHz)", float64(maxFreq)/1_000_000)
			applog.Warn("BPL: freq out of rig range",
				"requested_mhz", fmt.Sprintf("%.5f", freqMHz),
				"rig_max_mhz", fmt.Sprintf("%.0f", float64(maxFreq)/1_000_000),
				"rig", m.rig.name,
			)
			return bplTuneResultMsg{freqMHz: freqMHz, mode: mode, verify: verify}
		}

		if err := client.SetFrequency(ctx, freqHz); err != nil {
			if ctx.Err() != nil {
				return bplTuneResultMsg{freqMHz: freqMHz, mode: mode, err: fmt.Errorf("cancelled")}
			}
			applog.Warn("BPL: tune freq failed",
				"freq_mhz", fmt.Sprintf("%.5f", freqMHz), "error", err,
			)
			return bplTuneResultMsg{freqMHz: freqMHz, mode: mode, err: fmt.Errorf("rig did not respond — try again")}
		}

		// Wait for rig to settle, then verify.
		time.Sleep(400 * time.Millisecond)
		if ctx.Err() != nil {
			return bplTuneResultMsg{freqMHz: freqMHz, mode: mode, err: fmt.Errorf("cancelled")}
		}
		status, err := client.Status(ctx)
		var verifyMsg string
		if err != nil {
			applog.Debug("BPL: tune status check failed", "error", err)
		} else if !status.Connected {
			applog.Debug("BPL: tune status reports disconnected")
		} else {
			actualMHz := status.FrequencyMHz
			diff := actualMHz - freqMHz
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.001 {
				verifyMsg = fmt.Sprintf(" (rig at %.5f MHz)", actualMHz)
				applog.Warn("BPL: tune freq mismatch",
					"requested_mhz", fmt.Sprintf("%.5f", freqMHz),
					"actual_mhz", fmt.Sprintf("%.5f", actualMHz),
				)
			} else {
				applog.Debug("BPL: tune freq OK",
					"freq_mhz", fmt.Sprintf("%.5f", freqMHz),
				)
			}
		}

		// Fetch modes fresh (not from the stale rigState cache) and find the
		// best match index — same pattern as WaveLogGate's SetFreqMode.
		modes, modeErr := client.GetModes(ctx)
		if modeErr != nil {
			applog.Warn("BPL: tune get_modes failed", "error", modeErr)
		} else {
			// Update the cached modes so future calls benefit.
			m.rig.modes = modes
		}
		flrigModeName := findRigModeName(mode, modes)
		applog.Debug("BPL: tune mode lookup",
			"want", mode,
			"found", flrigModeName,
			"modes_count", len(modes),
		)

		if flrigModeName != "" {
			if ctx.Err() != nil {
				return bplTuneResultMsg{freqMHz: freqMHz, mode: mode, err: fmt.Errorf("cancelled")}
			}
			if err := client.SetMode(ctx, flrigModeName); err != nil {
				if ctx.Err() != nil {
					return bplTuneResultMsg{freqMHz: freqMHz, mode: mode, err: fmt.Errorf("cancelled")}
				}
				applog.Warn("BPL: tune mode failed",
					"mode", flrigModeName, "error", err,
				)
				return bplTuneResultMsg{freqMHz: freqMHz, mode: mode, err: fmt.Errorf("rig did not respond — try again")}
			}
			// Verify mode was applied.
			time.Sleep(200 * time.Millisecond)
			status2, err2 := client.Status(ctx)
			if err2 == nil && status2.Connected && status2.RawMode != "" {
				if !strings.EqualFold(status2.RawMode, flrigModeName) &&
					!strings.HasPrefix(strings.ToUpper(status2.RawMode), strings.ToUpper(flrigModeName)) {
					applog.Warn("BPL: tune mode mismatch",
						"sent", flrigModeName,
						"rig_reports", status2.RawMode,
					)
				}
			}
			_ = err2
		}

		return bplTuneResultMsg{
			freqMHz: freqMHz,
			mode:    mode,
			verify:  verifyMsg,
		}
	}
}
