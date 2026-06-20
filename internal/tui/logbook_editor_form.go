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
	s(qefMyCQZone, q.MyCQZone)
	s(qefMyITUZone, q.MyITUZone)
	s(qefMyDXCC, q.MyDXCC)
	s(qefMySIG, q.MySIG)
	s(qefMySIGInfo, q.MySIGInfo)
	s(qefSource, q.Source)
	s(qefWLStatus, q.WavelogUploaded)
	sf(qefDistance, q.Distance)
	sf(qefBearing, q.Bearing)
	s(qefIOTA, q.IOTA)
	s(qefSOTA, q.SOTARef)
	s(qefPOTA, q.POTARef)
	s(qefWWFF, q.WWFFRef)
	s(qefSIG, q.SIG)
	s(qefMySOTA, q.MySOTARef)
	s(qefMyPOTA, q.MyPOTARef)
	s(qefMyWWFF, q.MyWWFFRef)
	s(qefCQZone, q.CQZone)
	s(qefITUZone, q.ITUZone)
	s(qefExchSent, q.ExchSent)
	s(qefExchRcvd, q.ExchRcvd)
	s(qefSTX, fmt.Sprintf("%d", q.STX))
	s(qefSRX, fmt.Sprintf("%d", q.SRX))
	s(qefSTXString, q.STXString)
	s(qefSRXString, q.SRXString)
	if q.ContestADIFID != "" {
		s(qefContestID, q.ContestADIFID)
	} else {
		s(qefContestID, q.ContestID)
	}
}

func (le *LogbookEditor) readEditForm() *qso.QSO {
	g := func(f qsoEditField) string { return strings.TrimSpace(le.fields[f].Value()) }
	gf := func(f qsoEditField) float64 {
		var v float64
		fmt.Sscanf(g(f), "%f", &v)
		return v
	}
	gi := func(f qsoEditField) int {
		var v int
		fmt.Sscanf(g(f), "%d", &v)
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
		MyRig: g(qefMyRig), MyAntenna: g(qefMyAntenna),
		MyCQZone: g(qefMyCQZone), MyITUZone: g(qefMyITUZone), MyDXCC: g(qefMyDXCC), MySIG: g(qefMySIG), MySIGInfo: g(qefMySIGInfo), Source: g(qefSource),
		Distance: gf(qefDistance), Bearing: gf(qefBearing),
		IOTA: g(qefIOTA), SOTARef: g(qefSOTA), POTARef: g(qefPOTA), WWFFRef: g(qefWWFF),
		SIG:       g(qefSIG),
		MySOTARef: g(qefMySOTA), MyPOTARef: g(qefMyPOTA), MyWWFFRef: g(qefMyWWFF),
		CQZone: g(qefCQZone), ITUZone: g(qefITUZone),
		ExchSent:        g(qefExchSent),
		ExchRcvd:        g(qefExchRcvd),
		STX:             gi(qefSTX),
		SRX:             gi(qefSRX),
		STXString:       g(qefSTXString),
		SRXString:       g(qefSRXString),
		ContestADIFID:   g(qefContestID),
		ContestID:       le.editing.ContestID, // preserve internal hash for filtering
		WavelogUploaded: g(qefWLStatus),
		CreatedAt:       le.editing.CreatedAt,
	}
}

func (le *LogbookEditor) nextField() {
	le.fields[le.focus].Blur()
	for {
		le.focus = qsoEditField(wrapNext(int(le.focus), int(qefCount)))
		if le.focus != qefWLStatus && le.focus != qefSource {
			break
		}
	}
	le.fields[le.focus].Focus()
}

func (le *LogbookEditor) prevField() {
	le.fields[le.focus].Blur()
	for {
		le.focus = qsoEditField(wrapPrev(int(le.focus), int(qefCount)))
		if le.focus != qefWLStatus && le.focus != qefSource {
			break
		}
	}
	le.fields[le.focus].Focus()
}

// =============================================================================
// Edit form rendering
// =============================================================================

func (le *LogbookEditor) viewEdit(bodyW int, contentH int) string {
	// Cap width to match QSO form — prevents absurdly wide form on large monitors.
	if bodyW > partnerMapMaxW {
		bodyW = partnerMapMaxW
	}
	header := S.Title.Width(bodyW).Render("Edit QSO")

	// Two-column form layout — matches QSO form pattern.
	innerW := bodyW - 2
	if innerW < 20 {
		innerW = 20
	}
	colW := innerW / 2
	if colW < 28 {
		colW = innerW
	}
	colStyle := lipgloss.NewStyle().Width(colW).MaxWidth(colW).Align(lipgloss.Left).Inline(true)

	half := (qefCount + 1) / 2

	var lines []string
	for i := qsoEditField(0); i < half; i++ {
		left := colStyle.Render(le.renderEditField(i, colW))
		rightIdx := i + half
		if rightIdx < qefCount {
			right := colStyle.Render(le.renderEditField(rightIdx, colW))
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, left, right))
		} else {
			lines = append(lines, left)
		}
	}
	formContent := lipgloss.JoinVertical(lipgloss.Left, lines...)
	formBox := drawBorderedBox(formContent, bodyW)

	body := lipgloss.JoinVertical(lipgloss.Left, header, formBox)
	return fillBody(body, contentH)
}

func (le *LogbookEditor) renderEditField(f qsoEditField, colW int) string {
	label := qefLabels[f]
	focused := f == le.focus
	raw := strings.TrimSpace(le.fields[f].Value())

	// Label part — matches QSO form pattern: "> " prefix when focused.
	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
	}

	// Calculate available value width — same pattern as QSO form.
	// labelW accounts for: 2-char prefix + width of rendered label.
	labelW := 2 + lipgloss.Width(lbl)
	vw := colW - labelW - 1 // -1 for the space separator
	if vw < 3 {
		vw = 3
	}
	if vw > 40 {
		vw = 40
	}

	// Set textinput width so bubbles knows the bounds — prevents wrapping.
	if focused {
		ti := le.fields[f]
		ti.SetWidth(vw)
		if lipgloss.Width(raw) > vw {
			ti.SetWidth(vw - 1)
		}
		ti.SetCursor(ti.Position())
		le.fields[f] = ti
	}

	// Value part — WLStatus and Source are always read-only DimStyle.
	var val string
	switch {
	case f == qefWLStatus || f == qefSource:
		if raw == "" {
			raw = "No"
		}
		val = DimStyle.Render(truncateText(raw, vw))
	case focused:
		val = le.fields[f].View()
	case raw == "":
		val = DimStyle.Render("\u2014")
	default:
		val = ValueStyle.Render(truncateText(raw, vw))
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
}
