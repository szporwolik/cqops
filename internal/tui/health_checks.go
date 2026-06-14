package tui

import (
	"net/http"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
)

// =============================================================================
// Health checks — periodic internet connectivity and date/time management.
// =============================================================================

// autoUpdateDateTime keeps the QSO form date/time fields current.
func (m *Model) autoUpdateDateTime() {
	if !m.dateTimeAuto {
		return
	}
	now := time.Now().UTC()
	m.fields[fieldDate].SetValue(now.Format("2006-01-02"))
	m.fields[fieldTime].SetValue(now.Format("15:04:05"))
}

// maybeCheckInet returns a tea.Cmd to check internet connectivity at intervals.
func (m *Model) maybeCheckInet() tea.Cmd {
	if m.tickCount%healthCheckTicks == 0 {
		return checkInetCmd()
	}
	return nil
}

// checkInetCmd returns a tea.Cmd that checks internet connectivity
// by attempting to reach Google's generate_204 endpoint.
func checkInetCmd() tea.Cmd {
	return func() tea.Msg {
		applog.Debug("Internet: testing connectivity")
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("https://clients3.google.com/generate_204")
		if err != nil {
			applog.Warn("Internet: unreachable", "error", err)
			return inetResultMsg(false)
		}
		resp.Body.Close()
		applog.Info("Internet: reachable")
		return inetResultMsg(true)
	}
}
