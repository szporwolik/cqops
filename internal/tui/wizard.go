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
	stepTimezone
	stepSummary
	stepCount // sentinel
)

type Wizard struct {
	App       *app.App
	step      wizardStep
	station   *StationForm
	rigForm   *RigForm
	tzIndex   int
	toasts    *ToastQueue
	width     int
	height    int
	Completed bool // true only when full wizard finished
	Offline   bool // when true, skip all network-dependent operations

	// Cached form box style — rebuilt only when width changes.
	cachedFormBox  lipgloss.Style
	cachedFormBoxW int

	// Wavelog async state (for wizard step 1 buttons)
	wlUpdating   bool
	wlTesting    bool
	wlStatus     string
	wlStations   []wavelog.StationProfile
	wlStationIdx int
}

func NewWizard(a *app.App) *Wizard {
	applog.Info("Wizard started — first-run setup")
	sf := NewStationForm("", "", "")
	sf.HideGPSGrid = true // GPS Grid is not relevant during first-run setup
	return &Wizard{
		App:     a,
		step:    stepStation,
		station: sf,
		rigForm: NewRigForm("Xiegu G90 (optional)", "HWEF 20.5 (optional)", "20"),
		tzIndex: config.SystemTimezoneIndex(),
		toasts:  NewToastQueue(),
	}
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
		w.wlUpdating = false
		if msg.err != nil {
			w.wlStatus = msg.err.Error()
			w.toasts.Error(w.wlStatus)
		} else {
			w.wlStations = msg.stations
			w.wlStationIdx = 0
			if len(msg.stations) > 0 {
				s := msg.stations[0]
				w.station.WlStationID.SetValue(fmt.Sprintf("%s — %s (%s) %s", s.ID, s.Callsign, s.Name, s.Gridsquare))
			}
			w.wlStatus = fmt.Sprintf("%d stations loaded — Space to cycle", len(msg.stations))
			w.toasts.Success(fmt.Sprintf("%d stations loaded, use Space to toggle", len(msg.stations)))
		}

	case wlTestMsg:
		w.wlTesting = false
		if msg.err != nil {
			w.wlStatus = msg.err.Error()
			w.toasts.Error(w.wlStatus)
		} else {
			w.wlStatus = "OK — Wavelog reachable"
			w.toasts.Success("Wavelog connection OK")
		}

	case wlCycleStation:
		if len(w.wlStations) > 0 {
			w.wlStationIdx = (w.wlStationIdx + 1) % len(w.wlStations)
			s := w.wlStations[w.wlStationIdx]
			w.station.WlStationID.SetValue(fmt.Sprintf("%s — %s (%s) %s", s.ID, s.Callsign, s.Name, s.Gridsquare))
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
				applog.Debug("Wizard: step back", "step", int(w.step)+1, "total", stepCount)
				return w, nil
			}

		default:
			switch w.step {
			case stepStation:
				// Space cycles loaded Wavelog stations when Station ID is focused.
				if (k.String() == " " || msg.Code == tea.KeySpace) && w.station.WlStationID.Focused() && len(w.wlStations) > 0 {
					w.wlStationIdx = (w.wlStationIdx + 1) % len(w.wlStations)
					s := w.wlStations[w.wlStationIdx]
					w.station.WlStationID.SetValue(fmt.Sprintf("%s — %s (%s) %s", s.ID, s.Callsign, s.Name, s.Gridsquare))
					return w, nil
				}
				if cmd := w.station.HandleKey(msg); cmd != nil {
					switch cmd().(type) {
					case enterOnLastFieldMsg:
						nm, cs, _, gr, _, _, _, wlEnabled, _, _, wlStationID, _, _, _, _, _, _, _ := w.station.Values()
						if nm == "" {
							w.toasts.Warn("Station name is required")
							return w, nil
						}
						if cs == "" {
							w.toasts.Warn("Callsign is required")
							return w, nil
						}
						if !qso.IsValidCall(cs) {
							w.toasts.Warn("Not a valid callsign")
							return w, nil
						}
						if gr == "" {
							w.toasts.Warn("Grid locator is required")
							return w, nil
						}
						if !qso.IsValidLocator(gr) {
							w.toasts.Warn("Not a valid grid locator")
							return w, nil
						}
						if wlEnabled {
							if wlStationID == "" {
								w.toasts.Warn("Wavelog: no Station ID — press Update then Space")
								return w, nil
							}
							if len(w.wlStations) == 0 {
								w.toasts.Warn("No stations loaded — press Update to fetch from Wavelog")
								return w, nil
							}
						}
						w.step = stepRig
						applog.InfoDetail("Wizard: station step done", fmt.Sprintf("call=%s grid=%s", cs, gr))
					case wlUpdateAction:
						_, _, _, _, _, _, _, _, wlURL, wlKey, _, _, _, _, _, _, _, _ := w.station.Values()
						if wlURL == "" || wlKey == "" {
							w.toasts.Warn("Wavelog URL and API Key are required")
							return w, nil
						}
						w.wlUpdating = true
						w.wlStatus = "Fetching stations…"
						return w, func() tea.Msg {
							stations, err := wavelog.FetchStations(wlURL, wlKey)
							return wlUpdateMsg{stations: stations, err: err}
						}
					case wlTestAction:
						_, _, _, _, _, _, _, _, wlURL, wlKey, _, _, _, _, _, _, _, _ := w.station.Values()
						if wlURL == "" || wlKey == "" {
							w.toasts.Warn("Wavelog URL and API Key are required")
							return w, nil
						}
						w.wlTesting = true
						w.wlStatus = "Testing…"
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
				if cmd := w.rigForm.HandleKey(msg); cmd != nil {
					switch cmd().(type) {
					case enterOnLastFieldMsg:
						nm, rig, _, _ := w.rigForm.Values()
						if nm == "" {
							w.toasts.Warn("Rig name is required")
							return w, nil
						}
						radioBackend, _, _ := w.rigForm.BackendValues()
						rawHost := strings.TrimSpace(w.rigForm.BackendHost.Value())
						rawPort := strings.TrimSpace(w.rigForm.BackendPort.Value())
						if radioBackend == "flrig" {
							if rawHost == "" {
								w.toasts.Warn("Flrig host is required")
								return w, nil
							}
							if rawPort == "" {
								w.toasts.Warn("Flrig port is required")
								return w, nil
							}
						}
						if radioBackend == "hamlib" {
							if rawHost == "" {
								w.toasts.Warn("Hamlib host is required")
								return w, nil
							}
							if rawPort == "" {
								w.toasts.Warn("Hamlib port is required")
								return w, nil
							}
						}
						w.step = stepTimezone
						applog.InfoDetail("Wizard: rig step done", fmt.Sprintf("rig=%s flrig=%v", rig, radioBackend == "flrig"))
					}
					return w, nil
				}
			case stepTimezone:
				if k.String() == "ctrl+s" || k.String() == "\x13" {
					w.step = stepSummary
					applog.InfoDetail("Wizard: timezone step done", fmt.Sprintf("tz=%s", config.Timezones[w.tzIndex]))
					return w, nil
				}
				if msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k" {
					if w.tzIndex > 0 {
						w.tzIndex--
					}
				}
				if msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j" {
					if w.tzIndex < len(config.Timezones)-1 {
						w.tzIndex++
					}
				}
			case stepSummary:
				if k.String() == "ctrl+s" || k.String() == "\x13" {
					return w, w.handleEnter()
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
	case stepTimezone:
		content = w.viewTimezone()
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

// wizardFormBox builds the bordered box style for wizard forms.
// Style is cached and rebuilt only when width changes.
func (w *Wizard) wizardFormBox() lipgloss.Style {
	formW := w.width - 6
	if formW < 56 {
		formW = 56
	}
	if formW > 80 {
		formW = 80
	}
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

func (w *Wizard) banner() string {
	ver := version.Resolved()
	name := S.WizardAccent.Render("CQOps v" + ver)
	tag := LabelStyle.Render("Portable Ham Radio Logger")

	// Plain OSC-8 hyperlink — no lipgloss styling to avoid mangling escape sequences.
	// The link is rendered as the raw ANSI hyperlink and then centered by wizardLayout.
	gh := osc8Link("https://github.com/szporwolik/cqops",
		"github.com/szporwolik/cqops")

	return lipgloss.JoinVertical(lipgloss.Center,
		name+"  —  "+tag,
		gh,
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
	case stepTimezone:
		name = "General"
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
	body := w.wizardFormBox().Render(w.station.View().Content)
	help := wizHelp(
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save & Next")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "Navigate")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Toggle")),
		key.NewBinding(key.WithKeys("f10"), key.WithHelp("F10", "Quit")),
	)
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewRig() string {
	w.rigForm.width = w.width
	body := w.wizardFormBox().Render(w.rigForm.View().Content)
	help := wizHelp(
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save & Next")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Toggle flrig")),
		key.NewBinding(key.WithKeys("↑/↓", "tab"), key.WithHelp("↑↓/Tab", "Navigate")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		key.NewBinding(key.WithKeys("f10"), key.WithHelp("F10", "Quit")),
	)
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewTimezone() string {
	detectedIdx := config.SystemTimezoneIndex()
	availW := w.width
	if availW < 40 {
		availW = 80
	}

	start := w.tzIndex - 4
	if start < 0 {
		start = 0
	}
	end := start + 9
	if end > len(config.Timezones) {
		end = len(config.Timezones)
		start = end - 9
		if start < 0 {
			start = 0
		}
	}

	var inner strings.Builder
	for i := start; i < end; i++ {
		tz := config.Timezones[i]
		prefix := "  "
		lbl := S.FormLabelWide.Align(lipgloss.Left).Render(tz)
		if i == w.tzIndex {
			prefix = S.FormPrefixOn.Render("> ")
			lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(tz)
		}
		inner.WriteString(padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl), availW))
		inner.WriteString("\n")
	}

	detected := config.Timezones[detectedIdx]
	inner.WriteString(padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, "  ", DimStyle.Render("System: "+detected)), availW))

	body := w.wizardFormBox().Render(inner.String())
	help := wizHelp(
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save & Next")),
		key.NewBinding(key.WithKeys("↑↓"), key.WithHelp("↑↓", "Choose")),
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
		LabelStyle.Render("We recommend visiting the Configuration menu after"),
		LabelStyle.Render("starting the program to set additional options and"),
		LabelStyle.Render("enable new features."),
		"",
		S.WizardAccent.Render("Press Ctrl+S to generate the configuration"),
		S.WizardAccent.Render("file and start the program."),
	)

	body := w.wizardFormBox().Render(inner)
	help := wizHelp(
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save & Start")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		key.NewBinding(key.WithKeys("f10"), key.WithHelp("F10", "Quit")),
	)
	return w.wizardLayout(body, help)
}

