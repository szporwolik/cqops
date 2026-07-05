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
	rigFieldBackend
	rigFieldBackendHost
	rigFieldBackendPort
	rigFieldRotor
	rigFieldRotorHost
	rigFieldRotorPort
	rigFieldWsjtx
	rigFieldWsjtxHost
	rigFieldWsjtxPort
	rigFieldEnd
)

// backendOptions maps backend index to label and host/port defaults.
var backendOptions = []struct {
	label       string
	defaultHost string
	defaultPort string
}{
	{"None", "", ""},
	{"Hamlib", "127.0.0.1", "4532"},
	{"Flrig", "localhost", "12345"},
}

// rotorOptions maps rotor backend index to label and defaults.
var rotorOptions = []struct {
	label       string
	defaultHost string
	defaultPort string
}{
	{"None", "", ""},
	{"Hamlib", "127.0.0.1", "4533"},
}

type RigForm struct {
	Name         textinput.Model
	Rig          textinput.Model
	Antenna      textinput.Model
	Power        textinput.Model
	BackendIdx   int // 0=None, 1=Hamlib, 2=Flrig
	BackendHost  textinput.Model
	BackendPort  textinput.Model
	RotorIdx     int // 0=None, 1=Hamlib
	RotorHost    textinput.Model
	RotorPort    textinput.Model
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

	bh := newTextinput()
	bh.CharLimit = 40
	bh.SetWidth(28)
	bh.Placeholder = "127.0.0.1"

	bp := newTextinput()
	bp.CharLimit = 6
	bp.SetWidth(28)
	bp.Placeholder = "4532"

	wh := newTextinput()
	wh.CharLimit = 40
	wh.SetWidth(28)
	wh.Placeholder = "127.0.0.1"

	wp := newTextinput()
	wp.CharLimit = 6
	wp.SetWidth(28)
	wp.Placeholder = "2233"

	rh := newTextinput()
	rh.CharLimit = 40
	rh.SetWidth(28)
	rh.Placeholder = "127.0.0.1"

	rp := newTextinput()
	rp.CharLimit = 6
	rp.SetWidth(28)
	rp.Placeholder = "4533"

	rf := &RigForm{
		Name:        nm,
		Rig:         ri,
		Antenna:     an,
		Power:       pw,
		BackendHost: bh,
		BackendPort: bp,
		RotorHost:   rh,
		RotorPort:   rp,
		WsjtxHost:   wh,
		WsjtxPort:   wp,
		focus:       rigFieldName,
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
	case rigFieldBackendHost:
		f.BackendHost, _ = f.BackendHost.Update(msg)
	case rigFieldBackendPort:
		f.BackendPort, _ = f.BackendPort.Update(msg)
	case rigFieldRotorHost:
		f.RotorHost, _ = f.RotorHost.Update(msg)
	case rigFieldRotorPort:
		f.RotorPort, _ = f.RotorPort.Update(msg)
	case rigFieldWsjtxHost:
		f.WsjtxHost, _ = f.WsjtxHost.Update(msg)
	case rigFieldWsjtxPort:
		f.WsjtxPort, _ = f.WsjtxPort.Update(msg)
	}
}

func (f *RigForm) NextInput() {
	f.blurAll()
	next := rigFormField(wrapNext(int(f.focus), int(rigFieldEnd)))
	if f.BackendIdx == 0 && next == rigFieldBackendHost {
		next = rigFieldRotor // skip radio host/port, jump to rotor
	}
	if f.RotorIdx == 0 && next == rigFieldRotorHost {
		next = rigFieldWsjtx // skip rotor host/port, jump to WSJT-X
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
		prev = rigFieldWsjtx
	}
	if f.RotorIdx == 0 && (prev == rigFieldRotorPort || prev == rigFieldRotorHost) {
		prev = rigFieldRotor
	}
	if f.BackendIdx == 0 && (prev == rigFieldBackendPort || prev == rigFieldBackendHost) {
		prev = rigFieldBackend
	}
	f.focus = prev
	f.focusField()
}

func (f *RigForm) blurAll() {
	blurTextinputs(&f.Name, &f.Rig, &f.Antenna, &f.Power, &f.BackendHost, &f.BackendPort, &f.RotorHost, &f.RotorPort, &f.WsjtxHost, &f.WsjtxPort)
}

