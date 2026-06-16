package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	cacheDir := m.pskCacheDir
	return func() tea.Msg {
		reports, ft, err := psk.FetchReports(call, cacheDir)
		return pskFetchMsg{reports: reports, fetchTime: ft, err: err}
	}
}

// PSK Reporter time filter steps (minutes).
var pskFilterSteps = []int{5, 15, 30, 60, 120, 360}

// Band marker colors per band.
var pskMark160 = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
var pskMark80 = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
var pskMark40 = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
var pskMark20 = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
var pskMark15 = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
var pskMark10 = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
var pskMarkOther = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)

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

func pskTableRow(r psk.Report, callW, gridW, freqW, snrW, modeW int) string {
	call := truncateText(r.ReceiverCallsign, callW)
	loc := truncateText(r.ReceiverLocator, gridW)
	freq := truncateText(fmt.Sprintf("%.3f", r.Frequency/1_000_000), freqW)
	snr := ""
	if r.SNR != 0 {
		snr = truncateText(fmt.Sprintf("%d", r.SNR), snrW)
	}
	mode := truncateText(r.Mode, modeW)
	age := formatAge(r.FlowStartSeconds)
	return fmt.Sprintf("%-*s %-*s %*s %*s %-*s %s",
		callW, call, gridW, loc, freqW, freq, snrW, snr, modeW, mode, age)
}

