package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

func TestParseWSJTXADIFValid(t *testing.T) {
	adif := "SP9MOA de DJ7NT\n" +
		"<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500 " +
		"<QSO_DATE:8>20260614 <TIME_ON:6>120000 " +
		"<RST_SENT:2>59 <RST_RCVD:2>59 <GRIDSQUARE:4>JO90 " +
		"<NAME:4>John <QTH:6>Krakow <COUNTRY:6>Poland <EOR>"

	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Call != "SP9MOA" {
		t.Errorf("Call = %q; want SP9MOA", qs.Call)
	}
	if qs.Band != "20m" {
		t.Errorf("Band = %q; want 20m", qs.Band)
	}
	if qs.Mode != "SSB" {
		t.Errorf("Mode = %q; want SSB", qs.Mode)
	}
	if qs.Freq != 14.2500 {
		t.Errorf("Freq = %f; want 14.2500", qs.Freq)
	}
	if qs.QSODate != "20260614" {
		t.Errorf("QSODate = %q; want 20260614", qs.QSODate)
	}
	if qs.TimeOn != "120000" {
		t.Errorf("TimeOn = %q; want 120000", qs.TimeOn)
	}
	if qs.RSTSent != "59" {
		t.Errorf("RSTSent = %q; want 59", qs.RSTSent)
	}
	if qs.RSTRcvd != "59" {
		t.Errorf("RSTRcvd = %q; want 59", qs.RSTRcvd)
	}
	if qs.GridSquare != "JO90" {
		t.Errorf("GridSquare = %q; want JO90", qs.GridSquare)
	}
	if qs.Name != "John" {
		t.Errorf("Name = %q; want John", qs.Name)
	}
	if qs.QTH != "Krakow" {
		t.Errorf("QTH = %q; want Krakow", qs.QTH)
	}
	if qs.Country != "Poland" {
		t.Errorf("Country = %q; want Poland", qs.Country)
	}
}

func TestParseWSJTXADIFEmpty(t *testing.T) {
	qs := parseWSJTXADIF("")
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil for empty input")
	}
	// Should return an empty QSO, not nil
	if qs.Call != "" {
		t.Errorf("Call should be empty for empty ADIF, got %q", qs.Call)
	}
}

func TestParseWSJTXADIFNoCall(t *testing.T) {
	adif := "<BAND:3>20m <MODE:3>SSB <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Call != "" {
		t.Errorf("Call should be empty, got %q", qs.Call)
	}
	if qs.Band != "20m" {
		t.Errorf("Band = %q; want 20m", qs.Band)
	}
}

func TestParseWSJTXADIFModeSubmode(t *testing.T) {
	adif := "<CALL:6>SP9MOA <MODE:4>FT8 <SUBMODE:0> <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Mode != "FT8" {
		t.Errorf("Mode = %q; want FT8", qs.Mode)
	}
}

func TestParseWSJTXADIFBandFromFreq(t *testing.T) {
	adif := "<CALL:6>SP9MOA <FREQ:7>14.2500 <MODE:3>SSB <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Band != "20m" {
		t.Errorf("Band = %q; want 20m (derived from 14.250 MHz)", qs.Band)
	}
}

func TestParseWSJTXADIFAllFields(t *testing.T) {
	adif := strings.Join([]string{
		"<CALL:6>SP9MOA",
		"<BAND:3>20m",
		"<FREQ:7>14.2500",
		"<FREQ_RX:7>14.2500",
		"<MODE:3>SSB",
		"<SUBMODE:3>USB",
		"<QSO_DATE:8>20260614",
		"<TIME_ON:6>120000",
		"<TIME_OFF:6>120500",
		"<RST_SENT:2>59",
		"<RST_RCVD:2>59",
		"<GRIDSQUARE:4>JO90",
		"<NAME:4>John",
		"<QTH:6>Krakow",
		"<COUNTRY:6>Poland",
		"<COMMENT:12>Nice contact",
		"<TX_PWR:4>100W",
		"<STATION_CALLSIGN:5>DJ7NT",
		"<OPERATOR:5>DJ7NT",
		"<MY_GRIDSQUARE:4>JO30",
		"<SOTA_REF:9>SP/TA-001",
		"<POTA_REF:7>SP-0001",
		"<WWFF_REF:9>SPFF-0001",
		"<IOTA:6>EU-001",
		"<MY_SOTA_REF:9>SP/TA-002",
		"<MY_POTA_REF:7>SP-0002",
		"<MY_WWFF_REF:9>SPFF-0002",
		"<EOR>",
	}, " ")

	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	if qs.Call != "SP9MOA" {
		t.Errorf("Call = %q", qs.Call)
	}
	if qs.FreqRx != 14.2500 {
		t.Errorf("FreqRx = %f", qs.FreqRx)
	}
	if qs.Submode != "USB" {
		t.Errorf("Submode = %q", qs.Submode)
	}
	if qs.TimeOff != "120500" {
		t.Errorf("TimeOff = %q", qs.TimeOff)
	}
	if qs.Comment != "Nice contact" {
		t.Errorf("Comment = %q", qs.Comment)
	}
	if qs.TXPower != "100W" {
		t.Errorf("TXPower = %q", qs.TXPower)
	}
	if qs.StationCallsign != "DJ7NT" {
		t.Errorf("StationCallsign = %q", qs.StationCallsign)
	}
	if qs.Operator != "DJ7NT" {
		t.Errorf("Operator = %q", qs.Operator)
	}
	if qs.MyGridSquare != "JO30" {
		t.Errorf("MyGridSquare = %q", qs.MyGridSquare)
	}
	if qs.SOTARef != "SP/TA-001" {
		t.Errorf("SOTARef = %q", qs.SOTARef)
	}
	if qs.POTARef != "SP-0001" {
		t.Errorf("POTARef = %q", qs.POTARef)
	}
	if qs.IOTA != "EU-001" {
		t.Errorf("IOTA = %q", qs.IOTA)
	}
	if qs.MySOTARef != "SP/TA-002" {
		t.Errorf("MySOTARef = %q", qs.MySOTARef)
	}
}

func TestParseWSJTXADIFMalformed(t *testing.T) {
	// Malformed ADIF should not panic
	adif := "garbage data not valid adif"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil for malformed input")
	}
}

func TestParseWSJTXADIFSource(t *testing.T) {
	adif := "<CALL:6>SP9MOA <MODE:3>SSB <EOR>"
	qs := parseWSJTXADIF(adif)
	if qs == nil {
		t.Fatal("parseWSJTXADIF returned nil")
	}
	// Source should not be set by parser (it's set by NewQSO)
	if qs.Source != "manual" {
		t.Errorf("Source = %q; want 'manual' (from NewQSO)", qs.Source)
	}
}

// Verify qso types import
var _ = qso.NewQSO
