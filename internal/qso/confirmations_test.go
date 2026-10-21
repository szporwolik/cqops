package qso

import (
	"strings"
	"testing"

	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
)

// TestParseADIFRecordConfirmations: the three QSL-received statuses are
// normalized to Y/N/unknown, and any Y (or ADIF's import-only V) confirms.
func TestParseADIFRecordConfirmations(t *testing.T) {
	r := adif.NewRecord()
	r[adifield.CALL] = "SP9MOA"
	r[adifield.QSO_DATE] = "20240501"
	r[adifield.TIME_ON] = "120000"
	r[adifield.BAND] = "20m"
	r[adifield.MODE] = "SSB"
	r[adifield.QSL_RCVD] = "y"
	r[adifield.LOTW_QSL_RCVD] = "V"
	r[adifield.EQSL_QSL_RCVD] = "N"

	q := ParseADIFRecord(r, "import")
	if q.QSLRcvd != "Y" {
		t.Errorf("QSLRcvd = %q, want Y", q.QSLRcvd)
	}
	if q.LOTWQSLRcvd != "Y" {
		t.Errorf("LOTWQSLRcvd = %q, want Y (V is verified)", q.LOTWQSLRcvd)
	}
	if q.EQSLQSLRcvd != "N" {
		t.Errorf("EQSLQSLRcvd = %q, want N", q.EQSLQSLRcvd)
	}
	if !q.Confirmed() {
		t.Error("a lotw/QSL confirmation must count as confirmed")
	}

	// Unknown/garbage statuses are not confirmations.
	r2 := adif.NewRecord()
	r2[adifield.CALL] = "SP9MOA"
	r2[adifield.QSO_DATE] = "20240501"
	r2[adifield.TIME_ON] = "120000"
	r2[adifield.BAND] = "20m"
	r2[adifield.MODE] = "SSB"
	r2[adifield.QSL_RCVD] = "maybe"
	q2 := ParseADIFRecord(r2, "import")
	if q2.QSLRcvd != "" {
		t.Errorf("garbage status = %q, want empty", q2.QSLRcvd)
	}
	if q2.Confirmed() {
		t.Error("garbage status must not confirm a QSO")
	}
}

// TestADIFExportConfirmations: confirmations are exported when set and left
// out entirely when not provided (a missing field is not a "no").
func TestADIFExportConfirmations(t *testing.T) {
	q := &QSO{
		Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB", RSTSent: "59", RSTRcvd: "59",
		QSLRcvd: "Y", LOTWQSLRcvd: "N",
	}
	out := q.ToADIF()
	if !strings.Contains(out, "QSL_RCVD") || !strings.Contains(out, "LOTW_QSL_RCVD") {
		t.Errorf("confirmations missing from export: %s", out)
	}
	if strings.Contains(out, "EQSL_QSL_RCVD") {
		t.Error("an unset status must stay absent from the ADIF record")
	}

	plain := &QSO{Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"}
	if out := plain.ToADIF(); strings.Contains(out, "QSL_RCVD") {
		t.Errorf("unconfirmed QSO exported a status: %s", out)
	}
}

// TestDXCCSourceFromADIF: an entity number carried by the record is recorded
// as imported provenance, not as a local guess.
func TestDXCCSourceFromADIF(t *testing.T) {
	r := adif.NewRecord()
	r[adifield.CALL] = "SP9MOA"
	r[adifield.QSO_DATE] = "20240501"
	r[adifield.TIME_ON] = "120000"
	r[adifield.BAND] = "20m"
	r[adifield.MODE] = "SSB"
	r[adifield.DXCC] = "269"

	q := ParseADIFRecord(r, "wavelog")
	if q.DXCCSource != DXCCSourceImported {
		t.Errorf("DXCCSource = %q, want %q", q.DXCCSource, DXCCSourceImported)
	}

	r2 := adif.NewRecord()
	r2[adifield.CALL] = "SP9MOA"
	r2[adifield.QSO_DATE] = "20240501"
	r2[adifield.TIME_ON] = "120000"
	r2[adifield.BAND] = "20m"
	r2[adifield.MODE] = "SSB"
	if q := ParseADIFRecord(r2, "wavelog"); q.DXCCSource != "" {
		t.Errorf("DXCCSource without an entity = %q, want empty (resolved later)", q.DXCCSource)
	}
}
