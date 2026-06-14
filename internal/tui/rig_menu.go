package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

type rigChooserMode int

const (
	rigChooserList rigChooserMode = iota
	rigChooserEdit
	rigChooserCreate
	rigChooserConfirmDelete
)

type RigChooser struct {
	app     *app.App
	mode    rigChooserMode
	names   []string
	cursor  int
	form    *RigForm
	editing string
	toasts  *ToastQueue
	dialog  *DialogModel
	width   int
	height  int
	done    bool
}

func NewRigChooser(a *app.App, tq *ToastQueue) *RigChooser {
	names := make([]string, 0, len(a.Config.Rigs))
	for name := range a.Config.Rigs {
		names = append(names, name)
	}
	slices.Sort(names)

	rf := NewRigForm("", "", "")
	rf.SetFlrig(false, "localhost", "12345")

	return &RigChooser{
		app:    a,
		mode:   rigChooserList,
		names:  names,
		form:   rf,
		toasts: tq,
	}
}

func (rc *RigChooser) Init() tea.Cmd { return nil }

func (rc *RigChooser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rc.width = msg.Width
		rc.height = msg.Height

	case tea.KeyPressMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if rc.mode == rigChooserList {
				rc.done = true
				return rc, nil
			}
			rc.mode = rigChooserList

		case rc.mode == rigChooserConfirmDelete:
		if rc.dialog == nil {
			// Skip - dialog not yet created
		} else {
			updated, _ := rc.dialog.Update(msg)
			d := updated.(DialogModel)
			*rc.dialog = d
			if d.Done() {
				if d.Result.Value == "delete" {
					return rc, rc.deleteRig()
				}
				rc.dialog = nil
				rc.mode = rigChooserList
			}
			return rc, nil
		}

		case rc.mode == rigChooserList && k.String() == "enter":
			return rc, rc.selectRig()

		case rc.mode == rigChooserList && k.String() == "e":
			if len(rc.names) > 0 {
				rc.startEdit(rc.names[rc.cursor])
			}

		case rc.mode == rigChooserList && k.String() == "insert":
			rc.startCreate()

		case rc.mode == rigChooserList && (k.String() == "delete" || msg.Code == tea.KeyDelete):
			if len(rc.names) > 0 {
				rc.mode = rigChooserConfirmDelete
				name := rc.names[rc.cursor]
				d := NewDialog("Delete Rig", "Delete \""+name+"\" configuration?",
					DangerOption("Delete", "delete"),
					Option{Label: "Cancel", Value: "cancel"},
				)
				rc.dialog = &d
			}

		case rc.mode == rigChooserList && (msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if rc.cursor == 0 {
				rc.cursor = len(rc.names) - 1
			} else {
				rc.cursor--
			}

		case rc.mode == rigChooserList && (msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if rc.cursor == len(rc.names)-1 {
				rc.cursor = 0
			} else {
				rc.cursor++
			}

		case rc.mode == rigChooserEdit || rc.mode == rigChooserCreate:
			if cmd := rc.form.HandleKey(msg); cmd != nil {
				return rc, rc.saveForm()
			}
		}
	}

	return rc, nil
}

func (rc *RigChooser) FooterText() string {
	switch rc.mode {
	case rigChooserList:
		return "Enter to activate  e to edit  Ins to create  Del to delete  Esc to go back"
	case rigChooserEdit, rigChooserCreate:
		return "Ctrl+S to save  ↑↓/Tab to navigate  Esc to discard"
	case rigChooserConfirmDelete:
		return "Delete this rig? (y/N)"
	}
	return ""
}

func (rc *RigChooser) View() tea.View {
	if rc.done {
		return tea.NewView("")
	}

	switch rc.mode {
	case rigChooserList:
		return tea.NewView(rc.viewList())
	case rigChooserEdit, rigChooserCreate:
		return tea.NewView(rc.viewForm())
	case rigChooserConfirmDelete:
		body := rc.viewList()
		if rc.dialog != nil {
			body = RenderDialogOverlay(body, *rc.dialog, rc.width, rc.height)
		}
		return tea.NewView(body)
	}
	return tea.NewView("")
}

func (rc *RigChooser) viewList() string {
	var b strings.Builder
	w := rc.width
	if w < 40 {
		w = 80
	}
	h := rc.height
	if h < 10 {
		h = 24
	}
	contentH := contentHeight(h)
	b.WriteString(menuTitle("Configuration — Rigs", w))
	b.WriteString("\n\n")

	if len(rc.names) == 0 {
		b.WriteString(menuLine("No rigs configured.", w))
		b.WriteString("\n")
		return fillBody(b.String(), contentH)
	}

	activeRig := rc.app.Logbook.Station.RigName
	for i, name := range rc.names {
		rp := rc.app.Config.Rigs[name]
		marker := "  "
		if i == rc.cursor {
			marker = CursorStyle.Render("> ")
		}
		active := "        "
		if name == activeRig {
			active = "[Active]"
		}
		info := rp.Model
		if rp.Antenna != "" {
			info += "  /  " + rp.Antenna
		}
		flrig := " "
		if rp.FlrigEnabled {
			flrig = "flrig"
		}
		// Selected row: wrap in pink+Surface to prevent bg leak after
		// CursorStyle's \x1b[0m reset.
		line := fmt.Sprintf("%s%s %s %s  %s", marker, active, name, info, flrig)
		if i == rc.cursor {
			line = CursorStyle.Render("> ") + CursorStyle.Render(fmt.Sprintf("%s %s %s  %s", active, name, info, flrig))
		}
		b.WriteString(menuLine(line, w))
		b.WriteString("\n")
	}

	return fillBody(b.String(), contentH)
}

