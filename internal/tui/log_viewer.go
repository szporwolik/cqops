package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/szporwolik/cqops/internal/log"
)

type LogViewer struct {
	offset  int
	done    bool
	width   int
	height  int
}

func NewLogViewer() *LogViewer { return &LogViewer{} }

func (lv *LogViewer) Init() tea.Cmd { return nil }

func (lv *LogViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg: lv.width, lv.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "f9": lv.done = true
		case "up", "k": if lv.offset > 0 { lv.offset-- }
		case "down", "j": lv.offset++
		}
	}
	return lv, nil
}

func (lv *LogViewer) FooterText() string {
	return "↑↓ to scroll  F1 QSO form  Esc/F9 to close"
}

func (lv *LogViewer) View() string {
	if lv.done { return "" }
	entries := log.Entries()
	if len(entries) == 0 { return "No log entries yet." }

	maxRows := lv.height - 4
	if maxRows < 3 { maxRows = 10 }

	if lv.offset > len(entries)-maxRows { lv.offset = len(entries) - maxRows }
	if lv.offset < 0 { lv.offset = 0 }

	var b strings.Builder
	b.WriteString(titleStyle.Render("Log Viewer"))
	b.WriteString("\n\n")

	for i := len(entries) - 1 - lv.offset; i >= 0 && len(entries)-1-i-lv.offset < maxRows; i-- {
		if i < 0 || i >= len(entries) { continue }
		e := entries[i]
		style := helpStyle
		switch e.Level {
		case "ERROR": style = errorStyle
		case "WARN": style = warningStyle
		case "DEBUG": style = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		}
		line := fmt.Sprintf("%s [%s] %s", e.Time, e.Level, e.Message)
		if e.Details != "" { line += " — " + e.Details }
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}
