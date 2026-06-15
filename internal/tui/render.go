package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// drawBorderedBox draws a NormalBorder box around content using lipgloss's
// built-in Border.
func drawBorderedBox(content string, boxW int) string {
	return borderBoxStyle.Width(boxW).Render(content)
}

// osc8Link returns an OSC-8 hyperlink sequence. Most modern terminals
// (Windows Terminal, iTerm2, Kitty, etc.) render these as clickable links.
// Ctrl+click opens the URL in the system browser.
func osc8Link(url, text string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}

// tern is a simple ternary for strings.
func tern(cond bool, t, f string) string {
	if cond {
		return t
	}
	return f
}

// fillBody returns the content with trailing newlines so the total height
// equals contentH. Use this in configuration menus to push the help bar down.
func fillBody(content string, contentH int) string {
	if contentH <= 0 {
		return content
	}
	current := lipgloss.Height(content)
	if current >= contentH {
		return content
	}
	return content + strings.Repeat("\n", contentH-current)
}

// menuTitle renders a configuration-menu title bar that fills the full
// terminal width.
func menuTitle(title string, width int) string {
	return S.Title.Width(width).Render(title)
}

// menuLine wraps a single menu row, filling to the given width.
func menuLine(content string, width int) string {
	if lipgloss.Width(content) >= width {
		return content
	}
	return content + strings.Repeat(" ", width-lipgloss.Width(content))
}

// fit renders s padded to exactly w cells. An empty string renders as a dim
// em-dash. Strings wider than w are truncated. Uses plain string ops — no
// lipgloss allocation.
func fit(s string, w int) string {
	if s == "" {
		return DimStyle.Render(padOrTrunc("\u2014", w))
	}
	return padOrTrunc(s, w)
}

// clamp renders s padded/truncated to exactly w cells with spaces.
// An empty string renders as w spaces.
func clamp(s string, w int) string {
	if s == "" {
		return strings.Repeat(" ", w)
	}
	return padOrTrunc(s, w)
}

// padOrTrunc returns s truncated or padded with spaces to exactly w cells.
func padOrTrunc(s string, w int) string {
	sw := lipgloss.Width(s)
	if sw > w {
		return truncate(s, w)
	}
	if sw < w {
		return s + strings.Repeat(" ", w-sw)
	}
	return s
}

// --- standalone utilities (moved from model.go) ---

func stripNonDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatDate(adif string) string {
	if len(adif) < 8 {
		return "—"
	}
	return adif[0:4] + "-" + adif[4:6] + "-" + adif[6:8]
}

func formatTime(adif string) string {
	if len(adif) < 6 {
		return "—"
	}
	return adif[0:2] + ":" + adif[2:4] + ":" + adif[4:6]
}

func truncate(s string, max int) string {
	if max < 3 {
		return s
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// =============================================================================
// Layout helpers
// =============================================================================

// FixedZoneHeight is the number of rows consumed by the fixed UI zones:
// status bar (1) + profile line (0-1) + tab bar (1) + help bar (1).
const FixedZoneHeight = 4

// contentHeight returns the available content height for a given terminal height
// after accounting for fixed UI zones.
func contentHeight(terminalH int) int {
	h := terminalH - FixedZoneHeight
	if h < 3 {
		h = 3
	}
	return h
}

// =============================================================================
// Textinput helpers
// =============================================================================

// applyTextinputSurfaceStyle is a no-op. Background styling has been removed
// for performance; textinputs inherit terminal default background.
func applyTextinputSurfaceStyle(ti *textinput.Model) {}

// newTextinput creates a textinput with Prompt already cleared (the default
// "> " prompt is not useful in our forms). All other fields are at defaults.
func newTextinput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	return ti
}

// truncateText truncates s to maxW visual cells, appending "…" if needed.
func truncateText(s string, maxW int) string {
	if maxW <= 1 {
		return ""
	}
	w := 0
	runes := []rune(s)
	for i, r := range runes {
		rw := 1
		if r > 0xffff {
			rw = 2
		}
		if w+rw >= maxW {
			return string(runes[:i]) + "\u2026"
		}
		w += rw
	}
	return s
}
