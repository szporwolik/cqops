package tui

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

// MeasureLayout returns a Layout with measured dimensions.
// Bar heights are known constants (1 row each); profile line is 0 or 1.
// This avoids rendering bars twice — once here, once in View().
func MeasureLayout(m *Model) Layout {
	var l Layout
	l.TerminalW = m.width
	l.TerminalH = m.height

	if l.TerminalW < 40 {
		l.TerminalW = 80
	}
	if l.TerminalH < 10 {
		l.TerminalH = 24
	}

	l.StatusH = 1
	l.TabH = 3 // top border + content + bottom border (lipgloss tab pattern)
	l.HelpH = 1

	l.ContentW = l.TerminalW - 2
	if l.ContentW < 20 {
		l.ContentW = 20
	}

	usedH := l.StatusH + l.TabH + l.HelpH
	l.ContentH = l.TerminalH - usedH
	if l.ContentH < 3 {
		l.ContentH = 3
	}

	return l
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
