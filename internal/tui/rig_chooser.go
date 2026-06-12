package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

type rigChooserMode int

const (
	rigChooserList   rigChooserMode = iota
	rigChooserEdit
	rigChooserCreate
)

type RigChooser struct {
	app        *app.App
	mode       rigChooserMode
	names      []string
	cursor     int
	form       *RigForm
	editing    string
	statusMsg  string
	statusType string
	width      int
	height     int
	done       bool
}

func NewRigChooser(a *app.App) *RigChooser {
	names := make([]string, 0, len(a.Config.Rigs))
	for name := range a.Config.Rigs {
		names = append(names, name)
	}

	rf := NewRigForm("", "", "")
	rf.SetFlrig(false, "localhost", "12345")

	return &RigChooser{
		app:   a,
		mode:  rigChooserList,
		names: names,
		form:  rf,
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
		case k.String() == "ctrl+c" || k.String() == "esc":
			rc.done = true
			return rc, nil

		case rc.mode == rigChooserList && k.String() == "enter":
			return rc, rc.selectRig()

		case rc.mode == rigChooserList && k.String() == "c":
			rc.startCreate()

		case rc.mode == rigChooserList && k.String() == "e":
			if len(rc.names) > 0 {
				rc.startEdit(rc.names[rc.cursor])
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

func (rc *RigChooser) View() string {
	if rc.done {
		return ""
	}

	switch rc.mode {
	case rigChooserList:
		return rc.viewList()
	case rigChooserEdit, rigChooserCreate:
		return rc.viewForm()
	}
	return ""
}

func (rc *RigChooser) viewList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Rigs"))
	b.WriteString("\n\n")

	if len(rc.names) == 0 {
		b.WriteString("No rigs configured.\n\n")
		b.WriteString(helpStyle.Render("c to create  |  Esc to close"))
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
			flrig = "♜"
		}
		b.WriteString(fmt.Sprintf("%s%s %s %s  %s\n", marker, active, name, info, flrig))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter to select  |  e to edit  |  c to create  |  Esc to close"))
	return b.String()
}

func (rc *RigChooser) viewForm() string {
	var b strings.Builder
	if rc.mode == rigChooserEdit {
		b.WriteString(titleStyle.Render("Edit Rig " + rc.editing))
	} else {
		b.WriteString(titleStyle.Render("Create Rig"))
	}
	b.WriteString("\n\n")

	b.WriteString(rc.form.View())

	b.WriteString("\n\n")
	if rc.statusMsg != "" {
		if rc.statusType == "error" {
			b.WriteString(errorStyle.Render(rc.statusMsg))
		} else {
			b.WriteString(successStyle.Render(rc.statusMsg))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Ctrl+S to save  |  Enter/Tab/↓ to next  |  Shift+Tab/↑ to previous  |  Esc to cancel"))
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

	config.Save(rc.app.ConfigPath, rc.app.Config)
	rc.done = true
	return nil
}

func (rc *RigChooser) startCreate() {
	rc.mode = rigChooserCreate
	rc.form.SetValues("", "", "")
	rc.form.SetFlrig(false, "localhost", "12345")
	rc.form.Rig.Focus()
	rc.statusMsg = ""
	rc.editing = ""
}

func (rc *RigChooser) startEdit(name string) {
	rp := rc.app.Config.Rigs[name]
	rc.mode = rigChooserEdit
	rc.editing = name
	rc.form.SetValues(rp.Model, rp.Antenna, rp.Power)
	rc.form.SetFlrig(rp.FlrigEnabled, rp.FlrigHost, rp.FlrigPort)
	rc.form.Rig.Focus()
	rc.statusMsg = ""
}

func (rc *RigChooser) saveForm() tea.Cmd {
	rig, ant, pwr := rc.form.Values()
	flrigEnabled, flrigHost, flrigPort := rc.form.FlrigValues()

	if rig == "" {
		rc.statusMsg = "Rig model is required"
		rc.statusType = "error"
		return nil
	}
	if flrigEnabled {
		if flrigHost == "" {
			rc.statusMsg = "Flrig host is required"
			rc.statusType = "error"
			return nil
		}
		if flrigPort == "" {
			rc.statusMsg = "Flrig port is required"
			rc.statusType = "error"
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
	rc.statusMsg = ""
	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.statusMsg = "Config save failed: " + err.Error()
		rc.statusType = "error"
	}
	return nil
}
