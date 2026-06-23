package tui

import (
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Test helpers for ADIF persistence tests
// =============================================================================

// newADIFTestModel creates a Model backed by a temp SQLite database, with
// a minimal station config sufficient for ADIF-to-QSO logging.
func newADIFTestModel(t *testing.T) *Model {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		General: config.GeneralConfig{DistanceUnit: "km"},
		Logbooks: map[string]config.Logbook{
			"test": {
				Station: config.Station{
					Callsign: "DJ7NT",
					Grid:     "JO30",
				},
			},
		},
	}
	a := &app.App{
		Config:      cfg,
		DB:          db,
		LogbookName: "test",
		Logbook:     &config.Logbook{Station: config.Station{Callsign: "DJ7NT", Grid: "JO30"}},
	}
	return New(a, nil)
}

// adifFT8 is a realistic WSJT-X logged-QSO ADIF for FT8.
const adifFT8 = "<CALL:6>SP9MOA <BAND:3>20m <FREQ:8>14.074550 <MODE:3>FT8 <SUBMODE:0> " +
	"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <TIME_OFF:6>120100 " +
	"<RST_SENT:3>-10 <RST_RCVD:3>-05 <GRIDSQUARE:6>JO90aa " +
	"<NAME:4>John <QTH:6>Krakow <COUNTRY:6>Poland <COMMENT:6>73 GL! <EOR>"

// adifFT4 is a realistic FT4 ADIF (WSJT-X logs MFSK mode + FT4 submode).
const adifFT4 = "<CALL:4>W1AW <BAND:3>15m <FREQ:8>21.140500 <MODE:4>MFSK <SUBMODE:3>FT4 " +
	"<QSO_DATE:8>20260618 <TIME_ON:6>130000 <RST_SENT:3>-08 <RST_RCVD:3>+02 " +
	"<GRIDSQUARE:6>FN31pr <EOR>"

// adifSSB is a representative SSB ADIF.
const adifSSB = "<CALL:5>G4ABC <BAND:3>40m <FREQ:6>7.1850 <MODE:3>SSB <SUBMODE:3>LSB " +
	"<QSO_DATE:8>20260618 <TIME_ON:6>140000 <RST_SENT:2>59 <RST_RCVD:2>55 " +
	"<NAME:4>Paul <EOR>"

// =============================================================================
// End-to-end ADIF → QSO persistence tests (real parsing + real SQLite)
// =============================================================================

func TestADIFToQSO_FT8(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Config.Integrations.QRZ.Enabled = false
	// Disable Wavelog so upload is not triggered.
	m.App.Logbook.Wavelog = nil

	// Parse ADIF → QSO struct.
	qs := parseWSJTXADIF(adifFT8)
	if qs.Call != "SP9MOA" {
		t.Fatalf("parse call = %q, want SP9MOA", qs.Call)
	}

	// Apply station defaults (same as logQSOFromADIF does).
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	// Validate.
	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}

	// Insert into temp DB.
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	if id == 0 {
		t.Fatal("InsertQSO returned id=0")
	}

	// Read back.
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	// Verify critical fields.
	if stored.Call != "SP9MOA" {
		t.Errorf("Call = %q", stored.Call)
	}
	if stored.Band != "20m" {
		t.Errorf("Band = %q", stored.Band)
	}
	if stored.Mode != "FT8" {
		t.Errorf("Mode = %q, want FT8 (standalone mode per ADIF 3.1.4)", stored.Mode)
	}
	if stored.Freq < 14.074 || stored.Freq > 14.075 {
		t.Errorf("Freq = %f", stored.Freq)
	}
	if stored.QSODate != "20260618" {
		t.Errorf("QSODate = %q", stored.QSODate)
	}
	if stored.TimeOn != "120000" {
		t.Errorf("TimeOn = %q", stored.TimeOn)
	}
	if stored.RSTSent != "-10" {
		t.Errorf("RSTSent = %q", stored.RSTSent)
	}
	if stored.RSTRcvd != "-05" {
		t.Errorf("RSTRcvd = %q", stored.RSTRcvd)
	}
	if stored.GridSquare != "JO90AA" {
		t.Errorf("GridSquare = %q", stored.GridSquare)
	}
	if stored.Name != "John" {
		t.Errorf("Name = %q", stored.Name)
	}
	if stored.QTH != "Krakow" {
		t.Errorf("QTH = %q", stored.QTH)
	}
	if stored.Country != "Poland" {
		t.Errorf("Country = %q", stored.Country)
	}
	if stored.Comment != "73 GL!" {
		t.Errorf("Comment = %q", stored.Comment)
	}
	if stored.Source != "wsjtx" {
		t.Errorf("Source = %q, want wsjtx", stored.Source)
	}
	if stored.WavelogUploaded != "no" {
		t.Errorf("WavelogUploaded = %q, want no", stored.WavelogUploaded)
	}
	if stored.StationCallsign != "DJ7NT" {
		t.Errorf("StationCallsign = %q, want DJ7NT", stored.StationCallsign)
	}
}

