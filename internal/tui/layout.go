package tui

import (
	"fmt"
	"strings"
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

// MeasureLayout returns a Layout with measured dimensions.
// Bar heights are known constants (1 row each); profile line is 0 or 1.
// This avoids rendering bars twice — once here, once in View().
func MeasureLayout(m *Model) Layout {
	var l Layout
	l.TerminalW = m.width
	l.TerminalH = m.height

	// Substitute a size only when there is none yet (the frame before the
	// first WindowSizeMsg). A terminal that is genuinely small keeps its real
	// size so View can refuse it: the old clamp promoted everything below
	// 40x10 to 80x24, which silently defeated that refusal and painted a
	// 79-cell frame into a 39-cell terminal.
	if l.TerminalW <= 0 {
		l.TerminalW = 80
	}
	if l.TerminalH <= 0 {
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

// The smallest terminal every screen is laid out for. Below these numbers the
// form, the tables and the bars cannot be shown without overflowing, so the app
// refuses instead of painting a frame that wraps.
const (
	minTermW = 75
	minTermH = 24
)

// refusalView explains why nothing is drawn. Every line is cut to the terminal
// width: at 20 columns no wording fits, and a refusal that wraps looks like the
// rendering bug it exists to prevent.
func refusalView(w, h int) string {
	// Short lines on purpose: the reason and the numbers must survive a narrow
	// terminal, which is exactly where this text is read.
	lines := []string{
		"",
		"  CQOps — Terminal too small",
		fmt.Sprintf("  %dx%d (min %dx%d)", w, h, minTermW, minTermH),
		"",
		"  Press F10 and then Enter to quit",
	}
	for i, l := range lines {
		lines[i] = truncateText(l, w)
	}
	return ErrorStyle.Render(strings.Join(lines, "\n"))
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
