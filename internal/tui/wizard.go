package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/version"
	"github.com/szporwolik/cqops/internal/wavelog"
)

type wizardStep int

const (
	stepStation wizardStep = iota
	stepRig
	stepWSJTX
	stepTimezone
	stepSummary
	stepCount // sentinel
)

type Wizard struct {
	App           *app.App
	step          wizardStep
	station       *StationForm
	rigForm       *RigForm
	wsjtxEnable   bool
	wsjtxHost     textinput.Model
	wsjtxPort     textinput.Model
	qrzEnable     bool
	qrzUser       textinput.Model
	qrzPass       textinput.Model
	qrzTesting    bool
	qrzTestResult string
	integFocus    int // 0=wsjtx cb, 1=wsjtx host, 2=wsjtx port, 3=qrz cb, 4=qrz user, 5=qrz pass, 6=qrz test
	tzIndex       int
	toasts        *ToastQueue
	width         int
	height        int
	Completed     bool // true only when full wizard finished

	// Wavelog async state (for wizard step 1 buttons)
	wlUpdating   bool
	wlTesting    bool
	wlStatus     string
	wlStations   []wavelog.StationProfile
	wlStationIdx int
}

func NewWizard(a *app.App) *Wizard {
	host := newTextinput()
	host.CharLimit = 40
	host.SetWidth(22)
	host.SetValue("127.0.0.1")

	port := newTextinput()
	port.CharLimit = 6
	port.SetWidth(22)
	port.SetValue("2233")

	qrzUser := newTextinput()
	qrzUser.CharLimit = 30
	qrzUser.SetWidth(22)
	qrzUser.Placeholder = "QRZ.com username"

	qrzPass := newTextinput()
	qrzPass.CharLimit = 40
	qrzPass.SetWidth(22)
	qrzPass.Placeholder = "QRZ.com password"
	qrzPass.EchoMode = textinput.EchoPassword
	qrzPass.EchoCharacter = '*'

	applog.Info("Wizard started — first-run setup")
	return &Wizard{
		App:         a,
		step:        stepStation,
		station:     NewStationForm("", "", ""),
		rigForm:     NewRigForm("Xiegu G90", "HWEF 20.5", "20"),
		wsjtxEnable: false,
		wsjtxHost:   host,
		wsjtxPort:   port,
		qrzUser:     qrzUser,
		qrzPass:     qrzPass,
		tzIndex:     config.SystemTimezoneIndex(),
		toasts:      NewToastQueue(),
	}
}

