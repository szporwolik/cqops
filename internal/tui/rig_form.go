package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type rigFormField int

const (
	rigFieldRig rigFormField = iota
	rigFieldAntenna
	rigFieldPower
	rigFieldFlrig
	rigFieldFlrigHost
	rigFieldFlrigPort
	rigFieldEnd
)

type RigForm struct {
	Rig          textinput.Model
	Antenna      textinput.Model
	Power        textinput.Model
	FlrigEnabled bool
	FlrigHost    textinput.Model
	FlrigPort    textinput.Model
	focus        rigFormField
}

func NewRigForm(rigPlaceholder, antennaPlaceholder, powerPlaceholder string) *RigForm {
	ri := textinput.New()
	ri.CharLimit = 30
	ri.SetWidth(28)
	ri.Placeholder = rigPlaceholder
	ri.Focus()
	ri.Prompt = ""

	an := textinput.New()
	an.CharLimit = 30
	an.SetWidth(28)
	an.Placeholder = antennaPlaceholder
	an.Prompt = ""

	pw := textinput.New()
	pw.CharLimit = 10
	pw.SetWidth(28)
	pw.Placeholder = powerPlaceholder
	pw.Prompt = ""

	fh := textinput.New()
	fh.CharLimit = 40
	fh.SetWidth(28)
	fh.Placeholder = "localhost"
	fh.Prompt = ""

	fp := textinput.New()
	fp.CharLimit = 6
	fp.SetWidth(28)
	fp.Placeholder = "12345"
	fp.Prompt = ""

	// Apply Surface background BEFORE storing in struct (textinput.Model
	// is a value type — copies made after SetStyles are stale).
	for _, ti := range []*textinput.Model{&ri, &an, &pw, &fh, &fp} {
		s := ti.Styles()
		s.Focused.Text = s.Focused.Text.Background(P.Surface)
		s.Focused.Placeholder = s.Focused.Placeholder.Background(P.Surface)
		s.Focused.Prompt = s.Focused.Prompt.Background(P.Surface)
		s.Blurred.Text = s.Blurred.Text.Background(P.Surface)
		s.Blurred.Placeholder = s.Blurred.Placeholder.Background(P.Surface)
		s.Blurred.Prompt = s.Blurred.Prompt.Background(P.Surface)
		ti.SetStyles(s)
	}

	rf := &RigForm{
		Rig:       ri,
		Antenna:   an,
		Power:     pw,
		FlrigHost: fh,
		FlrigPort: fp,
		focus:     rigFieldRig,
	}
	return rf
}

func (f *RigForm) Update(msg tea.KeyPressMsg) {
	switch f.focus {
	case rigFieldRig:
		f.Rig, _ = f.Rig.Update(msg)
	case rigFieldAntenna:
		f.Antenna, _ = f.Antenna.Update(msg)
	case rigFieldPower:
		f.Power, _ = f.Power.Update(msg)
	case rigFieldFlrigHost:
		f.FlrigHost, _ = f.FlrigHost.Update(msg)
	case rigFieldFlrigPort:
		f.FlrigPort, _ = f.FlrigPort.Update(msg)
	}
}

func (f *RigForm) NextInput() {
	f.blurAll()
	next := f.focus + 1
	if !f.FlrigEnabled && next == rigFieldFlrigHost {
		next = rigFieldRig // skip host/port, wrap to start
	}
	if next >= rigFieldEnd {
		next = rigFieldRig
	}
	f.focus = next
	f.focusField()
}

func (f *RigForm) PrevInput() {
	f.blurAll()
	prev := f.focus
	if prev == rigFieldRig {
		prev = rigFieldEnd
	}
	prev--
	if !f.FlrigEnabled && (prev == rigFieldFlrigPort || prev == rigFieldFlrigHost) {
		prev = rigFieldFlrig // skip host/port, land on checkbox
	}
	f.focus = rigFormField(prev)
	f.focusField()
}

func (f *RigForm) blurAll() {
	f.Rig.Blur()
	f.Antenna.Blur()
	f.Power.Blur()
	f.FlrigHost.Blur()
	f.FlrigPort.Blur()
}

func (f *RigForm) focusField() {
	switch f.focus {
	case rigFieldRig:
		f.Rig.Focus()
	case rigFieldAntenna:
		f.Antenna.Focus()
	case rigFieldPower:
		f.Power.Focus()
	case rigFieldFlrigHost:
		f.FlrigHost.Focus()
	case rigFieldFlrigPort:
		f.FlrigPort.Focus()
	}
}

