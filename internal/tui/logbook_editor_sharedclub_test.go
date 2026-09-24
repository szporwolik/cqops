package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Shared club station — read-only synced QSOs
// =============================================================================

// seedSyncedUnsynced inserts one synced (WavelogID > 0) and one unsynced QSO
// and loads the editor page. The synced row is first in list order.
func seedSyncedUnsynced(t *testing.T, le *LogbookEditor) (syncedID, unsyncedID int64) {
	t.Helper()
	syncedID = insertQSO(t, le, &qso.QSO{Call: "SP9XYZ", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB", WavelogID: 42})
	unsyncedID = insertQSO(t, le, &qso.QSO{Call: "SP9ABC", QSODate: "20240502", TimeOn: "120000",
		Band: "40m", Mode: "SSB"})
	le.loadPage()
	return syncedID, unsyncedID
}

func cursorRow(t *testing.T, le *LogbookEditor) qso.QSO {
	t.Helper()
	idx := le.table.Cursor()
	if idx < 0 || idx >= len(le.qsos) {
		t.Fatalf("cursor out of range: %d (len %d)", idx, len(le.qsos))
	}
	return le.qsos[idx]
}

// The page order follows the list query (newest first), so the unsynced QSO
// (later date) is at the top. Move the cursor to the synced row explicitly.
func selectRow(t *testing.T, le *LogbookEditor, call string) {
	t.Helper()
	for i, q := range le.qsos {
		if q.Call == call {
			le.table.SetCursor(i)
			return
		}
	}
	t.Fatalf("row %s not found", call)
}

func TestSharedClub_SyncedQSOEditBlocked(t *testing.T) {
	le := newEditorWithDB(t)
	le.sharedClub = true
	seedSyncedUnsynced(t, le)

	selectRow(t, le, "SP9XYZ") // synced
	upd, cmd := le.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	le = upd.(*LogbookEditor)
	if cmd == nil {
		t.Fatal("expected a warning command")
	}
	msg := cmd()
	em, ok := msg.(editorMsg)
	if !ok || !strings.Contains(em.toastWarn, "read-only") {
		t.Fatalf("toast = %q, want read-only warning", em.toastWarn)
	}
	if le.mode != edModeList {
		t.Fatalf("mode = %v, want edModeList (edit blocked)", le.mode)
	}

	// The unsynced row still opens for editing.
	selectRow(t, le, "SP9ABC")
	upd, cmd = le.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	le = upd.(*LogbookEditor)
	if le.mode != edModeEdit {
		t.Fatalf("mode = %v, want edModeEdit for unsynced row", le.mode)
	}
}

func TestSharedClub_SyncedQSODeleteBlocked(t *testing.T) {
	le := newEditorWithDB(t)
	le.sharedClub = true
	syncedID, unsyncedID := seedSyncedUnsynced(t, le)

	selectRow(t, le, "SP9XYZ") // synced
	upd, cmd := le.Update(keyDelete())
	le = upd.(*LogbookEditor)
	if cmd == nil {
		t.Fatal("expected a warning command")
	}
	em, ok := cmd().(editorMsg)
	if !ok || !strings.Contains(em.toastWarn, "deleting is disabled") {
		t.Fatalf("toast = %q, want delete-blocked warning", em.toastWarn)
	}
	if le.mode != edModeList {
		t.Fatalf("mode = %v, want edModeList (delete blocked)", le.mode)
	}
	if _, err := le.db.Query("SELECT 1"); err != nil {
		t.Fatalf("db error: %v", err)
	}
	if q, err := store.GetQSOByID(le.db, syncedID); err != nil || q.Call != "SP9XYZ" {
		t.Fatalf("synced QSO must survive: %v", err)
	}

	// The unsynced row still reaches the confirmation dialog.
	selectRow(t, le, "SP9ABC")
	upd, _ = le.Update(keyDelete())
	le = upd.(*LogbookEditor)
	if le.mode != edModeConfirmDelete {
		t.Fatalf("mode = %v, want edModeConfirmDelete for unsynced row", le.mode)
	}
	_ = unsyncedID
}

func TestSharedClub_PurgeBlocked(t *testing.T) {
	le := newEditorWithDB(t)
	le.sharedClub = true
	seedSyncedUnsynced(t, le)

	upd, cmd := le.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	le = upd.(*LogbookEditor)
	if cmd == nil {
		t.Fatal("expected a warning command")
	}
	em, ok := cmd().(editorMsg)
	if !ok || !strings.Contains(em.toastWarn, "purging") {
		t.Fatalf("toast = %q, want purge-blocked warning", em.toastWarn)
	}
	if le.mode != edModeList {
		t.Fatalf("mode = %v, want edModeList (purge blocked)", le.mode)
	}
}

func TestSharedClub_AltPSyncBlocked(t *testing.T) {
	le := newEditorWithDB(t)
	le.sharedClub = true
	seedSyncedUnsynced(t, le)

	upd, cmd := le.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModAlt})
	le = upd.(*LogbookEditor)
	if cmd == nil {
		t.Fatal("expected a warning command")
	}
	em, ok := cmd().(editorMsg)
	if !ok || !strings.Contains(em.toastWarn, "remote edits are disabled") {
		t.Fatalf("toast = %q, want sync-blocked warning", em.toastWarn)
	}
	if le.mode != edModeList {
		t.Fatalf("mode = %v, want edModeList (sync blocked)", le.mode)
	}
}

func TestSharedClub_DoSaveBlockedForSynced(t *testing.T) {
	le := newEditorWithDB(t)
	le.sharedClub = true
	syncedID, _ := seedSyncedUnsynced(t, le)

	// Simulate an edit session on the synced row (should never open via the
	// UI, but doSave must be the last line of defense).
	le.mode = edModeEdit
	le.editing = &qso.QSO{ID: syncedID, Call: "SP9XYZ", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB", WavelogID: 42}
	le.fillEditForm(le.editing)

	cmd := le.doSave()
	if cmd == nil {
		t.Fatal("doSave returned no command")
	}
	em, ok := cmd().(editorMsg)
	if !ok || !strings.Contains(em.toastWarn, "not saved") {
		t.Fatalf("toast = %q, want blocked-save warning", em.toastWarn)
	}
}

func TestSharedClub_UnsyncedEditAndSaveStillWork(t *testing.T) {
	le := newEditorWithDB(t)
	le.sharedClub = true
	_, unsyncedID := seedSyncedUnsynced(t, le)

	le.mode = edModeEdit
	le.editing = &qso.QSO{ID: unsyncedID, Call: "SP9ABC", QSODate: "20240502", TimeOn: "120000",
		Band: "40m", Mode: "SSB"}
	le.fillEditForm(le.editing)

	cmd := le.doSave()
	if cmd == nil {
		t.Fatal("doSave returned no command for unsynced row")
	}
	msg := cmd()
	if em, ok := msg.(editorMsg); !ok || em.saved != unsyncedID {
		t.Fatalf("expected saved editorMsg with id %d, got %#v", unsyncedID, msg)
	}
}
