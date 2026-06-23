package tui

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

func init() {
	applog.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newChooserTestApp creates a minimal App with two logbooks and a temp DB.
func newChooserTestApp(t *testing.T) *app.App {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		State:   config.StateConfig{ActiveLogbook: "home"},
		General: config.GeneralConfig{DistanceUnit: "km"},
		Logbooks: map[string]config.Logbook{
			"home": {
				Name: "Home QTH",
				Station: config.Station{
					Callsign: "SP9MOA",
					Grid:     "JO90",
				},
			},
			"portable": {
				Name: "Field operations",
				Station: config.Station{
					Callsign: "SP9MOA/P",
					Grid:     "JO80",
				},
			},
		},
	}

	return &app.App{
		Config:      cfg,
		ConfigPath:  filepath.Join(t.TempDir(), "config.yaml"),
		LogbookName: "home",
		Logbook:     nil,
		DB:          db,
	}
}

// sendKey sends a key event to the chooser and returns the updated chooser.
// In Bubble Tea v2, KeyPressMsg is a Key struct with Code (rune) and Text (string).
// String() returns Text if non-empty, otherwise the key name derived from Code.
func sendKey(c *LogbookChooser, msg tea.KeyPressMsg) *LogbookChooser {
	updated, _ := c.Update(msg)
	return updated.(*LogbookChooser)
}

// Helper constructors for common keys.
func keyEnter() tea.KeyPressMsg  { return tea.KeyPressMsg{Code: tea.KeyEnter} }
func keyEsc() tea.KeyPressMsg    { return tea.KeyPressMsg{Code: tea.KeyEscape} }
func keyRight() tea.KeyPressMsg  { return tea.KeyPressMsg{Code: tea.KeyRight} }
func keyDown() tea.KeyPressMsg   { return tea.KeyPressMsg{Code: tea.KeyDown} }
func keyDelete() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyDelete} }

func TestLogbookChooserDeleteConfirmation(t *testing.T) {
	a := newChooserTestApp(t)
	tq := NewToastQueue()
	c := NewLogbookChooser(a, tq)

	// Verify initial state: 2 logbooks, active = "home"
	if len(c.names) != 2 {
		t.Fatalf("expected 2 logbooks, got %d", len(c.names))
	}
	if c.names[c.cursor] != a.Config.State.ActiveLogbook {
		t.Fatalf("cursor should be on active logbook %q, got %q", a.Config.State.ActiveLogbook, c.names[c.cursor])
	}

	// Move cursor to "portable" logbook (press down arrow)
	c = sendKey(c, keyDown())

	// --- Test 1: Press Delete, navigate to Cancel, press Enter — should NOT delete ---

	// Press Delete to open the confirm dialog.
	c = sendKey(c, keyDelete())
	if c.mode != chooserConfirmDelete {
		t.Fatal("expected chooserConfirmDelete mode after Delete key")
	}
	if c.dialog == nil {
		t.Fatal("expected dialog to be created")
	}

	// Navigate to Cancel (right arrow moves from Delete[0] to Cancel[1]).
	c = sendKey(c, keyRight())
	if c.dialog == nil {
		t.Fatal("dialog should still exist after navigation")
	}

	// Press Enter to confirm Cancel.
	c = sendKey(c, keyEnter())

	// After dismissing Cancel, mode should return to list and dialog cleared.
	if c.mode != chooserList {
		t.Errorf("expected chooserList after cancelling delete, got %d", c.mode)
	}
	if c.dialog != nil {
		t.Error("dialog should be nil after dismissal")
	}

	// "portable" logbook should STILL exist.
	if len(c.names) != 2 {
		t.Errorf("expected 2 logbooks after cancel, got %d", len(c.names))
	}
	if _, ok := a.Config.Logbooks["portable"]; !ok {
		t.Error("portable logbook was deleted even though user cancelled!")
	}

	// --- Test 2: Press Delete, keep Delete selected, press Enter — SHOULD delete ---

	// Move cursor to "portable" again
	for i, n := range c.names {
		if n == "portable" {
			c.cursor = i
			break
		}
	}

	// Press Delete.
	c = sendKey(c, keyDelete())
	if c.mode != chooserConfirmDelete {
		t.Fatal("expected chooserConfirmDelete mode")
	}

	// Press Enter immediately (Delete is default selected).
	c = sendKey(c, keyEnter())

	// "portable" should be deleted.
	if len(c.names) != 1 {
		t.Errorf("expected 1 logbook after delete, got %d", len(c.names))
	}
	if _, ok := a.Config.Logbooks["portable"]; ok {
		t.Error("portable logbook should have been deleted")
	}
	if _, ok := a.Config.Logbooks["home"]; !ok {
		t.Error("home logbook should still exist")
	}
}

func TestLogbookChooserDeleteEscCancels(t *testing.T) {
	a := newChooserTestApp(t)
	tq := NewToastQueue()
	c := NewLogbookChooser(a, tq)

	// Move to "portable"
	for i, n := range c.names {
		if n == "portable" {
			c.cursor = i
			break
		}
	}

	// Press Delete to open dialog.
	c = sendKey(c, keyDelete())
	if c.mode != chooserConfirmDelete {
		t.Fatal("expected chooserConfirmDelete mode")
	}

	// Press Esc to cancel.
	c = sendKey(c, keyEsc())

	// Should return to list mode without deleting.
	if c.mode != chooserList {
		t.Errorf("expected chooserList after Esc, got %d", c.mode)
	}
	if len(c.names) != 2 {
		t.Errorf("expected 2 logbooks after Esc cancel, got %d", len(c.names))
	}
	if _, ok := a.Config.Logbooks["portable"]; !ok {
		t.Error("portable logbook was deleted on Esc!")
	}
}

func TestLogbookChooserCannotDeleteActive(t *testing.T) {
	a := newChooserTestApp(t)
	tq := NewToastQueue()
	c := NewLogbookChooser(a, tq)

	// Cursor should be on "home" (the active logbook).
	if c.names[c.cursor] != "home" {
		for i, n := range c.names {
			if n == "home" {
				c.cursor = i
				break
			}
		}
	}

	// Press Delete.
	c = sendKey(c, keyDelete())

	// Press Enter (Delete is default).
	c = sendKey(c, keyEnter())

	// Active logbook should NOT be deleted.
	if len(c.names) != 2 {
		t.Errorf("expected 2 logbooks (active should not be deletable), got %d", len(c.names))
	}
	if _, ok := a.Config.Logbooks["home"]; !ok {
		t.Error("active logbook was deleted — should be protected!")
	}
}
