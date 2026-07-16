package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ref"
)

// refRebuildMsg is sent when an async REF database rebuild completes.
type refRebuildMsg struct {
	total int
	err   error
}

// refState holds the REF tab UI state: search input and result rows.
// The rendered table view is cached and invalidated when dimensions,
// row count, scroll, or cursor change.
type refState struct {
	input    textinput.Model
	rows     []ref.Row
	builtW   int
	builtH   int
	searched bool
	building bool
	ready    bool
	scroll   int // first visible row index
	cursor   int // highlighted row (absolute index into rows)

	// Render cache for the table portion (avoids table.New() every frame).
	cachedTableView   string
	cachedTableW      int
	cachedTableH      int
	cachedTableRows   int
	cachedTableScroll int
	cachedTableCursor int

	// REF names line cache — recomputed only when a REF field is exited.
	refNamesLine  string
	refNamesDirty bool
}

func newRefState() refState {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "type ref or name…"
	ti.CharLimit = 40
	ti.Focus()
	return refState{input: ti}
}

// isREFReady returns true when the REF database is open and populated.
func (m *Model) isREFReady() bool {
	return m.App != nil && m.App.RefDB != nil && !m.ref.building
}

// doRefSearch executes a search from the current input value.
func (m *Model) doRefSearch() {
	query := strings.TrimSpace(m.ref.input.Value())
	if query == "" {
		return
	}
	if m.App.RefDB == nil {
		m.toasts.Warn("REF: database not available")
		return
	}
	rows, err := m.App.RefDB.Search(query)
	if err != nil {
		applog.Warn("REF: search failed", "query", query, "error", err)
		m.toasts.Error("REF: search failed")
		return
	}
	m.ref.rows = rows
	m.ref.searched = true
	m.ref.scroll = 0
	m.ref.cursor = 0
	m.ref.cachedTableView = "" // invalidate render cache
	applog.InfoDetail("REF: search", fmt.Sprintf("query=%q results=%d", query, len(rows)))
}

// startRefRebuildCmd returns a command that asynchronously downloads CSVs
// and rebuilds the reference database. Returns nil if already building or
// the database is already populated with a modern schema.
func (m *Model) startRefRebuildCmd() tea.Cmd {
	if m.ref.building {
		return nil
	}
	if m.App.RefDB == nil {
		return nil
	}
	// Trigger rebuild when empty OR when the search column needs backfill
	// (databases created before the diacritic-insensitive search feature).
	if n, err := m.App.RefDB.Count(); err == nil && n > 0 {
		needRebuild, _ := m.App.RefDB.NeedsSearchBackfill()
		if !needRebuild {
			m.ref.ready = true
			return nil
		}
		applog.Info("REF: search column needs backfill, triggering rebuild")
	}
	m.ref.building = true
	applog.Info("REF: starting async rebuild")

	return func() tea.Msg {
		cacheDir, err := config.CacheDir()
		if err != nil {
			return refRebuildMsg{err: err}
		}
		total, err := m.App.RefDB.Rebuild(cacheDir, func(msg string) {
			applog.Info("REF: " + msg)
		})
		return refRebuildMsg{total: total, err: err}
	}
}

