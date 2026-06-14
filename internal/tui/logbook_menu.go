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
	"github.com/szporwolik/cqops/internal/wavelog"
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

	// Wavelog async state
	wlUpdating   bool
	wlTesting    bool
	wlStatus     string
	wlStations   []wavelog.StationProfile
	wlStationIdx int // index into wlStations, -1 if none
	wlStationID  string
}

// Wavelog async message types
type wlUpdateMsg struct {
	stations []wavelog.StationProfile
	err      error
}
type wlTestMsg struct {
	err error
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
		if n == a.Config.State.ActiveLogbook {
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

	case wlUpdateMsg:
		c.wlUpdating = false
		if msg.err != nil {
			c.wlStatus = msg.err.Error()
			c.wlStations = nil
			c.wlStationIdx = -1
		} else {
			c.wlStations = msg.stations
			c.wlStationIdx = 0
			if c.wlStationID != "" {
				for i, s := range c.wlStations {
					if s.ID == c.wlStationID {
						c.wlStationIdx = i
						break
					}
				}
			}
			c.updateStationIDField()
			c.wlStatus = fmt.Sprintf("OK — %d stations loaded — Space over Station ID to cycle", len(msg.stations))
		}

	case wlTestMsg:
		c.wlTesting = false
		if msg.err != nil {
			c.wlStatus = msg.err.Error()
		} else {
			c.wlStatus = "OK — Wavelog reachable"
		}

	case tea.KeyPressMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if c.mode == chooserList {
				c.done = true
				return c, nil
			}
			c.station.BlurAll()
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
				// Execute the command to inspect the message. Save (enterOnLastFieldMsg)
				// triggers saveForm; WL button actions are handled below.
				msg := cmd()
				switch msg.(type) {
				case enterOnLastFieldMsg:
					return c, c.saveForm()
				case wlUpdateAction:
					return c, c.fetchWavelogStations()
				case wlTestAction:
					return c, c.testWavelogConnection()
				case wlCycleStation:
					if len(c.wlStations) > 0 {
						c.wlStationIdx = (c.wlStationIdx + 1) % len(c.wlStations)
						c.updateStationIDField()
					}
					return c, c.testWavelogConnection()
				}
				return c, nil
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
		return "Ctrl+S to save  ↑↓/Tab to navigate  Space cycle station  Esc to discard"
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
		if name == c.app.Config.State.ActiveLogbook {
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

	// Wavelog status and station info (below the form buttons).
	if c.station.WlEnabled {
		// Status line
		if c.wlStatus != "" {
			b.WriteString("\n    ")
			if strings.HasPrefix(c.wlStatus, "OK") {
				b.WriteString(SuccessStyle.Render(c.wlStatus))
			} else if c.wlUpdating || c.wlTesting {
				b.WriteString(SubtleStyle.Render(c.wlStatus))
			} else {
				b.WriteString(ErrorStyle.Render(c.wlStatus))
			}
		}
	}

	return fillBody(b.String(), contentH)
}

// logbookSwitchedMsg is sent when the user switches active logbook via Enter.
type logbookSwitchedMsg struct{}

func (c *LogbookChooser) handleEnter() tea.Cmd {
	if len(c.names) > 0 {
		name := c.names[c.cursor]
		if name == c.app.Config.State.ActiveLogbook {
			c.toasts.Info("Logbook \"" + name + "\" is already active")
			return nil
		}
		if err := c.app.SwitchLogbook(name); err != nil {
			c.toasts.Error("Switch to " + name + " failed: " + err.Error())
			return nil
		}
		c.toasts.Success("Switched to logbook \"" + name + "\"")
		applog.Info("Logbook switched", "name", name)
		c.refreshNames()
		return func() tea.Msg { return logbookSwitchedMsg{} }
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
		if n == c.app.Config.State.ActiveLogbook {
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
	c.station.BlurAll()
	c.station.Callsign.Focus()
	c.editing = ""
}

func (c *LogbookChooser) startEdit(name string) {
	lb := c.app.Config.Logbooks[name]
	c.mode = chooserEdit
	c.editing = name
	c.station.SetValues(lb.Station.Callsign, lb.Station.Operator, lb.Station.Grid, lb.Station.SOTARef, lb.Station.POTARef, lb.Station.WWFFRef)
	c.station.SetWavelogValues(lb.Wavelog)
	c.wlStatus = ""
	c.wlStations = nil
	c.wlStationIdx = -1
	if lb.Wavelog != nil {
		c.wlStationID = lb.Wavelog.StationProfileID
	}
	c.station.BlurAll()
	c.station.Callsign.Focus()
}

func (c *LogbookChooser) saveForm() tea.Cmd {
	cs, op, gr, sotaRef, potaRef, wwffRef, wlEnabled, wlURL, wlKey, wlStationID := c.station.Values()

	if err := c.station.Validate(); err != nil {
		c.toasts.Error(err.Error())
		return nil
	}

	// Build Wavelog config from form.
	var wl *config.WavelogConfig
	if wlEnabled && wlURL != "" && wlKey != "" {
		wl = &config.WavelogConfig{
			Enabled:          wlEnabled,
			URL:              wlURL,
			APIKey:           wlKey,
			StationProfileID: wlStationID,
		}
	}

	var savedName string
	if c.mode == chooserCreate {
		name := cs
		if _, ok := c.app.Config.Logbooks[name]; ok {
			c.toasts.Error("Logbook " + name + " already exists")
			return nil
		}
		prevRigName := c.app.Logbook.Station.RigName
		c.app.Config.Logbooks[name] = config.Logbook{
			Description: "Created from TUI",
			Station: config.Station{
				Callsign: cs,
				Operator: op,
				Grid:     gr,
				SOTARef:  sotaRef,
				POTARef:  potaRef,
				WWFFRef:  wwffRef,
				RigName:  prevRigName,
			},
			Wavelog: wl,
		}
		c.app.Config.State.ActiveLogbook = name
		c.app.LogbookName = name
		lb := c.app.Config.Logbooks[name]
		c.app.Logbook = &lb

		c.names = append(c.names, name)
		savedName = name
	} else {
		name := c.editing
		lb := c.app.Config.Logbooks[name]
		lb.Station.Callsign = cs
		lb.Station.Operator = op
		lb.Station.Grid = gr
		lb.Station.SOTARef = sotaRef
		lb.Station.POTARef = potaRef
		lb.Station.WWFFRef = wwffRef
		lb.Wavelog = wl
		c.app.Config.Logbooks[name] = lb

		if name == c.app.LogbookName {
			c.app.Logbook = &lb
		}
		savedName = name
	}

	c.mode = chooserList
	c.station.BlurAll()
	if err := config.Save(c.app.ConfigPath, c.app.Config); err != nil {
		c.toasts.Error("Save " + savedName + " failed: " + err.Error())
		return nil
	}
	c.toasts.Success("Logbook " + savedName + " saved")
	applog.Info("Logbook saved", "name", savedName)
	return nil
}

// updateStationIDField sets the Station ID text field to show the currently
// selected station's ID, callsign, name, and locator.
func (c *LogbookChooser) updateStationIDField() {
	if c.wlStationIdx >= 0 && c.wlStationIdx < len(c.wlStations) {
		s := c.wlStations[c.wlStationIdx]
		c.station.WlStationID.SetValue(fmt.Sprintf("%s — %s (%s) %s", s.ID, s.Callsign, s.Name, s.Gridsquare))
		c.wlStationID = s.ID
	}
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

	if name == c.app.Config.State.ActiveLogbook {
		c.toasts.Error("Cannot delete " + name + " — it is the active logbook. Switch to another first.")
		c.mode = chooserList
		return nil
	}

	if len(c.names) <= 1 {
		c.toasts.Error("Cannot delete " + name + " — at least one logbook must remain.")
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
		c.toasts.Error("Delete " + name + " failed: " + err.Error())
	} else {
		go func() { os.Remove(dbPath) }()
		c.toasts.Success("Logbook " + name + " deleted")
		applog.Info("Logbook deleted", "name", name)
	}
	return nil
}

// fetchWavelogStations fetches station profiles from the Wavelog API.
func (c *LogbookChooser) fetchWavelogStations() tea.Cmd {
	c.wlUpdating = true
	c.wlStatus = "Fetching stations…"
	u := strings.TrimRight(strings.TrimSpace(c.station.WlURL.Value()), "/")
	k := strings.TrimSpace(c.station.WlKey.Value())
	return func() tea.Msg {
		stations, err := wavelog.FetchStations(u, k)
		return wlUpdateMsg{stations: stations, err: err}
	}
}

// testWavelogConnection tests Wavelog connectivity and station validity.
func (c *LogbookChooser) testWavelogConnection() tea.Cmd {
	c.wlTesting = true
	c.wlStatus = "Testing…"
	u := strings.TrimRight(strings.TrimSpace(c.station.WlURL.Value()), "/")
	k := strings.TrimSpace(c.station.WlKey.Value())
	var sid string
	if c.wlStationIdx >= 0 && c.wlStationIdx < len(c.wlStations) {
		sid = c.wlStations[c.wlStationIdx].ID
	}
	return func() tea.Msg {
		if err := wavelog.TestConnection(u, k); err != nil {
			return wlTestMsg{err: err}
		}
		if sid != "" {
			if err := wavelog.TestStation(u, k, sid); err != nil {
				return wlTestMsg{err: err}
			}
		}
		return wlTestMsg{}
	}
}
