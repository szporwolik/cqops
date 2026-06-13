package tui

import (
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
