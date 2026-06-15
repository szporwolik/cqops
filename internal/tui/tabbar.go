package tui

import (
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
func (m *Model) tabView() string {
	hasPartner := m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""
	w := m.width
	if w < 20 {
		w = 80
	}

	type tab struct {
		label    string
		active   bool
		disabled bool
	}

	allTabs := []tab{
		{"F1 QSO Form", m.screen == screenQSO && m.confirm == nil, false},
		{"F2 Partner", (m.screen == screenPartner || m.screen == screenImage) && hasPartner, !hasPartner},
		{"F6 Log Editor", m.screen == screenLogbookEditor, false},
		{"F7 Config", m.screen == screenMainMenu || m.screen == screenConfig || m.screen == screenCallbook || m.screen == screenIntegration || m.screen == screenChooser || m.screen == screenRigEdit || m.screen == screenNotifications, false},
		{"F8 Logs", m.screen == screenLogView, false},
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

	return lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
}

// renderTabBar is the canonical entry point for tab bar rendering.
func (m *Model) renderTabBar() string { return m.tabView() }