func TestADIFToQSO_FT4(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	qs := parseWSJTXADIF(adifFT4)
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	if stored.Call != "W1AW" {
		t.Errorf("Call = %q", stored.Call)
	}
	if stored.Band != "15m" {
		t.Errorf("Band = %q", stored.Band)
	}
	if stored.Mode != "MFSK" {
		t.Errorf("Mode = %q, want MFSK (FT4 normalized)", stored.Mode)
	}
	if stored.Submode != "FT4" {
		t.Errorf("Submode = %q, want FT4", stored.Submode)
	}
	if stored.GridSquare != "FN31PR" {
		t.Errorf("GridSquare = %q", stored.GridSquare)
	}
}

func TestADIFToQSO_SSB(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	qs := parseWSJTXADIF(adifSSB)
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	if stored.Call != "G4ABC" {
		t.Errorf("Call = %q", stored.Call)
	}
	if stored.Band != "40m" {
		t.Errorf("Band = %q", stored.Band)
	}
	if stored.Mode != "SSB" {
		t.Errorf("Mode = %q", stored.Mode)
	}
	if stored.Submode != "LSB" {
		t.Errorf("Submode = %q, want LSB", stored.Submode)
	}
	if stored.RSTSent != "59" {
		t.Errorf("RSTSent = %q", stored.RSTSent)
	}
}

func TestADIFToQSO_CW(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	adifCW := "<CALL:6>DL1ABC <BAND:3>30m <FREQ:7>10.1180 <MODE:2>CW " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>150000 <RST_SENT:3>599 <RST_RCVD:3>579 <EOR>"

	qs := parseWSJTXADIF(adifCW)
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	if stored.Call != "DL1ABC" {
		t.Errorf("Call = %q", stored.Call)
	}
	if stored.Band != "30m" {
		t.Errorf("Band = %q", stored.Band)
	}
	if stored.Mode != "CW" {
		t.Errorf("Mode = %q", stored.Mode)
	}
}

func TestADIFToQSO_BandFromFrequency(t *testing.T) {
	// ADIF with frequency but no explicit band — band should be derived.
	adif := "<CALL:6>SP9MOA <FREQ:6>3.7950 <MODE:3>SSB " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>160000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	qs := parseWSJTXADIF(adif)
	// The parser derives band from frequency when band is empty.
	if qs.Band != "80m" {
		t.Errorf("band should be derived from 3.795 MHz as 80m, got %q", qs.Band)
	}
}

