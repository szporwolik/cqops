package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
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

// scrollHint returns a visual indicator showing whether content extends
// above or below the visible area. Returns "" when all content fits.
func scrollHint(vp viewport.Model) string {
	total := vp.TotalLineCount()
	visible := vp.VisibleLineCount()
	if total <= visible || total == 0 {
		return ""
	}
	atTop := vp.AtTop()
	atBottom := vp.AtBottom()
	switch {
	case atTop && !atBottom:
		return "  ▼ more below"
	case !atTop && atBottom:
		return "  ▲ more above"
	case !atTop && !atBottom:
		return "  ▲ above · ▼ below"
	default:
		return ""
	}
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
// ANSI escape sequences (e.g. \x1b[38;5;212m) are skipped and don't count
// toward the visual width.
func truncateText(s string, maxW int) string {
	if maxW <= 1 {
		return ""
	}
	// Fast path: most strings don't need truncation. If the content is
	// plain text (no ANSI escapes) and the rune count fits, return early.
	if !strings.ContainsRune(s, '\x1b') && len([]rune(s)) <= maxW {
		return s
	}
	w := 0
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	inEsc := false
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		// Detect ANSI escape start: ESC [ ... (CSI sequences)
		if !inEsc && r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			inEsc = true
			out = append(out, r, runes[i+1])
			i++ // skip the '['
			continue
		}
		if inEsc {
			out = append(out, r)
			// ANSI sequences end with a letter (m, J, H, etc.) or bell
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		rw := 1
		if r > 0xffff {
			rw = 2
		}
		if w+rw >= maxW {
			out = append(out, '\u2026') // ellipsis
			return string(out)
		}
		w += rw
		out = append(out, r)
	}
	return string(out)
}
