package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/version"
	"github.com/szporwolik/cqops/internal/wavelog"
)

type wizardStep int

const (
	stepStation wizardStep = iota
	stepRig
	stepSummary
	stepCount // sentinel
)

type Wizard struct {
	App       *app.App
	step      wizardStep
	station   *StationForm
	rigForm   *RigForm
	toasts    *ToastQueue
	width     int
	height    int
	Completed bool // true only when full wizard finished
	Offline   bool // when true, skip all network-dependent operations

	// Cached form box style — rebuilt only when width changes.
	cachedFormBox  lipgloss.Style
	cachedFormBoxW int

	// Wavelog async state (for wizard step 1 buttons)
	wlStations   []wavelog.StationProfile
	wlStationIdx int
	wlStation    *wavelog.Station // fetched full profile, applied on save

	// fm owns wizard-level navigation: the Save & Next / Save & Start button
	// and the focused row of the current step's form.
	// Enter works throughout the forms, but a visible button with a (Space)
	// hint is a much clearer affordance for new users.
	fm menuFocus
}

func NewWizard(a *app.App) *Wizard {
	applog.Info("Wizard started — first-run setup")
	sf := NewStationForm("", "", "")
	sf.HideGPSGrid = true  // GPS Grid is not relevant during first-run setup
	sf.HideOperator = true // operators don't exist yet during first-run wizard
	sf.HideIARU = true     // show only the Continent selector
	sf.Advanced = false    // keep the wizard simple — optional fields are hidden (Ctrl+A reveals them)
	return &Wizard{
		App:     a,
		step:    stepStation,
		station: sf,
		rigForm: NewRigForm("Xiegu G90 (optional)", "HWEF 20.5 (optional)", "20"),
		toasts:  NewToastQueue(),
	}
}

// focusableRows implementation for the shared menuFocus engine — the rows
// belong to the current step's form.
func (w *Wizard) rowCount() int {
	if w.step == stepStation {
		return w.station.rowCount()
	}
	return int(rigFieldEnd)
}
func (w *Wizard) rowVisible(i int) bool {
	if w.step == stepStation {
		return w.station.rowVisible(i)
	}
	return w.rigForm.visible(rigFormField(i))
}
func (w *Wizard) blurAll() {
	if w.step == stepStation {
		w.station.BlurAll()
		return
	}
	w.rigForm.blurAll()
	w.rigForm.focus = -1 // leave the form: no field may stay highlighted
}
func (w *Wizard) focusRow(i int) tea.Cmd {
	if w.step == stepStation {
		return w.station.focusRow(i)
	}
	return w.rigForm.focusRow(rigFormField(i))
}

