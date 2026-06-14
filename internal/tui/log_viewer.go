package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	vp.FillHeight = true // always fill allocated height to push help bar down
	lv.viewport = vp
	return lv
}

func (lv *LogViewer) Init() tea.Cmd { return nil }

// ScrollInfo returns a short string for the help bar showing viewport position.
func (lv *LogViewer) ScrollInfo() string {
	total := lv.viewport.TotalLineCount()
	visible := lv.viewport.VisibleLineCount()
	atTop := lv.viewport.AtTop()
	atBottom := lv.viewport.PastBottom()
	if total <= visible {
		return fmt.Sprintf("log %d/%d", total, total)
	}
	if atTop {
		return fmt.Sprintf("log ↓ %d/%d", visible, total)
	}
	if atBottom {
		return fmt.Sprintf("log ↑ %d/%d", visible, total)
	}
	return fmt.Sprintf("log ↕ %d/%d", visible, total)
}

func (lv *LogViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lv.width = msg.Width
		lv.height = msg.Height
		// Viewport fills ContentH exactly: status+profile+tab+help = 4 fixed rows.
		vh := contentHeight(msg.Height)
		if vh < 5 {
			vh = 5
		}
		lv.viewport.SetWidth(msg.Width)
		lv.viewport.SetHeight(vh)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "f8":
			lv.done = true
		case "insert":
			// Scroll to top and refresh content.
			lv.viewport.SetYOffset(0)
		default:
			var cmd tea.Cmd
			lv.viewport, cmd = lv.viewport.Update(msg)
			return lv, cmd
		}
	}
	return lv, nil
}

func (lv *LogViewer) View() tea.View {
	if lv.done {
		return tea.NewView("")
	}

	// Sync viewport dimensions from stored width/height (covers first render
	// before any WindowSizeMsg arrives).
	if lv.width > 0 && lv.height > 0 {
		vh := contentHeight(lv.height)
		if vh < 5 {
			vh = 5
		}
		if lv.viewport.Width() != lv.width || lv.viewport.Height() != vh {
			lv.viewport.SetWidth(lv.width)
			lv.viewport.SetHeight(vh)
		}
	}

	entries := applog.Entries()
	if len(entries) == 0 {
		return tea.NewView(DimStyle.Render("  No log entries yet."))
	}

	// Build colored log lines (newest first).
	timeStyle := lipgloss.NewStyle().Width(9).Foreground(P.TextDim).PaddingRight(0)
	levelStyleW := lipgloss.NewStyle().Width(6)
	bodyW := lv.width
	if bodyW < 40 {
		bodyW = 80
	}
	msgW := bodyW - 16
	if msgW < 10 {
		msgW = 10
	}

	var lines []string
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]

		ls := S.LogDebug
		switch e.Level {
		case "ERROR":
			ls = S.LogError
		case "WARN":
			ls = S.LogWarn
		case "INFO":
			ls = S.LogInfo
		}

		msg := e.Message
		if e.Details != "" {
			msg += "  " + e.Details
		}

		lines = append(lines,
			lipgloss.JoinHorizontal(lipgloss.Top,
				timeStyle.Render(e.Time),
				levelStyleW.Render(ls.Render(e.Level)),
				S.ValueStyle.Render(truncate(msg, msgW)),
			),
		)
	}

	lv.viewport.SetContent(strings.Join(lines, "\n"))

	// Viewport fills all rows; main View's MaxHeight + JoinVertical places
	// the help bar at the bottom with any extra space as gap.
	return tea.NewView(lv.viewport.View())
}
