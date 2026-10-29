package tui

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// Paging in the logbook editor is cut from the height (page size = table height
// minus the header), so the page number alone cannot survive a resize: page N of
// p-row pages starts at row (N-1)*p, which at a different p is an unrelated
// contact. The selected contact is the anchor instead.

// pagingTestEditor builds an editor over a logbook of `rows` contacts, newest
// first, with a 20-row page (height 29). all[i] is the contact at absolute row i.
func pagingTestEditor(t *testing.T, rows int) (*LogbookEditor, []qso.QSO) {
	t.Helper()
	m := newLifecycleTestModel(t)
	base := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	for i := 0; i < rows; i++ {
		if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
			Call:    fmt.Sprintf("T%03d", i),
			QSODate: base.AddDate(0, 0, -i).Format("20060102"),
			TimeOn:  "120000",
			Band:    "20m",
			Mode:    "CW",
			RSTSent: "599", RSTRcvd: "599",
		}); err != nil {
			t.Fatalf("InsertQSO(%d): %v", i, err)
		}
	}
	all, total, err := store.ListQSOsPageWithCount(m.App.DB, rows, 0, "")
	if err != nil || total != rows {
		t.Fatalf("ListQSOsPageWithCount: %v (total %d, want %d)", err, total, rows)
	}

	le := NewLogbookEditor(LogbookEditorConfig{DB: m.App.DB})
	le.width, le.height = 120, 29 // page size 20
	le.loadPage()
	if le.pageSize != 20 {
		t.Fatalf("page size %d at height 29, want 20", le.pageSize)
	}
	return le, all
}

// selectedID is the contact under the table cursor.
func selectedID(t *testing.T, le *LogbookEditor) int64 {
	t.Helper()
	cur := le.table.Cursor()
	if cur < 0 || cur >= len(le.qsos) {
		t.Fatalf("cursor %d is outside the %d loaded contacts", cur, len(le.qsos))
	}
	return le.qsos[cur].ID
}

// TestEditorResizeKeepsTheSelectedContact: enlarging the window must leave the
// cursor on the contact it was on. The page size doubles, the page number has to
// follow the contact rather than the other way round.
func TestEditorResizeKeepsTheSelectedContact(t *testing.T) {
	le, all := pagingTestEditor(t, 120)

	le.goToPage(2)        // absolute rows 20..39
	le.table.SetCursor(5) // absolute row 25
	want := all[25].ID
	if got := selectedID(t, le); got != want {
		t.Fatalf("setup: cursor is on contact %d, want %d", got, want)
	}

	le.Update(tea.WindowSizeMsg{Width: 120, Height: 49}) // page size 40
	if le.pageSize != 40 {
		t.Fatalf("page size %d after enlarging, want 40", le.pageSize)
	}
	if le.currentPage != 1 {
		t.Errorf("page %d after enlarging, want 1 (row 25 fits in the first 40)", le.currentPage)
	}
	if got := selectedID(t, le); got != want {
		t.Errorf("enlarging moved the selection to contact %d, want %d", got, want)
	}
}

// TestEditorResizeDoesNotSkipContacts is the reported sequence: page 2 of a
// 20-row page, then enlarge to 40 rows. The page that follows must start right
// after the last contact that was on screen, not after the old page boundary.
func TestEditorResizeDoesNotSkipContacts(t *testing.T) {
	le, all := pagingTestEditor(t, 120)

	le.goToPage(2)
	if len(le.qsos) != 20 {
		t.Fatalf("page 2 holds %d contacts, want 20", len(le.qsos))
	}
	last := le.qsos[len(le.qsos)-1].ID // absolute row 39
	if last != all[39].ID {
		t.Fatalf("setup: page 2 ends at contact %d, want %d", last, all[39].ID)
	}

	le.Update(tea.WindowSizeMsg{Width: 120, Height: 49}) // page size 40
	le.goToPage(le.currentPage + 1)

	if len(le.qsos) == 0 {
		t.Fatal("the page after the resize is empty")
	}
	if got, want := le.qsos[0].ID, all[40].ID; got != want {
		t.Errorf("the page after the resize starts at contact %d, want %d — contacts in between were skipped",
			got, want)
	}
}

