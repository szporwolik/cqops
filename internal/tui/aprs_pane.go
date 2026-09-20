package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ftl/hamradio/latlon"
	"github.com/ftl/hamradio/locator"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/aprs"
	"github.com/szporwolik/cqops/internal/geo"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// F3 APRS pane — nearby APRS stations list (left) + details (right)
// =============================================================================

// aprsPaneState holds the F3 APRS pane state: the full station list, the
// filtered view, the selection, and render caches.
type aprsPaneState struct {
	listAll  []aprsStation // full distance-sorted list from the cache
	stations []aprsStation // filtered view of listAll
	sel      int           // selected row (mirrors the table cursor)

	// Filters — same pattern as the DXC pane (idx + value + choices).
	distIdx    int    // index into aprsDistFilterChoices
	distFilter int    // km, 0 = all
	timeIdx    int    // index into aprsTimeFilterChoices
	timeFilter int    // minutes, 0 = all
	typeIdx    int    // index into aprsTypeFilterChoices
	typeFilter string // "" = all, "operators" = humans only

	// Table — same bubbles/table component as the DXC pane.
	table      table.Model
	tableReady bool
	builtW     int
	builtH     int

	// Render caches.
	filterView string
	filterSig  string
	detailView string
	detailSig  string
}

// aprsStation is a cached APRS station plus precomputed display data.
type aprsStation struct {
	rec     aprs.StationRecord
	distKm  float64
	bearing int
	grid    string // 6-character Maidenhead locator derived from lat/lon
	dtype   string // Yaesu-style station type code (E, P, p, W, O, I, ...)
}

// aprsReloadIntervalTicks is the list reload cadence while the pane is visible.
const aprsReloadIntervalTicks = 10 // ~10 s

// Filter choices — cycle on the same pattern as the DXC pane.
var aprsDistFilterChoices = []int{0, 1, 5, 10, 25, 50, 100} // km, 0 = all
var aprsTimeFilterChoices = []int{0, 60, 30, 15, 10, 5}     // minutes, 0 = all
var aprsTypeFilterChoices = []string{"", "operators"}       // "" (all) first — DXC convention

// aprsIsOperatorSymbol reports whether an APRS symbol code belongs to an
// actual operator (home, vehicle, portable) rather than infrastructure
// (digipeaters, repeaters, WX stations, beacons, ...). Both the primary
// ("/") and alternate ("\") tables use the same code meanings.
func aprsIsOperatorSymbol(sym string) bool {
	if len(sym) < 2 || (sym[0] != '/' && sym[0] != '\\') {
		return false
	}
	switch sym[1] {
	case '>', '-', 'R', 'k', 'v', 'j', '<', 'b', '[', 's', 'Y', '^', '\'', 'X', 'O', 'p':
		return true
	}
	return false
}

// aprsPaneRefresh reloads the nearby-station list from the APRS cache.
// Mirrors the dashboard push: 60-minute cutoff, optional radius filter,
// source selection by active service, sorted by distance ascending.
func (m *Model) aprsPaneRefresh() {
	st := &m.aprsPane
	if m.App == nil || m.App.APRSCache == nil {
		st.listAll = nil
		st.stations = nil
		st.sel = 0
		st.tableReady = false
		st.filterSig = ""
		st.detailSig = ""
		return
	}

	src := "aprs_is"
	if svc := m.App.Config.Integrations.APRS.Service; svc == "kiss" || svc == "kiss_server" {
		src = "kiss"
	}
	records, err := m.App.APRSCache.RecentStations(200, src)
	if err != nil {
		applog.Debug("APRS pane: cache read failed", "error", err)
		return
	}

	var stLat, stLon, radiusKm float64
	if g := m.effectiveGrid(); g != "" {
		stLat, stLon = gridToLatLon(g)
	}
	if aprsCfg := m.App.Logbook.APRS; aprsCfg != nil && aprsCfg.Enabled && aprsCfg.RadiusKm > 0 {
		radiusKm = float64(aprsCfg.RadiusKm)
	}
	cutoff := time.Now().Add(-60 * time.Minute)

	// Own-station filtering. While transmitting, hide only our own beacon
	// echo — the exact callsign we send with (an omitted SSID equals -0
	// per the APRS spec); other SSIDs of our callsign ("SP9SPM-0") are
	// real stations and stay in the list. In receive-only mode nothing is
	// transmitted, so all SSIDs of our own callsign are only QTH clutter
	// and are hidden.
	ownCall := ""
	ownBase := ""
	if m.App.Logbook != nil {
		if aprsCfg := m.App.Logbook.APRS; aprsCfg != nil && aprsCfg.Enabled && aprsCfg.SendLocation {
			ownCall = aprsCfg.Callsign
			if ownCall == "" {
				base := m.App.Logbook.Station.Callsign
				if idx := strings.IndexAny(base, "/"); idx >= 0 {
					base = base[:idx]
				}
				if base != "" {
					ownCall = base + "-10"
				}
			}
			ownCall = aprs.CanonicalCall(ownCall)
		} else {
			base := ""
			if m.App.Logbook.APRS != nil {
				base = aprs.BaseCall(m.App.Logbook.APRS.Callsign)
			}
			if base == "" {
				base = aprs.BaseCall(m.App.Logbook.Station.Callsign)
			}
			ownBase = base
		}
	}

	list := make([]aprsStation, 0, len(records))
	for _, r := range records {
		if r.LastHeard.Before(cutoff) {
			continue
		}
		if ownCall != "" && aprs.CanonicalCall(r.Callsign) == ownCall {
			continue
		}
		if ownBase != "" && aprs.BaseCall(r.Callsign) == ownBase {
			continue
		}
		distKm := 0.0
		bearing := 0
		if stLat != 0 && stLon != 0 {
			distKm = geo.HaversineKm(stLat, stLon, r.Lat, r.Lon)
			bearing = int(geo.BearingDeg(stLat, stLon, r.Lat, r.Lon) + 0.5)
			if radiusKm > 0 && distKm > radiusKm {
				continue
			}
		}
		list = append(list, aprsStation{
			rec:     r,
			distKm:  distKm,
			bearing: bearing,
			grid:    latLonToGrid(r.Lat, r.Lon, 6),
			dtype:   aprs.PacketType(r.RawPacket),
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].distKm < list[j].distKm })

	// Keep the selection on the same callsign when the list refreshes.
	prevCall := ""
	if st.sel >= 0 && st.sel < len(st.stations) {
		prevCall = st.stations[st.sel].rec.Callsign
	}
	st.listAll = list
	st.stations = nil // aprsApplyFilters rebuilds the view
	m.aprsApplyFilters(prevCall)
	st.filterSig = ""
}

