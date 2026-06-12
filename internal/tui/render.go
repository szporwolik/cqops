package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func section(title string, width int) string {
	rem := width - lipgloss.Width(title)
	if rem > 0 {
		return SectionStyle.Render(title + strings.Repeat("─", rem))
	}
	return SectionStyle.Render(title)
}

func fit(s string, w int) string {
	if s == "" {
		return DimStyle.Render(strings.Repeat("—", 1))
	}
	if lipgloss.Width(s) > w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func ellipsis(s string, w int) string {
	if s == "" {
		return DimStyle.Render("—")
	}
	return ValueStyle.Render(truncate(s, w))
}

func kv(label, value string, labelW, valueW int, focused bool) string {
	lbl := fit(label, labelW)
	val := fit(value, valueW)
	if !focused {
		lbl = LabelStyle.Render(lbl)
	} else {
		lbl = CursorStyle.Render(lbl)
	}
	if strings.TrimSpace(value) == "" {
		val = DimStyle.Render("—")
	} else {
		val = ValueStyle.Render(val)
	}
	return fmt.Sprintf("  %s %s", lbl, val)
}

func tableRow(cols []string, widths []int) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		if c == "" {
			c = "—"
		}
		if len(c) > widths[i] {
			c = c[:widths[i]]
		}
		parts[i] = fmt.Sprintf("%-*s", widths[i], c)
	}
	return strings.Join(parts, " ")
}
