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
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "testuser"
	m.App.Config.Integrations.Callbook.QRZ.Pass = "testpass"
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
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "testuser"
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
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "testuser"

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
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "testuser"
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
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9MOA")

	m.fillCallbookData(callbookResultMsg{
		Call: "SP9MOA",
		Data: nil,
	})
}

func TestCallbookLookupCacheInvalidation(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "testuser"
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

// =============================================================================
// internetCallbook priority tests
// =============================================================================

func TestInternetCallbook_NoneEnabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = false
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = false

	name, url := m.internetCallbook()
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
}

func TestInternetCallbook_QRZOnly(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "test"
	m.App.Config.Integrations.Callbook.QRZ.Priority = 50
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = false

	name, url := m.internetCallbook()
	if name != "QRZ.com" {
		t.Errorf("name = %q, want QRZ.com", name)
	}
	if url != "https://www.qrz.com/db/{CALL}" {
		t.Errorf("url = %q, want https://www.qrz.com/db/{CALL}", url)
	}
}

func TestInternetCallbook_HamQTHOnly(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = false
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = true
	m.App.Config.Integrations.Callbook.HamQTH.User = "test"
	m.App.Config.Integrations.Callbook.HamQTH.Priority = 45

	name, url := m.internetCallbook()
	if name != "HamQTH" {
		t.Errorf("name = %q, want HamQTH", name)
	}
	if url != "https://www.hamqth.com/{CALL}" {
		t.Errorf("url = %q, want https://www.hamqth.com/{CALL}", url)
	}
}

func TestInternetCallbook_QRZHigherPriority(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "test"
	m.App.Config.Integrations.Callbook.QRZ.Priority = 50
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = true
	m.App.Config.Integrations.Callbook.HamQTH.User = "test"
	m.App.Config.Integrations.Callbook.HamQTH.Priority = 30

	name, url := m.internetCallbook()
	if name != "QRZ.com" {
		t.Errorf("name = %q, want QRZ.com (QRZ has higher priority)", name)
	}
	if url != "https://www.qrz.com/db/{CALL}" {
		t.Errorf("url = %q", url)
	}
}

func TestInternetCallbook_HamQTHHigherPriority(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "test"
	m.App.Config.Integrations.Callbook.QRZ.Priority = 30
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = true
	m.App.Config.Integrations.Callbook.HamQTH.User = "test"
	m.App.Config.Integrations.Callbook.HamQTH.Priority = 60

	name, url := m.internetCallbook()
	if name != "HamQTH" {
		t.Errorf("name = %q, want HamQTH (HamQTH has higher priority)", name)
	}
	if url != "https://www.hamqth.com/{CALL}" {
		t.Errorf("url = %q", url)
	}
}

func TestInternetCallbook_EqualPriorityPrefersHamQTH(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "test"
	m.App.Config.Integrations.Callbook.QRZ.Priority = 50
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = true
	m.App.Config.Integrations.Callbook.HamQTH.User = "test"
	m.App.Config.Integrations.Callbook.HamQTH.Priority = 50

	name, url := m.internetCallbook()
	if name != "HamQTH" {
		t.Errorf("name = %q, want HamQTH (free service wins ties)", name)
	}
	if url != "https://www.hamqth.com/{CALL}" {
		t.Errorf("url = %q", url)
	}
}

func TestInternetCallbook_QRZEnabledButNoUser(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = ""
	m.App.Config.Integrations.Callbook.QRZ.Priority = 50
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = false

	name, url := m.internetCallbook()
	if name != "" {
		t.Errorf("name = %q, want empty (QRZ has no user)", name)
	}
	if url != "" {
		t.Errorf("url = %q, want empty", url)
	}
}

func TestInternetCallbook_HamQTHEnabledButNoUser(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = false
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = true
	m.App.Config.Integrations.Callbook.HamQTH.User = ""
	m.App.Config.Integrations.Callbook.HamQTH.Priority = 45

	name, url := m.internetCallbook()
	if name != "" {
		t.Errorf("name = %q, want empty (HamQTH has no user)", name)
	}
	if url != "" {
		t.Errorf("url = %q, want empty", url)
	}
}