// aprsApplyFilters rebuilds the filtered station view from listAll using the
// active distance, last-heard, and type filters. The selection follows the
// previously selected callsign when it is still visible.
func (m *Model) aprsApplyFilters(prevCall string) {
	st := &m.aprsPane

	var cutoff time.Time
	if st.timeFilter > 0 {
		cutoff = time.Now().Add(-time.Duration(st.timeFilter) * time.Minute)
	}
	view := make([]aprsStation, 0, len(st.listAll))
	for _, s := range st.listAll {
		if st.distFilter > 0 && s.distKm > float64(st.distFilter) {
			continue
		}
		if !cutoff.IsZero() && s.rec.LastHeard.Before(cutoff) {
			continue
		}
		if st.typeFilter == "operators" && !aprsIsOperatorSymbol(s.rec.Symbol) {
			continue
		}
		view = append(view, s)
	}
	st.stations = view
	st.sel = 0
	if prevCall != "" {
		for i, s := range view {
			if s.rec.Callsign == prevCall {
				st.sel = i
				break
			}
		}
	}
	st.tableReady = false
	st.filterSig = ""
	st.detailSig = ""
}

// aprsRadiusKm returns the configured APRS radius from the active logbook
// config, 0 when unset.
func (m *Model) aprsRadiusKm() int {
	if m.App == nil || m.App.Logbook == nil || m.App.Logbook.APRS == nil {
		return 0
	}
	return m.App.Logbook.APRS.RadiusKm
}

// aprsClosestDistStep returns the distance filter step closest to the given
// radius, or 0 (all) when no radius is configured.
func aprsClosestDistStep(radius int) int {
	if radius <= 0 {
		return 0
	}
	best, bestDiff := aprsDistFilterChoices[1], radius
	for _, step := range aprsDistFilterChoices[1:] {
		d := step - radius
		if d < 0 {
			d = -d
		}
		if d < bestDiff {
			best, bestDiff = step, d
		}
	}
	return best
}

// aprsEnterPane refreshes the station list and aligns the distance filter
// with the configured APRS radius, so opening the pane starts at the range
// the operator actually receives. The alignment runs on entry only — manual
// filter changes are kept while the pane stays open.
func (m *Model) aprsEnterPane() {
	m.aprsPaneRefresh()
	if radius := m.aprsRadiusKm(); radius > 0 {
		st := &m.aprsPane
		st.distFilter = aprsClosestDistStep(radius)
		for i, step := range aprsDistFilterChoices {
			if step == st.distFilter {
				st.distIdx = i
				break
			}
		}
		m.aprsApplyFilters(selectedCall(m))
	}
}

// aprsPaneSel returns the selected station or nil.
func (m *Model) aprsPaneSel() *aprsStation {
	if m.aprsPane.sel >= 0 && m.aprsPane.sel < len(m.aprsPane.stations) {
		return &m.aprsPane.stations[m.aprsPane.sel]
	}
	return nil
}

// aprsPaneSelect moves the selection to the given row.
func (m *Model) aprsPaneSelect(row int) {
	st := &m.aprsPane
	if len(st.stations) == 0 {
		st.sel = 0
		return
	}
	if row < 0 {
		row = 0
	}
	if row >= len(st.stations) {
		row = len(st.stations) - 1
	}
	st.sel = row
	st.detailSig = ""
}

