package tui

import (
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/ref"
)

// =============================================================================
// Offline-switch network audit — regression tests for every network gate that
// previously checked only m.inetOnline (which stays true in offline mode).
// =============================================================================

// TestOfflineSolarDoesNotFetch: the Solar panel must not query hamqsl.com.
func TestOfflineSolarDoesNotFetch(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.Offline = true
	m.inetOnline = true
	m.App.Config.General.SolarAtQSOPane = true
	m.solar.cacheDir = t.TempDir() // no cached data

	if c := m.maybeFetchSolar(); c != nil {
		t.Fatalf("offline maybeFetchSolar returned a command: %v", c)
	}
	if m.solar.fetching {
		t.Fatal("offline solar fetch flag must stay false")
	}
}

// TestOfflineDataFilesDoNotRefresh: CTY/SCP/REF downloads must not start.
func TestOfflineDataFilesDoNotRefresh(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.Offline = true
	m.inetOnline = true

	if c := m.maybeRefreshDataFiles(); c != nil {
		t.Fatalf("offline maybeRefreshDataFiles returned a command: %v", c)
	}
}

// TestOfflineRefRebuildDoesNotStart: the REF rebuild downloads CSVs and must
// not run offline; the online path is untouched.
func TestOfflineRefRebuildDoesNotStart(t *testing.T) {
	m := newLifecycleTestModel(t)
	rdb, err := ref.Open(filepath.Join(t.TempDir(), "ref.db"))
	if err != nil {
		t.Fatalf("ref.Open: %v", err)
	}
	t.Cleanup(func() { rdb.Close() })
	m.App.RefDB = rdb

	m.Offline = true
	m.inetOnline = true
	if c := m.startRefRebuildCmd(); c != nil {
		t.Fatalf("offline startRefRebuildCmd returned a command: %v", c)
	}
	if m.ref.building {
		t.Fatal("offline REF rebuild must not start")
	}

	// Online the empty database triggers a rebuild command.
	m.Offline = false
	if c := m.startRefRebuildCmd(); c == nil || !m.ref.building {
		t.Fatalf("online rebuild should start: cmd=%v building=%v", c, m.ref.building)
	}
}

// TestOfflineMenusMirrorOfflineState: the integration and callbook menus
// mirror the internet state and must see offline as no-internet, so their
// Test Connection buttons refuse to fire.
func TestOfflineMenusMirrorOfflineState(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.Offline = true
	m.inetOnline = true
	m.ui.integrationMenu = NewIntegrationMenu(m.App.Config)
	m.ui.callbookMenu = NewCallbookMenu(m.App.Config)

	m.handleIntegrationUpdate(nil, nil)
	m.handleCallbookUpdate(nil, nil)

	if m.ui.integrationMenu.inetOnline {
		t.Error("integration menu must see offline as no-internet")
	}
	if m.ui.callbookMenu.inetOnline {
		t.Error("callbook menu must see offline as no-internet")
	}
}
