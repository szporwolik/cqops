package dashboard

import (
	"strings"
	"testing"
)

// TestMapFitsTheWorkingQsoPath pins the map-focus rule for an active contact.
//
// The defect it guards (reported from a live log): the now-working path was
// drawn but ran off the right edge of the map, so the partner was invisible.
// focusMapOnGrid did frame the whole path, but the next snapshot pushed while a
// contact was being worked found activeGrid set, skipped its own fit and fell
// through to "flyTo(station, 6)" — a station-centred zoom that puts a partner
// more than a few hundred kilometres away outside the viewport. The smaller the
// map (narrow window, split view), the more of the path was lost.
//
// The rule now: the active path owns the view. It is fitted whenever any part
// of it is off screen, and skipped only while it is fully visible, so a manual
// zoom or pan survives the next snapshot.
func TestMapFitsTheWorkingQsoPath(t *testing.T) {
	js, err := staticFS.ReadFile("static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	appJS := string(js)

	for _, want := range []string{
		// the path's own bounds, not just the station
		"activeBounds=L.latLngBounds([[ownStationLat,ownStationLon],al]).extend(actPts[24])",
		// fitted only when it does not already fit
		"if(!map.getBounds().contains(activeBounds))fitActiveBounds(activeBounds)",
		// focusMapOnGrid and the snapshot path share one flight helper
		"fitActiveBounds(b,{padding:[60,60],maxZoom:17,duration:2.5})",
		// guard cleared when the map settles, so a wrong view self-corrects
		"map.once('moveend'",
	} {
		if !strings.Contains(appJS, want) {
			t.Errorf("app.js is missing %q", want)
		}
	}

	// The regression itself: dropping the fit whenever an active QSO exists,
	// which lets the station-centred zoom below take over.
	if strings.Contains(appJS, "bounds.length>1&&!activeGrid") {
		t.Error("the active-QSO branch skips the fit again — a station-centred zoom leaves the partner off screen")
	}
	// The whole-day overview still has to work when no contact is being worked.
	if !strings.Contains(appJS, "else if(bounds.length>1)map.flyToBounds(bounds,{padding:[50,50],maxZoom:18})") {
		t.Error("the overview fit over the day's markers is gone")
	}
}

// TestQsoLoggedRedrawsTheMapOnce: a saved QSO used to rebuild the map twice —
// once from the handler, then again from switchToOverview — recreating up to
// 200 markers and 150 great-circle paths each time before the next paint. The
// handler sets the view state first (overviewState, which does not draw) and
// redraws once.
func TestQsoLoggedRedrawsTheMapOnce(t *testing.T) {
	js, err := staticFS.ReadFile("static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	appJS := string(js)

	if strings.Contains(appJS, "updateMapFromToday();switchToOverview()") {
		t.Error("a saved QSO rebuilds the map twice again")
	}
	for _, want := range []string{
		"function overviewState(){D('state','overview');setState(false);activeGrid=null}",
		"function switchToOverview(){overviewState();updateMapFromToday()}",
		"appendTodayQSO(q);prependRecentRow(q);overviewState();updateMapFromToday();",
	} {
		if !strings.Contains(appJS, want) {
			t.Errorf("app.js is missing %q", want)
		}
	}
}

// TestOwnStationIsTheAprsSymbol pins the own-station marker: while the beacon
// is on, the station is drawn as the APRS symbol CQOps transmits, not as a
// green dot with the symbol stuck underneath it. The anchor sits at the symbol
// centre so switching the beacon on does not move the station, and the marker
// is layered above the neighbouring APRS stations — on the local map it used to
// live in the shadow pane, i.e. behind exactly the stations it hears.
func TestOwnStationIsTheAprsSymbol(t *testing.T) {
	js, err := staticFS.ReadFile("static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	css, err := staticFS.ReadFile("static/style.css")
	if err != nil {
		t.Fatalf("read style.css: %v", err)
	}
	appJS, styleCSS := string(js), string(css)

	if strings.Contains(appJS, "own-aprs-dot") || strings.Contains(styleCSS, "own-aprs-dot") {
		t.Error("the own station still draws a dot in front of its APRS symbol")
	}
	if !strings.Contains(appJS, "iconAnchor:[55,12]") {
		t.Error("the APRS symbol must be anchored at its centre, or the station jumps when the beacon comes on")
	}
	if !strings.Contains(appJS, "zIndexOffset:100000") {
		t.Error("the own symbol must outrank the neighbouring APRS markers")
	}
	if strings.Contains(appJS, "if(small)opts.pane='shadowPane'") {
		t.Error("the beaconing marker is back in the shadow pane, i.e. behind the APRS stations")
	}
	// Receive-only mode keeps the plain dot, and it still pulses while active.
	if !strings.Contains(appJS, "className:'station-dot'") || !strings.Contains(styleCSS, ".mode-active .own-aprs-sym{animation:") {
		t.Error("the receive-only dot or the beaconing pulse is missing")
	}
}