// aprsFillFromSelected fills the QSO form with the selected station's
// callsign and grid, then returns a Cmd that triggers callbook lookups.
func (m *Model) aprsFillFromSelected() tea.Cmd {
	st := m.aprsPaneSel()
	if st == nil {
		return nil
	}

	// Log the bare callsign — APRS SSIDs ("SP9ABC-10") are connection
	// identifiers, not part of the station's callsign.
	call := qso.NormalizeCall(st.rec.Callsign)
	if i := strings.IndexByte(call, '-'); i >= 0 {
		call = call[:i]
	}
	if call == "" {
		return nil
	}

	prevCall := qso.NormalizeCall(m.fields[fieldCall].Value())
	if !strings.EqualFold(call, prevCall) {
		m.lookup.partnerData = nil
		m.lookup.wlPrivateData = nil
		m.lookup.wlLookupDone = false
		m.invalidatePartnerMapCache()
	}
	m.fields[fieldCall].SetValue(call)
	m.fields[fieldGrid].SetValue(st.grid)
	// Clear callbook-populated fields — APRS carries no name/QTH/country.
	m.fields[fieldName].SetValue("")
	m.fields[fieldQTH].SetValue("")
	m.fields[fieldCountry].SetValue("")
	applog.Info("APRS: populated QSO form from station",
		"call", call,
		"grid", st.grid,
		"dist_km", fmt.Sprintf("%.1f", st.distKm),
	)
	return m.lookupCallCmd(call)
}

// handleAPRSUpdate routes messages for the F3 APRS pane.
func (m *Model) handleAPRSUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	st := &m.aprsPane
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		st.filterSig = ""
		st.detailSig = ""
		return m, cmd

	case tickMsg:
		// Refresh the list periodically while visible (~10 s).
		if m.tickCount%aprsReloadIntervalTicks == 0 {
			m.aprsPaneRefresh()
		}
		return m, cmd

	case tea.KeyPressMsg:
		k := msg.String()
		switch k {
		case "esc":
			m.screen = screenQSO
			return m, cmd

		case "t":
			st.timeIdx = (st.timeIdx + 1) % len(aprsTimeFilterChoices)
			st.timeFilter = aprsTimeFilterChoices[st.timeIdx]
			m.aprsApplyFilters(selectedCall(m))
			return m, cmd
		case "d":
			st.distIdx = (st.distIdx + 1) % len(aprsDistFilterChoices)
			st.distFilter = aprsDistFilterChoices[st.distIdx]
			m.aprsApplyFilters(selectedCall(m))
			return m, cmd
		case "s":
			st.typeIdx = (st.typeIdx + 1) % len(aprsTypeFilterChoices)
			st.typeFilter = aprsTypeFilterChoices[st.typeIdx]
			m.aprsApplyFilters(selectedCall(m))
			return m, cmd
		case "backspace":
			st.distFilter, st.distIdx = 0, 0
			st.timeFilter, st.timeIdx = 0, 0
			st.typeFilter, st.typeIdx = aprsTypeFilterChoices[0], 0
			m.aprsApplyFilters(selectedCall(m))
			return m, cmd

		case "b":
			// Manual beacon — send the current position now.
			if m.App == nil {
				return m, cmd
			}
			if err := m.App.SendAPRSBeaconNow(); err != nil {
				m.toasts.Warn("APRS beacon: " + err.Error())
			}
			return m, cmd

		case "enter":
			// Fill the QSO form with the selected station and jump back.
			lookupCmd := m.aprsFillFromSelected()
			m.screen = screenQSO
			cmd = tea.Batch(cmd, lookupCmd)
			return m, cmd
		}

		// Navigation keys — forwarded to the table, same as the DXC pane.
		if st.tableReady {
			t, c := st.table.Update(msg)
			st.table = t
			m.aprsSyncSelection()
			if c != nil {
				cmd = tea.Batch(cmd, c)
			}
		}
	}
	return m, cmd
}

// aprsBeaconConfigured reports whether the active logbook is set up to
// transmit APRS beacons (global integration + logbook + SendLocation).
func (m *Model) aprsBeaconConfigured() bool {
	if m.App == nil || m.App.Logbook == nil || !m.App.Config.Integrations.APRS.Enabled {
		return false
	}
	cfg := m.App.Logbook.APRS
	return cfg != nil && cfg.Enabled && cfg.SendLocation
}

// selectedCall returns the callsign of the currently selected station,
// used to keep the selection stable across filter changes.
func selectedCall(m *Model) string {
	if s := m.aprsPaneSel(); s != nil {
		return s.rec.Callsign
	}
	return ""
}

