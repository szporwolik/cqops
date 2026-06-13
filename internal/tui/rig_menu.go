package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	width   int
	height  int
	done    bool
}

func NewRigChooser(a *app.App, tq *ToastQueue) *RigChooser {
	names := make([]string, 0, len(a.Config.Rigs))
	for name := range a.Config.Rigs {
		names = append(names, name)
	}

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

	case tea.KeyMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if rc.mode == rigChooserList {
				rc.done = true
				return rc, nil
			}
			rc.mode = rigChooserList

		case rc.mode == rigChooserConfirmDelete:
			switch k.String() {
			case "y", "Y":
				return rc, rc.deleteRig()
			default:
				rc.mode = rigChooserList
			}
			return rc, nil

		case rc.mode == rigChooserList && k.String() == "enter":
			return rc, rc.selectRig()

		case rc.mode == rigChooserList && k.String() == "c":
			rc.startCreate()

		case rc.mode == rigChooserList && k.String() == "e":
			if len(rc.names) > 0 {
				rc.startEdit(rc.names[rc.cursor])
			}

		case rc.mode == rigChooserList && k.String() == "d":
			if len(rc.names) > 0 {
				rc.mode = rigChooserConfirmDelete
			}

		case rc.mode == rigChooserList && (msg.Type == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if rc.cursor > 0 {
				rc.cursor--
			}

		case rc.mode == rigChooserList && (msg.Type == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if rc.cursor < len(rc.names)-1 {
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
		return "Enter to select  e to edit  c to create  d to delete  Esc to go back"
	case rigChooserEdit, rigChooserCreate:
		return "Ctrl+S to save  Tab/↓/↑ to navigate  Esc to discard"
	case rigChooserConfirmDelete:
		return "Delete this rig? (y/N)"
	}
	return ""
}

func (rc *RigChooser) View() string {
	if rc.done {
		return ""
	}

	switch rc.mode {
	case rigChooserList:
		return rc.viewList()
	case rigChooserEdit, rigChooserCreate:
		return rc.viewForm()
	case rigChooserConfirmDelete:
		return rc.viewConfirmDelete()
	}
	return ""
}

func (rc *RigChooser) viewList() string {
	var b strings.Builder
	bodyW := rc.width - 2
	if bodyW < 30 {
		bodyW = 30
	}
	title := "── Configuration — Rigs "
	b.WriteString(section(title, bodyW))
	b.WriteString("\n\n")

	if len(rc.names) == 0 {
		b.WriteString("No rigs configured.\n\n")
		return b.String()
	}

	activeRig := rc.app.Logbook.Station.RigName
	for i, name := range rc.names {
		rp := rc.app.Config.Rigs[name]
		marker := "  "
		if i == rc.cursor {
			marker = cursorStyle.Render("> ")
		}
		active := " "
		if name == activeRig {
			active = "*"
		}
		info := rp.Model
		if rp.Antenna != "" {
			info += "  /  " + rp.Antenna
		}
		flrig := " "
		if rp.FlrigEnabled {
			flrig = "flrig"
		}
		b.WriteString(fmt.Sprintf("%s%s %s %s  %s\n", marker, active, name, info, flrig))
	}

	return b.String()
}

func (rc *RigChooser) viewForm() string {
	var b strings.Builder
	bodyW := rc.width - 2
	if bodyW < 30 {
		bodyW = 30
	}
	t := "── Configuration — Create Rig "
	if rc.mode == rigChooserEdit {
		t = "── Configuration — Edit Rig " + rc.editing + " "
	}
	b.WriteString(section(t, bodyW))
	b.WriteString("\n\n")

	b.WriteString(rc.form.View())

	return b.String()
}

func (rc *RigChooser) selectRig() tea.Cmd {
	if len(rc.names) == 0 {
		return nil
	}
	name := rc.names[rc.cursor]
	rp := rc.app.Config.Rigs[name]

	lb := rc.app.Config.Logbooks[rc.app.LogbookName]
	lb.Station.Rig = rp.Model
	lb.Station.Antenna = rp.Antenna
	lb.Station.Power = rp.Power
	lb.Station.RigName = name
	rc.app.Config.Logbooks[rc.app.LogbookName] = lb
	rc.app.Logbook = &lb

	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Config save failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig \"" + name + "\" selected")
		applog.Info("Rig selected", "name", name)
	}
	rc.done = true
	return nil
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
	bodyW := rc.width - 2
	if bodyW < 30 {
		bodyW = 30
	}
	b.WriteString(section("── Delete Rig ", bodyW))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Delete rig %q?\n", name))
	b.WriteString("  (y/N)")
	return b.String()
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
