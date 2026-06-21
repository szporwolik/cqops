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
	if m.solar.fetching {
		return nil
	}
	if !m.inetOnline {
		return nil
	}
	if !m.App.Config.General.SolarAtQSOPane {
		return nil
	}
	if time.Since(m.solar.lastFetch) < 5*time.Minute {
		return nil
	}

	cached, fresh := solar.Cached(m.solar.cacheDir)
	if fresh && cached != nil && m.inetOnline {
		m.solar.data = cached
		m.solar.cachedSig = ""
		m.solar.failed = false
		m.solar.lastFetch = time.Now()
		applog.Info("Solar: data from cache",
			"solarflux", cached.SolarFlux,
			"aindex", cached.AIndex,
			"kindex", fmt.Sprintf("%.1f", cached.KIndex),
		)
		return nil
	}

	m.solar.fetching = true
	m.solar.lastFetch = time.Now()
	m.toasts.Info("Solar: fetching hamqsl.com\u2026")
	return solarFetchCmd(m.solar.cacheDir)
}

// handleSolarResult processes the result of an async solar fetch.
func (m *Model) handleSolarResult(msg solarFetchMsg) {
	m.solar.fetching = false
	if msg.err != nil {
		m.solar.failed = true
		m.solar.data = nil
		m.solar.cachedSig = ""
		m.solar.cachedView = ""
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
	m.solar.data = msg.data
	m.solar.failed = false
	m.solar.cachedSig = ""
	applog.Info("Solar: hamqsl.com data updated",
		"attempts", msg.attempts,
	)
	m.toasts.Info("Solar: hamqsl.com data updated")
}
