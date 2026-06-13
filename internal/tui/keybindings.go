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
			key.WithKeys("insert", "ctrl+l"),
			key.WithHelp("Ins", "QRZ Lookup"),
		),
		Retain: key.NewBinding(
			key.WithKeys("ctrl+r", "space"),
			key.WithHelp("Space", "Retain"),
		),
		FocusCall: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("", ""),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "Next"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", "Prev"),
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
			key.WithHelp("Enter", "Save QSO"),
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
	// Only show F10 Quit in the footer help bar.
	// F1/F2/F5/F8/F9 shortcuts are displayed in the tab labels.
	return []key.Binding{m.keys.Quit}
}
