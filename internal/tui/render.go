package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
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

// --- standalone utilities (moved from model.go) ---

func clamp(s string, w int) string {
	if s == "" {
		return strings.Repeat(" ", w)
	}
	if lipgloss.Width(s) > w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

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

func trunc(s string, w int) string {
	if s == "" {
		return ""
	}
	if len(s) > w {
		return s[:w]
	}
	return s
}

func toAny(ss []string) []any {
	aa := make([]any, len(ss))
	for i, s := range ss {
		aa[i] = s
	}
	return aa
}
