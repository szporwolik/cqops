package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

const partnerMapMaxW = 128 // also used as max page width for QSO form consistency

// continentName maps 2-letter CTY.DAT continent codes to full names.
func continentName(code string) string {
	switch strings.ToUpper(code) {
	case "EU":
		return "Europe"
	case "NA":
		return "North America"
	case "SA":
		return "South America"
	case "AF":
		return "Africa"
	case "AS":
		return "Asia"
	case "OC":
		return "Oceania"
	case "AN", "AA":
		return "Antarctica"
	default:
		return code
	}
}

// row is a label+value pair used by all three partner info boxes.
type row struct{ label, value string }

// formatRowPairs joins rows into left-aligned label+value lines.
func formatRowPairs(rows []row, labelStyle lipgloss.Style) string {
	var lines []string
	for _, r := range rows {
		lbl := labelStyle.Align(lipgloss.Right).Render(r.label)
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ", r.value))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderFlagStatus returns styled "Y" (yes/new, green), "N" (no/worked, dim),
// or "?" (unknown, dim) based on the isNew and known flags.
func renderFlagStatus(isNew, known bool, newStyle, oldStyle lipgloss.Style) string {
	if !known {
		return DimStyle.Render("?")
	}
	if isNew {
		return newStyle.Render("Y")
	}
	return oldStyle.Render("N")
}

func (m *Model) viewPartner() string {
	d := m.lookup.partnerData
	if d == nil {
		d = m.formPartnerData()
		if d == nil || d.Callsign == "" {
			m.rc.partnerViewSig = ""
			return ""
		}
	}

	w := m.width
	if w < 30 {
		w = 80
	}

	// Build cache signature — uses pointer identity for QRZ/WL data structs
	// instead of enumerating all fields, cutting the pre-frame string work.
	// The pointers are replaced on every new lookup, so identity == freshness.
	var sigB strings.Builder
	fmt.Fprintf(&sigB, "%d|%d|", m.width, m.height)
	if m.lookup.partnerData != nil {
		fmt.Fprintf(&sigB, "pd=%p|", m.lookup.partnerData)
	} else {
		sigB.WriteString("pd:nil|")
	}
	sigB.WriteString(m.rc.logStatsSig)
	sigB.WriteByte('|')
	if m.lookup.wlPrivateData != nil {
		fmt.Fprintf(&sigB, "wl=%p,%v,%v,%v,%v|",
			m.lookup.wlPrivateData,
			m.lookup.wlPrivateData.WorkedBand(), m.lookup.wlPrivateData.WorkedBandMode(),
			m.lookup.wlPrivateData.ConfirmedBand(), m.lookup.wlPrivateData.ConfirmedBandMode())
	} else {
		sigB.WriteString("wl:nil|")
	}
	fmt.Fprintf(&sigB, "wldone=%v|wlband=%s|wlmode=%s|qrz=%v|wlcfg=%v|rmap=%v|gray=%v",
		m.lookup.wlLookupDone, m.lookup.wlLastBand, m.lookup.wlLastMode,
		m.App.Config.Integrations.QRZ.Enabled,
		m.App.Logbook.Wavelog != nil && m.App.Logbook.Wavelog.Enabled,
		m.App.Config.General.RenderMap,
		m.App.Config.General.DrawGrayline)
	fmt.Fprintf(&sigB, "|fmgrid=%s|gridsrc=%s", m.fields[fieldGrid].Value(), m.gridSource)
	// Inline photo — only bust cache on significant content changes,
	// not on every progressive-render frame (avoids 100% CPU on slow PCs).
	if d != nil && d.ImageURL != "" {
		picContent := m.photo.partnerPicViewer.View().Content
		// Hash on coarse length buckets (32-byte granularity) so the
		// cache only invalidates a few times during photo download.
		bucket := len(picContent) >> 5
		fmt.Fprintf(&sigB, "|pic=%s,%d", d.ImageURL, bucket)
	}

	sig := sigB.String()
	if m.rc.partnerViewSig == sig && m.rc.partnerView != "" {
		return m.rc.partnerView
	}

	totalW := w - 2
	if totalW > partnerMapMaxW {
		totalW = partnerMapMaxW
	}
	if totalW < 60 {
		totalW = w - 2
	}

	// Detect Wavelog availability — when not configured, hide the box and split 50:50.
	wlEnabled := m.App.Logbook.Wavelog != nil && m.App.Logbook.Wavelog.Enabled &&
		m.App.Logbook.Wavelog.URL != "" && m.App.Logbook.Wavelog.APIKey != ""

	// Inline partner photo — right-side column on wide screens (≥180 cols).
	showPhoto := m.width >= 180 && m.App.Config.General.PictureAtQRZPane &&
		d != nil && d.ImageURL != ""
	if showPhoto && d.ImageURL != m.photo.partnerPicURL {
		m.photo.partnerPicURL = d.ImageURL
		m.photo.partnerPicNeedLoad = true
	}

	// Left column stays at the standard max-width cap. Photo fills remaining space.
	leftW := totalW
	photoW := 0
	if showPhoto {
		photoW = (w - 2) - totalW
		if photoW < 25 {
			photoW = 25
		}
	}

	var cbW, lbW, wlW int
	if wlEnabled {
		cbW = leftW * 40 / 100
		lbW = leftW * 28 / 100
		wlW = leftW - cbW - lbW
	} else {
		cbW = leftW * 50 / 100
		lbW = leftW - cbW
		wlW = 0
	}
	if cbW < 20 {
		cbW = leftW
	}
	if lbW < 15 {
		lbW = 20
	}
	if wlW > 0 && wlW < 12 {
		wlW = 20
	}

	cbContent := m.renderCallbookRows(d, cbW-4)
	lbContent := m.renderLogbookRows(d, lbW-4)

	// Compute max inner height (header + content lines), then pad all.
	cbInner := lipgloss.Height(cbContent) + 1
	lbInner := lipgloss.Height(lbContent) + 1
	maxInner := cbInner
	if lbInner > maxInner {
		maxInner = lbInner
	}

	cbBox := m.renderPartnerBox("Callbook"+m.qrzSuffix(), cbContent, cbW, maxInner)
	lbBox := m.renderPartnerBox("Logbook", lbContent, lbW, maxInner)

	var topRow string
	if wlEnabled {
		wlContent := m.renderWLInfo(wlW - 4)
		wlInner := lipgloss.Height(wlContent) + 1
		if wlInner > maxInner {
			maxInner = wlInner
		}
		cbBox = m.renderPartnerBox("Callbook"+m.qrzSuffix(), cbContent, cbW, maxInner)
		lbBox = m.renderPartnerBox("Logbook", lbContent, lbW, maxInner)
		wlBox := m.renderPartnerBox("Wavelog", wlContent, wlW, maxInner)
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, cbBox, lbBox, wlBox)
	} else {
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, cbBox, lbBox)
	}

	// Build left column: topRow + map (if enabled).
	var leftCol string
	if m.App.Config.General.RenderMap {
		topH := lipgloss.Height(topRow)
		mapAvailH := contentHeight(m.height) - topH - 3
		if mapAvailH < 3 {
			mapAvailH = 3
		}
		contentW := leftW - 4
		if contentW < 20 {
			contentW = leftW
		}
		mapBox := m.getOrBuildMap(d, contentW, mapAvailH)
		if mapBox != "" {
			mapBox = centerAndBorderMap(mapBox, contentW, leftW)
			leftCol = lipgloss.JoinVertical(lipgloss.Left, topRow, mapBox)
		} else {
			leftCol = topRow
		}
	} else {
		leftCol = topRow
	}

	// Right column: photo (if enabled). Force height to match leftCol exactly.
	var block string
	if showPhoto {
		leftH := lipgloss.Height(leftCol)
		picRaw := m.photo.partnerPicViewer.View().Content
		if picRaw == "" {
			if m.photo.partnerPicViewer.Err() != nil {
				picRaw = DimStyle.Render("Photo unavailable")
			} else {
				picRaw = DimStyle.Render("Loading\u2026")
			}
		}
		picContentH := leftH - 3 // header + border
		if picContentH < 1 {
			picContentH = 1
		}
		picLines := strings.Split(picRaw, "\n")
		if len(picLines) > picContentH {
			picLines = picLines[:picContentH]
		}
		for len(picLines) < picContentH {
			picLines = append(picLines, "")
		}
		header := S.Label.Width(photoW - 4).MaxWidth(photoW - 4).Inline(true).Render("Photo")
		inner := lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(picLines, "\n"))
		picBox := drawBorderedBox(inner, photoW+1)
		// Pad the shorter column with newlines instead of using
		// lipgloss.Place, which wraps content in ANSI escapes
		// that can shift Kitty virtual image placement.
		picH := lipgloss.Height(picBox)
		if leftH > picH {
			picBox += strings.Repeat("\n", leftH-picH)
		} else if picH > leftH {
			leftCol += strings.Repeat("\n", picH-leftH)
			leftH = picH
		}
		m.photo.partnerPicW = photoW - 3
		m.photo.partnerPicH = picContentH
		if m.photo.partnerPicW < 25 {
			m.photo.partnerPicW = 25
		}
		if m.photo.partnerPicH < 4 {
			m.photo.partnerPicH = 4
		}
		// Trigger resize if terminal dimensions changed (handled in handlePartnerUpdate).
		block = lipgloss.JoinHorizontal(lipgloss.Top, leftCol, picBox)
	} else {
		block = leftCol
	}

	if w > totalW+2 {
		block = PartnerBlock.Width(w).Render(block)
	}
	result := fillBody(block, contentHeight(m.height))
	m.rc.partnerViewSig = sig
	m.rc.partnerView = result
	return result
}

