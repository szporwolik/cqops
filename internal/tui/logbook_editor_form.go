package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Form field population, reading, and navigation
// =============================================================================

func (le *LogbookEditor) fillEditForm(q *qso.QSO) {
	s := func(f qsoEditField, v string) { le.fields[f].SetValue(v) }
	sf := func(f qsoEditField, v float64) {
		if v != 0 {
			le.fields[f].SetValue(fmt.Sprintf("%.4f", v))
		}
	}

	s(qefCall, q.Call)
	s(qefDate, q.QSODate)
	s(qefTimeOn, q.TimeOn)
	s(qefTimeOff, q.TimeOff)
	s(qefBand, q.Band)
	sf(qefFreq, q.Freq)
	sf(qefFreqRx, q.FreqRx)
	s(qefMode, q.Mode)
	s(qefSubmode, q.Submode)
	s(qefRSTSent, q.RSTSent)
	s(qefRSTRcvd, q.RSTRcvd)
	s(qefGrid, q.GridSquare)
	s(qefName, q.Name)
	s(qefQTH, q.QTH)
	s(qefCountry, q.Country)
	s(qefComment, q.Comment)
	s(qefNotes, q.Notes)
	s(qefTXPower, q.TXPower)
	s(qefStationCall, q.StationCallsign)
	s(qefOperator, q.Operator)
	s(qefMyGrid, q.MyGridSquare)
	s(qefMyRig, q.MyRig)
	s(qefMyAntenna, q.MyAntenna)
	s(qefSource, q.Source)
	sf(qefDistance, q.Distance)
	sf(qefBearing, q.Bearing)
	s(qefIOTA, q.IOTA)
	s(qefSOTA, q.SOTARef)
	s(qefPOTA, q.POTARef)
	s(qefWWFF, q.WWFFRef)
	s(qefMySOTA, q.MySOTARef)
	s(qefMyPOTA, q.MyPOTARef)
	s(qefMyWWFF, q.MyWWFFRef)
	// WL Status is read-only — set via async upload
	le.fields[qefWLStatus].SetValue(q.WavelogUploaded)
}

func (le *LogbookEditor) readEditForm() *qso.QSO {
	g := func(f qsoEditField) string { return strings.TrimSpace(le.fields[f].Value()) }
	gf := func(f qsoEditField) float64 {
		var v float64
		fmt.Sscanf(g(f), "%f", &v)
		return v
	}
	return &qso.QSO{
		ID: le.editing.ID, Call: g(qefCall), QSODate: g(qefDate),
		TimeOn: g(qefTimeOn), TimeOff: g(qefTimeOff), Band: g(qefBand),
		Freq: gf(qefFreq), FreqRx: gf(qefFreqRx), Mode: g(qefMode), Submode: g(qefSubmode),
		RSTSent: g(qefRSTSent), RSTRcvd: g(qefRSTRcvd),
		GridSquare: g(qefGrid), Name: g(qefName), QTH: g(qefQTH),
		Country: g(qefCountry), Comment: g(qefComment), Notes: g(qefNotes),
		TXPower: g(qefTXPower), StationCallsign: g(qefStationCall),
		Operator: g(qefOperator), MyGridSquare: g(qefMyGrid),
		MyRig: g(qefMyRig), MyAntenna: g(qefMyAntenna), Source: g(qefSource),
		Distance: gf(qefDistance), Bearing: gf(qefBearing),
		IOTA: g(qefIOTA), SOTARef: g(qefSOTA), POTARef: g(qefPOTA), WWFFRef: g(qefWWFF),
		MySOTARef: g(qefMySOTA), MyPOTARef: g(qefMyPOTA), MyWWFFRef: g(qefMyWWFF),
		WavelogUploaded: g(qefWLStatus),
		CreatedAt:       le.editing.CreatedAt,
	}
}

func (le *LogbookEditor) nextField() {
	le.fields[le.focus].Blur()
	for {
		le.focus = qsoEditField(wrapNext(int(le.focus), int(qefCount)))
		if le.focus != qefWLStatus {
			break
		}
	}
	le.fields[le.focus].Focus()
}

func (le *LogbookEditor) prevField() {
	le.fields[le.focus].Blur()
	for {
		le.focus = qsoEditField(wrapPrev(int(le.focus), int(qefCount)))
		if le.focus != qefWLStatus {
			break
		}
	}
	le.fields[le.focus].Focus()
}

// =============================================================================
// Edit form rendering
// =============================================================================

func (le *LogbookEditor) viewEdit(bodyW int, contentH int) string {
	bg := lipgloss.NewStyle().Background(P.Surface)
	header := bg.Width(bodyW).Render(
		S.Title.Copy().Background(P.Surface).Render("Edit QSO"),
	)

	// Two-column form layout with Surface background on every element.
	innerW := bodyW - 2 // drawBorderedBox borders consume 2 chars
	if innerW < 20 {
		innerW = 20
	}
	colW := innerW / 2
	if colW < 28 {
		colW = innerW
	}
	half := (qefCount + 1) / 2

	gap := bg.Render("  ")
	var lines []string
	for i := qsoEditField(0); i < half; i++ {
		left := le.renderEditField(i, colW)
		rightIdx := i + half
		if rightIdx < qefCount {
			right := le.renderEditField(rightIdx, colW)
			lines = append(lines, bg.Width(innerW).Render(
				lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right),
			))
		} else {
			lines = append(lines, bg.Width(innerW).Render(left))
		}
	}
	formContent := lipgloss.JoinVertical(lipgloss.Left, lines...)
	formBox := drawBorderedBox(formContent, innerW, bodyW)

	// Fill remaining height with Surface background so there is no black
	// gap below the form.
	body := lipgloss.JoinVertical(lipgloss.Left, header, formBox)
	return fillBody(body, contentH)
}

func (le *LogbookEditor) renderEditField(f qsoEditField, colW int) string {
	label := qefLabels[f]
	focused := f == le.focus
	raw := strings.TrimSpace(le.fields[f].Value())

	// Build label part. Focused fields get CursorStyle colouring
	// (no "> " prefix — just the colour change, matching the QSO form).
	lbl := LabelStyle.Render(fit(label, 14))
	if focused {
		lbl = CursorStyle.Render(fit(label, 14))
	}

	// Value part: textinput view only when focused (shows cursor);
	// otherwise render the raw value with ValueStyle, matching the
	// QSO form pattern. WLStatus is always read-only DimStyle.
	var val string
	switch {
	case f == qefWLStatus:
		if raw == "" {
			raw = "No"
		}
		val = DimStyle.Render(raw)
	case focused:
		val = le.fields[f].View()
	case raw == "":
		val = SubtleStyle.Render("\u2014")
	default:
		val = ValueStyle.Render(raw)
	}

	// One-char Surface-background gap — prevents bg leaks between label and value.
	gap := lipgloss.NewStyle().Width(1).Background(P.Surface).Render(" ")

	// Wrap the whole row with Surface background to fill column width.
	return lipgloss.NewStyle().Width(colW).Background(P.Surface).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, lbl, gap, val),
	)
}
