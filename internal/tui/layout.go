package tui

import (
	"charm.land/lipgloss/v2"
)

// Layout holds pre-computed dimensions for all visual zones of the TUI.
// All values are measured from actual rendered content, not hard-coded.
type Layout struct {
	TerminalW int // terminal width from WindowSizeMsg
	TerminalH int // terminal height from WindowSizeMsg

	// Measured heights (set after rendering each zone)
	StatusH  int
	TabH     int
	ProfileH int // may be 0
	HelpH    int
	ToastH   int // may be 0

	// Derived
	ContentW int // terminal width minus 2 (for padding)
	ContentH int // remaining height after fixed zones
}

// MeasureLayout renders each fixed zone and returns a Layout with measured
// dimensions. This is the single source of truth for all sizing — no magic numbers.
func MeasureLayout(m *Model) Layout {
	var l Layout
	l.TerminalW = m.width
	l.TerminalH = m.height

	// Clamp terminal to sane minimums (Bubble Tea itself doesn't do this;
	// we provide a reasonable floor for the UI to not break)
	if l.TerminalW < 40 {
		l.TerminalW = 80
	}
	if l.TerminalH < 10 {
		l.TerminalH = 24
	}

	// Render each fixed zone and measure its height
	statusBar := m.renderStatusBar()
	l.StatusH = lipgloss.Height(statusBar)

	tabBar := m.renderTabBar()
	l.TabH = lipgloss.Height(tabBar)

	// Profile line: only shown on QSO form when no sub-model is active
	if !m.isSubmodelActive() && m.confirm == nil {
		profileLine := m.renderProfileLine()
		if profileLine != "" {
			l.ProfileH = lipgloss.Height(profileLine)
		}
	}

	helpBar := m.renderHelpBar()
	l.HelpH = lipgloss.Height(helpBar)

	// Toasts overlay the content area — they do not consume vertical space.
	// Their height is tracked only for positioning calculations.

	// Content area fills remaining vertical space
	l.ContentW = l.TerminalW - 2
	if l.ContentW < 20 {
		l.ContentW = 20
	}

	usedH := l.StatusH + l.TabH + l.ProfileH + l.HelpH
	l.ContentH = l.TerminalH - usedH
	if l.ContentH < 3 {
		l.ContentH = 3
	}

	return l
}

// =============================================================================
// Content area height breakdown for QSO form + table
// =============================================================================

// QSOFormHeight returns the height of the QSO entry form.
// This is measured rather than hard-coded, so if fields change the layout adapts.
func QSOFormHeight(m *Model, contentW int) int {
	form := m.viewForm(contentW)
	pathLine := m.formPathRow(contentW)
	if pathLine != "" {
		form += "\n" + pathLine
	}
	return lipgloss.Height(form)
}

// ViewportHeight returns the height available for the QSO table viewport,
// given the content area height and the measured form height.
func ViewportHeight(contentH, formH int) int {
	// Reserve 1 line for spacing between form and table
	vpH := contentH - formH - 1
	if vpH < 3 {
		vpH = 3
	}
	return vpH
}

// ContentWidth returns a content-area width suitable for sub-views.
// It applies the canonical clamping: terminal width minus 2 for padding,
// with a minimum of 30 columns. All sub-views should use this instead of
// duplicating width math.
func ContentWidth(terminalW int) int {
	w := terminalW - 2
	if w < 30 {
		w = 30
	}
	return w
}