// handleRefUpdate handles messages for the REF tab.
func (m *Model) handleRefUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ref.builtW = msg.Width
		m.ref.builtH = msg.Height

	case refRebuildMsg:
		m.ref.building = false
		if msg.err != nil {
			applog.Warn("REF: rebuild failed", "error", msg.err)
			m.toasts.Error("REF: database build failed")
		} else {
			m.ref.ready = true
			applog.Info("REF: rebuild complete", "total", msg.total)
			m.toasts.Success(fmt.Sprintf("REF: database ready — %d references", msg.total))
		}
		return m, cmd

	case tea.KeyPressMsg:
		// Block all interaction while building.
		if m.ref.building {
			return m, cmd
		}

		switch msg.String() {
		case "esc", "f6":
			m.screen = screenQSO
			return m, cmd

		case "enter", "insert":
			// If results are visible and cursor is on a row, add to QSO form.
			if m.ref.searched && len(m.ref.rows) > 0 && m.ref.cursor >= 0 && m.ref.cursor < len(m.ref.rows) {
				r := m.ref.rows[m.ref.cursor]
				m.addRefToQSO(r)
				return m, cmd
			}
			// Otherwise, execute search.
			m.doRefSearch()
			return m, cmd

		case "delete":
			m.ref.input.SetValue("")
			m.ref.searched = false
			m.ref.rows = nil
			m.ref.scroll = 0
			m.ref.cursor = 0
			m.ref.cachedTableView = ""
			return m, cmd

		case "up", "down", "pgup", "pgdown":
			if m.ref.searched && len(m.ref.rows) > 0 {
				tableH := (contentHeight(m.height) - 6)
				if tableH < 3 {
					tableH = 3
				}
				total := len(m.ref.rows)

				switch msg.String() {
				case "up":
					if m.ref.cursor > 0 {
						m.ref.cursor--
					}
				case "down":
					if m.ref.cursor < total-1 {
						m.ref.cursor++
					}
				case "pgup":
					m.ref.cursor -= tableH
					if m.ref.cursor < 0 {
						m.ref.cursor = 0
					}
				case "pgdown":
					m.ref.cursor += tableH
					if m.ref.cursor >= total {
						m.ref.cursor = total - 1
					}
				}

				// Keep cursor visible: adjust scroll window.
				if m.ref.cursor < m.ref.scroll {
					m.ref.scroll = m.ref.cursor
				}
				if m.ref.cursor >= m.ref.scroll+tableH {
					m.ref.scroll = m.ref.cursor - tableH + 1
				}
				if m.ref.scroll < 0 {
					m.ref.scroll = 0
				}
			}
			return m, cmd
		}

		// Forward other keys to search input when focused.
		if m.ref.input.Focused() {
			ti, c := m.ref.input.Update(msg)
			m.ref.input = ti
			if c != nil {
				cmd = tea.Batch(cmd, c)
			}
		}
	}
	return m, cmd
}

// viewRef renders the REF tab: search input + results, or build/empty states.
func (m *Model) viewRef() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	ch := contentHeight(m.height)
	if ch < 8 {
		ch = 8
	}

	// Building state — show progress message.
	if m.ref.building {
		msg := DimStyle.Width(w).Align(lipgloss.Center).Render("Building REF database — downloading & importing…")
		return fillBody(msg, ch)
	}

	// Not ready — database not built yet or disabled.
	if !m.ref.ready {
		msg := "REF database not built — enable \"Use REF database\" in General settings"
		if m.App != nil && m.App.RefDB == nil {
			msg = "REF database not available — check \"Use REF database\" in General settings"
		}
		return fillBody(DimStyle.Width(w).Align(lipgloss.Center).Render(msg), ch)
	}

	if m.ref.builtW != w {
		m.ref.builtW = w
	}

	searchW := w - 4
	if searchW < 20 {
		searchW = 20
	}
	m.ref.input.SetWidth(searchW)

	var b strings.Builder

	// Title header — same style as bandplan and DXC.
	b.WriteString(S.Title.Width(w).Render("References \u2014 SOTA " + middot() + " POTA " + middot() + " WWFF " + middot() + " IOTA"))
	b.WriteString("\n")

	searchLabel := S.FormLabel.Render("Search: ")
	inputView := m.ref.input.View()
	searchRow := lipgloss.JoinHorizontal(lipgloss.Top, " ", searchLabel, inputView)
	b.WriteString(padOrTrunc(searchRow, w))
	b.WriteString("\n")

	if m.ref.searched {
		// Empty separator row between search line and results.
		b.WriteString("\n")

		if len(m.ref.rows) == 0 {
			b.WriteString(DimStyle.Width(w).Align(lipgloss.Center).Render("No matches found"))
		} else {
			bodyW := w - 2
			if bodyW < 30 {
				bodyW = 30
			}
			// Reserve lines: title (1) + search (1) +
			// separator (1) + scroll indicator (1) = 4 non-table rows.
			tableH := ch - 6
			if tableH < 3 {
				tableH = 3
			}
			// Clamp scroll to valid range.
			total := len(m.ref.rows)
			if m.ref.scroll < 0 {
				m.ref.scroll = 0
			}
			maxScroll := total - tableH
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.ref.scroll > maxScroll {
				m.ref.scroll = maxScroll
			}

			// Use cached table view when dimensions and positions are unchanged.
			useCache := m.ref.cachedTableView != "" &&
				m.ref.cachedTableW == bodyW &&
				m.ref.cachedTableH == tableH &&
				m.ref.cachedTableRows == total &&
				m.ref.cachedTableScroll == m.ref.scroll &&
				m.ref.cachedTableCursor == m.ref.cursor

			if useCache {
				b.WriteString(m.ref.cachedTableView)
			} else {
				refW := 14
				gridW := 8
				altW := 10
				nameW := bodyW - refW - gridW - altW - 3
				if nameW < 10 {
					nameW = 10
				}

				cols := []table.Column{
					{Title: "Ref", Width: refW},
					{Title: "Name", Width: nameW},
					{Title: "Grid", Width: gridW},
					{Title: "Alt (m)", Width: altW},
				}

				// Build visible window with cursor marker on the first row.
				end := m.ref.scroll + tableH
				if end > total {
					end = total
				}
				marker := m.ref.cursor // highlighted row
				var rows []table.Row
				for i := m.ref.scroll; i < end; i++ {
					r := m.ref.rows[i]
					alt := "\u2014"
					if r.Height > 0 {
						alt = fmt.Sprintf("%d", r.Height)
					}
					rows = append(rows, table.Row{
						string(r.RefType) + " " + r.Ref,
						r.Name,
						r.Grid,
						alt,
					})
				}

				tbl := table.New(
					table.WithColumns(cols),
					table.WithRows(rows),
					table.WithHeight(tableH+1),
					table.WithWidth(bodyW),
					table.WithFocused(true),
				)
				// Move cursor to the marker row within the visible window.
				if marker >= m.ref.scroll && marker < end {
					tbl.SetCursor(marker - m.ref.scroll)
				}

				s := table.DefaultStyles()
				s.Header = s.Header.
					BorderForeground(P.TextDim).
					BorderBottom(true).
					Bold(false).
					Foreground(P.Text)
				tbl.SetStyles(s)

				m.ref.cachedTableView = tbl.View()
				m.ref.cachedTableW = bodyW
				m.ref.cachedTableH = tableH
				m.ref.cachedTableRows = total
				m.ref.cachedTableScroll = m.ref.scroll
				m.ref.cachedTableCursor = m.ref.cursor
				b.WriteString(m.ref.cachedTableView)
			}
		}
	} else {
		b.WriteString(DimStyle.Width(w).Align(lipgloss.Center).Render("Enter a reference or name to search"))
	}

	// Return raw content — buildBodyForScreen handles height clamping/padding
	// so the scroll indicator sits at the true bottom of the content area.
	return b.String()
}

