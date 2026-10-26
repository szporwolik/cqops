package tui

import (
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// Contest data must reach the Wavelog payload builders — a contest contact
// uploaded as a plain QSO was the reported symptom ("STX/SRX never made it to
// Wavelog"), and only the ADIF bulk path carried it.
func TestBuildCreateQSOInputCarriesContestExchange(t *testing.T) {
	q := &qso.QSO{
		Call: "DL1ABC", Band: "40m", Mode: "SSB", Submode: "LSB",
		QSODate: "20261025", TimeOn: "193200", RSTSent: "59", RSTRcvd: "59",
		ContestID: "c1", ContestADIFID: "CQ-WW-SSB",
		ExchSent: "59 15", ExchRcvd: "59 16",
		STX: 15, SRX: 16, STXString: "15", SRXString: "16",
	}
	in := buildCreateQSOInput(1, q)

	if in.ContestID != "CQ-WW-SSB" {
		t.Errorf("ContestID = %q", in.ContestID)
	}
	if in.STX != 15 || in.SRX != 16 {
		t.Errorf("STX/SRX = %d/%d", in.STX, in.SRX)
	}
	if in.STXString != "15" || in.SRXString != "16" {
		t.Errorf("STXString/SRXString = %q/%q", in.STXString, in.SRXString)
	}
	if in.Submode != "LSB" {
		t.Errorf("Submode = %q, want the SSB sideband", in.Submode)
	}
	if in.Mode != "SSB" {
		t.Errorf("Mode = %q, want SSB", in.Mode)
	}
}

// A canonical digital mode keeps its submode out of the payload (FT8 is a
// mode, not an MFSK submode, on the wire).
func TestBuildCreateQSOInputDropsRedundantSubmode(t *testing.T) {
	q := &qso.QSO{Call: "DL1ABC", Band: "20m", Mode: "MFSK", Submode: "FT8", QSODate: "20261025", TimeOn: "193200"}
	in := buildCreateQSOInput(1, q)
	if in.Mode != "FT8" {
		t.Errorf("Mode = %q, want FT8", in.Mode)
	}
	if in.Submode != "" {
		t.Errorf("Submode = %q, want empty", in.Submode)
	}
}

func TestBuildUpdateInputCarriesContestExchange(t *testing.T) {
	q := &qso.QSO{
		Call: "DL1ABC", Band: "40m", Mode: "SSB", QSODate: "20261025", TimeOn: "193200",
		ContestID: "c1", ContestADIFID: "CQ-WW-SSB",
		STX: 21, SRX: 22, STXString: "21", SRXString: "22",
	}
	in := buildUpdateInput(q)

	if in.STX == nil || *in.STX != 21 {
		t.Errorf("STX = %v", in.STX)
	}
	if in.SRX == nil || *in.SRX != 22 {
		t.Errorf("SRX = %v", in.SRX)
	}
	if in.STXString == nil || *in.STXString != "21" {
		t.Errorf("STXString = %v", in.STXString)
	}
	if in.SRXString == nil || *in.SRXString != "22" {
		t.Errorf("SRXString = %v", in.SRXString)
	}
}

// A non-contest contact sends zeros/empties, which the wavelog layer turns into
// JSON null — that clears stale serials instead of leaving them behind.
func TestBuildUpdateInputClearsExchangeWithoutContest(t *testing.T) {
	q := &qso.QSO{Call: "DL1ABC", Band: "20m", Mode: "CW", QSODate: "20261025", TimeOn: "193200"}
	in := buildUpdateInput(q)
	if in.STX == nil || *in.STX != 0 || in.SRX == nil || *in.SRX != 0 {
		t.Errorf("serials = %v/%v, want 0/0", in.STX, in.SRX)
	}
	if in.STXString == nil || *in.STXString != "" {
		t.Errorf("STXString = %v, want empty", in.STXString)
	}
}
