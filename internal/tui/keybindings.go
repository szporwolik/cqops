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
	Ref          key.Binding
	BPL          key.Binding
	CON          key.Binding
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
	CycleContest key.Binding
	Spot         key.Binding
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
			key.WithKeys("f8"),
			key.WithHelp("F8", "Editor"),
		),
		Config: key.NewBinding(
			key.WithKeys("f9"),
			key.WithHelp("F9", "CFG"),
		),
		Logs: key.NewBinding(
			key.WithKeys("ctrl+f9"),
			key.WithHelp("Ctrl+F9", "Logs"),
		),
		Ref: key.NewBinding(
			key.WithKeys("f6"),
			key.WithHelp("F6", "REF"),
		),
		BPL: key.NewBinding(
			key.WithKeys("f7"),
			key.WithHelp("F7", "BPL"),
		),
		// CON: key.NewBinding(
		// 	key.WithKeys("f3"),
		// 	key.WithHelp("F3", "CON"),
		// ),
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
		Spot: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("Ctrl+D", "Spot"),
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
			key.WithKeys("ctrl+l"),
			key.WithHelp("C-L", "Logbook"),
		),
		CycleRig: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("C-R", "Rig"),
		), CycleContest: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("C-C", "Contest"),
		)}
}

// ActiveBindings returns the currently visible key bindings based on app state.
func (m *Model) ActiveBindings() []key.Binding {
	var bindings []key.Binding

	// QSO form — show editing shortcuts when no sub-model is active
	if !m.isSubmodelActive() {
		bindings = append(bindings,
			m.keys.Enter,
			m.keys.NextField,
			m.keys.Save,
			m.keys.Spot,
			m.keys.Lookup,
			m.keys.Delete,
			m.keys.CycleLogbook,
			m.keys.CycleRig,
			m.keys.CycleContest,
		)
	}

	// Log editor — table navigation shortcuts
	if m.screen == screenLogbookEditor {
		if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsEditing() {
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
				key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("C-E", "Export")),
				key.NewBinding(key.WithKeys("ctrl+i"), key.WithHelp("C-I", "Import")),
				m.keys.CycleContest,
			)
			wl := m.App.Logbook.Wavelog
			if wl != nil && wl.Enabled {
				bindings = append(bindings,
					key.NewBinding(key.WithKeys("w"), key.WithHelp("W", "WL upload")),
					key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("C-W", "WL download")),
				)
			}
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
	if m.screen == screenChooser {
		if m.ui.chooser != nil && (m.ui.chooser.mode == chooserEdit || m.ui.chooser.mode == chooserCreate) {
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
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit")),
				key.NewBinding(key.WithKeys("space"), key.WithHelp("Spc", "Activate")),
				key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		}
	}
	if m.screen == screenRigEdit {
		if m.ui.rigChooser != nil && (m.ui.rigChooser.mode == rigChooserEdit || m.ui.rigChooser.mode == rigChooserCreate) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("tab", "down", "shift+tab", "up"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit")),
				key.NewBinding(key.WithKeys("space"), key.WithHelp("Spc", "Activate")),
				key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		}
	}
	if m.screen == screenContest {
		if m.ui.contestChooser != nil && (m.ui.contestChooser.mode == contestEdit || m.ui.contestChooser.mode == contestCreate) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("tab", "down", "shift+tab", "up"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Toggle/Cycle")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else if m.ui.contestChooser != nil && m.ui.contestChooser.mode == contestConfirmDelete {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Confirm")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Cancel")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit")),
				key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Activate")),
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
			key.NewBinding(key.WithKeys(`\`), key.WithHelp("\\", "De Cont")),
			key.NewBinding(key.WithKeys("insert", "delete"), key.WithHelp("Ins/Del", "Mode")),
			key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "Band")),
			key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Time")),
			key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Bksp", "Clear")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "QSO+Tune")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("Spc", "Tune")),
		)
	}
	if m.screen == screenRef {
		if m.ref.searched && len(m.ref.rows) > 0 {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("\u2191\u2193", "Navigate")),
				key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Page")),
				key.NewBinding(key.WithKeys("enter", "insert"), key.WithHelp("Enter/Ins", "Add to QSO")),
				key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Bksp", "Clear")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("enter", "insert"), key.WithHelp("Enter/Ins", "Search")),
				key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Bksp", "Clear")),
			)
		}
	}
	if m.screen == screenBPL {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("\u2191\u2193", "Scroll")),
			key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("\u2190\u2192", "Tabs")),
			key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Page")),
			key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "Top/Bottom")),
			key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("C-E", "Export")),
		)
		if m.rig.connected && !m.wsjtx.online {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys(" "), key.WithHelp("Spc", "Tune")),
			)
		}
	}
	// Partner screen — show F2 Photo when image available.
	if m.screen == screenPartner && m.lookup.partnerData != nil && m.lookup.partnerData.ImageURL != "" {
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
			key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Bksp", "Clear")),
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