func (w *Wizard) Init() tea.Cmd {
	// Warn if the encrypted secrets file is corrupted or from another machine.
	if w.App.Secrets != nil && w.App.Secrets.Corrupted {
		w.toasts.Warn("Secrets: encrypted store could not be decrypted — passwords and API keys must be re-entered")
		applog.Warn("Secrets: encrypted store corrupted or from different machine")
	}

	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (w *Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	w.toasts.Expire()

	switch msg := msg.(type) {
	case tickMsg:
		return w, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		})

	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height

	case wlUpdateMsg:
		if msg.err != nil {
			w.toasts.Error("Wavelog: " + msg.err.Error())
		} else {
			w.wlStations = msg.stations
			w.wlStationIdx = 0
			if len(msg.stations) > 0 {
				w.setSelectedStation()
			}
			w.toasts.Success(fmt.Sprintf("Wavelog: %d stations loaded", len(msg.stations)))
			return w, w.stationDetailCmd()
		}

	case wlStationDetailMsg:
		if msg.err != nil {
			w.toasts.Warn("Wavelog: station details unavailable")
		} else if msg.station != nil && w.selectedStationID() == msg.stationID {
			w.wlStation = msg.station
			fillStationFormFromWavelog(w.station, msg.station)
		}

	case wlTestMsg:
		if msg.err != nil {
			w.toasts.Error("Wavelog: " + msg.err.Error())
		} else {
			w.toasts.Success("Wavelog: connection OK")
		}

	case wlCycleStation:
		if len(w.wlStations) > 0 {
			w.wlStationIdx = (w.wlStationIdx + 1) % len(w.wlStations)
			w.setSelectedStation()
			return w, w.stationDetailCmd()
		}

	case tea.PasteMsg:
		// Forward paste to the focused text input so clipboard paste works
		// in the wizard (station form and rig form).
		switch w.step {
		case stepStation:
			if cmd := w.station.HandlePaste(msg.Content); cmd != nil {
				return w, nil
			}
		case stepRig:
			if cmd := w.rigForm.HandlePaste(msg.Content); cmd != nil {
				return w, nil
			}
		}
		return w, nil

	case tea.KeyPressMsg:
		k := msg
		switch {
		case k.String() == "f10":
			return w, tea.Quit

		case k.String() == "esc":
			if w.step > stepStation {
				w.step--
				w.fm.reset()
				applog.Debug("Wizard: step back", "step", int(w.step)+1, "total", stepCount)
				return w, nil
			}

		default:
			switch w.step {
			case stepStation:
				// Ctrl+A reveals/hides the optional advanced fields.
				if k.String() == "ctrl+a" {
					w.station.Advanced = !w.station.Advanced
					return w, nil
				}
				// Space cycles loaded Wavelog stations when Station ID is focused.
				if (k.String() == " " || msg.Code == tea.KeySpace) && w.station.WlStationID.Focused() && len(w.wlStations) > 0 {
					w.wlStationIdx = (w.wlStationIdx + 1) % len(w.wlStations)
					w.setSelectedStation()
					return w, w.stationDetailCmd()
				}
				// Shared navigation: Tab/Down, Shift+Tab/Up, and the Save & Next
				// button (Space/Enter advances to the rig step).
				if handled, cmd := w.fm.onKey(msg, w, func() tea.Cmd {
					w.finishStation()
					return nil
				}); handled {
					return w, cmd
				}
				if cmd := w.station.HandleKey(msg); cmd != nil {
					switch cmd().(type) {
					case enterOnLastFieldMsg:
						w.finishStation()
					case wlUpdateAction:
						_, _, _, _, _, _, _, _, wlURL, wlKey, _, _, _, _, _, _, _, _, _ := w.station.Values()
						if wlURL == "" || wlKey == "" {
							w.toasts.Warn("Wavelog: URL and API Key are required")
							return w, nil
						}
						return w, func() tea.Msg {
							stations, err := wavelog.FetchStations(wlURL, wlKey)
							return wlUpdateMsg{stations: stations, err: err}
						}
					case wlTestAction:
						_, _, _, _, _, _, _, _, wlURL, wlKey, _, _, _, _, _, _, _, _, _ := w.station.Values()
						if wlURL == "" || wlKey == "" {
							w.toasts.Warn("Wavelog: URL and API Key are required")
							return w, nil
						}
						return w, func() tea.Msg {
							if err := wavelog.TestConnection(wlURL, wlKey); err != nil {
								return wlTestMsg{err: err}
							}
							return wlTestMsg{}
						}
					}
					return w, nil
				}
			case stepRig:
				// Shared navigation: Tab/Down, Shift+Tab/Up, and the Save & Next
				// button (Space/Enter advances to the summary).
				if handled, cmd := w.fm.onKey(msg, w, func() tea.Cmd {
					w.finishRig()
					return nil
				}); handled {
					return w, cmd
				}
				if cmd := w.rigForm.HandleKey(msg); cmd != nil {
					switch cmd().(type) {
					case enterOnLastFieldMsg:
						w.finishRig()
					}
					return w, nil
				}
			case stepSummary:
				// Space or Enter on the Save & Start button finishes setup.
				if k.String() == "enter" || k.String() == " " || k.String() == "space" || msg.Code == tea.KeySpace {
					return w, w.handleEnter()
				}
				if k.String() == "tab" || k.String() == "shift+tab" {
					w.fm.btn.Focus = !w.fm.btn.Focus
					return w, nil
				}
			}
		}
	}

	return w, nil
}

func (w *Wizard) View() tea.View {
	// Minimum terminal size check — same as the main app.
	if w.width > 0 && w.height > 0 && (w.width < 75 || w.height < 24) {
		msg := fmt.Sprintf("\n  CQOps — Terminal too small: %dx%d (min 75x24)\n\n  Press F10 and then Enter to quit",
			w.width, w.height)
		return tea.NewView(ErrorStyle.Render(msg))
	}

	var content string
	switch w.step {
	case stepStation:
		content = w.viewStation()
	case stepRig:
		content = w.viewRig()
	case stepSummary:
		content = w.viewSummary()
	}

	// Composite toasts as floating overlay (same pattern as model.go)
	finalView := w.toasts.RenderOverlay(content, w.width, w.height)

	v := tea.NewView(finalView)
	v.AltScreen = true
	v.WindowTitle = "CQOps — Setup Wizard"
	return v
}