// ── Helpers ──────────────────────────────────────────────────────

func (w *Wizard) handleEnter() tea.Cmd {
	w.App.Config.General.Timezone = config.Timezones[w.tzIndex]
	applog.InfoDetail("Wizard: summary step done — saving config", fmt.Sprintf("tz=%s", config.Timezones[w.tzIndex]))
	if err := w.saveConfig(); err != nil {
		w.toasts.Error(fmt.Sprintf("Setup error: %v", err))
		applog.Error("Wizard: config validation failed", "error", err)
		return nil
	}
	w.Completed = true
	applog.Info("Wizard completed — launching CQOps")
	return tea.Quit
}

func (w *Wizard) saveConfig() error {
	sn, cs, op, gr, sotaRef, potaRef, wwffRef, wlEnabled, wlURL, wlKey, wlStationID, iaruRegion, cqZone, ituZone, dxcc, sig, sigInfo, continent := w.station.Values()
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
			RotorBackend:    rotorBackend,
			RotorHamlibHost: rotorHost,
			RotorHamlibPort: rotorPort,
			WsjtxEnabled:    wsjtxEnabled,
			WsjtxUDPHost:    wsjtxHost,
			WsjtxUDPPort:    wsjtxPort,
		},
	}

	// Callbook providers (QRZ etc.) are configured post-wizard via the
	// Integration → Callbook menu.

	var wl *config.WavelogConfig
	if wlEnabled && wlURL != "" && wlKey != "" {
		sid := wlStationID
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
	w.App.Config.State.ActiveLogbook = lbID
	w.App.Config.Logbooks = map[string]config.Logbook{
		lbID: {
			ID:             lbID,
			Name:           lbName,
			ActiveOperator: activeOpID,
			Station: config.Station{
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
			},
			Wavelog: wl,
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
		cs, rig, radioBackend == "flrig", wsjtxEnabled, wlEnabled, config.Timezones[w.tzIndex]))
	return nil
}
