package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// A page read that fails must not look like an empty logbook. The editor used
// to assign an empty row set on error, so a transient database problem wiped
// the visible contacts — and with them the operator's place in the log.
//
// The failure is forced by renaming the table for the duration of the read: the
// editor keeps its *sql.DB, so this is the same connection a real transient
// error would come from, and the very next load is a normal one.

// breakQSOs makes every qsos read fail until restoreQSOs is called.
func breakQSOs(t *testing.T, le *LogbookEditor) (restore func()) {
	t.Helper()
	if _, err := le.db.Exec(`ALTER TABLE qsos RENAME TO qsos_unreadable`); err != nil {
		t.Fatalf("hide qsos: %v", err)
	}
	return func() {
		if _, err := le.db.Exec(`ALTER TABLE qsos_unreadable RENAME TO qsos`); err != nil {
			t.Fatalf("restore qsos: %v", err)
		}
	}
}

// TestFailedPageLoadKeepsThePageOnScreen: a read error must leave the rows, the
// page number and the cursor exactly as they were, and report itself instead of
// presenting an empty table.
func TestFailedPageLoadKeepsThePageOnScreen(t *testing.T) {
	le, all := pagingTestEditor(t, 120)
	le.goToPage(2)        // absolute rows 20..39
	le.table.SetCursor(5) // absolute row 25
	before := le.table.Cursor() + 1
	if before != 6 {
		t.Fatalf("setup: cursor at row %d, want 6", before)
	}
	rowsBefore := append([]int64(nil), qsoIDs(le)...)
	pageBefore, countBefore := le.currentPage, le.totalCount
	// The whole frame, not just the rows: the layout must be identical too —
	// nothing may grow, shrink or move because a read failed.
	frameBefore := le.View().Content

	restore := breakQSOs(t, le)
	defer restore()
	if err := le.goToPage(3); err == nil {
		t.Fatal("goToPage succeeded while qsos was unreadable")
	}

	if got := qsoIDs(le); !equalIDs(got, rowsBefore) {
		t.Fatalf("rows changed after a failed read:\n got %v\nwant %v", got, rowsBefore)
	}
	if le.currentPage != pageBefore {
		t.Errorf("page %d after a failed read, want %d", le.currentPage, pageBefore)
	}
	if le.totalCount != countBefore {
		t.Errorf("count %d after a failed read, want %d", le.totalCount, countBefore)
	}
	if got := le.table.Cursor() + 1; got != before {
		t.Errorf("cursor at row %d after a failed read, want %d", got, before)
	}

	// The rendered table still shows the contacts — an empty table is what the
	// bug looked like.
	view := le.View().Content
	if view != frameBefore {
		t.Errorf("the frame changed after a failed read (layout shift):\n%s", view)
	}
	for _, want := range []string{"T020", "T025", "T039"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view is missing %s after a failed read:\n%s", want, view)
		}
	}
	if !strings.Contains(view, all[25].Call) {
		t.Fatalf("view no longer shows the selected contact %s", all[25].Call)
	}
}

// TestPageLoadReportsTheFailure: the caller learns about the error (the owner
// loop turns it into a toast), and a resize or page key does not silently do
// nothing.
func TestPageLoadReportsTheFailure(t *testing.T) {
	le, _ := pagingTestEditor(t, 60)
	restore := breakQSOs(t, le)
	defer restore()

	if err := le.loadPage(); err == nil {
		t.Fatal("loadPage swallowed the read error")
	}

	// A message that needs no reload must not swallow a pending failure either:
	// the resize path reports through the same command.
	_, cmd := le.Update(tea.WindowSizeMsg{Width: 120, Height: 49})
	if cmd == nil {
		t.Fatal("a failed reload after a resize produced no command")
	}
	msg, ok := cmd().(editorMsg)
	if !ok || msg.err == nil {
		t.Fatalf("resize result is %#v, want an editorMsg carrying the error", msg)
	}
}

// TestPageLoadRecoversAfterAFailure: the retry installs the page it asked for,
// with the right rows and cursor — the failed attempt left nothing behind.
func TestPageLoadRecoversAfterAFailure(t *testing.T) {
	le, all := pagingTestEditor(t, 120)
	le.goToPage(2)
	le.table.SetCursor(5)

	restore := breakQSOs(t, le)
	if err := le.goToPage(3); err == nil {
		t.Fatal("goToPage succeeded while qsos was unreadable")
	}
	restore()

	if err := le.goToPage(3); err != nil {
		t.Fatalf("retry after the failure: %v", err)
	}
	if le.currentPage != 3 {
		t.Fatalf("page %d after the retry, want 3", le.currentPage)
	}
	if len(le.qsos) != 20 {
		t.Fatalf("%d rows after the retry, want 20", len(le.qsos))
	}
	if got, want := le.qsos[0].ID, all[40].ID; got != want {
		t.Errorf("first row of page 3 is %d, want %d", got, want)
	}
	if got, want := selectedID(t, le), all[40].ID; got != want {
		t.Errorf("cursor is on %d after the retry, want the top of page 3 (%d)", got, want)
	}
}

// TestFailedReloadAfterResizeKeepsTheGeometry: the page size is part of the
// snapshot too — a failed re-page must not leave the editor claiming a page
// size that no rows were loaded at.
func TestFailedReloadAfterResizeKeepsTheGeometry(t *testing.T) {
	le, all := pagingTestEditor(t, 120)
	le.goToPage(2)
	le.table.SetCursor(5) // absolute row 25
	want := all[25].ID

	restore := breakQSOs(t, le)
	defer restore()
	le.Update(tea.WindowSizeMsg{Width: 120, Height: 49}) // page size would be 40

	if le.pageSize != 20 {
		t.Errorf("page size %d after a failed re-page, want the loaded 20", le.pageSize)
	}
	if le.currentPage != 2 {
		t.Errorf("page %d after a failed re-page, want 2", le.currentPage)
	}
	if got := len(le.qsos); got != 20 {
		t.Errorf("%d rows after a failed re-page, want the loaded 20", got)
	}
	if got := selectedID(t, le); got != want {
		t.Errorf("cursor moved to %d after a failed re-page, want %d", got, want)
	}
	if le.pageSize == le.pageSizeForHeight() {
		t.Fatal("test precondition: the resize should have asked for a different page size")
	}
}

func qsoIDs(le *LogbookEditor) []int64 {
	ids := make([]int64, len(le.qsos))
	for i, q := range le.qsos {
		ids[i] = q.ID
	}
	return ids
}

func equalIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestPageErrorCommandIsNilOnSuccess keeps the plumbing honest: a successful
// update must not queue an empty toast.
func TestPageErrorCommandIsNilOnSuccess(t *testing.T) {
	if cmd := editorErrorCmd(nil); cmd != nil {
		t.Fatalf("editorErrorCmd(nil) = %v, want nil", cmd)
	}
	msg := editorErrorCmd(fmt.Errorf("boom"))().(editorMsg)
	if msg.err == nil || msg.err.Error() != "boom" {
		t.Fatalf("editorErrorCmd error message = %#v, want boom", msg.err)
	}
}
