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
	rigFieldName rigFormField = iota
	rigFieldRig
	rigFieldAntenna
	rigFieldPower
	rigFieldFlrig
	rigFieldFlrigHost
	rigFieldFlrigPort
	rigFieldWsjtx
	rigFieldWsjtxHost
	rigFieldWsjtxPort
	rigFieldEnd
)

type RigForm struct {
	Name         textinput.Model
	Rig          textinput.Model
	Antenna      textinput.Model
	Power        textinput.Model
	FlrigEnabled bool
	FlrigHost    textinput.Model
	FlrigPort    textinput.Model
	WsjtxEnabled bool
	WsjtxHost    textinput.Model
	WsjtxPort    textinput.Model
	focus        rigFormField
	width        int // terminal width for responsive layout
}

func NewRigForm(rigPlaceholder, antennaPlaceholder, powerPlaceholder string) *RigForm {
	nm := newTextinput()
	nm.CharLimit = 30
	nm.SetWidth(28)
	nm.Placeholder = "e.g. Home, Portable"
	nm.Focus()

	ri := newTextinput()
	ri.CharLimit = 30
	ri.SetWidth(28)
	ri.Placeholder = rigPlaceholder

	an := newTextinput()
	an.CharLimit = 30
	an.SetWidth(28)
	an.Placeholder = antennaPlaceholder

	pw := newTextinput()
	pw.CharLimit = 10
	pw.SetWidth(28)
	pw.Placeholder = powerPlaceholder

	fh := newTextinput()
	fh.CharLimit = 40
	fh.SetWidth(28)
	fh.Placeholder = "localhost"

	fp := newTextinput()
	fp.CharLimit = 6
	fp.SetWidth(28)
	fp.Placeholder = "12345"

	wh := newTextinput()
	wh.CharLimit = 40
	wh.SetWidth(28)
	wh.Placeholder = "127.0.0.1"

	wp := newTextinput()
	wp.CharLimit = 6
	wp.SetWidth(28)
	wp.Placeholder = "2233"

	rf := &RigForm{
		Name:      nm,
		Rig:       ri,
		Antenna:   an,
		Power:     pw,
		FlrigHost: fh,
		FlrigPort: fp,
		WsjtxHost: wh,
		WsjtxPort: wp,
		focus:     rigFieldName,
	}
	return rf
}

func (f *RigForm) Update(msg tea.KeyPressMsg) {
	switch f.focus {
	case rigFieldName:
		f.Name, _ = f.Name.Update(msg)
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
	case rigFieldWsjtxHost:
		f.WsjtxHost, _ = f.WsjtxHost.Update(msg)
	case rigFieldWsjtxPort:
		f.WsjtxPort, _ = f.WsjtxPort.Update(msg)
	}
}

func (f *RigForm) NextInput() {
	f.blurAll()
	next := rigFormField(wrapNext(int(f.focus), int(rigFieldEnd)))
	if !f.FlrigEnabled && next == rigFieldFlrigHost {
		next = rigFieldWsjtx // skip flrig host/port, jump to WSJT-X
	}
	if !f.WsjtxEnabled && next == rigFieldWsjtxHost {
		next = rigFieldRig // skip wsjtx host/port, wrap to start
	}
	f.focus = next
	f.focusField()
}

func (f *RigForm) PrevInput() {
	f.blurAll()
	prev := rigFormField(wrapPrev(int(f.focus), int(rigFieldEnd)))
	if !f.WsjtxEnabled && (prev == rigFieldWsjtxPort || prev == rigFieldWsjtxHost) {
		prev = rigFieldWsjtx // skip wsjtx host/port, land on checkbox
	}
	if !f.FlrigEnabled && (prev == rigFieldFlrigPort || prev == rigFieldFlrigHost) {
		prev = rigFieldFlrig // skip flrig host/port, land on checkbox
	}
	f.focus = prev
	f.focusField()
}

func (f *RigForm) blurAll() {
	blurTextinputs(&f.Name, &f.Rig, &f.Antenna, &f.Power, &f.FlrigHost, &f.FlrigPort, &f.WsjtxHost, &f.WsjtxPort)
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
	case rigFieldWsjtxHost:
		f.WsjtxHost.Focus()
	case rigFieldWsjtxPort:
		f.WsjtxPort.Focus()
	}
}

func (f *RigForm) OnLastField() bool {
	if f.WsjtxEnabled {
		return f.focus == rigFieldWsjtxPort
	}
	if f.FlrigEnabled {
		return f.focus == rigFieldFlrigPort
	}
	return f.focus == rigFieldWsjtx
}

