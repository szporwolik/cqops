package tui

import (
	"fmt"
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

type chooserMode int

const (
	chooserList chooserMode = iota
	chooserEdit
	chooserCreate
	chooserConfirmDelete
)

type LogbookChooser struct {
	app     *app.App
	mode    chooserMode
	names   []string
	cursor  int
	station *StationForm
	editing string
	toasts  *ToastQueue
	dialog  *DialogModel
	width   int
	height  int
	done    bool
}

func NewLogbookChooser(a *app.App, tq *ToastQueue) *LogbookChooser {
	names := make([]string, 0, len(a.Config.Logbooks))
	for name := range a.Config.Logbooks {
		names = append(names, name)
	}
	slices.Sort(names)

	// Start cursor on the active logbook.
	cursor := 0
	for i, n := range names {
		if n == a.Config.ActiveLogbook {
			cursor = i
			break
		}
	}

	return &LogbookChooser{
		app:     a,
		mode:    chooserList,
		names:   names,
		cursor:  cursor,
		station: NewStationForm("CALLSIGN", "operator", "GRID"),
		toasts:  tq,
	}
}

func (c *LogbookChooser) Init() tea.Cmd { return nil }

func (c *LogbookChooser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case tea.KeyPressMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if c.mode == chooserList {
				c.done = true
				return c, nil
			}
			c.mode = chooserList

		case c.mode == chooserConfirmDelete:
		if c.dialog == nil {
			// Skip - dialog not yet created
		} else {
			updated, _ := c.dialog.Update(msg)
			d := updated.(DialogModel)
			*c.dialog = d
			if d.Done() {
				if d.Result.Value == "delete" {
					return c, c.deleteLogbook()
				}
				c.dialog = nil
				c.mode = chooserList
			}
			return c, nil
		}

		case c.mode == chooserList && k.String() == "enter":
			return c, c.handleEnter()

		case c.mode == chooserList && k.String() == "e":
			if len(c.names) > 0 {
				c.startEdit(c.names[c.cursor])
			}

		case c.mode == chooserList && k.String() == "insert":
			c.startCreate()

		case c.mode == chooserList && (k.String() == "delete" || msg.Code == tea.KeyDelete):
			if len(c.names) > 0 {
				c.mode = chooserConfirmDelete
				name := c.names[c.cursor]
				d := NewDialog("Delete Logbook", "Delete \""+name+"\" and all its QSOs?",
					DangerOption("Delete", "delete"),
					Option{Label: "Cancel", Value: "cancel"},
				)
				c.dialog = &d
			}

		case c.mode == chooserList && (msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if c.cursor == 0 {
				c.cursor = len(c.names) - 1
			} else {
				c.cursor--
			}

		case c.mode == chooserList && (msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if c.cursor == len(c.names)-1 {
				c.cursor = 0
			} else {
				c.cursor++
			}

		case c.mode == chooserEdit || c.mode == chooserCreate:
			if cmd := c.station.HandleKey(msg); cmd != nil {
				return c, c.saveForm()
			}
		}
	}

	return c, nil
}

func (c *LogbookChooser) FooterText() string {
	switch c.mode {
	case chooserList:
		return "Enter to activate  e to edit  Ins to create  Del to delete  Esc to go back"
	case chooserEdit, chooserCreate:
		return "Ctrl+S to save  ↑↓/Tab to navigate  Esc to discard"
	case chooserConfirmDelete:
		return "←/→ choose  Enter confirm  Esc cancel"
	}
	return ""
}

func (c *LogbookChooser) View() tea.View {
	if c.done {
		return tea.NewView("")
	}

	switch c.mode {
	case chooserList:
		return tea.NewView(c.viewList())
	case chooserEdit, chooserCreate:
		return tea.NewView(c.viewForm())
	case chooserConfirmDelete:
		body := c.viewList()
		if c.dialog != nil {
			body = RenderDialogOverlay(body, *c.dialog, c.width, c.height)
		}
		return tea.NewView(body)
	}
	return tea.NewView("")
}

func (c *LogbookChooser) viewList() string {
	var b strings.Builder
	w := c.width
	if w < 40 {
		w = 80
	}
	h := c.height
	if h < 10 {
		h = 24
	}
	contentH := contentHeight(h)
	b.WriteString(menuTitle("Configuration — Logbooks", w))
	b.WriteString("\n\n")

	if len(c.names) == 0 {
		b.WriteString(menuLine("No logbooks configured.", w))
		b.WriteString("\n")
		return fillBody(b.String(), contentH)
	}

	for i, name := range c.names {
		lb := c.app.Config.Logbooks[name]
		marker := "  "
		if i == c.cursor {
			marker = CursorStyle.Render("> ")
		}
		active := "        "
		if name == c.app.Config.ActiveLogbook {
			active = "[Active]"
		}
		info := lb.Station.Callsign
		if lb.Station.Grid != "" {
			info += "  " + lb.Station.Grid
		}
		if info == "" {
			info = lb.Description
		}
		line := fmt.Sprintf("%s%s %s  %s", marker, active, name, info)
		// Selected row: wrap name in pink, rest in ValueStyle to keep
		// Surface background after CursorStyle's \x1b[0m reset.
		if i == c.cursor {
			line = CursorStyle.Render("> ") + CursorStyle.Render(fmt.Sprintf("%s %s  %s", active, name, info))
		}
		b.WriteString(menuLine(line, w))
		b.WriteString("\n")
	}

	return fillBody(b.String(), contentH)
}

func (c *LogbookChooser) viewForm() string {
	var b strings.Builder
	w := c.width
	if w < 40 {
		w = 80
	}
	h := c.height
	if h < 10 {
		h = 24
	}
	contentH := h - 4
	if contentH < 3 {
		contentH = 3
	}
	b.WriteString(menuTitle("Configuration — Edit Logbook", w))
	b.WriteString("\n\n")

	b.WriteString(c.station.View().Content)

	return fillBody(b.String(), contentH)
}

func (c *LogbookChooser) handleEnter() tea.Cmd {
	if len(c.names) > 0 {
		name := c.names[c.cursor]
		if name == c.app.Config.ActiveLogbook {
			c.toasts.Info("Logbook \"" + name + "\" is already active")
			return nil
		}
		if err := c.app.SwitchLogbook(name); err != nil {
			c.toasts.Error(err.Error())
			return nil
		}
		c.toasts.Success("Switched to logbook \"" + name + "\"")
		applog.Info("Logbook switched", "name", name)
		// Refresh names and [Active] indicator — stay in the menu.
		c.refreshNames()
	}
	return nil
}

func (c *LogbookChooser) refreshNames() {
	names := make([]string, 0, len(c.app.Config.Logbooks))
	for n := range c.app.Config.Logbooks {
		names = append(names, n)
	}
	slices.Sort(names)
	c.names = names
	// Keep cursor on the active logbook after refresh.
	for i, n := range names {
		if n == c.app.Config.ActiveLogbook {
			c.cursor = i
			return
		}
	}
	if c.cursor >= len(c.names) {
		c.cursor = len(c.names) - 1
	}
}

func (c *LogbookChooser) startCreate() {
	c.mode = chooserCreate
	c.station.SetValues("", "", "", "", "", "")
	c.station.Callsign.Focus()
	c.editing = ""
}

func (c *LogbookChooser) startEdit(name string) {
	lb := c.app.Config.Logbooks[name]
	c.mode = chooserEdit
	c.editing = name
	c.station.SetValues(lb.Station.Callsign, lb.Station.Operator, lb.Station.Grid, lb.Station.SOTARef, lb.Station.POTARef, lb.Station.WWFFRef)
	c.station.Callsign.Focus()
}

func (c *LogbookChooser) saveForm() tea.Cmd {
	cs, op, gr, sotaRef, potaRef, wwffRef := c.station.Values()

	if err := c.station.Validate(); err != nil {
		c.toasts.Error(err.Error())
		return nil
	}

	if c.mode == chooserCreate {
		name := cs
		if _, ok := c.app.Config.Logbooks[name]; ok {
			c.toasts.Error("Logbook already exists")
			return nil
		}
		c.app.Config.Logbooks[name] = config.Logbook{
			Description: "Created from TUI",
			Station: config.Station{
				Callsign: cs,
				Operator: op,
				Grid:     gr,
				SOTARef:  sotaRef,
				POTARef:  potaRef,
				WWFFRef:  wwffRef,
			},
		}
		c.app.Config.ActiveLogbook = name
		c.app.LogbookName = name
		lb := c.app.Config.Logbooks[name]
		c.app.Logbook = &lb

		c.names = append(c.names, name)
	} else {
		name := c.editing
		lb := c.app.Config.Logbooks[name]
		lb.Station.Callsign = cs
		lb.Station.Operator = op
		lb.Station.Grid = gr
		lb.Station.SOTARef = sotaRef
		lb.Station.POTARef = potaRef
		lb.Station.WWFFRef = wwffRef
		c.app.Config.Logbooks[name] = lb

		if name == c.app.LogbookName {
			c.app.Logbook = &lb
		}
	}

	c.mode = chooserList
	if err := config.Save(c.app.ConfigPath, c.app.Config); err != nil {
		c.toasts.Error("Config save failed: " + err.Error())
	} else {
		c.toasts.Success("Logbook saved")
		applog.Info("Logbook config saved")
	}
	return nil
}

func (c *LogbookChooser) viewConfirmDelete() string {
	name := c.names[c.cursor]
	var b strings.Builder
	w := c.width
	if w < 40 {
		w = 80
	}
	h := c.height
	if h < 10 {
		h = 24
	}
	contentH := h - 4
	if contentH < 3 {
		contentH = 3
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Delete logbook %q and ALL its QSOs?\n", name))
	b.WriteString("  This cannot be undone. (y/N)")
	return fillBody(b.String(), contentH)
}

func (c *LogbookChooser) deleteLogbook() tea.Cmd {
	if len(c.names) == 0 {
		return nil
	}
	name := c.names[c.cursor]

	if name == c.app.Config.ActiveLogbook {
		c.toasts.Error("Cannot delete active logbook. Switch to another first.")
		c.mode = chooserList
		return nil
	}

	if len(c.names) <= 1 {
		c.toasts.Error("Cannot delete the last logbook. At least one must remain.")
		c.mode = chooserList
		return nil
	}

	lb := c.app.Config.Logbooks[name]
	dbPath, _ := config.DBPath(name, &lb)

	delete(c.app.Config.Logbooks, name)

	for i, n := range c.names {
		if n == name {
			c.names = append(c.names[:i], c.names[i+1:]...)
			break
		}
	}
	if c.cursor >= len(c.names) {
		c.cursor = len(c.names) - 1
	}

	c.mode = chooserList
	if err := config.Save(c.app.ConfigPath, c.app.Config); err != nil {
		c.toasts.Error("Config save failed: " + err.Error())
	} else {
		go func() { os.Remove(dbPath) }()
		c.toasts.Success("Logbook " + name + " deleted")
		applog.Info("Logbook deleted", "name", name)
	}
	return nil
}
