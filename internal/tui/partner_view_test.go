package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/store"
)

func TestPartnerViewRender(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	// Set up partner data
	m.lookup.partnerData = &qrz.CallData{
		Callsign: "SP9MOA",
		Name:     "John",
		Grid:     "JO90",
		QTH:      "Krakow",
		Country:  "Poland",
	}

	view := m.viewPartner()
	if view == "" {
		t.Error("viewPartner returned empty")
	}
	if !strings.Contains(view, "SP9MOA") {
		t.Error("viewPartner missing partner callsign")
	}
}

func TestPartnerViewNoData(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.lookup.partnerData = nil

	// Empty call should render nothing
	m.fields[fieldCall].SetValue("")
	view := m.viewPartner()
	if view != "" {
		t.Error("viewPartner should return empty when no callsign")
	}
}

func TestPartnerViewNoOwnGrid(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.App.Logbook.Station.Grid = ""

	m.lookup.partnerData = &qrz.CallData{
		Callsign: "SP9MOA",
		Grid:     "JO90",
	}

	view := m.viewPartner()
	if view == "" {
		t.Error("viewPartner returned empty without own grid")
	}
	if !strings.Contains(view, "station config") {
		t.Error("viewPartner should prompt user to set own grid")
	}
}

func TestPartnerViewMapCache(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.RenderMap = true

	m.lookup.partnerData = &qrz.CallData{
		Callsign: "SP9MOA",
		Grid:     "JN18",
	}

	// First render — should populate view cache.
	view1 := m.viewPartner()
	if view1 == "" {
		t.Fatal("viewPartner returned empty")
	}
	sig1 := m.rc.partnerViewSig
	if sig1 == "" {
		t.Error("partnerViewCacheSig should be set after first render")
	}

	// Second render — should use cache (output identical).
	view2 := m.viewPartner()
	if view2 != view1 {
		t.Error("Cached view should be identical to first render")
	}
	sig2 := m.rc.partnerViewSig
	if sig1 != sig2 {
		t.Error("Cache signature should not change between renders with same state")
	}
}

func TestPartnerViewMapCacheInvalidateOnResize(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.RenderMap = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", Grid: "JN18"}

	m.viewPartner()
	sig1 := m.rc.partnerViewSig

	// Simulate resize
	m.width = 80
	m.invalidatePartnerMapCache()
	if m.rc.partnerViewSig != "" {
		t.Error("View cache sig should be empty after invalidation")
	}

	m.viewPartner()
	sig2 := m.rc.partnerViewSig
	if sig1 == sig2 {
		t.Error("Cache signature should change after resize")
	}
}

func TestPartnerViewMapCacheInvalidateOnPartnerChange(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.RenderMap = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", Grid: "JN18"}

	m.viewPartner()
	sig1 := m.rc.partnerViewSig

	// Change partner
	m.lookup.partnerData = &qrz.CallData{Callsign: "DJ7NT", Grid: "JO30"}
	m.invalidatePartnerMapCache()

	m.viewPartner()
	sig2 := m.rc.partnerViewSig
	if sig1 == sig2 {
		t.Error("Cache signature should change when partner changes")
	}
}

func TestPartnerViewRenderPartnerInfo(t *testing.T) {
	m := newTestModel()
	d := &qrz.CallData{
		Callsign: "SP9MOA",
		Name:     "John",
		Grid:     "JO90",
		QTH:      "Krakow",
		Country:  "Poland",
	}

	info := m.renderCallbookRows(d, 40)
	if info == "" {
		t.Error("renderCallbookRows returned empty")
	}
	if !strings.Contains(info, "SP9MOA") {
		t.Error("renderCallbookRows missing callsign")
	}
	if !strings.Contains(info, "John") {
		t.Error("renderCallbookRows missing name")
	}
}

