package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
)

type LogViewer struct {
	name     string
	viewport viewport.Model
	done     bool
	width    int
	height   int
}

func NewLogViewer(name string) *LogViewer {
	lv := &LogViewer{name: name}
	lv.viewport = viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	return lv
}

func (lv *LogViewer) Init() tea.Cmd { return nil }

func (lv *LogViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lv.width = msg.Width
		lv.height = msg.Height
		lv.viewport.SetWidth(msg.Width - 2)
		lv.viewport.SetHeight(msg.Height - 10)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f9":
			lv.done = true
		default:
			var cmd tea.Cmd
			lv.viewport, cmd = lv.viewport.Update(msg)
			return lv, cmd
		}
	}
	return lv, nil
}

func (lv *LogViewer) FooterText() string {
	return "↑↓/j,k scroll  PgUp/PgDn page  F9 close"
}

func (lv *LogViewer) View() tea.View {
	if lv.done {
		return tea.NewView("")
	}
	entries := applog.Entries()
	if len(entries) == 0 {
		return tea.NewView("No log entries yet.")
	}

	w := lv.width
	if w < 40 {
		w = 80
	}
	bodyW := w - 2

	infoColor := S.LogInfo
	errColor := S.LogError
	warnColor := S.LogWarn
	debugColor := S.LogDebug

	var b strings.Builder
	for i := len(entries) - 1; i >= 0; i-- {
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

		line := fmt.Sprintf("%-8s %s %s",
			e.Time,
			levelStyle.Render(fmt.Sprintf("%-6s", e.Level)),
			truncate(msg, bodyW-18),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	lv.viewport.SetContent(b.String())

	title := "── Logs: " + lv.name + " "
	header := section(title, bodyW)

	return tea.NewView(header + "\n\n" + lv.viewport.View())
}
