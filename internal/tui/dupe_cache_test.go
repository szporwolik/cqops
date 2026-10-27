package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// The DUPE verdict compares the form against an existing contact: the call,
// band, mode and date have to match, and the reference fields (SOTA, POTA, WWFF,
// IOTA) have to match too — a second activation of the same summit is not a
// dupe, the same summit again is. The answer therefore depends on five form
// values and on what the logbook currently holds.

// dupeTestModel is a QSO form with only the fields the dupe check reads filled
// in, so a verdict cannot be influenced by anything else.
func dupeTestModel(t *testing.T) *Model {
	t.Helper()
	m := newLifecycleTestModel(t)
	m.dateTimeAuto = false
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("CW")
	m.fields[fieldDate].SetValue("2026-10-27")
	return m
}

// logContact writes a contact into the logbook under test.
func logContact(t *testing.T, m *Model, refs map[field]string) qso.QSO {
	t.Helper()
	q := &qso.QSO{
		Call: "SP9MOA", Band: "20m", Mode: "CW", QSODate: "20261027", TimeOn: "120000",
		RSTSent: "599", RSTRcvd: "599",
	}
	if v, ok := refs[fieldSOTA]; ok {
		q.SOTARef = v
	}
	if v, ok := refs[fieldPOTA]; ok {
		q.POTARef = v
	}
	if v, ok := refs[fieldWWFF]; ok {
		q.WWFFRef = v
	}
	if v, ok := refs[fieldIOTA]; ok {
		q.IOTA = v
	}
	if _, err := store.InsertQSO(m.App.DB, q); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	return *q
}

// TestDupeFollowsReferenceChanges: changing a reference field must change the
// verdict in both directions. The cached result was keyed on
// (call, band, mode, date) only, so the first answer stuck — the operator kept
// seeing "DUPE!" for a different summit until another field changed.
func TestDupeFollowsReferenceChanges(t *testing.T) {
	for _, tc := range []struct {
		name            string
		field           field
		stored, changed string
	}{
		{"SOTA", fieldSOTA, "SP/BZ-001", "SP/BZ-002"},
		{"POTA", fieldPOTA, "SP-0001", "SP-0002"},
		{"WWFF", fieldWWFF, "SPFF-0001", "SPFF-0002"},
		{"IOTA", fieldIOTA, "EU-002", "EU-005"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := dupeTestModel(t)
			logContact(t, m, map[field]string{tc.field: tc.stored})

			// Same reference as the logged contact: this is the same activation.
			m.fields[tc.field].SetValue(tc.stored)
			m.checkDupe()
			if !m.dupe {
				t.Fatalf("%s %q: not reported as a dupe", tc.name, tc.stored)
			}

			// A different reference is a different activation, not a dupe.
			m.fields[tc.field].SetValue(tc.changed)
			m.checkDupe()
			if m.dupe {
				t.Errorf("%s: still a dupe after changing %q to %q", tc.name, tc.stored, tc.changed)
			}

			// Changing it back makes it the same activation again.
			m.fields[tc.field].SetValue(tc.stored)
			m.checkDupe()
			if !m.dupe {
				t.Errorf("%s: not a dupe again after restoring %q", tc.name, tc.stored)
			}
		})
	}
}

// TestDupeInBothDirectionsWithoutReference: a contact without any reference is a
// dupe of the same contact without a reference, and stops being one as soon as a
// reference is entered.
func TestDupeInBothDirectionsWithoutReference(t *testing.T) {
	m := dupeTestModel(t)
	logContact(t, m, nil)

	m.checkDupe()
	if !m.dupe {
		t.Fatal("the identical contact was not reported as a dupe")
	}
	m.fields[fieldSOTA].SetValue("SP/BZ-001")
	m.checkDupe()
	if m.dupe {
		t.Error("a logged contact without a summit is still a dupe after entering one")
	}
	m.fields[fieldSOTA].SetValue("")
	m.checkDupe()
	if !m.dupe {
		t.Error("clearing the summit did not make it the same contact again")
	}
}

// TestDupeCacheInvalidatesWhenTheLogChanges: the verdict is a query, so anything
// that changes the logbook — a save, a WSJT-X contact, an import, a download, a
// delete, a logbook switch — has to discard it. refreshQSOS is the update every
// one of those paths runs.
func TestDupeCacheInvalidatesWhenTheLogChanges(t *testing.T) {
	m := dupeTestModel(t)
	m.checkDupe()
	if m.dupe {
		t.Fatal("precondition: an empty logbook already reports a dupe")
	}
	m.checkDupe() // second call, served from the cache
	if m.dupe {
		t.Fatal("precondition: the cached answer is not 'no dupe'")
	}

	// The same contact arrives while the form is open.
	logContact(t, m, nil)
	m.refreshQSOS()
	m.checkDupe()
	if !m.dupe {
		t.Fatal("the new contact was not picked up — the cached answer survived a data change")
	}

	// ...and removing it clears the warning again.
	qsos, err := store.ListQSOs(m.App.DB, 10, "")
	if err != nil || len(qsos) == 0 {
		t.Fatalf("ListQSOs: %v (%d rows)", err, len(qsos))
	}
	if err := store.DeleteQSO(m.App.DB, qsos[0].ID); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}
	m.refreshQSOS()
	m.checkDupe()
	if m.dupe {
		t.Error("the warning survived the contact being deleted")
	}
}

// TestDupeBadgeKeepsItsPlace: fixing the verdict must not move the warning. With
// a call and a grid the badge line always exists (it carries the path), so the
// warning can be checked in place: same row, same frame, appended to the path.
func TestDupeBadgeKeepsItsPlace(t *testing.T) {
	m := dupeTestModel(t)
	m.fields[fieldGrid].SetValue("JO90AA")
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if got := m.commitCall(); got == "" {
		t.Fatal("setup: the callsign was rejected")
	}
	m.rc.pathSig = "" // render the badge line from scratch for each state
	m.dupe = false

	without := captureLayout(t, m)
	if without.Panels["profile"] < 0 {
		t.Fatalf("no badge line is drawn: %s", without)
	}
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	noBadge := lines[without.Panels["profile"]]
	if !strings.Contains(noBadge, "Path") {
		t.Fatalf("precondition: the badge line has no path: %q", noBadge)
	}
	if strings.Contains(noBadge, "DUPE!") {
		t.Fatalf("precondition: a dupe badge without a dupe: %q", noBadge)
	}

	m.dupe = true
	m.rc.pathSig = ""
	with := captureLayout(t, m)
	withBadge := strings.Split(stripANSI(m.View().Content), "\n")[with.Panels["profile"]]
	if !strings.Contains(withBadge, "DUPE!") {
		t.Fatalf("the dupe flag did not reach the frame: %q", withBadge)
	}
	if with.Panels["profile"] != without.Panels["profile"] {
		t.Errorf("the warning moved from line %d to %d", without.Panels["profile"], with.Panels["profile"])
	}
	if with.Rows["form"] != without.Rows["form"] || with.Lines != without.Lines {
		t.Errorf("the warning changed the layout: form %d -> %d rows, frame %d -> %d lines",
			without.Rows["form"], with.Rows["form"], without.Lines, with.Lines)
	}
	// Same line, same path in front of it — the badge is only appended.
	if !strings.HasPrefix(withBadge, strings.SplitN(noBadge, "  ", 2)[0]) {
		t.Errorf("the warning replaced the path instead of following it: %q", withBadge)
	}
}
