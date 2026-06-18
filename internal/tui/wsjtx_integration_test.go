package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textinput"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
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
	// FT8 normalizes to MFSK/FT8 since Pass 6.
	if qs.Mode != "MFSK" {
		t.Errorf("Mode = %q; want MFSK (FT8 normalized)", qs.Mode)
	}
	if qs.Submode != "FT8" {
		t.Errorf("Submode = %q; want FT8", qs.Submode)
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

// =============================================================================
// WSJT-X TX message handling tests
// =============================================================================

func TestApplyWSJTXStatus_StoresTxMessage(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.fields = [fieldCount]textinput.Model{}
	for i := range m.fields {
		m.fields[i] = newTextinput()
	}

	m.applyWSJTXStatus("SP9ABC", "JO90", 14074000, "FT8", "", "-12", "CQ SP9XXX JO90", true)

	if !m.wsjtx.online {
		t.Error("wsjtxOnline should be true after status")
	}
	if m.wsjtx.txMsg != "CQ SP9XXX JO90" {
		t.Errorf("wsjtxTxMsg = %q; want 'CQ SP9XXX JO90'", m.wsjtx.txMsg)
	}
	if !m.wsjtx.tx {
		t.Error("wsjtxTx should be true when transmitting=true")
	}
	if m.wsjtx.lastSeen.IsZero() {
		t.Error("wsjtxLastSeen should be set")
	}
	if m.rc.status != "" {
		t.Error("cachedStatus should be invalidated (empty)")
	}
}

func TestApplyWSJTXStatus_EmptyTxMessage(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.fields = [fieldCount]textinput.Model{}
	for i := range m.fields {
		m.fields[i] = newTextinput()
	}

	m.wsjtx.online = true
	m.wsjtx.txMsg = "CQ SP9XXX JO90"
	m.applyWSJTXStatus("", "", 0, "", "", "", "", false)

	if m.wsjtx.txMsg != "" {
		t.Errorf("wsjtxTxMsg should be cleared to empty, got %q", m.wsjtx.txMsg)
	}
	if !m.wsjtx.online {
		t.Error("wsjtxOnline should remain true even with empty message")
	}
}

func TestWSJTXWatchdog_Expires(t *testing.T) {
	m := &Model{}
	m.wsjtx.online = true
	m.wsjtx.txMsg = "CQ SP9XXX JO90"
	m.wsjtx.lastSeen = time.Now().Add(-20 * time.Second)

	// Simulate watchdog check.
	if m.wsjtx.online && time.Since(m.wsjtx.lastSeen) > 15*time.Second {
		m.wsjtx.online = false
		m.wsjtx.txMsg = ""
		m.rc.status = ""
	}

	if m.wsjtx.online {
		t.Error("watchdog should set wsjtxOnline to false after 15s of inactivity")
	}
	if m.wsjtx.txMsg != "" {
		t.Error("watchdog should clear wsjtxTxMsg")
	}
	if m.rc.status != "" {
		t.Error("watchdog should invalidate cachedStatus")
	}
}

func TestWSJTXWatchdog_NotExpired(t *testing.T) {
	m := &Model{}
	m.wsjtx.online = true
	m.wsjtx.txMsg = "CQ SP9XXX JO90"
	m.wsjtx.lastSeen = time.Now()

	if m.wsjtx.online && time.Since(m.wsjtx.lastSeen) > 15*time.Second {
		m.wsjtx.online = false
	}

	if !m.wsjtx.online {
		t.Error("watchdog should NOT expire when last seen is recent")
	}
	if m.wsjtx.txMsg != "CQ SP9XXX JO90" {
		t.Error("tx message should be preserved when watchdog does not expire")
	}
}

func TestApplyWSJTXStatus_DirectedCall(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.fields = [fieldCount]textinput.Model{}
	for i := range m.fields {
		m.fields[i] = newTextinput()
	}

	m.applyWSJTXStatus("K1ABC", "FN42", 21074000, "FT8", "", "-05", "K1ABC SP9XXX -05", true)

	if m.wsjtx.txMsg != "K1ABC SP9XXX -05" {
		t.Errorf("wsjtxTxMsg = %q; want directed call message", m.wsjtx.txMsg)
	}
	// Verify the call field was updated.
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if call != "K1ABC" {
		t.Errorf("call field = %q; want K1ABC", call)
	}
}
