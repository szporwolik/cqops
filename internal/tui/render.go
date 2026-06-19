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

// centerAndBorderMap centers each line of a map to the given content width,
// then wraps the result in a bordered box. Used by Partner and PSK Reporter views.
func centerAndBorderMap(mapBox string, contentW, boxW int) string {
	lines := strings.Split(mapBox, "\n")
	for i, l := range lines {
		lw := lipgloss.Width(l)
		if lw > contentW {
			lines[i] = truncateText(l, contentW)
		} else if lw < contentW {
			left := (contentW - lw) / 2
			right := contentW - lw - left
			lines[i] = strings.Repeat(" ", left) + l + strings.Repeat(" ", right)
		}
	}
	return drawBorderedBox(strings.Join(lines, "\n"), boxW)
}

// menuBoxStyle is the bordered box used by config menus.
var menuBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(P.Border).
	Padding(1, 2)

// drawMenuWithHeader renders a title header above a bordered menu box.
// Trailing newlines in the content are trimmed to avoid a blank row
// between the last item and the bottom border.
// Box width is capped at partnerMapMaxW for consistency with the QSO form.
func drawMenuWithHeader(title, content string, boxW int) string {
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	header := S.Title.Width(boxW).Render(title)
	box := menuBoxStyle.Width(boxW).Render(strings.TrimRight(content, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
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

// =============================================================================
// Content helpers
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
		return truncateText(s, w)
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