func TestADIFToQSO_ModeSubmodeNormalization(t *testing.T) {
	// MFSK + FT4 submodes should normalize correctly.
	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:8>14.080500 <MODE:4>MFSK <SUBMODE:3>FT4 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>170000 <RST_SENT:3>-05 <RST_RCVD:3>+02 <EOR>"

	qs := parseWSJTXADIF(adif)
	// MFSK + FT4 should stay as MFSK/FT4 (not remapped).
	if qs.Mode != "MFSK" {
		t.Errorf("Mode = %q, want MFSK", qs.Mode)
	}
	if qs.Submode != "FT4" {
		t.Errorf("Submode = %q, want FT4", qs.Submode)
	}

	// USB alone should normalize to SSB/USB.
	adif2 := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:7>14.2500 <MODE:3>USB " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>170500 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"
	qs2 := parseWSJTXADIF(adif2)
	if qs2.Mode != "SSB" {
		t.Errorf("USB should normalize to SSB, got %q", qs2.Mode)
	}
	if qs2.Submode != "USB" {
		t.Errorf("USB submode = %q, want USB", qs2.Submode)
	}
}

// =============================================================================
// Invalid/malformed ADIF rejection tests
// =============================================================================

func TestADIFToQSO_MissingCall(t *testing.T) {
	m := newADIFTestModel(t)
	adif := "<BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	qs := parseWSJTXADIF(adif)
	if qs.Call != "" {
		t.Errorf("Call should be empty for ADIF with no CALL field, got %q", qs.Call)
	}
	// logQSOFromADIF would skip this permanently (returns nil, false).
	cmd, retry := m.logQSOFromADIF(adif)
	if cmd != nil {
		t.Error("logQSOFromADIF should return nil cmd for missing call")
	}
	if retry {
		t.Error("logQSOFromADIF should not retry missing-call ADIF")
	}
}

func TestADIFToQSO_InvalidCall(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil
	adif := "<CALL:4>!!!! <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	qs := parseWSJTXADIF(adif)
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	err := qso.ValidateForSave(qs)
	if err == nil {
		t.Error("ValidateForSave should reject invalid callsign '!!!!'")
	}
}

func TestADIFToQSO_EmptyAdifNoPanic(t *testing.T) {
	qs := parseWSJTXADIF("")
	if qs == nil {
		t.Fatal("parseWSJTXADIF should not return nil for empty input")
	}
	if qs.Call != "" {
		t.Errorf("Call should be empty, got %q", qs.Call)
	}

	// logQSOFromADIF should skip empty ADIF.
	m := newADIFTestModel(t)
	cmd, retry := m.logQSOFromADIF("")
	if cmd != nil {
		t.Error("logQSOFromADIF should return nil cmd for empty ADIF")
	}
	if retry {
		t.Error("logQSOFromADIF should not retry empty ADIF")
	}
}

func TestADIFToQSO_MalformedAdifNoPanic(t *testing.T) {
	qs := parseWSJTXADIF("garbage not valid")
	if qs == nil {
		t.Fatal("parseWSJTXADIF should not return nil for malformed input")
	}

	m := newADIFTestModel(t)
	cmd, retry := m.logQSOFromADIF("garbage not valid")
	if cmd != nil {
		t.Error("logQSOFromADIF should return nil cmd for malformed ADIF with no call")
	}
	if retry {
		t.Error("logQSOFromADIF should not retry malformed ADIF")
	}
}

func TestADIFToQSO_InvalidFrequency(t *testing.T) {
	// Frequency 0 or negative should still parse but may fail validation.
	adif := "<CALL:6>SP9MOA <FREQ:3>999 <MODE:3>SSB " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	qs := parseWSJTXADIF(adif)
	// 999 MHz doesn't map to a ham band, so band derivation will fail.
	if qs.Band != "" {
		t.Errorf("999 MHz should not derive a band, got %q", qs.Band)
	}
}

func TestADIFToQSO_InvalidGrid(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil
	adif := "<CALL:6>SP9MOA <GRIDSQUARE:4>XXXX <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	qs := parseWSJTXADIF(adif)
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	// Validate should reject invalid locator.
	err := qso.ValidateForSave(qs)
	if err == nil {
		t.Error("ValidateForSave should reject invalid grid 'XXXX'")
	}
}

