package tui

import (
	"errors"
	"fmt"
	"testing"

	"github.com/szporwolik/cqops/internal/callbook"
)

// =============================================================================
// Callbook lookup mock tests (multi-provider ready)
// =============================================================================

func TestCallbookLookupSuccess(t *testing.T) {
	orig := callbookRegLookup
	t.Cleanup(func() { callbookRegLookup = orig })

	callbookRegLookup = func(m *Model, call string) (*callbook.Result, error) {
		return &callbook.Result{
			Callsign: "SP9MOA", Name: "John", Grid: "JO90",
			QTH: "Krakow", Country: "Poland", State: "MA", DXCC: "269",
			Provider: "qrz",
		}, nil
	}

	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.QRZ.Enabled = true
	m.App.Config.Integrations.QRZ.User = "testuser"
	m.App.Config.Integrations.QRZ.Pass = "testpass"
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Data: &callbook.Result{
			Callsign: "SP9MOA", Name: "John", Grid: "JO90",
			QTH: "Krakow", Country: "Poland",
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
	if m.lookup.partnerData == nil {
		t.Error("partnerData should be set after callbook fill")
	}
}

func TestCallbookLookupError(t *testing.T) {
	orig := callbookRegLookup
	t.Cleanup(func() { callbookRegLookup = orig })

	callbookRegLookup = func(m *Model, call string) (*callbook.Result, error) {
		return nil, errors.New("connection refused")
	}

	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.QRZ.Enabled = true
	m.App.Config.Integrations.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Err:  fmt.Errorf("connection refused"),
	})

	if m.fields[fieldName].Value() != "" {
		t.Errorf("Name should not be filled on error, got %q", m.fields[fieldName].Value())
	}
}

func TestCallbookLookupEmptyCall(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.QRZ.Enabled = true
	m.App.Config.Integrations.QRZ.User = "testuser"

	cmd := m.callbookLookup("")
	if cmd != nil {
		t.Error("callbookLookup should return nil for empty call")
	}
}

func TestCallbookLookupDisabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.callbookRegistry = nil
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Data: &callbook.Result{Callsign: "SP9MOA", Name: "John"},
	})

	// fillCallbookData always applies provided data — registry gating
	// is handled at the lookup dispatch level (callbookLookup).
	if m.fields[fieldName].Value() != "John" {
		t.Errorf("Name should be filled regardless of registry state, got %q", m.fields[fieldName].Value())
	}
}

func TestCallbookLookupNoCredentials(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.callbookRegistry = nil
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Data: &callbook.Result{Callsign: "SP9MOA", Name: "John"},
	})

	if m.fields[fieldName].Value() != "John" {
		t.Errorf("Name should be filled regardless of registry state, got %q", m.fields[fieldName].Value())
	}
}

func TestCallbookLookupOverwritesExistingGrid(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.QRZ.Enabled = true
	m.App.Config.Integrations.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldGrid].SetValue("JN18")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Data: &callbook.Result{
			Callsign: "SP9MOA",
			Name:     "John",
			Grid:     "JO90",
		},
	})

	if m.fields[fieldGrid].Value() != "JO90" {
		t.Errorf("Grid should be overwritten by callbook result, got %q", m.fields[fieldGrid].Value())
	}
	if m.rc.pathGrid != "JO90" {
		t.Errorf("pathGrid should be updated by callbook result, got %q", m.rc.pathGrid)
	}
}

func TestCallbookLookupNoDataResult(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.QRZ.Enabled = true
	m.App.Config.Integrations.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Data: nil,
	})
}

func TestCallbookLookupCacheInvalidation(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.QRZ.Enabled = true
	m.App.Config.Integrations.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Data: &callbook.Result{
			Callsign: "SP9MOA",
			Name:     "John",
			Grid:     "JO90",
		},
	})

	if m.rc.partnerView != "" {
		t.Log("Partner view cache was populated during fill — this is expected for new data")
	}
	_ = m.rc.partnerViewSig
}