func (w *Wizard) Init() tea.Cmd {
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

	case callbookTestMsg:
		w.qrzTesting = false
		if msg.err != nil {
			w.qrzTestResult = msg.err.Error()
			w.toasts.Error(msg.err.Error())
		} else if msg.ok {
			w.qrzTestResult = "OK — QRZ.com connected"
			w.toasts.Success("QRZ connection OK")
		} else {
			w.qrzTestResult = "QRZ test failed"
		}

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
				if cmd := w.station.HandleKey(msg); cmd != nil {
					switch cmd().(type) {
					case enterOnLastFieldMsg:
						cs, _, gr, _, _, _, wlEnabled, _, _, wlStationID, _, _, _, _, _, _ := w.station.Values()
						if cs == "" {
							w.toasts.Error("Callsign is required")
							return w, nil
						}
						if !qso.IsValidCall(cs) {
							w.toasts.Error("Not a valid callsign")
							return w, nil
						}
						if gr == "" {
							w.toasts.Error("Grid locator is required")
							return w, nil
						}
						if !qso.IsValidLocator(gr) {
							w.toasts.Error("Not a valid grid locator")
							return w, nil
						}
						if wlEnabled {
							if wlStationID == "" {
								w.toasts.Error("Wavelog: no Station ID — press Update then Space")
								return w, nil
							}
							if len(w.wlStations) == 0 {
								w.toasts.Error("No stations loaded — press Update to fetch from Wavelog")
								return w, nil
							}
						}
						w.step = stepRig
						applog.InfoDetail("Wizard: station step done", fmt.Sprintf("call=%s grid=%s", cs, gr))
					case wlUpdateAction:
						_, _, _, _, _, _, _, wlURL, wlKey, _, _, _, _, _, _, _ := w.station.Values()
						if wlURL == "" || wlKey == "" {
							w.toasts.Error("Wavelog URL and API Key are required")
							return w, nil
						}
						w.wlUpdating = true
						w.wlStatus = "Fetching stations…"
						return w, func() tea.Msg {
							stations, err := wavelog.FetchStations(wlURL, wlKey)
							return wlUpdateMsg{stations: stations, err: err}
						}
					case wlTestAction:
						_, _, _, _, _, _, _, wlURL, wlKey, _, _, _, _, _, _, _ := w.station.Values()
						if wlURL == "" || wlKey == "" {
							w.toasts.Error("Wavelog URL and API Key are required")
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
						rig, _, _ := w.rigForm.Values()
						if rig == "" {
							w.toasts.Error("Rig model is required")
							return w, nil
						}
						flrigOn, _, _ := w.rigForm.FlrigValues()
						// Check raw field values (FlrigValues fills defaults, so check directly).
						rawHost := strings.TrimSpace(w.rigForm.FlrigHost.Value())
						rawPort := strings.TrimSpace(w.rigForm.FlrigPort.Value())
						if flrigOn {
							if rawHost == "" {
								w.toasts.Error("Flrig host is required when flrig is enabled")
								return w, nil
							}
							if rawPort == "" {
								w.toasts.Error("Flrig port is required when flrig is enabled")
								return w, nil
							}
						}
						w.step = stepWSJTX
						applog.InfoDetail("Wizard: rig step done", fmt.Sprintf("rig=%s flrig=%v", rig, flrigOn))
					}
					return w, nil
				}
			case stepWSJTX:
				// Space toggles checkboxes
				if k.String() == " " || msg.Code == tea.KeySpace {
					switch w.integFocus {
					case 0:
						w.wsjtxEnable = !w.wsjtxEnable
						if w.wsjtxEnable {
							if w.wsjtxHost.Value() == "" {
								w.wsjtxHost.SetValue("127.0.0.1")
							}
							if w.wsjtxPort.Value() == "" {
								w.wsjtxPort.SetValue("2233")
							}
						}
					case 3:
						w.qrzEnable = !w.qrzEnable
					}
					return w, nil
				}
				// Enter triggers Test button
				if k.String() == "enter" && w.integFocus == 6 {
					user := strings.TrimSpace(w.qrzUser.Value())
					pass := w.qrzPass.Value()
					if user == "" || pass == "" {
						w.toasts.Error("QRZ username and password required")
						return w, nil
					}
					w.qrzTesting = true
					w.qrzTestResult = "Testing…"
					return w, func() tea.Msg {
						_, err := qrz.Lookup(user, pass, "SP9MOA")
						if err != nil {
							return callbookTestMsg{ok: false, err: err}
						}
						return callbookTestMsg{ok: true}
					}
				}
				// Ctrl+S advances
				if k.String() == "ctrl+s" || k.String() == "\x13" {
					if w.wsjtxEnable {
						if strings.TrimSpace(w.wsjtxHost.Value()) == "" {
							w.toasts.Error("UDP Host is required when WSJT-X is enabled")
							return w, nil
						}
						if strings.TrimSpace(w.wsjtxPort.Value()) == "" {
							w.toasts.Error("UDP Port is required when WSJT-X is enabled")
							return w, nil
						}
					}
					if w.qrzEnable {
						if strings.TrimSpace(w.qrzUser.Value()) == "" {
							w.toasts.Error("QRZ username is required when QRZ is enabled")
							return w, nil
						}
						if w.qrzPass.Value() == "" {
							w.toasts.Error("QRZ password is required when QRZ is enabled")
							return w, nil
						}
					}
					w.step = stepTimezone
					_, _, _, _, _, _, wlOn, _, _, _, _, _, _, _, _, _ := w.station.Values()
					applog.InfoDetail("Wizard: integrations step done", fmt.Sprintf("wsjtx=%v qrz=%v wavelog=%v", w.wsjtxEnable, w.qrzEnable, wlOn))
					return w, nil
				}
				// Tab / Down navigation
				if k.String() == "tab" || msg.Code == tea.KeyDown || k.String() == "down" {
					w.integFocus++
					// Skip hidden WSJT-X sub-fields: jump to QRZ checkbox
					if !w.wsjtxEnable && w.integFocus >= 1 && w.integFocus <= 2 {
						w.integFocus = 3
					}
					// Skip hidden QRZ sub-fields: wrap to WSJT-X checkbox
					if !w.qrzEnable && w.integFocus >= 4 && w.integFocus <= 6 {
						w.integFocus = 0
					}
					if w.integFocus > 6 {
						w.integFocus = 0
					}
					w.updateIntegFocus()
					return w, nil
				}
				// Shift+Tab / Up navigation
				if msg.Code == tea.KeyUp || k.String() == "shift+tab" || k.String() == "up" {
					w.integFocus--
					// Skip hidden QRZ sub-fields going up: jump to QRZ checkbox
					if !w.qrzEnable && w.integFocus >= 4 && w.integFocus <= 6 {
						w.integFocus = 3
					}
					// Skip hidden WSJT-X sub-fields going up: jump to WSJT-X checkbox
					if !w.wsjtxEnable && w.integFocus >= 1 && w.integFocus <= 2 {
						w.integFocus = 0
					}
					if w.integFocus < 0 {
						if w.qrzEnable {
							w.integFocus = 6
						} else {
							w.integFocus = 3
						}
					}
					w.updateIntegFocus()
					return w, nil
				}
				// Text input for focused fields
				switch w.integFocus {
				case 1:
					w.wsjtxHost, _ = w.wsjtxHost.Update(msg)
				case 2:
					w.wsjtxPort, _ = w.wsjtxPort.Update(msg)
				case 4:
					w.qrzUser, _ = w.qrzUser.Update(msg)
				case 5:
					w.qrzPass, _ = w.qrzPass.Update(msg)
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
	case stepWSJTX:
		content = w.viewWSJTX()
	case stepTimezone:
		content = w.viewTimezone()
	case stepSummary:
		content = w.viewSummary()
	}

	// Composite toasts as floating overlay (same pattern as model.go)
	finalView := RenderToastOverlay(content, w.toasts.Active(), w.width, w.height)

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
func (w *Wizard) wizardFormBox() lipgloss.Style {
	formW := w.width - 6
	if formW < 56 {
		formW = 56
	}
	if formW > 80 {
		formW = 80
	}
	return lipgloss.NewStyle().
		Width(formW).
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.TextDim).
		Padding(1, 2)
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

	h, tw := w.clampedDims()
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
	case stepWSJTX:
		name = "Integrations"
	case stepTimezone:
		name = "General"
	case stepSummary:
		name = "Summary"
	}
	return S.Title.Render(fmt.Sprintf("First time wizard — Step %d/%d — %s", current, total, name))
}

