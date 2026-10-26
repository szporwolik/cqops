package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/store"
)

// A callbook answer that arrives after the operator typed the next callsign
// used to be discarded, leaving the contact that was just logged with an
// empty name/grid/QTH (seen in the field: QRZ needed 60-80 s and the row in
// Recent QSOs stayed half-filled forever). The late result now lands in the
// stored row instead.
func TestLateCallbookResultEnrichesSavedQSO(t *testing.T) {
	m := newLifecycleTestModel(t)

	fillMinimalValidQSO(m)
	m.fields[fieldFreq].SetValue("7.1968")
	m.saveQSO()

	saved := latestQSO(t, m)
	if m.lookup.savedID != saved.ID || m.lookup.savedCall != saved.Call {
		t.Fatalf("saved contact not remembered: id=%d call=%q (want %d/%q)",
			m.lookup.savedID, m.lookup.savedCall, saved.ID, saved.Call)
	}

	// The operator has already moved on to the next callsign.
	m.fields[fieldCall].SetValue("DL2XYZ")

	enriched, syncCmd := m.fillCallbookData(callbookResultMsg{
		Call: saved.Call,
		Data: &callbook.Result{
			Callsign: saved.Call,
			Name:     "Associazione Radioamatori Ticinesi",
			Grid:     "JN46kc",
			QTH:      "S.Bernardino",
			Country:  "Switzerland",
			DXCC:     "287",
		},
	})
	if !enriched {
		t.Fatal("late result was not applied to the saved contact")
	}
	// The contact was never uploaded in this fixture, so there is nothing to
	// PATCH — the row only carries a bumped revision.
	if syncCmd != nil {
		t.Error("a contact without a remote id must not schedule a PATCH")
	}

	stored, err := store.GetQSOByID(m.App.DB, saved.ID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Name != "Associazione Radioamatori Ticinesi" {
		t.Errorf("Name = %q", stored.Name)
	}
	if stored.GridSquare != "JN46kc" {
		t.Errorf("GridSquare = %q", stored.GridSquare)
	}
	if stored.QTH != "S.Bernardino" {
		t.Errorf("QTH = %q", stored.QTH)
	}
	if stored.DXCC != "287" {
		t.Errorf("DXCC = %q", stored.DXCC)
	}
	// The form belongs to the new call — it must not have been touched.
	if got := m.fields[fieldName].Value(); got != "" {
		t.Errorf("form Name = %q, want empty (next call)", got)
	}
	// One shot per saved contact.
	if m.lookup.savedID != 0 {
		t.Errorf("saved id not cleared: %d", m.lookup.savedID)
	}
	if again, _ := m.fillCallbookData(callbookResultMsg{Call: saved.Call, Data: &callbook.Result{Callsign: saved.Call, Name: "x"}}); again {
		t.Error("second late result was applied again")
	}
}

// The reported case: the QSO is saved, the form is cleared and left empty,
// and the answer arrives afterwards. The empty form used to swallow it —
// "saved row Name=\"\", blank form Name=Late operator" — so the row stayed
// half-filled while the form showed a contact that no longer exists.
func TestLateCallbookResultFillsSavedRowNotBlankForm(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldFreq].SetValue("7.155")

	session := m.lookup.formGen
	m.saveQSO()
	saved := latestQSO(t, m)

	// What the controller does after saveQSO: the form starts over.
	m.clearForm()
	if got := strings.TrimSpace(m.fields[fieldCall].Value()); got != "" {
		t.Fatalf("form call = %q, want empty after save", got)
	}
	if m.lookup.formGen == session {
		t.Fatal("form session did not advance on save")
	}

	enriched, _ := m.fillCallbookData(callbookResultMsg{
		Call:    saved.Call,
		Session: session, // dispatched before the save
		Data: &callbook.Result{
			Callsign: saved.Call,
			Name:     "Late operator",
			Grid:     "JN46kc",
			QTH:      "S.Bernardino",
		},
	})
	if !enriched {
		t.Fatalf("late result did not reach the saved contact (savedID=%d savedCall=%q formGen=%d session=%d)",
			m.lookup.savedID, m.lookup.savedCall, m.lookup.formGen, session)
	}

	stored, err := store.GetQSOByID(m.App.DB, saved.ID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Name != "Late operator" {
		t.Errorf("saved row Name = %q, want \"Late operator\"", stored.Name)
	}
	if stored.GridSquare != "JN46kc" {
		t.Errorf("saved row Grid = %q", stored.GridSquare)
	}
	// The blank form stays blank — the data belongs to the logged contact.
	if got := m.fields[fieldName].Value(); got != "" {
		t.Errorf("form Name = %q, want empty", got)
	}
	if got := m.fields[fieldGrid].Value(); got != "" {
		t.Errorf("form Grid = %q, want empty", got)
	}
}

// An answer that still belongs to the form it was asked for keeps filling the
// form — the saved-contact path must not steal normal lookups.
func TestCallbookAnswerFillsTheFormItWasAskedFor(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.fields[fieldCall].SetValue("DL7XYZ")

	session := m.lookup.formGen
	filled, _ := m.fillCallbookData(callbookResultMsg{
		Call:    "DL7XYZ",
		Session: session,
		Data:    &callbook.Result{Callsign: "DL7XYZ", Name: "Marty", Grid: "JO31aa", QTH: "Essen"},
	})
	if filled {
		t.Error("a form-owned answer must not be reported as a saved-contact enrichment")
	}
	if got := m.fields[fieldName].Value(); got != "Marty" {
		t.Errorf("form Name = %q, want Marty", got)
	}
	if got := m.fields[fieldGrid].Value(); !strings.EqualFold(got, "JO31aa") {
		t.Errorf("form Grid = %q, want JO31aa", got)
	}
}

// A result for a different callsign than the one just logged must still be
// dropped — the enrichment is only for the contact being finished.
func TestLateCallbookResultIgnoresOtherCalls(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldFreq].SetValue("14.25")
	m.saveQSO()
	saved := latestQSO(t, m)

	m.fields[fieldCall].SetValue("DL2XYZ")
	if applied, _ := m.fillCallbookData(callbookResultMsg{
		Call: "SP9ZZZ",
		Data: &callbook.Result{Callsign: "SP9ZZZ", Name: "Someone else", Grid: "JO90aa"},
	}); applied {
		t.Fatal("result for an unrelated call was applied")
	}
	stored, err := store.GetQSOByID(m.App.DB, saved.ID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Name != "" || stored.GridSquare != "" {
		t.Errorf("saved row was modified: name=%q grid=%q", stored.Name, stored.GridSquare)
	}
}