func (f *RigForm) Values() (name, rig, antenna, power string) {
	return strings.TrimSpace(f.Name.Value()),
		strings.TrimSpace(f.Rig.Value()),
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

func (f *RigForm) SetValues(name, rig, antenna, power string) {
	f.Name.SetValue(name)
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

func (f *RigForm) WsjtxValues() (enabled bool, host, port string) {
	host = strings.TrimSpace(f.WsjtxHost.Value())
	port = strings.TrimSpace(f.WsjtxPort.Value())
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "2233"
	}
	return f.WsjtxEnabled, host, port
}

func (f *RigForm) SetWsjtx(enabled bool, host, port string) {
	f.WsjtxEnabled = enabled
	if host != "" {
		f.WsjtxHost.SetValue(host)
	} else {
		f.WsjtxHost.SetValue("127.0.0.1")
	}
	if port != "" {
		f.WsjtxPort.SetValue(port)
	} else {
		f.WsjtxPort.SetValue("2233")
	}
}

func (f *RigForm) View() tea.View {
	// Calculate available value width — same pattern as QSO form.
	// labelW: 2-char prefix + 17-char label (FormLabelWide/FormFocusedWide).
	const labelW = 2 + 17
	const maxVW = 40
	availW := f.width
	if availW < 40 {
		availW = 80
	}
	renderField := func(label string, ti *textinput.Model, focused bool) string {
		raw := strings.TrimSpace(ti.Value())

		prefix := "  "
		lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
		vw := availW - labelW - 1 // -1 for separator space
		if vw < 3 {
			vw = 3
		}
		if vw > maxVW {
			vw = maxVW
		}

		if focused {
			prefix = S.FormPrefixOn.Render("> ")
			lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
			ti.SetWidth(vw)
			if lipgloss.Width(raw) > vw {
				ti.SetWidth(vw - 1)
			}
			ti.SetCursor(ti.Position())
		}

		var val string
		if focused {
			val = ti.View()
		} else if raw == "" {
			val = DimStyle.Render("\u2014")
		} else {
			val = ValueStyle.Render(truncateText(raw, vw))
		}
		return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
	}

	var b strings.Builder

	b.WriteString(padOrTrunc(renderField("Name:", &f.Name, f.focus == rigFieldName), availW))
	b.WriteString("\n")
	b.WriteString(padOrTrunc(renderField("Rig model:", &f.Rig, f.focus == rigFieldRig), availW))
	b.WriteString("\n")
	b.WriteString(padOrTrunc(renderField("Antenna (opt):", &f.Antenna, f.focus == rigFieldAntenna), availW))
	b.WriteString("\n")
	b.WriteString(padOrTrunc(renderField("Power W (opt):", &f.Power, f.focus == rigFieldPower), availW))
	b.WriteString("\n")

	// flrig checkbox
	checkbox := "[ ]"
	if f.FlrigEnabled {
		checkbox = "[x]"
	}
	flPrefix := "  "
	flLabel := S.FormLabelWide.Align(lipgloss.Left).Render("Use flrig:")
	if f.focus == rigFieldFlrig {
		flPrefix = S.FormPrefixOn.Render("> ")
		flLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("Use flrig:")
		checkbox = CursorStyle.Render(checkbox)
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, flPrefix, flLabel, " ", checkbox),
		availW))

	if f.FlrigEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField("  Flrig host:", &f.FlrigHost, f.focus == rigFieldFlrigHost), availW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField("  Flrig port:", &f.FlrigPort, f.focus == rigFieldFlrigPort), availW))
	}

	b.WriteString("\n")

	// WSJT-X checkbox
	wsjtxCheckbox := "[ ]"
	if f.WsjtxEnabled {
		wsjtxCheckbox = "[x]"
	}
	wxPrefix := "  "
	wxLabel := S.FormLabelWide.Align(lipgloss.Left).Render("Use WSJT-X:")
	if f.focus == rigFieldWsjtx {
		wxPrefix = S.FormPrefixOn.Render("> ")
		wxLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("Use WSJT-X:")
		wsjtxCheckbox = CursorStyle.Render(wsjtxCheckbox)
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, wxPrefix, wxLabel, " ", wsjtxCheckbox),
		availW))

	if f.WsjtxEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField("  UDP Host:", &f.WsjtxHost, f.focus == rigFieldWsjtxHost), availW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField("  UDP Port:", &f.WsjtxPort, f.focus == rigFieldWsjtxPort), availW))
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

	if f.focus == rigFieldWsjtx && (k.String() == " " || msg.Code == tea.KeySpace) {
		f.WsjtxEnabled = !f.WsjtxEnabled
		if f.WsjtxEnabled {
			if f.WsjtxHost.Value() == "" {
				f.WsjtxHost.SetValue("127.0.0.1")
			}
			if f.WsjtxPort.Value() == "" {
				f.WsjtxPort.SetValue("2233")
			}
		}
		return nil
	}

	if k.String() == "tab" || msg.Code == tea.KeyDown {
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
	nm, rig, _, _ := f.Values()
	if nm == "" {
		return fmt.Errorf("rig name is required")
	}
	if rig == "" {
		return fmt.Errorf("rig model is required")
	}
	return nil
}
