package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/log"
)

type chooserMode int

const (
	chooserList  chooserMode = iota
	chooserEdit
	chooserCreate
)

type LogbookChooser struct {
	app     *app.App
	mode    chooserMode
	names   []string
	cursor  int
	station *StationForm
	editing string
	toasts  *ToastQueue
	width   int
	height  int
	done    bool
}

func NewLogbookChooser(a *app.App, tq *ToastQueue) *LogbookChooser {
	names := make([]string, 0, len(a.Config.Logbooks))
	for name := range a.Config.Logbooks {
		names = append(names, name)
	}

	return &LogbookChooser{
		app:     a,
		mode:    chooserList,
		names:   names,
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

	case tea.KeyMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if c.mode == chooserList {
				c.done = true
				return c, nil
			}
			c.mode = chooserList

		case c.mode == chooserList && k.String() == "enter":
			return c, c.handleEnter()

		case c.mode == chooserList && k.String() == "c":
			c.startCreate()

		case c.mode == chooserList && k.String() == "e":
			if len(c.names) > 0 {
				c.startEdit(c.names[c.cursor])
			}

		case c.mode == chooserList && (msg.Type == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if c.cursor > 0 {
				c.cursor--
			}

		case c.mode == chooserList && (msg.Type == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if c.cursor < len(c.names)-1 {
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
		return "Enter to switch  e to edit  c to create  Esc to go back"
	case chooserEdit, chooserCreate:
		return "Ctrl+S to save  Tab/↓/↑ to navigate  Esc to discard"
	}
	return ""
}

func (c *LogbookChooser) View() string {
	if c.done {
		return ""
	}

	switch c.mode {
	case chooserList:
		return c.viewList()
	case chooserEdit, chooserCreate:
		return c.viewForm()
	}
	return ""
}

func (c *LogbookChooser) viewList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Configuration — Logbooks"))
	b.WriteString("\n\n")

	if len(c.names) == 0 {
		b.WriteString("No logbooks configured.\n\n")
		return b.String()
	}

	for i, name := range c.names {
		lb := c.app.Config.Logbooks[name]
		marker := "  "
		if i == c.cursor {
			marker = cursorStyle.Render("> ")
		}
		active := " "
		if name == c.app.Config.ActiveLogbook {
			active = "*"
		}
		info := lb.Station.Callsign
		if lb.Station.Grid != "" {
			info += "  " + lb.Station.Grid
		}
		if info == "" {
			info = lb.Description
		}
		b.WriteString(fmt.Sprintf("%s%s %s  %s\n", marker, active, name, info))
	}

	return b.String()
}

func (c *LogbookChooser) viewForm() string {
	var b strings.Builder
	if c.mode == chooserEdit {
		b.WriteString(titleStyle.Render("Configuration — Edit " + c.editing))
	} else {
		b.WriteString(titleStyle.Render("Configuration — Create Logbook"))
	}
	b.WriteString("\n\n")

	b.WriteString(c.station.View())

	return b.String()
}

func (c *LogbookChooser) handleEnter() tea.Cmd {
	if len(c.names) > 0 {
		name := c.names[c.cursor]
		if name == c.app.Config.ActiveLogbook {
			c.done = true
			return nil
		}
		if err := c.app.SwitchLogbook(name); err != nil {
			c.toasts.Error(err.Error())
			return nil
		}
		c.done = true
	}
	return nil
}

func (c *LogbookChooser) startCreate() {
	c.mode = chooserCreate
	c.station.SetValues("", "", "")
	c.station.Callsign.Focus()
	c.editing = ""
}

func (c *LogbookChooser) startEdit(name string) {
	lb := c.app.Config.Logbooks[name]
	c.mode = chooserEdit
	c.editing = name
	c.station.SetValues(lb.Station.Callsign, lb.Station.Operator, lb.Station.Grid)
	c.station.Callsign.Focus()
}

func (c *LogbookChooser) saveForm() tea.Cmd {
	cs, op, gr := c.station.Values()

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
		log.Info("Logbook config saved")
	}
	return nil
}
