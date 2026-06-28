package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/solar"
)

// Pre-allocated solar cell base style — only Width changes per render.
var solarCellBaseStyle = lipgloss.NewStyle().Align(lipgloss.Right)

// Pre-allocated solar label style — invariant muted foreground.
var solarLabelStyle = lipgloss.NewStyle().Foreground(P.TextMuted)

// Pre-allocated solar placeholder base style — only Width varies.
var solarPlaceholderBaseStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(P.Border).
	Padding(0, 2).
	Height(10).
	Align(lipgloss.Center, lipgloss.Top)

// Fixed box width: 5 columns × 6 cells + 2 borders + 4 padding = 36.
const solarBoxW = 36

// Pre-allocated solar border box style — invariant.
var solarBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(P.Border).
	Padding(0, 2).
	Width(solarBoxW).
	Align(lipgloss.Right, lipgloss.Top)

// renderSolarPanel builds a compact bordered solar conditions panel for the
// right side of the QSO form on wide screens. Returns empty string when
// no solar data is available and we're not in a loading state.
//
// Cached — only rebuilds when solar data content or loading state changes.
// Box width is fixed so borders never resize when transitioning from
// placeholder to real data.
func (m *Model) renderSolarPanel(availW int) string {
	if availW < 30 {
		return ""
	}

	d := m.solar.data

	// Loading / offline / failed placeholder — cached separately.
	if d == nil {
		if m.solar.failed {
			return ""
		}
		placeholderSig := fmt.Sprintf("ld:%d|%v", availW, m.inetOnline)
		if m.solar.cachedSig == placeholderSig && m.solar.cachedView != "" {
			return m.solar.cachedView
		}
		var result string
		if m.inetOnline {
			result = m.renderSolarPlaceholder(solarBoxW, "Loading hamqsl.com\u2026")
		}
		m.solar.cachedSig = placeholderSig
		m.solar.cachedView = result
		return result
	}

	// Build cache signature.
	var sigB strings.Builder
	fmt.Fprintf(&sigB, "%d|%d|%d|%.1f|%d|%s|%s|%s|%s|%.0f|%.0f|%.0f|",
		availW, d.SolarFlux, d.AIndex, d.KIndex, d.Sunspots,
		d.GeomagField, d.SignalNoise, d.XRay, d.Updated,
		d.SolarWind, d.ProtonFlux, d.Aurora)
	for _, b := range solar.BandOrder {
		fmt.Fprintf(&sigB, "%s:%s|", b+"_day", d.Bands[b+"_day"])
		fmt.Fprintf(&sigB, "%s:%s|", b+"_night", d.Bands[b+"_night"])
	}
	sig := sigB.String()
	if m.solar.cachedSig == sig && m.solar.cachedView != "" {
		return m.solar.cachedView
	}

	err := S.Error // red for problematic values.

	// --- Thresholds based on N0NBH hamqsl.com reference table ---
	// RED = actually problematic for HF propagation.
	sfiStyle, aStyle, kStyle, ssnStyle := solarLabelStyle, solarLabelStyle, solarLabelStyle, solarLabelStyle
	lbl := solarLabelStyle // local alias for remaining uses
	if d.SolarFlux < 70 {
		sfiStyle = err // bands above 40m unusable
	}
	if d.AIndex >= 15 {
		aStyle = err // Active geomagnetic field
	}
	if d.KIndex >= 5 {
		kStyle = err // Minor Geomagnetic Storm (K=4 is just "Active")
	}
	if d.Sunspots < 20 {
		ssnStyle = err // poor band conditions
	}

	// Compact summary: labels dimmed, values more visible.
	dim := DimStyle
	summary := dim.Render("SFI") + " " + sfiStyle.Render(fmt.Sprintf("%d", d.SolarFlux)) +
		" " + dim.Render("A") + " " + aStyle.Render(fmt.Sprintf("%d", d.AIndex)) +
		" " + dim.Render("K") + " " + kStyle.Render(fmt.Sprintf("%.1f", d.KIndex)) +
		" " + dim.Render("SSN") + " " + ssnStyle.Render(fmt.Sprintf("%d", d.Sunspots))

	// --- Band condition table ---
	const colW = 6 // 1-char gap between columns (band names are 5 chars)
	bands := solar.BandOrder
	times := []string{"Day", "Night"}

	renderCell := func(s string, w int) string {
		return solarCellBaseStyle.Width(w).Render(s)
	}

	// Header row — first column same width as data cols.
	header := renderCell("", colW)
	for _, b := range bands {
		header += renderCell(solar.BandShort[b], colW)
	}

	// Data rows.
	var tbl []string
	tbl = append(tbl, dim.Render(header))
	for _, tm := range times {
		row := dim.Render(renderCell(tm, colW))
		for _, b := range bands {
			key := b + "_" + strings.ToLower(tm)
			cond := strings.TrimSpace(d.Bands[key])
			style := lbl
			if strings.EqualFold(cond, "good") {
				style = S.Success
			}
			row += style.Render(renderCell(cond, colW))
		}
		tbl = append(tbl, row)
	}
	table := lipgloss.JoinVertical(lipgloss.Left, tbl...)

	// --- Extra row 1: Geomag + Signal noise ---
	var extra1B strings.Builder
	if d.GeomagField != "" {
		gfStyle := lbl
		upper := strings.ToUpper(d.GeomagField)
		if strings.Contains(upper, "STORM") {
			gfStyle = err
		}
		extra1B.WriteString(dim.Render("Geomag"))
		extra1B.WriteByte(' ')
		extra1B.WriteString(gfStyle.Render(d.GeomagField))
	}
	if d.SignalNoise != "" {
		if extra1B.Len() > 0 {
			extra1B.WriteByte(' ')
		}
		sigStyle := lbl
		us := strings.ToUpper(d.SignalNoise)
		if strings.Contains(us, "S6") || strings.Contains(us, "S7") ||
			strings.Contains(us, "S8") || strings.Contains(us, "S9") {
			sigStyle = err // Moderate+ geomagnetic storm noise level
		}
		extra1B.WriteString(dim.Render("Sig"))
		extra1B.WriteByte(' ')
		extra1B.WriteString(sigStyle.Render(d.SignalNoise))
	}
	extra1 := extra1B.String()

	// --- Extra row 2: SW, PF, Aur, XRY ---
	swStyle, pfStyle, aurStyle, xrStyle := lbl, lbl, lbl, lbl
	if d.SolarWind > 500 {
		swStyle = err // Moderate Geomagnetic Storm
	}
	if d.ProtonFlux > 100 {
		pfStyle = err // Minor Solar Radiation Storm (PF=10 is just Active)
	}
	if d.Aurora > 7 {
		aurStyle = err // Minor Storm aurora level (Aur=5-7 is Active)
	}
	if strings.HasPrefix(strings.ToUpper(d.XRay), "M") ||
		strings.HasPrefix(strings.ToUpper(d.XRay), "X") {
		xrStyle = err
	}
	var extra2B strings.Builder
	extra2B.WriteString(dim.Render("SW"))
	extra2B.WriteByte(' ')
	extra2B.WriteString(swStyle.Render(fmt.Sprintf("%.0f", d.SolarWind)))
	extra2B.WriteByte(' ')
	extra2B.WriteString(dim.Render("PF"))
	extra2B.WriteByte(' ')
	extra2B.WriteString(pfStyle.Render(fmt.Sprintf("%.0f", d.ProtonFlux)))
	extra2B.WriteByte(' ')
	extra2B.WriteString(dim.Render("Aur"))
	extra2B.WriteByte(' ')
	extra2B.WriteString(aurStyle.Render(fmt.Sprintf("%.0f", d.Aurora)))
	extra2B.WriteByte(' ')
	extra2B.WriteString(dim.Render("XRY"))
	extra2B.WriteByte(' ')
	extra2B.WriteString(xrStyle.Render(d.XRay))
	extra2 := extra2B.String()

	// --- Assemble: left-aligned, no stretching ---
	// Build content first to measure natural width.
	var contentParts []string
	contentParts = append(contentParts, summary)
	if extra2 != "" {
		contentParts = append(contentParts, extra2)
	}
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, table)
	if extra1 != "" {
		contentParts = append(contentParts, "")
		contentParts = append(contentParts, extra1)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)

	// Fixed-width box — borders never change size.
	result := solarBoxStyle.Render(content)

	m.solar.cachedSig = sig
	m.solar.cachedView = result
	return result
}

// renderSolarPlaceholder returns a bordered box with a centered message.
// Used while waiting for the first solar data fetch.
func (m *Model) renderSolarPlaceholder(availW int, msg string) string {
	if availW < 32 {
		availW = 32
	}
	return solarPlaceholderBaseStyle.Width(availW).Render(DimStyle.Render(msg))
}