// buildRefNamesLine resolves the SOTA/POTA/WWFF/IOTA form field references
// into human-readable names and returns them as a single line. Returns ""
// when all fields are empty or the REF database is not available.
// Results are cached until a REF field is exited (blur) — typing alone
// does not trigger a lookup and does not show the border.
func (m *Model) buildRefNamesLine() string {
	if m.App == nil || m.App.RefDB == nil {
		return ""
	}

	// If not dirty, return whatever we cached (may be "").
	if !m.ref.refNamesDirty {
		return m.ref.refNamesLine
	}

	// Dirty: rebuild from current field values.
	m.ref.refNamesDirty = false

	sota := strings.TrimSpace(m.fields[fieldSOTA].Value())
	pota := strings.TrimSpace(m.fields[fieldPOTA].Value())
	wwff := strings.TrimSpace(m.fields[fieldWWFF].Value())
	iota := strings.TrimSpace(m.fields[fieldIOTA].Value())

	if sota == "" && pota == "" && wwff == "" && iota == "" {
		m.ref.refNamesLine = ""
		return ""
	}

	db := m.App.RefDB
	var parts []string

	resolveOne := func(prefix string, rt ref.RefType, csv string) string {
		if csv == "" {
			return ""
		}
		refs := strings.Split(csv, ",")
		var names []string
		for _, r := range refs {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			name := db.NameForRef(rt, r)
			if name == "" || strings.EqualFold(name, r) {
				// Unresolved — render in red.
				names = append(names, S.Error.Render(r))
			} else {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			return ""
		}
		return prefix + strings.Join(names, ", ")
	}

	if p := resolveOne("SOTA: ", ref.RefSOTA, sota); p != "" {
		parts = append(parts, p)
	}
	if p := resolveOne("POTA: ", ref.RefPOTA, pota); p != "" {
		parts = append(parts, p)
	}
	if p := resolveOne("WWFF: ", ref.RefWWFF, wwff); p != "" {
		parts = append(parts, p)
	}
	if p := resolveOne("IOTA: ", ref.RefIOTA, iota); p != "" {
		parts = append(parts, p)
	}

	if len(parts) == 0 {
		m.ref.refNamesLine = ""
		return ""
	}
	m.ref.refNamesLine = strings.Join(parts, "  \u2502  ") // "│" separator
	return m.ref.refNamesLine
}

// refPriority is the lookup order for grid autofill — most accurate first.
var refPriority = []struct {
	field field
	rt    ref.RefType
}{
	{fieldSOTA, ref.RefSOTA},
	{fieldPOTA, ref.RefPOTA},
	{fieldWWFF, ref.RefWWFF},
	{fieldIOTA, ref.RefIOTA},
}

// applyRefGridAndQTH looks up the grid from the REF database in priority
// order (SOTA > POTA > WWFF > IOTA) and sets the Grid field to the first
// found value. It also builds the QTH field from the joined names of all
// referenced programmes. Called on REF field exit so activation location
// takes precedence over QRZ/CTY home QTH.
func (m *Model) applyRefGridAndQTH() {
	if m.App == nil || m.App.RefDB == nil {
		return
	}
	db := m.App.RefDB

	// Collect all resolved names for the QTH field.
	var qthParts []string
	var bestGrid string
	var bestSource gridSource

	for _, rp := range refPriority {
		csv := strings.TrimSpace(m.fields[rp.field].Value())
		if csv == "" {
			continue
		}
		refs := strings.Split(csv, ",")
		for _, r := range refs {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			row, ok := db.Lookup(rp.rt, r)
			if !ok {
				continue
			}
			// Collect resolved name for QTH (use name if available, else ref).
			name := row.Name
			if name == "" {
				name = r
			}
			qthParts = append(qthParts, name)

			// Take the first available grid in priority order.
			if bestGrid == "" && row.Grid != "" {
				bestGrid = row.Grid
				switch rp.rt {
				case ref.RefSOTA:
					bestSource = gridSourceSOTA
				case ref.RefPOTA:
					bestSource = gridSourcePOTA
				case ref.RefWWFF:
					bestSource = gridSourceWWFF
				case ref.RefIOTA:
					bestSource = gridSourceIOTA
				}
			}
		}
	}

	// Set grid from the highest-priority REF that has one.
	if bestGrid != "" {
		m.fields[fieldGrid].SetValue(bestGrid)
		m.rc.pathGrid = bestGrid
		m.gridSource = bestSource
		m.invalidatePartnerMapCache()
	}

	// Set QTH to joined names of all referenced programmes.
	if len(qthParts) > 0 {
		m.fields[fieldQTH].SetValue(strings.Join(qthParts, " | "))
	}
}

// addRefToQSO adds the selected REF row's reference code to the matching
// QSO form field (SOTA/POTA/WWFF/IOTA). Multiple references of the same
// type can be comma-separated. The REF screen stays open so the user can
// add more references from the same search.
func (m *Model) addRefToQSO(r ref.Row) {
	if r.Ref == "" {
		return
	}
	var targetField field
	var prefix string
	switch r.RefType {
	case ref.RefSOTA:
		targetField = fieldSOTA
		prefix = "SOTA: "
	case ref.RefPOTA:
		targetField = fieldPOTA
		prefix = "POTA: "
	case ref.RefWWFF:
		targetField = fieldWWFF
		prefix = "WWFF: "
	case ref.RefIOTA:
		targetField = fieldIOTA
		prefix = "IOTA: "
	default:
		return
	}

	current := strings.TrimSpace(m.fields[targetField].Value())
	if current == "" {
		m.fields[targetField].SetValue(r.Ref)
	} else {
		// Check if this ref is already present.
		existing := strings.Split(current, ",")
		for _, e := range existing {
			if strings.TrimSpace(e) == r.Ref {
				m.toasts.Info(prefix + r.Ref + " already added")
				m.invalidateRefNamesCache()
				return
			}
		}
		m.fields[targetField].SetValue(current + ", " + r.Ref)
	}
	m.invalidateRefNamesCache()
	m.toasts.Success(prefix + r.Ref + " added to QSO")
	applog.Info("REF: added to QSO", "type", string(r.RefType), "ref", r.Ref, "name", r.Name)
}

// invalidateRefNamesCache marks the REF names line cache as dirty.
func (m *Model) invalidateRefNamesCache() {
	m.ref.refNamesDirty = true
	m.rc.status = ""
}