// =============================================================================
// Duplicate detection tests
// =============================================================================

func TestADIFToQSO_DuplicateDetection(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	adif := "<CALL:6>SP9MOA <BAND:3>20m <MODE:4>MFSK <SUBMODE:3>FT4 " +
		"<FREQ:8>14.080500 <QSO_DATE:8>20260618 <TIME_ON:6>120000 " +
		"<RST_SENT:3>-05 <RST_RCVD:3>+02 <EOR>"

	// First insert should succeed.
	qs := parseWSJTXADIF(adif)
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})
	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("first InsertQSO: %v", err)
	}
	if id == 0 {
		t.Fatal("first insert returned id=0")
	}

	// Second attempt via logQSOFromADIF should be detected as duplicate.
	cmd, retry := m.logQSOFromADIF(adif)
	if cmd != nil {
		t.Error("duplicate should not produce an upload command")
	}
	if retry {
		t.Error("duplicate should not request retry")
	}
}

// =============================================================================
// adifQueue boundary tests
// =============================================================================

func TestAdifQueue_EnqueueDrain(t *testing.T) {
	m := newADIFTestModel(t)

	// Enqueue a valid item.
	m.adifQ.mu.Lock()
	m.adifQ.adifs = append(m.adifQ.adifs, adifFT8)
	m.adifQ.mu.Unlock()

	// Lock and drain.
	m.adifQ.mu.Lock()
	adifs := m.adifQ.adifs
	m.adifQ.adifs = nil
	m.adifQ.mu.Unlock()

	if len(adifs) != 1 {
		t.Fatalf("expected 1 ADIF, got %d", len(adifs))
	}
	if adifs[0] != adifFT8 {
		t.Error("drained ADIF does not match enqueued")
	}
}

func TestAdifQueue_EmptyIsSafe(t *testing.T) {
	m := newADIFTestModel(t)

	// Lock and drain an empty queue.
	m.adifQ.mu.Lock()
	adifs := m.adifQ.adifs
	m.adifQ.adifs = nil
	m.adifQ.mu.Unlock()

	if len(adifs) != 0 {
		t.Errorf("empty queue should return 0 items, got %d", len(adifs))
	}
}

func TestAdifQueue_MultipleItemsPreserveOrder(t *testing.T) {
	m := newADIFTestModel(t)

	m.adifQ.mu.Lock()
	m.adifQ.adifs = append(m.adifQ.adifs, adifFT8, adifFT4, adifSSB)
	m.adifQ.mu.Unlock()

	m.adifQ.mu.Lock()
	adifs := m.adifQ.adifs
	m.adifQ.adifs = nil
	m.adifQ.mu.Unlock()

	if len(adifs) != 3 {
		t.Fatalf("expected 3 items, got %d", len(adifs))
	}
	if adifs[0] != adifFT8 || adifs[1] != adifFT4 || adifs[2] != adifSSB {
		t.Error("items lost order")
	}
}

func TestAdifQueue_MalformedItemDoesNotBlock(t *testing.T) {
	m := newADIFTestModel(t)

	// Enqueue malformed followed by valid.
	m.adifQ.mu.Lock()
	m.adifQ.adifs = append(m.adifQ.adifs, "garbage", adifFT8)
	m.adifQ.mu.Unlock()

	m.adifQ.mu.Lock()
	adifs := m.adifQ.adifs
	m.adifQ.adifs = nil
	m.adifQ.mu.Unlock()

	if len(adifs) != 2 {
		t.Fatalf("expected 2 items, got %d", len(adifs))
	}

	// Process malformed first — should skip silently.
	m.App.Logbook.Wavelog = nil
	cmd1, retry1 := m.logQSOFromADIF(adifs[0])
	if cmd1 != nil || retry1 {
		t.Error("malformed ADIF should be skipped, not retried")
	}

	// Process valid second — should succeed.
	cmd2, retry2 := m.logQSOFromADIF(adifs[1])
	if retry2 {
		t.Error("valid FT8 ADIF should not request retry")
	}
	_ = cmd2
}

