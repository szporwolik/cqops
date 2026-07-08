package tui

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/psk"
	"github.com/szporwolik/cqops/internal/store"
)

// pskFetchMsg is sent when an async PSK Reporter fetch completes.
type pskFetchMsg struct {
	reports   []psk.Report
	fetchTime time.Time
	err       error
}

// pskFetchCmd returns a tea.Cmd that fetches PSK data asynchronously.
func (m *Model) pskFetchCmd() tea.Cmd {
	call := strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Callsign))
	cacheDir := m.psk.cacheDir
	return func() tea.Msg {
		reports, ft, err := psk.FetchReports(call, cacheDir)
		return pskFetchMsg{reports: reports, fetchTime: ft, err: err}
	}
}

// pskSpotsLoadedMsg carries spots loaded from SQLite asynchronously.
type pskSpotsLoadedMsg struct {
	spots   []psk.Report
	spotKey string
	err     error
}

// loadPSKSpotsCmd returns a tea.Cmd that queries spots from SQLite
// asynchronously, avoiding DB I/O during View().
func (m *Model) loadPSKSpotsCmd(call string, cutoff int64, spotKey string) tea.Cmd {
	db := m.App.DB
	return func() tea.Msg {
		rawSpots, err := store.QueryPSKSpots(db, call, cutoff)
		if err != nil {
			return pskSpotsLoadedMsg{spotKey: spotKey, err: err}
		}
		spots := make([]psk.Report, len(rawSpots))
		for i, s := range rawSpots {
			spots[i] = psk.Report{
				ReceiverCallsign: s.ReceiverCall,
				ReceiverLocator:  s.ReceiverLoc,
				Frequency:        s.Frequency,
				SNR:              s.SNR,
				Mode:             s.Mode,
				FlowStartSeconds: s.FlowStart,
			}
		}
		sort.Slice(spots, func(i, j int) bool {
			return spots[i].FlowStartSeconds > spots[j].FlowStartSeconds
		})
		return pskSpotsLoadedMsg{spots: spots, spotKey: spotKey}
	}
}

// PSK Reporter time filter steps (minutes).
var pskFilterSteps = []int{5, 15, 30, 60, 120, 360}

// Band marker colors per band — use semantic palette instead of ANSI
// 8-bit codes so bands are clearly distinguishable on modern terminals.
var pskMark160 = lipgloss.NewStyle().Foreground(P.Text).Bold(true)
var pskMark80 = lipgloss.NewStyle().Foreground(P.Primary).Bold(true)
var pskMark40 = lipgloss.NewStyle().Foreground(P.Warning).Bold(true)
var pskMark20 = lipgloss.NewStyle().Foreground(P.Success).Bold(true)
var pskMark15 = lipgloss.NewStyle().Foreground(P.Accent).Bold(true)
var pskMark10 = lipgloss.NewStyle().Foreground(P.Info).Bold(true)
var pskMarkOther = lipgloss.NewStyle().Foreground(P.Error).Bold(true)

// pskHeaderStyle colors the PSK Reporter table header — avoids DimStyle's
// gray-on-gray appearance on dark terminals.
var pskHeaderStyle = lipgloss.NewStyle().Foreground(P.Info).Bold(true)

// pskHintStyle colors the "N more above/below" scroll hint.
var pskHintStyle = lipgloss.NewStyle().Foreground(P.TextMuted)

func pskBandStyle(freqHz float64) lipgloss.Style {
	switch freqToBandName(freqHz) {
	case "160m":
		return pskMark160
	case "80m":
		return pskMark80
	case "40m":
		return pskMark40
	case "20m":
		return pskMark20
	case "15m":
		return pskMark15
	case "10m":
		return pskMark10
	default:
		return pskMarkOther
	}
}

// pskResetCaches clears the PSK spot cache, view cache, and resets the
// cursor. Call after any filter change (band, mode, time) so the next
// View() re-queries the DB and re-renders from scratch.
func (m *Model) pskResetCaches() {
	m.psk.selected = 0
	m.psk.spotKey = ""
	m.psk.viewKey = ""
	m.psk.view = ""
}

