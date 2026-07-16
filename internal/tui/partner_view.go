package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

const partnerMapMaxW = 142 // also used as max page width for QSO form consistency

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

func hasTerminalGraphics(content string) bool {
	return strings.Contains(content, "\x1b_G") || strings.Contains(content, "\x1b]1337;") || strings.Contains(content, "\x1b]1338;")
}

func renderPartnerPaneContent(content string, boxW int, preserveGraphics bool) string {
	trimmed := strings.TrimRight(content, "\n")
	if preserveGraphics {
		return trimmed
	}
	return menuBoxStyle.Width(boxW).Render(trimmed)
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
	fmt.Fprintf(&sigB, "wldone=%v|wlband=%s|wlmode=%s|qrz=%v|wlcfg=%v|rmap=%v|gray=%v|picpane=%v",
		m.lookup.wlLookupDone, m.lookup.wlLastBand, m.lookup.wlLastMode,
		m.App.Config.Integrations.Callbook.QRZ.Enabled,
		m.App.Logbook.Wavelog != nil && m.App.Logbook.Wavelog.Enabled,
		m.App.Config.General.RenderMap,
		m.App.Config.General.DrawGrayline,
		m.App.Config.General.PictureAtPartnerPane)
	fmt.Fprintf(&sigB, "|fmgrid=%s|gridsrc=%s", m.fields[fieldGrid].Value(), m.gridSource)
	// Kitty map readiness — bust cache when the real Kitty grid
	// replaces the glyph fallback so the map switches quality.
	if m.mapView != nil && m.mapView.KittyOn() && m.App.Config.General.RenderMap {
		kittyContent := m.mapView.KittyContent()
		// Bucket by 256-byte chunks; 0 = not ready (glyph/empty),
		// non-zero = real kitty grid arrived.
		fmt.Fprintf(&sigB, "|kmap=%d", len(kittyContent)>>8)
	}
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

	// Inline partner photo — right-side column on wide screens (≥180 cols).
	showPhoto := m.width >= 180 && m.App.Config.General.PictureAtPartnerPane &&
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

	// Two-panel layout: Callbook (40%) + Worked (60%).
	cbW := leftW * 40 / 100
	wkW := leftW - cbW
	if cbW < 20 {
		cbW = leftW
		wkW = 0
	}
	if wkW < 25 {
		wkW = 25
	}

	cbContent := m.renderCallbookRows(d, cbW-4, 0)
	wkContent := m.renderWorkedPanel(d, wkW-4)

	cbInner := lipgloss.Height(cbContent) + 1
	wkInner := lipgloss.Height(wkContent) + 1
	maxInner := cbInner
	if wkInner > maxInner {
		maxInner = wkInner
	}

	cbBox := m.renderPartnerBox("Callbook"+m.callbookSuffix(), cbContent, cbW, maxInner)
	wkTitle := m.workedTitle()
	wkBox := m.renderPartnerBox(wkTitle, wkContent, wkW, maxInner)
	var topRow string
	if wkW > 0 {
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, cbBox, wkBox)
	} else {
		topRow = cbBox
	}

	// Build left column: topRow + map (if enabled).
	var leftCol string
	if m.App.Config.General.RenderMap {
		topH := lipgloss.Height(topRow)
		mapAvailH := contentHeight(m.height) - topH - 1
		if mapAvailH < 3 {
			mapAvailH = 3
		}
		contentW := leftW - 4
		if contentW < 20 {
			contentW = leftW
		}
		mapBox := m.getOrBuildMap(d, contentW, mapAvailH)
		if mapBox != "" {
			if !hasTerminalGraphics(mapBox) {
				mapBox = lipgloss.PlaceHorizontal(leftW, lipgloss.Center, mapBox)
			}
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
		picContentH := leftH // no header — all space for the image
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
		inner := strings.Join(picLines, "\n")
		// Preserve raw terminal-graphics content when Kitty is active so the
		// picture model's placement stays stable. Normal text still uses the
		// shared menu box wrapper.
		picBox := renderPartnerPaneContent(inner, photoW+1, hasTerminalGraphics(picRaw))
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

func (m *Model) callbookSuffix() string {
	if m.lookup.partnerData == nil {
		return ""
	}
	providers := m.lookup.partnerData.Providers
	call := m.lookup.partnerData.Callsign
	if len(providers) == 0 || call == "" {
		return ""
	}
	seen := map[string]bool{}
	var links []string
	for _, name := range providers {
		if seen[name] {
			continue
		}
		seen[name] = true
		url := providerURLByName(name, call)
		if url != "" {
			links = append(links, osc8Link(url, shortProviderName(name)))
		} else {
			links = append(links, shortProviderName(name))
		}
	}
	if len(links) == 0 {
		return ""
	}
	// Show primary provider + count of additional sources.
	primary := links[0]
	extra := len(links) - 1
	if extra > 0 {
		return " \u00b7 " + primary + " +" + strconv.Itoa(extra)
	}
	return " \u00b7 " + primary
}

// shortProviderName returns a compact display name for a provider.
func shortProviderName(name string) string {
	switch name {
	case "Local Logbook":
		return "Local"
	case "QRZ.com":
		return "QRZ.com"
	case "QRZ.RU":
		return "QRZ.RU"
	case "HamQTH":
		return "HamQTH"
	case "Callook.info":
		return "Callook"
	case "Wavelog":
		return "Wavelog"
	case "CTY.DAT":
		return "CTY.DAT"
	default:
		return name
	}
}

// checkMark returns a checkmark symbol, falling back to ASCII "Y"
// on bare TTYs and terminals without Unicode support (same approach
// as the toast renderer).
func checkMark() string {
	if isTTYWithoutDisplay() {
		return "Y"
	}
	return "\u2713"
}

// middot returns a middle-dot separator, falling back to "-" on bare TTYs
// and terminals without Unicode support.
func middot() string {
	if isTTYWithoutDisplay() {
		return "-"
	}
	return "\u00b7"
}

// resolveClass maps callbook licence class codes to human-readable labels.
func resolveClass(cls string) string {
	// Normalize common FCC/QRZ class codes.
	switch strings.ToUpper(strings.TrimSpace(cls)) {
	case "E", "EXTRA":
		return "Extra"
	case "G", "GENERAL":
		return "General"
	case "T", "TECHNICIAN":
		return "Technician"
	case "A", "ADVANCED":
		return "Advanced"
	case "N", "NOVICE":
		return "Novice"
	case "C", "CLUB":
		return "Club"
	case "I", "1":
		return "Class I"
	case "II", "2":
		return "Class II"
	case "III", "3":
		return "Class III"
	default:
		return cls // pass through unknown codes as-is
	}
}

// providerURLByName maps a provider's display name to its callsign-lookup URL.
func providerURLByName(name, call string) string {
	switch name {
	case "QRZ.com":
		return "https://www.qrz.com/db/" + call
	case "QRZ.RU":
		return "https://www.qrz.ru/db/" + call
	case "HamQTH":
		return "https://www.hamqth.com/" + call
	case "Callook.info":
		return "https://callook.info/" + call
	default:
		return ""
	}
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

// --- Callbook rows ---

// renderCallbookRows builds a compact callbook display with merged fields.
// Fields are combined into fewer rows so the panel uses vertical space
// efficiently.  When maxH > 0, rows beyond the limit are omitted by
// priority (Contact < Refs < Licence < Also < Locator details).
// Call, Name, QTH, Entity, and Locator grid are never omitted.
func (m *Model) renderCallbookRows(d *callbook.Result, maxW int, maxH int) string {
	if d == nil || d.Callsign == "" {
		return ""
	}

	// Resolve continent from DXCC prefix lookup (cached per callsign).
	var cont string
	if d.Callsign != m.rc.dxccContCall {
		m.rc.dxccContCall = d.Callsign
		m.rc.dxccContValue = ""
		if p := m.dxccLookup(d.Callsign); p != nil && p.Continent != "" {
			m.rc.dxccContValue = p.Continent
		}
	}
	if m.rc.dxccContValue != "" {
		cont = m.rc.dxccContValue
	}

	// Grid resolution: prefer QSO-form grid when sourced from REF/SOTA/etc.
	formGrid := strings.TrimSpace(m.fields[fieldGrid].Value())
	grid := d.Grid
	gridFromForm := formGrid != "" && m.gridSource != "" && m.gridSource != gridSourceCallbook
	if gridFromForm {
		grid = formGrid
	}

	// Format coordinates compactly: 50.03309°N 20.22108°E
	var coord string
	if d.Lat != "" || d.Lon != "" {
		coord = formatCoord(d.Lat, "N", "S") + " " + formatCoord(d.Lon, "E", "W")
	}

	// Resolve state code → name when possible.
	stateName := resolveStateName(d.State, d.Country)

	labelW := 8
	valW := maxW - labelW - 1
	if valW < 3 {
		valW = 3
	}

	// Build candidate rows in priority order.  Each row has a priority;
	// when maxH is set, lower-priority rows are dropped first.
	type cand struct {
		label    string
		value    string
		priority int // lower = keep first
	}
	var candidates []cand
	add := func(label, value string, prio int) {
		if value == "" {
			return
		}
		candidates = append(candidates, cand{label, value, prio})
	}

	// Priority 0-9: essential fields (never omitted).
	// Callsign as a clickable link to the main internet callbook.
	_, icbURL := m.internetCallbook()
	callVal := S.Info.Render(d.Callsign)
	if icbURL != "" {
		callVal = osc8Link(strings.Replace(icbURL, "{CALL}", d.Callsign, 1), callVal)
	}
	if d.Class != "" {
		cls := resolveClass(d.Class)
		if cls != "" {
			callVal += "  " + DimStyle.Render("\u00b7 "+cls)
		}
	}
	add("Call", callVal, 0)

	if d.Name != "" {
		add("Name", d.Name, 1)
	}

	var qthParts []string
	if d.QTH != "" {
		qthParts = append(qthParts, d.QTH)
	}
	if stateName != "" {
		qthParts = append(qthParts, stateName)
	} else if d.State != "" {
		qthParts = append(qthParts, d.State)
	}
	if d.Zip != "" {
		qthParts = append(qthParts, d.Zip)
	}
	add("QTH", strings.Join(qthParts, " \u00b7 "), 2)

	var entParts []string
	if d.Country != "" {
		entParts = append(entParts, d.Country)
	}
	if cont != "" {
		entParts = append(entParts, cont)
	}
	// When the call has a foreign operating prefix, d.DXCC is the
	// HOME entity (from HamQTH base-fallback), not the operating
	// entity. Skip it — the worked panel resolves the real DXCC.
	if d.DXCC != "" && !qso.ParseCallsign(d.Callsign).HasForeignPrefix() {
		entParts = append(entParts, "DXCC "+d.DXCC)
	}
	add("Entity", strings.Join(entParts, " \u00b7 "), 3)

	var locParts []string
	if grid != "" {
		g := grid
		if gridFromForm {
			g += "  " + DimStyle.Render("("+string(m.gridSource)+")")
		}
		locParts = append(locParts, g)
	}
	if coord != "" {
		locParts = append(locParts, coord)
	}
	add("Locator", strings.Join(locParts, " \u00b7 "), 4)

	// Priority 10-19: QSL (high operational value, keep when possible).
	var qslParts []string
	ck := checkMark()
	if d.LoTW {
		qslParts = append(qslParts, "LoTW "+ck)
	}
	if d.EQSL {
		qslParts = append(qslParts, "eQSL "+ck)
	}
	if d.QSLManager != "" {
		qslParts = append(qslParts, "via "+d.QSLManager)
	}
	add("QSL", strings.Join(qslParts, " \u00b7 "), 10)

	// Priority 20-29: References.
	var refParts []string
	cqZone := d.CQZone
	ituZone := d.ITUZone
	parsed := qso.ParseCallsign(d.Callsign)
	if parsed.HasForeignPrefix() {
		if p := m.dxccLookup(parsed.OperatingPrefix); p != nil {
			if p.CQZone != 0 {
				cqZone = strconv.Itoa(int(p.CQZone))
			}
			if p.ITUZone != 0 {
				ituZone = strconv.Itoa(int(p.ITUZone))
			}
		}
	}
	if cqZone != "" {
		refParts = append(refParts, "CQ "+cqZone)
	}
	if ituZone != "" {
		refParts = append(refParts, "ITU "+ituZone)
	}
	// TODO: append IOTA, DOK, P-OT, county, oblast from extended fields
	add("Refs", strings.Join(refParts, " \u00b7 "), 20)

	// Priority 30-39: Previous calls / aliases.
	// TODO: populate from d.PreviousCalls, d.Aliases
	add("Also", "", 30)

	// Priority 40-49: Licence.
	// TODO: populate from d.LicenceStatus, d.LicenceExpires
	add("Licence", "", 40)

	// Priority 50-59: Contact (lowest priority).
	var contactParts []string
	if d.Email != "" {
		contactParts = append(contactParts, osc8Link("mailto:"+d.Email, d.Email))
	}
	if d.URL != "" {
		contactParts = append(contactParts, d.URL)
	}
	add("Contact", strings.Join(contactParts, " \u00b7 "), 50)

	// Filter by height if maxH is set.
	if maxH > 0 && len(candidates) > maxH {
		// Sort by priority (lower = keep first), keep top maxH.
		// Since candidates are already in priority order, just truncate.
		candidates = candidates[:maxH]
	}

	// Render rows with segment-aware width handling.
	var lines []string
	for _, c := range candidates {
		lbl := S.FormLabel.Align(lipgloss.Right).Width(labelW).Render(c.label)

		// For Name, allow 2 lines when value is long.
		if c.label == "Name" {
			nameStyle := ValueStyle.Width(valW).MaxWidth(valW)
			if lipgloss.Width(c.value) <= valW {
				lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ",
					nameStyle.Inline(true).Render(c.value)))
			} else {
				lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ",
					nameStyle.Render(truncateText(c.value, valW*2))))
			}
			continue
		}

		// For multi-segment rows (QTH, Entity, Locator, Refs, Contact),
		// remove trailing segments before truncating the entire line.
		val := fitSegments(c.value, valW)
		valStyled := ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(val)
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, lbl, " ", valStyled))
	}

	if len(lines) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// fitSegments truncates a " · "-separated value by removing trailing
