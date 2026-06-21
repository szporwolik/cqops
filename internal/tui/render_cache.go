package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// renderCache holds all per-frame view caches and signatures used to avoid
// redundant Lip Gloss rendering and layout computation on every frame.
type renderCache struct {
	// Layout cache — avoids redundant MeasureLayout() calls.
	lastLayout   Layout
	lastLayoutW  int
	lastLayoutH  int
	lastLayoutSc screenKind

	// Bar caches — avoids rebuilding status/tabs/help on every frame.
	// Status bar has a 1-second TTL because it contains the UTC clock.
	status    string
	statusSec int
	tabs      string
	help      string
	barSc     screenKind
	barW      int

	// Partner view cache.
	partnerView    string
	partnerViewSig string

	// Path line cache.
	pathLine string
	pathSig  string

	// Form column style cache.
	formColW         int
	formColStyle     lipgloss.Style
	formCommentStyle lipgloss.Style

	// Partner logbook stats cache.
	logStats    store.LogbookStats
	logStatsSig string

	// Per-frame view caches.
	formView string
	formSig  string
	tabView  string
	tabSig   string
	helpView string
	helpSig  string

	// Path state (committed call/grid, updated on field exit).
	pathCall string
	pathGrid string
}
