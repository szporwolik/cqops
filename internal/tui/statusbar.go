package tui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
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
	logName := config.LogbookDisplayName(m.App.Logbook)
	if logName == "" {
		logName = "\u2014"
	}
	op := s.Operator
	if op == "" {
		op = "\u2014"
	}

	left := lipgloss.JoinHorizontal(lipgloss.Top,
		S.StatusApp.Render(" CQOps v"+version.Resolved()+" "),
		S.StatusLabel.Render("Call"),
		" "+S.StatusValue.Render(clamp(callsign, 10)),
		" "+S.StatusLabel.Render("Log"),
		" "+S.StatusValue.Render(clamp(logName, 12)),
		" "+S.StatusLabel.Render("Op"),
		" "+S.StatusValue.Render(clamp(op, 10)),
	)

	var rightParts []string

	rightParts = append(rightParts, statusDotStyled(m.inetOnline, "Net"))
	if m.App.Config.WSJTX.Enabled {
		rightParts = append(rightParts, statusDotStyled(m.wsjtxOnline, "WSJT"))
	}
	if cfgRig, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok && cfgRig.FlrigEnabled {
		if m.rigPTT || m.wsjtxTx {
			rightParts = append(rightParts, txDotStyle.Render("TX")+" ")
		} else {
			rightParts = append(rightParts, statusDotStyled(m.rigConnected, "Rig"))
		}
	}
	wl := m.App.Logbook.Wavelog
	if wl != nil && wl.Enabled {
		rightParts = append(rightParts, statusDotStyled(m.wlOnline, "WL"))
	}
	rightParts = append(rightParts,
		utcLabelStyle.Render("UTC"),
		S.StatusTime.Render(utc.Format("15:04:05")),
	)

	right := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)

	// Build WSJT-X TX message segment, if applicable.
	// Use WSJT-X Transmitting for sub-second TX detection; fall back to flrig PTT.
	var txMsg string
	txActive := m.wsjtxTx || (m.rigPTT && !m.wsjtxOnline)
	if m.wsjtxTxMsg != "" && m.App.Config.WSJTX.Enabled && m.wsjtxOnline {
		style := S.StatusValue
		if txActive {
			style = txDotStyle
		}
		txMsg = style.Render(m.wsjtxTxMsg)
	}
	txW := lipgloss.Width(txMsg)

	// Show local time only when there is room to spare.
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
	fillerW = m.width - leftW - rightW

	// Show WSJT-X TX message in the filler space when there is room.
	var filler string
	if txMsg != "" && fillerW >= txW+4 {
		pad := (fillerW - txW) / 2
		padR := fillerW - txW - pad
		filler = strings.Repeat(" ", pad) + txMsg + strings.Repeat(" ", padR)
	} else {
		if fillerW < 1 {
			fillerW = 1
		}
		filler = strings.Repeat(" ", fillerW)
	}

	return left + filler + right
}

// txDotStyle — transmit indicator, uses accent color to avoid confusion with
// the red "disconnected" convention used by statusDotOffStyle.
var txDotStyle = lipgloss.NewStyle().Foreground(P.Accent).Bold(true)

// statusDotStyled renders an integration indicator dot with label.
func statusDotStyled(on bool, label string) string {
	s := statusDotOffStyle
	if on {
		s = statusDotOnStyle
	}
	return s.Render(label) + " "
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
