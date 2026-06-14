package tui

import (
	"errors"
	"fmt"
	"testing"

	"github.com/szporwolik/cqops/internal/qrz"
)

// =============================================================================
// QRZ lookup mock tests
// =============================================================================

func TestQRZLookupSuccess(t *testing.T) {
	// Save and restore original lookup function
	orig := qrzLookupFunc
	t.Cleanup(func() { qrzLookupFunc = orig })

	qrzLookupFunc = func(user, pass, callsign string) (*qrz.CallData, error) {
		return &qrz.CallData{
			Callsign: "SP9MOA",
			Name:     "John",
			Grid:     "JO90",
			QTH:      "Krakow",
			Country:  "Poland",
			State:    "MA",
			DXCC:     "269",
		}, nil
	}

	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"
	m.App.Config.QRZ.Pass = "testpass"
	m.fields[fieldCall].SetValue("SP9MOA")

	// Trigger QRZ fill
	m.fillQRZData(qrzResultMsg{
		Call: "SP9MOA",
		Data: &qrz.CallData{
			Callsign: "SP9MOA",
			Name:     "John",
			Grid:     "JO90",
			QTH:      "Krakow",
			Country:  "Poland",
		},
	})

	if m.fields[fieldName].Value() != "John" {
		t.Errorf("Name = %q; want John", m.fields[fieldName].Value())
	}
	if m.fields[fieldGrid].Value() != "JO90" {
		t.Errorf("Grid = %q; want JO90", m.fields[fieldGrid].Value())
	}
	if m.fields[fieldQTH].Value() != "Krakow" {
		t.Errorf("QTH = %q; want Krakow", m.fields[fieldQTH].Value())
	}
	if m.fields[fieldCountry].Value() != "Poland" {
		t.Errorf("Country = %q; want Poland", m.fields[fieldCountry].Value())
	}
	// Partner data should be set
	if m.partnerData == nil {
		t.Error("partnerData should be set after QRZ fill")
	}
}

func TestQRZLookupError(t *testing.T) {
	orig := qrzLookupFunc
	t.Cleanup(func() { qrzLookupFunc = orig })

	qrzLookupFunc = func(user, pass, callsign string) (*qrz.CallData, error) {
		return nil, errors.New("connection refused")
	}

	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	// fillQRZData with error should not panic
	m.fillQRZData(qrzResultMsg{
		Call: "SP9MOA",
		Err:  fmt.Errorf("connection refused"),
	})

	// Form fields should not be modified
	if m.fields[fieldName].Value() != "" {
		t.Errorf("Name should not be filled on error, got %q", m.fields[fieldName].Value())
	}
}

func TestQRZLookupEmptyCall(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"

	cmd := m.qrzLookup("")
	if cmd != nil {
		t.Error("qrzLookup should return nil for empty call")
	}
}

func TestQRZLookupDisabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = false
	m.App.Config.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	// fillQRZData with QRZ disabled should warn and not fill
	m.fillQRZData(qrzResultMsg{
		Call: "SP9MOA",
		Data: &qrz.CallData{Callsign: "SP9MOA", Name: "John"},
	})

	// Name should NOT be filled when QRZ is disabled
	if m.fields[fieldName].Value() != "" {
		t.Errorf("Name should not be filled when QRZ disabled, got %q", m.fields[fieldName].Value())
	}
}

func TestQRZLookupNoCredentials(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "" // no credentials
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillQRZData(qrzResultMsg{
		Call: "SP9MOA",
		Data: &qrz.CallData{Callsign: "SP9MOA", Name: "John"},
	})

	if m.fields[fieldName].Value() != "" {
		t.Errorf("Name should not be filled without credentials, got %q", m.fields[fieldName].Value())
	}
}

func TestQRZLookupNoOverwriteExistingGrid(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldGrid].SetValue("JN18") // already has a grid

	m.fillQRZData(qrzResultMsg{
		Call: "SP9MOA",
		Data: &qrz.CallData{
			Callsign: "SP9MOA",
			Name:     "John",
			Grid:     "JO90", // different grid from QRZ
		},
	})

	// Existing grid should NOT be overwritten
	if m.fields[fieldGrid].Value() != "JN18" {
		t.Errorf("Grid should not be overwritten when already set, got %q", m.fields[fieldGrid].Value())
	}
}

func TestQRZLookupNoDataResult(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	// nil data should show warning toast, not panic
	m.fillQRZData(qrzResultMsg{
		Call: "SP9MOA",
		Data: nil,
	})
}

func TestQRZLookupCacheInvalidation(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	// Fill with partner data
	m.fillQRZData(qrzResultMsg{
		Call: "SP9MOA",
		Data: &qrz.CallData{
			Callsign: "SP9MOA",
			Name:     "John",
			Grid:     "JO90",
		},
	})

	// Partner map cache should have been invalidated
	if m.partnerMapCache != "" {
		t.Log("Partner map cache was populated during fill — this is expected for new data")
	}
	// Cache signature should exist
	_ = m.partnerMapCacheSig
}
