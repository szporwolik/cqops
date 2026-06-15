package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/store"
)

const partnerColMaxW = 35
const partnerMapMaxW = 100

func (m *Model) viewPartner() string {
	d := m.partnerData
	if d == nil {
		d = m.formPartnerData()
		if d == nil || d.Callsign == "" {
			return ""
		}
	}

	w := m.width
	if w < 30 {
		w = 80
	}

	totalW := w - 2
	if totalW > partnerMapMaxW {
		totalW = partnerMapMaxW
	}
	if totalW < 60 {
		totalW = w - 2
	}

	cbW := totalW * 40 / 100
	lbW := totalW * 30 / 100
	wlW := totalW - cbW - lbW
	if cbW < 20 {
		cbW = totalW
	}
	if lbW < 15 {
		lbW = 20
	}
	if wlW < 12 {
		wlW = 20
	}

	cbContent := m.renderCallbookRows(d, cbW-4)
	lbContent := m.renderLogbookRows(d, lbW-4)
	wlContent := m.renderWLInfo(wlW - 4)

	// Compute max inner height (header + content lines), then pad all.
	cbInner := lipgloss.Height(cbContent) + 1
	lbInner := lipgloss.Height(lbContent) + 1
	wlInner := lipgloss.Height(wlContent) + 1
	maxInner := cbInner
	if lbInner > maxInner {
		maxInner = lbInner
	}
	if wlInner > maxInner {
		maxInner = wlInner
	}

	cbBox := m.renderPartnerBox("Callbook information"+m.qrzSuffix(), cbContent, cbW, maxInner)
	lbBox := m.renderPartnerBox("Logbook", lbContent, lbW, maxInner)
	wlBox := m.renderPartnerBox("Wavelog", wlContent, wlW, maxInner)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cbBox, lbBox, wlBox)

	mapW := totalW
	topH := lipgloss.Height(topRow)
	mapAvailH := contentHeight(m.height) - topH
	if mapAvailH < 3 {
		mapAvailH = 3
	}
	mapBox := drawBorderedBox(m.getOrBuildMap(d, mapW, mapAvailH), mapW)

	block := lipgloss.JoinVertical(lipgloss.Left, topRow, mapBox)
	if w > totalW+2 {
		block = lipgloss.NewStyle().Width(w).Align(lipgloss.Left).Render(block)
	}
	return fillBody(block, contentHeight(m.height))
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
	header := S.Label.Width(boxW - 4).MaxWidth(boxW - 4).Render(title)
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
	type row struct{ label, value string }
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
// Wavelog-first precedence: if WL has data it wins; otherwise fall back to local.
// Y = green (yes it IS new), N = dim (already worked), ? = dim (unknown).

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
	renderFlag := func(isNew, known bool) string {
		if !known {
			return DimStyle.Render("?")
		}
		if isNew {
			return newStyle.Render("Y")
		}
		return oldStyle.Render("N")
	}

	type row struct{ label, value string }
	var rows []row

	// New call
	isNew, _ := wlFirst(wl != nil && wl.Worked(), s.CallWorked)
	rows = append(rows, row{"New call", renderFlag(isNew, true)})

	// New on band
	if band != "" {
		isNew, _ := wlFirst(wl != nil && wl.WorkedBand(), s.CallOnBand)
		rows = append(rows, row{"New on band", renderFlag(isNew, true)})
	} else {
		rows = append(rows, row{"New on band", DimStyle.Render("?")})
	}

	// New on mode
	if mode != "" {
		isNew, _ := wlFirst(wl != nil && wl.WorkedBandMode(), s.CallOnMode)
		rows = append(rows, row{"New on mode", renderFlag(isNew, true)})
	} else {
		rows = append(rows, row{"New on mode", DimStyle.Render("?")})
	}

	// New DXCC (WL only — local doesn't track DXCC)
	isNew, known := wlOnly(wl != nil && wl.DXCCConfirmed())
	rows = append(rows, row{"New DXCC", renderFlag(isNew, known)})

	// New DXCC on band
	if band != "" {
		isNew, known = wlOnly(wl != nil && wl.ConfirmedBand())
		rows = append(rows, row{"New DXCC on band", renderFlag(isNew, known)})
	} else {
		rows = append(rows, row{"New DXCC on band", DimStyle.Render("?")})
	}

	// New DXCC on mode
	if mode != "" {
		isNew, known = wlOnly(wl != nil && wl.ConfirmedBandMode())
		rows = append(rows, row{"New DXCC on mode", renderFlag(isNew, known)})
	} else {
		rows = append(rows, row{"New DXCC on mode", DimStyle.Render("?")})
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

	var lines []string
	for _, r := range rows {
		lbl := S.FormLabel.Align(lipgloss.Right).Render(r.label)
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ", r.value))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// --- WL Info ---
// Airbus approach: N = green (new), Y = dim (normal). LoTW: N = red (problem), last on list.

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

	type row struct{ label, value string }
	var rows []row

	hasBand := m.wlLastBand != ""
	hasMode := m.wlLastMode != ""

	// New call: Y = yes new (green), N = not new (dim)
	v := "Y"
	sty := newStyle
	if d.Worked() {
		v = "N"
		sty = oldStyle
	}
	rows = append(rows, row{"New call", sty.Render(v)})

	// New on band
	v = "?"
	sty = DimStyle
	if hasBand {
		v = "Y"
		sty = newStyle
		if d.WorkedBand() {
			v = "N"
			sty = oldStyle
		}
	}
	rows = append(rows, row{"New on band", sty.Render(v)})

	// New on mode
	v = "?"
	sty = DimStyle
	if hasBand && hasMode {
		v = "Y"
		sty = newStyle
		if d.WorkedBandMode() {
			v = "N"
			sty = oldStyle
		}
	}
	rows = append(rows, row{"New on mode", sty.Render(v)})

	// New DXCC
	v = "Y"
	sty = newStyle
	if d.DXCCConfirmed() {
		v = "N"
		sty = oldStyle
	}
	rows = append(rows, row{"New DXCC", sty.Render(v)})

	// New DXCC on band
	v = "?"
	sty = DimStyle
	if hasBand {
		v = "Y"
		sty = newStyle
		if d.ConfirmedBand() {
			v = "N"
			sty = oldStyle
		}
	}
	rows = append(rows, row{"New DXCC on band", sty.Render(v)})

	// New DXCC on mode
	v = "?"
	sty = DimStyle
	if hasBand && hasMode {
		v = "Y"
		sty = newStyle
		if d.ConfirmedBandMode() {
			v = "N"
			sty = oldStyle
		}
	}
	rows = append(rows, row{"New DXCC on mode", sty.Render(v)})

	// LoTW member — last, separate logic: N = red.
	v = "N"
	sty = badStyle
	if d.LoTW() {
		v = "Y"
		sty = oldStyle
	}
	rows = append(rows, row{"LoTW member", sty.Render(v)})

	var lines []string
	for _, r := range rows {
		lbl := S.FormLabelWide.Align(lipgloss.Right).Render(r.label)
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ", r.value))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
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
	cacheKey := m.partnerMapCacheKey()
	if m.partnerMapCacheSig == cacheKey && m.partnerMapCache != "" {
		return m.partnerMapCache
	}
	ownGrid := m.App.Logbook.Station.Grid
	partnerGrid := d.Grid
	switch {
	case ownGrid == "":
		m.partnerMapCache = DimStyle.Render("Set your grid in station config to enable the map")
	case partnerGrid == "" && d.Lat == "":
		m.partnerMapCache = DimStyle.Render("No partner location — enter a grid or use QRZ lookup")
	default:
		if mapAvailH >= NativeMapHeight+5 && mapW >= NativeMapWidth {
			ownLat, ownLon := gridToLatLon(ownGrid)
			pl, plon := 0.0, 0.0
			if partnerGrid != "" {
				pl, plon = gridToLatLon(partnerGrid)
			}
			if d.Lat != "" {
				pl = parseCoord(d.Lat)
				plon = parseCoord(d.Lon)
			}
			if s := renderWorldMap(ownLat, ownLon, pl, plon, mapW, NativeMapHeight); s != "" {
				m.partnerMapCache = s
			} else {
				m.partnerMapCache = DimStyle.Render("Terminal too small for map")
			}
		} else {
			m.partnerMapCache = DimStyle.Render("Terminal too small for map")
		}
	}
	m.partnerMapCacheSig = cacheKey
	return m.partnerMapCache
}

func (m *Model) partnerMapCacheKey() string {
	var b strings.Builder
	fmt.Fprintf(&b, "w%d|h%d|own:%s|", m.width, m.height, m.App.Logbook.Station.Grid)
	if m.partnerData != nil {
		fmt.Fprintf(&b, "p:%s|g:%s|lat:%s|lon:%s|",
			m.partnerData.Callsign, m.partnerData.Grid, m.partnerData.Lat, m.partnerData.Lon)
	}
	return b.String()
}

func (m *Model) invalidatePartnerMapCache() {
	m.partnerMapCache = ""
	m.partnerMapCacheSig = ""
}
