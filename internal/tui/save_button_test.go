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
	gm.fm.row = generalRowNotifications
	gm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !gm.fm.btn.Focus {
		t.Fatal("down from last item should focus the Save & Back button")
	}
	if gm.fm.row != -1 {
		t.Errorf("button focused but list cursor still active: %d, want -1", gm.fm.row)
	}

	// Space on the button saves.
	gm.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !gm.saved || !gm.done {
		t.Error("Space on the button should save")
	}

	// Down from the button returns to the first item.
	gm2 := NewGeneralMenu(config.DefaultConfig())
	gm2.fm.btn.Focus = true
	gm2.fm.row = -1
	gm2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if gm2.fm.btn.Focus || gm2.fm.row != 0 {
		t.Errorf("down from button: focus=%v row=%d, want focus=false row=0", gm2.fm.btn.Focus, gm2.fm.row)
	}

	// Up from the first item focuses the button.
	gm3 := NewGeneralMenu(config.DefaultConfig())
	gm3.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !gm3.fm.btn.Focus {
		t.Error("up from first item should focus the Save & Back button")
	}
	if gm3.fm.row != -1 {
		t.Errorf("button focused but list cursor still active: %d, want -1", gm3.fm.row)
	}
}

func TestNotificationsMenuSaveBackButton(t *testing.T) {
	nm := NewNotificationsMenu(config.DefaultConfig())
	nm.width = 100
	nm.height = 30

	nm.fm.row = notifItemCount - 1
	nm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !nm.fm.btn.Focus {
		t.Fatal("down from last item should focus the Save & Back button")
	}
	if nm.fm.row != -1 {
		t.Errorf("button focused but list cursor still active: %d, want -1", nm.fm.row)
	}
	if n := strings.Count(nm.View().Content, "> "); n != 1 {
		t.Errorf("notifications menu shows %d active markers, want 1 (button only)", n)
	}
	nm.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !nm.saved || !nm.done {
		t.Error("Space on the button should save")
	}

	// Up from the first item reaches the button too, with a single marker.
	nm2 := NewNotificationsMenu(config.DefaultConfig())
	nm2.width = 100
	nm2.height = 30
	nm2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !nm2.fm.btn.Focus || nm2.fm.row != -1 {
		t.Fatalf("up from first item: saveBtn=%v row=%d, want button focused and row -1", nm2.fm.btn.Focus, nm2.fm.row)
	}
	if n := strings.Count(nm2.View().Content, "> "); n != 1 {
		t.Errorf("notifications menu shows %d active markers, want 1 (button only)", n)
	}
}

func TestLogbookChooserSaveBackButton(t *testing.T) {
	a := newChooserTestApp(t)
	c := NewLogbookChooser(a, NewToastQueue())
	c.mode = chooserEdit

	// Tab from the last field focuses the button.
	// APRS disabled: the APRS TX checkbox (row 21) is the last field.
	c.fm.row = 21
	c.station.focusRow(21)
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !c.fm.btn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}

	// Shift+Tab from the button returns to the form's last field.
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if c.fm.btn.Focus || !c.station.aprsCbFocus {
		t.Error("shift+tab from button should return to the last field")
	}

	// Up from the first field focuses the button.
	c.fm.row = 0
	c.station.focusRow(0)
	c.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !c.fm.btn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
}

func TestLogbookChooserSaveBackButtonActivates(t *testing.T) {
	a := newChooserTestApp(t)
	c := NewLogbookChooser(a, NewToastQueue())
	c.startEdit("home")

	c.fm.row = 21
	c.station.focusRow(21)
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !c.fm.btn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	c.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if c.mode != chooserList {
		t.Errorf("mode = %v, want list after saving", c.mode)
	}
	// Reopening the form must start with the button unfocused.
	c.startEdit("home")
	if c.fm.btn.Focus {
		t.Error("save button should not stay focused after reopening the form")
	}
}

