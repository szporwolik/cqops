package tui

import (
	"fmt"
	"strings"

	"github.com/szporwolik/cqops/internal/qso"
)

// qsoFieldHint returns a validation hint for the given QSO form field, or ""
// if the current value is valid. Empty values do not produce hints —
// emptiness is handled at save time by qso.ValidateForSave.
func (m *Model) qsoFieldHint(f field) string {
	raw := strings.TrimSpace(m.fields[f].Value())

	switch f {
	case fieldCall:
		if raw != "" && !qso.IsValidCall(raw) {
			return "Invalid callsign"
		}
	case fieldGrid:
		if raw != "" && !qso.IsValidLocator(raw) {
			return "Invalid locator"
		}
	case fieldFreq:
		if raw == "" {
			return ""
		}
		// Frequency must be parseable as a positive float.
		var freq float64
		if _, err := fmt.Sscanf(raw, "%f", &freq); err != nil || freq <= 0 {
			return "Invalid frequency"
		}
		// Check if it maps to a known band. Out-of-band frequencies are valid
		// (e.g. satellite, transverter) but the band field won't auto-derive.
	case fieldBand:
		if raw != "" && !qso.IsValidBand(raw) {
			// Also check normalized form.
			norm := qso.NormalizeBand(raw)
			if norm == "" || !qso.IsValidBand(norm) {
				return "Invalid band"
			}
		}
	case fieldMode:
		if raw == "" {
			return ""
		}
		if qso.IsValidMode(raw) {
			return ""
		}
		// Accept import-only modes (FT8, FT4, etc.) that will be
		// normalized to MFSK + submode before save.
		if normalized, _ := qso.NormalizeMode(raw, ""); normalized != raw && qso.IsValidMode(normalized) {
			return ""
		}
		return "Invalid mode"
	case fieldSubmode:
		if raw == "" {
			return ""
		}
		mode := strings.TrimSpace(m.fields[fieldMode].Value())
		if mode == "" {
			return "" // can't validate submode without mode
		}
		if !qso.IsValidSubmode(mode, raw) {
			return "Invalid submode"
		}
	}
	return ""
}
