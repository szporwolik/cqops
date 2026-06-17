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
// Below 85 chars terminal width, separators are dropped to save space.
func (m *Model) autoUpdateDateTime() {
	if !m.dateTimeAuto {
		return
	}
	now := time.Now().UTC()
	dateFmt, timeFmt := dateTimeFormats(m.width)
	m.fields[fieldDate].SetValue(now.Format(dateFmt))
	m.fields[fieldTime].SetValue(now.Format(timeFmt))
}

// dateTimeFormats returns the format strings for date and time fields
// based on terminal width. Below 85 chars, dashes and colons are omitted.
func dateTimeFormats(width int) (dateFmt, timeFmt string) {
	if width < 85 {
		return "20060102", "150405"
	}
	return "2006-01-02", "15:04:05"
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
		client := &http.Client{Timeout: 1 * time.Second}
		resp, err := client.Get("https://clients3.google.com/generate_204")
		if err != nil {
			applog.Warn("Internet: unreachable", "error", err)
			return inetResultMsg(false)
		}
		defer resp.Body.Close()
		applog.Info("Internet: reachable")
		return inetResultMsg(true)
	}
}
