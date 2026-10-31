package qso

import (
	"fmt"
	"strings"
)

func Validate(q *QSO) []string {
	return validateQSO(q, true)
}

// weakSignalModes are the modes in which the exchanged "report" is a
// machine-generated SNR value rather than something the operator types. A
// contact in one of them can legitimately arrive without a received report:
// WSJT-X writes <rst_rcvd:0> when the operator logs the QSO before the report
// is decoded, or the partner closes with RR73 instead. ADIF agrees — the
// suggested minimum for a QSO record is QSO_DATE, TIME_ON, BAND/FREQ, CALL and
// MODE, and RST_RCVD carries no default and is not required. Rejecting such a
// record dropped a real contact silently (2026-10-31: a second IZ3XAN QSO on
// 40 m never reached the log while WSJT-X's own log had it).
var weakSignalModes = map[string]bool{
	"FT8": true, "FT4": true, "FT2": true, "JTTY": true,
	"JT65": true, "JT9": true, "JT4": true, "JT6M": true, "JT44": true,
	"MFSK": true, "Q65": true, "FSK441": true, "MSK144": true, "ISCAT": true,
}

// reportOptionalFor reports whether a missing received report is acceptable for
// this contact's mode. Both the mode and the submode are checked because the
// caller may hand over an un-normalized record (MFSK + FT4).
func reportOptionalFor(mode, submode string) bool {
	m, sub := NormalizeMode(mode, submode)
	return weakSignalModes[m] || weakSignalModes[sub]
}

// ValidateForImport validates a contact that an integration delivered rather
// than an operator typing it in the form. It is Validate with one exception: a
// weak-signal contact may arrive without a received report, which is stored
// as-is (empty) instead of losing the whole contact. Everything else — the
// callsign, band, mode, date and time — stays as strict as the manual form,
// and the manual form itself keeps requiring both reports.
func ValidateForImport(q *QSO) []string {
	return validateQSO(q, !reportOptionalFor(q.Mode, q.Submode))
}

func validateQSO(q *QSO, requireReports bool) []string {
	var errs []string

	if strings.TrimSpace(q.Call) == "" {
		errs = append(errs, "call is required")
	} else if !IsValidCall(q.Call) {
		errs = append(errs, "Not a valid callsign")
	}

	// Locator is optional, but if provided it must be valid.
	if g := NormalizeLocator(q.GridSquare); g != "" && !IsValidLocator(g) {
		errs = append(errs, "Not a valid locator")
	}

	// Band or frequency is required. If band is empty, try to derive from freq.
	band := strings.TrimSpace(q.Band)
	if band == "" {
		if q.Freq > 0 {
			band = DeriveBand(q.Freq)
			if band == "" {
				errs = append(errs, "frequency does not match any band — set band manually")
			}
		} else {
			errs = append(errs, "band is required (enter frequency or set band)")
		}
	} else if !IsValidBand(band) {
		errs = append(errs, "unknown band: "+q.Band)
	}

	if strings.TrimSpace(q.Mode) == "" {
		errs = append(errs, "mode is required")
	} else if !IsValidMode(q.Mode) {
		errs = append(errs, "unknown mode: "+q.Mode)
	}

	if q.Submode != "" && !IsValidSubmode(q.Mode, q.Submode) {
		errs = append(errs, "invalid submode "+q.Submode+" for mode "+q.Mode)
	}

	if strings.TrimSpace(q.RSTSent) == "" {
		errs = append(errs, "rst_sent is required")
	}

	if requireReports && strings.TrimSpace(q.RSTRcvd) == "" {
		errs = append(errs, "rst_rcvd is required")
	}

	if strings.TrimSpace(q.QSODate) == "" {
		errs = append(errs, "qso_date is required")
	}

	if strings.TrimSpace(q.TimeOn) == "" {
		errs = append(errs, "time_on is required")
	}

	return errs
}

func ValidateForSave(q *QSO) error {
	errs := Validate(q)
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
