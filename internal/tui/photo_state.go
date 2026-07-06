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

	partnerPicURL      string           // inline photo URL on Partner page
	partnerPicViewer   pictureurl.Model // inline photo viewer for Partner page (wide screens)
	partnerPicNeedLoad bool             // set when photo URL changes; consumed by Update
	partnerPicW        int              // photo box content width (computed in View, used in Update)
	partnerPicH        int              // photo box content height
	partnerPicLastW    int              // last SetSize width sent to viewer
	partnerPicLastH    int              // last SetSize height sent to viewer

	kittyToggled bool // true once viewers have been switched to Kitty mode
}

// ensureKitty toggles both viewers into Kitty mode once the probe
// resolves to Supported.  Toggle() silently no-ops until then.
// Pattern from ntcharts-lorem-picsum example.
func (ps *photoState) ensureKitty() tea.Cmd {
	if ps.kittyToggled {
		return nil
	}
	if ps.viewer.KittySupported() != picture.KittyCapabilitySupported {
		return nil
	}
	ps.kittyToggled = true
	return tea.Batch(ps.viewer.Toggle(), ps.partnerPicViewer.Toggle())
}