func (f *RigForm) focusField() {
	switch f.focus {
	case rigFieldRig:
		f.Rig.Focus()
	case rigFieldAntenna:
		f.Antenna.Focus()
	case rigFieldPower:
		f.Power.Focus()
	case rigFieldBackendHost:
		f.BackendHost.Focus()
	case rigFieldBackendPort:
		f.BackendPort.Focus()
	case rigFieldRotorHost:
		f.RotorHost.Focus()
	case rigFieldRotorPort:
		f.RotorPort.Focus()
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
	if f.RotorIdx != 0 {
		return f.focus == rigFieldRotorPort
	}
	if f.BackendIdx != 0 {
		return f.focus == rigFieldBackendPort
	}
	return f.focus == rigFieldWsjtx
}

func (f *RigForm) FlrigURL() string {
	return "http://" + f.BackendHost.Value() + ":" + f.BackendPort.Value()
}

func (f *RigForm) SetValues(name, rig, antenna, power string) {
	f.Name.SetValue(name)
	f.Rig.SetValue(rig)
	f.Antenna.SetValue(antenna)
	f.Power.SetValue(power)
}

func (f *RigForm) Values() (name, rig, antenna, power string) {
	return strings.TrimSpace(f.Name.Value()),
		strings.TrimSpace(f.Rig.Value()),
		strings.TrimSpace(f.Antenna.Value()),
		strings.TrimSpace(f.Power.Value())
}

// SetBackend configures the radio control select. idx: 0=None, 1=Hamlib, 2=Flrig.
func (f *RigForm) SetBackend(idx int, host, port string) {
	if idx < 0 || idx >= len(backendOptions) {
		idx = 0
	}
	f.BackendIdx = idx
	if host != "" {
		f.BackendHost.SetValue(host)
	} else {
		f.BackendHost.SetValue(backendOptions[idx].defaultHost)
	}
	if port != "" {
		f.BackendPort.SetValue(port)
	} else {
		f.BackendPort.SetValue(backendOptions[idx].defaultPort)
	}
}

// SetRotor configures the rotor control select. idx: 0=None, 1=Hamlib.
func (f *RigForm) SetRotor(idx int, host, port string) {
	if idx < 0 || idx >= len(rotorOptions) {
		idx = 0
	}
	f.RotorIdx = idx
	if host != "" {
		f.RotorHost.SetValue(host)
	} else {
		f.RotorHost.SetValue(rotorOptions[idx].defaultHost)
	}
	if port != "" {
		f.RotorPort.SetValue(port)
	} else {
		f.RotorPort.SetValue(rotorOptions[idx].defaultPort)
	}
}

// RotorValues returns the selected rotor backend and host/port.
func (f *RigForm) RotorValues() (backend string, host, port string) {
	host = strings.TrimSpace(f.RotorHost.Value())
	port = strings.TrimSpace(f.RotorPort.Value())
	if f.RotorIdx == 1 {
		backend = "hamlib"
		if host == "" {
			host = "127.0.0.1"
		}
		if port == "" {
			port = "4533"
		}
	}
	return backend, host, port
}
func (f *RigForm) BackendValues() (backend string, host, port string) {
	host = strings.TrimSpace(f.BackendHost.Value())
	port = strings.TrimSpace(f.BackendPort.Value())
	if f.BackendIdx < 0 || f.BackendIdx >= len(backendOptions) {
		return "", host, port
	}
	switch f.BackendIdx {
	case 1:
		backend = "hamlib"
		if host == "" {
			host = "127.0.0.1"
		}
		if port == "" {
			port = "4532"
		}
	case 2:
		backend = "flrig"
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "12345"
		}
	}
	return backend, host, port
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
	b.WriteString(padOrTrunc(renderField("Rig model (opt):", &f.Rig, f.focus == rigFieldRig), availW))
	b.WriteString("\n")
	b.WriteString(padOrTrunc(renderField("Antenna (opt):", &f.Antenna, f.focus == rigFieldAntenna), availW))
	b.WriteString("\n")
	b.WriteString(padOrTrunc(renderField("Power W (opt):", &f.Power, f.focus == rigFieldPower), availW))
	b.WriteString("\n")

	// Radio control — cycles None → Hamlib → Flrig on Space.
	backendLabel := backendOptions[f.BackendIdx].label
	bePrefix := "  "
	beLbl := S.FormLabelWide.Align(lipgloss.Left).Render("Radio control:")
	if f.focus == rigFieldBackend {
		bePrefix = S.FormPrefixOn.Render("> ")
		beLbl = S.FormFocusedWide.Align(lipgloss.Left).Render("Radio control:")
		backendLabel = CursorStyle.Render(backendLabel) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, bePrefix, beLbl, " ", backendLabel),
		availW))

	if f.BackendIdx != 0 {
		hostLabel := fmt.Sprintf("  %s host:", backendOptions[f.BackendIdx].label)
		portLabel := fmt.Sprintf("  %s port:", backendOptions[f.BackendIdx].label)
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField(hostLabel, &f.BackendHost, f.focus == rigFieldBackendHost), availW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField(portLabel, &f.BackendPort, f.focus == rigFieldBackendPort), availW))
	}

	b.WriteString("\n")

	// Rotor control — cycles None → Hamlib on Space.
	rotorLabel := rotorOptions[f.RotorIdx].label
	roPrefix := "  "
	roLbl := S.FormLabelWide.Align(lipgloss.Left).Render("Rotator control:")
	if f.focus == rigFieldRotor {
		roPrefix = S.FormPrefixOn.Render("> ")
		roLbl = S.FormFocusedWide.Align(lipgloss.Left).Render("Rotator control:")
		rotorLabel = CursorStyle.Render(rotorLabel) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, roPrefix, roLbl, " ", rotorLabel),
		availW))

	if f.RotorIdx != 0 {
		hostLabel := fmt.Sprintf("  %s host:", rotorOptions[f.RotorIdx].label)
		portLabel := fmt.Sprintf("  %s port:", rotorOptions[f.RotorIdx].label)
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField(hostLabel, &f.RotorHost, f.focus == rigFieldRotorHost), availW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(renderField(portLabel, &f.RotorPort, f.focus == rigFieldRotorPort), availW))
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
		wsjtxCheckbox = CursorStyle.Render(wsjtxCheckbox) + " " + DimStyle.Render("(Space)")
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