// segments until it fits within maxW cells.  Dangling separators and
// trailing whitespace are stripped.  If even the first segment doesn't
// fit, it is truncated with "…".
func fitSegments(value string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= maxW {
		return value
	}
	parts := strings.Split(value, " \u00b7 ")
	for len(parts) > 1 {
		parts = parts[:len(parts)-1]
		candidate := strings.Join(parts, " \u00b7 ")
		if lipgloss.Width(candidate) <= maxW {
			return candidate
		}
	}
	// Only one segment left — truncate with "…".
	return truncateText(parts[0], maxW)
}

// resolveStateName maps a state/province code to its human-readable name
// when a known mapping exists for the given country.
func resolveStateName(code, country string) string {
	if code == "" {
		return ""
	}
	if len(code) > 3 {
		return code // already a full name (e.g. "California")
	}
	up := strings.ToUpper(code)
	// Scope by country to avoid clashes (MA = Massachusetts vs Małopolskie).
	isUS := country == "United States" || country == "USA" || country == "United States of America"
	isCA := country == "Canada"
	if isUS || isCA {
		if name, ok := stateCodeToName[up]; ok {
			return name
		}
	}
	// For other countries, return the code as-is — most non-US/CA codes
	// are voivodeships, regions, or provinces without universal mappings.
	return code
}