func (w *Wizard) updateIntegFocus() {
	blurTextinputs(&w.wsjtxHost, &w.wsjtxPort, &w.qrzUser, &w.qrzPass)
	switch w.integFocus {
	case 1:
		w.wsjtxHost.Focus()
	case 2:
		w.wsjtxPort.Focus()
	case 4:
		w.qrzUser.Focus()
	case 5:
		w.qrzPass.Focus()
	}
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

func (w *Wizard) viewWSJTX() string {
	var inner strings.Builder
	availW := w.width
	if availW < 40 {
		availW = 80
	}

	// ── WSJT-X section ──
	wsjtxCb := "[ ]"
	if w.wsjtxEnable {
		wsjtxCb = "[x]"
	}
	wsjtxPrefix := "  "
	wsjtxLabel := S.FormLabelWide.Align(lipgloss.Left).Render("WSJT-X:")
	if w.integFocus == 0 {
		wsjtxPrefix = S.FormPrefixOn.Render("> ")
		wsjtxLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("WSJT-X:")
		wsjtxCb = CursorStyle.Render(wsjtxCb)
	}
	inner.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, wsjtxPrefix, wsjtxLabel, " ", wsjtxCb),
		availW))
	inner.WriteString("\n")

	if w.wsjtxEnable {
		inner.WriteString(renderIntegField("  UDP Host:", &w.wsjtxHost, w.integFocus == 1, false, availW))
		inner.WriteString(renderIntegField("  UDP Port:", &w.wsjtxPort, w.integFocus == 2, false, availW))
	}

	// ── QRZ section ──
	qrzCb := "[ ]"
	if w.qrzEnable {
		qrzCb = "[x]"
	}
	qrzPrefix := "  "
	qrzLabel := S.FormLabelWide.Align(lipgloss.Left).Render("QRZ.com:")
	if w.integFocus == 3 {
		qrzPrefix = S.FormPrefixOn.Render("> ")
		qrzLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("QRZ.com:")
		qrzCb = CursorStyle.Render(qrzCb)
	}
	inner.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, qrzPrefix, qrzLabel, " ", qrzCb),
		availW))
	inner.WriteString("\n")

	if w.qrzEnable {
		inner.WriteString(renderIntegField("  Username:", &w.qrzUser, w.integFocus == 4, false, availW))
		inner.WriteString(renderIntegField("  Password:", &w.qrzPass, w.integFocus == 5, true, availW))

		// Test button — fixed padding so it never shifts on focus.
		testBtn := "[ Test ]"
		testPrefix := "    "
		styledBtn := InputStyle.Render(testBtn)
		if w.integFocus == 6 {
			testPrefix = S.FormPrefixOn.Render("> ") + "  "
			styledBtn = CursorStyle.Render(testBtn)
		}
		status := ""
		if w.qrzTesting {
			status = DimStyle.Render("Testing…")
		} else if w.qrzTestResult != "" {
			if strings.Contains(w.qrzTestResult, "OK") {
				status = SuccessStyle.Render(w.qrzTestResult)
			} else {
				status = ErrorStyle.Render(w.qrzTestResult)
			}
		}
		inner.WriteString(padOrTrunc(testPrefix+styledBtn+"  "+status, availW))
		inner.WriteString("\n")
	}

	body := w.wizardFormBox().Render(inner.String())
	help := wizHelp(
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save & Next")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Toggle")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Test")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "Navigate")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		key.NewBinding(key.WithKeys("f10"), key.WithHelp("F10", "Quit")),
	)
	return w.wizardLayout(body, help)
}

