package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

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
