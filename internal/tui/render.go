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

// drawBorderedBoxPad is like drawBorderedBox but with Padding(1,2) for
// content-heavy forms like config menus.
var menuBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(P.Border).
	Padding(1, 2)

func drawMenuBox(content string, boxW int) string {
	return menuBoxStyle.Width(boxW).Render(content)
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

// =============================================================================
// Shared form-field rendering — used by QSO form and all config menus.
// =============================================================================

// fromField renders a label+value line for a textinput.Model in a form.
// When focused, the "> " cursor prefix is shown and the textinput View() is
// used so the cursor blinks. When unfocused, empty values show an em-dash.
// labelWide: use wider label (14 cells) for menus; use narrow (11) for QSO form.
func formField(label string, ti *textinput.Model, focused bool, labelWide bool) string {
	raw := strings.TrimSpace(ti.Value())

	var labelStyle lipgloss.Style
	if labelWide {
		labelStyle = S.FormLabelWide
	} else {
		labelStyle = S.FormLabel
	}

	var prefix, lbl, val string
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		if labelWide {
			lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
		} else {
			lbl = S.FormFocused.Align(lipgloss.Left).Render(label)
		}
		val = ti.View()
	} else {
		prefix = S.FormPrefixOff.Render("  ")
		lbl = labelStyle.Align(lipgloss.Left).Render(label)
		if raw == "" {
			val = DimStyle.Render("\u2014")
		} else {
			val = ValueStyle.Render(raw)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
}

// formFieldLine returns formField result padded to exactly the given width.
func formFieldLine(label string, ti *textinput.Model, focused bool, labelWide bool, width int) string {
	return padOrTrunc(formField(label, ti, focused, labelWide), width)
}

// =============================================================================
// Legacy helpers — kept for backward compat; prefer formField above.
// =============================================================================

// fillBody returns the content with trailing newlines so the total height
// equals contentH.
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

// --- standalone utilities ---

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
// status bar (1) + tab bar (3 with borders) + help bar (1).
const FixedZoneHeight = 5

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
