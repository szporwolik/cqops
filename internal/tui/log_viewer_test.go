package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
)

// The log viewer caches the rendered lines. The buffer behind it is a ring that
// keeps only the newest entries, so the stored COUNT stops changing once the
// ring is full while entries keep arriving — a cache keyed on the count never
// refreshes again. The buffer revision is the thing that moves.

// logViewerTestFrames renders the viewer at a fixed size and returns the visible
// frame with ANSI stripped.
func logViewerTestFrames(t *testing.T, lv *LogViewer) string {
	t.Helper()
	return stripANSI(lv.View().Content)
}

// TestLogViewerShowsEntriesAfterTheRingIsFull is the reported bug: the buffer
// holds 100 entries, entries 101-150 arrived, and the view stayed on the stale
// content because the count was still 100.
func TestLogViewerShowsEntriesAfterTheRingIsFull(t *testing.T) {
	lv := NewLogViewer("test log")
	lv.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	for i := 0; i < 100; i++ {
		applog.Append("INFO", fmt.Sprintf("ring-entry-%03d", i), "")
	}
	first := logViewerTestFrames(t, lv)
	if !strings.Contains(first, "ring-entry-099") {
		t.Fatalf("precondition: the newest entry is not on screen:\n%s", first)
	}

	// The operator keeps working: 50 more entries arrive while the ring stays
	// at its 100-entry limit.
	for i := 100; i < 150; i++ {
		applog.Append("INFO", fmt.Sprintf("ring-entry-%03d", i), "")
	}
	second := logViewerTestFrames(t, lv)
	if !strings.Contains(second, "ring-entry-149") {
		t.Errorf("the viewer did not refresh once the buffer was full — the newest entry is missing:\n%s", second)
	}
	if strings.Contains(second, "ring-entry-099") {
		t.Errorf("the viewer is still showing the content from before the buffer filled up:\n%s", second)
	}
	if strings.Contains(second, "ring-entry-049") {
		t.Errorf("an evicted entry is still on screen:\n%s", second)
	}
}

// TestLogViewerCacheFollowsTheRevision pins the cache rule itself: an unchanged
// revision means no rebuild (and therefore no SetContent), a new entry or a new
// geometry means one.
func TestLogViewerCacheFollowsTheRevision(t *testing.T) {
	lv := NewLogViewer("test log")
	lv.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	applog.Append("INFO", "revision-probe", "")
	lv.View()

	vh := contentHeight(lv.height)
	if lv.needsRebuild(applog.Revision(), lv.width, vh) {
		t.Error("the frame asks for a rebuild with an unchanged revision")
	}
	applog.Append("INFO", "revision-probe-2", "")
	if !lv.needsRebuild(applog.Revision(), lv.width, vh) {
		t.Error("a new entry did not invalidate the cached frame")
	}
	lv.View()
	if !lv.needsRebuild(applog.Revision(), lv.width, vh-1) {
		t.Error("a geometry change did not invalidate the cached frame")
	}
}

// TestLogViewerKeepsTheScrollPosition: refreshing the frame must not move the
// viewport, neither when nothing new arrived nor when entries did.
func TestLogViewerKeepsTheScrollPosition(t *testing.T) {
	lv := NewLogViewer("test log")
	lv.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	for i := 0; i < 100; i++ {
		applog.Append("INFO", fmt.Sprintf("scroll-entry-%03d", i), "")
	}
	lv.View()

	lv.viewport.SetYOffset(40)
	lv.View() // nothing new — a redraw
	if got := lv.viewport.YOffset(); got != 40 {
		t.Errorf("an idle redraw moved the viewport to offset %d, want 40", got)
	}

	applog.Append("INFO", "scroll-entry-new", "")
	lv.View() // one new entry — the ring still holds 100 lines
	if got := lv.viewport.YOffset(); got != 40 {
		t.Errorf("the refresh after a new entry moved the viewport to offset %d, want 40", got)
	}

	// Insert asks for the newest entry, which is the top of the newest-first list.
	upd, _ := lv.Update(tea.KeyPressMsg{Code: tea.KeyInsert})
	if upd.(*LogViewer).viewport.YOffset() != 0 {
		t.Errorf("Insert did not return to the newest entry (offset %d)", lv.viewport.YOffset())
	}
}

// TestLogViewerLineFormat: the fix is a cache key, not a renderer change — a
// line still starts with the fixed-width time and level columns.
func TestLogViewerLineFormat(t *testing.T) {
	lv := NewLogViewer("test log")
	lv.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	applog.Append("WARN", "format-probe", "the details")
	lv.View()

	lines := strings.Split(logViewerTestFrames(t, lv), "\n")
	if len(lines) == 0 {
		t.Fatal("no lines rendered")
	}
	first := lines[0]
	if !strings.Contains(first, "WARN") || !strings.Contains(first, "format-probe") || !strings.Contains(first, "the details") {
		t.Fatalf("the rendered line lost part of the entry: %q", first)
	}
	// time column (9) + level column (6) before the message.
	if idx := strings.Index(first, "WARN"); idx != 9 {
		t.Errorf("the level column starts at %d, want 9 (time column is fixed width)", idx)
	}
	if idx := strings.Index(first, "format-probe"); idx != 15 {
		t.Errorf("the message starts at %d, want 15", idx)
	}
}
