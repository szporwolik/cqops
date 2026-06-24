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
	barOp     string // active operator ID; busts status cache on change
	barLog    string // active logbook ID; busts status cache on change
	barRig    string // active rig ID; busts status cache on change

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

	// Async fetch state — avoids DB queries during View().
	logStatsNeedFetch bool
	logStatsFetchCall string
	logStatsFetchBand string
	logStatsFetchMode string

	// DXCC continent cache — avoids prefix-tree lookup on every partner-view frame.
	dxccContCall  string
	dxccContValue string

	// Per-frame view caches.
	formView string
	formSig  string
	formSec  int // second of last form cache; busts at 1 Hz for clock fields
	tabView  string
	tabSig   string
	helpView string
	helpSig  string

	// Path state (committed call/grid, updated on field exit).
	pathCall string
	pathGrid string
}