// aprsFilterLine renders the compact filter row below the header — same
// style as the DXC pane: dim label, value (info-blue when the filter is
// active), dim middot separators, key hints, and a station count.
func (m *Model) aprsFilterLine(w int) string {
	st := &m.aprsPane
	sig := fmt.Sprintf("%d|%d|%d|%s|%d", w, st.distFilter, st.timeFilter, st.typeFilter, len(st.stations))
	if st.filterView != "" && st.filterSig == sig {
		return st.filterView
	}

	distVal := "all"
	if st.distFilter > 0 {
		distVal = fmt.Sprintf("%dkm", st.distFilter)
	}
	timeVal := "all"
	if st.timeFilter > 0 {
		timeVal = fmt.Sprintf("%dm", st.timeFilter)
	}
	typeVal := "all"
	if st.typeFilter == "operators" {
		typeVal = "operators"
	}

	addPart := func(label, value string, active bool) string {
		v := ValueStyle.Render(value)
		if active {
			v = S.Info.Render(value)
		}
		return DimStyle.Render(label) + " " + v
	}
	sep := "  " + DimStyle.Render(middot()) + "  "
	compactParts := []string{
		addPart("Distance", distVal, st.distFilter > 0),
		addPart("Type", typeVal, st.typeFilter != ""),
		addPart("Last heard", timeVal, st.timeFilter > 0),
		DimStyle.Render("Stations") + " " + ValueStyle.Render(strconv.Itoa(len(st.stations))),
	}
	compact := " " + strings.Join(compactParts, sep)

	hintParts := []string{
		addPart("Distance", distVal, st.distFilter > 0) + "  " + DimStyle.Render("(d)"),
		addPart("Type", typeVal, st.typeFilter != "") + "  " + DimStyle.Render("(s)"),
		addPart("Last heard", timeVal, st.timeFilter > 0) + "  " + DimStyle.Render("(t)"),
		DimStyle.Render("Stations") + " " + ValueStyle.Render(strconv.Itoa(len(st.stations))) + "  " + DimStyle.Render("(Bksp clear)"),
	}
	hinted := " " + strings.Join(hintParts, sep)

	line := hinted
	if lipgloss.Width(hinted) > w {
		line = compact
	}
	line = padOrTrunc(line, w)
	st.filterView = line
	st.filterSig = sig
	return line
}

// viewAPRS renders the F3 APRS pane: a DXC-style station table on the left
// and a detail panel on the right, both in rounded border boxes.
func (m *Model) viewAPRS(l Layout) string {
	st := &m.aprsPane
	w := l.TerminalW
	if w < 40 {
		w = 80
	}
	ch := l.ContentH
	if ch < 6 {
		ch = 6
	}

	var b strings.Builder
	b.WriteString(S.Title.Width(w).Render("APRS \u2014 Nearby Stations"))
	b.WriteString("\n")
	// Without a station grid no distance, bearing, or radar is computable —
	// render nothing but the hint.
	if m.effectiveGrid() == "" {
		msg := DimStyle.Width(w).Align(lipgloss.Center).Render("Station grid not set \u2014 enter your grid locator in the station settings")
		return b.String() + msg
	}
	b.WriteString(m.aprsFilterLine(w))
	b.WriteString("\n\n")

	// Empty / unavailable states — the own-station status panel stays
	// visible on the right even when there is nothing to list.
	if m.App == nil {
		msg := DimStyle.Width(w).Align(lipgloss.Center).Render("APRS not receiving \u2014 enable APRS in Integration settings")
		return b.String() + msg
	}
	if m.App.APRSCache == nil {
		return b.String() + m.aprsEmptyLayout("APRS not receiving \u2014 enable APRS in Integration settings", l)
	}
	if len(st.listAll) == 0 {
		return b.String() + m.aprsEmptyLayout("No nearby stations heard in the last 60 minutes", l)
	}
	if len(st.stations) == 0 {
		return b.String() + m.aprsEmptyLayout("No stations match the current filters", l)
	}

	listW, detailW := aprsSplit(l.ContentW)

	tableH := ch - 3 // title + filter line + spacer
	if tableH < 3 {
		tableH = 3
	}
	// Border boxes cost 4 cells of width (2 border + 2 padding) and 2 rows
	// of height — the table is built for the inner area.
	tableInnerW := listW - 4
	if tableInnerW < 40 {
		tableInnerW = 40
	}
	tableInnerH := tableH - 2
	if tableInnerH < 3 {
		tableInnerH = 3
	}
	if !st.tableReady || st.builtW != tableInnerW || st.builtH != tableInnerH {
		m.buildAPRSTable(tableInnerW, tableInnerH)
		m.aprsSyncSelection()
	}

	tablePart := borderBoxStyle.Width(listW).Height(tableH).Render(st.table.View())
	// The right column builds its own bordered boxes (own status on top,
	// selected contact + radar below) to exactly tableH rows.
	detailPart := m.aprsRightPanel(st, detailW, tableH)
	return b.String() + lipgloss.JoinHorizontal(lipgloss.Top, tablePart, detailPart)
}

// aprsSplit computes the list/detail column widths for a given content
// width. The two bordered columns sit flush against each other (like the
// tab bar) and together span the full content width; each column's 4 cells
// of border/padding chrome are included in its width.
func aprsSplit(cw int) (int, int) {
	if cw < 40 {
		cw = 40
	}
	listW := cw * 45 / 100
	if listW < 44 {
		listW = 44
	}
	if listW > 52 {
		listW = 52
	}
	detailW := cw - listW
	if detailW < 24 {
		detailW = 24
	}
	return listW, detailW
}

