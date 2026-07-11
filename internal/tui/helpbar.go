package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// Pre-allocated confirm dialog key bindings — reused across all confirm screens.
var confirmBindings = []key.Binding{
	key.NewBinding(key.WithKeys("←/→"), key.WithHelp("←/→", "choose")),
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

// Pre-allocated spot dialog key bindings.
var spotBindings = []key.Binding{
	key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "toggle")),
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "send")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "cancel")),
}

// Pre-allocated help overlay styles — invariant, created once at init.
var helpTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(P.Cursor)
var helpDismissStyle = lipgloss.NewStyle().Foreground(P.TextMuted)
var helpBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(P.Cursor).
	Padding(0, 1)

// adifExtsMap avoids allocating a map every frame in helpSuffix().
var adifExtsMap = map[string]bool{".adi": true, ".adif": true}

// helpView renders the bottom help/footer bar with context-sensitive key bindings.
// Cached — only rebuilds when screen, confirm state, or dynamic suffix change.
func (m *Model) helpView() string {
	// Build a compact cache key covering all confirm-dialog sources
	// and form modes (logbook editor, chooser, rig chooser).
	suffix := m.helpSuffix()
	conf := 0
	spot := 0
	editing := 0
	chooserForm := 0
	rigForm := 0
	contestForm := 0
	contestConfirm := 0
	if m.confirm != nil {
		conf = 1
	}
	if m.spotDialog != nil {
		spot = 1
	}
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.isModalMode() {
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
	if m.ui.contestChooser != nil && (m.ui.contestChooser.mode == contestEdit || m.ui.contestChooser.mode == contestCreate) {
		contestForm = 1
	}
	if m.ui.contestChooser != nil && m.ui.contestChooser.mode == contestConfirmDelete {
		contestConfirm = 1
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
	sig := m.buildHelpSig(m.screen, conf, spot, editing, chooserForm, rigForm, contestForm, contestConfirm, exporting, importing, suffix)
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

	// Spot dialog
	if m.spotDialog != nil {
		result = HelpStyle.Render(m.help.ShortHelpView(spotBindings))
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
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.isModalMode() && m.ui.logbookEditor.dialog != nil {
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

	bindings := m.minimalBarBindings()
	helpText := m.help.ShortHelpView(bindings)

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

// buildHelpSig builds the help-bar cache key using strings.Builder to avoid
// fmt.Sprintf allocation on every frame.
func (m *Model) buildHelpSig(screen screenKind, conf, spot, editing, chooserForm, rigForm, contestForm, contestConfirm, exporting, importing int, suffix string) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(int(screen)))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(m.width))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(conf))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(spot))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(editing))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(chooserForm))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(rigForm))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(contestForm))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(contestConfirm))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(exporting))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(importing))
	b.WriteByte('|')
	if m.rig.connected {
		b.WriteByte('1')
	} else {
		b.WriteByte('0')
	}
	b.WriteByte('|')
	if m.wsjtx.online {
		b.WriteByte('1')
	} else {
		b.WriteByte('0')
	}
	b.WriteByte('|')
	b.WriteString(suffix)
	return b.String()
}

// helpSuffix returns the dynamic right-aligned suffix for the help bar
// (QSO counter in log editor, scroll info in log viewer). Cached with
// a signature to avoid per-frame fmt.Sprintf allocation.
func (m *Model) helpSuffix() string {
	// Build signature from all state that affects the suffix.
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(int(m.screen)))
	sb.WriteByte('|')
	if m.ui.logbookEditor != nil {
		le := m.ui.logbookEditor
		sb.WriteString(strconv.Itoa(int(le.mode)))
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(le.currentPage))
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(le.table.Cursor()))
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(le.totalCount))
		sb.WriteByte('|')
		if le.mode == edModeExport || le.mode == edModeImport {
			fp := le.FilePicker()
			sb.WriteString(fp.CurrentDirectory)
		}
	} else {
		sb.WriteString("0|0|0|0")
	}
	if m.ui.logViewer != nil {
		sb.WriteByte('|')
		sb.WriteString(m.ui.logViewer.ScrollInfo())
	}
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(len(m.ref.rows)))
	sig := sb.String()
	if m.rc.helpSuffixSig == sig && m.rc.helpSuffix != "" {
		return m.rc.helpSuffix
	}
	m.rc.helpSuffixSig = sig
	m.rc.helpSuffix = m.buildHelpSuffix()
	return m.rc.helpSuffix
}

