package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/NimbleMarkets/ntcharts/v2/picture/pictureurl"
)

// photoState holds partner photo viewer state for both full-screen image view
// and inline photo on the Partner page.
type photoState struct {
	viewer  pictureurl.Model // full-screen image viewer
	lastErr error            // dedup image error logging
	lastURL string           // track photo URL to detect partner changes

	viewerLastW int // last SetSize width sent to full-screen viewer
	viewerLastH int // last SetSize height sent to full-screen viewer

	partnerPicURL      string           // inline photo URL on Partner page
	partnerPicViewer   pictureurl.Model // inline photo viewer for Partner page (wide screens)
	partnerPicNeedLoad bool             // set when photo URL changes; consumed by Update
	partnerPicW        int              // photo box content width (computed in View, used in Update)
	partnerPicH        int              // photo box content height
	partnerPicLastW    int              // last SetSize width sent to viewer (init 25)
	partnerPicLastH    int              // last SetSize height sent to viewer (init 4)
	partnerPicNeedSize bool             // force SetSize on next handlePartnerUpdate frame

	kittyToggled bool // true once viewers have been switched to Kitty mode
}

// ensureKitty toggles both viewers into Kitty mode once the probe
// resolves to Supported. Toggle() silently switches the internal
// mode even when it returns nil (no image set yet — renderCmd
// short-circuits on nil image). The mode change persists and
// takes effect when the next SetURL / SetImage arrives.
func (ps *photoState) ensureKitty(enabled bool) tea.Cmd {
	// Toggle back to Glyph when Kitty is disabled mid-run.
	if !enabled && ps.kittyToggled {
		ps.kittyToggled = false
		return tea.Batch(ps.viewer.Toggle(), ps.partnerPicViewer.Toggle())
	}
	if !enabled || ps.kittyToggled {
		return nil
	}
	if ps.viewer.KittySupported() != picture.KittyCapabilitySupported {
		return nil
	}
	ps.kittyToggled = true
	return tea.Batch(ps.viewer.Toggle(), ps.partnerPicViewer.Toggle())
}
