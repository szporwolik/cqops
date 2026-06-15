package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/store"
)

const partnerMapMaxW = 128 // also used as max page width for QSO form consistency

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
	d := m.partnerData
	if d == nil {
		d = m.formPartnerData()
		if d == nil || d.Callsign == "" {
			m.partnerViewCacheSig = ""
			return ""
		}
	}

	w := m.width
	if w < 30 {
		w = 80
	}

	// Build cache signature — includes all inputs that affect output.
	var sigB strings.Builder
	fmt.Fprintf(&sigB, "%d|%d|", m.width, m.height)
	if m.partnerData != nil {
		fmt.Fprintf(&sigB, "pd:%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s|",
			m.partnerData.Callsign, m.partnerData.Name, m.partnerData.Grid,
			m.partnerData.QTH, m.partnerData.Country, m.partnerData.State,
			m.partnerData.County, m.partnerData.Zip, m.partnerData.Class,
			m.partnerData.Email, m.partnerData.URL, m.partnerData.Lat,
			m.partnerData.Lon, m.partnerData.DXCC, m.partnerData.CQZone, m.partnerData.ITUZone)
	} else {
		sigB.WriteString("pd:nil|")
	}
	sigB.WriteString(m.cachedLogStatsSig)
	sigB.WriteByte('|')
	if m.wlPrivateData != nil {
		fmt.Fprintf(&sigB, "wl:wk=%v,dxcc=%v,band=%v,bm=%v,cband=%v,cbm=%v,lotw=%v|",
			m.wlPrivateData.Worked(), m.wlPrivateData.DXCCConfirmed(),
			m.wlPrivateData.WorkedBand(), m.wlPrivateData.WorkedBandMode(),
			m.wlPrivateData.ConfirmedBand(), m.wlPrivateData.ConfirmedBandMode(),
			m.wlPrivateData.LoTW())
	} else {
		sigB.WriteString("wl:nil|")
	}
	fmt.Fprintf(&sigB, "wldone=%v|wlband=%s|wlmode=%s|qrz=%v|wlcfg=%v|rmap=%v",
		m.wlLookupDone, m.wlLastBand, m.wlLastMode,
		m.App.Config.QRZ.Enabled,
		m.App.Logbook.Wavelog != nil && m.App.Logbook.Wavelog.Enabled,
		m.App.Config.General.RenderMap)

	sig := sigB.String()
	if m.partnerViewCacheSig == sig && m.partnerViewCache != "" {
		return m.partnerViewCache
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

	var cbW, lbW, wlW int
	if wlEnabled {
		cbW = totalW * 40 / 100
		lbW = totalW * 28 / 100
		wlW = totalW - cbW - lbW
	} else {
		cbW = totalW * 50 / 100
		lbW = totalW - cbW
		wlW = 0
	}
	if cbW < 20 {
		cbW = totalW
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
		// Re-render callbook/logbook boxes with updated maxInner.
		cbBox = m.renderPartnerBox("Callbook"+m.qrzSuffix(), cbContent, cbW, maxInner)
		lbBox = m.renderPartnerBox("Logbook", lbContent, lbW, maxInner)
		wlBox := m.renderPartnerBox("Wavelog", wlContent, wlW, maxInner)
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, cbBox, lbBox, wlBox)
	} else {
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, cbBox, lbBox)
	}

	var block string
	if m.App.Config.General.RenderMap {
		mapW := totalW
		topH := lipgloss.Height(topRow)
		// Reserve space for legend (1) + border top/bottom (2).
		mapAvailH := contentHeight(m.height) - topH - 3
		if mapAvailH < 3 {
			mapAvailH = 3
		}
		// Content is 4 cells narrower than the border: 2 for border chars + 2 for Padding(0,1).
		contentW := mapW - 4
		if contentW < 20 {
			contentW = mapW
		}
		mapBox := m.getOrBuildMap(d, contentW, mapAvailH)
		if mapBox != "" {
			// Force every line to exactly contentW columns — centered if narrower.
			lines := strings.Split(mapBox, "\n")
			for i, l := range lines {
				lw := lipgloss.Width(l)
				if lw > contentW {
					lines[i] = truncateText(l, contentW)
				} else if lw < contentW {
					left := (contentW - lw) / 2
					right := contentW - lw - left
					lines[i] = strings.Repeat(" ", left) + l + strings.Repeat(" ", right)
				}
			}
			mapBox = strings.Join(lines, "\n")
			mapBox = drawBorderedBox(mapBox, mapW)
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
	result := fillBody(block, contentHeight(m.height))
	m.partnerViewCacheSig = sig
	m.partnerViewCache = result
	return result
}

// --- Box helpers ---

func (m *Model) qrzSuffix() string {
	if m.partnerData != nil && m.App.Config.QRZ.Enabled {
		return " (QRZ)"
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
		inner += strings.Repeat("\n", maxInner-curH)
	}
	return drawBorderedBox(inner, boxW)
}

func infoRow(label, value string, maxW int) string {
	lbl := S.FormLabel.Align(lipgloss.Right).Render(label)
	valW := maxW - 12
	if valW < 3 {
		valW = 3
	}
	val := ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(truncate(value, valW))
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
		add("Callsign", S.Info.Render(d.Callsign))
	}
	add("Name", d.Name)
	add("Grid", d.Grid)
	add("QTH", d.QTH)
	add("Country", d.Country)
	add("State", d.State)
	add("County", d.County)
	add("Zip", d.Zip)
	add("Class", d.Class)
	add("Email", d.Email)
	add("URL", d.URL)
	if d.Lat != "" || d.Lon != "" {
		add("Coordinates", strings.TrimSpace(d.Lat+" "+d.Lon))
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
	if m.cachedLogStatsSig != sig && m.App.DB != nil {
		stats, err := store.GetLogbookStats(m.App.DB, call, band, mode)
		if err == nil {
			m.cachedLogStats = stats
			m.cachedLogStatsSig = sig
		}
	}
	s := m.cachedLogStats
	wl := m.wlPrivateData

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
	rows = append(rows, row{"Last QSO", ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(truncate(last, valW))})

	return formatRowPairs(rows, S.FormLabel)
}

// --- WL Info ---
// Y = green (yes it IS new), N = dim (already worked), ? = dim (unknown).
// LoTW: N = red (not a member), last on list.

func (m *Model) renderWLInfo(maxW int) string {
	d := m.wlPrivateData
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || wl.URL == "" || wl.APIKey == "" {
		return DimStyle.Width(maxW).Align(lipgloss.Center).Render("Wavelog not configured")
	}
	if d == nil {
		msg := "WL lookup pending\u2026"
		if m.wlLookupDone {
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

	hasBand := m.wlLastBand != ""
	hasMode := m.wlLastMode != ""

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
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
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

	ownGrid := m.App.Logbook.Station.Grid
	partnerGrid := d.Grid

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
	if d.Lat != "" {
		pl = parseCoord(d.Lat)
		plon = parseCoord(d.Lon)
	}

	// Use embedded image map renderer.
	if m.mapView != nil {
		return m.mapView.View(ownLat, ownLon, pl, plon, mapW, mapAvailH)
	}
	// Fallback: ASCII map.
	return renderWorldMap(ownLat, ownLon, pl, plon, mapW, mapAvailH)
}

func (m *Model) invalidatePartnerMapCache() {
	m.partnerViewCache = ""
	m.partnerViewCacheSig = ""
}
