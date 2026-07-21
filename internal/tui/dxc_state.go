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
	cachedRaw    []store.DXCSpot // raw unfiltered spots; new spots appended here
	rawGen       int             // incremented on every cachedRaw change; busts dxcPathLine cache

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
	cachedFilterSig  string

	// Path line state — caches matched spots for Ctrl+P cycling.
	pathSpots   []store.DXCSpot // spots at/near current frequency (from dxcPathLine)
	pathSpotIdx int             // cycling index for Ctrl+P

	// Dupe set — computed once per table rebuild, keyed by "CALL|BAND|MODE".
	// Checked per spot row to dim already-worked calls. Invalidated when
	// the logbook or contest changes (cache key stored alongside the set).
	dupeSet        map[string]bool
	dupeSetLogbook string // logbook name at time of computation
	dupeSetContest string // contest ID at time of computation

	// DXCC worked sets for spot highlighting (new DXCC / new band / new mode).
	dxccBandSet     map[string]bool // "230|20m"
	dxccBandModeSet map[string]bool // "230|20m|FT8"
	dxccSetLogbook  string          // logbook name at time of computation
}
