package tui

import (
	"os"
	"strconv"
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

	// Cache key: screen + partner presence + confirm state + width
	// + connectivity + REF ready state.
	conf := 0
	if m.confirm != nil {
		conf = 1
	}
	refReady := 0
	if m.isREFReady() {
		refReady = 1
	}
	var sigB strings.Builder
	sigB.WriteString(strconv.Itoa(int(m.screen)))
	sigB.WriteByte('|')
	if hasPartner {
		sigB.WriteByte('1')
	} else {
		sigB.WriteByte('0')
	}
	sigB.WriteByte('|')
	sigB.WriteString(strconv.Itoa(conf))
	sigB.WriteByte('|')
	sigB.WriteString(strconv.Itoa(m.width))
	sigB.WriteByte('|')
	if m.inetOnline {
		sigB.WriteByte('1')
	} else {
		sigB.WriteByte('0')
	}
	sigB.WriteByte('|')
	if m.dxc.online {
		sigB.WriteByte('1')
	} else {
		sigB.WriteByte('0')
	}
	sigB.WriteByte('|')
	sigB.WriteString(strconv.Itoa(refReady))
	sig := sigB.String()
	if m.rc.tabSig == sig && m.rc.tabView != "" {
		return m.rc.tabView
	}

	w := m.width
	if w < 20 {
		w = 80
	}

	type tab struct {
		label    string
		short    string
		minimal  string
		active   bool
		disabled bool
	}

	dxcOnline := m.App.Config.Integrations.DXC.Enabled && m.dxc.online

	// On the raw Linux console (TERM=linux) function keys are not
	// delivered to userspace.  Show Alt+digit shortcuts instead so
	// the tab labels match what actually works.
	mk := func(f string, a string) string {
		if strings.ToLower(os.Getenv("TERM")) == "linux" {
			return a
		}
		return f
	}

	allTabs := []tab{
		{"F1 QSO", "QSO", mk("F1", "A1"), m.screen == screenQSO && m.confirm == nil, false},
		{"F2 QRZ", "QRZ", mk("F2", "A2"), (m.screen == screenPartner || m.screen == screenImage) && hasPartner, !hasPartner},
		{"F4 DXC", "DXC", mk("F4", "A4"), m.screen == screenDXC, !dxcOnline},
		{"F5 HRD", "HRD", mk("F5", "A5"), m.screen == screenPSKReporter, !m.inetOnline},
		{"F6 REF", "REF", mk("F6", "A6"), m.screen == screenRef, !m.isREFReady()},
		{"F7 BPL", "BPL", mk("F7", "A7"), m.screen == screenBPL, false},
		{"F8 LOG", "LOG", mk("F8", "A8"), m.screen == screenLogbookEditor, false},
		{"F9 CFG", "CFG", mk("F9", "A9"), m.screen == screenMainMenu || m.screen == screenConfig || m.screen == screenIntegration || m.screen == screenChooser || m.screen == screenRigEdit || m.screen == screenNotifications, false},
	}

	// Pick the best-fit label tier and build the final parts in one pass.
	// Tiers: full → medium (F1 QSO + short rest) → minimal (F1..F9).
	type tier struct {
		name string
		fn   func(tab) string
	}
	tiers := []tier{
		{"full", func(t tab) string { return t.label }},
		{"medium", func(t tab) string {
			if t.label == "F1 QSO" {
				return t.label
			}
			return t.short
		}},
		{"minimal", func(t tab) string { return t.minimal }},
	}

	var parts []string
	var row string
	for _, tier := range tiers {
		parts = parts[:0]
		for _, t := range allTabs {
			s := tabStyle
			if t.active {
				s = activeTabStyle
			}
			if t.disabled {
				s = disabledTabStyle
			}
			parts = append(parts, s.Render(" "+tier.fn(t)+" "))
		}
		row = lipgloss.JoinHorizontal(lipgloss.Top, parts...)
		if lipgloss.Width(row) <= w || w <= 0 {
			break // parts already built with correct tier
		}
	}

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
