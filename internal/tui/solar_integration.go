package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/solar"
)

// =============================================================================
// Solar data — 15-minute fetch from hamqsl.com
// =============================================================================

// solarFetchMsg is sent when an async solar data fetch completes.
type solarFetchMsg struct {
	data     *solar.Data
	err      error
	attempts int
}

const solarRetryDelay = 5 * time.Second
const solarMaxRetries = 3

// solarFetchCmd returns a tea.Cmd that fetches solar data asynchronously
// with up to 3 retries on failure. Never blocks the UI.
func solarFetchCmd(cacheDir string) tea.Cmd {
	return func() tea.Msg {
		var lastErr error
		for attempt := 1; attempt <= solarMaxRetries; attempt++ {
			d, err := solar.Fetch(cacheDir)
			if err == nil {
				return solarFetchMsg{data: d, attempts: attempt}
			}
			lastErr = err
			if attempt < solarMaxRetries {
				applog.Warn("Solar: fetch failed, retrying",
					"attempt", attempt, "error", err,
				)
				time.Sleep(solarRetryDelay)
			}
		}
		return solarFetchMsg{
			err:      fmt.Errorf("failed after %d attempts: %w", solarMaxRetries, lastErr),
			attempts: solarMaxRetries,
		}
	}
}

// maybeFetchSolar returns a tea.Cmd to fetch solar data if conditions are met.
func (m *Model) maybeFetchSolar() tea.Cmd {
	if m.solarFetching {
		return nil
	}
	if !m.inetOnline {
		return nil
	}
	if !m.App.Config.General.SolarAtQSOPane {
		return nil
	}
	if time.Since(m.solarLastFetch) < 5*time.Minute {
		return nil
	}

	cached, fresh := solar.Cached(m.solarCacheDir)
	if fresh && cached != nil {
		m.solarData = cached
		m.cachedSolarSig = ""
		m.solarFailed = false
		m.solarLastFetch = time.Now()
		applog.Info("Solar: data from cache",
			"solarflux", cached.SolarFlux,
			"aindex", cached.AIndex,
			"kindex", fmt.Sprintf("%.1f", cached.KIndex),
		)
		return nil
	}

	m.solarFetching = true
	m.solarLastFetch = time.Now()
	applog.Info("Solar: fetching hamqsl.com")
	m.toasts.Info("Solar: fetching hamqsl.com\u2026")
	return solarFetchCmd(m.solarCacheDir)
}

// handleSolarResult processes the result of an async solar fetch.
func (m *Model) handleSolarResult(msg solarFetchMsg) {
	m.solarFetching = false
	if msg.err != nil {
		m.solarFailed = true
		m.solarData = nil
		m.cachedSolarSig = ""
		m.cachedSolarView = ""
		m.toasts.Error("Solar: unavailable — will retry later")
		applog.Warn("Solar: all fetch attempts failed",
			"attempts", msg.attempts, "error", msg.err,
		)
		return
	}
	if msg.data == nil {
		applog.Warn("Solar: nil data returned")
		return
	}
	m.solarData = msg.data
	m.solarFailed = false
	m.cachedSolarSig = ""
	applog.Info("Solar: fetched OK",
		"solarflux", msg.data.SolarFlux,
		"aindex", msg.data.AIndex,
		"kindex", fmt.Sprintf("%.1f", msg.data.KIndex),
		"sunspots", msg.data.Sunspots,
		"attempts", msg.attempts,
	)
	m.toasts.Info("Solar: hamqsl.com data updated")
}
