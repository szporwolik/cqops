package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

func newTestRigChooser(t *testing.T) *RigChooser {
	t.Helper()
	cfg := &config.Config{
		Logbooks: map[string]config.Logbook{
			"test": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}},
		},
		Rigs: map[string]config.RigPreset{},
	}
	lb := cfg.Logbooks["test"]
	a := &app.App{Config: cfg, Logbook: &lb, LogbookName: "test"}
	rc := NewRigChooser(a, NewToastQueue())
	rc.width = 100
	rc.height = 30
	return rc
}

func newTestOperatorChooser(t *testing.T) *OperatorChooser {
	t.Helper()
	cfg := &config.Config{
		Logbooks: map[string]config.Logbook{
			"test": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}},
		},
		Operators: map[string]config.Operator{},
	}
	lb := cfg.Logbooks["test"]
	a := &app.App{Config: cfg, Logbook: &lb, LogbookName: "test"}
	oc := NewOperatorChooser(a, NewToastQueue())
	oc.width = 100
	oc.height = 30
	return oc
}

func TestSaveBackButtonLine(t *testing.T) {
	b := saveBackButton{}
	line := b.line("Save & Back", 60)
	if !strings.Contains(line, "[ Save & Back ]") || !strings.Contains(line, "(Space)") {
		t.Errorf("unfocused button line wrong: %q", line)
	}
	b.Focus = true
	line = b.line("Save & Back", 60)
	if !strings.Contains(line, ">") {
		t.Errorf("focused button line should show a cursor: %q", line)
	}
}

func TestGeneralMenuSaveBackButton(t *testing.T) {
	gm := NewGeneralMenu(config.DefaultConfig())
	gm.width = 100
	gm.height = 30

	// Down from the last item reaches the button.
	gm.cursor = 9
	gm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !gm.saveBtn.Focus {
		t.Fatal("down from last item should focus the Save & Back button")
	}

	// Space on the button saves.
	gm.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !gm.saved || !gm.done {
		t.Error("Space on the button should save")
	}

	// Down from the button returns to the first item.
	gm2 := NewGeneralMenu(config.DefaultConfig())
	gm2.saveBtn.Focus = true
	gm2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if gm2.saveBtn.Focus || gm2.cursor != 0 {
		t.Errorf("down from button: focus=%v cursor=%d, want focus=false cursor=0", gm2.saveBtn.Focus, gm2.cursor)
	}

	// Up from the first item focuses the button.
	gm3 := NewGeneralMenu(config.DefaultConfig())
	gm3.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !gm3.saveBtn.Focus {
		t.Error("up from first item should focus the Save & Back button")
	}
}

func TestNotificationsMenuSaveBackButton(t *testing.T) {
	nm := NewNotificationsMenu(config.DefaultConfig())
	nm.width = 100
	nm.height = 30

	nm.cursor = notifItemCount - 1
	nm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !nm.saveBtn.Focus {
		t.Fatal("down from last item should focus the Save & Back button")
	}
	nm.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !nm.saved || !nm.done {
		t.Error("Space on the button should save")
	}
}

func TestLogbookChooserSaveBackButton(t *testing.T) {
	a := newChooserTestApp(t)
	c := NewLogbookChooser(a, NewToastQueue())
	c.mode = chooserEdit

	// Tab from the last field focuses the button.
	c.station.wlCbFocus = true // Wavelog disabled: checkbox is the last field
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !c.saveBtn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}

	// Shift+Tab from the button returns to the form's last field.
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if c.saveBtn.Focus || !c.station.lastFieldFocused() {
		t.Error("shift+tab from button should return to the last field")
	}

	// Up from the first field focuses the button.
	c.station.Name.Focus()
	c.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !c.saveBtn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
}

func TestLogbookChooserSaveBackButtonActivates(t *testing.T) {
	a := newChooserTestApp(t)
	c := NewLogbookChooser(a, NewToastQueue())
	c.startEdit("home")

	c.station.wlCbFocus = true
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !c.saveBtn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	c.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if c.mode != chooserList {
		t.Errorf("mode = %v, want list after saving", c.mode)
	}
}

func TestRigChooserSaveBackButton(t *testing.T) {
	rc := newTestRigChooser(t)
	rc.startCreate()
	rc.form.FocusLast()

	rc.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !rc.saveBtn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	rc.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if rc.saveBtn.Focus || !rc.form.OnLastField() {
		t.Error("shift+tab from button should return to the last field")
	}
	rc.form.FocusFirst()
	rc.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !rc.saveBtn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
}

func TestOperatorChooserSaveBackButton(t *testing.T) {
	oc := newTestOperatorChooser(t)
	oc.startCreate()
	oc.form.FocusLast()

	oc.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !oc.saveBtn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	oc.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if oc.saveBtn.Focus || !oc.form.OnLastField() {
		t.Error("shift+tab from button should return to the last field")
	}
	oc.form.FocusFirst()
	oc.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !oc.saveBtn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
}

func TestLogbookEditorSaveBackButton(t *testing.T) {
	le := NewLogbookEditor(LogbookEditorConfig{DB: nil, StationOperator: "OP", StationGrid: "JO90", StationCall: "SP9MOA"})
	le.mode = edModeEdit

	le.focus = qefContestID
	le.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !le.saveBtn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	le.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if le.saveBtn.Focus || le.focus != qefContestID {
		t.Error("shift+tab from button should return to the last field")
	}
	le.focus = qefCall
	le.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !le.saveBtn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
}

func TestIntegrationMenuSaveBackButton(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logbooks = map[string]config.Logbook{"test": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}}}
	im := NewIntegrationMenu(cfg)
	im.width = 100
	im.height = 30

	// Tab from the last visible item focuses the button.
	im.focus = im.lastVisiblePos()
	im.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !im.saveBtn.Focus {
		t.Fatal("tab from last item should focus the Save & Back button")
	}
	// Space on the button saves.
	im.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !im.saved || !im.done {
		t.Error("Space on the button should save")
	}
}

func TestCallbookMenuSaveBackButton(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logbooks = map[string]config.Logbook{"test": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}}}
	cm := NewCallbookMenu(cfg)
	cm.width = 100
	cm.height = 30

	// Up from the first item focuses the button.
	cm.focus = cm.firstVisiblePos()
	cm.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !cm.saveBtn.Focus {
		t.Fatal("up from first item should focus the Save & Back button")
	}
	// Space on the button saves.
	cm.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !cm.saved || !cm.done {
		t.Error("Space on the button should save")
	}
}
