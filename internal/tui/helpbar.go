package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// Pre-allocated confirm dialog key bindings — reused across all confirm screens.
var confirmBindings = []key.Binding{
	key.NewBinding(key.WithKeys("←/→"), key.WithHelp("←/→", "choose")),
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

// helpView renders the bottom help/footer bar with context-sensitive key bindings.
// Cached — only rebuilds when screen, confirm state, or dynamic suffix change.
func (m *Model) helpView() string {
	// Build a compact cache key covering all confirm-dialog sources
	// and form modes (logbook editor, chooser, rig chooser).
	suffix := m.helpSuffix()
	conf := 0
	editing := 0
	chooserForm := 0
	rigForm := 0
	if m.confirm != nil {
		conf = 1
	}
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.isConfirmMode() {
		conf = 1
	}
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsEditing() {
		editing = 1
	}
	if m.ui.chooser != nil && (m.ui.chooser.mode == chooserEdit || m.ui.chooser.mode == chooserCreate) {
		chooserForm = 1
	}
	if m.ui.rigChooser != nil && (m.ui.rigChooser.mode == rigChooserEdit || m.ui.rigChooser.mode == rigChooserCreate) {
		rigForm = 1
	}
	if m.ui.rigChooser != nil && m.ui.rigChooser.dialog != nil {
		conf = 1
	}
	if m.ui.chooser != nil && m.ui.chooser.dialog != nil {
		conf = 1
	}
	sig := fmt.Sprintf("%d|%d|%d|%d|%d|%d|%t|%t|%s", m.screen, m.width, conf, editing, chooserForm, rigForm, m.rig.connected, m.wsjtx.online, suffix)
	if m.rc.helpSig == sig && m.rc.helpView != "" {
		return m.rc.helpView
	}

	var result string

	// Global confirm dialog (quit, etc.)
	if m.confirm != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.rc.helpSig = sig
		m.rc.helpView = result
		return result
	}

	// Internal confirm dialogs: rig/chooser delete confirmations.
	if m.ui.rigChooser != nil && m.ui.rigChooser.mode == rigChooserConfirmDelete && m.ui.rigChooser.dialog != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.rc.helpSig = sig
		m.rc.helpView = result
		return result
	}
	if m.ui.chooser != nil && m.ui.chooser.mode == chooserConfirmDelete && m.ui.chooser.dialog != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.rc.helpSig = sig
		m.rc.helpView = result
		return result
	}
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.isConfirmMode() && m.ui.logbookEditor.dialog != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.rc.helpSig = sig
		m.rc.helpView = result
		return result
	}

	bindings := m.ActiveBindings()
	if len(bindings) == 0 {
		bindings = []key.Binding{m.keys.Quit}
	}
	helpText := m.help.ShortHelpView(bindings)
	if helpText == "" {
		helpText = m.help.ShortHelpView([]key.Binding{m.keys.Quit})
	}

	// Dynamic suffix (counter / scroll info) — always computed, never cached.
	if suffix != "" {
		suffixW := lipgloss.Width(suffix)
		spacerW := m.width - lipgloss.Width(helpText) - suffixW - 2
		if spacerW < 1 {
			spacerW = 1
		}
		result = HelpStyle.Render(helpText + strings.Repeat(" ", spacerW) + suffix)
	} else {
		result = HelpStyle.Render(helpText)
	}
	m.rc.helpSig = sig
	m.rc.helpView = result
	return result
}

// helpSuffix returns the dynamic right-aligned suffix for the help bar
// (QSO counter in log editor, scroll info in log viewer). This is
// computed every frame and not cached.
func (m *Model) helpSuffix() string {
	if m.screen == screenLogbookEditor && m.ui.logbookEditor != nil {
		le := m.ui.logbookEditor
		cursor := le.table.Cursor()
		total := le.totalCount
		if total > 0 {
			globalPos := (le.currentPage-1)*le.pageSize + cursor + 1
			pageInfo := fmt.Sprintf("Page %d/%d", le.currentPage, le.totalPages())
			return fmt.Sprintf("QSO %d/%d  %s", globalPos, total, pageInfo)
		}
	}
	if m.screen == screenLogView && m.ui.logViewer != nil {
		return m.ui.logViewer.ScrollInfo()
	}
	if m.screen == screenDXC {
		return ""
	}
	return ""
}

// renderHelpBar is the canonical entry point for help bar rendering.
func (m *Model) renderHelpBar() string { return m.helpView() }