func (f *RigForm) OnLastField() bool {
	if f.FlrigEnabled {
		return f.focus == rigFieldFlrigPort
	}
	return f.focus == rigFieldFlrig
}

func (f *RigForm) Values() (rig, antenna, power string) {
	return strings.TrimSpace(f.Rig.Value()),
		strings.TrimSpace(f.Antenna.Value()),
		strings.TrimSpace(f.Power.Value())
}

func (f *RigForm) FlrigValues() (enabled bool, host, port string) {
	host = strings.TrimSpace(f.FlrigHost.Value())
	port = strings.TrimSpace(f.FlrigPort.Value())
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "12345"
	}
	return f.FlrigEnabled, host, port
}

func (f *RigForm) FlrigURL() string {
	_, host, port := f.FlrigValues()
	return "http://" + host + ":" + port
}

func (f *RigForm) SetValues(rig, antenna, power string) {
	f.Rig.SetValue(rig)
	f.Antenna.SetValue(antenna)
	f.Power.SetValue(power)
}

func (f *RigForm) SetFlrig(enabled bool, host, port string) {
	f.FlrigEnabled = enabled
	if host != "" {
		f.FlrigHost.SetValue(host)
	} else {
		f.FlrigHost.SetValue("localhost")
	}
	if port != "" {
		f.FlrigPort.SetValue(port)
	} else {
		f.FlrigPort.SetValue("12345")
	}
}

func (f *RigForm) View() tea.View {
	bg := lipgloss.NewStyle().Background(P.Surface)
	renderField := func(label string, ti *textinput.Model, focused bool, w int) string {
		prefix := "  "
		l := LabelStyle.Render(fit(label, 22))
		if focused {
			prefix = CursorStyle.Render("> ")
			l = CursorStyle.Render(fit(label, 22))
		}
		val := ValueStyle.Render(strings.TrimSpace(ti.Value()))
		if focused {
			val = ti.View()
		}
		return prefix + l + bg.Render(" ") + val
	}

	var b strings.Builder

	// Rig
	b.WriteString(menuLine(renderField("Rig (radio):", &f.Rig, f.focus == rigFieldRig, 80), 80))
	b.WriteString("\n")

	// Antenna
	b.WriteString(menuLine(renderField("Antenna (optional):", &f.Antenna, f.focus == rigFieldAntenna, 80), 80))
	b.WriteString("\n")

	// Power
	b.WriteString(menuLine(renderField("Power (W) (optional):", &f.Power, f.focus == rigFieldPower, 80), 80))
	b.WriteString("\n")

	// flrig checkbox
	checkbox := "[ ]"
	if f.FlrigEnabled {
		checkbox = "[x]"
	}
	flPrefix := "  "
	if f.focus == rigFieldFlrig {
		flPrefix = CursorStyle.Render("> ")
		checkbox = CursorStyle.Render(checkbox)
	} else {
		checkbox = bg.Render(checkbox)
	}
	flLabel := LabelStyle.Render(fit("Use flrig:", 22))
	if f.focus == rigFieldFlrig {
		flLabel = CursorStyle.Render(fit("Use flrig:", 22))
	}
	b.WriteString(menuLine(flPrefix+flLabel+bg.Render(" ")+checkbox, 80))

	if f.FlrigEnabled {
		b.WriteString("\n")
		b.WriteString(menuLine(renderField("Flrig host:", &f.FlrigHost, f.focus == rigFieldFlrigHost, 80), 80))
		b.WriteString("\n")
		b.WriteString(menuLine(renderField("Flrig port:", &f.FlrigPort, f.focus == rigFieldFlrigPort, 80), 80))
	}

	return tea.NewView(b.String())
}

func (f *RigForm) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg

	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
	}

	if f.focus == rigFieldFlrig && (k.String() == " " || msg.Code == tea.KeySpace) {
		f.FlrigEnabled = !f.FlrigEnabled
		if f.FlrigEnabled {
			if f.FlrigHost.Value() == "" {
				f.FlrigHost.SetValue("localhost")
			}
			if f.FlrigPort.Value() == "" {
				f.FlrigPort.SetValue("12345")
			}
		}
		return nil
	}

	if k.String() == "tab" || msg.Code == tea.KeyDown || k.String() == "enter" {
		f.NextInput()
		return nil
	}
	if k.String() == "shift+tab" || msg.Code == tea.KeyUp {
		f.PrevInput()
		return nil
	}
	f.Update(msg)
	return nil
}

func (f *RigForm) Validate() error {
	rig, _, _ := f.Values()
	if rig == "" {
		return fmt.Errorf("rig model is required")
	}
	return nil
}
