package tui

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap holds all key bindings for the application.
type KeyMap struct {
	Quit          key.Binding
	QSOForm       key.Binding
	Partner       key.Binding
	PSKReporter   key.Binding
	DXC           key.Binding
	LogEditor     key.Binding
	Config        key.Binding
	Logs          key.Binding
	Ref           key.Binding
	BPL           key.Binding
	Save          key.Binding
	Delete        key.Binding
	Lookup        key.Binding
	Retain        key.Binding
	FocusCall     key.Binding
	NextField     key.Binding
	PrevField     key.Binding
	NextRow       key.Binding
	PrevRow       key.Binding
	CycleUp       key.Binding
	CycleDown     key.Binding
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Confirm       key.Binding
	Cancel        key.Binding
	CycleLogbook  key.Binding
	CycleRig      key.Binding
	CycleContest  key.Binding
	CycleOperator key.Binding
	Spot          key.Binding
	Help          key.Binding
	RotorLeft     key.Binding
	RotorRight    key.Binding
	RotorUp       key.Binding
	RotorDown     key.Binding
	RotorBearing  key.Binding
	RotorStop     key.Binding
	DXCSpotFill   key.Binding
	RigTuneUp     key.Binding
	RigTuneDown   key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("f10"),
			key.WithHelp("F10", "Quit"),
		),
		QSOForm: key.NewBinding(
			key.WithKeys("f1", "alt+1"),
			key.WithHelp("F1/Alt+1", "QSO"),
		),
		Partner: key.NewBinding(
			key.WithKeys("f2", "alt+2"),
			key.WithHelp("F2/Alt+2", "QRZ"),
		),
		PSKReporter: key.NewBinding(
			key.WithKeys("f5", "alt+5"),
			key.WithHelp("F5/Alt+5", "PSK Reporter"),
		),
		DXC: key.NewBinding(
			key.WithKeys("f4", "alt+4"),
			key.WithHelp("F4/Alt+4", "cluster"),
		),
		LogEditor: key.NewBinding(
			key.WithKeys("f8", "alt+8"),
			key.WithHelp("F8/Alt+8", "Editor"),
		),
		Config: key.NewBinding(
			key.WithKeys("f9", "alt+9"),
			key.WithHelp("F9/Alt+9", "CFG"),
		),
		Logs: key.NewBinding(
			key.WithKeys("ctrl+f9", "ctrl+alt+9"),
		),
		Ref: key.NewBinding(
			key.WithKeys("f6", "alt+6"),
			key.WithHelp("F6/Alt+6", "REF"),
		),
		BPL: key.NewBinding(
			key.WithKeys("f7", "alt+7"),
			key.WithHelp("F7/Alt+7", "BPL"),
		),
		Save: key.NewBinding(
			key.WithKeys(), // Enter logs QSO; Ctrl+S is Spot
			key.WithHelp("", ""),
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
			key.WithKeys("ctrl+s"),
			key.WithHelp("Ctrl+S", "Spot"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "Help"),
		),
		RotorLeft: key.NewBinding(
			key.WithKeys("alt+,"),
			key.WithHelp("Alt+,", "Az −5°"),
		),
		RotorRight: key.NewBinding(
			key.WithKeys("alt+."),
			key.WithHelp("Alt+.", "Az +5°"),
		),
		RotorUp: key.NewBinding(
			key.WithKeys("alt+;"),
			key.WithHelp("Alt+;", "El +5°"),
		),
		RotorDown: key.NewBinding(
			key.WithKeys("alt+'"),
			key.WithHelp("Alt+'", "El −5°"),
		),
		RotorBearing: key.NewBinding(
			key.WithKeys("alt+\\"),
			key.WithHelp("Alt+\\", "→ Path"),
		),
		RotorStop: key.NewBinding(
			key.WithKeys("alt+/"),
			key.WithHelp("Alt+/", "Stop"),
		),
		Retain: key.NewBinding(
			key.WithKeys(),
			key.WithHelp("", ""),
		),
		FocusCall: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("", ""),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "Col Right"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", "Col Left"),
		),
		NextRow: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "Next row"),
		),
		PrevRow: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "Prev row"),
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
		CycleLogbook: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("Ctrl+L", "Logbook"),
		),
		CycleRig: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("Ctrl+R", "Rig"),
		), CycleContest: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("Ctrl+C", "Contest"),
		), CycleOperator: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("Ctrl+O", "Operator"),
		), DXCSpotFill: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("Ctrl+P", "Spot→Call"),
		), RigTuneUp: key.NewBinding(
			key.WithKeys("ctrl+up"),
			key.WithHelp("Ctrl+↑", "Rig +step"),
		), RigTuneDown: key.NewBinding(
			key.WithKeys("ctrl+down"),
			key.WithHelp("Ctrl+↓", "Rig −step"),
		)}
}

// ShortHelp returns keybindings shown in the mini help bar at the bottom
// of the screen. Delegates to the context-sensitive ActiveBindings; the
// help overlay uses FullHelp instead.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns all keybindings organised into columns for the
// full-screen help overlay triggered by ?.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Column 1: navigation & screens
		{k.QSOForm, k.Partner, k.DXC, k.PSKReporter, k.Ref, k.BPL, k.LogEditor, k.Config, k.Logs},
		// Column 2: editing & actions
		{k.Save, k.Spot, k.Lookup, k.Delete, k.Retain, k.NextField, k.PrevField},
		// Column 3: cycling & meta
		{k.CycleLogbook, k.CycleRig, k.CycleContest, k.CycleOperator, k.DXCSpotFill, k.RigTuneUp, k.RigTuneDown, k.Up, k.Down, k.Enter},
		// Column 4: system
		{k.CycleUp, k.CycleDown, k.Confirm, k.Cancel, k.Help, k.Quit},
	}
}