// aprsEmptyLayout renders the empty-state layout: the message fills the
// left bordered list box while the own-station status panel stays visible
// on the right, so the operator's TX status is never hidden.
func (m *Model) aprsEmptyLayout(msg string, l Layout) string {
	listW, detailW := aprsSplit(l.ContentW)
	tableH := l.ContentH - 3 // title + filter line + spacer
	if tableH < 3 {
		tableH = 3
	}
	innerH := tableH - 2
	if innerH < 1 {
		innerH = 1
	}
	left := fillBody(DimStyle.Width(listW-4).Align(lipgloss.Center).Render(msg), innerH)
	leftBox := borderBoxStyle.Width(listW).Height(tableH).Render(left)
	rightBox := m.aprsOwnPanel(detailW, tableH)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

// aprsOwnPanel renders the right column when there is no selection: the
// own-station status block in a single bordered box padded to the full
// panel height.
func (m *Model) aprsOwnPanel(detailW, h int) string {
	if h < 3 {
		h = 3
	}
	panelW := detailW - 2 // content width convention: rows render to panelW-2
	content := fillBody(strings.Join(m.aprsStatusRows(panelW), "\n"), h-2)
	return borderBoxStyle.Width(detailW).Height(h).Render(content)
}

// buildAPRSTable constructs the bubbles/table for nearby stations — same
// component and header style as the DXC cluster table.
func (m *Model) buildAPRSTable(listW, tableH int) {
	st := &m.aprsPane

	// Six columns fit the minimum list width. Per-column padding in the
	// table component costs 2 cells each — keep the base sum at
	// listW-2n (40 at the minimum list width).
	cols := []table.Column{
		{Title: "Sym", Width: 3},
		{Title: "Typ", Width: 4},
		{Title: "Call", Width: 7},
		{Title: "Brg", Width: 4},
		{Title: "Dist", Width: 6},
		{Title: "Age", Width: 4},
	}
	// Distribute extra width: Call first, then Age.
	minTotal := 3 + 4 + 7 + 4 + 6 + 4 + 2*len(cols)
	extra := listW - minTotal
	if extra > 0 {
		add := 3
		if extra < add {
			add = extra
		}
		cols[2].Width += add
		extra -= add
	}
	if extra > 0 {
		cols[5].Width += extra
	}

	var rows []table.Row
	for _, s := range st.stations {
		age := aprsAge(s.rec.LastHeard)
		sym := s.rec.Symbol
		if len(sym) > 2 {
			sym = sym[:2]
		}
		typ := s.dtype
		if typ == "" {
			typ = "\u2014"
		}
		brg := "\u2014"
		dist := "\u2014"
		if s.distKm > 0 {
			brg = fmt.Sprintf("%03d\u00b0", s.bearing)
			dist = fmt.Sprintf("%.1fkm", s.distKm)
		}
		rows = append(rows, table.Row{sym, typ, s.rec.Callsign, brg, dist, age})
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableH),
		table.WithWidth(listW),
	)
	sty := table.DefaultStyles()
	sty.Header = sty.Header.
		BorderForeground(P.TextDim).
		BorderBottom(true).
		Bold(false).
		Foreground(P.Text)
	if st.distFilter > 0 || st.timeFilter > 0 || st.typeFilter != "" {
		sty.Header = sty.Header.Foreground(P.Cursor)
	}
	t.SetStyles(sty)
	t.Focus()

	// Restore the previous selection after a rebuild.
	if st.sel > 0 && st.sel < len(rows) {
		t.GotoTop()
		t.MoveDown(st.sel)
	}

	st.table = t
	st.tableReady = true
	st.builtW = listW
	st.builtH = tableH
	st.sel = t.Cursor()
}

// aprsSyncSelection mirrors the table cursor into the pane state.
func (m *Model) aprsSyncSelection() {
	st := &m.aprsPane
	if !st.tableReady {
		return
	}
	cur := st.table.Cursor()
	if cur < 0 || cur >= len(st.stations) {
		st.sel = 0
	} else {
		st.sel = cur
	}
	st.detailSig = ""
}

