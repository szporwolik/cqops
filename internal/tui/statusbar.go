package tui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/version"
)

// headerView renders the top status bar with callsign, logbook name,
// integration status dots, and UTC clock.
func (m *Model) headerView() string {
	s := m.App.Logbook.Station
	utc := time.Now().UTC()

	callsign := s.Callsign
	if callsign == "" {
		callsign = "\u2014"
	}
	logName := m.App.LogbookName
	if logName == "" {
		logName = "\u2014"
	}

	left := lipgloss.JoinHorizontal(lipgloss.Top,
		S.StatusApp.Render(" CQOPS v"+version.Resolved()+" "),
		S.StatusLabel.Render(" Call "),
		S.StatusValue.Render(clamp(callsign, 8)),
		S.StatusLabel.Render(" Log "),
		S.StatusValue.Render(clamp(logName, 10)),
	)

	right := lipgloss.JoinHorizontal(lipgloss.Top,
		statusDotStyled(m.inetOnline, "Net"),
	)
	if m.App.Config.WSJTX.Enabled {
		right = lipgloss.JoinHorizontal(lipgloss.Top,
			right,
			statusDotStyled(m.wsjtxOnline, "WSJT"),
		)
	}
	if cfgRig, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok && cfgRig.FlrigEnabled {
		right = lipgloss.JoinHorizontal(lipgloss.Top,
			right,
			statusDotStyled(m.rigConnected, "Rig"),
		)
	}
	if m.App.Config.Wavelog.Enabled {
		right = lipgloss.JoinHorizontal(lipgloss.Top,
			right,
			statusDotStyled(m.wlOnline, "WL"),
		)
	}
	right = lipgloss.JoinHorizontal(lipgloss.Top,
		right,
		S.StatusRight.Render(" "),
		S.StatusLabel.Render("UTC "),
		S.StatusTime.Render(utc.Format("15:04:05")),
	)

	fillerW := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if fillerW < 1 {
		fillerW = 1
	}

	return left + strings.Repeat(" ", fillerW) + right
}

// statusDotStyled renders an integration indicator dot with label.
func statusDotStyled(on bool, label string) string {
	fg := P.Error
	if on {
		fg = P.Success
	}
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(P.Background).
		Render(label) + S.StatusRight.Render(" ")
}

// renderStatusBar is the canonical entry point for status bar rendering.
func (m *Model) renderStatusBar() string { return m.headerView() }

// renderToastBar renders active toasts as a bar for layout measurement.
func (m *Model) renderToastBar() string {
	return RenderToasts(m.toasts.Active(), m.width)
}

// windowTitle returns the terminal window title for the main TUI.
func (m *Model) windowTitle() string {
	s := m.App.Logbook.Station
	callsign := s.Callsign
	logbook := m.App.LogbookName
	if callsign == "" && logbook == "" {
		return "CQOps"
	}
	if callsign == "" {
		return "CQOps \u2014 " + logbook
	}
	if logbook == "" {
		return "CQOps \u2014 " + callsign
	}
	return "CQOps \u2014 " + callsign + " @ " + logbook
}
