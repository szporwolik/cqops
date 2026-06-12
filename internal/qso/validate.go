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

	if strings.TrimSpace(q.Mode) == "" {
		errs = append(errs, "mode is required")
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
