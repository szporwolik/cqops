package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Selection across a resize
//
// The editor's selection anchor used to be the row *position*, and a resize
// re-reads the page from the database. A contact appended while the page was on
// screen shifts every position below it, so the same position pointed at a
// neighbour: the operator selected one contact, WSJT-X logged another, the
// terminal was enlarged, and the cursor sat on a different contact — the next
// edit or delete would have hit the wrong row.
// =============================================================================

// seedEditorContacts inserts n contacts, one per day, oldest first, and returns
// the ids in insertion order (so ids[n-1] is the newest contact).
func seedEditorContacts(t *testing.T, le *LogbookEditor, n int) []int64 {
	t.Helper()
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		id := insertQSO(t, le, &qso.QSO{
			Call:    fmt.Sprintf("T%02dAA", i),
			QSODate: fmt.Sprintf("2026%02d%02d", 1+i/28, 1+i%28),
			TimeOn:  "120000",
			Band:    "20m",
			Mode:    "SSB",
		})
		ids = append(ids, id)
	}
	return ids
}

func TestEditorResizeKeepsTheSelectionWhenANewerContactArrives(t *testing.T) {
	le := newEditorWithDB(t)
	ids := seedEditorContacts(t, le, 20)
	if err := le.loadPage(); err != nil {
		t.Fatalf("loadPage: %v", err)
	}
	if le.pageSize != 15 {
		t.Fatalf("page size = %d at height %d; want 15", le.pageSize, le.height)
	}

	// Second row from the top: the contact the operator is looking at.
	le.table.SetCursor(1)
	want := ids[18]
	if got := selectedID(t, le); got != want {
		t.Fatalf("setup: selected id %d, want %d", got, want)
	}

	// A newer contact arrives while the page is on screen (WSJT-X log).
	insertQSO(t, le, &qso.QSO{Call: "NEW1AA", QSODate: "20261231", TimeOn: "235900", Band: "20m", Mode: "FT8"})

	// The operator enlarges the terminal.
	le.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	if got := selectedID(t, le); got != want {
		t.Errorf("the resize moved the selection from id %d to id %d — the next edit would hit the wrong contact", want, got)
	}
}

// TestEditorResizeFollowsTheContactAcrossAPageBoundary covers the case the
// loaded page cannot answer: the appended contact pushed the selection onto the
// next page, so the editor has to ask where the contact is now.
func TestEditorResizeFollowsTheContactAcrossAPageBoundary(t *testing.T) {
	le := newEditorWithDB(t)
	ids := seedEditorContacts(t, le, 40)
	if err := le.loadPage(); err != nil {
		t.Fatalf("loadPage: %v", err)
	}

	// Last row of page 1.
	le.table.SetCursor(14)
	want := ids[len(ids)-15]
	if got := selectedID(t, le); got != want {
		t.Fatalf("setup: selected id %d, want %d", got, want)
	}

	// Two newer contacts: the selection is now on page 2.
	insertQSO(t, le, &qso.QSO{Call: "NEW1AA", QSODate: "20261230", TimeOn: "235800", Band: "20m", Mode: "FT8"})
	insertQSO(t, le, &qso.QSO{Call: "NEW2AA", QSODate: "20261231", TimeOn: "235900", Band: "20m", Mode: "FT8"})

	// A shorter terminal re-reads the page at the smaller size, and the old
	// position (14) now holds a different contact — the selection is on the
	// next page.
	le.Update(tea.WindowSizeMsg{Width: le.width, Height: 20})

	if got := selectedID(t, le); got != want {
		t.Errorf("the resize moved the selection from id %d to id %d across the page boundary", want, got)
	}
}

