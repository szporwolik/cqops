package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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
	ri.Placeholder = rigPlaceholder
	ri.Focus()
	ri.Prompt = ""

	an := textinput.New()
	an.CharLimit = 30
	an.Placeholder = antennaPlaceholder
	an.Prompt = ""

	pw := textinput.New()
	pw.CharLimit = 10
	pw.Placeholder = powerPlaceholder
	pw.Prompt = ""

	fh := textinput.New()
	fh.CharLimit = 40
	fh.Placeholder = "localhost"
	fh.Prompt = ""

	fp := textinput.New()
	fp.CharLimit = 6
	fp.Placeholder = "12345"
	fp.Prompt = ""

	return &RigForm{
		Rig:       ri,
		Antenna:   an,
		Power:     pw,
		FlrigHost: fh,
		FlrigPort: fp,
		focus:     rigFieldRig,
	}
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
	next := (f.focus + 1) % rigFieldEnd
	if !f.FlrigEnabled && (next == rigFieldFlrigHost || next == rigFieldFlrigPort) {
		next = (next + 2) % rigFieldEnd
	}
	f.focus = next
	f.focusField()
}

func (f *RigForm) PrevInput() {
	f.blurAll()
	prev := f.focus
	if prev == 0 {
		prev = rigFieldEnd
	}
	prev--
	if !f.FlrigEnabled && (prev == rigFieldFlrigPort || prev == rigFieldFlrigHost) {
		if prev == rigFieldFlrigPort {
			prev = rigFieldFlrig - 1
		}
		if prev == rigFieldFlrigHost {
			prev = rigFieldFlrig - 1
		}
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
	var b strings.Builder
	b.WriteString(formLabelStyle.Render("Rig (radio):"))
	b.WriteString(inputStyle.Render(f.Rig.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("Antenna:"))
	b.WriteString(inputStyle.Render(f.Antenna.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("Power (W):"))
	b.WriteString(inputStyle.Render(f.Power.View()))
	b.WriteString("\n\n")

	checkbox := "[ ]"
	if f.FlrigEnabled {
		checkbox = "[x]"
	}
	if f.focus == rigFieldFlrig {
		checkbox = cursorStyle.Render(checkbox)
	}
	b.WriteString(formLabelStyle.Render("Use flrig:"))
	b.WriteString(checkbox)

	if f.FlrigEnabled {
		b.WriteString("\n\n")
		b.WriteString(formLabelStyle.Render("Flrig host:"))
		b.WriteString(inputStyle.Render(f.FlrigHost.View()))
		b.WriteString("\n\n")
		b.WriteString(formLabelStyle.Render("Flrig port:"))
		b.WriteString(inputStyle.Render(f.FlrigPort.View()))
	}

	return tea.NewView(b.String())
}

func (f *RigForm) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg

	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
	}

	if f.focus == rigFieldFlrig && (k.String() == " " || (k.String() == "enter" && !f.FlrigEnabled)) {
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
