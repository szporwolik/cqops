package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

type wizardStep int

const (
	stepStation wizardStep = iota
	stepRig
	stepWSJTX
	stepTimezone
	stepCount // sentinel
)

type Wizard struct {
	App         *app.App
	step        wizardStep
	station     *StationForm
	rigForm     *RigForm
	wsjtxEnable bool
	wsjtxHost   string
	wsjtxPort   string
	tzIndex     int
	toasts      *ToastQueue
	width       int
	height      int
}

func NewWizard(a *app.App) *Wizard {
	applog.Info("Wizard started — first-run setup")
	return &Wizard{
		App:         a,
		step:        stepStation,
		station:     NewStationForm("", "", ""),
		rigForm:     NewRigForm("", "", ""),
		wsjtxEnable: true,
		wsjtxHost:   "127.0.0.1",
		wsjtxPort:   "2233",
		tzIndex:     config.SystemTimezoneIndex(),
		toasts:      NewToastQueue(),
	}
}

func (w *Wizard) Init() tea.Cmd {
	return nil
}

func (w *Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height

	case tea.KeyPressMsg:
		k := msg
		switch {
		case k.String() == "ctrl+c" || k.String() == "ctrl+q":
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
						// Rig is optional — allow skipping
					}
					w.step = stepWSJTX
					flrigOn, _, _ := w.rigForm.FlrigValues()
					applog.InfoDetail("Wizard: rig step done", fmt.Sprintf("rig=%s flrig=%v", rig, flrigOn))
					return w, nil
				}
			case stepWSJTX:
				switch k.String() {
				case " ", "enter":
					w.wsjtxEnable = !w.wsjtxEnable
				case "y", "Y":
					w.step = stepTimezone
					w.wsjtxEnable = true
					return w, nil
				case "n", "N":
					w.step = stepTimezone
					w.wsjtxEnable = false
					return w, nil
				case "tab", "down":
					if w.wsjtxEnable {
						// Would toggle to host/port fields
					}
				}
			case stepTimezone:
				if k.String() == "enter" {
					return w, w.handleEnter()
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
			}
		}
	}

	return w, nil
}

func (w *Wizard) View() tea.View {
	v := tea.NewView("")
	switch w.step {
	case stepStation:
		v = tea.NewView(w.viewStation())
	case stepRig:
		v = tea.NewView(w.viewRig())
	case stepWSJTX:
		v = tea.NewView(w.viewWSJTX())
	case stepTimezone:
		v = tea.NewView(w.viewTimezone())
	}
	v.AltScreen = true
	v.WindowTitle = "CQOPS — Setup Wizard"
	return v
}

func (w *Wizard) viewStation() string {
	var b strings.Builder
	b.WriteString(w.banner())
	b.WriteString("\n")
	b.WriteString(w.stepIndicator("Station Setup", "Enter your callsign, grid locator, and optional park references."))
	b.WriteString("\n\n")
	b.WriteString(w.station.View().Content)
	b.WriteString("\n\n")
	b.WriteString(w.helpLine("Ctrl+S save & next  |  Tab/↓ next field  |  Shift+Tab/↑ previous  |  Ctrl+Q quit"))
	return b.String()
}

func (w *Wizard) viewRig() string {
	var b strings.Builder
	b.WriteString(w.banner())
	b.WriteString("\n")
	b.WriteString(w.stepIndicator("Rig & Antenna", "Configure your radio and optionally enable flrig for automatic frequency/mode reading."))
	b.WriteString("\n\n")
	b.WriteString(w.rigForm.View().Content)
	b.WriteString("\n\n")
	b.WriteString(w.helpLine("Ctrl+S save & next  |  Space toggle flrig  |  Tab/↓ next  |  Esc back  |  Ctrl+Q quit"))
	return b.String()
}