// pskApplyFilters narrows the spot list by band, mode, and time window
// in-memory. The DB query only filters by time; band/mode narrowing and
// a stricter time check are applied here so the table and map update
// instantly on filter change without waiting for an async DB reload.
// The time cutoff ensures cached spots from a wider window don't leak
// through when the user switches to a narrower time filter.
func pskApplyFilters(spots []psk.Report, bandFilter, modeFilter string, since int64) []psk.Report {
	if bandFilter == "" && modeFilter == "" && since == 0 {
		return spots
	}
	var out []psk.Report
	for _, r := range spots {
		if since > 0 && r.FlowStartSeconds < since {
			continue
		}
		if bandFilter != "" && freqToBandName(r.Frequency) != bandFilter {
			continue
		}
		if modeFilter != "" && !strings.EqualFold(r.Mode, modeFilter) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func pskTableRow(r psk.Report, callW, gridW, freqW, snrW, modeW int) string {
	call := truncateText(r.ReceiverCallsign, callW)
	loc := truncateText(r.ReceiverLocator, gridW)
	freq := truncateText(strconv.FormatFloat(r.Frequency/1_000_000, 'f', 3, 64), freqW)
	snr := ""
	if r.SNR != 0 {
		snr = truncateText(strconv.Itoa(r.SNR), snrW)
	}
	mode := truncateText(r.Mode, modeW)
	age := formatAge(r.FlowStartSeconds)

	// Manual padding via strings.Builder — avoids fmt.Sprintf allocation.
	var sb strings.Builder
	padRight(&sb, call, callW)
	sb.WriteByte(' ')
	padRight(&sb, loc, gridW)
	sb.WriteByte(' ')
	padLeft(&sb, freq, freqW)
	sb.WriteByte(' ')
	padLeft(&sb, snr, snrW)
	sb.WriteByte(' ')
	padRight(&sb, mode, modeW)
	sb.WriteByte(' ')
	sb.WriteString(age)
	return sb.String()
}

// padRight appends s to sb with space-padding to reach at least targetW.
func padRight(sb *strings.Builder, s string, targetW int) {
	sb.WriteString(s)
	for i := len(s); i < targetW; i++ {
		sb.WriteByte(' ')
	}
}

// padLeft appends s to sb with leading space-padding to reach at least targetW.
func padLeft(sb *strings.Builder, s string, targetW int) {
	for i := len(s); i < targetW; i++ {
		sb.WriteByte(' ')
	}
	sb.WriteString(s)
}

func formatAge(ts int64) string {
	d := time.Since(time.Unix(ts, 0))
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	default:
		return strconv.Itoa(int(d.Hours())) + "h"
	}
}

func (m *Model) viewPSKReporter() string {
	w := m.width
	if w < 30 {
		w = 80
	}

	if m.psk.cacheDir == "" {
		return fillBody(DimStyle.Render("Cache directory unavailable"), contentHeight(m.height))
	}

	// PSK Reporter is about "who heard me?" — always use the station callsign.
	call := strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Callsign))
	if call == "" {
		return fillBody(DimStyle.Render("Station callsign not set — check station config"), contentHeight(m.height))
	}

	if !m.inetOnline {
		return fillBody(DimStyle.Render("PSK Reporter unavailable — no internet connection"), contentHeight(m.height))
	}

	// --- Output cache: skip ALL expensive work when inputs unchanged. ---
	// Include the grayline 5-min slot so the terminator updates without a
	// data fetch — same logic as mapRenderer.renderBase().
	var graySlot int
	if m.App.Config.General.DrawGrayline {
		now := time.Now().UTC()
		graySlot = now.Hour()*12 + now.Minute()/5
	}
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(w))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(m.height))
	sb.WriteByte('|')
	sb.WriteString(call)
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(m.psk.filterMins))
	sb.WriteByte('|')
	sb.WriteString(m.psk.bandFilter)
	sb.WriteByte('|')
	sb.WriteString(m.psk.modeFilter)
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(m.psk.selected))
	sb.WriteByte('|')
	sb.WriteString(strconv.FormatInt(m.psk.lastFetchByCall[call].Unix(), 10))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(graySlot))
	sb.WriteByte('|')
	sb.WriteString(strconv.FormatBool(m.psk.fetching))
	sb.WriteByte('|')
	sb.WriteString(m.psk.spotKey)
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(len(m.psk.spots)))
	// Kitty frame readiness — when the PSK kitty grid arrives its
	// content length changes, busting the outer view cache.
	if m.mapView != nil {
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(len(m.mapView.PSKView())))
	}
	sig := sb.String()
	// While waiting for the Kitty frame, skip the outer cache so
	// buildPSKMap runs every frame and SetPSKImage gets dispatched.
	// PSKView() returns the glyph fallback when kitty isn't ready,
	// so we must check PSKKittyReady() — not just non-empty.
	kittyPending := m.mapView != nil && m.mapView.kittyOn &&
		picture.KittySupported() == picture.KittyCapabilitySupported &&
		!m.mapView.PSKKittyReady()
	if !kittyPending && m.psk.viewKey == sig && m.psk.view != "" {
		return m.psk.view
	}

	// --- Spot cache: skip SQL when filters unchanged. ---
	sb.Reset()
	sb.WriteString(call)
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(m.psk.filterMins))
	sb.WriteByte('|')
	sb.WriteString(m.psk.bandFilter)
	sb.WriteByte('|')
	sb.WriteString(m.psk.modeFilter)
	spotKey := sb.String()
	var filtered []psk.Report
	if m.psk.spotKey == spotKey && len(m.psk.spots) > 0 {
		filtered = m.psk.spots
	} else {
		// Spot cache miss — set flag for async DB load. Fall back to
		// existing cached spots (if any) so the table isn't blank while
		// waiting; the in-memory band/mode filter still applies below.
		m.psk.needDBLoad = true
		m.psk.pendingSpotKey = spotKey
		m.psk.pendingCall = call
		m.psk.pendingCutoff = time.Now().UTC().Add(-time.Duration(m.psk.filterMins) * time.Minute).Unix()
		filtered = m.psk.spots
	}

	// Apply in-memory filters. The DB query only filters by time window;
	// band/mode/time narrowing happens here so the user sees immediate
	// results without waiting for an async DB reload. The time cutoff
	// prevents stale cached spots from a wider window leaking through
	// when the user switches to a narrower time filter.
	cutoff := time.Now().UTC().Add(-time.Duration(m.psk.filterMins) * time.Minute).Unix()
	filtered = pskApplyFilters(filtered, m.psk.bandFilter, m.psk.modeFilter, cutoff)

	// Clamp cursor only when there are reports.
	if len(filtered) > 0 {
		if m.psk.selected >= len(filtered) {
			m.psk.selected = len(filtered) - 1
		}
		if m.psk.selected < 0 {
			m.psk.selected = 0
		}
	}

	totalW := w - 2
	if totalW > partnerMapMaxW {
		totalW = partnerMapMaxW
	}
	if totalW < 60 {
		totalW = w - 2
	}

	// Two boxes side by side: table (70%) + filters (30%).
	tableW := totalW * 70 / 100
	filtW := totalW - tableW
	if filtW < 20 {
		filtW = 20
		tableW = totalW - filtW
	}

	// Build table content — always 5 visible rows for consistent layout.
	const fixedRows = 5
	var tableContent string
	if len(filtered) == 0 {
		// Show status inside the table box to avoid layout shift.
		var msg string
		if m.psk.fetching {
			msg = fmt.Sprintf("Fetching PSK Reporter data for %s\u2026", call)
		} else if !m.psk.fetched {
			msg = fmt.Sprintf("Press F5 to fetch PSK Reporter data for %s", call)
		} else {
			msg = fmt.Sprintf("Nobody heard %s in the last %d minutes", call, m.psk.filterMins)
		}
		tableContent = DimStyle.Render(msg)
		for i := 1; i < 7; i++ {
			tableContent += "\n"
		}
	} else {
		tableContent = m.buildPSKTable(filtered, tableW-6, fixedRows)
	}
	tableBox := m.renderPartnerBox(
		fmt.Sprintf("Heard by (%d)", len(filtered)),
		tableContent, tableW, 0)

	// Build filters box.
	filtContent := m.buildPSKFilters(filtW - 6)
	filtBox := m.renderPartnerBox("Filters", filtContent, filtW, 0)

	// Equalize heights.
	th := lipgloss.Height(tableBox)
	fh := lipgloss.Height(filtBox)
	maxH := th
	if fh > maxH {
		maxH = fh
	}
	if th < maxH {
		tableBox += strings.Repeat("\n", maxH-th)
	}
	if fh < maxH {
		filtBox += strings.Repeat("\n", maxH-fh)
	}
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, tableBox, filtBox)

	var block string
	if m.App.Config.General.RenderMap && m.mapView != nil {
		mapW := totalW
		topH := lipgloss.Height(topRow)
		mapAvailH := contentHeight(m.height) - topH - 1 // -1 for legend line
		if mapAvailH < 3 {
			mapAvailH = 3
		}
		contentW := mapW - 4
		if contentW < 20 {
			contentW = mapW
		}

		var mapBox string
		if len(filtered) > 0 {
			mapBox = m.buildPSKMap(filtered, contentW, mapAvailH)
		} else {
			mapBox = m.buildPSKMap(nil, contentW, mapAvailH)
		}
		if mapBox != "" {
			mapBox = lipgloss.PlaceHorizontal(mapW, lipgloss.Center, mapBox)
			block = lipgloss.JoinVertical(lipgloss.Left, topRow, mapBox)
		} else {
			block = topRow
		}
	} else {
		block = topRow
	}

	if w > totalW+2 {
		block = PartnerBlock.Width(w).Render(block)
	}

	// Clamp to content height so the help bar never gets pushed off-screen.
	ch := contentHeight(m.height)
	h := lipgloss.Height(block)
	if h > ch && ch > 0 {
		lines := strings.Split(block, "\n")
		if len(lines) > ch {
			block = strings.Join(lines[:ch], "\n")
		}
	}
	result := fillBody(block, ch)
	m.psk.view = result
	m.psk.viewKey = sig
	return result
}

