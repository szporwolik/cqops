package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/version"
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
	App         *app.App
	step        wizardStep
	station     *StationForm
	rigForm     *RigForm
	wsjtxEnable bool
	wsjtxHost   textinput.Model
	wsjtxPort   textinput.Model
	wsjtxFocus  int // 0=toggle, 1=host, 2=port
	tzIndex     int
	toasts      *ToastQueue
	width       int
	height      int
	Completed   bool // true only when full wizard finished
}

func NewWizard(a *app.App) *Wizard {
	host := textinput.New()
	host.CharLimit = 40
	host.SetWidth(22)
	host.SetValue("127.0.0.1")
	host.Prompt = ""

	port := textinput.New()
	port.CharLimit = 6
	port.SetWidth(22)
	port.SetValue("2233")
	port.Prompt = ""

	applog.Info("Wizard started — first-run setup")
	return &Wizard{
		App:         a,
		step:        stepStation,
		station:     NewStationForm("", "", ""),
		rigForm:     NewRigForm("Xiegu G90", "HWEF 20.5", "20"),
		wsjtxEnable: true,
		wsjtxHost:   host,
		wsjtxPort:   port,
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

	case tea.KeyPressMsg:
		k := msg
		switch {
		case k.String() == "f10":
			return w, tea.Quit

		case k.String() == "esc":
			if w.step > stepStation {
				w.step--
				return w, nil
			}

		default:
			switch w.step {
			case stepStation:
				if cmd := w.station.HandleKey(msg); cmd != nil {
					cs, _, gr, _, _, _ := w.station.Values()
					if cs == "" {
						w.toasts.Error("Callsign is required")
						return w, nil
					}
					if gr == "" {
						w.toasts.Error("Grid locator is required")
						return w, nil
					}
					w.step = stepRig
					applog.InfoDetail("Wizard: station step done", fmt.Sprintf("call=%s grid=%s", cs, gr))
					return w, nil
				}
			case stepRig:
				if cmd := w.rigForm.HandleKey(msg); cmd != nil {
					rig, _, _ := w.rigForm.Values()
					if rig == "" {
						w.toasts.Error("Rig model is required")
						return w, nil
					}
					w.step = stepWSJTX
					flrigOn, _, _ := w.rigForm.FlrigValues()
					applog.InfoDetail("Wizard: rig step done", fmt.Sprintf("rig=%s flrig=%v", rig, flrigOn))
					return w, nil
				}
			case stepWSJTX:
				// Space always toggles WSJT-X at this step
				if k.String() == " " || msg.Code == tea.KeySpace {
					w.wsjtxEnable = !w.wsjtxEnable
					return w, nil
				}
				// Ctrl+S advances to next step (same as steps 1-2)
				if k.String() == "ctrl+s" || k.String() == "\x13" {
					w.step = stepTimezone
					return w, nil
				}
				if k.String() == "tab" || k.String() == "enter" || msg.Code == tea.KeyDown || k.String() == "down" {
					w.wsjtxFocus++
					if !w.wsjtxEnable && w.wsjtxFocus > 0 {
						w.wsjtxFocus = 0
					}
					if w.wsjtxFocus > 2 {
						w.wsjtxFocus = 0
					}
					w.updateWSJTXFocus()
					return w, nil
				}
				if msg.Code == tea.KeyUp || k.String() == "shift+tab" || k.String() == "up" {
					w.wsjtxFocus--
					if w.wsjtxFocus < 0 {
						if w.wsjtxEnable {
							w.wsjtxFocus = 2
						} else {
							w.wsjtxFocus = 0
						}
					}
					w.updateWSJTXFocus()
					return w, nil
				}
				switch w.wsjtxFocus {
				case 1:
					w.wsjtxHost, _ = w.wsjtxHost.Update(msg)
				case 2:
					w.wsjtxPort, _ = w.wsjtxPort.Update(msg)
				}
			case stepTimezone:
				if k.String() == "ctrl+s" || k.String() == "\x13" {
					w.step = stepSummary
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
		msg := fmt.Sprintf("Terminal too small: %dx%d (min 75x24)\n\nPress F10 to quit",
			w.width, w.height)
		return tea.NewView(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(msg))
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
	h, tw := w.clampedDims()
	center := lipgloss.NewStyle().Width(tw).Align(lipgloss.Center)

	// Compose top section: banner + gap + indicator + body
	top := lipgloss.JoinVertical(lipgloss.Center,
		center.Render(w.banner()),
		"",
		center.Render(w.stepIndicator()),
		center.Render(body),
	)

	// Pad to leave exactly one row for the help bar at the bottom
	contentH := h - lipgloss.Height(help)
	top = fillBody(top, contentH)

	return lipgloss.JoinVertical(lipgloss.Left, top, help)
}

// ── Banner ───────────────────────────────────────────────────────

func (w *Wizard) banner() string {
	ver := version.Resolved()
	name := S.WizardAccent.Render("CQOps v" + ver)
	tag := S.WizardDim.Render("Portable Ham Radio Logger")

	// OSC-8 wraps styled text — more compatible with some terminals.
	// The style is rendered first so ANSI codes are inside the OSC-8 wrapper.
	gh := osc8Link("https://github.com/szporwolik/cqops",
		S.WizardDim.Render("github.com/szporwolik/cqops"))

	return lipgloss.JoinVertical(lipgloss.Center,
		name+"  —  "+tag,
		gh,
	)
}

// ── Step indicator ───────────────────────────────────────────────

func (w *Wizard) stepIndicator() string {
	current := int(w.step) + 1
	total := int(stepCount)
	return S.Title.Render(fmt.Sprintf("First time wizard — Step %d/%d", current, total))
}

func (w *Wizard) updateWSJTXFocus() {
	w.wsjtxHost.Blur()
	w.wsjtxPort.Blur()
	switch w.wsjtxFocus {
	case 1:
		w.wsjtxHost.Focus()
	case 2:
		w.wsjtxPort.Focus()
	}
}

// ── Step views ───────────────────────────────────────────────────

func (w *Wizard) viewStation() string {
	body := w.wizardFormBox().Render(w.station.View().Content)
	help := HelpStyle.Render("Ctrl+S save & next  |  Tab/↓ next field  |  Shift+Tab/↑ previous  |  F10 quit")
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewRig() string {
	body := w.wizardFormBox().Render(w.rigForm.View().Content)
	help := HelpStyle.Render("Ctrl+S save & next  |  Space on flrig toggles  |  Tab/↓ next  |  Esc back  |  F10 quit")
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewWSJTX() string {
	labelW := lipgloss.NewStyle().Width(22).Foreground(P.TextMuted)

	checkbox := "[ ]"
	cbStyle := DimStyle
	if w.wsjtxEnable {
		checkbox = "[x]"
		cbStyle = S.WizardSelected
	}
	if w.wsjtxFocus == 0 {
		checkbox = cursorStyle.Render(checkbox)
	}

	var inner strings.Builder
	inner.WriteString(labelW.Render("WSJT-X:"))
	inner.WriteString(cbStyle.Render(checkbox))
	inner.WriteString("\n")

	if w.wsjtxEnable {
		inner.WriteString("\n")
		inner.WriteString(labelW.Render("UDP Host:"))
		inner.WriteString(inputStyle.Render(w.wsjtxHost.View()))
		inner.WriteString("\n\n")
		inner.WriteString(labelW.Render("UDP Port:"))
		inner.WriteString(inputStyle.Render(w.wsjtxPort.View()))
		inner.WriteString("\n\n")
		inner.WriteString(DimStyle.Render("WSJT-X → Settings → Reporting → UDP Server"))
	}

	body := w.wizardFormBox().Render(inner.String())
	help := HelpStyle.Render("Ctrl+S save & next  |  Space toggle  |  Tab/↓ next  |  Esc back  |  F10 quit")
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewTimezone() string {
	selectedStyle := S.WizardSelected
	dimStyle := S.WizardDim
	detectedIdx := config.SystemTimezoneIndex()

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

	var lines []string
	for i := start; i < end; i++ {
		tz := config.Timezones[i]
		if i == w.tzIndex {
			lines = append(lines, selectedStyle.Render("> "+tz))
		} else {
			lines = append(lines, "  "+tz)
		}
	}

	// Detected timezone info below the list
	detected := config.Timezones[detectedIdx]
	info := dimStyle.Render("System detected: " + detected)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"",
		info,
	)

	body := w.wizardFormBox().Render(inner)
	help := HelpStyle.Render("↑↓ choose  |  Ctrl+S next  |  Esc back  |  F10 quit")
	return w.wizardLayout(body, help)
}

func (w *Wizard) viewSummary() string {
	inner := lipgloss.JoinVertical(lipgloss.Left,
		S.WizardHeader.Render("Configuration ready"),
		"",
		DimStyle.Render("Your configuration file is almost complete."),
		"",
		DimStyle.Render("We recommend visiting the Configuration menu after"),
		DimStyle.Render("starting the program to enter:"),
		"",
		DimStyle.Render("  • QRZ.com credentials (Callbook)"),
		DimStyle.Render("  • Wavelog integration settings"),
		"",
		S.WizardAccent.Render("Press Ctrl+S to generate the configuration"),
		S.WizardAccent.Render("file and start the program."),
	)

	body := w.wizardFormBox().Render(inner)
	help := HelpStyle.Render("Ctrl+S save & start  |  Esc back  |  F10 quit")
	return w.wizardLayout(body, help)
}

// ── Helpers ──────────────────────────────────────────────────────

func (w *Wizard) handleEnter() tea.Cmd {
	w.App.Config.Timezone = config.Timezones[w.tzIndex]
	applog.Info("Wizard: timezone selected", "timezone", config.Timezones[w.tzIndex])
	w.saveConfig()
	w.Completed = true
	return tea.Quit
}

func (w *Wizard) saveConfig() {
	cs, op, gr, sotaRef, potaRef, wwffRef := w.station.Values()
	rig, ant, pwr := w.rigForm.Values()
	flrigEnabled, flrigHost, flrigPort := w.rigForm.FlrigValues()

	if op == "" {
		op = cs
	}

	w.App.Config.Rigs = map[string]config.RigPreset{
		"default": {
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
		w.App.Config.WSJTX.UDPPort = 2233
	}

	w.App.Config.Logbooks["default"] = config.Logbook{
		Description: "Default station logbook",
		Station: config.Station{
			Callsign: cs,
			Operator: op,
			Grid:     gr,
			Rig:      rig,
			Antenna:  ant,
			Power:    pwr,
			RigName:  "default",
			SOTARef:  sotaRef,
			POTARef:  potaRef,
			WWFFRef:  wwffRef,
		},
	}

	lb := w.App.Config.Logbooks["default"]
	w.App.Logbook = &lb

	applog.InfoDetail("Wizard completed", fmt.Sprintf("call=%s rig=%s flrig=%v wsjtx=%v tz=%s",
		cs, rig, flrigEnabled, w.wsjtxEnable, config.Timezones[w.tzIndex]))
}