// TestLogbookChooserAPRSTXReachable: the APRS TX checkbox and fields must stay
// reachable by Tab in the edit form, regardless of the Wavelog toggle.
func TestLogbookChooserAPRSTXReachable(t *testing.T) {
	// Wavelog enabled: Tab from Station ID lands on the shared-club checkbox,
	// then on APRS TX — never on the button before the section ends.
	a := newChooserTestApp(t)
	c := NewLogbookChooser(a, NewToastQueue())
	c.mode = chooserEdit
	c.station.WlEnabled = true
	c.station.BlurAll()
	c.fm.row = 19
	c.station.focusRow(19)
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if c.fm.btn.Focus || !c.station.wlSharedCbFocus {
		t.Fatalf("tab from Station ID: saveBtn.Focus=%v wlSharedCbFocus=%v, want shared-club checkbox focused",
			c.fm.btn.Focus, c.station.wlSharedCbFocus)
	}
	// Next Tab reaches APRS TX.
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if c.fm.btn.Focus || !c.station.aprsCbFocus {
		t.Fatalf("tab from shared-club checkbox: saveBtn.Focus=%v aprsCbFocus=%v, want button unfocused and APRS TX focused",
			c.fm.btn.Focus, c.station.aprsCbFocus)
	}
	// Next Tab reaches the button.
	c.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !c.fm.btn.Focus {
		t.Error("tab from APRS TX should focus the Save & Back button")
	}

	// Wavelog disabled: Tab from the Wavelog checkbox must still reach APRS TX.
	c2 := NewLogbookChooser(a, NewToastQueue())
	c2.mode = chooserEdit
	c2.station.WlEnabled = false
	c2.station.BlurAll()
	c2.fm.row = 15
	c2.station.focusRow(15)
	c2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if c2.fm.btn.Focus || !c2.station.aprsCbFocus {
		t.Fatalf("tab from Wavelog checkbox (disabled): saveBtn.Focus=%v aprsCbFocus=%v, want APRS TX focused",
			c2.fm.btn.Focus, c2.station.aprsCbFocus)
	}

	// APRS enabled: Tab from AprsComment reaches the APRS test button first,
	// and only Tab from the test button reaches Save & Back.
	c3 := NewLogbookChooser(a, NewToastQueue())
	c3.mode = chooserEdit
	c3.station.AprsEnabled = true
	c3.station.BlurAll()
	c3.fm.row = 27
	c3.station.focusRow(27)
	c3.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if c3.fm.btn.Focus || c3.station.aprsBtnFocus != 1 {
		t.Fatalf("tab from APRS comment: saveBtn.Focus=%v aprsBtnFocus=%d, want test button focused",
			c3.fm.btn.Focus, c3.station.aprsBtnFocus)
	}
	c3.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !c3.fm.btn.Focus {
		t.Error("tab from APRS test button should focus the Save & Back button")
	}
}

func TestRigChooserSaveBackButton(t *testing.T) {
	rc := newTestRigChooser(t)
	rc.startCreate()
	last := rc.fm.lastVisible(rc)
	rc.fm.row = last
	rc.form.focusRow(rigFormField(last))

	rc.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !rc.fm.btn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	if rc.fm.onLast(rc) {
		t.Error("rig form should not keep a field active while the button is focused")
	}
	if n := strings.Count(rc.viewForm(), "> "); n != 1 {
		t.Errorf("edit form shows %d active markers, want 1 (button only)", n)
	}
	rc.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if rc.fm.btn.Focus || !rc.fm.onLast(rc) {
		t.Error("shift+tab from button should return to the last field")
	}
	rc.fm.row = 0
	rc.form.focusRow(rigFieldName)
	rc.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !rc.fm.btn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
	if n := strings.Count(rc.viewForm(), "> "); n != 1 {
		t.Errorf("edit form shows %d active markers, want 1 (button only)", n)
	}
}

// TestRigChooserEditFocusConsistent: opening Edit Rig after a previous
// save/back session must keep the form focus index and the focused field in
// sync — exactly one active marker, on Rig name.
func TestRigChooserEditFocusConsistent(t *testing.T) {
	rc := newTestRigChooser(t)
	rc.app.Config.Rigs["r1"] = config.RigPreset{Name: "Main Rig", RadioBackend: "flrig", WsjtxEnabled: true}
	rc.names = []string{"r1"}
	rc.cursor = 0
	rc.mode = rigChooserList

	rc.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if rc.mode != rigChooserEdit {
		t.Fatalf("mode = %v, want edit", rc.mode)
	}
	if n := strings.Count(rc.viewForm(), "> "); n != 1 {
		t.Fatalf("edit form shows %d active markers after opening, want 1", n)
	}

	// Reach the button, save, and reopen — focus must reset cleanly.
	last := rc.fm.lastVisible(rc)
	rc.fm.row = last
	rc.form.focusRow(rigFormField(last))
	rc.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	rc.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if rc.mode != rigChooserList {
		t.Fatalf("mode = %v after save, want list", rc.mode)
	}

	rc.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if rc.mode != rigChooserEdit {
		t.Fatalf("mode = %v, want edit after reopening", rc.mode)
	}
	if !rc.form.Name.Focused() {
		t.Error("Rig name should be focused after reopening the edit form")
	}
	if n := strings.Count(rc.viewForm(), "> "); n != 1 {
		t.Errorf("edit form shows %d active markers after reopening, want 1", n)
	}
}