// --- Box helpers ---

func (m *Model) qrzSuffix() string {
	if m.lookup.partnerData != nil && m.App.Config.Integrations.QRZ.Enabled {
		return " (QRZ.com)"
	}
	return ""
}

// renderPartnerBox wraps header+content in a bordered box. Content is padded
// to maxInner lines so all boxes in a row have equal height.
func (m *Model) renderPartnerBox(title, content string, boxW, maxInner int) string {
	header := S.Label.Width(boxW - 4).MaxWidth(boxW - 4).Inline(true).Render(title)
	inner := lipgloss.JoinVertical(lipgloss.Left, header, content)
	curH := lipgloss.Height(inner)
	if curH < maxInner {
		var sb strings.Builder
		sb.WriteString(inner)
		for i := curH; i < maxInner; i++ {
			sb.WriteByte('\n')
		}
		inner = sb.String()
	}
	return drawBorderedBox(inner, boxW)
}

func infoRow(label, value string, maxW int) string {
	lbl := S.FormLabel.Align(lipgloss.Right).Render(label)
	valW := maxW - 12
	if valW < 3 {
		valW = 3
	}
	// Let Lip Gloss handle clipping via MaxWidth+Inline — never manually
	// truncate a value that may contain OSC-8 hyperlink ANSI sequences.
	val := ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(value)
	return lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ", val)
}