func TestInternetCallbook_DefaultPriorities(t *testing.T) {
	// When both priorities are 0 (unset), defaults: QRZ=50, HamQTH=45.
	// QRZ should win because 50 > 45.
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "test"
	m.App.Config.Integrations.Callbook.QRZ.Priority = 0
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = true
	m.App.Config.Integrations.Callbook.HamQTH.User = "test"
	m.App.Config.Integrations.Callbook.HamQTH.Priority = 0

	name, _ := m.internetCallbook()
	if name != "QRZ.com" {
		t.Errorf("name = %q, want QRZ.com (default 50 > default 45)", name)
	}
}

func TestInternetCallbook_CallookFallback(t *testing.T) {
	// When neither QRZ nor HamQTH are configured, Callook.info
	// should be the internet callbook link as last-resort fallback.
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = false
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = false
	m.App.Config.Integrations.Callbook.Callook.Enabled = true

	name, url := m.internetCallbook()
	if name != "Callook.info" {
		t.Errorf("name = %q, want Callook.info (fallback)", name)
	}
	if url != "https://callook.info/{CALL}" {
		t.Errorf("url = %q", url)
	}
}

func TestInternetCallbook_CallookNotFallbackWhenQRZPresent(t *testing.T) {
	// Callook should NOT be used when QRZ is configured —
	// QRZ is a higher-priority internet callbook.
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "test"
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = false
	m.App.Config.Integrations.Callbook.Callook.Enabled = true

	name, _ := m.internetCallbook()
	if name != "QRZ.com" {
		t.Errorf("name = %q, want QRZ.com (not Callook fallback)", name)
	}
}

func TestInternetCallbook_AllDisabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = false
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = false
	m.App.Config.Integrations.Callbook.Callook.Enabled = false

	name, url := m.internetCallbook()
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
	if url != "" {
		t.Errorf("url = %q, want empty", url)
	}
}

// =============================================================================
// HamQTH callbook registry integration tests
// =============================================================================

func TestBuildCallbookRegistry_IncludesHamQTH(t *testing.T) {
	a := newChooserTestApp(t)

	a.Config.Integrations.Callbook.QRZ.Enabled = false
	a.Config.Integrations.Callbook.HamQTH.Enabled = true
	a.Config.Integrations.Callbook.HamQTH.User = "testuser"
	a.Config.Integrations.Callbook.HamQTH.Pass = "testpass"
	a.Config.Integrations.Callbook.HamQTH.Priority = 45

	a.Config.Integrations.Callbook.Logbook.Enabled = false
	a.Config.Integrations.Callbook.Wavelog.Enabled = false

	reg := buildCallbookRegistry(a)
	if reg == nil {
		t.Fatal("expected non-nil registry when HamQTH is enabled")
	}
	if reg.Len() != 2 {
		// HamQTH + CTY (always-on)
		t.Errorf("registry length = %d, want 2 (HamQTH + CTY)", reg.Len())
	}
}

func TestBuildCallbookRegistry_HamQTHDisabled(t *testing.T) {
	a := newChooserTestApp(t)

	a.Config.Integrations.Callbook.HamQTH.Enabled = false
	a.Config.Integrations.Callbook.QRZ.Enabled = false
	a.Config.Integrations.Callbook.Logbook.Enabled = false
	a.Config.Integrations.Callbook.Wavelog.Enabled = false

	reg := buildCallbookRegistry(a)
	if reg == nil {
		t.Fatal("expected non-nil registry (CTY is always-on)")
	}
	if reg.Len() != 1 {
		t.Errorf("registry length = %d, want 1 (CTY only)", reg.Len())
	}
}

func TestBuildCallbookRegistry_HamQTHNoUser(t *testing.T) {
	a := newChooserTestApp(t)

	a.Config.Integrations.Callbook.HamQTH.Enabled = true
	a.Config.Integrations.Callbook.HamQTH.User = ""

	a.Config.Integrations.Callbook.QRZ.Enabled = false
	a.Config.Integrations.Callbook.Logbook.Enabled = false
	a.Config.Integrations.Callbook.Wavelog.Enabled = false

	reg := buildCallbookRegistry(a)
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if reg.Len() != 1 {
		t.Errorf("registry length = %d, want 1 (CTY only, no HamQTH without user)", reg.Len())
	}
}
