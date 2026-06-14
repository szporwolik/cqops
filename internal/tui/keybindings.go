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
		key.WithKeys("f6"),
		key.WithHelp("F6", "Log Editor"),
		),
		Config: key.NewBinding(
		key.WithKeys("f7"),
		key.WithHelp("F7", "Config"),
		),
		Logs: key.NewBinding(
		key.WithKeys("f8"),
		key.WithHelp("F8", "Logs"),
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

	// Log editor — table navigation shortcuts
	if m.screen == screenLogbookEditor {
		if m.logbookEditor != nil && m.logbookEditor.IsEditing() {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("tab", "down"), key.WithHelp("↓/Tab", "Next")),
				key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/S-Tab", "Prev")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Scroll")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit QSO")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("p"), key.WithHelp("P", "Purge")),
				key.NewBinding(key.WithKeys("w"), key.WithHelp("W", "Wavelog")),
			)
		}
	}

	// Log viewer — scroll keybindings
	if m.screen == screenLogView {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Scroll")),
			key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Page")),
			key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "Top/Bottom")),
			key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Refresh")),
		)
	}

	// Configuration / menu screens — screen-specific keybindings
	if m.screen == screenMainMenu {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Select")),
		)
	}
	if m.screen == screenConfig {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle")),
			key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}
	if m.screen == screenCallbook {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down", "tab"), key.WithHelp("↑↓/Tab", "Navigate")),
			key.NewBinding(key.WithKeys(" ", "enter"), key.WithHelp("Space/Enter", "Toggle")),
			key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}
	if m.screen == screenChooser {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Select")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("E", "Edit")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("C", "Create")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("D", "Delete")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}
	if m.screen == screenRigEdit {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Select")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("E", "Edit")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("C", "Create")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("D", "Delete")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}
	if m.screen == screenIntegration {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down", "tab"), key.WithHelp("↑↓/Tab", "Navigate")),
			key.NewBinding(key.WithKeys(" ", "enter"), key.WithHelp("Space/Enter", "Toggle")),
			key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}

	// F10 Quit always visible, always last
	bindings = append(bindings, m.keys.Quit)

	return bindings
}
