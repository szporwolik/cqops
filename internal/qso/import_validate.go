package qso

import (
	"fmt"
	"strings"
)

// ValidateImportRecord validates and normalizes a QSO record imported from an
// external source (Wavelog ADIF download). It applies cleanup in-place and
// returns an error if the record should be skipped entirely.
//
// Policy:
//   - Empty callsign → skip (caller should already guard this)
//   - Invalid callsign format → skip
//   - Empty mode → skip (mode is essential)
//   - Invalid mode (unknown mode) → skip
//   - Invalid grid → cleared to "" (non-fatal, don't reject the whole QSO)
//   - Invalid submode for mode → cleared to "" (non-fatal)
//   - Frequency is always accepted as-is; if band is empty and freq>0, band is
//     derived via DeriveBand; if band is also empty/missing after derivation,
//     the record is skipped.
func ValidateImportRecord(q *QSO) error {
	// --- Callsign ---
	q.Call = NormalizeCall(q.Call)
	if q.Call == "" {
		return fmt.Errorf("callsign is empty")
	}
	if !IsValidCall(q.Call) {
		return fmt.Errorf("invalid callsign: %q", q.Call)
	}

	// --- Mode ---
	q.Mode = strings.ToUpper(strings.TrimSpace(q.Mode))
	q.Submode = strings.ToUpper(strings.TrimSpace(q.Submode))
	if q.Mode == "" {
		return fmt.Errorf("mode is empty for %s", q.Call)
	}
	// Normalize mode/submode through the same path as local QSO edit.
	q.Mode, q.Submode = NormalizeMode(q.Mode, q.Submode)
	if !IsValidMode(q.Mode) {
		return fmt.Errorf("unknown mode: %q", q.Mode)
	}
	if q.Submode != "" && !IsValidSubmode(q.Mode, q.Submode) {
		// Invalid submode is non-fatal: just clear it.
		q.Submode = ""
	}

	// --- Grid / locator ---
	q.GridSquare = NormalizeLocator(q.GridSquare)
	if q.GridSquare != "" && !IsValidLocator(q.GridSquare) {
		// Invalid grid is non-fatal: clear it to avoid storing garbage.
		q.GridSquare = ""
	}

	// --- Band / frequency ---
	q.Band = NormalizeBand(q.Band)
	if q.Band == "" && q.Freq > 0 {
		q.Band = DeriveBand(q.Freq)
	}
	if q.Band == "" {
		return fmt.Errorf("band is empty and cannot be derived from frequency %.6f", q.Freq)
	}
	if !IsValidBand(q.Band) {
		return fmt.Errorf("unknown band: %q", q.Band)
	}

	// --- RST defaults (empty RST is allowed for import) ---
	q.RSTSent = strings.TrimSpace(q.RSTSent)
	q.RSTRcvd = strings.TrimSpace(q.RSTRcvd)

	return nil
}
