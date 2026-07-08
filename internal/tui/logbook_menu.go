package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/aprs"
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

	// Viewport for scrolling form content on small terminals.
	vp              viewport.Model
	lastFormContent string
	lastListContent string

	// APRS async state.
	aprsTesting bool
	aprsStatus  string
}

// Wavelog async message types
type wlUpdateMsg struct {
	stations []wavelog.StationProfile
	err      error
}
type wlTestMsg struct {
	err error
}

// APRS async message type.
type aprsTestMsg struct {
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
		vp:      viewport.New(viewport.WithWidth(80), viewport.WithHeight(20)),
	}
}

func (c *LogbookChooser) Init() tea.Cmd {
	c.vp.FillHeight = true
	return nil
}

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
			c.toasts.Error("Wavelog: " + msg.err.Error())
		} else {
			c.wlStatus = "OK — Wavelog reachable"
			c.toasts.Success("Wavelog: connection verified")
		}
		c.scrollViewportToEnd()

	case aprsTestMsg:
		c.aprsTesting = false
		if msg.err != nil {
			c.aprsStatus = msg.err.Error()
			c.toasts.Error("APRS: " + msg.err.Error())
		} else {
			c.aprsStatus = "OK — connection verified"
			c.toasts.Success("APRS: connection verified")
		}
		c.scrollViewportToEnd()

	case tea.PasteMsg:
		// Forward clipboard paste to the focused field in the station
		// edit form (callsign, grid, Wavelog URL/key, etc.).
		if c.mode == chooserEdit || c.mode == chooserCreate {
			c.station.HandlePaste(msg.Content)
		}
		return c, nil

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
				d, ok := updated.(DialogModel)
				if !ok {
					return c, nil
				}
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

		case c.mode == chooserList && (k.String() == "pgup" || k.String() == "pgdown" || k.String() == "home" || k.String() == "end"):
			c.vp, _ = c.vp.Update(msg)
			return c, nil

		case c.mode == chooserList && (msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if c.cursor == 0 {
				c.cursor = len(c.names) - 1
			} else {
				c.cursor--
			}
			scrollVpToLine(&c.vp, c.cursor)

		case c.mode == chooserList && (msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if c.cursor == len(c.names)-1 {
				c.cursor = 0
			} else {
				c.cursor++
			}
			scrollVpToLine(&c.vp, c.cursor)

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
				case aprsTestAction:
					return c, c.testAPRSConnection()
				case wlCycleStation:
					if len(c.wlStations) > 0 {
						c.wlStationIdx = (c.wlStationIdx + 1) % len(c.wlStations)
						c.updateStationIDField()
					}
					return c, c.testWavelogConnection()
				case scrollFormToEnd:
					c.scrollViewportToEnd()
					return c, nil
				}
				return c, nil
			}
			// Key not handled by station form — forward to viewport for scrolling,
			// then clamp to prevent scrolling past the content end.
			var cmd tea.Cmd
			c.vp, cmd = c.vp.Update(msg)
			c.autoScrollViewport()
			return c, cmd
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

	return renderScrollableMenu("Configuration \u2014 Logbooks", b.String(), &c.vp, &c.lastListContent, w, h)
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

	// APRS status line below the form.
	if c.station.AprsEnabled && c.aprsStatus != "" {
		b.WriteString("\n    ")
		if strings.HasPrefix(c.aprsStatus, "OK") {
			b.WriteString(SuccessStyle.Render(c.aprsStatus))
		} else if c.aprsTesting {
			b.WriteString(DimStyle.Render(c.aprsStatus))
		} else {
			b.WriteString(ErrorStyle.Render(c.aprsStatus))
		}
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
	contentH := contentHeight(h)
	if contentH < 8 {
		contentH = 8
	}
	vpH := contentH - 3
	if vpH < 4 {
		vpH = 4
	}
	c.vp.SetWidth(vpW)
	c.vp.SetHeight(vpH)
	bodyStr := b.String()
	if c.vp.TotalLineCount() == 0 || bodyStr != c.lastFormContent {
		c.vp.SetContent(bodyStr)
		c.lastFormContent = bodyStr
	}
	// Prevent scrolling past the end: if past bottom, snap back.
	if c.vp.PastBottom() {
		c.autoScrollViewport()
	}

	header := S.Title.Width(boxW).Render("Configuration \u2014 Logbooks \u2014 Edit Logbook")
	vpContent := c.vp.View()
	hint := scrollHint(c.vp)
	hintLine := DimStyle.Width(vpW).Render(hint)
	if hintLine == "" {
		hintLine = strings.Repeat(" ", vpW)
	}
	vpContent = lipgloss.JoinVertical(lipgloss.Left, vpContent, hintLine)
	box := menuBoxStyle.Width(boxW).Render(vpContent)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", box)
}

// autoScrollViewport adjusts the viewport Y offset to keep the currently
// focused form field visible, using the StationForm's ScrollFraction hint.
func (c *LogbookChooser) autoScrollViewport() {
	total := c.vp.TotalLineCount()
	visible := c.vp.VisibleLineCount()
	if total <= visible {
		c.vp.SetYOffset(0)
		return
	}
	frac := c.station.ScrollFraction()
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := int(float64(maxOffset) * frac)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	c.vp.SetYOffset(offset)
}