// stateCodeToName maps known ADIF/QRZ state codes to readable names.
var stateCodeToName = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas",
	"CA": "California", "CO": "Colorado", "CT": "Connecticut", "DE": "Delaware",
	"FL": "Florida", "GA": "Georgia", "HI": "Hawaii", "ID": "Idaho",
	"IL": "Illinois", "IN": "Indiana", "IA": "Iowa", "KS": "Kansas",
	"KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
	"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi",
	"MO": "Missouri", "MT": "Montana", "NE": "Nebraska", "NV": "Nevada",
	"NH": "New Hampshire", "NJ": "New Jersey", "NM": "New Mexico", "NY": "New York",
	"NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio", "OK": "Oklahoma",
	"OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
	"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah",
	"VT": "Vermont", "VA": "Virginia", "WA": "Washington", "WV": "West Virginia",
	"WI": "Wisconsin", "WY": "Wyoming",
	// Canadian provinces
	"AB": "Alberta", "BC": "British Columbia", "MB": "Manitoba",
	"NB": "New Brunswick", "NL": "Newfoundland and Labrador",
	"NS": "Nova Scotia", "NT": "Northwest Territories", "NU": "Nunavut",
	"ON": "Ontario", "PE": "Prince Edward Island", "QC": "Quebec",
	"SK": "Saskatchewan", "YT": "Yukon",
}