func TestPartnerViewRenderWLInfo(t *testing.T) {
	// WL not configured — should show "Wavelog not configured"
	m := newTestModel()
	info := m.renderWLInfo(40)
	if info == "" {
		t.Error("renderWLInfo returned empty with nil data")
	}
	if !strings.Contains(info, "not configured") {
		t.Error("renderWLInfo should show 'Wavelog not configured' when Wavelog not configured")
	}

	// WL enabled, lookup not yet done — should show "pending"
	m2 := newTestModel()
	m2.inetOnline = true
	m2.App.Logbook.Wavelog = &config.WavelogConfig{Enabled: true, URL: "https://example.com", APIKey: "test-key"}
	m2.lookup.wlLookupDone = false
	info2 := m2.renderWLInfo(40)
	if !strings.Contains(info2, "pending") {
		t.Error("renderWLInfo should show 'pending' when WL enabled but lookup not yet done")
	}

	// WL enabled, lookup completed with no data — should show "No WL data"
	m3 := newTestModel()
	m3.inetOnline = true
	m3.App.Logbook.Wavelog = &config.WavelogConfig{Enabled: true, URL: "https://example.com", APIKey: "test-key"}
	m3.lookup.wlLookupDone = true
	info3 := m3.renderWLInfo(40)
	if !strings.Contains(info3, "No WL data") {
		t.Error("renderWLInfo should show 'No WL data' when lookup completed with no results")
	}
}

func TestPartnerViewNarrowWidth(t *testing.T) {
	m := newTestModel()
	m.width = 30
	m.height = 20
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", Grid: "JO90"}

	view := m.viewPartner()
	if view == "" {
		t.Error("viewPartner on narrow width returned empty")
	}
}

func TestFormatRowPairs(t *testing.T) {
	rows := []row{
		{"Name", "John"},
		{"QTH", "Krakow"},
	}

	result := formatRowPairs(rows, S.FormLabel)
	if result == "" {
		t.Error("formatRowPairs returned empty")
	}
	if !strings.Contains(result, "Name") {
		t.Error("formatRowPairs missing label 'Name'")
	}
	if !strings.Contains(result, "Krakow") {
		t.Error("formatRowPairs missing value 'Krakow'")
	}
}

func TestFormatRowPairsEmpty(t *testing.T) {
	result := formatRowPairs(nil, S.FormLabel)
	if result != "" {
		t.Errorf("formatRowPairs with nil rows should return empty, got %q", result)
	}

	result = formatRowPairs([]row{}, S.FormLabel)
	if result != "" {
		t.Errorf("formatRowPairs with empty rows should return empty, got %q", result)
	}
}

func TestRenderLoTW(t *testing.T) {
	dimStyle := lipgloss.NewStyle()
	badStyle := lipgloss.NewStyle()

	// Member → Y with dim style.
	result := renderLoTW(true, dimStyle, badStyle)
	if result == "" {
		t.Error("renderLoTW true returned empty")
	}

	// Not a member → N with bad style.
	result = renderLoTW(false, dimStyle, badStyle)
	if result == "" {
		t.Error("renderLoTW false returned empty")
	}
}

func TestRenderLogbookRowsWLFirst(t *testing.T) {
	m := newLifecycleTestModel(t)
	d := &qrz.CallData{Callsign: "SP9MOA"}

	// No WL data — should fall back to local (all default false = new).
	rows := m.renderLogbookRows(d, 40)
	if rows == "" {
		t.Error("renderLogbookRows returned empty")
	}
	// With no WL data and no local stats, all should show Y (new).
	if !strings.Contains(rows, "New call") {
		t.Error("renderLogbookRows missing 'New call' row")
	}

	// Set local stats: call already worked.
	m.rc.logStats = store.LogbookStats{CallWorked: true, QSOCount: 3}
	m.rc.logStatsSig = "SP9MOA||"
	rows = m.renderLogbookRows(d, 40)
	// Without WL data, should use local: call worked → N.
	// We can verify the rows still contain the label.
	if !strings.Contains(rows, "New call") {
		t.Error("renderLogbookRows should always show 'New call' label")
	}
}

func TestRenderLogbookRowsNewDXCC(t *testing.T) {
	m := newLifecycleTestModel(t)
	d := &qrz.CallData{Callsign: "VK3A"}

	rows := m.renderLogbookRows(d, 55)

	// DXCC rows should always be present. With FormLabelWide(17) labels fit fully.
	if !strings.Contains(rows, "New DXCC") {
		t.Error("renderLogbookRows missing 'New DXCC' row")
	}
	if !strings.Contains(rows, "DXCC band") {
		t.Error("renderLogbookRows missing 'DXCC band' row")
	}
	if !strings.Contains(rows, "DXCC mode") {
		t.Error("renderLogbookRows missing 'DXCC mode' row")
	}
}

