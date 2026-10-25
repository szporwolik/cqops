package tui

import (
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

	enriched := m.fillCallbookData(callbookResultMsg{
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
	if m.fillCallbookData(callbookResultMsg{Call: saved.Call, Data: &callbook.Result{Callsign: saved.Call, Name: "x"}}) {
		t.Error("second late result was applied again")
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
	if m.fillCallbookData(callbookResultMsg{
		Call: "SP9ZZZ",
		Data: &callbook.Result{Callsign: "SP9ZZZ", Name: "Someone else", Grid: "JO90aa"},
	}) {
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