// formatCoord converts a decimal-degree string (e.g. "50.03309") with
// hemisphere suffixes to a compact coordinate: 50.03309°N.
func formatCoord(val, posSuffix, negSuffix string) string {
	if val == "" {
		return ""
	}
	v := strings.TrimSpace(val)
	// Parse as float.
	var f float64
	if _, err := fmt.Sscanf(v, "%f", &f); err != nil {
		return v // non-numeric, return as-is
	}
	suffix := posSuffix
	if f < 0 {
		suffix = negSuffix
		f = -f
	}
	return fmt.Sprintf("%.5f\u00b0%s", f, suffix)
}

// --- Worked panel (unified Logbook + remote sources) ---

// workedTitle returns the panel title with compact source names.
// The remote source name is rendered as a clickable OSC8 hyperlink
// to the Wavelog instance main page.
func (m *Model) workedTitle() string {
	lb := m.App.Logbook
	wl := lb.Wavelog
	hasWl := wl != nil && wl.Enabled && wl.URL != "" && wl.APIKey != ""
	if hasWl {
		src := CompactSourceName("", wl.URL)
		if src != "" {
			// Link to the Wavelog base URL (strip trailing path).
			baseURL := strings.TrimRight(wl.URL, "/")
			return "Worked \u00b7 Local + " + osc8Link(baseURL, src)
		}
	}
	return "Worked \u00b7 Local"
}

