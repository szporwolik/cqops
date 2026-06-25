package tui

import (
	"time"

	"charm.land/bubbles/v2/table"
	"github.com/szporwolik/cqops/internal/dxc"
	"github.com/szporwolik/cqops/internal/store"
)

// dxcState holds all DX Cluster connection, table, filter, and selection state.
type dxcState struct {
	client       *dxc.Client
	online       bool
	connecting   bool
	lastAttempt  time.Time
	reconnectIdx int
	lastPurge    time.Time
	lastDrain    time.Time // last time drainDXCSpots was called; throttle to 4s

	table      table.Model
	tableReady bool
	builtW     int
	builtH     int
	spotCount  int

	bandFilter   string          // "" = all, band name = filter, "other" = unclassified
	timeFilter   int             // minutes, 0 = all
	timeIdx      int             // index into dxcTimeWindows
	bandIdx      int             // index into dxcBandChoices
	contFilter   string          // "" = all, 2-letter continent code = filter
	contIdx      int             // index into continent choices
	modeFilter   string          // "" = all, CW/DIGI/PHONE = filter
	modeIdx      int             // index into mode choices
	selectedCall string          // callsign of the currently highlighted spot
	selectedSpot store.DXCSpot   // full spot data captured at cursor-move time
	tuneCancel   func()          // cancel previous tune command if running
	cachedSpots  []store.DXCSpot // cached result of last filteredSpots() call

	// Filter state at time of cache — used to detect staleness.
	cachedBandFilter string
	cachedTimeFilter int
	cachedContFilter string
	cachedModeFilter string
	cachedSortBand   string // band filter active when last sorted; avoids redundant sort

	// Band/continent cache — avoids DB query on every filter-cycle keypress.
	cachedBands []string
	cachedConts []string

	// Cross-cutting lookup request flags.
	need bool   // re-trigger DXC freq lookup after live spot arrives
	call string // callsign for pending DXC lookup

	// Render cache for filter info line — rebuilt only when filters change.
	cachedFilterInfo string
	cachedFilterW    int
}
