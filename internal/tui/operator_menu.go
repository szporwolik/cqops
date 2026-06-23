package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

type operatorMenuMode int

const (
	operatorList operatorMenuMode = iota
	operatorEdit
	operatorCreate
	operatorConfirmDelete
)

// OperatorChooser provides a list/add/edit/delete UI for operators.
type OperatorChooser struct {
	app    *app.App
	mode   operatorMenuMode
	ids    []string // first entry is "" (None), followed by operator IDs
	names  []string // first entry is "None", followed by display names
	cursor int
	done   bool

	form    OperatorForm
	editing string // id of operator being edited

	toasts *ToastQueue
	dialog *DialogModel
	width  int
	height int
}

// NewOperatorChooser creates a new operator chooser.
func NewOperatorChooser(a *app.App, tq *ToastQueue) *OperatorChooser {
	oc := &OperatorChooser{
		app:    a,
		mode:   operatorList,
		toasts: tq,
		form:   NewOperatorForm(),
	}
	oc.refreshIDs()
	return oc
}

// Init implements tea.Model.
func (oc *OperatorChooser) Init() tea.Cmd { return nil }

// refreshIDs reloads the operator list from config.
func (oc *OperatorChooser) refreshIDs() {
	oc.ids = []string{""}
	oc.names = []string{"None"}
	sorted := config.SortedOperatorIDs(oc.app.Config)
	for _, id := range sorted {
		op := oc.app.Config.Operators[id]
		oc.ids = append(oc.ids, id)
		oc.names = append(oc.names, config.OperatorDisplayName(&op))
	}
	// Keep cursor on active operator.
	oc.cursor = 0
	if oc.app.Logbook.ActiveOperator != "" {
		for i, id := range oc.ids {
			if id == oc.app.Logbook.ActiveOperator {
				oc.cursor = i
				break
			}
		}
	}
	if oc.cursor >= len(oc.ids) {
		oc.cursor = 0
	}
}

// Update handles input for the operator chooser.
func (oc *OperatorChooser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		oc.width = msg.Width
		oc.height = msg.Height

	case tea.KeyPressMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if oc.mode == operatorList {
				oc.done = true
				return oc, nil
			}
			oc.form.BlurAll()
			oc.mode = operatorList
			return oc, nil

		case oc.mode == operatorConfirmDelete:
			if oc.dialog == nil {
				// Skip – dialog not yet created.
			} else {
				updated, _ := oc.dialog.Update(msg)
				d := updated.(DialogModel)
				*oc.dialog = d
				if d.Done() {
					if d.Result.Value == "delete" {
						return oc, oc.deleteOperator()
					}
					oc.dialog = nil
					oc.mode = operatorList
				}
				return oc, nil
			}

		case oc.mode == operatorList && k.String() == "enter":
			if oc.cursor == 0 {
				return oc, oc.selectOperator()
			}
			if oc.cursor < len(oc.ids) {
				oc.startEdit(oc.ids[oc.cursor])
			}

		case oc.mode == operatorList && (k.String() == " " || msg.Code == ' ' || k.String() == "a"):
			return oc, oc.selectOperator()

		case oc.mode == operatorList && k.String() == "insert":
			oc.startCreate()

		case oc.mode == operatorList && (k.String() == "delete" || msg.Code == tea.KeyDelete):
			if oc.cursor > 0 && oc.cursor < len(oc.ids) {
				oc.mode = operatorConfirmDelete
				id := oc.ids[oc.cursor]
				op := oc.app.Config.Operators[id]
				displayName := config.OperatorDisplayName(&op)
				d := NewDialog("Delete Operator", "Delete \""+displayName+"\"?",
					DangerOption("Delete", "delete"),
					Option{Label: "Cancel", Value: "cancel"},
				)
				oc.dialog = &d
			}

		case oc.mode == operatorList && (msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if oc.cursor == 0 {
				oc.cursor = len(oc.ids) - 1
			} else {
				oc.cursor--
			}

		case oc.mode == operatorList && (msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if oc.cursor == len(oc.ids)-1 {
				oc.cursor = 0
			} else {
				oc.cursor++
			}

		case oc.mode == operatorEdit || oc.mode == operatorCreate:
			if cmd := oc.form.HandleKey(msg); cmd != nil {
				return oc, oc.saveForm()
			}
		}
	}

	return oc, nil
}

func (oc *OperatorChooser) startCreate() {
	oc.mode = operatorCreate
	oc.form = NewOperatorForm()
	oc.form.Focus()
	oc.editing = ""
}

func (oc *OperatorChooser) startEdit(id string) {
	op := oc.app.Config.Operators[id]
	oc.mode = operatorEdit
	oc.editing = id
	oc.form.SetOperator(&op)
	oc.form.Focus()
}

func (oc *OperatorChooser) selectOperator() tea.Cmd {
	if oc.cursor == 0 {
		oc.app.SetActiveOperator("")
		oc.toasts.Success("Operator: None (station operator)")
		applog.Info("Operator activated", "id", "none", "display", "None")
		if err := config.Save(oc.app.ConfigPath, oc.app.Config); err != nil {
			oc.toasts.Error("Save operator selection failed: " + err.Error())
		}
		return nil
	}
	id := oc.ids[oc.cursor]
	op := oc.app.Config.Operators[id]
	oc.app.SetActiveOperator(id)
	dn := config.OperatorDisplayName(&op)
	oc.toasts.Success(fmt.Sprintf("Operator activated: %s", dn))
	applog.Info("Operator activated", "id", id, "display", dn)
	if err := config.Save(oc.app.ConfigPath, oc.app.Config); err != nil {
		oc.toasts.Error("Save operator selection failed: " + err.Error())
	}
	return nil
}

