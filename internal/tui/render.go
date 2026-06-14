package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// drawBorderedBox draws a NormalBorder box where every character (including
// borders) has explicit Surface background. This prevents the right │ leak
// that occurs with lipgloss's built-in Border when content has SGR resets.
func drawBorderedBox(content string, innerW, boxW int) string {
	bg := lipgloss.NewStyle().Background(P.Surface)
	fg := lipgloss.NewStyle().Foreground(P.Border).Background(P.Surface)

	top := fg.Render("┌" + strings.Repeat("─", innerW) + "┐")
	bot := fg.Render("└" + strings.Repeat("─", innerW) + "┘")
	left := fg.Render("│")
	right := fg.Render("│")

	var b strings.Builder
	b.WriteString(top)
	b.WriteString("\n")

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		b.WriteString(left)
		b.WriteString(bg.Width(innerW).Render(line))
		b.WriteString(right)
		b.WriteString("\n")
	}
	b.WriteString(bot)

	return lipgloss.NewStyle().Width(boxW).Background(P.Surface).Render(b.String())
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
	h := lipgloss.Height(content)
	fillerH := contentH - h
	if fillerH < 0 {
		fillerH = 0
	}
	if fillerH > 0 {
		return content + strings.Repeat("\n", fillerH)
	}
	return content
}

// section renders a titled horizontal rule: "── Title ──────────"
func section(title string, width int) string {
	rem := width - lipgloss.Width(title)
	if rem > 0 {
		return SectionStyle.Render(title + strings.Repeat("─", rem))
	}
	return SectionStyle.Render(title)
}

// fit renders s padded to exactly w cells using lipgloss. An empty string
// renders as a dim em-dash. Strings wider than w are truncated.
func fit(s string, w int) string {
	if s == "" {
		return DimStyle.Width(w).Render("\u2014")
	}
	if lipgloss.Width(s) > w {
		return lipgloss.NewStyle().Width(w).Render(truncate(s, w))
	}
	return lipgloss.NewStyle().Width(w).Render(s)
}

// clamp renders s padded/truncated to exactly w cells with spaces.
// An empty string renders as w spaces.
func clamp(s string, w int) string {
	if s == "" {
		return lipgloss.NewStyle().Width(w).Render("")
	}
	if lipgloss.Width(s) > w {
		return lipgloss.NewStyle().Width(w).Render(truncate(s, w))
	}
	return lipgloss.NewStyle().Width(w).Render(s)
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
// Use this instead of magic number 4 throughout the codebase.
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

// safeWidth returns a clamped width suitable for content rendering.
// Ensures a minimum of 30 columns.
func safeWidth(w int) int {
	if w < 30 {
		return 30
	}
	return w
}

// safeHeight returns a clamped height with the given minimum.
func safeHeight(h, min int) int {
	if h < min {
		return min
	}
	return h
}

// emptyState returns a dimmed placeholder string for empty content areas.
func emptyState() string {
	return DimStyle.Render("\u2014")
}

// renderSectionTitle renders a titled horizontal rule separator.
func renderSectionTitle(title string, width int) string {
	return section(title, width)
}

// truncWithEllipsis truncates a string to max cells with ellipsis if needed.
func truncWithEllipsis(s string, max int) string {
	return truncate(s, max)
}