// HandlePaste forwards clipboard-paste content to the currently focused
// text input. Non-text focus states (backend selector, rotor selector,
// WSJT-X checkbox) are ignored.
func (f *RigForm) HandlePaste(content string) tea.Cmd {
	msg := tea.PasteMsg{Content: content}
	switch f.focus {
	case rigFieldName:
		f.Name, _ = f.Name.Update(msg)
	case rigFieldRig:
		f.Rig, _ = f.Rig.Update(msg)
	case rigFieldAntenna:
		f.Antenna, _ = f.Antenna.Update(msg)
	case rigFieldPower:
		f.Power, _ = f.Power.Update(msg)
	case rigFieldBackendHost:
		f.BackendHost, _ = f.BackendHost.Update(msg)
	case rigFieldBackendPort:
		f.BackendPort, _ = f.BackendPort.Update(msg)
	case rigFieldRotorHost:
		f.RotorHost, _ = f.RotorHost.Update(msg)
	case rigFieldRotorPort:
		f.RotorPort, _ = f.RotorPort.Update(msg)
	case rigFieldWsjtxHost:
		f.WsjtxHost, _ = f.WsjtxHost.Update(msg)
	case rigFieldWsjtxPort:
		f.WsjtxPort, _ = f.WsjtxPort.Update(msg)
	default:
		return nil // Non-text focus — no paste target.
	}
	return nil
}

func (f *RigForm) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg

	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
	}

	if f.focus == rigFieldBackend && (k.String() == " " || msg.Code == tea.KeySpace) {
		f.BackendIdx = (f.BackendIdx + 1) % len(backendOptions)
		if f.BackendIdx != 0 {
			opt := backendOptions[f.BackendIdx]
			f.BackendHost.SetValue(opt.defaultHost)
			f.BackendPort.SetValue(opt.defaultPort)
		}
		return nil
	}

	if f.focus == rigFieldRotor && (k.String() == " " || msg.Code == tea.KeySpace) {
		f.RotorIdx = (f.RotorIdx + 1) % len(rotorOptions)
		if f.RotorIdx != 0 {
			opt := rotorOptions[f.RotorIdx]
			f.RotorHost.SetValue(opt.defaultHost)
			f.RotorPort.SetValue(opt.defaultPort)
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
	nm, _, _, _ := f.Values()
	if nm == "" {
		return fmt.Errorf("rig name is required")
	}
	return nil
}
