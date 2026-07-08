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

// menuBoxStyle is the box used by config menus — no border, light padding.
var menuBoxStyle = lipgloss.NewStyle().Padding(0, 2)

// drawMenuWithHeader renders a title header, a blank row, then the content.
// Box width is capped at partnerMapMaxW for consistency with the QSO form.
// Prefer renderScrollableMenu for lists that may overflow the terminal height.
func drawMenuWithHeader(title, content string, boxW int) string {
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	header := S.Title.Width(boxW).Render(title)
	box := menuBoxStyle.Width(boxW).Render(strings.TrimRight(content, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, header, "", box)
}

// renderScrollableMenu wraps content in a viewport-based menu layout:
// header, blank row, scrollable viewport, scroll hint — all inside menuBoxStyle.
// The viewport fills the available terminal height so the hint sits just
// above the bottom help bar. This is used by list/edit views that may
// overflow on small terminals.
func renderScrollableMenu(headerTitle string, content string, vp *viewport.Model, lastContent *string, w, h int) string {
	boxW := w
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	vpW := boxW - 4 // menuBoxStyle left+right padding
	if vpW < 20 {
		vpW = 20
	}
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}
	// Overhead: header(1) + blank row(1) + scroll hint(1) = 3 lines.
	vpH := contentH - 3
	if vpH < 4 {
		vpH = 4
	}
	header := S.Title.Width(boxW).Render(headerTitle)
	vp.SetWidth(vpW)
	vp.SetHeight(vpH)
	if vp.TotalLineCount() == 0 || content != *lastContent {
		vp.SetContent(content)
		*lastContent = content
	}
	if vp.PastBottom() {
		vp.SetYOffset(vp.TotalLineCount() - vp.VisibleLineCount())
	}
	vpContent := vp.View()
	hintLine := DimStyle.Width(vpW).Render(scrollHint(*vp))
	if hintLine == "" {
		hintLine = strings.Repeat(" ", vpW)
	}
	vpContent = lipgloss.JoinVertical(lipgloss.Left, vpContent, hintLine)
	box := menuBoxStyle.Width(boxW).Render(vpContent)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", box)
}

// osc8Link returns an OSC-8 hyperlink sequence. Most modern terminals
// (Windows Terminal, iTerm2, Kitty, etc.) render these as clickable links.
// Ctrl+click opens the URL in the system browser.
func osc8Link(url, text string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}

// scrollVpToLine ensures the given line index is visible in the viewport,
// scrolling the viewport if needed. line is the 0-based index in the content.
func scrollVpToLine(vp *viewport.Model, line int) {
	if line < 0 || vp.TotalLineCount() == 0 {
		return
	}
	visible := vp.VisibleLineCount()
	if visible <= 0 {
		return
	}
	top := vp.YOffset()
	bottom := top + visible - 1
	if line < top {
		vp.SetYOffset(line)
	} else if line > bottom {
		vp.SetYOffset(line - visible + 1)
	}
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
// equals contentH. Each trailing line carries a single space to force
// Bubble Tea's render diff to write to every line.
func fillBody(content string, contentH int) string {
	return fillBodyEpoch(content, contentH, 0)
}

// fillBodyEpoch is like fillBody but includes an epoch in the padding
// to force a full render diff on screen transitions. When epoch > 0 and
// has changed since the last frame, every padded line differs from the
// previous frame, clearing stale content on Linux terminals.
func fillBodyEpoch(content string, contentH int, epoch uint64) string {
	if contentH <= 0 {
		return content
	}
	current := lipgloss.Height(content)
	if current >= contentH {
		return content
	}
	// Use non-breaking space (U+00A0) when forcing clear — it differs
	// from the regular space (U+0020) normally used, making cellbuf
	// detect a change on every padded line.
	pad := " "
	if epoch > 0 && epoch%2 == 1 {
		pad = "\u00A0"
	}
	return content + strings.Repeat(pad+"\n", contentH-current)
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
