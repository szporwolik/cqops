package tui

import (
	"fmt"
	"os"
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
	names := config.SortedLogbookIDs(a.Config)

	// Start cursor on the active logbook.
	cursor := 0
	for i, id := range names {
		if id == a.Config.State.ActiveLogbook {
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
			return c, nil

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
			if len(c.names) > 0 {
				c.startEdit(c.names[c.cursor])
			}

		case c.mode == chooserList && (k.String() == " " || k.Code == ' ' || k.String() == "a"):
			if len(c.names) > 0 {
				return c, c.handleEnter()
			}

		case c.mode == chooserList && k.String() == "insert":
			c.startCreate()

		case c.mode == chooserList && (k.String() == "delete" || msg.Code == tea.KeyDelete):
			if len(c.names) > 0 {
				c.mode = chooserConfirmDelete
				id := c.names[c.cursor]
				lb := c.app.Config.Logbooks[id]
				displayName := config.LogbookDisplayName(&lb)
				d := NewDialog("Delete Logbook", "Delete \""+displayName+"\" and all its QSOs?",
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
	if contentH < 3 {
		contentH = 3
	}

	if len(c.names) == 0 {
		b.WriteString("No logbooks configured.\n")
	} else {
		contentW := w - 8
		if contentW > partnerMapMaxW-8 {
			contentW = partnerMapMaxW - 8
		}
		if contentW < 20 {
			contentW = 20
		}

		for i, id := range c.names {
			lb := c.app.Config.Logbooks[id]
			dn := config.LogbookDisplayName(&lb)
			call := lb.Station.Callsign
			grid := lb.Station.Grid
			if grid == "" {
				grid = "—"
			}

			// Truncate/pad raw values to fixed column widths before styling.
			nameVal := padOrTrunc(dn, 24)
			callVal := padOrTrunc(call, 12)

			prefix := "  "
			activeBadge := padOrTrunc("[      ]", 10)
			if id == c.app.Config.State.ActiveLogbook {
				activeBadge = S.ToastSuccess.Render(padOrTrunc("[Active]", 10))
			}

			if i == c.cursor {
				prefix = S.FormPrefixOn.Render("> ")
				nameVal = CursorStyle.Render(nameVal)
				callVal = CursorStyle.Render(callVal)
				grid = CursorStyle.Render(grid)
			}

			line := prefix + activeBadge + nameVal + callVal + grid
			b.WriteString(padOrTrunc(line, contentW))
			if i < len(c.names)-1 {
				b.WriteString("\n")
			}
		}
	}

	body := drawMenuWithHeader("Configuration \u2014 Logbooks", b.String(), w)
	return fillBody(body, contentH)
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
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	c.station.width = w - 6 // account for menu box border + padding
	b.WriteString(c.station.View().Content)

	// Wavelog status line below the form.
	if c.station.WlEnabled && c.wlStatus != "" {
		b.WriteString("\n    ")
		if strings.HasPrefix(c.wlStatus, "OK") {
			b.WriteString(SuccessStyle.Render(c.wlStatus))
		} else if c.wlUpdating || c.wlTesting {
			b.WriteString(DimStyle.Render(c.wlStatus))
		} else {
			b.WriteString(ErrorStyle.Render(c.wlStatus))
		}
	}

	body := drawMenuWithHeader("Configuration \u2014 Logbooks \u2014 Edit Logbook", b.String(), w)
	return fillBody(body, contentH)
}

// logbookSwitchedMsg is sent when the user switches active logbook via Enter.
type logbookSwitchedMsg struct{}

func (c *LogbookChooser) handleEnter() tea.Cmd {
	if len(c.names) > 0 {
		id := c.names[c.cursor]
		lb := c.app.Config.Logbooks[id]
		displayName := config.LogbookDisplayName(&lb)
		if id == c.app.Config.State.ActiveLogbook {
			c.toasts.Info("Logbook \"" + displayName + "\" is already active")
			return nil
		}
		if err := c.app.SwitchLogbook(id); err != nil {
			c.toasts.Error("Switch to " + displayName + " failed: " + err.Error())
			return nil
		}
		c.toasts.Success("Switched to logbook \"" + displayName + "\"")
		applog.Info("Logbook switched", "name", displayName)
		c.refreshNames()
		return func() tea.Msg { return logbookSwitchedMsg{} }
	}
	return nil
}

func (c *LogbookChooser) refreshNames() {
	c.names = config.SortedLogbookIDs(c.app.Config)
	// Keep cursor on the active logbook after refresh.
	for i, id := range c.names {
		if id == c.app.Config.State.ActiveLogbook {
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
	c.station.SetValues("", "", "", "", "", "", "", 1, 0, 0, 0, "", "", "EU")
	c.station.SetOperators(config.OperatorSlice(c.app.Config))
	c.station.BlurAll()
	c.station.Name.Focus()
	c.editing = ""
}

func (c *LogbookChooser) startEdit(id string) {
	lb := c.app.Config.Logbooks[id]
	c.mode = chooserEdit
	c.editing = id
	// Resolve active operator to callsign for the form selector.
	opCallsign := ""
	if lb.ActiveOperator != "" {
		if op, ok := c.app.Config.Operators[lb.ActiveOperator]; ok {
			opCallsign = op.Callsign
		}
	}
	c.station.SetValues(lb.Name, lb.Station.Callsign, opCallsign, lb.Station.Grid, lb.Station.SOTARef, lb.Station.POTARef, lb.Station.WWFFRef, lb.Station.IARURegion, lb.Station.CQZone, lb.Station.ITUZone, lb.Station.DXCC, lb.Station.SIG, lb.Station.SIGInfo, lb.Station.Continent)
	c.station.SetOperators(config.OperatorSlice(c.app.Config))
	c.station.SetWavelogValues(lb.Wavelog)
	c.wlStatus = ""
	c.wlStations = nil
	c.wlStationIdx = -1
	if lb.Wavelog != nil {
		c.wlStationID = lb.Wavelog.StationProfileID
	}
	c.station.BlurAll()
	c.station.Name.Focus()
}

func (c *LogbookChooser) saveForm() tea.Cmd {
	nm, cs, op, gr, sotaRef, potaRef, wwffRef, wlEnabled, wlURL, wlKey, wlStationID, iaruRegion, cqZone, ituZone, dxcc, sig, sigInfo, continent := c.station.Values()

	// Resolve operator callsign to operator ID for ActiveOperator.
	var activeOpID string
	if op != "" {
		if oid, _, found := config.FindOperatorByCallsign(c.app.Config, op); found {
			activeOpID = oid
		}
	}

	if err := c.station.Validate(); err != nil {
		c.toasts.Warn(err.Error())
		return nil
	}

	if nm == "" {
		c.toasts.Warn("Station name cannot be empty")
		return nil
	}

	// Build Wavelog config from form.
	var wl *config.WavelogConfig
	if wlEnabled {
		if wlStationID == "" {
			c.toasts.Warn("Wavelog enabled but Station ID not set — press Update to fetch")
			return nil
		}
		if wlURL != "" && wlKey != "" {
			wl = &config.WavelogConfig{
				Enabled:          wlEnabled,
				URL:              wlURL,
				APIKey:           wlKey,
				StationProfileID: wlStationID,
			}
		}
	}

	var savedName string
	if c.mode == chooserCreate {
		// Check for duplicate by callsign.
		if _, _, found := config.FindLogbookByCallsign(c.app.Config, cs); found {
			c.toasts.Warn("Logbook with callsign " + cs + " already exists")
			return nil
		}
		id := config.NewID(cs)
		prevRigName := c.app.Logbook.Station.RigName
		c.app.Config.Logbooks[id] = config.Logbook{
			ID:             id,
			Name:           nm,
			ActiveOperator: activeOpID,
			Station: config.Station{
				Callsign: cs,

				Grid:       gr,
				SOTARef:    sotaRef,
				POTARef:    potaRef,
				WWFFRef:    wwffRef,
				RigName:    prevRigName,
				IARURegion: iaruRegion,
				CQZone:     cqZone,
				ITUZone:    ituZone,
				DXCC:       dxcc,
				SIG:        sig,
				SIGInfo:    sigInfo,
				Continent:  continent,
			},
			Wavelog: wl,
		}
		c.app.Config.State.ActiveLogbook = id
		c.app.LogbookName = id
		lb := c.app.Config.Logbooks[id]
		c.app.Logbook = &lb

		c.names = append(c.names, id)
		savedName = cs

		// Open the new logbook's database and switch to it.
		if err := c.app.SwitchLogbook(id); err != nil {
			c.toasts.Error("Failed to open logbook: " + err.Error())
			return nil
		}

		c.mode = chooserList
		c.station.BlurAll()
		if err := config.Save(c.app.ConfigPath, c.app.Config); err != nil {
			c.toasts.Error("Save " + savedName + " failed: " + err.Error())
			return nil
		}
		c.toasts.Success("Logbook " + savedName + " created")
		applog.Info("Logbook created", "name", savedName)
		return func() tea.Msg { return logbookSwitchedMsg{} }
	}

	// Edit existing logbook.
	id := c.editing
	lb := c.app.Config.Logbooks[id]
	lb.Name = nm
	lb.Station.Callsign = cs
	lb.ActiveOperator = activeOpID
	lb.Station.Grid = gr
	lb.Station.SOTARef = sotaRef
	lb.Station.POTARef = potaRef
	lb.Station.WWFFRef = wwffRef
	lb.Station.IARURegion = iaruRegion
	lb.Station.CQZone = cqZone
	lb.Station.ITUZone = ituZone
	lb.Station.DXCC = dxcc
	lb.Station.SIG = sig
	lb.Station.SIGInfo = sigInfo
	lb.Station.Continent = continent
	lb.Wavelog = wl
	c.app.Config.Logbooks[id] = lb

	if id == c.app.LogbookName {
		c.app.Logbook = &lb
	}
	savedName = cs

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

func (c *LogbookChooser) deleteLogbook() tea.Cmd {
	if len(c.names) == 0 {
		return nil
	}
	id := c.names[c.cursor]
	lb := c.app.Config.Logbooks[id]
	displayName := config.LogbookDisplayName(&lb)

	if id == c.app.Config.State.ActiveLogbook {
		c.toasts.Error("Cannot delete " + displayName + " — it is the active logbook. Switch to another first.")
		c.mode = chooserList
		return nil
	}

	if len(c.names) <= 1 {
		c.toasts.Error("Cannot delete " + displayName + " — at least one logbook must remain.")
		c.mode = chooserList
		return nil
	}

	dbPath, _ := config.DBPath(id, &lb)

	delete(c.app.Config.Logbooks, id)

	for i, n := range c.names {
		if n == id {
			c.names = append(c.names[:i], c.names[i+1:]...)
			break
		}
	}
	if c.cursor >= len(c.names) {
		c.cursor = len(c.names) - 1
	}

	c.mode = chooserList
	if err := config.Save(c.app.ConfigPath, c.app.Config); err != nil {
		c.toasts.Error("Delete " + displayName + " failed: " + err.Error())
	} else {
		go func() {
			if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
				applog.Warn("Failed to remove logbook database", "path", dbPath, "error", err)
			}
		}()
		c.toasts.Success("Logbook " + displayName + " deleted")
		applog.Info("Logbook deleted", "name", displayName)
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
