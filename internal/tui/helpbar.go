package tui

import (
	"fmt"
	"os"
	"path/filepath"
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
	exporting := 0
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsExporting() {
		exporting = 1
	}
	importing := 0
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsImporting() {
		importing = 1
	}
	sig := fmt.Sprintf("%d|%d|%d|%d|%d|%d|%d|%d|%t|%t|%s", m.screen, m.width, conf, editing, chooserForm, rigForm, exporting, importing, m.rig.connected, m.wsjtx.online, suffix)
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

	// File export mode — show directory picker key bindings.
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsExporting() {
		exportBindings := []key.Binding{
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←→", "Folder up/down")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Confirm")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Cancel")),
			m.keys.Quit,
		}
		helpText := m.help.ShortHelpView(exportBindings)
		if suffix != "" {
			suffixW := lipgloss.Width(suffix)
			spacerW := m.width - lipgloss.Width(helpText) - suffixW - 2
			if spacerW < 1 {
				spacerW = 1
			}
			helpText = helpText + strings.Repeat(" ", spacerW) + suffix
		}
		result = HelpStyle.Render(helpText)
		m.rc.helpSig = sig
		m.rc.helpView = result
		return result
	}

	// File import mode — show file picker key bindings.
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsImporting() {
		importBindings := []key.Binding{
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "Navigate")),
			key.NewBinding(key.WithKeys("left", "right", "enter"), key.WithHelp("←→/Enter", "Open/Select")),
			key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Back", "Up")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Cancel")),
			m.keys.Quit,
		}
		helpText := m.help.ShortHelpView(importBindings)
		if suffix != "" {
			suffixW := lipgloss.Width(suffix)
			spacerW := m.width - lipgloss.Width(helpText) - suffixW - 2
			if spacerW < 1 {
				spacerW = 1
			}
			helpText = helpText + strings.Repeat(" ", spacerW) + suffix
		}
		result = HelpStyle.Render(helpText)
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

		// File picker modes — show position counter.
		if le.mode == edModeExport || le.mode == edModeImport {
			fp := le.FilePicker()
			entries, err := os.ReadDir(fp.CurrentDirectory)
			if err != nil {
				return fmt.Sprintf("Path: %s", fp.CurrentDirectory)
			}

			// Export: count directories.
			if le.IsExporting() && !le.IsImporting() {
				total := 0
				for _, e := range entries {
					if e.IsDir() {
						total++
					}
				}
				if total > 0 {
					highlighted := fp.HighlightedPath()
					pos := 0
					for _, e := range entries {
						if e.IsDir() {
							pos++
							if filepath.Join(fp.CurrentDirectory, e.Name()) == highlighted {
								break
							}
						}
					}
					if pos == 0 {
						pos = 1
					}
					return fmt.Sprintf("Folder %d/%d", pos, total)
				}
				return fmt.Sprintf("Path: %s", fp.CurrentDirectory)
			}

			// Import: count only ADIF files (.adi, .adif).
			adifExts := map[string]bool{".adi": true, ".adif": true}
			adifFiles := make([]string, 0)
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(e.Name()))
				if adifExts[ext] {
					adifFiles = append(adifFiles, e.Name())
				}
			}
			if len(adifFiles) > 0 {
				highlighted := fp.HighlightedPath()
				pos := 0
				for _, name := range adifFiles {
					pos++
					if filepath.Join(fp.CurrentDirectory, name) == highlighted {
						break
					}
				}
				if pos == 0 {
					pos = 1
				}
				return fmt.Sprintf("File %d/%d", pos, len(adifFiles))
			}
			return fmt.Sprintf("Path: %s", fp.CurrentDirectory)
		}

		// QSO list counter.
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
	if m.screen == screenRef && m.ref.searched && len(m.ref.rows) > 0 {
		total := len(m.ref.rows)
		tableH := contentHeight(m.height) - 2
		if tableH < 1 {
			tableH = 1
		}
		page := m.ref.cursor/tableH + 1
		totalPages := (total + tableH - 1) / tableH
		if totalPages < 1 {
			totalPages = 1
		}
		return fmt.Sprintf("Result %d/%d  Page %d/%d", m.ref.cursor+1, total, page, totalPages)
	}
	if m.screen == screenBPL {
		total := m.bplRowCount()
		if total > 0 {
			tableH := bplTableHeight(m.height)
			page := m.bpl.cursor/tableH + 1
			totalPages := (total + tableH - 1) / tableH
			if totalPages < 1 {
				totalPages = 1
			}
			return fmt.Sprintf("Line %d/%d  Page %d/%d", m.bpl.cursor+1, total, page, totalPages)
		}
	}
	if m.screen == screenDXC {
		return ""
	}
	return ""
}

// renderHelpBar is the canonical entry point for help bar rendering.
func (m *Model) renderHelpBar() string { return m.helpView() }