// ActiveBindings returns the currently visible key bindings based on app state.
func (m *Model) ActiveBindings() []key.Binding {
	var bindings []key.Binding

	// Pane navigation — Ctrl+←/→ cycles through available screens everywhere.
	bindings = append(bindings,
		key.NewBinding(key.WithKeys("ctrl+left", "ctrl+right"), key.WithHelp("Ctrl+←/→", "Pane")),
	)

	// QSO form — show editing shortcuts when no sub-model is active
	if !m.isSubmodelActive() {
		bindings = append(bindings,
			m.keys.Enter,
			m.keys.NextField,
			m.keys.PrevField,
			m.keys.NextRow,
			m.keys.PrevRow,
			m.keys.Spot,
			m.keys.DXCSpotFill,
			m.keys.Lookup,
			m.keys.Delete,
			m.keys.CycleUp,
			m.keys.CycleDown,
			m.keys.CycleLogbook,
			m.keys.CycleRig,
			m.keys.CycleContest,
			m.keys.CycleOperator,
			key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("Ctrl+K", "Keep Cmt")),
			key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("Ctrl+H", "Hold Form")),
		)
		// Rotor control — only when rotor is connected and on QSO screen.
		if m.rotor.connected && m.screen == screenQSO {
			bindings = append(bindings,
				m.keys.RotorLeft,
				m.keys.RotorRight,
				m.keys.RotorUp,
				m.keys.RotorDown,
				m.keys.RotorBearing,
				m.keys.RotorStop,
			)
		}
		// Rig tune — shift frequency up/down by band-appropriate step.
		if m.rig.connected && m.screen == screenQSO {
			bindings = append(bindings, m.keys.RigTuneUp, m.keys.RigTuneDown)
		}
		// Favorite slots — always available on QSO form.
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("alt+insert", "alt+home", "alt+pgup"), key.WithHelp("Alt+Ins/Hom/PUp", "Recall fav")),
			key.NewBinding(key.WithKeys("alt+shift+insert", "alt+shift+home", "alt+shift+pgup"), key.WithHelp("Alt+Shift+Ins/Hom/PUp", "Save fav")),
		)
	}

	// Log editor — table navigation shortcuts
	if m.screen == screenLogbookEditor {
		if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsEditing() {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Scroll")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit QSO")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("Ctrl+P", "Purge")),
				key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("Ctrl+E", "Export")),
				key.NewBinding(key.WithKeys("ctrl+i"), key.WithHelp("Ctrl+I", "Import")),
				m.keys.CycleContest,
			)
			wl := m.App.Logbook.Wavelog
			if wl != nil && wl.Enabled {
				bindings = append(bindings,
					key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("Ctrl+W", "Wavelog upload")),
					key.NewBinding(key.WithKeys("alt+w"), key.WithHelp("Alt+W", "Wavelog download")),
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
			key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Top")),
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
				key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Activate")),
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
				key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Activate")),
				key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
				key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("Ctrl+D", "Duplicate")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		}
	}
	if m.screen == screenOperator {
		if m.ui.operatorChooser != nil && (m.ui.operatorChooser.mode == operatorEdit || m.ui.operatorChooser.mode == operatorCreate) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("tab", "down", "shift+tab", "up"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back")),
			)
		} else if m.ui.operatorChooser != nil && m.ui.operatorChooser.mode == operatorConfirmDelete {
			bindings = append(bindings, confirmBindings...)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit")),
				key.NewBinding(key.WithKeys("space"), key.WithHelp("Space", "Activate")),
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
			key.NewBinding(key.WithKeys("f2", "esc"), key.WithHelp("F2/Esc", "Back")),
		)
	}
	if m.screen == screenDXC {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys(`\`), key.WithHelp("\\", "Sp Cont")),
			key.NewBinding(key.WithKeys("insert", "delete"), key.WithHelp("Ins/Del", "Mode")),
			key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "Band")),
			key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Time")),
			key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Bksp", "Clear")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "QSO+Tune")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "Tune")),
		)
	}
	if m.screen == screenRef {
		if m.ref.searched && len(m.ref.rows) > 0 {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("\u2191\u2193", "Navigate")),
				key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Page")),
				key.NewBinding(key.WithKeys("enter", "insert"), key.WithHelp("Enter/Ins", "Commit")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Clear")),
			)
		} else {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("enter", "insert"), key.WithHelp("Enter/Ins", "Search")),
				key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Clear")),
			)
		}
	}
	if m.screen == screenBPL {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("\u2191\u2193", "Scroll")),
			key.NewBinding(key.WithKeys("left", "right", "tab"), key.WithHelp("←→/Tab", "Tabs")),
			key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/Dn", "Page")),
			key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "Top/Bottom")),
			key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("Ctrl+E", "Export")),
		)
		if m.rig.connected && !m.wsjtx.online {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("Enter/Spc", "Tune")),
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
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Next/Test")),
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