// aprsRightPanel builds the right column as two stacked bordered boxes:
// the own-station APRS status box on top, then a box with the selected
// station's details and the radar filling the remaining height.
func (m *Model) aprsRightPanel(st *aprsPaneState, detailW, h int) string {
	if h < 6 {
		h = 6
	}
	sel := m.aprsPaneSel()
	if sel == nil {
		return m.aprsOwnPanel(detailW, h)
	}

	sig := fmt.Sprintf("%d|%d|%s|%s", detailW, h, sel.rec.Callsign, m.aprsTXStatusSig())
	if st.detailView != "" && st.detailSig == sig {
		return st.detailView
	}

	panelW := detailW - 2 // rows render to panelW-2, matching the box content width
	statusBox := borderBoxStyle.Width(detailW).
		Render(strings.Join(m.aprsStatusRows(panelW), "\n"))

	// Contact box gets everything below the status box.
	contactH := h - lipgloss.Height(statusBox)
	if contactH < 4 {
		contactH = 4
	}

	var rows []string
	rows = append(rows, m.aprsDetailRows(sel, panelW)...)

	// Radar uses whatever space remains inside the contact box.
	radarH := contactH - 2 - len(rows) - 1
	if radarH >= 8 {
		rows = append(rows, "")
		if radar := m.aprsRadarRows(st, panelW-2, radarH); len(radar) > 0 {
			rows = append(rows, radar...)
		}
	}

	content := fillBody(strings.Join(rows, "\n"), contactH-2)
	contactBox := borderBoxStyle.Width(detailW).Height(contactH).Render(content)

	joined := lipgloss.JoinVertical(lipgloss.Left, statusBox, contactBox)
	st.detailView = joined
	st.detailSig = sig
	return joined
}

// aprsTXStatusSig captures the own-beacon state that affects the status
// block — the panel must refresh when beacon settings or the last-sent
// timestamp change.
func (m *Model) aprsTXStatusSig() string {
	if m.App == nil || m.App.Logbook == nil {
		return "-"
	}
	cfg := m.App.Logbook.APRS
	if cfg == nil {
		return "rx"
	}
	return fmt.Sprintf("tx|%t|%t|%s|%s|%s", cfg.Enabled, cfg.SendLocation, cfg.Callsign, cfg.Symbol, cfg.LastBeaconAt)
}

// aprsStatusRows renders the compact own-station APRS status block: a
// receive-only notice, or — when beaconing is configured — what is being
// transmitted (callsign with SSID, symbol, comment, interval) and when the
// last frame was transmitted.
func (m *Model) aprsStatusRows(detailW int) []string {
	innerW := detailW - 2
	if innerW < 16 {
		innerW = 16
	}
	row := func(label, value string) string {
		return padOrTrunc(fmt.Sprintf("%-11s %s", S.StatusLabel.Render(label), S.StatusValue.Render(truncateText(value, innerW-12))), innerW)
	}

	cfg := m.App.Logbook.APRS
	if cfg == nil || !cfg.Enabled || !cfg.SendLocation {
		headline := S.StatusLabel.Render("You are in") + " " +
			statusDotWarnStyle.Render("APRS-RX") + " " +
			S.StatusLabel.Render("mode")
		return []string{
			padOrTrunc(headline, innerW),
			DimStyle.Render(padOrTrunc("Receive-only \u2014 you are not", innerW)),
			DimStyle.Render(padOrTrunc("transmitting.", innerW)),
		}
	}

	callsign := cfg.Callsign
	if callsign == "" {
		base := m.App.Logbook.Station.Callsign
		if idx := strings.IndexAny(base, "/"); idx >= 0 {
			base = base[:idx]
		}
		if base != "" {
			callsign = base + "-10"
		}
	}
	symbol := cfg.Symbol
	if symbol == "" {
		symbol = "/-"
	}

	var rows []string
	rows = append(rows,
		S.StatusLabel.Render("Transmitting")+" "+statusDotOnStyle.Render("APRS")+" "+S.StatusLabel.Render("beacon"))
	// Same compact look as the station details: call first, then the
	// symbol with its name ("SP9MOA-10 · /- — House").
	call := callsign
	sym := symbol
	if name := aprs.SymbolName(symbol); name != "" {
		sym += " \u2014 " + name
	}
	call += " \u00b7 " + sym
	rows = append(rows, row("Call", call))
	if cfg.Comment != "" {
		rows = append(rows, row("Comment", cfg.Comment))
	}
	interval := cfg.IntervalMin
	if interval < 5 {
		interval = 5
	}
	if interval > 180 {
		interval = 180
	}
	rows = append(rows, row("Every", fmt.Sprintf("%d min", interval)))
	last := "never"
	if cfg.LastBeaconAt != "" {
		if t, err := time.Parse(time.RFC3339, cfg.LastBeaconAt); err == nil {
			last = t.UTC().Format("15:04Z") + " (" + aprsAge(t) + " ago)"
		}
	}
	rows = append(rows, row("Last TX", last))
	return rows
}

// aprsRadarNearKm is the distance below which a station is considered to be
// at our own position and is marked at the radar center.
const aprsRadarNearKm = 0.1