func (m *Model) buildPSKTable(reports []psk.Report, maxW, visibleRows int) string {
	// Dynamically size columns to fill available width without wrapping.
	// Fixed column widths — never shrink, extra space goes to right padding.
	callW := 14
	gridW := 10
	freqW := 8
	snrW := 4
	modeW := 6

	header := pskHeaderStyle.Width(maxW).MaxWidth(maxW).Inline(true).Render(
		fmt.Sprintf("%-*s %-*s %*s %*s %-*s %s",
			callW, "Call", gridW, "Grid", freqW, "Freq", snrW, "SNR", modeW, "Mode", "Age"))
	var lines []string
	lines = append(lines, header)

	// Paginate in blocks of visibleRows.
	pageSize := visibleRows
	start := (m.psk.selected / pageSize) * pageSize
	if start+pageSize > len(reports) {
		start = len(reports) - pageSize
		if start < 0 {
			start = 0
		}
	}
	end := start + pageSize
	if end > len(reports) {
		end = len(reports)
	}

	for i := start; i < end; i++ {
		row := pskTableRow(reports[i], callW, gridW, freqW, snrW, modeW)
		style := ValueStyle
		if i == m.psk.selected {
			style = CursorStyle.Bold(true)
		}
		lines = append(lines, style.Render(row))
	}
	// Pad with empty lines to always fill visibleRows slots.
	for i := end - start; i < visibleRows; i++ {
		lines = append(lines, "")
	}

	// Always show scroll hint row for consistent layout.
	above := start
	below := len(reports) - end
	if above > 0 || below > 0 {
		hint := ""
		if above > 0 {
			hint += fmt.Sprintf("\u2191 %d more above", above)
		}
		if below > 0 {
			if hint != "" {
				hint += "   "
			}
			hint += fmt.Sprintf("\u2193 %d more below", below)
		}
		lines = append(lines, pskHintStyle.Render(hint))
	} else {
		lines = append(lines, "")
	}

	// Force each line to exactly maxW width so the table fills the box.
	// Use a cached style — rebuilt only when maxW changes.
	if m.psk.tableRowStyleW != maxW {
		m.psk.tableRowStyle = lipgloss.NewStyle().Width(maxW).MaxWidth(maxW).Inline(true)
		m.psk.tableRowStyleW = maxW
	}
	for i, l := range lines {
		lines[i] = m.psk.tableRowStyle.Render(l)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// buildPSKFilters renders the current filter state in the right-hand box.
func (m *Model) buildPSKFilters(maxW int) string {
	var lines []string
	// Header is rendered by renderPartnerBox — no duplicate here.

	indent := " "
	add := func(label, value string) {
		lbl := S.FormLabel.Align(lipgloss.Right).Render(label)
		valW := maxW - 14
		if valW < 3 {
			valW = 3
		}
		val := ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(truncateText(value, valW))
		lines = append(lines, indent+lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ", val))
	}

	timeLabel := fmt.Sprintf("%d min", m.psk.filterMins)
	add("Time window", timeLabel)

	bandLabel := "all"
	if m.psk.bandFilter != "" {
		bandLabel = m.psk.bandFilter
	}
	add("Band filter", bandLabel)

	modeLabel := "all"
	if m.psk.modeFilter != "" {
		modeLabel = m.psk.modeFilter
	}
	add("Mode filter", modeLabel)

	nextUpdate := ""
	pskCall := ""
	if m.App != nil {
		pskCall = strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Callsign))
	}
	if last := m.psk.lastFetchByCall[pskCall]; !last.IsZero() {
		nextFetch := last.Add(5 * time.Minute)
		remaining := time.Until(nextFetch)
		if remaining > 0 {
			nextUpdate = fmt.Sprintf("%dm", int(remaining.Minutes())+1)
		} else {
			nextUpdate = "now"
		}
	}
	if nextUpdate != "" {
		add("Next update", nextUpdate)
	}

	// Pad so the box height matches the table (header + 5 data + scroll hint = 7).
	for len(lines) < 7 {
		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) buildPSKMap(reports []psk.Report, mapW, mapAvailH int) string {
	ownGrid := m.effectiveGrid()
	if ownGrid == "" {
		return ""
	}

	// Map-with-spots cache — key includes kitty mode (different render paths).
	kittyOn := m.mapView != nil && m.mapView.kittyOn &&
		picture.KittySupported() == picture.KittyCapabilitySupported
	var graySlot int
	if m.App.Config.General.DrawGrayline {
		now := time.Now().UTC()
		graySlot = now.Hour()*12 + now.Minute()/5
	}
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(len(reports)))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(mapW))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(mapAvailH))
	sb.WriteByte('|')
	sb.WriteString(ownGrid)
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(graySlot))
	sb.WriteByte('|')
	sb.WriteString(strconv.FormatBool(kittyOn))
	if len(reports) > 0 {
		sb.WriteByte('|')
		sb.WriteString(reports[0].ReceiverCallsign)
		sb.WriteByte('|')
		sb.WriteString(strconv.FormatInt(reports[0].FlowStartSeconds, 10))
		sb.WriteByte('|')
		sb.WriteString(reports[len(reports)-1].ReceiverCallsign)
		sb.WriteByte('|')
		sb.WriteString(strconv.FormatInt(reports[len(reports)-1].FlowStartSeconds, 10))
	}
	mapSig := sb.String()
	if m.psk.mapSig == mapSig && m.psk.mapView != "" {
		return m.psk.mapView
	}

	ownLat, ownLon := gridToLatLon(ownGrid)

	if m.mapView == nil {
		return ""
	}

	if kittyOn {
		applog.Debug("PSK: kitty path", "reports", len(reports), "mapSig", mapSig)
		result := m.buildPSKMapKitty(reports, ownLat, ownLon, mapW, mapAvailH, mapSig)
		if result != "" {
			applog.Debug("PSK: kitty OK", "len", len(result))
			return result
		}
		applog.Debug("PSK: kitty pending, fallback to ANSI")
		// Kitty frame not ready — fall through to ANSI path below.
	} else if m.mapView != nil {
		applog.Debug("PSK: kitty OFF",
			"kittyOn", m.mapView.kittyOn,
			"kittySupported", picture.KittySupported(),
			"config", m.App.Config.General.KittyGraphics,
		)
	}

	// --- ANSI path ---
	baseMap := m.mapView.BaseImage(mapW, mapAvailH, m.App.Config.General.DrawGrayline)
	if baseMap == "" {
		return ""
	}

	lines := strings.Split(baseMap, "\n")
	mapH := len(lines)
	actualW := 0
	if mapH > 0 {
		actualW = lipgloss.Width(lines[0])
	}
	if actualW < 20 {
		actualW = mapW
	}

	for i := len(reports) - 1; i >= 0; i-- {
		r := reports[i]
		if r.ReceiverLocator == "" {
			continue
		}
		rlat, rlon := gridToLatLon(r.ReceiverLocator)
		cx, cy := pskCellCoords(rlat, rlon, actualW, mapH)
		if cy >= 0 && cy < mapH && cx >= 0 && cx < actualW {
			mark := pskBandStyle(r.Frequency).Render("\u25cf")
			lines[cy] = replaceANSICell(lines[cy], cx, mark)
		}
	}

	ownX, ownY := pskCellCoords(ownLat, ownLon, actualW, mapH)
	if ownY >= 0 && ownY < mapH && ownX >= 0 && ownX < actualW {
		lines[ownY] = replaceANSICell(lines[ownY], ownX, S.MapOwn.Render("\u25c6"))
	}

	legend := pskLegend()
	lines = append(lines, legend)
	result := strings.Join(lines, "\n")
	// Only cache when kitty is off; when kitty is on but the frame
	// hasn't arrived yet, skip the cache so the kitty path can
	// take over on the next frame.
	if !kittyOn {
		m.psk.mapView = result
		m.psk.mapSig = mapSig
	}
	return result
}

