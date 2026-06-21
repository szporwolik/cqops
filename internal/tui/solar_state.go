package tui

import (
	"time"

	"github.com/szporwolik/cqops/internal/solar"
)

// solarState holds solar-terrestrial data fetch and cache state.
type solarState struct {
	data       *solar.Data
	lastFetch  time.Time
	fetching   bool
	failed     bool // true after all retries exhausted
	cacheDir   string
	cachedView string
	cachedSig  string
}
