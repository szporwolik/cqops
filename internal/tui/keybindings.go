package tui

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap holds all key bindings for the application.
type KeyMap struct {
	Quit      key.Binding
	QSOForm   key.Binding
	Partner   key.Binding
	LogEditor key.Binding
	Config    key.Binding
	Logs      key.Binding
	Save      key.Binding
	Delete    key.Binding
	Lookup    key.Binding
	Retain    key.Binding
	FocusCall key.Binding
	NextField key.Binding
	PrevField key.Binding
	CycleUp   key.Binding
	CycleDown key.Binding
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Confirm   key.Binding
	Cancel    key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("f10"),
			key.WithHelp("F10", "Quit"),
		),
		QSOForm: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("F1", "QSO Form"),
		),
		Partner: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("F2", "Partner"),
		),
		LogEditor: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("F5", "Log Editor"),
		),
		Config: key.NewBinding(
			key.WithKeys("f8"),
			key.WithHelp("F8", "Config"),
		),
		Logs: key.NewBinding(
			key.WithKeys("f9"),
			key.WithHelp("F9", "Logs"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("Ctrl+S", "Save"),
		),
		Delete: key.NewBinding(
			key.WithKeys("delete"),
			key.WithHelp("Del", "Clear"),
		),
		Lookup: key.NewBinding(
			key.WithKeys("insert"),
			key.WithHelp("Ins", "Callbook"),
		),
		Retain: key.NewBinding(
			key.WithKeys("ctrl+r", "space"),
			key.WithHelp("Space", "Toggle retain"),
		),
		FocusCall: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("", ""),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab", "down"),
			key.WithHelp("↓/Tab", "Next"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab", "up"),
			key.WithHelp("↑/S-Tab", "Prev"),
		),
		CycleUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("PgUp", "Cycle +"),
		),
		CycleDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("PgDn", "Cycle −"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "Scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "Scroll down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "Log QSO"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("Y", "Confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("N", "Cancel"),
		),
	}
}

// ActiveBindings returns the currently visible key bindings based on app state.
func (m *Model) ActiveBindings() []key.Binding {
	var bindings []key.Binding

	// QSO form — show editing shortcuts when no sub-model is active
	if !m.isSubmodelActive() {
		bindings = append(bindings,
			m.keys.Enter,
			m.keys.NextField,
			m.keys.PrevField,
			m.keys.Lookup,
			m.keys.Delete,
		)
	}

	// F10 Quit always visible, always last
	bindings = append(bindings, m.keys.Quit)

	return bindings
}