// buildPSKMapKitty renders the PSK map at full resolution via Kitty protocol.
// Mirrors the partner map's renderKitty() — only dispatches SetImage/SetSize
// when data actually changed (guarded by mapSig), avoiding seq flooding.
func (m *Model) buildPSKMapKitty(reports []psk.Report, ownLat, ownLon float64, mapW, mapAvailH int, mapSig string) string {
	rgba, pixW, pixH, adjW, adjH := m.mapView.BaseImageRGBA(mapW, mapAvailH, m.App.Config.General.DrawGrayline)
	if rgba == nil {
		return ""
	}

	applog.Debug("PSK: buildPSKMapKitty", "mode", m.mapView.PSKMode(), "ready", m.mapView.PSKKittyReady(), "adjW", adjW, "adjH", adjH)
	for i := len(reports) - 1; i >= 0; i-- {
		r := reports[i]
		if r.ReceiverLocator == "" {
			continue
		}
		rlat, rlon := gridToLatLon(r.ReceiverLocator)
		c := pskBandColor(r.Frequency)
		m.mapView.PSKDrawDot(rgba, pixW, pixH, rlat, rlon, c)
	}

	// Own station — green, drawn last so always visible.
	ownCol := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	m.mapView.PSKDrawDot(rgba, pixW, pixH, ownLat, ownLon, ownCol)

	// Only dispatch if data changed — prevents seq advance that would
	// discard the in-flight KittyFrameMsg.
	if mapSig != m.psk.mapSig {
		applog.Debug("PSK: dispatching SetPSKImage", "mapSig", mapSig, "oldSig", m.psk.mapSig)
		_ = m.mapView.SetPSKImage(rgba, adjW, adjH, mapSig)
		m.psk.mapSig = mapSig // block re-dispatch until data changes
		m.psk.mapView = ""    // clear stale cached output
	}

	kittyOut := m.mapView.PSKView()
	applog.Debug("PSK: PSKView len", "len", len(kittyOut), "ready", m.mapView.PSKKittyReady())
	if !m.mapView.PSKKittyReady() {
		return "" // glyph fallback, frame not ready
	}
	if kittyOut == "" {
		return "" // frame not ready, caller falls through to ANSI
	}

	legend := pskLegend()
	result := kittyOut + "\n" + legend
	m.psk.mapView = result
	m.psk.mapSig = mapSig
	return result
}

