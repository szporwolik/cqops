package qso

import (
	"fmt"
	"strings"
)

func Validate(q *QSO) []string {
	var errs []string

	if strings.TrimSpace(q.Call) == "" {
		errs = append(errs, "call is required")
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

	if strings.TrimSpace(q.RSTRcvd) == "" {
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
