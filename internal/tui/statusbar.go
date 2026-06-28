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
	now := time.Now()
	utc := now.UTC()

	// Look up rig config once — used 4× below.
	rp, hasRig := m.App.Config.Rigs[s.RigName]

	callsign := s.Callsign
	if callsign == "" {
		callsign = "\u2014"
	}
	logName := config.LogbookDisplayName(m.App.Logbook)
	if logName == "" {
		logName = "\u2014"
	}
	op := m.App.Logbook.ActiveOperator
	if op != "" {
		if opCfg, ok := m.App.Config.Operators[op]; ok {
			op = config.OperatorDisplayName(&opCfg)
		}
	}

	rigName := ""
	if hasRig {
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
			" "+S.StatusValue.Render(truncateText(op, 24))+" ",
		)
	}

	// Build right side first so we know how much width remains for MSG.
	var rightParts []string

	rightParts = append(rightParts, statusDotStyled(m.inetOnline, "Net", m.Offline))
	if hasRig && rp.WsjtxEnabled {
		rightParts = append(rightParts, statusDotStyled(m.wsjtx.online, "WSJT"))
	}
	if hasRig && rp.RadioBackend != "" {
		rigLabel := "Rig"
		switch rp.RadioBackend {
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
	if hasRig && rp.RotorBackend == "hamlib" {
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
		S.StatusTime.Render(now.Format("15:04")+"L  "+utc.Format("1504")+"Z"),
	)

	right := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)
	left := lipgloss.JoinHorizontal(lipgloss.Top, leftParts...)
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	fillerW := m.width - leftW - rightW

	// WSJT-X TX message — only when at least 20 cells of free space remain.
	wsjtxOn := hasRig && rp.WsjtxEnabled
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