// scrollViewportToEnd scrolls the viewport to the last visible page so the
// user can see the APRS/Wavelog test status lines without manual scrolling.
func (c *LogbookChooser) scrollViewportToEnd() {
	total := c.vp.TotalLineCount()
	visible := c.vp.VisibleLineCount()
	if total <= visible {
		c.vp.SetYOffset(0)
		return
	}
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	c.vp.SetYOffset(maxOffset)
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
	c.station.GPSGrid = lb.Station.GPSGrid
	c.station.SetOperators(config.OperatorSlice(c.app.Config))
	c.station.SetWavelogValues(lb.Wavelog)
	c.station.SetAPRSValues(lb.APRS)
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
	// Build APRS config from form.
	aprs := c.station.APRSValues()
	if aprs != nil && !aprs.Enabled {
		aprs = nil // store nil when disabled
	}
	// Validate APRS fields when enabled.
	if aprs != nil {
		if aprs.Callsign == "" {
			c.toasts.Warn("APRS: callsign is required when APRS is enabled")
			return nil
		}
		if aprs.IntervalMin < 1 {
			c.toasts.Warn("APRS: interval must be at least 1 minute")
			return nil
		}
		if aprs.Symbol == "" {
			aprs.Symbol = "/-" // default if empty
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
				GPSGrid:    c.station.GPSGrid,
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
			APRS:    aprs,
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
	lb.Station.GPSGrid = c.station.GPSGrid
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
	lb.APRS = aprs
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
	// Restart APRS if config changed (debounced).
	c.app.ScheduleAPRSRestart()
	c.app.RequestAPRSRefresh()
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
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			applog.Warn("Failed to remove logbook database", "path", dbPath, "error", err)
			c.toasts.Success("Logbook " + displayName + " deleted (DB cleanup failed — " + err.Error() + ")")
		} else {
			c.toasts.Success("Logbook " + displayName + " deleted")
		}
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
	u := strings.TrimRight(strings.TrimSpace(c.station.WlURL.Value()), "/")
	k := strings.TrimSpace(c.station.WlKey.Value())

	// Validate required fields before testing.
	if u == "" {
		c.wlStatus = "API URL is required"
		c.toasts.Warn("Wavelog: API URL is required")
		c.scrollViewportToEnd()
		return nil
	}
	if k == "" {
		c.wlStatus = "API Key is required"
		c.toasts.Warn("Wavelog: API Key is required")
		c.scrollViewportToEnd()
		return nil
	}

	c.wlTesting = true
	c.wlStatus = "Testing…"
	c.scrollViewportToEnd()
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

// testAPRSConnection tests APRS connectivity using the global
// integration config and the logbook config for callsign/passcode.
func (c *LogbookChooser) testAPRSConnection() tea.Cmd {
	aprsGlobal := c.app.Config.Integrations.APRS
	if !aprsGlobal.Enabled {
		c.aprsStatus = "APRS not configured in Integrations"
		c.toasts.Warn("APRS: enable and configure APRS in Integrations first")
		c.scrollViewportToEnd()
		return nil
	}

	cfg := c.station.APRSValues()
	call := cfg.Callsign
	pass := cfg.Passcode

	// Validate required fields before testing.
	if pass == "" {
		c.aprsStatus = "Passcode is required"
		c.toasts.Warn("APRS: passcode is required")
		c.scrollViewportToEnd()
		return nil
	}
	if call == "" {
		c.aprsStatus = "Callsign is required"
		c.toasts.Warn("APRS: callsign is required")
		c.scrollViewportToEnd()
		return nil
	}

	switch aprsGlobal.Service {
	case "kiss":
		prt := aprsGlobal.Port
		baud := aprsGlobal.BaudRate
		if prt == "" || baud == 0 {
			c.aprsStatus = "KISS port/baud not configured in Integrations"
			c.toasts.Warn("APRS: configure KISS port and baud in Integrations first")
			c.scrollViewportToEnd()
			return nil
		}
		dataBits := aprsGlobal.DataBits
		if dataBits < 5 || dataBits > 8 {
			dataBits = 8
		}
		par := parityFromString(aprsGlobal.Parity)
		stop := stopBitsFromString(aprsGlobal.StopBits)
		c.aprsTesting = true
		c.aprsStatus = "Testing KISS…"
		c.scrollViewportToEnd()
		return func() tea.Msg {
			if err := testKISSPort(prt, baud, dataBits, par, stop, aprsGlobal.DTR, aprsGlobal.RTS); err != nil {
				return aprsTestMsg{err: err}
			}
			return aprsTestMsg{}
		}
	case "kiss_server":
		host := aprsGlobal.KISSServerHost
		if host == "" {
			host = "127.0.0.1"
		}
		port := aprsGlobal.KISSServerPort
		if port == "" {
			port = "8001"
		}
		addr := host + ":" + port
		c.aprsTesting = true
		c.aprsStatus = "Testing KISS server…"
		c.scrollViewportToEnd()
		return func() tea.Msg {
			if err := aprs.TestKISSServerConnection(addr); err != nil {
				return aprsTestMsg{err: err}
			}
			return aprsTestMsg{}
		}
	default: // "aprs_is" or empty
		srv := aprsGlobal.Server
		if srv == "" {
			srv = "euro.aprs2.net:14580"
		}
		c.aprsTesting = true
		c.aprsStatus = "Testing…"
		c.scrollViewportToEnd()
		return func() tea.Msg {
			if err := aprs.TestConnection(srv, call, pass); err != nil {
				return aprsTestMsg{err: err}
			}
			return aprsTestMsg{}
		}
	}
}
