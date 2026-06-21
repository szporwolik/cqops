package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Favorites — save / recall / persistence
// =============================================================================

func newFavoriteTestModel(t *testing.T) *Model {
	t.Helper()

	m := newLifecycleTestModel(t)

	// Give it a config path so config.Save works.
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	m.App.Config.Favorites = make(map[int]config.Favorite)

	// Fill form with known values.
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldFreq].SetValue("14250")
	m.fields[fieldSubmode].SetValue("")
	m.fields[fieldBand].SetValue("20m")

	return m
}

func TestFavoriteSave(t *testing.T) {
	m := newFavoriteTestModel(t)

	m.favoriteSave(3)

	fav, ok := m.App.Config.Favorites[3]
	if !ok {
		t.Fatal("favorite slot 3 not found after save")
	}
	if fav.Mode != "SSB" {
		t.Errorf("Mode = %q, want SSB", fav.Mode)
	}
	if fav.Freq != 14250 {
		t.Errorf("Freq = %f, want 14250", fav.Freq)
	}
	if fav.Band != "20m" {
		t.Errorf("Band = %q, want 20m", fav.Band)
	}
	if fav.Submode != "" {
		t.Errorf("Submode = %q, want empty", fav.Submode)
	}
}

func TestFavoriteRecall(t *testing.T) {
	m := newFavoriteTestModel(t)

	// Save to slot 5.
	m.favoriteSave(5)

	// Clear form fields.
	m.fields[fieldMode].SetValue("")
	m.fields[fieldFreq].SetValue("")
	m.fields[fieldBand].SetValue("")
	m.fields[fieldSubmode].SetValue("")

	// Recall.
	m.favoriteRecall(5)

	if got := m.fields[fieldMode].Value(); got != "SSB" {
		t.Errorf("Mode = %q, want SSB", got)
	}
	if got := m.fields[fieldFreq].Value(); got != "14250.000000" {
		t.Errorf("Freq = %q, want 14250.000000", got)
	}
	if got := m.fields[fieldBand].Value(); got != "20m" {
		t.Errorf("Band = %q, want 20m", got)
	}
}

func TestFavoriteRecallEmpty(t *testing.T) {
	m := newFavoriteTestModel(t)

	// Recall slot that was never saved — should warn, not crash.
	m.favoriteRecall(7)

	// Verify no values were injected.
	if got := m.fields[fieldMode].Value(); got != "SSB" {
		t.Errorf("Mode should be unchanged after empty recall, got %q", got)
	}
}

func TestFavoriteRecallDerivesBand(t *testing.T) {
	m := newFavoriteTestModel(t)

	// Save with freq but explicitly clear band (simulate user who only stores freq).
	m.fields[fieldBand].SetValue("")
	m.fields[fieldFreq].SetValue("7100")
	m.fields[fieldMode].SetValue("SSB")
	m.favoriteSave(1)

	// Recall and verify band was derived.
	m.fields[fieldBand].SetValue("")
	m.fields[fieldMode].SetValue("")
	m.fields[fieldFreq].SetValue("")
	m.favoriteRecall(1)

	expectedBand := qso.DeriveBand(7100)
	if got := m.fields[fieldBand].Value(); got != expectedBand {
		t.Errorf("Band = %q, want derived %q", got, expectedBand)
	}
}

func TestFavoriteSavePersists(t *testing.T) {
	m := newFavoriteTestModel(t)
	cfgPath := m.App.ConfigPath

	m.favoriteSave(9)

	// Re-read config from disk.
	cfg2, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	fav, ok := cfg2.Favorites[9]
	if !ok {
		t.Fatal("favorite slot 9 not persisted to disk")
	}
	if fav.Mode != "SSB" || fav.Freq != 14250 {
		t.Errorf("persisted favorite mismatch: %+v", fav)
	}
}

func TestFavoriteOverwrite(t *testing.T) {
	m := newFavoriteTestModel(t)

	m.favoriteSave(0)

	// Change form and save again to same slot.
	m.fields[fieldMode].SetValue("CW")
	m.fields[fieldFreq].SetValue("3550")
	m.fields[fieldBand].SetValue("80m")
	m.favoriteSave(0)

	fav := m.App.Config.Favorites[0]
	if fav.Mode != "CW" {
		t.Errorf("Mode = %q, want CW (overwrite)", fav.Mode)
	}
}

// =============================================================================
// handleFavoriteKey — key string parsing
// =============================================================================

func TestHandleFavoriteKey_Save(t *testing.T) {
	m := newFavoriteTestModel(t)

	// Test the standard form: alt+shift+digit
	for digit := 0; digit <= 9; digit++ {
		r := '0' + rune(digit)
		msg := tea.KeyPressMsg{Code: r, Mod: tea.ModAlt | tea.ModShift}
		_, handled := m.handleFavoriteKey(msg)
		if !handled {
			t.Errorf("alt+shift+%d should be handled (string=%q)", digit, msg.String())
		}
		fav, ok := m.App.Config.Favorites[digit]
		if !ok {
			t.Errorf("favorite slot %d not found after alt+shift+%d", digit, digit)
		}
		_ = fav
	}

	// Test the alternate form: alt+shifted_char (what most terminals send).
	// Shift+1 = '!' ... Shift+0 = ')'
	shifted := []rune{')', '!', '@', '#', '$', '%', '^', '&', '*', '('}
	for digit := 0; digit <= 9; digit++ {
		msg := tea.KeyPressMsg{Code: shifted[digit], Mod: tea.ModAlt}
		_, handled := m.handleFavoriteKey(msg)
		if !handled {
			t.Errorf("alt+%c should be handled as save for slot %d (string=%q)", shifted[digit], digit, msg.String())
		}
		// Verify saved correctly — note: overwrites previous save for same slot.
		fav, ok := m.App.Config.Favorites[digit]
		if !ok {
			t.Errorf("favorite slot %d not found after alt+%c", digit, shifted[digit])
		}
		_ = fav
	}
}