// =============================================================================
// Wavelog boundary tests
// =============================================================================

func TestWavelogBoundary_DisabledDoesNotUpload(t *testing.T) {
	m := newADIFTestModel(t)
	// Wavelog is nil → disabled.
	m.App.Logbook.Wavelog = nil

	cmd, retry := m.logQSOFromADIF(adifFT8)
	if retry {
		t.Error("FT8 ADIF should not request retry")
	}
	// When Wavelog is disabled, maybeUploadRawADIFToWavelog returns nil.
	// The cmd from logQSOFromADIF may still be non-nil (refreshQSOS).
	_ = cmd
}

func TestWavelogBoundary_UploadState(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	qs := parseWSJTXADIF(adifFT8)
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})
	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	// Simulate successful upload by updating wavelog status.
	err = store.UpdateWavelogStatus(m.App.DB, id, "yes")
	if err != nil {
		t.Fatalf("UpdateWavelogStatus: %v", err)
	}

	// Read back and verify.
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogUploaded != "yes" {
		t.Errorf("WavelogUploaded = %q, want yes", stored.WavelogUploaded)
	}
}

// =============================================================================
// Real logQSOFromADIF flow with temp DB (full pipeline)
// =============================================================================

func TestLogQSOFromADIF_FullPipeline(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil
	m.App.Config.Integrations.QRZ.Enabled = false

	// Run the full logQSOFromADIF pipeline.
	cmd, retry := m.logQSOFromADIF(adifFT8)
	if retry {
		t.Fatal("logQSOFromADIF should not request retry for valid ADIF")
	}

	// Verify QSO was persisted (logQSOFromADIF calls InsertQSO internally).
	// The cmd may be nil if no upload/refresh is needed.
	_ = cmd

	// Query back from the DB to confirm persistence.
	qsos, err := store.ListQSOs(m.App.DB, 1, "")
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) == 0 {
		t.Fatal("no QSO found after logQSOFromADIF")
	}
	q := qsos[0]
	if q.Call != "SP9MOA" {
		t.Errorf("Call = %q", q.Call)
	}
	if q.Band != "20m" {
		t.Errorf("Band = %q", q.Band)
	}
	if q.Mode != "FT8" {
		t.Errorf("Mode = %q, want FT8 (standalone per ADIF 3.1.4)", q.Mode)
	}
	if q.Source != "wsjtx" {
		t.Errorf("Source = %q, want wsjtx", q.Source)
	}
	// Verify station defaults were applied.
	if q.StationCallsign != "DJ7NT" {
		t.Errorf("StationCallsign = %q, want DJ7NT", q.StationCallsign)
	}
}

// =============================================================================
// Standalone FT8/FT4 normalization persistence (Pass 6 regression tests)
// =============================================================================

func TestADIFToQSO_StandaloneFT8Normalized(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	// ADIF with FT8 as standalone mode (some tools export this way).
	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:8>14.074550 <MODE:3>FT8 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:3>-10 <RST_RCVD:3>-05 <EOR>"

	qs := parseWSJTXADIF(adif)
	// FT8 is a standalone mode per ADIF 3.1.4, no normalization needed.
	if qs.Mode != "FT8" {
		t.Errorf("standalone FT8 mode should remain FT8, got %q", qs.Mode)
	}
	if qs.Submode != "" {
		t.Errorf("standalone FT8 should have no submode, got %q", qs.Submode)
	}

	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	if stored.Mode != "FT8" {
		t.Errorf("stored mode = %q, want FT8", stored.Mode)
	}
	if stored.Submode != "" {
		t.Errorf("stored submode = %q, want empty (FT8 is standalone)", stored.Submode)
	}
}

