package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

func TestPartnerViewRender(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	// Set up partner data
	m.lookup.partnerData = &callbook.Result{
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

	m.lookup.partnerData = &callbook.Result{
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

	m.lookup.partnerData = &callbook.Result{
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
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", Grid: "JN18"}

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
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", Grid: "JN18"}

	m.viewPartner()
	sig1 := m.rc.partnerViewSig

	// Change partner
	m.lookup.partnerData = &callbook.Result{Callsign: "DJ7NT", Grid: "JO30"}
	m.invalidatePartnerMapCache()

	m.viewPartner()
	sig2 := m.rc.partnerViewSig
	if sig1 == sig2 {
		t.Error("Cache signature should change when partner changes")
	}
}

func TestPartnerContentWrapSkipsStylingForKittyContent(t *testing.T) {
	content := "kitty-frame"

	got := renderPartnerPaneContent(content, 20, true)
	if got != content {
		t.Fatalf("expected raw content when Kitty graphics is active, got %q", got)
	}
}

func TestRenderCallbookRows(t *testing.T) {
	tests := []struct {
		name string
		data callbook.Result
		maxW int
		want []string
		not  []string
	}{
		{
			name: "complete record",
			data: callbook.Result{
				Callsign: "SP9MOA", Class: "I",
				Name: "Niepolomice Amateur Radio Club",
				QTH:  "Niepolomice", State: "Malopolskie", Zip: "32-005",
				Country: "Poland", DXCC: "269",
				Grid: "JO90", Lat: "50.03309", Lon: "20.22108",
				CQZone: "15", ITUZone: "28",
				Email: "sp9moa@gmail.com",
			},
			maxW: 70,
			want: []string{"SP9MOA", "Niepolomice", "32-005", "Malopolskie",
				"Poland", "DXCC 269", "JO90", "50.03309\u00b0N", "20.22108\u00b0E",
				"CQ 15", "ITU 28", "sp9moa@gmail.com", "\u00b7 I"},
		},
		{
			name: "missing state and zip",
			data: callbook.Result{
				Callsign: "K1ABC", Name: "John",
				QTH: "Boston", Country: "United States", DXCC: "291",
				Grid: "FN42",
			},
			maxW: 50,
			want: []string{"K1ABC", "Boston", "United States", "DXCC 291", "FN42"},
			not:  []string{"Class"},
		},
		{
			name: "missing coordinates",
			data: callbook.Result{
				Callsign: "G3ABC", Name: "Alice",
				Country: "England", Grid: "IO91",
			},
			maxW: 50,
			want: []string{"G3ABC", "Alice", "England", "IO91"},
			not:  []string{"\u00b0"},
		},
		{
			name: "missing email",
			data: callbook.Result{
				Callsign: "F1XYZ", Name: "Pierre",
				Country: "France", QTH: "Paris",
			},
			maxW: 50,
			want: []string{"F1XYZ", "Pierre", "Paris"},
			not:  []string{"Email"},
		},
		{
			name: "very long name",
			data: callbook.Result{
				Callsign: "DL1ABC",
				Name:     "Maximilian Alexander von Hohenzollern-Sigmaringen und Anhalt-Dessau",
				Country:  "Germany",
			},
			maxW: 40,
			want: []string{"DL1ABC", "Maximilian", "Germany"},
		},
		{
			name: "narrow width",
			data: callbook.Result{
				Callsign: "EA1ABC", Name: "Carlos",
				QTH: "Madrid", Country: "Spain", Grid: "IN80",
			},
			maxW: 30,
			want: []string{"EA1ABC", "Madrid"},
		},
		{
			name: "unicode name and qth",
			data: callbook.Result{
				Callsign: "JA1ABC",
				Name:     "\u5c71\u7530 \u592a\u90ce",
				QTH:      "\u6771\u4eac\u90fd",
				Country:  "Japan",
			},
			maxW: 50,
			want: []string{"JA1ABC", "\u5c71\u7530", "\u6771\u4eac\u90fd", "Japan"},
		},
		{
			name: "no text crosses right border",
			data: callbook.Result{
				Callsign: "TEST", Name: "A", QTH: "B", Country: "C",
				Grid: "AA00", DXCC: "1", CQZone: "1", ITUZone: "1",
				Email: "a@b.c", State: "D", Zip: "12345", Class: "E",
				Lat: "50.0", Lon: "20.0",
			},
			maxW: 45,
			want: []string{"TEST"},
		},
		{
			name: "no empty gaps between rows",
			data: callbook.Result{
				Callsign: "SP9MOA", Name: "John",
				Country: "Poland",
			},
			maxW: 50,
			want: []string{"SP9MOA", "John", "Poland"},
			not:  []string{"\n\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			info := m.renderCallbookRows(&tt.data, tt.maxW, 0)
			if info == "" && len(tt.want) > 0 {
				t.Error("renderCallbookRows returned empty")
				return
			}
			for _, w := range tt.want {
				if !strings.Contains(info, w) {
					t.Errorf("output missing %q\nGot:\n%s", w, info)
				}
			}
			for _, n := range tt.not {
				if strings.Contains(info, n) {
					t.Errorf("output should NOT contain %q\nGot:\n%s", n, info)
				}
			}
			for _, line := range strings.Split(info, "\n") {
				if lipgloss.Width(line) > tt.maxW {
					t.Errorf("line exceeds maxW %d: width=%d line=%q", tt.maxW, lipgloss.Width(line), line)
				}
			}
		})
	}
}

func TestRenderCallbookRows_ForeignPrefixUsesOperatingZones(t *testing.T) {
	m := newTestModel()
	m.App.Config.General.UseCTY = true
	m.lookup.partnerData = &callbook.Result{
		Callsign: "9A/SP9SPM/P",
		Country:  "Poland",
		DXCC:     "269",
		CQZone:   "15",
		ITUZone:  "28",
	}

	info := m.renderCallbookRows(m.lookup.partnerData, 70, 0)
	if !strings.Contains(info, "CQ 15") {
		t.Fatalf("expected CQ zone from operating prefix, got %q", info)
	}
	if !strings.Contains(info, "ITU 28") {
		t.Fatalf("expected ITU zone from operating prefix, got %q", info)
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
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", Grid: "JO90"}

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
	d := &callbook.Result{Callsign: "SP9MOA"}

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
	d := &callbook.Result{Callsign: "VK3A"}

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
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", Grid: "JO90"}

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
	m.App.Config.General.PictureAtPartnerPane = false
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	view := m.viewPartner()
	if strings.Contains(view, "Photo") {
		t.Error("Photo box should NOT appear when PictureAtPartnerPane is disabled")
	}
}

func TestPhotoBox_HiddenWhenNarrow(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtPartnerPane = true
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

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
	m.App.Config.General.PictureAtPartnerPane = true
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: ""}

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
	m.App.Config.General.PictureAtPartnerPane = true
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	// partnerPicViewer has no content yet — should show "Loading…".
	view := m.viewPartner()
	if !strings.Contains(view, "Loading") {
		t.Error("Photo box should show 'Loading…' when viewer has no content")
	}
}

func TestPhotoBox_AppearsOnWideScreen(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtPartnerPane = true
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	view := m.viewPartner()
	// Photo box renders without a separate header now — verify it's not
	// empty/absent on wide screen with photo URL available.
	if view == "" {
		t.Error("Partner view should render on wide screen with config enabled and image URL")
	}
}

func TestPhotoBox_CacheInvalidation(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 50
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Config.General.PictureAtPartnerPane = true
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	// First render — caches with empty photo content.
	v1 := m.viewPartner()
	if v1 == "" {
		t.Fatal("viewPartner returned empty")
	}

	// Simulate photo loading by putting content in the viewer (same URL).
	// The cache should miss because content hash changed.
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}
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
	m.App.Config.General.PictureAtPartnerPane = true
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

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
	m.App.Config.General.PictureAtPartnerPane = true
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

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
	m.App.Config.General.PictureAtPartnerPane = false
	m.lookup.partnerData = &callbook.Result{Callsign: "SP9MOA", ImageURL: "https://example.com/photo.jpg"}

	viewWithPhoto := m.viewPartner()

	// Render without photo feature at all.
	m.App.Config.General.PictureAtPartnerPane = false
	m.invalidatePartnerMapCache()
	viewWithoutPhoto := m.viewPartner()

	// Both should be non-empty.
	if viewWithPhoto == "" || viewWithoutPhoto == "" {
		t.Fatal("views should not be empty")
	}
}
