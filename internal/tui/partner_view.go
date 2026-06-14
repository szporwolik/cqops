package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qrz"
)

// =============================================================================
// Partner view rendering (F2 screen)
// =============================================================================

// viewPartner renders the full partner details screen including QRZ info,
// Wavelog info, and the ASCII world map.
func (m *Model) viewPartner() string {
	d := m.partnerData
	if d == nil {
		d = m.formPartnerData()
		if d == nil || d.Callsign == "" {
			return ""
		}
	}
	bodyW := m.width
	if bodyW < 30 {
		bodyW = 30
	}

	halfW := (bodyW - 1) / 2
	if halfW < 20 {
		halfW = bodyW
	}

	info := m.renderPartnerInfo(d, halfW)
	infoBox := S.QSOFormBox.Width(halfW).Render(info)
	infoH := lipgloss.Height(infoBox)

	wlContent := m.renderWLInfo(halfW)
	wlBox := S.QSOFormBox.Width(halfW).Height(infoH).Render(wlContent)

	gap := lipgloss.NewStyle().Width(1).Render("")
	var topRow string
	if halfW >= 20 {
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, infoBox, gap, wlBox)
	} else {
		topRow = lipgloss.JoinVertical(lipgloss.Left, infoBox, wlBox)
	}

	// Map section
	mapW := bodyW - 2
	topH := lipgloss.Height(topRow)
	mapAvailH := contentHeight(m.height) - topH
	if mapAvailH < 3 {
		mapAvailH = 3
	}

	cacheKey := m.partnerMapCacheKey()
	var mapInner string
	if m.partnerMapCacheSig == cacheKey && m.partnerMapCache != "" {
		mapInner = m.partnerMapCache
	} else {
		ownGrid := m.App.Logbook.Station.Grid
		partnerGrid := d.Grid

		switch {
		case ownGrid == "":
			mapInner = DimStyle.Render("Set your grid in station config to enable the map")
		case partnerGrid == "" && d.Lat == "":
			mapInner = DimStyle.Render("No partner location — enter a grid or use QRZ lookup")
		default:
			if mapAvailH >= NativeMapHeight+5 && mapW >= NativeMapWidth {
				ownLat, ownLon := gridToLatLon(ownGrid)
				partnerLat, partnerLon := 0.0, 0.0

				if partnerGrid != "" {
					partnerLat, partnerLon = gridToLatLon(partnerGrid)
				} else if d.Lat != "" {
					partnerLat = parseCoord(d.Lat)
					partnerLon = parseCoord(d.Lon)
				}

				mapStr := renderWorldMap(ownLat, ownLon, partnerLat, partnerLon, mapW, NativeMapHeight)
				if mapStr != "" {
					mapInner = mapStr
				} else {
					mapInner = DimStyle.Render("Terminal too small for map")
				}
			} else {
				mapInner = DimStyle.Render("Terminal too small for map")
			}
		}
		m.partnerMapCacheSig = cacheKey
		m.partnerMapCache = mapInner
	}

	mapBox := S.MapBox.Width(bodyW).Render(mapInner)

	mapBoxH := lipgloss.Height(mapBox)
	fillerH := contentHeight(m.height) - topH - mapBoxH
	if fillerH < 0 {
		fillerH = 0
	}
	filler := lipgloss.NewStyle().Height(fillerH).Render("")
	return lipgloss.JoinVertical(lipgloss.Left, topRow, mapBox, filler)
}

// formPartnerData builds a CallData from the current QSO form fields.
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

