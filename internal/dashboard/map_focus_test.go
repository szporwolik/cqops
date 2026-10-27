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
