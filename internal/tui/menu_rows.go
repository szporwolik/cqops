package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// rowStyle is a set of label styles for a config menu's rows. Menus pick
// the width variant that fits their labels; the shared row renderers below
// produce the same layout for every menu.
type rowStyle struct {
	label   lipgloss.Style
	focused lipgloss.Style
}

// checkboxRow renders a "[x]" row. When focused it shows the "> " marker,
// a highlighted label and a "(Space)" hint (plus hint when non-empty).
// dim renders the label in DimStyle (e.g. disabled sub-options).
func checkboxRow(b *strings.Builder, w int, focused bool, label string, checked bool, hint string, dim bool, st rowStyle) {
	cb := "[ ]"
	if checked {
		cb = "[x]"
	}
	prefix := S.FormPrefixOff.Render("  ")
	lbl := st.label.Align(lipgloss.Left).Render(label)
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = st.focused.Align(lipgloss.Left).Render(label)
		cb = CursorStyle.Render(cb) + " " + DimStyle.Render("(Space)")
		if hint != "" {
			cb += " " + DimStyle.Render(hint)
		}
	}
	if dim {
		lbl = DimStyle.Render(st.label.Align(lipgloss.Left).Render(label))
	}
	b.WriteString(padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", cb), w))
	b.WriteString("\n")
}

// valueRow renders a label/value row (e.g. Units, Timezone). When focused it
// shows the "> " marker and a "(Space)" hint.
func valueRow(b *strings.Builder, w int, focused bool, label, value, hint string, st rowStyle) {
	prefix := S.FormPrefixOff.Render("  ")
	lbl := st.label.Align(lipgloss.Left).Render(label)
	val := ValueStyle.Render(value)
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = st.focused.Align(lipgloss.Left).Render(label)
		val = CursorStyle.Render(value) + " " + DimStyle.Render("(Space)")
		if hint != "" {
			val += " " + DimStyle.Render(hint)
		}
	}
	b.WriteString(padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val), w))
	b.WriteString("\n")
}

// buttonRow renders an action button row (e.g. Test notification). The
// fixed padding keeps buttons from shifting when the marker appears.
func buttonRow(b *strings.Builder, w int, focused bool, text string) {
	prefix := "    "
	styled := InputStyle.Render(text)
	if focused {
		prefix = S.FormPrefixOn.Render("> ") + "  "
		styled = CursorStyle.Render(text)
	}
	b.WriteString(padOrTrunc(prefix+styled, w))
	b.WriteString("\n")
}

// infoBox writes a dimmed info paragraph wrapped in a rounded border.
// maxW is the available text width (clamped to at least 30).
func infoBox(b *strings.Builder, text string, maxW int) {
	if maxW < 30 {
		maxW = 30
	}
	lines := wrapLines(text, maxW)
	var content strings.Builder
	for i, line := range lines {
		content.WriteString(DimStyle.Render(line))
		if i < len(lines)-1 {
			content.WriteString("\n")
		}
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(P.Border)
	b.WriteString(boxStyle.Render(content.String()))
	b.WriteString("\n")
}