// buildHelpSuffix is the uncached suffix builder, called only on cache miss.
func (m *Model) buildHelpSuffix() string {
	if m.screen == screenLogbookEditor && m.ui.logbookEditor != nil {
		le := m.ui.logbookEditor

		// File picker modes — show position counter.
		if le.mode == edModeExport || le.mode == edModeImport {
			fp := le.FilePicker()
			dir := fp.CurrentDirectory
			// Cache directory listing with 500ms TTL to avoid per-frame disk I/O.
			entries, err := m.cachedReadDir(dir)
			if err != nil {
				return fmt.Sprintf("Path: %s", dir)
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
							if filepath.Join(dir, e.Name()) == highlighted {
								break
							}
						}
					}
					if pos == 0 {
						pos = 1
					}
					return fmt.Sprintf("Folder %d/%d", pos, total)
				}
				return fmt.Sprintf("Path: %s", dir)
			}

			// Import: count only ADIF files (.adi, .adif).
			adifExts := adifExtsMap
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
					if filepath.Join(dir, name) == highlighted {
						break
					}
				}
				if pos == 0 {
					pos = 1
				}
				return fmt.Sprintf("File %d/%d", pos, len(adifFiles))
			}
			return fmt.Sprintf("Path: %s", dir)
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

// minimalBarBindings returns the 2–3 key entries shown in the bottom bar.
// Always includes ? Help and F10 Quit; the middle entry is screen-specific.
func (m *Model) minimalBarBindings() []key.Binding {
	h := m.keys.Help
	q := m.keys.Quit
	e := key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back"))
	switch m.screen {
	case screenQSO:
		// Enter is the primary action; Ctrl+F/Ctrl+↑/Ctrl+↓ are
		// available via the ? help overlay and kept out of the bar
		// to keep the bottom line clean for portable/small screens.
		return []key.Binding{h, m.keys.Enter, q}
	case screenPartner:
		if m.lookup.partnerData != nil && m.lookup.partnerData.ImageURL != "" {
			return []key.Binding{h, key.NewBinding(key.WithKeys("f2"), key.WithHelp("F2", "Photo")), q}
		}
		return []key.Binding{h, q}
	case screenDXC:
		return []key.Binding{h, key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "QSO+Tune")), e, q}
	case screenLogbookEditor:
		if m.ui.logbookEditor != nil && m.ui.logbookEditor.IsEditing() {
			return []key.Binding{h, key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")), e, q}
		}
		return []key.Binding{h, e, q}
	case screenConfig, screenIntegration, screenNotifications:
		if m.isSubmodelActive() {
			return []key.Binding{h, key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")), e, q}
		}
		return []key.Binding{h, e, q}
	case screenChooser:
		if m.ui.chooser != nil && (m.ui.chooser.mode == chooserEdit || m.ui.chooser.mode == chooserCreate) {
			return []key.Binding{h, key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")), e, q}
		}
		return []key.Binding{h,
			key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
			key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
			key.NewBinding(key.WithKeys("space"), key.WithHelp("Spc", "Activate")),
			e, q}
	case screenRigEdit:
		if m.ui.rigChooser != nil && (m.ui.rigChooser.mode == rigChooserEdit || m.ui.rigChooser.mode == rigChooserCreate) {
			return []key.Binding{h, key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")), e, q}
		}
		return []key.Binding{h,
			key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
			key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
			key.NewBinding(key.WithKeys("space"), key.WithHelp("Spc", "Activate")),
			e, q}
	case screenContest:
		if m.ui.contestChooser != nil && (m.ui.contestChooser.mode == contestEdit || m.ui.contestChooser.mode == contestCreate) {
			return []key.Binding{h, key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")), e, q}
		}
		return []key.Binding{h,
			key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
			key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
			key.NewBinding(key.WithKeys("space"), key.WithHelp("Spc", "Activate")),
			e, q}
	case screenOperator:
		if m.ui.operatorChooser != nil && (m.ui.operatorChooser.mode == operatorEdit || m.ui.operatorChooser.mode == operatorCreate) {
			return []key.Binding{h, key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")), e, q}
		}
		return []key.Binding{h,
			key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Create")),
			key.NewBinding(key.WithKeys("delete"), key.WithHelp("Del", "Delete")),
			key.NewBinding(key.WithKeys("space"), key.WithHelp("Spc", "Activate")),
			e, q}
	case screenMainMenu:
		return []key.Binding{h, key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Select")), e, q}
	case screenLogView:
		return []key.Binding{h, key.NewBinding(key.WithKeys("insert"), key.WithHelp("Ins", "Top")), q}
	case screenRef:
		if m.ref.searched && len(m.ref.rows) > 0 {
			return []key.Binding{h, key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Commit")), q}
		}
		return []key.Binding{h, key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Search")), q}
	case screenBPL:
		if m.rig.connected && !m.wsjtx.online {
			return []key.Binding{h, key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Tune")), q}
		}
		return []key.Binding{h, key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←→", "Tabs")), q}
	case screenImage:
		return []key.Binding{h, e, q}
	case screenPSKReporter:
		return []key.Binding{h, key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Bksp", "Clear")), q}
	default:
		return []key.Binding{h, q}
	}
}

// renderHelpBar is the canonical entry point for help bar rendering.
func (m *Model) renderHelpBar() string { return m.helpView() }

// activeHelpKeyMap adapts the context-sensitive ActiveBindings list into
// the help.KeyMap interface so the help overlay can render columns.
type activeHelpKeyMap struct {
	bindings []key.Binding
}

func (a activeHelpKeyMap) ShortHelp() []key.Binding {
	// Show all bindings that have help text.
	out := make([]key.Binding, 0, len(a.bindings))
	for _, b := range a.bindings {
		if b.Help().Key != "" || b.Help().Desc != "" {
			out = append(out, b)
		}
	}
	return out
}

func (a activeHelpKeyMap) FullHelp() [][]key.Binding {
	flat := a.ShortHelp()
	if len(flat) == 0 {
		return nil
	}
	// Distribute into 3 columns, filling column-first.
	colSize := (len(flat) + 2) / 3
	cols := make([][]key.Binding, 3)
	for i, b := range flat {
		col := i / colSize
		if col > 2 {
			col = 2
		}
		cols[col] = append(cols[col], b)
	}
	// Remove empty trailing columns.
	for len(cols) > 0 && len(cols[len(cols)-1]) == 0 {
		cols = cols[:len(cols)-1]
	}
	return cols
}

// screenTitle returns a short human-readable name for the current screen.
func (m *Model) screenTitle() string {
	switch m.screen {
	case screenQSO:
		return "QSO"
	case screenPartner:
		return "Partner"
	case screenImage:
		return "Photo"
	case screenDXC:
		return "DX Cluster"
	case screenPSKReporter:
		return "PSK Reporter"
	case screenRef:
		return "References"
	case screenBPL:
		return "Band Plan"
	case screenLogbookEditor:
		return "Log Editor"
	case screenLogView:
		return "Log Viewer"
	case screenConfig:
		return "Settings"
	case screenIntegration:
		return "Integrations"
	case screenNotifications:
		return "Notifications"
	case screenMainMenu:
		return "Menu"
	case screenChooser:
		return "Logbooks"
	case screenRigEdit:
		return "Rigs"
	case screenContest:
		return "Contests"
	case screenOperator:
		return "Operators"
	default:
		return "CQOps"
	}
}

// renderHelpOverlay composites a floating help overlay in the bottom-left
// corner showing the current screen's keybindings in columns.
// Dismissed with ? or Esc.
func (m *Model) renderHelpOverlay(mainView string, l Layout) string {
	// Never render before the first tick completes — initialization
	// commands must not be delayed by overlay compositing.
	if m.tickCount < 1 {
		return mainView
	}
	bindings := m.ActiveBindings()
	adapter := activeHelpKeyMap{bindings: bindings}

	// Set help width so columns wrap gracefully.
	m.help.SetWidth(max(l.TerminalW-4, 40))

	helpContent := m.help.View(adapter)

	// Build the floating box with screen title and dismiss hint.
	// No extra spaces — padding (0,1) on the box already provides margins.
	title := helpTitleStyle.Render(m.screenTitle() + " Keys")
	dismiss := helpDismissStyle.Render("?/Esc close")

	// Pad the shorter element so title and help content share the same width,
	// preventing the title from wrapping on screens with few bindings.
	contentW := lipgloss.Width(helpContent)
	titleW := lipgloss.Width(title)
	w := max(contentW, titleW)
	title = lipgloss.NewStyle().Width(w).Render(title)
	// Only set width on help content if it's narrower — wider content
	// already has columns set by m.help.SetWidth.
	if contentW < w {
		helpContent = lipgloss.NewStyle().Width(w).Render(helpContent)
	}

	boxContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		helpContent,
		"",
		dismiss,
	)

	box := helpBoxStyle.Render(boxContent)

	boxH := lipgloss.Height(box)

	// Position: bottom-left corner with 1-cell margin.
	x := 1
	y := l.TerminalH - boxH - 1
	if y < 0 {
		y = 0
	}

	base := lipgloss.NewLayer(mainView)
	helpLayer := lipgloss.NewLayer(box).
		X(x).
		Y(y).
		Z(2) // above toasts

	return lipgloss.NewCompositor(base, helpLayer).Render()
}

// cachedReadDir returns a cached directory listing with a 500ms TTL
// to avoid per-frame os.ReadDir() during View().
func (m *Model) cachedReadDir(dir string) ([]os.DirEntry, error) {
	now := time.Now()
	if m.rc.dirCachePath == dir && now.Sub(m.rc.dirCacheTime) < 500*time.Millisecond {
		return m.rc.dirCacheEntries, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	m.rc.dirCachePath = dir
	m.rc.dirCacheTime = now
	m.rc.dirCacheEntries = entries
	return entries, nil
}
