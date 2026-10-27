package qso

import (
	"testing"
	"time"
)

// TestADIFTime pins the parser every "when did this happen" path uses. It has
// to accept exactly what the ADIF reader produces (YYYYMMDD + HHMMSS) and the
// loose forms older rows carry, and reject anything it cannot date — callers
// fall back to created_at in that case.
func TestADIFTime(t *testing.T) {
	cases := []struct {
		name       string
		date, time string
		want       string // RFC3339, empty = rejected
	}{
		{"full", "20260501", "123000", "2026-05-01T12:30:00Z"},
		{"separators", "2026-05-01", "12:30:00", "2026-05-01T12:30:00Z"},
		{"HHMM", "20260501", "1230", "2026-05-01T12:30:00Z"},
		{"no time", "20260501", "", "2026-05-01T00:00:00Z"},
		{"empty date", "", "120000", ""},
		{"short date", "2026050", "120000", ""},
		{"impossible day", "20260230", "120000", ""},
		{"impossible month", "20261301", "120000", ""},
		{"long time", "20260501", "1230000", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ADIFTime(c.date, c.time)
			if c.want == "" && ok {
				t.Fatalf("ADIFTime(%q,%q) = %s, want rejected", c.date, c.time, got)
			}
			if c.want == "" {
				return
			}
			if !ok {
				t.Fatalf("ADIFTime(%q,%q) rejected, want %s", c.date, c.time, c.want)
			}
			if got.Format(time.RFC3339) != c.want {
				t.Errorf("ADIFTime(%q,%q) = %s, want %s", c.date, c.time, got.Format(time.RFC3339), c.want)
			}
		})
	}
}