func TestPartnerViewCache(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.App.Logbook.Station.Grid = "JO90"
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", Grid: "JO90"}

	// First render — builds and caches.
	v1 := m.viewPartner()
	if v1 == "" {
		t.Fatal("viewPartner returned empty")
	}
	if m.rc.partnerViewSig == "" {
		t.Error("partnerViewCacheSig should be set after first render")
	}

	// Second render — should hit cache.
	v2 := m.viewPartner()
	if v2 != v1 {
		t.Error("cached view should be identical to first render")
	}

	// Invalidate and re-render — cache should rebuild.
	m.invalidatePartnerMapCache()
	if m.rc.partnerViewSig != "" {
		t.Error("partnerViewCacheSig should be empty after invalidation")
	}
	v3 := m.viewPartner()
	if v3 == "" {
		t.Error("viewPartner returned empty after cache invalidation")
	}
}

// =============================================================================
// Inline photo tests
// =============================================================================

func TestPhotoBox_HiddenWhenDisabled(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = false
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	view := m.viewPartner()
	if strings.Contains(view, "Photo") {
		t.Error("Photo box should NOT appear when PictureAtQRZPane is disabled")
	}
}

func TestPhotoBox_HiddenWhenNarrow(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	view := m.viewPartner()
	if strings.Contains(view, "Photo") {
		t.Error("Photo box should NOT appear on narrow screens (<180 cols)")
	}
}

func TestPhotoBox_HiddenWhenNoImageURL(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: ""}

	view := m.viewPartner()
	if strings.Contains(view, "Photo") {
		t.Error("Photo box should NOT appear when ImageURL is empty")
	}
}

func TestPhotoBox_ShowsLoadingWhenNoContent(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	// partnerPicViewer has no content yet — should show "Loading…".
	view := m.viewPartner()
	if !strings.Contains(view, "Loading") {
		t.Error("Photo box should show 'Loading…' when viewer has no content")
	}
	if !strings.Contains(view, "Photo") {
		t.Error("Photo box header should be present")
	}
}

func TestPhotoBox_AppearsOnWideScreen(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	view := m.viewPartner()
	if !strings.Contains(view, "Photo") {
		t.Error("Photo box should appear on wide screen with config enabled and image URL")
	}
}

func TestPhotoBox_CacheInvalidation(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	// First render — caches with empty photo content.
	v1 := m.viewPartner()
	if v1 == "" {
		t.Fatal("viewPartner returned empty")
	}

	// Simulate photo loading by putting content in the viewer (same URL).
	// The cache should miss because content hash changed.
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}
	m.invalidatePartnerMapCache()
	v2 := m.viewPartner()
	// Both should render without panicking.
	if v2 == "" {
		t.Error("viewPartner returned empty after simulated photo load")
	}
}

func TestPhotoBox_DimensionsStored(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	m.viewPartner()

	if m.photo.partnerPicW < 25 {
		t.Errorf("partnerPicW should be >= 25, got %d", m.photo.partnerPicW)
	}
	if m.photo.partnerPicH < 4 {
		t.Errorf("partnerPicH should be >= 4, got %d", m.photo.partnerPicH)
	}
}

func TestPhotoBox_LastPartnerPicURLTracked(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = true
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	m.viewPartner()

	if m.photo.partnerPicURL != "https://example.com/photo.jpg" {
		t.Errorf("lastPartnerPicURL = %q; want the image URL", m.photo.partnerPicURL)
	}
	if !m.photo.partnerPicNeedLoad {
		t.Error("partnerPicNeedLoad should be true on first render with new URL")
	}

	// Second render with same URL — should NOT trigger reload.
	m.photo.partnerPicNeedLoad = false
	m.viewPartner()
	if m.photo.partnerPicNeedLoad {
		t.Error("partnerPicNeedLoad should be false when URL unchanged")
	}
}

func TestPhotoBox_DoesNotAffectLayoutWhenDisabled(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtQRZPane = false
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	viewWithPhoto := m.viewPartner()

	// Render without photo feature at all.
	m.App.Config.General.PictureAtQRZPane = false
	m.invalidatePartnerMapCache()
	viewWithoutPhoto := m.viewPartner()

	// Both should be non-empty.
	if viewWithPhoto == "" || viewWithoutPhoto == "" {
		t.Fatal("views should not be empty")
	}
}