// pskBandColor returns the RGBA color matching the existing pskMark* styles
// (P.Text, P.Primary, P.Warning, P.Success, P.Accent, P.Info, P.Error).
func pskBandColor(freq float64) color.RGBA {
	s := pskBandStyle(freq)
	// Resolve the Lip Gloss foreground colour to its actual RGB.
	r, g, b, _ := s.GetForeground().RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
}

// pskLegend returns the text legend for the PSK map.
func pskLegend() string {
	return DimStyle.Render(" ") +
		S.MapOwn.Render("\u25c6") + S.MapOwn.Render(" My station") +
		DimStyle.Render("  ") +
		pskMark160.Render("\u25cf") + pskMark160.Render(" 160m") +
		DimStyle.Render("  ") +
		pskMark80.Render("\u25cf") + pskMark80.Render(" 80m") +
		DimStyle.Render("  ") +
		pskMark40.Render("\u25cf") + pskMark40.Render(" 40m") +
		DimStyle.Render("  ") +
		pskMark20.Render("\u25cf") + pskMark20.Render(" 20m") +
		DimStyle.Render("  ") +
		pskMark15.Render("\u25cf") + pskMark15.Render(" 15m") +
		DimStyle.Render("  ") +
		pskMark10.Render("\u25cf") + pskMark10.Render(" 10m") +
		DimStyle.Render("  ") +
		pskMarkOther.Render("\u25cf") + pskMarkOther.Render(" other")
}

func pskCellCoords(lat, lon float64, mapW, mapH int) (int, int) {
	px, py := lonLatToMapPixel(lon, lat, mapSrcW, mapSrcH)
	cx := px * mapW / mapSrcW
	cy := py * mapH / mapSrcH
	if cx < 0 {
		cx = 0
	}
	if cx >= mapW {
		cx = mapW - 1
	}
	if cy < 0 {
		cy = 0
	}
	if cy >= mapH {
		cy = mapH - 1
	}
	return cx, cy
}