// renderIntegField renders a labelled textinput line for the integrations step.
// Matches the pattern used by other forms: dynamic textinput width, truncation.
func renderIntegField(label string, ti *textinput.Model, focused bool, masked bool, maxW int) string {
	raw := strings.TrimSpace(ti.Value())

	const labelW = 2 + 17 // prefix + FormLabelWide width
	const maxVW = 40

	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	vw := maxW - labelW - 1
	if vw < 3 {
		vw = 3
	}
	if vw > maxVW {
		vw = maxVW
	}

	var val string
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
		ti.SetWidth(vw)
		if lipgloss.Width(raw) > vw {
			ti.SetWidth(vw - 1)
		}
		ti.SetCursor(ti.Position())
		val = ti.View()
	} else if raw == "" {
		val = DimStyle.Render("\u2014")
	} else if masked {
		val = ValueStyle.Render(truncateText(strings.Repeat("*", len(raw)), vw))
	} else {
		val = ValueStyle.Render(truncateText(raw, vw))
	}
	return padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val), maxW) + "\n"
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
	cs, op, gr, sotaRef, potaRef, wwffRef, wlEnabled, wlURL, wlKey, wlStationID, iaruRegion, cqZone, ituZone, dxcc, sig, sigInfo := w.station.Values()
	rig, ant, pwr := w.rigForm.Values()
	flrigEnabled, flrigHost, flrigPort := w.rigForm.FlrigValues()

	if op == "" {
		op = cs
	}

	rigID := config.NewID("default-rig")
	lbID := config.NewID("default-logbook")

	w.App.Config.Rigs = map[string]config.RigPreset{
		rigID: {
			ID:           rigID,
			Model:        rig,
			Antenna:      ant,
			Power:        pwr,
			FlrigEnabled: flrigEnabled,
			FlrigHost:    flrigHost,
			FlrigPort:    flrigPort,
		},
	}

	w.App.Config.WSJTX.Enabled = w.wsjtxEnable
	if w.wsjtxEnable {
		w.App.Config.WSJTX.UDPHost = strings.TrimSpace(w.wsjtxHost.Value())
		port, _ := strconv.Atoi(strings.TrimSpace(w.wsjtxPort.Value()))
		if port > 0 {
			w.App.Config.WSJTX.UDPPort = port
		} else {
			w.App.Config.WSJTX.UDPPort = 2233
		}
	}

	w.App.Config.QRZ.Enabled = w.qrzEnable
	if w.qrzEnable {
		w.App.Config.QRZ.User = strings.TrimSpace(w.qrzUser.Value())
		w.App.Config.QRZ.Pass = w.qrzPass.Value()
	}

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

	w.App.Config.State.ActiveLogbook = lbID
	w.App.Config.Logbooks = map[string]config.Logbook{
		lbID: {
			ID:          lbID,
			Description: "Default station logbook",
			Station: config.Station{
				Callsign:   cs,
				Operator:   op,
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
		cs, rig, flrigEnabled, w.wsjtxEnable, wlEnabled, config.Timezones[w.tzIndex]))
	return nil
}