func (oc *OperatorChooser) saveForm() tea.Cmd {
	call, name := oc.form.Values()
	call = strings.ToUpper(call)

	if call == "" {
		oc.toasts.Warn("Callsign is required")
		return nil
	}

	var savedName string
	if oc.mode == operatorCreate {
		// Check for duplicate callsign.
		if _, _, found := config.FindOperatorByCallsign(oc.app.Config, call); found {
			oc.toasts.Warn("Operator with callsign " + call + " already exists")
			return nil
		}
		id := config.NewID(call)
		if oc.app.Config.Operators == nil {
			oc.app.Config.Operators = make(map[string]config.Operator)
		}
		oc.app.Config.Operators[id] = config.Operator{ID: id, Callsign: call, Name: name}
		savedName = call
	} else {
		id := oc.editing
		op := oc.app.Config.Operators[id]
		op.Callsign = call
		op.Name = name
		oc.app.Config.Operators[id] = op
		savedName = call
	}

	oc.mode = operatorList
	oc.form.BlurAll()
	oc.refreshIDs()

	if err := config.Save(oc.app.ConfigPath, oc.app.Config); err != nil {
		oc.toasts.Error("Save " + savedName + " failed: " + err.Error())
	} else {
		oc.toasts.Success("Operator " + savedName + " saved")
		applog.Info("Operator saved", "callsign", savedName)
	}
	return nil
}

func (oc *OperatorChooser) deleteOperator() tea.Cmd {
	id := oc.ids[oc.cursor]
	displayName := id
	if op, ok := oc.app.Config.Operators[id]; ok {
		displayName = config.OperatorDisplayName(&op)
	}
	// Clear the deleted operator from any logbook that has it active.
	for k, lb := range oc.app.Config.Logbooks {
		if lb.ActiveOperator == id {
			lb.ActiveOperator = ""
			oc.app.Config.Logbooks[k] = lb
		}
	}
	// Also clear the in-memory logbook if it matches.
	if oc.app.Logbook.ActiveOperator == id {
		oc.app.Logbook.ActiveOperator = ""
	}
	delete(oc.app.Config.Operators, id)
	oc.mode = operatorList
	oc.refreshIDs()
	if err := config.Save(oc.app.ConfigPath, oc.app.Config); err != nil {
		oc.toasts.Error("Delete " + displayName + " failed: " + err.Error())
	} else {
		oc.toasts.Success("Operator " + displayName + " deleted")
		applog.Info("Operator deleted", "id", id)
	}
	return nil
}

// View renders the operator chooser.
func (oc *OperatorChooser) View() tea.View {
	if oc.done {
		return tea.NewView("")
	}

	switch oc.mode {
	case operatorList:
		return tea.NewView(oc.viewList())
	case operatorEdit, operatorCreate:
		return tea.NewView(oc.viewForm())
	case operatorConfirmDelete:
		body := oc.viewList()
		if oc.dialog != nil {
			body = RenderDialogOverlay(body, *oc.dialog, oc.width, oc.height)
		}
		return tea.NewView(body)
	}
	return tea.NewView("")
}

func (oc *OperatorChooser) viewList() string {
	var b strings.Builder
	w := oc.width
	if w < 40 {
		w = 80
	}
	h := oc.height
	if h < 10 {
		h = 24
	}
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	contentW := w - 8
	if contentW > partnerMapMaxW-8 {
		contentW = partnerMapMaxW - 8
	}
	if contentW < 20 {
		contentW = 20
	}

	activeOp := oc.app.Logbook.ActiveOperator

	for i, id := range oc.ids {
		prefix := "  "
		activeBadge := padOrTrunc("[      ]", 10)

		var callsignVal, nameVal string
		if id == "" {
			// "None" row.
			callsignVal = padOrTrunc("None", 16)
			nameVal = padOrTrunc("(station operator)", 20)
			if activeOp == "" {
				activeBadge = S.ToastSuccess.Render(padOrTrunc("[Active]", 10))
			}
		} else {
			op := oc.app.Config.Operators[id]
			callsignVal = padOrTrunc(op.Callsign, 16)
			nameVal = padOrTrunc(op.Name, 20)
			if id == activeOp {
				activeBadge = S.ToastSuccess.Render(padOrTrunc("[Active]", 10))
			}
		}

		if i == oc.cursor {
			prefix = S.FormPrefixOn.Render("> ")
			callsignVal = CursorStyle.Render(callsignVal)
			nameVal = CursorStyle.Render(nameVal)
			if activeBadge == padOrTrunc("[      ]", 10) {
				activeBadge = CursorStyle.Render(activeBadge)
			}
		}

		line := prefix + activeBadge + callsignVal + nameVal
		b.WriteString(padOrTrunc(line, contentW))
		if i < len(oc.ids)-1 {
			b.WriteString("\n")
		}
	}

	body := drawMenuWithHeader("Configuration \u2014 Operators", b.String(), w)
	return fillBody(body, contentH)
}

func (oc *OperatorChooser) viewForm() string {
	var b strings.Builder
	w := oc.width
	if w < 40 {
		w = 80
	}
	h := oc.height
	if h < 10 {
		h = 24
	}
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	title := "Create Operator"
	if oc.mode == operatorEdit {
		title = "Edit Operator"
	}

	b.WriteString(oc.form.View())

	body := drawMenuWithHeader("Configuration \u2014 Operators \u2014 "+title, b.String(), w)
	return fillBody(body, contentH)
}
