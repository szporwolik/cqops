package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

type wizardStep int

const (
	stepStation wizardStep = iota
	stepRig
	stepTimezone
)

type Wizard struct {
	App     *app.App
	step    wizardStep
	station *StationForm
	rigForm *RigForm
	tzIndex int
	toasts  *ToastQueue
	width   int
	height  int
}

func NewWizard(a *app.App) *Wizard {
	applog.Info("Wizard started — first-run setup")
	return &Wizard{
		App:     a,
		step:    stepStation,
		station: NewStationForm("SP9MOA", "", "KO00ca"),
		rigForm: NewRigForm("Xiegu G90", "EFHW 20.5m", "100"),
		tzIndex: config.SystemTimezoneIndex(),
		toasts:  NewToastQueue(),
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

	case tea.KeyMsg:
		k := msg
		switch {
		case k.String() == "ctrl+c" || k.String() == "ctrl+q":
			return w, tea.Quit

		default:
			switch w.step {
			case stepStation:
				if cmd := w.station.HandleKey(msg); cmd != nil {
					cs, _, gr := w.station.Values()
					if cs == "" {
						w.toasts.Error("Callsign is required")
						applog.Warn("Wizard validation: callsign missing")
						return w, nil
					}
					if gr == "" {
						w.toasts.Error("Grid locator is required")
						applog.Warn("Wizard validation: grid locator missing")
						return w, nil
					}
					w.step = stepRig
					applog.InfoDetail("Wizard station step completed", fmt.Sprintf("callsign=%s grid=%s", cs, gr))
					return w, nil
				}
			case stepRig:
				if cmd := w.rigForm.HandleKey(msg); cmd != nil {
					rig, _, _ := w.rigForm.Values()
					if rig == "" {
						w.toasts.Error("Rig model is required")
						applog.Warn("Wizard validation: rig model missing")
						return w, nil
					}
					if w.rigForm.FlrigEnabled {
						_, host, port := w.rigForm.FlrigValues()
						if strings.TrimSpace(host) == "" {
							w.toasts.Error("Flrig host is required")
							applog.Warn("Wizard validation: flrig host missing")
							return w, nil
						}
						if strings.TrimSpace(port) == "" {
							w.toasts.Error("Flrig port is required")
							applog.Warn("Wizard validation: flrig port missing")
							return w, nil
						}
					}
					w.step = stepTimezone
					w.tzIndex = config.SystemTimezoneIndex()
					flrigOn, _, _ := w.rigForm.FlrigValues()
					applog.InfoDetail("Wizard rig step completed", fmt.Sprintf("rig=%s flrig=%v", rig, flrigOn))
					return w, nil
				}
			case stepTimezone:
				if k.String() == "enter" {
					return w, w.handleEnter()
				}
				if msg.Type == tea.KeyUp || k.String() == "up" || k.String() == "k" {
					if w.tzIndex > 0 {
						w.tzIndex--
					}
				}
				if msg.Type == tea.KeyDown || k.String() == "down" || k.String() == "j" {
					if w.tzIndex < len(config.Timezones)-1 {
						w.tzIndex++
					}
				}
			}
		}
	}

	return w, nil
}

func (w *Wizard) View() string {
	switch w.step {
	case stepStation:
		return w.viewStation()
	case stepRig:
		return w.viewRig()
	case stepTimezone:
		return w.viewTimezone()
	}
	return ""
}

func (w *Wizard) logoHeader() string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 2).
		Align(lipgloss.Center)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("CQOPS")
	motto := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("229")).Render("Less clicking. More radio.")
	url := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
		"\x1b]8;;https://app.cqops.com\x1b\\https://app.cqops.com\x1b]8;;\x1b\\",
	)

	inner := lipgloss.JoinVertical(lipgloss.Center,
		title+"  —  "+motto,
		url,
	)

	return box.Render(inner)
}

func (w *Wizard) viewStation() string {
	var b strings.Builder
	b.WriteString(w.logoHeader())
	b.WriteString("\n\n")

	b.WriteString(w.station.View())

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Ctrl+S to save  |  Enter/Tab/↓ to next  |  Shift+Tab/↑ to previous  |  Ctrl+Q to quit"))
	return b.String()
}

func (w *Wizard) viewRig() string {
	var b strings.Builder
	b.WriteString(w.logoHeader())
	b.WriteString("\n\n")

	b.WriteString(w.rigForm.View())

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Ctrl+S to save  |  Space to toggle checkbox  |  Enter/Tab/↓ to next  |  Shift+Tab/↑ to previous  |  Ctrl+Q to quit"))
	return b.String()
}

func (w *Wizard) viewTimezone() string {
	var b strings.Builder
	b.WriteString(w.logoHeader())
	b.WriteString("\n\n")

	b.WriteString("Select your timezone:")
	b.WriteString("\n\n")

	detectedIdx := config.SystemTimezoneIndex()
	detectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)

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
		marker := "   "
		if i == w.tzIndex {
			marker = " > "
		}

		line := fmt.Sprintf("%s%s", marker, config.Timezones[i])
		if i == detectedIdx {
			line += " " + detectedStyle.Render("← detected")
		}

		if i == w.tzIndex {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓ to choose  |  Enter to save & finish  |  Ctrl+Q to quit"))
	return b.String()
}

func (w *Wizard) handleEnter() tea.Cmd {
	w.App.Config.Timezone = config.Timezones[w.tzIndex]
	applog.Info("Wizard timezone selected", "timezone", config.Timezones[w.tzIndex])
	w.saveConfig()
	return tea.Quit
}

func (w *Wizard) saveConfig() {
	cs, op, gr := w.station.Values()
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
		},
	}

	lb := w.App.Config.Logbooks["default"]
	w.App.Logbook = &lb

	applog.InfoDetail("Wizard completed — config saved", fmt.Sprintf("callsign=%s rig=%s flrig=%v tz=%s", cs, rig, flrigEnabled, config.Timezones[w.tzIndex]))
}