func (w *Wizard) viewWSJTX() string {
	var b strings.Builder
	b.WriteString(w.banner())
	b.WriteString("\n")
	b.WriteString(w.stepIndicator("WSJT-X Integration", "Automatically log QSOs from FT8, FT4, and other digital modes."))
	b.WriteString("\n\n")

	status := "[x]"
	label := S.WizardSelected.Render("Enabled")
	if !w.wsjtxEnable {
		status = "[ ]"
		label = S.WizardDim.Render("Disabled")
	}
	b.WriteString(fmt.Sprintf("  WSJT-X: %s %s\n", status, label))
	b.WriteString(fmt.Sprintf("  UDP:    %s:%s\n", w.wsjtxHost, w.wsjtxPort))
	b.WriteString("\n")
	b.WriteString(DimStyle.Render("  Press Space to toggle, Y/N to confirm and continue"))
	b.WriteString("\n")
	b.WriteString(DimStyle.Render("  WSJT-X must be configured to send UDP to this address in its Settings → Reporting tab"))
	b.WriteString("\n\n")
	b.WriteString(w.helpLine("Y accept  |  N skip  |  Space toggle  |  Esc back  |  Ctrl+Q quit"))
	return b.String()
}

func (w *Wizard) viewTimezone() string {
	var b strings.Builder
	b.WriteString(w.banner())
	b.WriteString("\n")
	b.WriteString(w.stepIndicator("Timezone", "Select your local timezone. All QSO times are stored in UTC — this setting controls how dates and times are displayed."))
	b.WriteString("\n\n")

	selectedStyle := S.WizardSelected
	dimStyle := S.WizardDim

	detectedIdx := config.SystemTimezoneIndex()

	// Show 9 entries centered around current selection
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

	for i := start; i < end; i++ {
		tz := config.Timezones[i]
		if i == w.tzIndex {
			b.WriteString("  ")
			b.WriteString(selectedStyle.Render("> " + tz))
		} else {
			b.WriteString("    ")
			b.WriteString(tz)
		}
		if i == detectedIdx {
			b.WriteString(dimStyle.Render("  (detected)"))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.helpLine("↑↓ choose  |  Enter save & finish  |  Esc back  |  Ctrl+Q quit"))
	return b.String()
}

// banner returns the wizard header with project name and tagline.
func (w *Wizard) banner() string {
	border := lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(P.Primary).
		Padding(0, 3).
		Align(lipgloss.Center)

	name := S.WizardAccent.Render("CQOps")
	tag := S.WizardTag.Render("Portable Ham Radio Logger")
	gh := S.WizardDim.Render("github.com/szporwolik/cqops")

	inner := lipgloss.JoinVertical(lipgloss.Center,
		name+"  —  "+tag,
		gh,
	)

	return border.Render(inner)
}

// stepIndicator renders a progress indicator and section description.
func (w *Wizard) stepIndicator(title, desc string) string {
	total := int(stepCount)
	current := int(w.step) + 1

	dots := make([]string, total)
	for i := 0; i < total; i++ {
		if i < current {
			dots[i] = "●"
		} else {
			dots[i] = "○"
		}
	}

	active := S.WizardActive
	inactive := S.WizardInactive

	progress := active.Render(strings.Join(dots[:current], "")) + inactive.Render(strings.Join(dots[current:], ""))

	header := S.WizardHeader.Render(title)
	subtitle := DimStyle.Render(desc)

	return fmt.Sprintf("  %s  %s  Step %d/%d\n  %s", progress, header, current, total, subtitle)
}

func (w *Wizard) helpLine(text string) string {
	return HelpStyle.Render(text)
}

func (w *Wizard) handleEnter() tea.Cmd {
	w.App.Config.Timezone = config.Timezones[w.tzIndex]
	applog.Info("Wizard: timezone selected", "timezone", config.Timezones[w.tzIndex])
	w.saveConfig()
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
		w.App.Config.WSJTX.UDPHost = w.wsjtxHost
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
