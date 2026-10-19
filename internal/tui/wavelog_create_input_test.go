package tui

import (
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// TestBuildCreateQSOInput_CarriesPowerAndReferences guards the mapping layer:
// every field the operator fills in must reach the JSON create payload, which
// is the upload path for manual saves, editor uploads and WSJT-X auto-logs.
// TX power in particular used to be dropped, so Wavelog kept its
// station-profile default (100 W) regardless of what the operator ran.
func TestBuildCreateQSOInput_CarriesPowerAndReferences(t *testing.T) {
	qs := &qso.QSO{
		Call:       "DJ7NT",
		Band:       "20m",
		Mode:       "MFSK",
		Submode:    "FT8",
		QSODate:    "20261013",
		TimeOn:     "203000",
		RSTSent:    "+05",
		RSTRcvd:    "-11",
		GridSquare: "JO31",
		Freq:       14.0745,
		TXPower:    "10",
		Operator:   "SP9SPM",
		SOTARef:    "SP/BZ-001",
		POTARef:    "SP-0001",
		WWFFRef:    "SPFF-0001",
		IOTA:       "EU-005",
		SIG:        "SOTA",
		SIGInfo:    "SP/BZ-001",
		CQZone:     "15",
		ITUZone:    "28",
	}

	in := buildCreateQSOInput(3, qs)

	if in.StationProfileID != 3 {
		t.Errorf("station profile = %d, want 3", in.StationProfileID)
	}
	if in.Mode != "FT8" {
		t.Errorf("mode = %q, want the canonical FT8 submode", in.Mode)
	}
	if in.QSODate != "2026-10-13" {
		t.Errorf("qso_date = %q, want ISO", in.QSODate)
	}
	if in.FreqHz != 14074500 {
		t.Errorf("freq = %d Hz, want 14074500", in.FreqHz)
	}

	checks := map[string][2]string{
		"TXPower":  {in.TXPower, "10"},
		"Operator": {in.Operator, "SP9SPM"},
		"SOTARef":  {in.SOTARef, "SP/BZ-001"},
		"POTARef":  {in.POTARef, "SP-0001"},
		"WWFFRef":  {in.WWFFRef, "SPFF-0001"},
		"IOTA":     {in.IOTA, "EU-005"},
		"SIG":      {in.SIG, "SOTA"},
		"SIGInfo":  {in.SIGInfo, "SP/BZ-001"},
		"CQZone":   {in.CQZone, "15"},
		"ITUZone":  {in.ITUZone, "28"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", name, c[0], c[1])
		}
	}
}
