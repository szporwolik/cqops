package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	names := config.SortedRigIDs(a.Config)

	// If no rig is active but rigs are configured, auto-select the first one.
	if a.Logbook.Station.RigName == "" && len(names) > 0 {
		id := names[0]
		a.Logbook.Station.RigName = id
		lb := a.Config.Logbooks[a.LogbookName]
		lb.Station.RigName = id
		a.Config.Logbooks[a.LogbookName] = lb
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

	case tea.KeyPressMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if rc.mode == rigChooserList {
				rc.done = true
				return rc, nil
			}
			rc.form.blurAll()
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
				id := rc.names[rc.cursor]
				rp := rc.app.Config.Rigs[id]
				displayName := config.RigDisplayName(&rp)
				d := NewDialog("Delete Rig", "Delete \""+displayName+"\" configuration?",
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
	if contentH < 3 {
		contentH = 3
	}

	if len(rc.names) == 0 {
		b.WriteString("No rigs configured.\n")
	} else {
		activeRig := rc.app.Logbook.Station.RigName
		for i, id := range rc.names {
			rp := rc.app.Config.Rigs[id]
			displayName := config.RigDisplayName(&rp)
			prefix := "  "
			active := ""
			if id == activeRig {
				active = "[Active]"
			}
			info := rp.Model
			if rp.Antenna != "" {
				info += "  /  " + rp.Antenna
			}
			flrig := ""
			if rp.FlrigEnabled {
				flrig = "flrig"
			}
			if i == rc.cursor {
				prefix = S.FormPrefixOn.Render("> ")
			}
			lbl := S.FormLabelWide.Align(lipgloss.Left).Render(active)
			val := fmt.Sprintf("%s  %s  %s", displayName, info, flrig)
			if i == rc.cursor {
				lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(active)
				val = CursorStyle.Render(val)
			}
			line := lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
			b.WriteString(padOrTrunc(line, w-4))
			b.WriteString("\n")
		}
	}

	body := drawMenuBox(b.String(), w)
	return fillBody(body, contentH)
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
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	b.WriteString(rc.form.View().Content)

	body := drawMenuBox(b.String(), w)
	return fillBody(body, contentH)
}

func (rc *RigChooser) selectRig() tea.Cmd {
	if len(rc.names) == 0 {
		return nil
	}
	id := rc.names[rc.cursor]
	rp := rc.app.Config.Rigs[id]
	displayName := config.RigDisplayName(&rp)

	// Update the in-memory logbook — only store the RigName reference.
	rc.app.Logbook.Station.RigName = id
	// Persist to config map.
	lb := rc.app.Config.Logbooks[rc.app.LogbookName]
	lb.Station.RigName = id
	rc.app.Config.Logbooks[rc.app.LogbookName] = lb

	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Select " + displayName + " failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig \"" + displayName + "\" selected")
		applog.Info("Rig selected", "name", displayName)
		// Refresh names and stay in the menu.
		rc.refreshNames()
	}
	return nil
}

func (rc *RigChooser) refreshNames() {
	rc.names = config.SortedRigIDs(rc.app.Config)
	if rc.cursor >= len(rc.names) {
		rc.cursor = len(rc.names) - 1
	}
}

func (rc *RigChooser) startCreate() {
	rc.mode = rigChooserCreate
	rc.form.SetValues("", "", "")
	rc.form.SetFlrig(false, "localhost", "12345")
	rc.form.blurAll()
	rc.form.Rig.Focus()
	rc.editing = ""
}

func (rc *RigChooser) startEdit(id string) {
	rp := rc.app.Config.Rigs[id]
	rc.mode = rigChooserEdit
	rc.editing = id
	rc.form.SetValues(rp.Model, rp.Antenna, rp.Power)
	rc.form.SetFlrig(rp.FlrigEnabled, rp.FlrigHost, rp.FlrigPort)
	rc.form.blurAll()
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

	var savedName string
	if rc.mode == rigChooserCreate {
		// Check for duplicate by model name.
		if _, _, found := config.FindRigByModel(rc.app.Config, rig); found {
			rc.toasts.Error("Rig with model " + rig + " already exists")
			return nil
		}
		id := config.NewID(rig)
		if rc.app.Config.Rigs == nil {
			rc.app.Config.Rigs = make(map[string]config.RigPreset)
		}
		rc.app.Config.Rigs[id] = config.RigPreset{
			ID:           id,
			Model:        rig,
			Antenna:      ant,
			Power:        pwr,
			FlrigEnabled: flrigEnabled,
			FlrigHost:    flrigHost,
			FlrigPort:    flrigPort,
		}
		rc.names = append(rc.names, id)
		savedName = rig
	} else {
		id := rc.editing
		rp := rc.app.Config.Rigs[id]
		rp.Model = rig
		rp.Antenna = ant
		rp.Power = pwr
		rp.FlrigEnabled = flrigEnabled
		rp.FlrigHost = flrigHost
		rp.FlrigPort = flrigPort
		rc.app.Config.Rigs[id] = rp
		savedName = rig
	}

	rc.mode = rigChooserList
	rc.form.blurAll()
	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Save " + savedName + " failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig " + savedName + " saved")
		applog.Info("Rig saved", "name", savedName)
	}
	return nil
}

func (rc *RigChooser) viewConfirmDelete() string {
	id := rc.names[rc.cursor]
	rp := rc.app.Config.Rigs[id]
	displayName := config.RigDisplayName(&rp)
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
	b.WriteString(fmt.Sprintf("  Delete rig %q?\n", displayName))
	b.WriteString("  (y/N)")
	return fillBody(b.String(), contentH)
}

func (rc *RigChooser) deleteRig() tea.Cmd {
	if len(rc.names) == 0 {
		return nil
	}
	id := rc.names[rc.cursor]
	rp := rc.app.Config.Rigs[id]
	displayName := config.RigDisplayName(&rp)

	// Active rig protection
	if id == rc.app.Logbook.Station.RigName {
		rc.toasts.Error("Cannot delete " + displayName + " — it is the active rig. Select another first.")
		rc.mode = rigChooserList
		return nil
	}

	if len(rc.names) <= 1 {
		rc.toasts.Error("Cannot delete " + displayName + " — at least one rig must remain.")
		rc.mode = rigChooserList
		return nil
	}

	delete(rc.app.Config.Rigs, id)
	for i, n := range rc.names {
		if n == id {
			rc.names = append(rc.names[:i], rc.names[i+1:]...)
			break
		}
	}
	if rc.cursor >= len(rc.names) {
		rc.cursor = len(rc.names) - 1
	}

	rc.mode = rigChooserList
	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Delete " + displayName + " failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig " + displayName + " deleted")
		applog.Info("Rig deleted", "name", displayName)
	}
	return nil
}