// renderPartnerInfo renders the QRZ partner info column.
func (m *Model) renderPartnerInfo(d *qrz.CallData, maxW int) string {
	type row struct{ label, value string }
	var rows []row
	add := func(label, value string) {
		if value != "" {
			rows = append(rows, row{label, value})
		}
	}
	add("Callsign", d.Callsign)
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
	if d.ImageURL != "" {
		add("Photo", osc8Link(d.ImageURL, "CLICK"))
	}

	if len(rows) == 0 {
		return ""
	}

	labelW := 13
	indentW := 2
	valW := maxW - indentW - labelW - 1
	if valW < 8 {
		valW = 8
	}

	lblStyle := LabelStyle.Width(labelW).Align(lipgloss.Right)
	valStyle := ValueStyle.Width(valW)

	var lines []string
	for _, r := range rows {
		label := lblStyle.Render(r.label)
		value := r.value
		if r.label == "Photo" {
			value = S.Info.Width(valW).Align(lipgloss.Left).Render(value)
		} else {
			value = valStyle.Render(truncate(r.value, valW))
		}
		indent := lipgloss.NewStyle().Width(indentW).Render("")
		gap := lipgloss.NewStyle().Width(1).Render("")
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, indent, label, gap, value))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderWLInfo renders the Wavelog lookup info column.
func (m *Model) renderWLInfo(maxW int) string {
	d := m.wlPrivateData
	if !m.App.Config.Wavelog.Enabled || m.App.Config.Wavelog.URL == "" || m.App.Config.Wavelog.APIKey == "" {
		return lipgloss.NewStyle().
			Width(maxW - 4).
			Align(lipgloss.Center).
			Foreground(P.TextMuted).
			Render("Wavelog not configured")
	}
	if d == nil {
		msg := "WL lookup pending\u2026"
		if m.wlLookupDone {
			msg = "No WL data"
		}
		return lipgloss.NewStyle().
			Width(maxW - 4).
			Align(lipgloss.Center).
			Foreground(P.TextMuted).
			Render(msg)
	}

	type row struct{ label, value string }
	var rows []row
	add := func(label string, value bool, yes, no string) {
		if yes == "" {
			yes = "yes"
		}
		if no == "" {
			no = "\u2014"
		}
		v := no
		if value {
			v = yes
		}
		rows = append(rows, row{label, v})
	}

	hasBand := m.wlLastBand != ""
	hasMode := m.wlLastMode != ""

	add("Call worked", d.Worked(), "Y", "N")
	add("Call on band", hasBand && d.WorkedBand(), "Y", tern(hasBand, "N", "?"))
	add("Call on mode", hasBand && hasMode && d.WorkedBandMode(), "Y", tern(hasBand && hasMode, "N", "?"))
	add("LoTW member", d.LoTW(), "Y", "N")
	add("DXCC confirmed", d.DXCCConfirmed(), "Y", "N")
	add("DXCC on band", hasBand && d.ConfirmedBand(), "Y", tern(hasBand, "N", "?"))
	add("DXCC on mode", hasBand && hasMode && d.ConfirmedBandMode(), "Y", tern(hasBand && hasMode, "N", "?"))

	labelW := 15
	indentW := 1
	valW := maxW - indentW - labelW - 1
	if valW < 3 {
		valW = 3
	}

	lblStyle := LabelStyle.Width(labelW).Align(lipgloss.Right)
	yesStyle := S.Success.Width(valW)
	noStyle := S.Error.Width(valW)
	qStyle := S.Warning.Width(valW)

	var lines []string
	for _, r := range rows {
		label := lblStyle.Render(r.label)
		indent := lipgloss.NewStyle().Width(indentW).Render("")
		gap := lipgloss.NewStyle().Width(1).Render("")
		val := r.value
		switch val {
		case "Y":
			val = yesStyle.Render(val)
		case "?":
			val = qStyle.Render(val)
		case "N":
			val = noStyle.Render(val)
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, indent, label, gap, val))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// =============================================================================
// Partner map cache — avoids expensive ASCII map generation on every View().
// =============================================================================

// partnerMapCacheKey computes a cache key from all inputs that affect
// the partner/map rendered output.
func (m *Model) partnerMapCacheKey() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("w%d|h%d|", m.width, m.height))
	b.WriteString(fmt.Sprintf("own:%s|", m.App.Logbook.Station.Grid))
	if m.partnerData != nil {
		b.WriteString(fmt.Sprintf("p:%s|g:%s|lat:%s|lon:%s|",
			m.partnerData.Callsign,
			m.partnerData.Grid,
			m.partnerData.Lat,
			m.partnerData.Lon))
	}
	return b.String()
}

// invalidatePartnerMapCache clears the partner map cache.
func (m *Model) invalidatePartnerMapCache() {
	m.partnerMapCache = ""
	m.partnerMapCacheSig = ""
}