// aprsRadarRows renders a compact ASCII radar sized to the given box: own
// station at the center, nearby stations plotted by bearing and distance
// using their Yaesu-style type markers (P, p, O, W, ...). Terminal cells
// are about twice as tall as they are wide, so vertical offsets are halved
// (aspect 2) to keep the radar round instead of egg shaped. N/E/S/W labels
// restore azimuth context. Stations sharing one cell collapse into a count
// digit; the selected station is highlighted. The last row is a dim range
// caption with the selected station's bearing and distance.
func (m *Model) aprsRadarRows(st *aprsPaneState, w, h int) []string {
	const aspect = 2.0 // terminal cells are ~2:1, taller than wide
	gridH := h - 1     // last row is the range caption
	innerW := w - 2    // one column each side for the W / E labels
	innerH := gridH - 2
	if w < 10 || innerW < 7 || innerH < 5 {
		return nil
	}

	cx, cy := innerW/2, innerH/2
	rh := float64(cx) // horizontal semi-axis in cells
	if r := float64(innerW - 1 - cx); r < rh {
		rh = r
	}
	if r := aspect * float64(cy); r < rh {
		rh = r
	}
	if r := aspect * float64(innerH-1-cy); r < rh {
		rh = r
	}
	if rh < 2 {
		return nil
	}

	// Range covers all visible stations, at least 10 km. An active
	// distance filter zooms the radar in: the outer ring equals the
	// filter radius, so the 1/5/10 km steps change the scale. A filter
	// wider than the visible stations never upscales the radar.
	maxDist := 10.0
	for i := range st.stations {
		if st.stations[i].distKm > maxDist {
			maxDist = st.stations[i].distKm
		}
	}
	if st.distFilter > 0 && float64(st.distFilter) < maxDist {
		maxDist = float64(st.distFilter)
	}

	grid := make([][]rune, innerH)
	for y := range grid {
		grid[y] = make([]rune, innerW)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}

	// Range rings at 1/3, 2/3 and full range: ellipses in cell space so
	// they render as circles on screen.
	for ring := 1; ring <= 3; ring++ {
		rr := rh * float64(ring) / 3.0
		for y := 0; y < innerH; y++ {
			for x := 0; x < innerW; x++ {
				dx, dy := float64(x-cx), float64(y-cy)
				if math.Abs(math.Sqrt(dx*dx+aspect*aspect*dy*dy)-rr) <= 0.5 && grid[y][x] == ' ' {
					grid[y][x] = '.'
				}
			}
		}
	}

	// Plot stations; collisions stack into a count. Stations effectively
	// at our position are marked at the center instead of the ring edge.
	// Each single-station cell carries the station's Yaesu-style marker.
	type cell struct {
		count int
		sel   bool
		typ   rune // type marker for single-station cells, '*' fallback
	}
	cells := map[[2]int]*cell{}
	selKey := [2]int{-1, -1}
	for i := range st.stations {
		s := &st.stations[i]
		if s.distKm <= 0 {
			continue
		}
		x, y := cx, cy
		if s.distKm > aprsRadarNearKm {
			b := float64(s.bearing) * math.Pi / 180
			dx := math.Sin(b) * rh * s.distKm / maxDist
			dy := math.Cos(b) * rh * s.distKm / maxDist
			x = cx + int(math.Round(dx))
			y = cy - int(math.Round(dy/aspect))
		}
		if x < 0 || x >= innerW || y < 0 || y >= innerH {
			continue
		}
		key := [2]int{x, y}
		c := cells[key]
		if c == nil {
			c = &cell{}
			cells[key] = c
		}
		c.count++
		if c.count == 1 {
			c.typ = '*'
			if s.dtype != "" {
				c.typ = rune(s.dtype[0])
			}
		}
		if i == st.sel {
			c.sel = true
			selKey = key
		}
	}

	grid[cy][cx] = '+'
	for key, c := range cells {
		ch := c.typ
		if c.count > 1 {
			if c.count > 9 {
				ch = '+'
			} else {
				ch = rune('0' + c.count)
			}
		}
		grid[key[1]][key[0]] = ch
	}

	// W / E hug the ring instead of the box edges — in wide boxes the ring
	// does not reach the sides and edge labels would float far away from
	// the compass.
	wPos := cx - int(rh)
	if wPos < 0 {
		wPos = 0
	}
	ePos := cx + int(rh) + 2
	if ePos > w-1 {
		ePos = w - 1
	}

	// Assemble the box: N / S rows above and below, W / E right next to
	// the ring on the center row, all aligned with the radar center.
	rows := make([]string, 0, gridH+1)
	centerCol := cx + 1
	centerRow := cy + 1
	for y := 0; y < gridH; y++ {
		var b strings.Builder
		for x := 0; x < w; x++ {
			key := [2]int{x - 1, y - 1}
			// A station cell at the label position wins — the marker is
			// more useful than the compass letter.
			wLabel := y == centerRow && x == wPos && cells[key] == nil
			eLabel := y == centerRow && x == ePos && cells[key] == nil
			var ch rune
			switch {
			case y == 0 && x == centerCol:
				ch = 'N'
			case y == gridH-1 && x == centerCol:
				ch = 'S'
			case wLabel:
				ch = 'W'
			case eLabel:
				ch = 'E'
			case x >= 1 && x <= innerW && y >= 1 && y <= innerH:
				ch = grid[y-1][x-1]
			default:
				ch = ' '
			}
			isSel := key == selKey && cells[key] != nil && cells[key].sel
			// Cardinal labels are identified by position, not character —
			// station type markers E/W are also single letters and must
			// keep the value style.
			isCardinal := (y == 0 && x == centerCol) ||
				(y == gridH-1 && x == centerCol) ||
				wLabel || eLabel
			switch {
			case ch == ' ':
				b.WriteByte(' ')
			case ch == '.':
				b.WriteString(DimStyle.Render("."))
			case isCardinal:
				b.WriteString(DimStyle.Render(string(ch)))
			case x == centerCol && y == centerRow:
				if isSel {
					b.WriteString(CursorStyle.Render("+"))
				} else {
					b.WriteString(S.StatusValue.Render("+"))
				}
			case isSel:
				b.WriteString(CursorStyle.Render(string(ch)))
			default:
				b.WriteString(ValueStyle.Render(string(ch)))
			}
		}
		rows = append(rows, b.String())
	}

	// Bottom range caption: the exact grid sent to APRS (full configured
	// precision, GPS-derived when active) names what sits at the center,
	// and the selected station's callsign, bearing, and distance on the
	// right restore azimuth context at a glance.
	caption := fmt.Sprintf(" ~ %.0f km", maxDist)
	if g := m.effectiveGrid(); g != "" {
		caption += " \u00b7 " + g
	}
	if sel := m.aprsPaneSel(); sel != nil {
		right := fmt.Sprintf(" %s %03d\u00b0", sel.rec.Callsign, sel.bearing)
		if sel.distKm > 0 {
			right += fmt.Sprintf(" \u00b7 %.1f km", sel.distKm)
		}
		right += " "
		// Widths, not byte lengths — the middot and degree sign are 2-byte
		// UTF-8 characters.
		if lipgloss.Width(caption)+lipgloss.Width(right) <= w {
			caption += strings.Repeat(" ", w-lipgloss.Width(caption)-lipgloss.Width(right)) + right
		}
	}
	rows = append(rows, DimStyle.Render(padOrTrunc(caption, w)))
	return rows
}