func TestOperatorChooserSaveBackButton(t *testing.T) {
	oc := newTestOperatorChooser(t)
	oc.startCreate()
	oc.fm.row = 1
	oc.form.focusRow(1)

	oc.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !oc.fm.btn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	// No field label may stay highlighted while the button is focused.
	if v := oc.form.View(); strings.Contains(v, fieldFocusedLabel.Render("Name")) ||
		strings.Contains(v, fieldFocusedLabel.Render("Callsign")) {
		t.Errorf("form should not highlight any field while the button is focused:\n%s", v)
	}
	oc.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if oc.fm.btn.Focus || !oc.fm.onLast(oc) {
		t.Error("shift+tab from button should return to the last field")
	}
	oc.fm.row = 0
	oc.form.focusRow(0)
	oc.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !oc.fm.btn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
	// Reopening the form must start with the button unfocused.
	oc.startCreate()
	if oc.fm.btn.Focus {
		t.Error("save button should not stay focused after reopening the form")
	}
}

func TestLogbookEditorSaveBackButton(t *testing.T) {
	le := NewLogbookEditor(LogbookEditorConfig{DB: nil, StationOperator: "OP", StationGrid: "JO90", StationCall: "SP9MOA"})
	le.mode = edModeEdit

	le.fm.row = int(qefContestID)
	le.focusRow(int(qefContestID))
	le.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !le.fm.btn.Focus {
		t.Fatal("tab from last field should focus the Save & Back button")
	}
	if le.fields[le.focus].Focused() {
		t.Error("editor should not keep a field active while the button is focused")
	}
	le.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if le.fm.btn.Focus || le.focus != qefContestID {
		t.Error("shift+tab from button should return to the last field")
	}
	le.fm.row = int(qefCall)
	le.focusRow(int(qefCall))
	le.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !le.fm.btn.Focus {
		t.Error("up from first field should focus the Save & Back button")
	}
}

func TestIntegrationMenuSaveBackButton(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logbooks = map[string]config.Logbook{"test": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}}}
	cfg.Integrations.HTTPServer.Enabled = true
	im := NewIntegrationMenu(cfg)
	im.width = 100
	im.height = 30

	// Tab away from QR Link must blur it — no stray cursor while focus moves.
	im.fm.row = imHTTPQRLink
	im.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if im.httpQRLink.Focused() {
		t.Error("QR Link should blur when focus moves away from it")
	}

	// Tab from the last visible item focuses the button.
	im.fm.row = im.fm.lastVisible(im)
	im.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !im.fm.btn.Focus {
		t.Fatal("tab from last item should focus the Save & Back button")
	}
	if im.fm.row != -1 {
		t.Errorf("button focused but list focus still active: %d, want -1", im.fm.row)
	}
	if im.httpQRLink.Focused() {
		t.Error("no field may stay focused while the Save & Back button is active")
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
	cfg.Integrations.Callbook.QRZ.Enabled = true
	cm := NewCallbookMenu(cfg)
	cm.width = 100
	cm.height = 30

	// Up from the first item focuses the button.
	cm.fm.row = cm.fm.firstVisible(cm)
	cm.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !cm.fm.btn.Focus {
		t.Fatal("up from first item should focus the Save & Back button")
	}
	if cm.fm.row != -1 {
		t.Errorf("button focused but list focus still active: %d, want -1", cm.fm.row)
	}
	if n := strings.Count(cm.View().Content, "> "); n != 1 {
		t.Errorf("callbook menu shows %d active markers, want 1 (button only)", n)
	}
	if cm.qrzUser.Focused() || cm.qrzPass.Focused() || cm.logPriority.Focused() {
		t.Error("no field may stay focused while the Save & Back button is active")
	}

	// Tab from the last item also focuses the button.
	cm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	cm.fm.row = cm.fm.lastVisible(cm)
	cm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !cm.fm.btn.Focus || cm.fm.row != -1 {
		t.Fatalf("tab from last item: saveBtn=%v row=%d, want button focused and list focus -1", cm.fm.btn.Focus, cm.fm.row)
	}

	// Space on the button attempts a save; with QRZ enabled but no username
	// it fails validation and the menu stays open with a single marker.
	cm.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cm.done || cm.saved {
		t.Error("save should be blocked by validation (QRZ username missing)")
	}
	if n := strings.Count(cm.View().Content, "> "); n > 1 {
		t.Errorf("callbook menu shows %d active markers after blocked save, want at most 1", n)
	}
}