func TestHandleFavoriteKey_Recall(t *testing.T) {
	m := newFavoriteTestModel(t)

	// Pre-save to slot 4.
	m.favoriteSave(4)

	// Clear form.
	m.fields[fieldMode].SetValue("")
	m.fields[fieldFreq].SetValue("")
	m.fields[fieldBand].SetValue("")

	msg := tea.KeyPressMsg{Code: '4', Mod: tea.ModAlt}
	_, handled := m.handleFavoriteKey(msg)
	if !handled {
		t.Error("alt+4 should be handled")
	}
	if got := m.fields[fieldMode].Value(); got != "SSB" {
		t.Errorf("Mode = %q after recall, want SSB", got)
	}
}

func TestHandleFavoriteKey_NotHandled(t *testing.T) {
	m := newFavoriteTestModel(t)

	for _, k := range []tea.KeyPressMsg{
		{Code: 'a', Mod: tea.ModAlt},
		{Code: 's', Mod: tea.ModAlt},
		{Code: 'x', Mod: tea.ModAlt | tea.ModShift},
		{Code: 'a', Mod: tea.ModCtrl},
		{Code: 's', Mod: tea.ModCtrl},
		{Code: tea.KeyEnter},
		{Code: tea.KeyTab},
	} {
		_, handled := m.handleFavoriteKey(k)
		if handled {
			t.Errorf("%v should NOT be handled as favorite key", k)
		}
	}
}

// =============================================================================
// Path row — multi-badge rendering
// =============================================================================

func TestFormPathRowDupeAndNewCallSimultaneously(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.fields[fieldCall].SetValue("VK3A")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldGrid].SetValue("PG66pa")
	m.rc.pathCall = "VK3A"
	m.dupe = true
	m.rc.logStats = store.LogbookStats{CallWorked: false} // not worked → New Call!

	row := m.formPathRow(100)
	if !strings.Contains(row, "DUPE!") {
		t.Error("row should contain DUPE! when dupe is true")
	}
	if !strings.Contains(row, "New Call!") {
		t.Error("row should contain New Call! when call not worked")
	}
}

func TestFormPathRowDupeOnly(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldGrid].SetValue("JN18")
	m.rc.pathCall = "SP9MOA"
	m.dupe = true
	m.rc.logStats = store.LogbookStats{CallWorked: true} // already worked → no New Call!

	row := m.formPathRow(100)
	if !strings.Contains(row, "DUPE!") {
		t.Error("row should contain DUPE!")
	}
	if strings.Contains(row, "New Call!") {
		t.Error("row should NOT contain New Call! when call already worked")
	}
}

func TestFormPathRowNoDupeWithNewCall(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.fields[fieldCall].SetValue("VK3A")
	m.fields[fieldGrid].SetValue("PG66pa")
	m.rc.pathCall = "VK3A"
	m.dupe = false
	m.rc.logStats = store.LogbookStats{CallWorked: false}

	row := m.formPathRow(100)
	if strings.Contains(row, "DUPE!") {
		t.Error("row should NOT contain DUPE! when dupe is false")
	}
	if !strings.Contains(row, "New Call!") {
		t.Error("row should contain New Call!")
	}
}

func TestFormPathRowNoCallShowsProfile(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.rc.pathCall = ""
	m.fields[fieldCall].SetValue("")

	row := m.formPathRow(90)
	if !strings.Contains(row, "Op") {
		t.Error("row should show station profile when no callsign entered")
	}
	if strings.Contains(row, "DUPE!") {
		t.Error("row should NOT contain DUPE! when no callsign")
	}
}

func TestFormPathRowNoBadgesNoGrid(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.App.Logbook.Station.Grid = ""
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldGrid].SetValue("")
	m.rc.pathCall = "SP9MOA"
	m.dupe = false
	m.rc.logStats = store.LogbookStats{CallWorked: true}

	row := m.formPathRow(90)
	// No dupe, no New Call, no New DXCC, no grid → row should be empty.
	if row != "" {
		t.Errorf("row should be empty when no badges and no grid, got %q", row)
	}
}

// =============================================================================
// needRefresh — does NOT block subsequent key presses
// =============================================================================

func TestNeedRefreshDoesNotBlockKeys(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.screen = screenMainMenu
	m.ui.mainMenu = NewMainMenu()
	m.ui.mainMenu.width = 100
	m.ui.mainMenu.height = 30

	// Simulate the exact scenario: chooser done → needRefresh set.
	m.needRefresh = true

	// Send a Down arrow key — this should NOT be consumed by handlePendingRequests.
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// The key should have been processed by the main menu handler.
	// If needRefresh short-circuited, the cursor would be at 0.
	// After handleMainMenuUpdate processes "down", cursor should be at 1.
	if cmd != nil {
		// If a command was returned (e.g., refreshQSOS), that's fine —
		// it means needRefresh was processed BUT the key still went through.
		// consume it to avoid leaking goroutines in tests.
	}
	if m.ui.mainMenu.cursor != 1 {
		t.Errorf("menu cursor = %d, want 1 (Down arrow should move cursor)", m.ui.mainMenu.cursor)
	}
}
