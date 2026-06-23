package tui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/version"
)

// headerView renders the top status bar with callsign, logbook name,
// operator, WSJT-X TX message, integration status dots, and UTC clock.
func (m *Model) headerView() string {
	s := m.App.Logbook.Station
	utc := time.Now().UTC()

	callsign := s.Callsign
	if callsign == "" {
		callsign = "\u2014"
	}
	logName := config.LogbookDisplayName(m.App.Logbook)
	if logName == "" {
		logName = "\u2014"
	}
	op := s.Operator

	rigName := ""
	if rp, ok := m.App.Config.Rigs[s.RigName]; ok {
		rigName = rp.Name
	}

	leftParts := []string{
		S.StatusApp.Render(" CQOps v" + version.Resolved() + " "),
		S.StatusLabel.Render("Log"),
		" " + S.StatusValue.Render(truncateText(logName, 16)) + " ",
		S.StatusLabel.Render("Rig"),
		" " + S.StatusValue.Render(truncateText(rigName, 8)) + " ",
		S.StatusLabel.Render("Call"),
		" " + S.StatusValue.Render(truncateText(callsign, 14)) + " ",
	}
	if op != "" {
		leftParts = append(leftParts,
			S.StatusLabel.Render("Op"),
			" "+S.StatusValue.Render(truncateText(op, 14))+" ",
		)
	}

	// Build right side first so we know how much width remains for MSG.
	var rightParts []string

	rightParts = append(rightParts, statusDotStyled(m.inetOnline, "Net", m.Offline))
	if rp, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok && rp.WsjtxEnabled {
		rightParts = append(rightParts, statusDotStyled(m.wsjtx.online, "WSJT"))
	}
	if cfgRig, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok && cfgRig.RadioBackend != "" {
		rigLabel := "Rig"
		switch cfgRig.RadioBackend {
		case "flrig":
			rigLabel = "Flrig"
		case "hamlib":
			rigLabel = "Hamlib"
		}
		if m.wsjtx.tx {
			rightParts = append(rightParts, txDotStyle.Render(rigLabel)+" ")
		} else {
			rightParts = append(rightParts, statusDotStyled(m.rig.connected, rigLabel))
		}
	}
	if cfgRig, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok && cfgRig.RotorBackend == "hamlib" {
		rightParts = append(rightParts, statusDotStyled(m.rotor.connected, "Rotator"))
	}
	if m.App.Config.Integrations.DXC.Enabled {
		rightParts = append(rightParts, statusDotStyled(m.dxc.online, "DXC", m.Offline))
	}
	wl := m.App.Logbook.Wavelog
	if wl != nil && wl.Enabled {
		rightParts = append(rightParts, statusDotStyled(m.lookup.wlOnline, "WL", m.Offline))
	}
	rightParts = append(rightParts,
		utcLabelStyle.Render("UTC"),
		S.StatusTime.Render(utc.Format("15:04:05")),
	)

	right := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)

	// Show local time only when there is room to spare.
	left := lipgloss.JoinHorizontal(lipgloss.Top, leftParts...)
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	fillerW := m.width - leftW - rightW

	localTime := time.Now().Format("15:04")
	ltSegment := lipgloss.JoinHorizontal(lipgloss.Top,
		utcLabelStyle.Render("LT"),
		S.StatusTime.Render(localTime)+" ",
	)
	ltW := lipgloss.Width(ltSegment)
	if fillerW >= ltW+6 {
		rightParts = rightParts[:len(rightParts)-2]
		rightParts = append(rightParts,
			utcLabelStyle.Render("LT"),
			S.StatusTime.Render(localTime)+" ",
			utcLabelStyle.Render("UTC"),
			S.StatusTime.Render(utc.Format("15:04:05")),
		)
		right = lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)
		rightW = lipgloss.Width(right)
	}

	// WSJT-X TX message — only when at least 20 cells of free space remain.
	wsjtxOn := false
	if rp, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok {
		wsjtxOn = rp.WsjtxEnabled
	}
	if m.wsjtx.txMsg != "" && wsjtxOn && m.wsjtx.online {
		style := S.StatusValue
		if m.wsjtx.tx {
			style = txDotStyle
		}
		msgSeg := " " + S.StatusLabel.Render("MSG") + " " + style.Render(truncateText(m.wsjtx.txMsg, 24))
		msgW := lipgloss.Width(msgSeg)
		avail := m.width - leftW - rightW
		if avail >= msgW+4 {
			leftParts = append(leftParts, msgSeg)
			left = lipgloss.JoinHorizontal(lipgloss.Top, leftParts...)
			leftW = lipgloss.Width(left)
		}
	}

	fillerW = m.width - leftW - rightW
	if fillerW < 1 {
		fillerW = 1
	}

	return left + strings.Repeat(" ", fillerW) + right
}

// txDotStyle — transmit indicator, uses accent color to avoid confusion with
// the red "disconnected" convention used by statusDotOffStyle.
var txDotStyle = lipgloss.NewStyle().Foreground(P.Accent).Bold(true)

// statusDotStyled renders an integration indicator dot with label.
// When offline is true, uses warning color instead of error for the off state.
func statusDotStyled(on bool, label string, offline ...bool) string {
	s := statusDotOffStyle
	if on {
		s = statusDotOnStyle
	} else if len(offline) > 0 && offline[0] {
		s = statusDotWarnStyle
	}
	return s.Render(label) + " "
}

// renderStatusBar is the canonical entry point for status bar rendering.
func (m *Model) renderStatusBar() string { return m.headerView() }

// windowTitle returns the terminal window title for the main TUI.
func (m *Model) windowTitle() string {
	s := m.App.Logbook.Station
	callsign := s.Callsign
	logbook := config.LogbookDisplayName(m.App.Logbook)
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
