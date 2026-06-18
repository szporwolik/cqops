package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Pre-allocated tab border styles — follows the lipgloss layout example pattern.
var (
	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	inactiveTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tabStyle = lipgloss.NewStyle().
			Border(inactiveTabBorder, true).
			BorderForeground(P.Border).
			Padding(0, 1)

	activeTabStyle = tabStyle.Border(activeTabBorder, true).
			BorderForeground(P.Cursor)

	disabledTabStyle = tabStyle.Foreground(P.TextDim)

	tabGapStyle = tabStyle.
			BorderTop(false).
			BorderLeft(false).
			BorderRight(false)
)

// tabView renders the function-key tab bar using lipgloss's tab border pattern.
// Cached — only rebuilds when screen, partner presence, or confirm state change.
func (m *Model) tabView() string {
	hasPartner := m.lookup.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""

	// Cache key: screen + partner presence + confirm state + width.
	conf := 0
	if m.confirm != nil {
		conf = 1
	}
	sig := fmt.Sprintf("%d|%v|%d|%d|%v|%v", m.screen, hasPartner, conf, m.width, m.inetOnline, m.dxc.online)
	if m.rc.tabSig == sig && m.rc.tabView != "" {
		return m.rc.tabView
	}

	w := m.width
	if w < 20 {
		w = 80
	}

	type tab struct {
		label    string
		active   bool
		disabled bool
	}

	dxcOnline := m.App.Config.DXC.Enabled && m.dxc.online
	allTabs := []tab{
		{"F1 QSO", m.screen == screenQSO && m.confirm == nil, false},
		{"F2 QRZ", (m.screen == screenPartner || m.screen == screenImage) && hasPartner, !hasPartner},
		{"F4 DXC", m.screen == screenDXC, !dxcOnline},
		{"F5 HRD", m.screen == screenPSKReporter, !m.inetOnline},
		{"F7 LOG", m.screen == screenLogbookEditor, false},
		{"F8 CFG", m.screen == screenMainMenu || m.screen == screenConfig || m.screen == screenCallbook || m.screen == screenIntegration || m.screen == screenChooser || m.screen == screenRigEdit || m.screen == screenNotifications, false},
	}

	var parts []string
	for _, t := range allTabs {
		s := tabStyle
		if t.active {
			s = activeTabStyle
		}
		if t.disabled {
			s = disabledTabStyle
		}
		parts = append(parts, s.Render(" "+t.label+" "))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	// Fill remaining width with border-only gap tab.
	gapW := w - lipgloss.Width(row)
	if gapW < 1 {
		gapW = 1
	}
	gap := tabGapStyle.Render(strings.Repeat(" ", gapW))

	result := lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	m.rc.tabSig = sig
	m.rc.tabView = result
	return result
}

// renderTabBar is the canonical entry point for tab bar rendering.
func (m *Model) renderTabBar() string { return m.tabView() }
