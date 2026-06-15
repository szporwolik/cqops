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
	// Build a compact cache key covering all confirm-dialog sources.
	suffix := m.helpSuffix()
	conf := 0
	if m.confirm != nil {
		conf = 1
	}
	if m.logbookEditor != nil && m.logbookEditor.isConfirmMode() {
		conf = 1
	}
	if m.rigChooser != nil && m.rigChooser.dialog != nil {
		conf = 1
	}
	if m.chooser != nil && m.chooser.dialog != nil {
		conf = 1
	}
	sig := fmt.Sprintf("%d|%d|%s", m.screen, conf, suffix)
	if m.cachedHelpSig == sig && m.cachedHelpView != "" {
		return m.cachedHelpView
	}

	var result string

	// Global confirm dialog (quit, etc.)
	if m.confirm != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.cachedHelpSig = sig
		m.cachedHelpView = result
		return result
	}

	// Internal confirm dialogs: rig/chooser delete confirmations.
	if m.rigChooser != nil && m.rigChooser.mode == rigChooserConfirmDelete && m.rigChooser.dialog != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.cachedHelpSig = sig
		m.cachedHelpView = result
		return result
	}
	if m.chooser != nil && m.chooser.mode == chooserConfirmDelete && m.chooser.dialog != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.cachedHelpSig = sig
		m.cachedHelpView = result
		return result
	}
	if m.logbookEditor != nil && m.logbookEditor.isConfirmMode() && m.logbookEditor.dialog != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
		m.cachedHelpSig = sig
		m.cachedHelpView = result
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
	m.cachedHelpSig = sig
	m.cachedHelpView = result
	return result
}

// helpSuffix returns the dynamic right-aligned suffix for the help bar
// (QSO counter in log editor, scroll info in log viewer). This is
// computed every frame and not cached.
func (m *Model) helpSuffix() string {
	if m.screen == screenLogbookEditor && m.logbookEditor != nil {
		le := m.logbookEditor
		cursor := le.table.Cursor()
		total := le.totalCount
		if total > 0 {
			globalPos := (le.currentPage-1)*le.pageSize + cursor + 1
			pageInfo := fmt.Sprintf("Page %d/%d", le.currentPage, le.totalPages())
			return fmt.Sprintf("QSO %d/%d  %s", globalPos, total, pageInfo)
		}
	}
	if m.screen == screenLogView && m.logViewer != nil {
		return m.logViewer.ScrollInfo()
	}
	return ""
}

// renderHelpBar is the canonical entry point for help bar rendering.
func (m *Model) renderHelpBar() string { return m.helpView() }
