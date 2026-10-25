package tui

import (
	"testing"

	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/store"
)

// A contact is uploaded before the callbook answers, so a late enrichment used
// to stop at the local database: the remote copy stayed incomplete while the
// sync machinery considered the row up to date (dirty=false, revision=0).
func TestLateEnrichmentMarksUploadedContactPendingSync(t *testing.T) {
	m := newLifecycleTestModel(t)

	fillMinimalValidQSO(m)
	m.fields[fieldFreq].SetValue("7.155")
	session := m.lookup.formGen
	m.saveQSO()
	saved := latestQSO(t, m)

	// Pretend the initial upload already attached the remote id.
	if err := store.SetWavelogID(m.App.DB, saved.ID, 77); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}

	m.clearForm()
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
		t.Fatal("late result did not reach the saved contact")
	}

	stored, err := store.GetQSOByID(m.App.DB, saved.ID)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogID != 77 {
		t.Fatalf("remote id = %d, want 77", stored.WavelogID)
	}
	if !stored.WavelogDirty {
		t.Error("enriched uploaded contact is not marked pending sync — the remote copy would stay incomplete")
	}
	if stored.WavelogDirtyRev == 0 {
		t.Error("enrichment did not bump the pending-sync revision")
	}

	// A second, identical answer must not manufacture another pending edit.
	if again, _ := m.fillCallbookData(callbookResultMsg{
		Call:    saved.Call,
		Session: session,
		Data:    &callbook.Result{Callsign: saved.Call, Name: "Late operator", Grid: "JN46kc", QTH: "S.Bernardino"},
	}); again {
		t.Error("the same enrichment was applied twice")
	}
}

// The row state is what the sync loop reads, so a no-op enrichment (row
// already complete) must leave the revision and the dirty flag untouched.
func TestEnrichmentOfCompleteRowLeavesSyncStateAlone(t *testing.T) {
	m := newLifecycleTestModel(t)
	id := minimalQSOForEnrich(t, m)
	if err := store.SetWavelogID(m.App.DB, id, 42); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}

	res, err := store.EnrichQSO(m.App.DB, id, store.EnrichmentData{Name: "Second pass"})
	if err != nil {
		t.Fatalf("EnrichQSO: %v", err)
	}
	if res.Changed {
		t.Error("a row that already had a name reported a change")
	}
	if res.Synced {
		t.Error("a no-op enrichment must not owe a PATCH")
	}

	stored, err := store.GetQSOByID(m.App.DB, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.WavelogDirty || stored.WavelogDirtyRev != 0 {
		t.Errorf("sync state touched: dirty=%v rev=%d", stored.WavelogDirty, stored.WavelogDirtyRev)
	}
	if stored.Name != "First" {
		t.Errorf("name overwritten: %q", stored.Name)
	}
}

// With Wavelog configured the enrichment must actually schedule the PATCH
// through the per-contact coordinator (one PATCH per contact, newest revision).
func TestLateEnrichmentSchedulesPatch(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = true
	m.App.Logbook.Wavelog.URL = "https://wavelog.example"
	m.App.Logbook.Wavelog.APIKey = "wl2_test"

	fillMinimalValidQSO(m)
	m.fields[fieldFreq].SetValue("7.155")
	session := m.lookup.formGen
	m.saveQSO()
	saved := latestQSO(t, m)
	if err := store.SetWavelogID(m.App.DB, saved.ID, 77); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}
	m.clearForm()

	enriched, syncCmd := m.fillCallbookData(callbookResultMsg{
		Call:    saved.Call,
		Session: session,
		Data:    &callbook.Result{Callsign: saved.Call, Name: "Late operator", Grid: "JN46kc"},
	})
	if !enriched {
		t.Fatal("late result did not reach the saved contact")
	}
	if syncCmd == nil {
		t.Fatal("no PATCH scheduled for an enriched, already-uploaded contact")
	}
	key := contactSyncKey{logbook: m.App.LogbookName, localID: saved.ID, url: "https://wavelog.example"}
	ctx := m.sync.inFlight[key]
	if ctx == nil {
		t.Fatal("contact not registered with the sync coordinator — a parallel PATCH would be possible")
	}
	// Release the chain lease this test will never drain.
	delete(m.sync.inFlight, key)
	ctx.release()
}

// minimalQSOForEnrich inserts a contact that already carries a callbook name.
func minimalQSOForEnrich(t *testing.T, m *Model) int64 {
	t.Helper()
	m.fields[fieldCall].SetValue("DJ7NT")
	m.fields[fieldName].SetValue("First")
	m.fields[fieldDate].SetValue("20260614")
	m.fields[fieldTime].SetValue("120000")
	m.fields[fieldBand].SetValue("40m")
	m.fields[fieldFreq].SetValue("7.155")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldRSTSent].SetValue("59")
	m.fields[fieldRSTRcvd].SetValue("59")
	m.saveQSO()
	return latestQSO(t, m).ID
}
