package tui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
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
			op = opCfg.Callsign
		}
	}

	rigName := ""
	if hasRig {
		rigName = rp.Name
	}

	// Compact display: merge labels when values are identical.
	// e.g. Log/Call SP9SPM instead of Log Priv Call SP9SPM.
	leftParts := []string{
		S.StatusApp.Render(" CQOps "),
		" ",
	}
	logEqCall := strings.EqualFold(logName, callsign)
	callEqOp := op != "" && strings.EqualFold(callsign, op)
	logEqOp := op != "" && strings.EqualFold(logName, op)

	switch {
	case logEqCall && callEqOp:
		// All three identical: Log/Call/Op VALUE
		leftParts = append(leftParts,
			S.StatusLabel.Render("Log/Call/Op"),
			" "+S.StatusValue.Render(truncateText(callsign, 20))+" ",
		)
	case logEqCall:
		// Log == Call, Op different: Log/Call VALUE Op OP
		leftParts = append(leftParts,
			S.StatusLabel.Render("Log/Call"),
			" "+S.StatusValue.Render(truncateText(callsign, 16))+" ",
		)
		if op != "" {
			leftParts = append(leftParts,
				S.StatusLabel.Render("Op"),
				" "+S.StatusValue.Render(truncateText(op, 24))+" ",
			)
		}
	case callEqOp:
		// Call == Op, Log different: Log LOG Call/Op VALUE
		leftParts = append(leftParts,
			S.StatusLabel.Render("Log"),
			" "+S.StatusValue.Render(truncateText(logName, 16))+" ",
			S.StatusLabel.Render("Call/Op"),
			" "+S.StatusValue.Render(truncateText(callsign, 20))+" ",
		)
	case logEqOp:
		// Log == Op, Call different: Log/Op VALUE Call CALL
		leftParts = append(leftParts,
			S.StatusLabel.Render("Log/Op"),
			" "+S.StatusValue.Render(truncateText(logName, 16))+" ",
			S.StatusLabel.Render("Call"),
			" "+S.StatusValue.Render(truncateText(callsign, 14))+" ",
		)
	default:
		// All distinct: Log LOG Call CALL [Op OP]
		leftParts = append(leftParts,
			S.StatusLabel.Render("Log"),
			" "+S.StatusValue.Render(truncateText(logName, 16))+" ",
			S.StatusLabel.Render("Call"),
			" "+S.StatusValue.Render(truncateText(callsign, 14))+" ",
		)
		if op != "" {
			leftParts = append(leftParts,
				S.StatusLabel.Render("Op"),
				" "+S.StatusValue.Render(truncateText(op, 24))+" ",
			)
		}
	}
	leftParts = append(leftParts,
		S.StatusLabel.Render("Rig"),
		" "+S.StatusValue.Render(truncateText(rigName, 8))+" ",
	)

	// Build right side first so we know how much width remains for MSG.
	var rightParts []string

	// Core: Rig — fundamental, can't operate without frequency data.
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

	// Core: Net — internet required for callbook, spots, Wavelog.
	rightParts = append(rightParts, statusDotStyled(m.inetOnline, "Net", m.Offline))

	// Digital: WSJT-X.
	if hasRig && rp.WsjtxEnabled {
		rightParts = append(rightParts, statusDotStyled(m.wsjtx.online, "WSJT"))
	}

	// Spots: DX Cluster.
	if m.App.Config.Integrations.DXC.Enabled {
		rightParts = append(rightParts, statusDotStyled(m.dxc.online, "DXC", m.Offline))
	}

	// Logging: Wavelog cloud sync.
	wl := m.App.Logbook.Wavelog
	if wl != nil && wl.Enabled {
		rightParts = append(rightParts, statusDotStyled(m.lookup.wlOnline, "WL", m.Offline))
	}

	// Position: APRS.
	if m.App.Config.Integrations.APRS.Enabled {
		aprsCfg := m.App.Logbook.APRS
		if aprsCfg != nil && aprsCfg.Enabled {
			label := "APRS-RX"
			if aprsCfg.SendLocation {
				label = "APRS"
			}
			online := m.App.APRSClient != nil && m.App.APRSClient.IsConnected()
			rightParts = append(rightParts, statusDotStyled(online, label, m.Offline))
		}
	}

	// Hardware: Rotator — optional, many stations don't have one.
	if hasRig && rp.RotorBackend == "hamlib" {
		rightParts = append(rightParts, statusDotStyled(m.rotor.connected, "ROT"))
	}

	// Hardware: GPS — auxiliary position source.
	if m.App.Config.Integrations.GPS.Enabled {
		switch {
		case m.gps.online && m.gps.hasFix:
			rightParts = append(rightParts, statusDotOnStyle.Render("GPS")+" ")
		case m.gps.online:
			rightParts = append(rightParts, statusDotWarnStyle.Render("GPS")+" ")
		default:
			rightParts = append(rightParts, statusDotOffStyle.Render("GPS")+" ")
		}
	}

	// Auxiliary: HTTP dashboard — web interface, lowest priority.
	if m.App.Config.Integrations.HTTPServer.Enabled {
		rightParts = append(rightParts, statusDotStyled(m.http.online, "HTTP"))
	}
	rightParts = append(rightParts,
		S.StatusTime.Render(now.Format("15:04")+"L "+utc.Format("1504")+"Z"),
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

// aprsConnected returns true when the APRS client is connected (either
// TCP APRS-IS or KISS serial). Used by the render cache to detect
// connection state changes and invalidate the status bar.
func (m *Model) aprsConnected() bool {
	return m.App.APRSClient != nil && m.App.APRSClient.IsConnected()
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