func (rc *RigChooser) viewForm() string {
	var b strings.Builder
	w := rc.width
	if w < 40 {
		w = 80
	}
	h := rc.height
	if h < 10 {
		h = 24
	}
	contentH := h - 4
	if contentH < 3 {
		contentH = 3
	}
	b.WriteString(menuTitle("Configuration — Edit Rig", w))
	b.WriteString("\n\n")

	b.WriteString(rc.form.View().Content)

	return fillBody(b.String(), contentH)
}

func (rc *RigChooser) selectRig() tea.Cmd {
	if len(rc.names) == 0 {
		return nil
	}
	name := rc.names[rc.cursor]
	rp := rc.app.Config.Rigs[name]

	// Update the in-memory logbook directly.
	rc.app.Logbook.Station.Rig = rp.Model
	rc.app.Logbook.Station.Antenna = rp.Antenna
	rc.app.Logbook.Station.Power = rp.Power
	rc.app.Logbook.Station.RigName = name
	// Persist to config map.
	lb := rc.app.Config.Logbooks[rc.app.LogbookName]
	lb.Station = rc.app.Logbook.Station
	rc.app.Config.Logbooks[rc.app.LogbookName] = lb

	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Config save failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig \"" + name + "\" selected")
		applog.Info("Rig selected", "name", name)
		// Refresh names and stay in the menu.
		rc.refreshNames()
	}
	return nil
}

func (rc *RigChooser) refreshNames() {
	names := make([]string, 0, len(rc.app.Config.Rigs))
	for n := range rc.app.Config.Rigs {
		names = append(names, n)
	}
	slices.Sort(names)
	rc.names = names
	if rc.cursor >= len(rc.names) {
		rc.cursor = len(rc.names) - 1
	}
}

func (rc *RigChooser) startCreate() {
	rc.mode = rigChooserCreate
	rc.form.SetValues("", "", "")
	rc.form.SetFlrig(false, "localhost", "12345")
	rc.form.Rig.Focus()
	rc.editing = ""
}

func (rc *RigChooser) startEdit(name string) {
	rp := rc.app.Config.Rigs[name]
	rc.mode = rigChooserEdit
	rc.editing = name
	rc.form.SetValues(rp.Model, rp.Antenna, rp.Power)
	rc.form.SetFlrig(rp.FlrigEnabled, rp.FlrigHost, rp.FlrigPort)
	rc.form.Rig.Focus()
}

func (rc *RigChooser) saveForm() tea.Cmd {
	rig, ant, pwr := rc.form.Values()
	flrigEnabled, flrigHost, flrigPort := rc.form.FlrigValues()

	if rig == "" {
		rc.toasts.Error("Rig model is required")
		return nil
	}
	if flrigEnabled {
		if flrigHost == "" {
			rc.toasts.Error("Flrig host is required")
			return nil
		}
		if flrigPort == "" {
			rc.toasts.Error("Flrig port is required")
			return nil
		}
	}

	if rc.mode == rigChooserCreate {
		name := rig
		for i := 1; rc.app.Config.Rigs[name].Model != ""; i++ {
			name = fmt.Sprintf("%s-%d", rig, i)
		}
		if rc.app.Config.Rigs == nil {
			rc.app.Config.Rigs = make(map[string]config.RigPreset)
		}
		rc.app.Config.Rigs[name] = config.RigPreset{
			Model:        rig,
			Antenna:      ant,
			Power:        pwr,
			FlrigEnabled: flrigEnabled,
			FlrigHost:    flrigHost,
			FlrigPort:    flrigPort,
		}
		rc.names = append(rc.names, name)
	} else {
		name := rc.editing
		rp := rc.app.Config.Rigs[name]
		rp.Model = rig
		rp.Antenna = ant
		rp.Power = pwr
		rp.FlrigEnabled = flrigEnabled
		rp.FlrigHost = flrigHost
		rp.FlrigPort = flrigPort
		rc.app.Config.Rigs[name] = rp
	}

	rc.mode = rigChooserList
	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Config save failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig saved")
		applog.Info("Rig config saved")
	}
	return nil
}

func (rc *RigChooser) viewConfirmDelete() string {
	name := rc.names[rc.cursor]
	var b strings.Builder
	w := rc.width
	if w < 40 {
		w = 80
	}
	h := rc.height
	if h < 10 {
		h = 24
	}
	contentH := h - 4
	if contentH < 3 {
		contentH = 3
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Delete rig %q?\n", name))
	b.WriteString("  (y/N)")
	return fillBody(b.String(), contentH)
}

func (rc *RigChooser) deleteRig() tea.Cmd {
	if len(rc.names) == 0 {
		return nil
	}
	name := rc.names[rc.cursor]

	// Active rig protection
	if name == rc.app.Logbook.Station.RigName || (name == "default" && rc.app.Logbook.Station.RigName == "") {
		rc.toasts.Error("Cannot delete active rig. Select another first.")
		rc.mode = rigChooserList
		return nil
	}

	if len(rc.names) <= 1 {
		rc.toasts.Error("Cannot delete the last rig. At least one must remain.")
		rc.mode = rigChooserList
		return nil
	}

	delete(rc.app.Config.Rigs, name)
	for i, n := range rc.names {
		if n == name {
			rc.names = append(rc.names[:i], rc.names[i+1:]...)
			break
		}
	}
	if rc.cursor >= len(rc.names) {
		rc.cursor = len(rc.names) - 1
	}

	rc.mode = rigChooserList
	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Config save failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig " + name + " deleted")
		applog.Info("Rig deleted", "name", name)
	}
	return nil
}