func TestADIFToQSO_StandaloneFT4Normalized(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	adif := "<CALL:6>SP9MOA <BAND:3>15m <FREQ:8>21.140500 <MODE:3>FT4 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>130000 <RST_SENT:3>-08 <RST_RCVD:3>+02 <EOR>"

	qs := parseWSJTXADIF(adif)
	if qs.Mode != "MFSK" {
		t.Errorf("standalone FT4 mode should normalize to MFSK, got %q", qs.Mode)
	}
	if qs.Submode != "FT4" {
		t.Errorf("standalone FT4 submode should be FT4, got %q", qs.Submode)
	}

	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Callsign,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})

	if err := qso.ValidateForSave(qs); err != nil {
		t.Fatalf("ValidateForSave: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}

	if stored.Mode != "MFSK" {
		t.Errorf("stored mode = %q, want MFSK", stored.Mode)
	}
	if stored.Submode != "FT4" {
		t.Errorf("stored submode = %q, want FT4", stored.Submode)
	}
}

func TestADIFToQSO_StandaloneFT8LogQSO(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil
	m.App.Config.Integrations.QRZ.Enabled = false

	// Full pipeline with standalone FT8.
	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:8>14.074550 <MODE:3>FT8 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:3>-10 <RST_RCVD:3>-05 <EOR>"

	cmd, retry := m.logQSOFromADIF(adif)
	if retry {
		t.Fatal("standalone FT8 should not request retry")
	}
	_ = cmd

	// Verify persisted with normalized mode.
	qsos, err := store.ListQSOs(m.App.DB, 1, "")
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) == 0 {
		t.Fatal("no QSO found")
	}
	if qsos[0].Mode != "FT8" {
		t.Errorf("stored mode = %q, want FT8 (standalone per ADIF 3.1.4)", qsos[0].Mode)
	}
	if qsos[0].Submode != "" {
		t.Errorf("stored submode = %q, want empty", qsos[0].Submode)
	}
}

func TestADIFToQSO_MFSK_FT8_LegacyNormalized(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	// Legacy MFSK+FT8 ADIF should be normalized to standalone FT8.
	adif := "<CALL:6>SP9MOA <BAND:3>20m <FREQ:8>14.074550 <MODE:4>MFSK <SUBMODE:3>FT8 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:3>-10 <RST_RCVD:3>-05 <EOR>"

	qs := parseWSJTXADIF(adif)
	if qs.Mode != "FT8" {
		t.Errorf("legacy MFSK+FT8 mode should normalize to FT8, got %q", qs.Mode)
	}
	if qs.Submode != "" {
		t.Errorf("legacy MFSK+FT8 submode should be empty, got %q", qs.Submode)
	}

	qs.Source = "import"
	if err := qso.ValidateImportRecord(qs); err != nil {
		t.Fatalf("ValidateImportRecord: %v", err)
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Mode != "FT8" {
		t.Errorf("stored mode = %q, want FT8", stored.Mode)
	}
	if stored.Submode != "" {
		t.Errorf("stored submode = %q, want empty", stored.Submode)
	}
}

func TestADIFToQSO_MFSK_FT4_StillValid(t *testing.T) {
	m := newADIFTestModel(t)
	m.App.Logbook.Wavelog = nil

	// FT4 IS a valid MFSK submode per ADIF 3.1.4.
	adif := "<CALL:6>SP9MOA <BAND:3>15m <FREQ:8>21.140500 <MODE:4>MFSK <SUBMODE:3>FT4 " +
		"<QSO_DATE:8>20260618 <TIME_ON:6>130000 <RST_SENT:3>-08 <RST_RCVD:3>+02 <EOR>"

	qs := parseWSJTXADIF(adif)
	if qs.Mode != "MFSK" {
		t.Errorf("MFSK+FT4 mode should stay MFSK, got %q", qs.Mode)
	}
	if qs.Submode != "FT4" {
		t.Errorf("MFSK+FT4 submode should stay FT4, got %q", qs.Submode)
	}
}
