package tui

import (
	"fmt"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// TestLogbookEditor_SearchCoversWholeLogbook verifies that the editor search
// finds QSOs beyond the currently displayed page.
func TestLogbookEditor_SearchCoversWholeLogbook(t *testing.T) {
	le := newEditorWithDB(t)

	// Insert 20 QSOs; page size is 15 (height 24 - 9), so the oldest five
	// fall outside the first page. The target is deliberately the oldest.
	for i := 1; i <= 19; i++ {
		insertQSO(t, le, &qso.QSO{
			Call:    fmt.Sprintf("S%02d", i),
			Name:    "Operator",
			Country: "Testland",
			QSODate: "20240501",
			TimeOn:  "120000",
			Band:    "20m",
			Mode:    "SSB",
		})
	}
	insertQSO(t, le, &qso.QSO{
		Call:    "ZZ9TARGET",
		Name:    "Rare Name",
		Country: "Testland",
		QSODate: "20240301", // oldest -> last page
		TimeOn:  "120000",
		Band:    "20m",
		Mode:    "SSB",
	})

	le.loadPage()
	if le.totalCount != 20 {
		t.Fatalf("totalCount = %d, want 20", le.totalCount)
	}
	if len(le.qsos) != 15 {
		t.Fatalf("first page has %d rows, want 15", len(le.qsos))
	}

	// Search by name for a QSO that is not on the first page.
	le.searchQuery = "Rare Name"
	le.applySearchFilter()
	if le.totalCount != 1 {
		t.Fatalf("search totalCount = %d, want 1", le.totalCount)
	}
	if len(le.qsos) != 1 || le.qsos[0].Call != "ZZ9TARGET" {
		t.Fatalf("search results = %+v, want ZZ9TARGET", le.qsos)
	}

	// Search by callsign fragment works across the whole logbook too.
	le.searchQuery = "zz9"
	le.applySearchFilter()
	if le.totalCount != 1 || le.qsos[0].Call != "ZZ9TARGET" {
		t.Fatalf("call search totalCount = %d, want 1 match", le.totalCount)
	}

	// Clearing the search restores full paged view.
	le.searchQuery = ""
	le.applySearchFilter()
	if le.totalCount != 20 || len(le.qsos) != 15 {
		t.Fatalf("after clear: totalCount = %d, rows = %d; want 20/15", le.totalCount, len(le.qsos))
	}
}

// TestLogbookEditor_SearchDebouncedAsync verifies the interactive search path:
// typing schedules a debounce, the debounce dispatches a worker, and the
// worker result applies to the list — the full-logbook scan never runs
// synchronously on a keystroke.
func TestLogbookEditor_SearchDebouncedAsync(t *testing.T) {
	le := newEditorWithDB(t)
	insertQSO(t, le, &qso.QSO{Call: "ZZ9TARGET", Name: "Rare", Country: "Testland",
		QSODate: "20240301", TimeOn: "120000", Band: "20m", Mode: "SSB"})
	le.loadPage()

	le.searchQuery = "zz9"
	cmd := le.scheduleSearch()
	if cmd == nil {
		t.Fatal("scheduleSearch returned no debounce command")
	}
	dbg, ok := execCmd(cmd).(searchDebounceMsg)
	if !ok {
		t.Fatalf("debounce command produced %T, want searchDebounceMsg", execCmd(cmd))
	}
	if dbg.gen != le.searchGen || dbg.query != "zz9" {
		t.Fatalf("debounce = gen:%d query:%q, want gen:%d query:%q", dbg.gen, dbg.query, le.searchGen, "zz9")
	}

	upd, workerCmd := le.Update(dbg)
	le = upd.(*LogbookEditor)
	if workerCmd == nil {
		t.Fatal("debounce must dispatch the search worker")
	}
	res, ok := execCmd(workerCmd).(editorMsg)
	if !ok || res.searchErr != "" {
		t.Fatalf("search worker result = %#v", res)
	}
	if len(res.searchRows) != 1 {
		t.Fatalf("search rows = %d, want 1", len(res.searchRows))
	}

	upd, _ = le.Update(res)
	le = upd.(*LogbookEditor)
	if le.totalCount != 1 || len(le.qsos) != 1 || le.qsos[0].Call != "ZZ9TARGET" {
		t.Fatalf("applied search = total %d, qsos %+v", le.totalCount, le.qsos)
	}
}

// TestLogbookEditor_SearchStaleGenerationDiscarded verifies that a debounce
// firing for a superseded query does not dispatch a worker.
func TestLogbookEditor_SearchStaleGenerationDiscarded(t *testing.T) {
	le := newEditorWithDB(t)
	le.searchQuery = "zz9"
	dbg1 := execCmd(le.scheduleSearch()).(searchDebounceMsg)

	le.searchQuery = "other"
	_ = le.scheduleSearch()

	upd, workerCmd := le.Update(dbg1)
	le = upd.(*LogbookEditor)
	if workerCmd != nil {
		t.Fatal("stale debounce must not dispatch a search worker")
	}
}

// TestLogbookEditor_SearchRespectsContestFilter verifies the search honors the
// active contest filter.
func TestLogbookEditor_SearchRespectsContestFilter(t *testing.T) {
	le := newEditorWithDB(t)

	insertQSO(t, le, &qso.QSO{Call: "A1A", Country: "Testland", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "c1hash"})
	insertQSO(t, le, &qso.QSO{Call: "B2B", Country: "Testland", QSODate: "20240502", TimeOn: "120000", Band: "20m", Mode: "SSB"})

	le.SetContestID("c1hash", "Test Contest", "TEST", "2024-05-01")

	le.searchQuery = "testland"
	le.applySearchFilter()
	if le.totalCount != 1 || le.qsos[0].Call != "A1A" {
		t.Fatalf("contest-scoped search got %d results, want A1A only", le.totalCount)
	}
}

// TestLogbookEditor_DeleteWhileSearchingClearsFilter reproduces the reported
// bug: deleting a QSO while a search filter is active left the deleted row
// visible (loadPage skips reloads during a search). The filter must clear so
// the list immediately shows the remaining rows.
func TestLogbookEditor_DeleteWhileSearchingClearsFilter(t *testing.T) {
	le := newEditorWithDB(t)
	target := insertQSO(t, le, &qso.QSO{Call: "ZZ9TARGET", Name: "Rare",
		QSODate: "20240301", TimeOn: "120000", Band: "20m", Mode: "SSB"})
	insertQSO(t, le, &qso.QSO{Call: "SP9OTHER", Name: "Other",
		QSODate: "20240302", TimeOn: "120000", Band: "20m", Mode: "SSB"})
	le.loadPage()

	// Run a search that matches only the target.
	le.searchQuery = "zz9"
	le.searchInput.SetValue("zz9")
	upd, workerCmd := le.Update(execCmd(le.scheduleSearch()).(searchDebounceMsg))
	le = upd.(*LogbookEditor)
	res := execCmd(workerCmd).(editorMsg)
	upd, _ = le.Update(res)
	le = upd.(*LogbookEditor)
	if len(le.qsos) != 1 || le.qsos[0].Call != "ZZ9TARGET" {
		t.Fatalf("search result = %+v, want only ZZ9TARGET", le.qsos)
	}

	// Delete completes while the filter is still active — the worker already
	// removed the row from the database before delivering the completion.
	if err := store.DeleteQSO(le.db, target); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}
	upd, _ = le.Update(editorMsg{deleted: target, delCall: "ZZ9TARGET", delDate: "2024-03-01", gen: le.gen, lbID: le.logbookID})
	le = upd.(*LogbookEditor)

	if le.searchQuery != "" {
		t.Errorf("search filter = %q, want cleared after deletion", le.searchQuery)
	}
	if len(le.qsos) != 1 || le.qsos[0].Call != "SP9OTHER" {
		t.Fatalf("list after delete = %+v, want only SP9OTHER (deleted row must disappear)", le.qsos)
	}
}
