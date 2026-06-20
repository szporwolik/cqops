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
	sig := fmt.Sprintf("%d|%v|%d|%d|%v|%v|%d", m.screen, hasPartner, conf, m.width, m.inetOnline, m.dxc.online, refReady)
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

	dxcOnline := m.App.Config.DXC.Enabled && m.dxc.online
	allTabs := []tab{
		{"F1 QSO", "QSO", "F1", m.screen == screenQSO && m.confirm == nil, false},
		{"F2 QRZ", "QRZ", "F2", (m.screen == screenPartner || m.screen == screenImage) && hasPartner, !hasPartner},
		{"F3 CON", "CON", "F3", m.screen == screenCON, false},
		{"F4 DXC", "DXC", "F4", m.screen == screenDXC, !dxcOnline},
		{"F5 HRD", "HRD", "F5", m.screen == screenPSKReporter, !m.inetOnline},
		{"F6 REF", "REF", "F6", m.screen == screenRef, !m.isREFReady()},
		{"F7 BPL", "BPL", "F7", m.screen == screenBPL, false},
		{"F8 LOG", "LOG", "F8", m.screen == screenLogbookEditor, false},
		{"F9 CFG", "CFG", "F9", m.screen == screenMainMenu || m.screen == screenConfig || m.screen == screenCallbook || m.screen == screenIntegration || m.screen == screenChooser || m.screen == screenRigEdit || m.screen == screenNotifications, false},
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
