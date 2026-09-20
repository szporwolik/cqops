package tui

import (
	"fmt"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
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