// ── Layout helpers ──────────────────────────────────────────────

// clampedDims returns safe terminal dimensions for the wizard.
func (w *Wizard) clampedDims() (h, ww int) {
	h = w.height
	if h < 10 {
		h = 24
	}
	ww = w.width
	if ww < 40 {
		ww = 80
	}
	return
}

// wizardFormWidth returns the inner width of the wizard form box.
func (w *Wizard) wizardFormWidth() int {
	formW := w.width - 6
	if formW < 56 {
		formW = 56
	}
	if formW > 80 {
		formW = 80
	}
	return formW
}

// wizardFormBox builds the bordered box style for wizard forms.
// Style is cached and rebuilt only when width changes.
func (w *Wizard) wizardFormBox() lipgloss.Style {
	formW := w.wizardFormWidth()
	if w.cachedFormBoxW == formW {
		return w.cachedFormBox
	}
	w.cachedFormBox = lipgloss.NewStyle().
		Width(formW).
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.TextDim).
		Padding(1, 2)
	w.cachedFormBoxW = formW
	return w.cachedFormBox
}

// wizardLayout composes banner, step indicator, bordered body, filler,
// and help bar using Bubble Tea ecosystem functions (lipgloss.JoinVertical,
// MaxHeight clipping — same pattern as model.go).
func (w *Wizard) wizardLayout(body string, help string) string {
	_, tw := w.clampedDims()
	center := wizardCenterBase.Width(tw)

	top := lipgloss.JoinVertical(lipgloss.Center,
		center.Render(w.banner()),
		"",
		center.Render(w.stepIndicator()),
		center.Render(body),
	)

	h, _ := w.clampedDims()
	contentH := h - lipgloss.Height(help)
	top = lipgloss.NewStyle().Height(contentH).MaxHeight(contentH).Render(top)

	return lipgloss.JoinVertical(lipgloss.Left, top, help)
}

// ── Banner ───────────────────────────────────────────────────────

// wizardLogoRows is the ASCII-art logo shown at the top of the wizard.
// Pure ASCII, every row exactly the same length: box-drawing glyphs have
// ambiguous terminal width and uneven rows make lipgloss misalign the logo.
var wizardLogoRows = []string{
	" ######   ######   ######  ######## ########",
	"##       ##    ## ##    ## ##    ## ##      ",
	"##       ##  # ## ##    ## ######## ########",
	"##       ##   ### ##    ## ##             ##",
	" ######   ######   ######  ##       ########",
}

// wizardLogoView renders the logo rows, each styled individually.
func wizardLogoView() string {
	rows := make([]string, 0, len(wizardLogoRows))
	for _, row := range wizardLogoRows {
		rows = append(rows, S.WizardAccent.Render(row))
	}
	return lipgloss.JoinVertical(lipgloss.Center, rows...)
}

func (w *Wizard) banner() string {
	ver := version.Resolved()
	// Plain OSC-8 hyperlink — no lipgloss styling to avoid mangling escape sequences.
	// The link is rendered as the raw ANSI hyperlink and then centered by wizardLayout.
	gh := osc8Link("https://github.com/szporwolik/cqops",
		"github.com/szporwolik/cqops")

	// Small terminals get a compact banner — the full logo layout needs
	// about 32 rows and would otherwise push the Save button off-screen.
	if w.height > 0 && w.height < 32 {
		name := S.WizardAccent.Render("CQOps v" + ver)
		return lipgloss.JoinVertical(lipgloss.Center, name, gh)
	}

	// Version and GitHub link share one line: the hyperlink keeps its raw
	// OSC-8 sequences (never wrapped in lipgloss styles).
	verLine := DimStyle.Render("v"+ver) + "  ·  " + gh

	return lipgloss.JoinVertical(lipgloss.Center,
		wizardLogoView(),
		"",
		verLine,
	)
}

// ── Step indicator ───────────────────────────────────────────────

func (w *Wizard) stepIndicator() string {
	current := int(w.step) + 1
	total := int(stepCount)
	name := ""
	switch w.step {
	case stepStation:
		name = "Station & Logbook"
	case stepRig:
		name = "Rig"
	case stepSummary:
		name = "Summary"
	}
	return S.Title.Render(fmt.Sprintf("First time wizard — Step %d/%d — %s", current, total, name))
}

