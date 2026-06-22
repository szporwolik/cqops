package tui

import (
	"time"

	"github.com/szporwolik/cqops/internal/psk"
)

// pskState holds PSK Reporter fetch, filter, and cache state.
type pskState struct {
	lastFetchByCall map[string]time.Time // per-callsign last successful fetch
	lastCall        string
	filterMins      int
	bandFilter      string // "" = all bands, or band name like "20m"
	modeFilter      string // "" = all modes, or mode name like "FT8"
	selected        int
	fetched         bool
	fetching        bool // true while async HTTP fetch is in flight
	cacheDir        string

	// Caches — avoids SQL + Lip Gloss work on every frame.
	spots   []psk.Report
	spotKey string
	view    string
	viewKey string
}