type workedRow struct {
	label string
	value string
}

type workedPanelLayout struct {
	TwoColumns    bool
	LeftHeading   string
	RightHeading  string
	LeftRows      []workedRow
	RightRows     []workedRow
	FullWidthRows []workedRow
	LeftWidth     int
	RightWidth    int
	Gap           int
	HistoryScope  historyLabel
}

// renderWorkedPanel builds a unified worked-status panel. The left side
// shows newness (call, band, mode, DXCC, grid) with inline values. The
// upper right side shows the best available history scope, while the lower
// section renders full-width distributions for that same scope.
func (m *Model) renderWorkedPanel(d *callbook.Result, maxW int) string {
	if d == nil || d.Callsign == "" {
		return DimStyle.Render("Enter a callsign to check worked status")
	}

	layout := m.buildWorkedPanelLayout(d, maxW)
	if layout.TwoColumns {
		var lines []string
		leftLines := []string{layout.LeftHeading}
		rightLines := []string{layout.RightHeading}
		for _, r := range layout.LeftRows {
			leftLines = append(leftLines, renderWorkedPanelRow(r.label, r.value, layout.LeftWidth, 10))
		}
		for _, r := range layout.RightRows {
			rightLines = append(rightLines, renderWorkedPanelRow(r.label, r.value, layout.RightWidth, 10))
		}
		for len(leftLines) < len(rightLines) {
			leftLines = append(leftLines, "")
		}
		for len(rightLines) < len(leftLines) {
			rightLines = append(rightLines, "")
		}
		for i := range leftLines {
			if leftLines[i] == "" && rightLines[i] == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top,
					lipgloss.NewStyle().Width(layout.LeftWidth).Render(leftLines[i]),
					strings.Repeat(" ", layout.Gap),
					lipgloss.NewStyle().Width(layout.RightWidth).Render(rightLines[i])))
			}
		}
		if len(layout.FullWidthRows) > 0 {
			if len(lines) > 0 && len(lines[len(lines)-1]) > 0 {
				lines = append(lines, "")
			}
			for _, r := range layout.FullWidthRows {
				lines = append(lines, renderWorkedPanelRow(r.label, r.value, maxW, 10))
			}
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	lines := []string{layout.LeftHeading}
	for _, r := range layout.LeftRows {
		lines = append(lines, renderWorkedPanelRow(r.label, r.value, maxW, 10))
	}
	if layout.RightHeading != "" {
		lines = append(lines, "")
		lines = append(lines, layout.RightHeading)
		for _, r := range layout.RightRows {
			lines = append(lines, renderWorkedPanelRow(r.label, r.value, maxW, 10))
		}
	}
	if len(layout.FullWidthRows) > 0 {
		if len(lines) > 0 && len(lines[len(lines)-1]) > 0 {
			lines = append(lines, "")
		}
		for _, r := range layout.FullWidthRows {
			lines = append(lines, renderWorkedPanelRow(r.label, r.value, maxW, 10))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) buildWorkedPanelLayout(d *callbook.Result, maxW int) workedPanelLayout {
	if d == nil || d.Callsign == "" {
		return workedPanelLayout{}
	}

	call := d.Callsign
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := strings.TrimSpace(m.fields[fieldMode].Value())
	s := m.rc.logStats
	wl := m.lookup.wlPrivateData

	parsed := qso.ParseCallsign(call)
	opEntity := d.Country
	opDXCC := d.DXCC
	if parsed.HasForeignPrefix() {
		if m.App != nil && m.App.BigCTY != nil && parsed.OperatingPrefix != "" {
			if p := m.dxccLookup(parsed.OperatingPrefix); p != nil {
				parsed.OperatingEntity = p.Name
				parsed.OperatingContinent = p.Continent
				parsed.OperatingCQZone = int(p.CQZone)
				parsed.OperatingITUZone = int(p.ITUZone)
				opEntity = p.Name
			}
		}
		if wl != nil && wl.DXCCID() != "" {
			opDXCC = wl.DXCCID()
		} else if m.App.DB != nil && opEntity != "" {
			var dbDXCC string
			_ = m.App.DB.QueryRow(
				`SELECT dxcc FROM qsos WHERE country = ? AND dxcc != '' LIMIT 1`,
				opEntity,
			).Scan(&dbDXCC)
			if dbDXCC != "" {
				opDXCC = dbDXCC
			}
		}
	}

	grid := d.Grid
	formGrid := strings.TrimSpace(m.fields[fieldGrid].Value())
	if formGrid != "" && m.gridSource != "" && m.gridSource != gridSourceCallbook {
		grid = formGrid
	}
	grid4 := ""
	if len(grid) >= 4 {
		grid4 = strings.ToUpper(grid[:4])
	}
	if opEntity == "" {
		opEntity = d.Country
	}

	var ws store.WorkedSummary
	if m.App.DB != nil && call != "" {
		// Cache GetWorkedSummary — it runs 15+ SQL queries and was
		// consuming 17.7% CPU by re-running on every frame.  Only
		// recompute when call, grid4, opDXCC or opEntity change.
		wsKey := call + "|" + grid4 + "|" + opDXCC + "|" + opEntity
		if m.rc.workedSummarySig == wsKey {
			ws = m.rc.workedSummary
		} else {
			var err error
			ws, err = store.GetWorkedSummary(m.App.DB, call, grid4, opDXCC, opEntity)
			if err != nil {
				ws = store.WorkedSummary{}
			}
			m.rc.workedSummary = ws
			m.rc.workedSummarySig = wsKey
		}
	}

	acc := S.Info
	workedMuted := DimStyle
	muted := S.Dim
	valStyle := ValueStyle
	pendDash := DimStyle.Render("—")
	state := func(isNew, known bool) string {
		if !known {
			return " · " + muted.Render("unknown")
		}
		if isNew {
			return " · " + acc.Render("NEW")
		}
		return " · " + workedMuted.Render("worked")
	}
	isNewForCall := func() (bool, bool) {
		if wl != nil {
			return !wl.Worked(), true
		}
		return !s.CallWorked, true
	}

	var leftRows []workedRow
	callNew, callKnown := isNewForCall()
	leftRows = append(leftRows, workedRow{"Call", acc.Render(call) + state(callNew, callKnown)})
	if parsed.BaseCall != "" && !strings.EqualFold(parsed.BaseCall, call) {
		leftRows = append(leftRows, workedRow{"Base", valStyle.Render(parsed.BaseCall)})
	}
	if opEntity != "" || opDXCC != "" {
		dxccNew, dxccKnown := false, true
		if wl != nil && wl.DXCCConfirmed() {
			dxccNew, dxccKnown = false, true
		} else if ws.DXCCHistory.QSOCount > 0 {
			dxccNew, dxccKnown = false, true
		} else {
			dxccNew, dxccKnown = true, true
		}
		label := opEntity
		if label == "" {
			label = opDXCC
		}
		leftRows = append(leftRows, workedRow{"DXCC", valStyle.Render(label) + state(dxccNew, dxccKnown)})
	}
	if grid4 != "" {
		gridNew := ws.GridHistory.QSOCount == 0
		leftRows = append(leftRows, workedRow{"Grid", valStyle.Render(grid4) + state(gridNew, true)})
	}
	if band != "" {
		bNew, bKnown := false, true
		if wl != nil {
			bNew, bKnown = !wl.WorkedBand(), true
		} else {
			bNew, bKnown = !s.CallOnBand, true
		}
		leftRows = append(leftRows, workedRow{"Band", valStyle.Render(band) + state(bNew, bKnown)})
	} else {
		leftRows = append(leftRows, workedRow{"Band", pendDash})
	}
	if mode != "" {
		mNew, mKnown := false, true
		mNew, mKnown = !s.CallOnMode, true
		leftRows = append(leftRows, workedRow{"Mode", valStyle.Render(mode) + state(mNew, mKnown)})
	} else {
		leftRows = append(leftRows, workedRow{"Mode", pendDash})
	}
	if band != "" && mode != "" {
		bmNew, bmKnown := false, true
		if wl != nil {
			bmNew, bmKnown = !wl.WorkedBandMode(), true
		} else {
			bmNew, bmKnown = !s.CallOnMode && !s.CallOnBand, true
		}
		bmVal := valStyle.Render(band + " " + mode)
		leftRows = append(leftRows, workedRow{"Band+Mode", bmVal + state(bmNew, bmKnown)})
	}

	var rightRows []workedRow
	addH := func(l, v string) {
		if v != "" {
			rightRows = append(rightRows, workedRow{l, v})
		}
	}
	callHasQSOs := ws.CallHistory.QSOCount > 0
	hasDXCCHistory := ws.DXCCHistory.QSOCount > 0
	hasGridHistory := ws.GridHistory.QSOCount > 0 && grid4 != ""
	scopeHead := "History"

	// Pick the most specific scope for distribution rows (Bands/Modes/Grids).
	var scopeHist store.ScopeHistory
	var scopePrefix string
	switch {
	case callHasQSOs:
		scopeHist = ws.CallHistory
		scopePrefix = "Call "
	case hasDXCCHistory:
		scopeHist = ws.DXCCHistory
		scopePrefix = "DXCC "
	case hasGridHistory:
		scopeHist = ws.GridHistory
		scopePrefix = "Grid "
	}

	if !callHasQSOs {
		qsoLabel := "QSOs"
		if hasDXCCHistory || hasGridHistory {
			qsoLabel = "Call QSOs"
		}
		if callKnown {
			addH(qsoLabel, "0 · "+muted.Render("first contact"))
		} else {
			addH(qsoLabel, "0 · "+muted.Render("unknown"))
		}
		if ws.DXCCHistory.QSOCount > 0 {
			dinfo := fmt.Sprintf("%d QSOs · %d bands",
				ws.DXCCHistory.QSOCount, ws.DXCCHistory.UniqueBands)
			addH("DXCC log", dinfo)
			if ws.DXCCHistory.LastQSO != nil {
				lq := ws.DXCCHistory.LastQSO
				last := lq.Date
				if lq.Band != "" {
					last += " · " + lq.Band
					if lq.Mode != "" {
						last += " " + lq.Mode
					}
				}
				addH("Last DXCC", last)
			}
		}
	} else {
		addH("QSOs", strconv.Itoa(ws.CallHistory.QSOCount))
		if ws.CallHistory.LastQSO != nil {
			lq := ws.CallHistory.LastQSO
			last := lq.Date
			if lq.Band != "" {
				last += " · " + lq.Band
				if lq.Mode != "" {
					last += " " + lq.Mode
				}
			}
			addH("Last", last)
		}
	}

	var fullWidthRows []workedRow
	if len(scopeHist.BandCounts) > 0 {
		fullWidthRows = append(fullWidthRows, workedRow{scopePrefix + "Bands", formatCountList(scopeHist.BandCounts)})
	}
	if len(scopeHist.ModeCounts) > 0 {
		fullWidthRows = append(fullWidthRows, workedRow{scopePrefix + "Modes", formatCountList(scopeHist.ModeCounts)})
	}
	if len(scopeHist.GridCounts) > 0 && !callHasQSOs {
		fullWidthRows = append(fullWidthRows, workedRow{scopePrefix + "Grids", formatCountList(scopeHist.GridCounts)})
	}

	leftHeading := muted.Render("Current QSO")
	rightHeading := muted.Render(scopeHead)
	if maxW < 58 {
		leftHeading = muted.Render("Status")
	}

	layout := workedPanelLayout{
		TwoColumns:    maxW >= 58,
		LeftHeading:   leftHeading,
		RightHeading:  rightHeading,
		LeftRows:      leftRows,
		RightRows:     rightRows,
		FullWidthRows: fullWidthRows,
		LeftWidth:     maxW*45/100 - 2,
		RightWidth:    maxW - (maxW*45/100 - 2) - 2,
		Gap:           2,
		HistoryScope:  historyLabel("call"),
	}
	if len(scopeHist.BandCounts) == 0 && len(scopeHist.ModeCounts) == 0 && len(scopeHist.GridCounts) == 0 {
		layout.HistoryScope = ""
	}
	if callHasQSOs {
		layout.HistoryScope = "call"
	} else if ws.DXCCHistory.QSOCount > 0 {
		layout.HistoryScope = "dxcc"
	} else if ws.GridHistory.QSOCount > 0 && grid4 != "" {
		layout.HistoryScope = "grid"
	}
	if layout.LeftWidth < 20 {
		layout.LeftWidth = maxW - 2
		layout.RightWidth = 0
		layout.TwoColumns = false
	}
	if layout.RightWidth < 20 {
		layout.RightWidth = maxW - 2
	}
	return layout
}

func renderWorkedPanelRow(label, value string, width, labelWidth int) string {
	if width <= 0 {
		return ""
	}
	lbl := S.Dim.Align(lipgloss.Right).Width(labelWidth).Render(label)
	valW := width - labelWidth - 1
	if valW < 3 {
		valW = 3
	}
	val := ValueStyle.MaxWidth(valW).Inline(true).Render(value)
	return lbl + " " + val
}

// historyLabel describes which scope is being shown.
type historyLabel string

// selectHistoryScope picks the most useful history: call → grid → DXCC.
func selectHistoryScope(ws store.WorkedSummary) struct {
	scope historyLabel
} {
	if ws.CallHistory.QSOCount > 0 {
		return struct{ scope historyLabel }{scope: "call"}
	}
	if ws.GridHistory.QSOCount > 0 && ws.GridHistory.UniqueCalls > 1 {
		return struct{ scope historyLabel }{scope: "grid"}
	}
	if ws.DXCCHistory.QSOCount > 0 {
		return struct{ scope historyLabel }{scope: "dxcc"}
	}
	return struct{ scope historyLabel }{}
}

// formatCountList renders a slice of CountItem as "80m×7 · 40m×3 · +2".
// Items beyond the available width are replaced by a "+N" overflow token.
func formatCountList(items []store.CountItem) string {
	if len(items) == 0 {
		return ""
	}
	// Show up to 4 items; rest become "+N".
	limit := 4
	var parts []string
	for i, it := range items {
		if i >= limit {
			remaining := len(items) - i
			parts = append(parts, "+"+strconv.Itoa(remaining))
			break
		}
		parts = append(parts, it.Value+"\u00d7"+strconv.Itoa(it.Count))
	}
	return strings.Join(parts, " \u00b7 ")
}

// --- Logbook rows (kept for backward compat — unused by new layout) ---

func (m *Model) renderLogbookRows(d *callbook.Result, maxW int) string {
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

func (m *Model) formPartnerData() *callbook.Result {
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	if call == "" {
		return nil
	}
	return &callbook.Result{
		Callsign: call,
		Name:     strings.TrimSpace(m.fields[fieldName].Value()),
		Grid:     strings.TrimSpace(m.fields[fieldGrid].Value()),
		QTH:      strings.TrimSpace(m.fields[fieldQTH].Value()),
		Country:  strings.TrimSpace(m.fields[fieldCountry].Value()),
	}
}

// --- Map cache ---

func (m *Model) getOrBuildMap(d *callbook.Result, mapW, mapAvailH int) string {
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