// ── Step views ───────────────────────────────────────────────────

func wizHelp(bindings ...key.Binding) string {
	h := help.New()
	return HelpStyle.Render(h.ShortHelpView(bindings))
}

func (w *Wizard) viewStation() string {
	w.station.width = w.width
	body := w.wizardFormBox().Render(lipgloss.JoinVertical(lipgloss.Left,
		w.station.View().Content, "", w.fm.btn.line("Save & Next", w.wizardFormWidth())))
	help := wizHelp(
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Save & Next")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "Navigate")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Toggle")),
		key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("Ctrl+A", "Advanced")),
		key.NewBinding(key.WithKeys("f10"), key.WithHelp("F10", "Quit")),
	)
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewRig() string {
	w.rigForm.width = w.width
	body := w.wizardFormBox().Render(lipgloss.JoinVertical(lipgloss.Left,
		w.rigForm.View().Content, "", w.fm.btn.line("Save & Next", w.wizardFormWidth())))
	help := wizHelp(
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Save & Next")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Toggle flrig")),
		key.NewBinding(key.WithKeys("↑/↓", "tab"), key.WithHelp("↑↓/Tab", "Navigate")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		key.NewBinding(key.WithKeys("f10"), key.WithHelp("F10", "Quit")),
	)
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewSummary() string {
	inner := lipgloss.JoinVertical(lipgloss.Left,
		S.WizardHeader.Render("Configuration ready"),
		"",
		LabelStyle.Render("Your configuration file is almost complete."),
		"",
		LabelStyle.Render("Timezone: "+config.Timezones[config.SystemTimezoneIndex()]+" (auto-detected)"),
		"",
		LabelStyle.Render("We recommend visiting the Configuration menu after"),
		LabelStyle.Render("starting the program to set additional options and"),
		LabelStyle.Render("enable new features."),
		"",
		S.WizardAccent.Render("Press the Save & Start button below to generate"),
		S.WizardAccent.Render("the configuration file and start the program."),
	)

	body := w.wizardFormBox().Render(lipgloss.JoinVertical(lipgloss.Left,
		inner, "", w.fm.btn.line("Save & Start", w.wizardFormWidth())))
	help := wizHelp(
		key.NewBinding(key.WithKeys("space", "enter"), key.WithHelp("Space", "Save & Start")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		key.NewBinding(key.WithKeys("f10"), key.WithHelp("F10", "Quit")),
	)
	return w.wizardLayout(body, help)
}

// ── Helpers ──────────────────────────────────────────────────────

// finishStation validates the station step and advances to the rig step.
func (w *Wizard) finishStation() {
	nm, cs, _, gr, _, _, _, wlEnabled, _, _, wlStationID, _, _, _, _, _, _, _, _ := w.station.Values()
	if nm == "" {
		w.toasts.Warn("Station: name is required")
		return
	}
	if cs == "" {
		w.toasts.Warn("Station: callsign is required")
		return
	}
	if !qso.IsValidCall(cs) {
		w.toasts.Warn("Station: not a valid callsign")
		return
	}
	if gr == "" {
		w.toasts.Warn("Station: grid locator is required")
		return
	}
	if !qso.IsValidLocator(gr) {
		w.toasts.Warn("Station: not a valid grid locator")
		return
	}
	if wlEnabled {
		if wlStationID == "" {
			w.toasts.Warn("Wavelog: no Station ID — press Update then Space")
			return
		}
		if len(w.wlStations) == 0 {
			w.toasts.Warn("Wavelog: no stations loaded — press Update to fetch")
			return
		}
	}
	w.step = stepRig
	w.fm.reset()
	applog.InfoDetail("Wizard: station step done", fmt.Sprintf("call=%s grid=%s", cs, gr))
}

// finishRig validates the rig step and advances to the summary step. The
// timezone needs no step — the system (Go) detection is used as-is.
func (w *Wizard) finishRig() {
	nm, rig, _, _ := w.rigForm.Values()
	if nm == "" {
		w.toasts.Warn("Rig: name is required")
		return
	}
	radioBackend, _, _ := w.rigForm.BackendValues()
	rawHost := strings.TrimSpace(w.rigForm.BackendHost.Value())
	rawPort := strings.TrimSpace(w.rigForm.BackendPort.Value())
	if radioBackend == "flrig" {
		if rawHost == "" {
			w.toasts.Warn("Rig: flrig host is required")
			return
		}
		if rawPort == "" {
			w.toasts.Warn("Rig: flrig port is required")
			return
		}
	}
	if radioBackend == "hamlib" {
		if rawHost == "" {
			w.toasts.Warn("Rig: hamlib host is required")
			return
		}
		if rawPort == "" {
			w.toasts.Warn("Rig: hamlib port is required")
			return
		}
	}
	w.step = stepSummary
	w.fm.btn.Focus = true // the Save & Start button is the only control there
	applog.InfoDetail("Wizard: rig step done", fmt.Sprintf("rig=%s flrig=%v", rig, radioBackend == "flrig"))
}

func (w *Wizard) handleEnter() tea.Cmd {
	// The timezone comes from the system (Go) detection — no wizard step needed.
	tz := config.Timezones[config.SystemTimezoneIndex()]
	w.App.Config.General.Timezone = tz
	applog.InfoDetail("Wizard: summary step done — saving config", fmt.Sprintf("tz=%s", tz))
	return func() tea.Msg {
		// Best-effort Wavelog station sync before saving (short timeout).
		// Skipped when the profile was already fetched while selecting.
		if w.wlStation == nil {
			w.syncStationFromWavelog()
		}
		if err := w.saveConfig(); err != nil {
			w.toasts.Error(fmt.Sprintf("Setup: %v", err))
			applog.Error("Wizard: config validation failed", "error", err)
			return nil
		}
		w.Completed = true
		applog.Info("Wizard completed — launching CQOps")
		return tea.Quit()
	}
}

// setSelectedStation shows the currently selected Wavelog station in the
// Station ID field and mirrors its callsign and grid into the form. The
// remaining fields are filled asynchronously once the full profile arrives.
func (w *Wizard) setSelectedStation() {
	s := w.wlStations[w.wlStationIdx]
	w.station.WlStationID.SetValue(fmt.Sprintf("%s — %s (%s) %s", s.ID, s.Callsign, s.Name, s.Gridsquare))
	if s.Callsign != "" {
		w.station.Callsign.SetValue(s.Callsign)
	}
	w.station.Locator.SetValue(s.Gridsquare)
}

// selectedStationID returns the Wavelog station ID currently shown, or "".
func (w *Wizard) selectedStationID() string {
	if w.wlStationIdx >= 0 && w.wlStationIdx < len(w.wlStations) {
		return w.wlStations[w.wlStationIdx].ID
	}
	return ""
}

// stationDetailCmd fetches the full profile of the currently selected
// station so the rest of the form can mirror it. Nil when offline or when
// Wavelog is not fully configured.
func (w *Wizard) stationDetailCmd() tea.Cmd {
	if w.Offline {
		return nil
	}
	_, _, _, _, _, _, _, wlEnabled, wlURL, wlKey, _, _, _, _, _, _, _, _, _ := w.station.Values()
	if !wlEnabled {
		return nil
	}
	sid := w.selectedStationID()
	if sid == "" {
		return nil
	}
	return fetchWavelogStationDetailCmd(wlURL, wlKey, sid)
}

// syncStationFromWavelog fetches the selected Wavelog station profile so the
// new logbook inherits its grid, DXCC, zones and reference fields. Failures
// keep the entered values (toast warning, never fatal).
func (w *Wizard) syncStationFromWavelog() {
	_, _, _, _, _, _, _, wlEnabled, wlURL, wlKey, wlStationID, _, _, _, _, _, _, _, _ := w.station.Values()
	if !wlEnabled || wlURL == "" || wlKey == "" || wlStationID == "" {
		return
	}
	st, err := wavelog.GetStation(wlURL, wlKey, wlStationID)
	if err != nil {
		applog.Warn("Wizard: Wavelog station sync failed", "error", err)
		w.toasts.Warn("Wavelog: station sync failed — using entered values")
		return
	}
	w.wlStation = st
	applog.InfoDetail("Wizard: Wavelog station synced", fmt.Sprintf("grid=%s dxcc=%d", st.Gridsquare, st.DXCC))
}

func (w *Wizard) saveConfig() error {
	sn, cs, op, gr, sotaRef, potaRef, wwffRef, wlEnabled, wlURL, wlKey, wlStationID, iaruRegion, cqZone, ituZone, dxcc, sig, sigInfo, continent, _ := w.station.Values()
	nm, rig, ant, pwr := w.rigForm.Values()
	radioBackend, radioBackendHost, radioBackendPort := w.rigForm.BackendValues()
	rotorBackend, rotorHost, rotorPort := w.rigForm.RotorValues()
	wsjtxEnabled, wsjtxHost, wsjtxPortStr := w.rigForm.WsjtxValues()
	wsjtxPort, _ := strconv.Atoi(wsjtxPortStr)
	if wsjtxPort <= 0 {
		wsjtxPort = 2233
	}

	// Create operator entry if one was selected in the form.
	var activeOpID string
	if op != "" {
		activeOpID = config.NewID(op)
		if w.App.Config.Operators == nil {
			w.App.Config.Operators = make(map[string]config.Operator)
		}
		w.App.Config.Operators[activeOpID] = config.Operator{ID: activeOpID, Callsign: op}
	}

	rigID := config.NewID("default-rig")
	lbID := config.NewID("default-logbook")

	flrigHost, flrigPort := "", ""
	hamlibHost, hamlibPort := "", ""
	switch radioBackend {
	case "flrig":
		flrigHost, flrigPort = radioBackendHost, radioBackendPort
	case "hamlib":
		hamlibHost, hamlibPort = radioBackendHost, radioBackendPort
	}

	pollInterval, clamped := w.rigForm.normalizePollInterval()
	w.App.Config.Rigs = map[string]config.RigPreset{
		rigID: {
			ID:              rigID,
			Name:            nm,
			Model:           rig,
			Antenna:         ant,
			Power:           pwr,
			RadioBackend:    radioBackend,
			FlrigHost:       flrigHost,
			FlrigPort:       flrigPort,
			HamlibRadioHost: hamlibHost,
			HamlibRadioPort: hamlibPort,
			PollIntervalS:   pollInterval,
			RotorBackend:    rotorBackend,
			RotorHamlibHost: rotorHost,
			RotorHamlibPort: rotorPort,
			WsjtxEnabled:    wsjtxEnabled,
			WsjtxUDPHost:    wsjtxHost,
			WsjtxUDPPort:    wsjtxPort,
		},
	}
	if clamped {
		w.toasts.Warn("Rig: poll interval adjusted to " + strconv.Itoa(pollInterval) + "s (valid range: 1–60)")
	}

	// Callbook providers (QRZ etc.) are configured post-wizard via the
	// Integration → Callbook menu.

	var wl *config.WavelogConfig
	if wlEnabled && wlURL != "" && wlKey != "" {
		sid := wavelog.SanitizeStationID(wlStationID)
		// Extract actual station ID from the selected station if available
		if w.wlStationIdx >= 0 && w.wlStationIdx < len(w.wlStations) {
			sid = w.wlStations[w.wlStationIdx].ID
		}
		wl = &config.WavelogConfig{
			Enabled:          wlEnabled,
			URL:              wlURL,
			APIKey:           wlKey,
			StationProfileID: sid,
		}
	}

	lbName := sn
	if lbName == "" {
		lbName = "Default"
	}
	station := config.Station{
		Callsign:   cs,
		Grid:       gr,
		RigName:    rigID,
		SOTARef:    sotaRef,
		POTARef:    potaRef,
		WWFFRef:    wwffRef,
		IARURegion: iaruRegion,
		CQZone:     cqZone,
		ITUZone:    ituZone,
		DXCC:       dxcc,
		SIG:        sig,
		SIGInfo:    sigInfo,
		Continent:  continent,
	}
	// When the Wavelog station profile was fetched, mirror its values
	// (grid, DXCC, zones, reference fields) into the new logbook.
	if w.wlStation != nil {
		applyWavelogStation(w.wlStation, &station)
	}
	w.App.Config.State.ActiveLogbook = lbID
	w.App.Config.Logbooks = map[string]config.Logbook{
		lbID: {
			ID:             lbID,
			Name:           lbName,
			ActiveOperator: activeOpID,
			Station:        station,
			Wavelog:        wl,
		},
	}

	// Validate the assembled config before finalizing.
	if err := w.App.Config.Validate(); err != nil {
		return fmt.Errorf("invalid setup: %w", err)
	}

	lb := w.App.Config.Logbooks[lbID]
	w.App.Logbook = &lb
	w.App.LogbookName = lbID

	applog.InfoDetail("Wizard completed", fmt.Sprintf("call=%s rig=%s flrig=%v wsjtx=%v wavelog=%v tz=%s",
		cs, rig, radioBackend == "flrig", wsjtxEnabled, wlEnabled, config.Timezones[config.SystemTimezoneIndex()]))
	return nil
}
