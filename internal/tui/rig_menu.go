package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
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
	app          *app.App
	mode         rigChooserMode
	names        []string
	cursor       int
	form         *RigForm
	editing      string
	toasts       *ToastQueue
	dialog       *DialogModel
	width        int
	height       int
	done         bool
	needsRefresh bool // set by saveForm when active rig config changed

	// Viewport for scrolling form/content on small terminals.
	vp              viewport.Model
	lastFormContent string
	lastListContent string
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

	rf := NewRigForm("e.g. IC-7300 (optional)", "e.g. Dipole (optional)", "e.g. 100")
	rf.SetBackend(0, "", "")

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
			return rc, nil

		case rc.mode == rigChooserConfirmDelete:
			if rc.dialog == nil {
				// Skip - dialog not yet created
			} else {
				updated, _ := rc.dialog.Update(msg)
				d, ok := updated.(DialogModel)
				if !ok {
					return rc, nil
				}
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
			if len(rc.names) > 0 {
				rc.startEdit(rc.names[rc.cursor])
			}

		case rc.mode == rigChooserList && (k.String() == " " || k.Code == ' ' || k.String() == "a"):
			if len(rc.names) > 0 {
				return rc, rc.selectRig()
			}

		case rc.mode == rigChooserList && k.String() == "insert":
			rc.startCreate()

		case rc.mode == rigChooserList && (k.String() == "ctrl+d"):
			if len(rc.names) > 0 {
				return rc, rc.duplicateRig()
			}

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

		case rc.mode == rigChooserList && (k.String() == "pgup" || k.String() == "pgdown" || k.String() == "home" || k.String() == "end"):
			rc.vp, _ = rc.vp.Update(msg)
			return rc, nil

		case rc.mode == rigChooserList && (msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if rc.cursor == 0 {
				rc.cursor = len(rc.names) - 1
			} else {
				rc.cursor--
			}
			scrollVpToLine(&rc.vp, rc.cursor)

		case rc.mode == rigChooserList && (msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if rc.cursor == len(rc.names)-1 {
				rc.cursor = 0
			} else {
				rc.cursor++
			}
			scrollVpToLine(&rc.vp, rc.cursor)

		case rc.mode == rigChooserEdit || rc.mode == rigChooserCreate:
			switch {
			case k.String() == "pgup", k.String() == "pgdown", k.String() == "home", k.String() == "end":
				rc.vp, _ = rc.vp.Update(msg)
				return rc, nil
			default:
				if cmd := rc.form.HandleKey(msg); cmd != nil {
					return rc, rc.saveForm()
				}
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

	if len(rc.names) == 0 {
		b.WriteString("No rigs configured.\n")
	} else {
		contentW := w - 8
		if contentW > partnerMapMaxW-8 {
			contentW = partnerMapMaxW - 8
		}
		if contentW < 20 {
			contentW = 20
		}

		activeRig := rc.app.Logbook.Station.RigName
		for i, id := range rc.names {
			rp := rc.app.Config.Rigs[id]
			dn := config.RigDisplayName(&rp)
			model := rp.Model
			antenna := rp.Antenna
			if antenna == "" {
				antenna = "—"
			}

			// Truncate/pad raw values to fixed column widths before styling.
			nameVal := padOrTrunc(dn, 24)
			modelVal := padOrTrunc(model, 16)

			prefix := "  "
			activeBadge := padOrTrunc("[      ]", 10)
			if id == activeRig {
				activeBadge = S.ToastSuccess.Render(padOrTrunc("[Active]", 10))
			}

			if i == rc.cursor {
				prefix = S.FormPrefixOn.Render("> ")
				nameVal = CursorStyle.Render(nameVal)
				modelVal = CursorStyle.Render(modelVal)
				antenna = CursorStyle.Render(antenna)
			}

			line := prefix + activeBadge + nameVal + modelVal + antenna
			b.WriteString(padOrTrunc(line, contentW))
			if i < len(rc.names)-1 {
				b.WriteString("\n")
			}
		}
	}

	return renderScrollableMenu("Configuration \u2014 Rig Profiles", b.String(), &rc.vp, &rc.lastListContent, w, h)
}

func (rc *RigChooser) viewForm() string {
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

	// Use viewport for scrollable form body on small terminals.
	boxW := w
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	vpW := boxW - 4 // account for menu box left+right padding
	if vpW < 20 {
		vpW = 20
	}
	// Overhead: header(1) + blank row(1) + scroll hint(1) = 3 lines.
	vpH := contentH - 3
	if vpH < 4 {
		vpH = 4
	}
	rc.form.width = vpW
	bodyStr := rc.form.View().Content
	rc.vp.SetWidth(vpW)
	rc.vp.SetHeight(vpH)
	if rc.vp.TotalLineCount() == 0 || bodyStr != rc.lastFormContent {
		rc.vp.SetContent(bodyStr)
		rc.lastFormContent = bodyStr
	}
	if rc.vp.PastBottom() {
		rc.vp.SetYOffset(rc.vp.TotalLineCount() - rc.vp.VisibleLineCount())
	}
	header := S.Title.Width(boxW).Render("Configuration \u2014 Rig Profiles \u2014 Edit Rig")
	vpContent := rc.vp.View()
	hintLine := DimStyle.Width(vpW).Render(scrollHint(rc.vp))
	if hintLine == "" {
		hintLine = strings.Repeat(" ", vpW)
	}
	vpContent = lipgloss.JoinVertical(lipgloss.Left, vpContent, hintLine)
	box := menuBoxStyle.Width(boxW).Render(vpContent)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", box)
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
		rc.needsRefresh = true // trigger rig client reconnect in parent
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
	rc.form.SetValues("", "", "", "")
	rc.form.SetBackend(0, "", "")
	rc.form.SetRotor(0, "", "")
	rc.form.SetWsjtx(false, "127.0.0.1", "2233")
	rc.form.blurAll()
	rc.form.Name.Focus()
	rc.editing = ""
}

func (rc *RigChooser) startEdit(id string) {
	rp := rc.app.Config.Rigs[id]
	rc.mode = rigChooserEdit
	rc.editing = id
	rc.form.SetValues(rp.Name, rp.Model, rp.Antenna, rp.Power)
	backendIdx := 0
	switch rp.RadioBackend {
	case "hamlib":
		backendIdx = 1
	case "flrig":
		backendIdx = 2
	}
	// Fallback: pre-backend configs with FlrigEnabled.
	if backendIdx == 0 && rp.FlrigEnabled {
		backendIdx = 2
	}
	host, port := rp.HamlibRadioHost, rp.HamlibRadioPort
	if backendIdx == 2 {
		host, port = rp.FlrigHost, rp.FlrigPort
	}
	rc.form.SetBackend(backendIdx, host, port)
	rotorIdx := 0
	if rp.RotorBackend == "hamlib" {
		rotorIdx = 1
	}
	rc.form.SetRotor(rotorIdx, rp.RotorHamlibHost, rp.RotorHamlibPort)
	rc.form.SetWsjtx(rp.WsjtxEnabled, rp.WsjtxUDPHost, fmt.Sprintf("%d", rp.WsjtxUDPPort))
	rc.form.blurAll()
	rc.form.Name.Focus()
}

func (rc *RigChooser) saveForm() tea.Cmd {
	nm, rig, ant, pwr := rc.form.Values()
	radioBackend, radioBackendHost, radioBackendPort := rc.form.BackendValues()
	rotorBackend, rotorHost, rotorPort := rc.form.RotorValues()
	wsjtxEnabled, wsjtxHost, wsjtxPortStr := rc.form.WsjtxValues()
	wsjtxPort, _ := strconv.Atoi(wsjtxPortStr)
	if wsjtxPort <= 0 {
		wsjtxPort = 2233
	}

	flrigHost, flrigPort := "", ""
	hamlibHost, hamlibPort := "", ""
	switch radioBackend {
	case "flrig":
		flrigHost, flrigPort = radioBackendHost, radioBackendPort
	case "hamlib":
		hamlibHost, hamlibPort = radioBackendHost, radioBackendPort
	}

	if nm == "" {
		rc.toasts.Warn("Rig name is required")
		return nil
	}
	if radioBackend == "flrig" {
		if flrigHost == "" {
			rc.toasts.Warn("Flrig host is required")
			return nil
		}
		if flrigPort == "" {
			rc.toasts.Warn("Flrig port is required")
			return nil
		}
	}
	if radioBackend == "hamlib" {
		if hamlibHost == "" {
			rc.toasts.Warn("Hamlib host is required")
			return nil
		}
		if hamlibPort == "" {
			rc.toasts.Warn("Hamlib port is required")
			return nil
		}
	}
	if rotorBackend == "hamlib" {
		if rotorHost == "" {
			rc.toasts.Warn("Rotator hamlib host is required")
			return nil
		}
		if rotorPort == "" {
			rc.toasts.Warn("Rotator hamlib port is required")
			return nil
		}
	}

	var savedName string
	if rc.mode == rigChooserCreate {
		// Skip duplicate check when rig model is empty (optional field).
		if rig != "" {
			if _, _, found := config.FindRigByModel(rc.app.Config, rig); found {
				rc.toasts.Warn("Rig with model " + rig + " already exists")
				return nil
			}
		}
		id := config.NewID(rig)
		if rc.app.Config.Rigs == nil {
			rc.app.Config.Rigs = make(map[string]config.RigPreset)
		}
		rc.app.Config.Rigs[id] = config.RigPreset{
			ID:              id,
			Name:            nm,
			Model:           rig,
			Antenna:         ant,
			Power:           pwr,
			RadioBackend:    radioBackend,
			FlrigHost:       flrigHost,
			FlrigPort:       flrigPort,
			HamlibRadioHost: hamlibHost,
			HamlibRadioPort: hamlibPort,
			RotorBackend:    rotorBackend,
			RotorHamlibHost: rotorHost,
			RotorHamlibPort: rotorPort,
			WsjtxEnabled:    wsjtxEnabled,
			WsjtxUDPHost:    wsjtxHost,
			WsjtxUDPPort:    wsjtxPort,
		}
		rc.names = append(rc.names, id)
		savedName = rig
	} else {
		id := rc.editing
		rp := rc.app.Config.Rigs[id]
		rp.Name = nm
		rp.Model = rig
		rp.Antenna = ant
		rp.Power = pwr
		rp.RadioBackend = radioBackend
		rp.FlrigHost = flrigHost
		rp.FlrigPort = flrigPort
		rp.HamlibRadioHost = hamlibHost
		rp.HamlibRadioPort = hamlibPort
		rp.RotorBackend = rotorBackend
		rp.RotorHamlibHost = rotorHost
		rp.RotorHamlibPort = rotorPort
		rp.WsjtxEnabled = wsjtxEnabled
		rp.WsjtxUDPHost = wsjtxHost
		rp.WsjtxUDPPort = wsjtxPort
		rc.app.Config.Rigs[id] = rp
		savedName = rig
	}

	rc.mode = rigChooserList
	rc.form.blurAll()
	rc.needsRefresh = true
	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Save " + savedName + " failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig " + savedName + " saved")
		applog.Info("Rig saved", "name", savedName)
	}
	return nil
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
		rc.toasts.Warn("Cannot delete " + displayName + " — it is the active rig. Select another first.")
		rc.mode = rigChooserList
		return nil
	}

	if len(rc.names) <= 1 {
		rc.toasts.Warn("Cannot delete " + displayName + " — at least one rig must remain.")
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

func (rc *RigChooser) duplicateRig() tea.Cmd {
	id := rc.names[rc.cursor]
	rp := rc.app.Config.Rigs[id]
	displayName := config.RigDisplayName(&rp)

	cloneName := displayName + " (Clone)"
	if rp.Name != "" {
		cloneName = rp.Name + " (Clone)"
	}

	// Build a clone with the same settings but a new ID and name.
	clone := rp
	clone.ID = config.NewID(cloneName)
	clone.Name = cloneName

	if rc.app.Config.Rigs == nil {
		rc.app.Config.Rigs = make(map[string]config.RigPreset)
	}
	rc.app.Config.Rigs[clone.ID] = clone
	rc.refreshNames()

	// Position cursor on the newly created clone.
	for i, n := range rc.names {
		if n == clone.ID {
			rc.cursor = i
			break
		}
	}

	if err := config.Save(rc.app.ConfigPath, rc.app.Config); err != nil {
		rc.toasts.Error("Duplicate " + displayName + " failed: " + err.Error())
	} else {
		rc.toasts.Success("Rig \"" + displayName + "\" duplicated as \"" + cloneName + "\"")
		applog.Info("Rig duplicated", "original", displayName, "clone", cloneName)
	}
	return nil
}
