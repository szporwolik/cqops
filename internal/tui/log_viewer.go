package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
)

// Pre-allocated log viewer styles — avoids per-frame allocations.
var (
	logTimeStyle  = lipgloss.NewStyle().Width(9).Foreground(P.TextDim)
	logLevelStyle = lipgloss.NewStyle().Width(6)
)

type LogViewer struct {
	viewport viewport.Model
	done     bool
	width    int
	height   int
	logName  string // active logbook name for display

	// Content cache — avoids rebuilding all log lines on every frame.
	// Keyed on the buffer revision: the ring keeps only the newest entries, so
	// the stored count stops moving once it is full while entries keep arriving.
	cachedContent string
	cachedW       int
	cachedH       int
	cachedRev     uint64
}

func NewLogViewer(logName string) *LogViewer {
	lv := &LogViewer{logName: logName}
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	vp.FillHeight = true
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
		vh := contentHeight(msg.Height)
		if vh < 5 {
			vh = 5
		}
		lv.viewport.SetWidth(msg.Width)
		lv.viewport.SetHeight(vh)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			lv.done = true
		case "insert":
			lv.viewport.SetYOffset(0)

		default:
			var cmd tea.Cmd
			lv.viewport, cmd = lv.viewport.Update(msg)
			return lv, cmd
		}
	}
	return lv, nil
}

// needsRebuild reports whether the viewport content has to be rebuilt for the
// given buffer revision and geometry. It is the revision that changes, not the
// number of stored entries: the ring keeps the newest maxStored entries, so once
// it is full the count stays put while new entries keep arriving — a cache keyed
// on the count never refreshes again.
func (lv *LogViewer) needsRebuild(rev uint64, w, h int) bool {
	return lv.cachedContent == "" || lv.cachedRev != rev || lv.cachedW != w || lv.cachedH != h
}

func (lv *LogViewer) View() tea.View {
	if lv.done {
		return tea.NewView("")
	}

	// Viewport height — consistent between dimension sync and cache key.
	vh := contentHeight(lv.height)
	if vh < 5 {
		vh = 5
	}

	// Sync viewport dimensions on first render / before first WindowSizeMsg.
	if lv.width > 0 && (lv.viewport.Width() != lv.width || lv.viewport.Height() != vh) {
		lv.viewport.SetWidth(lv.width)
		lv.viewport.SetHeight(vh)
	}

	// Return the cached view when neither the buffer nor the geometry changed.
	// SetContent is deliberately not repeated here: the viewport already holds
	// these lines, and re-setting them only resets its highlight state.
	if !lv.needsRebuild(applog.Revision(), lv.width, vh) {
		return tea.NewView(lv.viewport.View())
	}

	entries, rev := applog.EntriesWithRevision()
	entryCount := len(entries)
	if entryCount == 0 {
		return tea.NewView(DimStyle.Render("  No log entries yet."))
	}

	// Build colored log lines (newest first).
	bodyW := lv.width
	if bodyW < 40 {
		bodyW = 80
	}
	msgW := bodyW - 15 // 9 (time) + 6 (level)
	if msgW < 10 {
		msgW = 10
	}

	var b strings.Builder
	for i := entryCount - 1; i >= 0; i-- {
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
			msg = msg + "  " + e.Details
		}

		b.WriteString(logTimeStyle.Render(e.Time))
		b.WriteString(logLevelStyle.Render(ls.Render(e.Level)))
		b.WriteString(ValueStyle.Render(truncateText(msg, msgW)))
		if i > 0 {
			b.WriteByte('\n')
		}
	}

	content := b.String()
	lv.viewport.SetContent(content)

	lv.cachedContent = content
	lv.cachedW = lv.width
	lv.cachedH = vh
	lv.cachedRev = rev

	return tea.NewView(lv.viewport.View())
}