// TestEditorResizeSurvivesADeletedSelection: the anchor contact is gone, so the
// editor keeps the clamped position instead of failing the resize.
func TestEditorResizeSurvivesADeletedSelection(t *testing.T) {
	le := newEditorWithDB(t)
	ids := seedEditorContacts(t, le, 20)
	if err := le.loadPage(); err != nil {
		t.Fatalf("loadPage: %v", err)
	}
	le.table.SetCursor(3)
	gone := ids[16]

	if _, err := le.db.Exec(`DELETE FROM qsos WHERE id = ?`, gone); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, cmd := le.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if cmd != nil {
		if msg, ok := cmd().(editorMsg); ok && msg.err != nil {
			t.Fatalf("resize after a deletion reported an error: %v", msg.err)
		}
	}
	if row := le.table.Cursor(); row < 0 || row >= len(le.qsos) {
		t.Fatalf("cursor %d is outside the %d reloaded rows", row, len(le.qsos))
	}
	if got := selectedID(t, le); got == gone {
		t.Errorf("the deleted contact id %d is still selected", gone)
	}
}

// TestEditorResizeKeepsTheSelectionWhileSearching: a search result is not paged,
// so a resize only rebuilds the table — but rebuilding replaces the table (and
// its cursor), which used to drop the selection to the top of the list.
func TestEditorSearchResizeKeepsTheSelection(t *testing.T) {
	le := newEditorWithDB(t)
	seedEditorContacts(t, le, 30)
	for i := 0; i < 5; i++ {
		insertQSO(t, le, &qso.QSO{
			Call:    fmt.Sprintf("SP9ZZ%d", i),
			QSODate: fmt.Sprintf("202603%02d", 1+i),
			TimeOn:  "100000",
			Band:    "40m",
			Mode:    "CW",
		})
	}

	le.searchQuery = "SP9ZZ"
	if err := le.applySearchFilter(); err != nil {
		t.Fatalf("applySearchFilter: %v", err)
	}
	if len(le.qsos) != 5 {
		t.Fatalf("search returned %d rows; want 5", len(le.qsos))
	}

	le.table.SetCursor(3)
	want := selectedID(t, le)

	le.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if got := selectedID(t, le); got != want {
		t.Errorf("the resize reset the selection from id %d to id %d while searching", want, got)
	}
}

// TestEditorSearchResizeKeepsThe26thHit is the reported scenario: the operator
// selects a result deep in the list (the 26th) and enlarges the terminal. A
// search result is not paged, so the resize must keep the list as it is and put
// the cursor back on that contact — not recompute it as pages of the ordinary
// log, and not drop the selection to the first hit.
func TestEditorSearchResizeKeepsThe26thHit(t *testing.T) {
	le := newEditorWithDB(t)
	for i := 0; i < 40; i++ {
		insertQSO(t, le, &qso.QSO{
			Call:    fmt.Sprintf("SP9QQ%02d", i),
			QSODate: fmt.Sprintf("202604%02d", 1+i%28),
			TimeOn:  "100000",
			Band:    "40m",
			Mode:    "CW",
		})
	}

	le.searchQuery = "SP9QQ"
	if err := le.applySearchFilter(); err != nil {
		t.Fatalf("applySearchFilter: %v", err)
	}
	if len(le.qsos) != 40 {
		t.Fatalf("search returned %d rows; want 40", len(le.qsos))
	}

	le.table.SetCursor(25) // the 26th result
	want := selectedID(t, le)
	before := make([]int64, len(le.qsos))
	for i := range le.qsos {
		before[i] = le.qsos[i].ID
	}
	pageBefore := le.currentPage

	le.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if got := selectedID(t, le); got != want {
		t.Errorf("enlarging moved the selection from the 26th result (id %d) to id %d", want, got)
	}
	if len(le.qsos) != len(before) {
		t.Fatalf("the search list changed size across the resize: %d rows, want %d", len(le.qsos), len(before))
	}
	for i := range before {
		if le.qsos[i].ID != before[i] {
			t.Errorf("the search list was recomputed: row %d is id %d, want %d", i, le.qsos[i].ID, before[i])
			break
		}
	}
	if le.currentPage != pageBefore {
		t.Errorf("the resize re-paged the search result: page %d, want %d", le.currentPage, pageBefore)
	}
}
