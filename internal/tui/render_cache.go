package tui

import (
	"os"
	"time"

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

	// Bar caches — tabs and help are cached via tabSig/helpSig below; the
	// status bar is recomputed every frame for correctness.
	tabs string
	help string

	// Partner view cache.
	partnerView    string
	partnerViewSig string

	// Path line cache.
	pathLine string
	pathSig  string

	// DXC path line cache — shows nearby spots below the QSO form. The
	// rendered line is time-dependent (spot age filters, fallback TTL), so
	// the render timestamp bounds how long a cached line may be served.
	dxcPathLine       string
	dxcPathSig        string
	dxcPathRenderedAt time.Time // when the cached line was rendered; older than dxcPathSpotsTTL it must re-render

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

	// DXC path-line spot fallback, loaded off the render path.
	dxcSpots          []store.DXCSpot
	dxcSpotsBand      string
	dxcSpotsAt        time.Time // when the fallback was fetched; time-dependent results expire
	dxcSpotsNeedFetch bool
	dxcSpotsFetchBand string

	// DXC path-line dupe set, loaded off the render path.
	dxcDupeSet          map[string]bool
	dxcDupeSig          string
	dxcDupeNeedFetch    bool
	dxcDupeFetchDate    string
	dxcDupeFetchContest string

	// Worked panel summary cache (call + grid + DXCC statistics).
	// The query is heavy (six queries per scope), so it is fetched off the
	// render path like the other DB-backed panels.
	workedSummary           store.WorkedSummary
	workedSummarySig        string
	workedSummaryNeedFetch  bool
	workedSummaryFetchCall  string
	workedSummaryFetchGrid4 string
	workedSummaryFetchDXCC  string
	workedSummaryFetchName  string

	// Logbook-wide counts (total QSOs, today's QSOs). Updated on tick
	// and invalidated on QSO save / logbook switch / midnight.
	logbookTotal     int
	logbookToday     int
	logbookStatsDate string // YYYYMMDD; when stale, re-fetch on next tick

	// DXCC continent cache — avoids prefix-tree lookup on every partner-view frame.
	dxccContCall  string
	dxccContValue string

	// Directory listing cache — avoids os.ReadDir during View() in file picker.
	dirCachePath    string
	dirCacheTime    time.Time
	dirCacheEntries []os.DirEntry

	// Per-frame view caches.
	formView string
	formSig  string
	formSec  int // second of last form cache; busts at 1 Hz for clock fields
	tabView  string
	tabSig   string
	helpView string
	helpSig  string

	// Cached MaxHeight clip styles for root View() body and final compositing.
	// Rebuilt only on terminal height change (resize).
	clipStyle      lipgloss.Style
	clipStyleH     int
	bodyClipStyle  lipgloss.Style
	bodyClipStyleH int

	// Cached image-screen styles — rebuilt only on dimension change.
	imagePlaceholderStyle lipgloss.Style
	imagePlaceholderW     int
	imagePlaceholderH     int

	// Path state (committed call/grid, updated on field exit).
	pathCall string
	pathGrid string

	// Contest line cache — rebuilt only when contest or NextQSO changes.
	contestLine    string
	contestLineSig string

	// Help suffix cache — avoids per-frame fmt.Sprintf on editor/log screens.
	helpSuffix    string
	helpSuffixSig string
}