func formatAge(ts int64) string {
	d := time.Since(time.Unix(ts, 0))
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

func (m *Model) viewPSKReporter() string {
	w := m.width
	if w < 30 {
		w = 80
	}

	if m.pskCacheDir == "" {
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
	sig := fmt.Sprintf("%d|%d|%s|%d|%s|%s|%d|%d|%d|%v",
		w, m.height, call, m.pskFilterMins, m.pskBandFilter, m.pskModeFilter,
		m.pskSelected, int(m.pskLastFetch.Unix()), graySlot, m.pskFetching)
	if m.pskViewKey == sig && m.pskView != "" {
		return m.pskView
	}

	// --- Spot cache: skip SQL when filters unchanged. ---
	spotKey := fmt.Sprintf("%s|%d|%s|%s", call, m.pskFilterMins, m.pskBandFilter, m.pskModeFilter)
	var filtered []psk.Report
	if m.pskSpotKey == spotKey && len(m.pskSpots) > 0 {
		filtered = m.pskSpots
	} else {
		// Query SQLite for the current time window.  Never call the API from View() —
		// fetching is always done asynchronously via pskFetchCmd / pskFetchMsg.
		cutoff := time.Now().UTC().Add(-time.Duration(m.pskFilterMins) * time.Minute).Unix()
		spots, err := store.QueryPSKSpots(m.App.DB, call, cutoff)
		if err != nil {
			return fillBody(DimStyle.Render("PSK Reporter error: "+err.Error()), contentHeight(m.height))
		}

		// Convert to psk.Report for the table/map rendering.
		filtered = make([]psk.Report, len(spots))
		for i, s := range spots {
			filtered[i] = psk.Report{
				ReceiverCallsign: s.ReceiverCall,
				ReceiverLocator:  s.ReceiverLoc,
				Frequency:        s.Frequency,
				SNR:              s.SNR,
				Mode:             s.Mode,
				FlowStartSeconds: s.FlowStart,
			}
		}
		// Sort by time descending.
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].FlowStartSeconds > filtered[j].FlowStartSeconds
		})

		// Apply band filter if set.
		if m.pskBandFilter != "" {
			var bandFiltered []psk.Report
			for _, r := range filtered {
				if freqToBandName(r.Frequency) == m.pskBandFilter {
					bandFiltered = append(bandFiltered, r)
				}
			}
			filtered = bandFiltered
		}
		// Apply mode filter if set.
		if m.pskModeFilter != "" {
			var modeFiltered []psk.Report
			for _, r := range filtered {
				if strings.EqualFold(r.Mode, m.pskModeFilter) {
					modeFiltered = append(modeFiltered, r)
				}
			}
			filtered = modeFiltered
		}

		m.pskSpots = filtered
		m.pskSpotKey = spotKey
	}

	// Clamp cursor only when there are reports.
	if len(filtered) > 0 {
		if m.pskSelected >= len(filtered) {
			m.pskSelected = len(filtered) - 1
		}
		if m.pskSelected < 0 {
			m.pskSelected = 0
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
		if m.pskFetching {
			msg = fmt.Sprintf("Fetching PSK Reporter data for %s\u2026", call)
		} else if !m.pskFetched {
			msg = fmt.Sprintf("Press F5 to fetch PSK Reporter data for %s", call)
		} else {
			msg = fmt.Sprintf("Nobody heard %s in the last %d minutes", call, m.pskFilterMins)
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
		mapAvailH := contentHeight(m.height) - topH - 3
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
			mapBox = centerAndBorderMap(mapBox, contentW, mapW)
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
	m.pskView = result
	m.pskViewKey = sig
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

	header := DimStyle.Width(maxW).MaxWidth(maxW).Inline(true).Render(
		fmt.Sprintf("%-*s %-*s %*s %*s %-*s %s",
			callW, "Call", gridW, "Grid", freqW, "Freq", snrW, "SNR", modeW, "Mode", "Age"))
	var lines []string
	lines = append(lines, header)

	// Paginate in blocks of visibleRows.
	pageSize := visibleRows
	start := (m.pskSelected / pageSize) * pageSize
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
		if i == m.pskSelected {
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
		lines = append(lines, DimStyle.Render(hint))
	} else {
		lines = append(lines, "")
	}

	// Force each line to exactly maxW width so the table fills the box.
	for i, l := range lines {
		lines[i] = lipgloss.NewStyle().Width(maxW).MaxWidth(maxW).Inline(true).Render(l)
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
		val := ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(truncate(value, valW))
		lines = append(lines, indent+lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ", val))
	}

	timeLabel := fmt.Sprintf("%d min", m.pskFilterMins)
	add("Time window", timeLabel)

	bandLabel := "all"
	if m.pskBandFilter != "" {
		bandLabel = m.pskBandFilter
	}
	add("Band filter", bandLabel)

	modeLabel := "all"
	if m.pskModeFilter != "" {
		modeLabel = m.pskModeFilter
	}
	add("Mode filter", modeLabel)

	nextUpdate := ""
	if !m.pskLastFetch.IsZero() {
		nextFetch := m.pskLastFetch.Add(5 * time.Minute)
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
	ownGrid := m.App.Logbook.Station.Grid
	if ownGrid == "" {
		return ""
	}
	ownLat, ownLon := gridToLatLon(ownGrid)

	if m.mapView == nil {
		return ""
	}

	// Get raw map image — no markers, no legend.
	baseMap := m.mapView.BaseImage(mapW, mapAvailH, m.App.Config.General.DrawGrayline)
	if baseMap == "" {
		return ""
	}

	lines := strings.Split(baseMap, "\n")
	mapH := len(lines)

	// Draw band-colored markers for each receiver — oldest first, newest last
	// so the newest spot's color wins when multiple spots share a cell.
	for i := len(reports) - 1; i >= 0; i-- {
		r := reports[i]
		if r.ReceiverLocator == "" {
			continue
		}
		rlat, rlon := gridToLatLon(r.ReceiverLocator)
		cx, cy := pskCellCoords(rlat, rlon, mapW, mapH)
		if cy >= 0 && cy < mapH && cx >= 0 && cx < mapW {
			mark := pskBandStyle(r.Frequency).Render("\u25cf")
			lines[cy] = replaceANSICell(lines[cy], cx, mark)
		}
	}

	// Draw own station LAST — so it's always visible on top of any receiver.
	ownX, ownY := pskCellCoords(ownLat, ownLon, mapW, mapH)
	if ownY >= 0 && ownY < mapH && ownX >= 0 && ownX < mapW {
		lines[ownY] = replaceANSICell(lines[ownY], ownX, S.MapOwn.Render("\u25c6"))
	}

	// One-line legend: own marker + band dots.
	legend := DimStyle.Render(" ") +
		S.MapOwn.Render("\u25c6") + DimStyle.Render(" My station") +
		DimStyle.Render("  ") +
		pskMark160.Render("\u25cf") + DimStyle.Render(" 160m") +
		DimStyle.Render("  ") +
		pskMark80.Render("\u25cf") + DimStyle.Render(" 80m") +
		DimStyle.Render("  ") +
		pskMark40.Render("\u25cf") + DimStyle.Render(" 40m") +
		DimStyle.Render("  ") +
		pskMark20.Render("\u25cf") + DimStyle.Render(" 20m") +
		DimStyle.Render("  ") +
		pskMark15.Render("\u25cf") + DimStyle.Render(" 15m") +
		DimStyle.Render("  ") +
		pskMark10.Render("\u25cf") + DimStyle.Render(" 10m") +
		DimStyle.Render("  ") +
		pskMarkOther.Render("\u25cf") + DimStyle.Render(" other")
	lines = append(lines, legend)

	return strings.Join(lines, "\n")
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