// --- Callbook rows ---

func (m *Model) renderCallbookRows(d *qrz.CallData, maxW int) string {
	var rows []row
	add := func(label, value string) {
		if value != "" {
			rows = append(rows, row{label, value})
		}
	}
	if d.Callsign != "" {
		link := osc8Link("https://www.qrz.com/db/"+d.Callsign, S.Info.Render(d.Callsign))
		add("Callsign", link)
	}
	// Continent from DXCC prefix lookup — cached per callsign to avoid
	// prefix-tree access on every partner-view frame.
	if d.Callsign != "" {
		if d.Callsign != m.rc.dxccContCall {
			m.rc.dxccContCall = d.Callsign
			m.rc.dxccContValue = ""
			if p := m.dxccLookup(d.Callsign); p != nil && p.Continent != "" {
				m.rc.dxccContValue = p.Continent
			}
		}
		if m.rc.dxccContValue != "" {
			add("Continent", continentName(m.rc.dxccContValue))
		}
	}
	add("Name", d.Name)
	// Show the QSO form grid (which may differ from QRZ grid due to REF autofill)
	// with its source, or fall back to QRZ grid.
	formGrid := strings.TrimSpace(m.fields[fieldGrid].Value())
	if formGrid != "" && m.gridSource != "" && m.gridSource != gridSourceQRZ {
		add("Grid", osc8Link("http://www.levinecentral.com/ham/grid_square.php?Grid="+formGrid, formGrid)+"  "+DimStyle.Render("("+string(m.gridSource)+")"))
	} else if d.Grid != "" {
		add("Grid", osc8Link("http://www.levinecentral.com/ham/grid_square.php?Grid="+d.Grid, d.Grid))
	} else if formGrid != "" {
		add("Grid", formGrid)
	} else {
		add("Grid", "")
	}
	add("QTH", d.QTH)
	add("Country", d.Country)
	add("State", d.State)
	add("County", d.County)
	add("Zip", d.Zip)
	add("Class", d.Class)
	if d.Email != "" {
		add("Email", osc8Link("mailto:"+d.Email, d.Email))
	} else {
		add("Email", "")
	}
	add("URL", d.URL)
	if d.Lat != "" || d.Lon != "" {
		coordText := strings.TrimSpace(d.Lat + " " + d.Lon)
		coordURL := fmt.Sprintf("https://geohack.toolforge.org/geohack.php?params=%s_N_%s_E_type:town", d.Lat, d.Lon)
		add("Coordinates", osc8Link(coordURL, coordText))
	}
	add("DXCC", d.DXCC)
	add("CQ Zone", d.CQZone)
	add("ITU Zone", d.ITUZone)
	if len(rows) == 0 {
		return ""
	}
	var lines []string
	for _, r := range rows {
		lines = append(lines, infoRow(r.label, r.value, maxW))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// --- Logbook rows ---

func (m *Model) renderLogbookRows(d *qrz.CallData, maxW int) string {
	call := d.Callsign
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := strings.TrimSpace(m.fields[fieldMode].Value())
	sig := call + "|" + band + "|" + mode
	if m.rc.logStatsSig != sig && m.App.DB != nil {
		// Cache miss — dispatch async fetch and use previous data this frame.
		// The fetch will complete before the next View() call.
		m.rc.logStatsNeedFetch = true
		m.rc.logStatsFetchCall = call
		m.rc.logStatsFetchBand = band
		m.rc.logStatsFetchMode = mode
	}
	s := m.rc.logStats
	wl := m.lookup.wlPrivateData

	newStyle := S.Success // green — yes, it IS new
	oldStyle := DimStyle  // dim — no, already worked

	// Compute value column width. Label width 11 + space 1 = 12.
	valW := maxW - 12
	if valW < 3 {
		valW = 3
	}

	// WL-first helper: returns (isNew, known).
	// If WL has data, it wins. Otherwise falls back to local.
	wlFirst := func(wlVal, localVal bool) (bool, bool) {
		if wl != nil {
			return !wlVal, true
		}
		return !localVal, true
	}
	// WL-only helper: only WL can answer (DXCC fields).
	wlOnly := func(wlVal bool) (bool, bool) {
		if wl != nil {
			return !wlVal, true
		}
		return false, false
	}

	// Render Y/N/? with appropriate style.
	flag := func(isNew, known bool) string {
		return renderFlagStatus(isNew, known, newStyle, oldStyle)
	}

	var rows []row

	// New call
	isNew, _ := wlFirst(wl != nil && wl.Worked(), s.CallWorked)
	rows = append(rows, row{"New call", flag(isNew, true)})

	// New on band
	if band != "" {
		isNew, _ := wlFirst(wl != nil && wl.WorkedBand(), s.CallOnBand)
		rows = append(rows, row{"New on band", flag(isNew, true)})
	} else {
		rows = append(rows, row{"New on band", DimStyle.Render("?")})
	}

	// New on mode
	if mode != "" {
		isNew, _ := wlFirst(wl != nil && wl.WorkedBandMode(), s.CallOnMode)
		rows = append(rows, row{"New on mode", flag(isNew, true)})
	} else {
		rows = append(rows, row{"New on mode", DimStyle.Render("?")})
	}

	// New DXCC (WL only — local doesn't track DXCC)
	isNew, known := wlOnly(wl != nil && wl.DXCCConfirmed())
	rows = append(rows, row{"New DXCC", flag(isNew, known)})

	// New DXCC on band
	if band != "" {
		isNew, known = wlOnly(wl != nil && wl.ConfirmedBand())
		rows = append(rows, row{"DXCC band", flag(isNew, known)})
	} else {
		rows = append(rows, row{"DXCC band", DimStyle.Render("?")})
	}

	// New DXCC on mode
	if mode != "" {
		isNew, known = wlOnly(wl != nil && wl.ConfirmedBandMode())
		rows = append(rows, row{"DXCC mode", flag(isNew, known)})
	} else {
		rows = append(rows, row{"DXCC mode", DimStyle.Render("?")})
	}

	// QSO count
	cnt := "none"
	if s.QSOCount > 0 {
		cnt = fmt.Sprintf("%d", s.QSOCount)
	}
	rows = append(rows, row{"QSO count", ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(cnt)})

	// Last QSO — clipped, never wrapped.
	last := "none"
	if s.LastQSODate != "" {
		last = s.LastQSODate
	}
	rows = append(rows, row{"Last QSO", ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(truncateText(last, valW))})

	return formatRowPairs(rows, S.FormLabel)
}

// fetchLogbookStatsCmd returns a tea.Cmd that runs GetLogbookStats
// asynchronously, avoiding DB I/O during View().
func (m *Model) fetchLogbookStatsCmd(call, band, mode string) tea.Cmd {
	db := m.App.DB
	return func() tea.Msg {
		stats, err := store.GetLogbookStats(db, call, band, mode)
		if err != nil {
			return logbookStatsMsg{}
		}
		return logbookStatsMsg{stats: stats, sig: call + "|" + band + "|" + mode}
	}
}

// handleLogbookStats stores the async result for use by the next View().
func (m *Model) handleLogbookStats(msg logbookStatsMsg) {
	if msg.sig == "" {
		return
	}
	m.rc.logStats = msg.stats
	m.rc.logStatsSig = msg.sig
}

// --- WL Info ---
// Y = green (yes it IS new), N = dim (already worked), ? = dim (unknown).
// LoTW: N = red (not a member), last on list.

func (m *Model) renderWLInfo(maxW int) string {
	d := m.lookup.wlPrivateData
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || wl.URL == "" || wl.APIKey == "" {
		return DimStyle.Width(maxW).Align(lipgloss.Center).Render("Wavelog not configured")
	}
	if d == nil {
		msg := "WL lookup pending\u2026"
		if m.Offline || !m.inetOnline {
			msg = "Offline mode"
		} else if m.lookup.wlLookupDone {
			msg = "No WL data"
		}
		return DimStyle.Width(maxW).Align(lipgloss.Center).Render(msg)
	}

	newStyle := S.Success // green — yes, it IS new
	oldStyle := DimStyle  // dim — no, already there
	badStyle := S.Error   // red — LoTW: not a member

	flag := func(isNew, known bool) string {
		return renderFlagStatus(isNew, known, newStyle, oldStyle)
	}

	var rows []row

	hasBand := m.lookup.wlLastBand != ""
	hasMode := m.lookup.wlLastMode != ""

	// New call
	rows = append(rows, row{"New call", flag(!d.Worked(), true)})

	// New on band
	isNew, known := false, hasBand
	if hasBand {
		isNew = !d.WorkedBand()
	}
	rows = append(rows, row{"New on band", flag(isNew, known)})

	// New on mode
	isNew, known = false, hasBand && hasMode
	if hasBand && hasMode {
		isNew = !d.WorkedBandMode()
	}
	rows = append(rows, row{"New on mode", flag(isNew, known)})

	// New DXCC
	rows = append(rows, row{"New DXCC", flag(!d.DXCCConfirmed(), true)})

	// New DXCC on band
	isNew, known = false, hasBand
	if hasBand {
		isNew = !d.ConfirmedBand()
	}
	rows = append(rows, row{"New DXCC on band", flag(isNew, known)})

	// New DXCC on mode
	isNew, known = false, hasBand && hasMode
	if hasBand && hasMode {
		isNew = !d.ConfirmedBandMode()
	}
	rows = append(rows, row{"New DXCC on mode", flag(isNew, known)})

	// LoTW member — last, separate logic: N = red (not a member).
	rows = append(rows, row{"LoTW member", renderLoTW(d.LoTW(), oldStyle, badStyle)})

	return formatRowPairs(rows, S.FormLabelWide)
}

// renderLoTW returns styled "Y" (dim, member) or "N" (red, not a member).
func renderLoTW(isMember bool, dimStyle, badStyle lipgloss.Style) string {
	if isMember {
		return dimStyle.Render("Y")
	}
	return badStyle.Render("N")
}

// --- Form partner data ---

func (m *Model) formPartnerData() *qrz.CallData {
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	if call == "" {
		return nil
	}
	return &qrz.CallData{
		Callsign: call,
		Name:     strings.TrimSpace(m.fields[fieldName].Value()),
		Grid:     strings.TrimSpace(m.fields[fieldGrid].Value()),
		QTH:      strings.TrimSpace(m.fields[fieldQTH].Value()),
		Country:  strings.TrimSpace(m.fields[fieldCountry].Value()),
	}
}

// --- Map cache ---

func (m *Model) getOrBuildMap(d *qrz.CallData, mapW, mapAvailH int) string {
	// RenderMap config toggle — if off, don't show map.
	if !m.App.Config.General.RenderMap {
		return ""
	}

	ownGrid := m.effectiveGrid()
	// Use QSO form grid if set (may differ from QRZ due to REF autofill).
	partnerGrid := strings.TrimSpace(m.fields[fieldGrid].Value())
	if partnerGrid == "" {
		partnerGrid = d.Grid
	}

	// No location data — show hint instead of map.
	if ownGrid == "" {
		return DimStyle.Render("Set your grid in station config to enable the map")
	}
	if partnerGrid == "" && d.Lat == "" {
		return DimStyle.Render("No partner location — enter a grid or use QRZ lookup")
	}

	ownLat, ownLon := gridToLatLon(ownGrid)
	pl, plon := 0.0, 0.0
	if partnerGrid != "" {
		pl, plon = gridToLatLon(partnerGrid)
	}
	// Only fall back to QRZ lat/lon when no form grid is set (i.e. no REF/manual override).
	// This ensures field activation coordinates take precedence over home QTH.
	if partnerGrid == "" && d.Lat != "" {
		pl = parseCoord(d.Lat)
		plon = parseCoord(d.Lon)
	}

	// Use embedded image map renderer.
	if m.mapView != nil {
		return m.mapView.View(ownLat, ownLon, pl, plon, mapW, mapAvailH, m.App.Config.General.DrawGrayline)
	}
	return ""
}

func (m *Model) invalidatePartnerMapCache() {
	m.rc.partnerView = ""
	m.rc.partnerViewSig = ""
	if m.mapView != nil {
		m.mapView.Invalidate()
	}
}
