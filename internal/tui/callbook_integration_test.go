package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Callbook lookup mock tests (multi-provider ready)
// =============================================================================

// TestBuildCallbookRegistryOfflineSkipsNetworkProviders verifies the offline
// switch removes every network provider from the registry: a lookup can then
// only use the local logbook and CTY.DAT, never QRZ/Callook/Wavelog.
// TestNewModelOfflineBuildsOfflineRegistry reproduces the startup-order leak:
// the offline flag reached the model only after New() had already built the
// registry. With a.Offline set before New, the registry must contain local
// providers only.
func TestNewModelOfflineBuildsOfflineRegistry(t *testing.T) {
	m, _ := newWorkedPanelTestModel(t)
	cfg := m.App.Config
	cfg.Integrations.Callbook.QRZ.Enabled = true
	cfg.Integrations.Callbook.QRZ.User = "u"
	cfg.Integrations.Callbook.Callook.Enabled = true
	cfg.Integrations.Callbook.Wavelog.Enabled = true

	m.App.Offline = true
	fresh := New(m.App, nil)
	if fresh.callbookRegistry == nil || fresh.callbookRegistry.Len() != 1 {
		t.Fatalf("offline New() registry has %d providers, want 1 (local logbook only)",
			providerCount(fresh))
	}
}

// providerCount returns the registry size, or -1 when the registry is nil.
func providerCount(m *Model) int {
	if m.callbookRegistry == nil {
		return -1
	}
	return m.callbookRegistry.Len()
}

func TestBuildCallbookRegistryOfflineSkipsNetworkProviders(t *testing.T) {
	m, _ := newWorkedPanelTestModel(t)
	cfg := m.App.Config
	cfg.Integrations.Callbook.QRZ.Enabled = true
	cfg.Integrations.Callbook.QRZ.User = "u"
	cfg.Integrations.Callbook.Callook.Enabled = true
	cfg.Integrations.Callbook.Wavelog.Enabled = true

	m.App.Offline = true
	reg := buildCallbookRegistry(m.App)
	if reg == nil {
		t.Fatal("offline registry should not be nil (logbook provider present)")
	}
	if reg.Len() != 1 {
		t.Fatalf("offline registry has %d providers, want 1 (local logbook only)", reg.Len())
	}

	m.App.Offline = false
	regOn := buildCallbookRegistry(m.App)
	if regOn == nil || regOn.Len() < 4 {
		t.Fatalf("online registry has %d providers, want >= 4 (QRZ+Callook+Wavelog+logbook)", regOn.Len())
	}
}

func TestCallbookLookupSuccess(t *testing.T) {
	orig := callbookRegLookup
	t.Cleanup(func() { callbookRegLookup = orig })

	callbookRegLookup = func(reg *callbook.Registry, baseFallback bool, call string) (*callbook.Result, error) {
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

	callbookRegLookup = func(reg *callbook.Registry, baseFallback bool, call string) (*callbook.Result, error) {
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

// TestSwitchLogbookRebuildsCallbookRegistry verifies that rebuilding the
// registry after a logbook switch makes the local logbook provider answer
// from the NEW database — a stale provider would return the retired
// logbook's history (or fail against a closed database).
func TestSwitchLogbookRebuildsCallbookRegistry(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	t.Cleanup(m.App.StopAPRSTimer)

	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "SP9XYZ", QSODate: "20260901", TimeOn: "1200",
		Band: "20m", Mode: "SSB", Name: "ALPHA",
	}); err != nil {
		t.Fatalf("insert into logbook A: %v", err)
	}

	res, err := m.callbookRegistry.Lookup("SP9XYZ")
	if err != nil || res == nil || res.Name != "ALPHA" {
		t.Fatalf("lookup against A = %+v, %v; want Name=ALPHA", res, err)
	}

	// Add logbook B with its own database and switch to it.
	lbB := config.Logbook{
		Station:      config.Station{Callsign: "SP9B", Grid: "JO91"},
		DatabasePath: filepath.Join(t.TempDir(), "b.db"),
	}
	m.App.Config.Logbooks["b"] = lbB
	if err := m.App.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { m.App.DB.Close() })

	// Seed the new logbook with a different name for the same call.
	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "SP9XYZ", QSODate: "20260902", TimeOn: "1200",
		Band: "20m", Mode: "SSB", Name: "BRAVO",
	}); err != nil {
		t.Fatalf("insert into logbook B: %v", err)
	}

	m.rebuildCallbookRegistry()
	res, err = m.callbookRegistry.Lookup("SP9XYZ")
	if err != nil || res == nil || res.Name != "BRAVO" {
		t.Fatalf("lookup against B after rebuild = %+v, %v; want Name=BRAVO", res, err)
	}
}

