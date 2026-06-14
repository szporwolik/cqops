package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/qrz"
)

func TestPartnerViewRender(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	// Set up partner data
	m.partnerData = &qrz.CallData{
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
	m.partnerData = nil

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

	m.partnerData = &qrz.CallData{
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

	m.partnerData = &qrz.CallData{
		Callsign: "SP9MOA",
		Grid:     "JN18",
	}

	// First render — should populate cache
	view1 := m.viewPartner()
	if view1 == "" {
		t.Fatal("viewPartner returned empty")
	}
	cached := m.partnerMapCache
	sig1 := m.partnerMapCacheSig
	if cached == "" {
		t.Error("Map cache should be populated after first render")
	}

	// Second render — should use cache (output identical)
	view2 := m.viewPartner()
	if view2 != view1 {
		t.Error("Cached view should be identical to first render")
	}
	sig2 := m.partnerMapCacheSig
	if sig1 != sig2 {
		t.Error("Cache signature should not change between renders with same state")
	}
}

func TestPartnerViewMapCacheInvalidateOnResize(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.App.Logbook.Station.Grid = "JO90"
	m.partnerData = &qrz.CallData{Callsign: "SP9MOA", Grid: "JN18"}

	m.viewPartner()
	sig1 := m.partnerMapCacheSig

	// Simulate resize
	m.width = 80
	m.invalidatePartnerMapCache()
	if m.partnerMapCache != "" {
		t.Error("Cache should be empty after invalidation")
	}

	m.viewPartner()
	sig2 := m.partnerMapCacheSig
	if sig1 == sig2 {
		t.Error("Cache signature should change after resize")
	}
}

func TestPartnerViewMapCacheInvalidateOnPartnerChange(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.App.Logbook.Station.Grid = "JO90"
	m.partnerData = &qrz.CallData{Callsign: "SP9MOA", Grid: "JN18"}

	m.viewPartner()
	sig1 := m.partnerMapCacheSig

	// Change partner
	m.partnerData = &qrz.CallData{Callsign: "DJ7NT", Grid: "JO30"}
	m.invalidatePartnerMapCache()

	m.viewPartner()
	sig2 := m.partnerMapCacheSig
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

	info := m.renderPartnerInfo(d, 40)
	if info == "" {
		t.Error("renderPartnerInfo returned empty")
	}
	if !strings.Contains(info, "SP9MOA") {
		t.Error("renderPartnerInfo missing callsign")
	}
	if !strings.Contains(info, "John") {
		t.Error("renderPartnerInfo missing name")
	}
}

func TestPartnerViewRenderWLInfo(t *testing.T) {
	m := newTestModel()

	// With nil data — should show pending
	info := m.renderWLInfo(40)
	if info == "" {
		t.Error("renderWLInfo returned empty with nil data")
	}
	if !strings.Contains(info, "pending") {
		t.Error("renderWLInfo should show pending with nil data")
	}
}

func TestPartnerViewNarrowWidth(t *testing.T) {
	m := newTestModel()
	m.width = 30
	m.height = 20
	m.partnerData = &qrz.CallData{Callsign: "SP9MOA", Grid: "JO90"}

	view := m.viewPartner()
	if view == "" {
		t.Error("viewPartner on narrow width returned empty")
	}
}