// aprsDetailRows renders the selected station's details as label/value rows.
func (m *Model) aprsDetailRows(sel *aprsStation, detailW int) []string {
	innerW := detailW - 2
	if innerW < 16 {
		innerW = 16
	}
	row := func(label, value string) string {
		return padOrTrunc(fmt.Sprintf("%-11s %s", S.StatusLabel.Render(label), S.StatusValue.Render(truncateText(value, innerW-12))), innerW)
	}

	s := sel.rec
	var rows []string
	// Callsign and symbol share one compact row, call first.
	call := s.Callsign
	if s.Symbol != "" {
		sym := s.Symbol
		if name := aprs.SymbolName(s.Symbol); name != "" {
			sym += " \u2014 " + name
		}
		call += " \u00b7 " + sym
	}
	rows = append(rows, padOrTrunc(fmt.Sprintf("%-5s %s",
		S.StatusLabel.Render("Call"),
		S.StatusValue.Render(truncateText(call, innerW-6))), innerW))
	// Course and speed ride on the grid line — the panel never shifts when
	// a station starts or stops reporting movement. Distance and bearing
	// live in the table and the radar, not in the details.
	grid := sel.grid
	if s.Course != 0 || s.SpeedKmH != 0 {
		grid += fmt.Sprintf(" \u00b7 crs %d\u00b0 \u00b7 %d km/h", s.Course, s.SpeedKmH)
	}
	rows = append(rows, row("Grid", grid))
	if s.AltitudeM != 0 {
		rows = append(rows, row("Altitude", fmt.Sprintf("%d m", s.AltitudeM)))
	}
	if s.Comment != "" {
		// Emoji and decorative symbols do not render on terminal fonts —
		// sanitize the raw comment before display.
		comment := aprs.CleanComment(s.Comment)
		// Weather stations carry the APRS weather block at the start of
		// the comment — decode it into readable values.
		if len(s.Symbol) == 2 && s.Symbol[1] == '_' {
			if w, ok := aprs.ParseWeather(s.Comment); ok {
				comment = w.Format(m.App.Config.General.Units != "imperial")
			}
		}
		rows = append(rows, padOrTrunc(fmt.Sprintf("%-5s %s",
			S.StatusLabel.Render("Comment"),
			S.StatusValue.Render(truncateText(comment, innerW-8))), innerW))
	}
	// Last heard and source share one compact row, last heard first.
	last := s.LastHeard.UTC().Format("15:04Z") + " (" + aprsAge(s.LastHeard) + " ago)"
	if s.Source != "" {
		last += " \u00b7 " + s.Source
	}
	rows = append(rows, padOrTrunc(fmt.Sprintf("%-5s %s",
		S.StatusLabel.Render("Last"),
		S.StatusValue.Render(truncateText(last, innerW-6))), innerW))
	return rows
}

// aprsAge formats a timestamp as a compact human-readable age ("45s",
// "12m", "1.5h").
func aprsAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// latLonToGrid converts decimal coordinates to an n-character Maidenhead
// locator string.
func latLonToGrid(lat, lon float64, n int) string {
	ll := latlon.NewLatLon(latlon.Latitude(lat), latlon.Longitude(lon))
	g := locator.LatLonToLocator(ll, n)
	return strings.ToUpper(strings.TrimRight(string(g[:]), "\x00"))
}