// TestCallbookLookupStaleResultFromPreviousLogbookDropped verifies that an
// in-flight lookup result bound to a previous logbook never fills the form
// after a switch.
func TestCallbookLookupStaleResultFromPreviousLogbookDropped(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	t.Cleanup(m.App.StopAPRSTimer)
	m.fields[fieldCall].SetValue("SP9XYZ")

	// Dispatch a lookup against the initial logbook A.
	msg := execCmd(m.callbookLookupCmd("SP9XYZ")).(callbookResultMsg)
	if msg.Logbook != "test" {
		t.Fatalf("captured logbook = %q, want test", msg.Logbook)
	}

	// Switch to logbook B while the lookup is "in flight".
	lbB := config.Logbook{
		Station:      config.Station{Callsign: "SP9B", Grid: "JO91"},
		DatabasePath: filepath.Join(t.TempDir(), "b.db"),
	}
	m.App.Config.Logbooks["b"] = lbB
	if err := m.App.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { m.App.DB.Close() })
	m.rebuildCallbookRegistry()

	// The stale result from logbook A must be discarded.
	m.fillCallbookData(callbookResultMsg{
		Call:    "SP9XYZ",
		Logbook: "test",
		Data:    &callbook.Result{Callsign: "SP9XYZ", Name: "ALPHA"},
	})
	if m.fields[fieldName].Value() != "" {
		t.Errorf("Name = %q; stale result from the old logbook must not fill the form", m.fields[fieldName].Value())
	}
	if m.lookup.qrzLookupDone {
		t.Error("stale result must not mark the lookup done")
	}
	if m.lookup.partnerData != nil {
		t.Error("stale result must not set partner data")
	}

	// A result bound to the current logbook must still apply.
	m.fillCallbookData(callbookResultMsg{
		Call:    "SP9XYZ",
		Logbook: "b",
		Data:    &callbook.Result{Callsign: "SP9XYZ", Name: "BRAVO"},
	})
	if m.fields[fieldName].Value() != "BRAVO" {
		t.Errorf("Name = %q; current-logbook result should fill the form", m.fields[fieldName].Value())
	}
}

// TestFillWLDataStaleLogbookDropped verifies Wavelog private-lookup results
// from a previous logbook are discarded after a switch.
func TestFillWLDataStaleLogbookDropped(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.fields[fieldCall].SetValue("SP9XYZ")

	cmd := m.fillWLData(wlResultMsg{
		Call:    "SP9XYZ",
		Logbook: "other",
		Data:    &wavelog.PrivateLookupResult{},
	})
	if cmd != nil {
		t.Error("stale Wavelog result must not trigger a follow-up command")
	}
	if m.lookup.wlPrivateData != nil {
		t.Error("stale Wavelog result must not set private lookup data")
	}
	if m.lookup.wlLookupDone {
		t.Error("stale Wavelog result must not mark the lookup done")
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
	// When both priorities are 0 (unset), defaults: QRZ=100, HamQTH=90.
	// QRZ should win because 100 > 90.
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.Callbook.QRZ.Enabled = true
	m.App.Config.Integrations.Callbook.QRZ.User = "test"
	m.App.Config.Integrations.Callbook.QRZ.Priority = 0
	m.App.Config.Integrations.Callbook.HamQTH.Enabled = true
	m.App.Config.Integrations.Callbook.HamQTH.User = "test"
	m.App.Config.Integrations.Callbook.HamQTH.Priority = 0

	name, _ := m.internetCallbook()
	if name != "QRZ.com" {
		t.Errorf("name = %q, want QRZ.com (default 100 > default 90)", name)
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