// TestEditorShrinkKeepsTheSelectedContact: the other direction — a 40-row page
// becomes a 20-row page, the cursor has to stay on the same contact.
func TestEditorShrinkKeepsTheSelectedContact(t *testing.T) {
	le, all := pagingTestEditor(t, 120)
	le.Update(tea.WindowSizeMsg{Width: 120, Height: 49}) // page size 40
	if le.pageSize != 40 {
		t.Fatalf("page size %d, want 40", le.pageSize)
	}

	le.table.SetCursor(30) // absolute row 30
	want := all[30].ID
	le.Update(tea.WindowSizeMsg{Width: 120, Height: 29}) // back to 20
	if le.pageSize != 20 {
		t.Fatalf("page size %d after shrinking, want 20", le.pageSize)
	}
	if le.currentPage != 2 {
		t.Errorf("page %d after shrinking, want 2 (row 30 lives on page 2 of 20)", le.currentPage)
	}
	if got := selectedID(t, le); got != want {
		t.Errorf("shrinking moved the selection to contact %d, want %d", got, want)
	}
}

// TestEditorResizeRoundTripKeepsThePage: enlarging and shrinking back leaves the
// operator exactly where they were — same page, same contact, same rows.
func TestEditorResizeRoundTripKeepsThePage(t *testing.T) {
	le, _ := pagingTestEditor(t, 120)
	le.goToPage(3)
	le.table.SetCursor(7) // absolute row 47
	wantPage, wantCursor, wantID := le.currentPage, le.table.Cursor(), selectedID(t, le)

	le.Update(tea.WindowSizeMsg{Width: 140, Height: 49})
	le.Update(tea.WindowSizeMsg{Width: 120, Height: 29})

	if le.currentPage != wantPage || le.table.Cursor() != wantCursor {
		t.Errorf("round trip ended on page %d row %d, want page %d row %d",
			le.currentPage, le.table.Cursor(), wantPage, wantCursor)
	}
	if got := selectedID(t, le); got != wantID {
		t.Errorf("round trip ended on contact %d, want %d", got, wantID)
	}
}

// TestEditorWidthResizeKeepsThePageAndFollowsTheLayout: a width-only change must
// not reload (the page describes the same rows) and the columns still come from
// the current layout.
func TestEditorWidthResizeKeepsThePageAndFollowsTheLayout(t *testing.T) {
	m := newLifecycleTestModel(t)
	base := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
			Call: fmt.Sprintf("T%03d", i), QSODate: base.AddDate(0, 0, -i).Format("20060102"),
			TimeOn: "120000", Band: "20m", Mode: "CW", RSTSent: "599", RSTRcvd: "599",
		}); err != nil {
			t.Fatalf("InsertQSO: %v", err)
		}
	}
	le := NewLogbookEditor(LogbookEditorConfig{DB: m.App.DB})
	le.width, le.height = 120, 29
	le.loadPage()
	le.goToPage(2)
	le.table.SetCursor(3)
	wantID := selectedID(t, le)
	if hasColumn(le.formattedNames, "Contest") {
		t.Fatalf("setup: the 120-column layout already shows Contest: %v", le.formattedNames)
	}

	le.Update(tea.WindowSizeMsg{Width: 200, Height: 29}) // width only

	if le.currentPage != 2 || le.table.Cursor() != 3 {
		t.Errorf("a width change moved the table to page %d row %d, want page 2 row 3",
			le.currentPage, le.table.Cursor())
	}
	if got := selectedID(t, le); got != wantID {
		t.Errorf("a width change moved the selection to contact %d, want %d", got, wantID)
	}
	if !hasColumn(le.formattedNames, "Contest") {
		t.Errorf("the columns did not follow the new layout: %v", le.formattedNames)
	}
}
