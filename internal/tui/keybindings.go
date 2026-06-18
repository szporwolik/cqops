package tui

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap holds all key bindings for the application.
type KeyMap struct {
	Quit         key.Binding
	QSOForm      key.Binding
	Partner      key.Binding
	PSKReporter  key.Binding
	DXC          key.Binding
	LogEditor    key.Binding
	Config       key.Binding
	Logs         key.Binding
	Save         key.Binding
	Delete       key.Binding
	Lookup       key.Binding
	Retain       key.Binding
	FocusCall    key.Binding
	NextField    key.Binding
	PrevField    key.Binding
	CycleUp      key.Binding
	CycleDown    key.Binding
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Confirm      key.Binding
	Cancel       key.Binding
	CycleLogbook key.Binding
	CycleRig     key.Binding
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
			key.WithHelp("F1", "QSO"),
		),
		Partner: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("F2", "QRZ"),
		),
		PSKReporter: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("F5", "PSK Reporter"),
		),
		DXC: key.NewBinding(
			key.WithKeys("f4"),
			key.WithHelp("F4", "cluster"),
		),
		LogEditor: key.NewBinding(
			key.WithKeys("f7"),
			key.WithHelp("F7", "Editor"),
		),
		Config: key.NewBinding(
			key.WithKeys("f8"),
			key.WithHelp("F8", "CFG"),
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
			key.WithHelp("Ins", "QRZ"),
		),
		Retain: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("Ctrl+R", "Toggle retain"),
		),
		FocusCall: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("", ""),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab", "down"),
			key.WithHelp("↑↓", "Navigate"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab", "up"),
			key.WithHelp("", ""),
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
			key.WithHelp("Enter", "Save"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("Y", "Confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("N", "Cancel"),
		),
		CycleLogbook: key.NewBinding(
			key.WithKeys("ctrl+home"),
			key.WithHelp("C-Home", "Logbook"),
		),
		CycleRig: key.NewBinding(
			key.WithKeys("ctrl+end"),
			key.WithHelp("C-End", "Rig"),
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
			m.keys.Lookup,
			m.keys.Delete,
			m.keys.CycleLogbook,
			m.keys.CycleRig,
		)
	}

	// Log editor — table navigation shortcuts
	if m.screen == screenLogbookEditor {
		if m.logbookEditor != nil && m.logbookEditor.IsEditing() {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("tab", "down", "shift+tab", "up"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Scroll")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit QSO")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("p"), key.WithHelp("P", "Purge")),
				key.NewBinding(key.WithKeys("w"), key.WithHelp("W", "WL upload")),
				key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("C-W", "WL download")),
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
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Accept")),
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
			key.NewBinding(key.WithKeys("up", "down", "tab"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle")),
			key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}
	if m.screen == screenChooser {
		if m.chooser != nil && (m.chooser.mode == chooserEdit || m.chooser.mode == chooserCreate) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("tab", "down", "shift+tab", "up"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Accept")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Activate")),
				key.NewBinding(key.WithKeys("e"), key.WithHelp("E", "Edit")),
				key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		}
	}
	if m.screen == screenRigEdit {
		if m.rigChooser != nil && (m.rigChooser.mode == rigChooserEdit || m.rigChooser.mode == rigChooserCreate) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("tab", "down", "shift+tab", "up"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Activate")),
				key.NewBinding(key.WithKeys("e"), key.WithHelp("E", "Edit")),
				key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		}
	}
	// Image screen — simple navigation.
	if m.screen == screenImage {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("f2", "esc"), key.WithHelp("F2/Esc", "Back to Partner")),
		)
	}
	if m.screen == screenDXC {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Time")),
			key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "Band")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "QSO")),
		)
		if m.rigConnected && !m.wsjtxOnline {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("ctrl+enter"), key.WithHelp("C-Enter", "Tune")),
			)
		}
	}
	// Partner screen — show F2 Photo when image available.
	if m.screen == screenPartner && m.partnerData != nil && m.partnerData.ImageURL != "" {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("f2"), key.WithHelp("F2", "Photo")),
		)
	}
	// PSK Reporter screen.
	if m.screen == screenPSKReporter {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Time")),
			key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "Band")),
			key.NewBinding(key.WithKeys("insert", "delete"), key.WithHelp("Ins/Del", "Mode")),
		)
	}
	if m.screen == screenIntegration {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down", "tab"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle")),
			key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}
	if m.screen == screenNotifications {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Accept")),
			key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
		)
	}

	// F10 Quit always visible, always last
	bindings = append(bindings, m.keys.Quit)

	return bindings
}
