package store

import (
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// TestQSORankMatchesThePagedOrder: the rank is the contact's position in the
// same order the paged list is cut from, tie-break included — the logbook
// editor uses it to find a selected contact again after a resize, and a rank
// that disagreed with the page it is loaded from would select its neighbour.
func TestQSORankMatchesThePagedOrder(t *testing.T) {
	db := pagingTestDB(t)
	var ids []int64
	ids = append(ids, seedPagingContacts(t, db, "20260101", "120000", 3)...) // tie: same minute
	ids = append(ids, seedPagingContacts(t, db, "20260102", "120000", 2)...) // newer date
	ids = append(ids, seedPagingContacts(t, db, "20251231", "235900", 2)...) // older date
	ids = append(ids, seedPagingContacts(t, db, "20260101", "130000", 1)...) // newer time, older date

	// The list the editor pages through, in its own order.
	const all = 100
	page, total, err := ListQSOsPageWithCount(db, all, 0, "")
	if err != nil {
		t.Fatalf("ListQSOsPageWithCount: %v", err)
	}
	if total != len(ids) || len(page) != len(ids) {
		t.Fatalf("paged list holds %d of %d contacts", len(page), total)
	}

	for want, q := range page {
		rank, found, err := QSORank(db, q.ID, "")
		if err != nil {
			t.Fatalf("QSORank(%d): %v", q.ID, err)
		}
		if !found {
			t.Fatalf("QSORank(%d): contact exists but was reported missing", q.ID)
		}
		if rank != want {
			t.Errorf("rank of %s (id %d) = %d, want %d — the paged order is %v",
				q.Call, q.ID, rank, want, callsOf(page))
		}
	}
}

// TestQSORankOfAMissingContact: a contact that is gone has no position, which is
// what lets a resize fall back instead of jumping somewhere arbitrary.
func TestQSORankOfAMissingContact(t *testing.T) {
	db := pagingTestDB(t)
	ids := seedPagingContacts(t, db, "20260101", "120000", 2)
	if _, err := db.Exec(`DELETE FROM qsos WHERE id = ?`, ids[0]); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rank, found, err := QSORank(db, ids[0], "")
	if err != nil {
		t.Fatalf("QSORank: %v", err)
	}
	if found {
		t.Errorf("a deleted contact reported rank %d", rank)
	}
}

// TestQSORankHonoursTheContestScope: with a contest filter the position is
// inside that contest's merged list, not the whole logbook.
func TestQSORankHonoursTheContestScope(t *testing.T) {
	db := pagingTestDB(t)
	// Two contest contacts and three ordinary ones, interleaved by date.
	insert := func(call, date, contestID, adifID string) int64 {
		t.Helper()
		id, err := InsertQSO(db, &qso.QSO{
			Call: call, QSODate: date, TimeOn: "120000", Band: "20m", Mode: "CW",
			RSTSent: "599", RSTRcvd: "599", ContestID: contestID, ContestADIFID: adifID,
		})
		if err != nil {
			t.Fatalf("InsertQSO(%s): %v", call, err)
		}
		return id
	}
	insert("PLAIN1", "20260105", "", "")
	first := insert("CONT1", "20260104", "c1", "")
	insert("PLAIN2", "20260103", "", "")
	second := insert("BYADIF", "20260102", "", "c1") // matches by ADIF id only
	insert("PLAIN3", "20260101", "", "")

	if rank, found, err := QSORank(db, first, "c1"); err != nil || !found || rank != 0 {
		t.Errorf("newest contest contact: rank %d found %v err %v; want rank 0", rank, found, err)
	}
	if rank, found, err := QSORank(db, second, "c1"); err != nil || !found || rank != 1 {
		t.Errorf("ADIF-id contact: rank %d found %v err %v; want rank 1", rank, found, err)
	}
	// The same contact is not part of a different contest.
	if _, found, err := QSORank(db, first, "c2"); err != nil || found {
		t.Errorf("contact outside contest c2 reported as found (err %v)", err)
	}
}

func callsOf(qs []qso.QSO) []string {
	out := make([]string, 0, len(qs))
	for _, q := range qs {
		out = append(out, q.Call)
	}
	return out
}
