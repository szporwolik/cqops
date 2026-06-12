package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/szporwolik/cqops/internal/log"
)

type LogViewer struct {
	name   string
	offset int
	done   bool
	width  int
	height int
}

func NewLogViewer(name string) *LogViewer { return &LogViewer{name: name} }

func (lv *LogViewer) Init() tea.Cmd { return nil }

func (lv *LogViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lv.width, lv.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "f9":
			lv.done = true
		case "up", "k":
			if lv.offset > 0 {
				lv.offset--
			}
		case "down", "j":
			lv.offset++
		}
	}
	return lv, nil
}

func (lv *LogViewer) FooterText() string {
	return "↑↓ to scroll  F1 QSO form  F9 to close"
}

func (lv *LogViewer) View() string {
	if lv.done {
		return ""
	}
	entries := log.Entries()
	if len(entries) == 0 {
		return "No log entries yet."
	}

	bodyW := lv.width - 2
	if bodyW < 30 {
		bodyW = 30
	}

	maxRows := lv.height - 5
	if maxRows < 3 {
		maxRows = 10
	}

	if lv.offset > len(entries)-maxRows {
		lv.offset = len(entries) - maxRows
	}
	if lv.offset < 0 {
		lv.offset = 0
	}

	infoColor := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	errColor := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnColor := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	debugColor := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var b strings.Builder

	title := "── Logs: " + lv.name + " "
	rem := bodyW - lipgloss.Width(title)
	if rem > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + strings.Repeat("─", rem)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title))
	}
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  %-8s %-6s %s", "Time", "Level", "Message"))
	b.WriteString("\n\n")

	for i := len(entries) - 1 - lv.offset; i >= 0 && len(entries)-1-i-lv.offset < maxRows; i-- {
		if i < 0 || i >= len(entries) {
			continue
		}
		e := entries[i]

		levelStyle := debugColor
		switch e.Level {
		case "ERROR":
			levelStyle = errColor
		case "WARN":
			levelStyle = warnColor
		case "INFO":
			levelStyle = infoColor
		}

		msg := e.Message
		if e.Details != "" {
			msg += " — " + e.Details
		}

		line := fmt.Sprintf("  %-8s %s %s",
			e.Time,
			levelStyle.Render(fmt.Sprintf("%-6s", e.Level)),
			trunc(msg, bodyW-18),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
