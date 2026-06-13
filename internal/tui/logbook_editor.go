package tui

import (
	"database/sql"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

type editorMode int

const (
	edModeList editorMode = iota
	edModeConfirmDelete
	edModeConfirmPurge
	edModeConfirmWLSend
	edModeConfirmNormalize
	edModeEdit
)

type qsoEditField int

const (
	qefCall qsoEditField = iota
	qefDate
	qefTimeOn
	qefTimeOff
	qefBand
	qefFreq
	qefFreqRx
	qefMode
	qefSubmode
	qefRSTSent
	qefRSTRcvd
	qefGrid
	qefName
	qefQTH
	qefCountry
	qefComment
	qefNotes
	qefTXPower
	qefStationCall
	qefOperator
	qefMyGrid
	qefWLStatus
	qefMyRig
	qefMyAntenna
	qefSource
	qefDistance
	qefBearing
	qefIOTA
	qefSOTA
	qefPOTA
	qefWWFF
	qefMySOTA
	qefMyPOTA
	qefMyWWFF
	qefCount
)

var qefLabels = []string{
	"Call", "Date", "Time On", "Time Off", "Band", "Frequency", "Freq RX",
	"Mode", "Submode", "RST Sent", "RST Rcvd", "Grid", "Name",
	"QTH", "Country", "Comment", "Notes", "TX Power",
	"Station Call", "Operator", "My Grid", "WL Status", "My Rig", "My Antenna",
	"Source", "Distance km", "Bearing",
	"IOTA", "SOTA Ref", "POTA Ref", "WWFF Ref",
	"My SOTA", "My POTA", "My WWFF",
}

type LogbookEditor struct {
	db             *sql.DB
	qsos           []qso.QSO
	table          table.Model
	mode           editorMode
	confirm        *Confirm
	editing        *qso.QSO
	fields         [qefCount]textinput.Model
	focus          qsoEditField
	done           bool
	needsReload    bool
	width          int
	height         int
	wlURL          string
	wlKey          string
	wlStationID    string
	wlStationCall  string
	logStationOp   string
	logStationGrid string
	mismatchQSOs   []qso.QSO
	mismatchFields []string
}

var editorCols = []table.Column{
	{Title: "ID", Width: 3}, {Title: "Date", Width: 10}, {Title: "Time", Width: 8},
	{Title: "Call", Width: 7}, {Title: "Band", Width: 5}, {Title: "Mode", Width: 5},
	{Title: "RSTs", Width: 4}, {Title: "RSTr", Width: 4}, {Title: "Grid", Width: 6},
	{Title: "DXCC", Width: 6}, {Title: "Name", Width: 7}, {Title: "QTH", Width: 8},
	{Title: "WL", Width: 8}, {Title: "Comment", Width: 10},
}

func editorRow(q *qso.QSO) table.Row {
	b := qso.NormalizeBand(q.Band)
	if b == "" && q.Freq > 0 {
		b = fmt.Sprintf("%.0f", q.Freq)
	}
	wl := q.WavelogUploaded
	if wl != "yes" && wl != "no" {
		wl = ""
	}
	return table.Row{
		fmt.Sprintf("%d", q.ID), formatDate(q.QSODate), formatTime(q.TimeOn),
		q.Call, b, q.Mode, q.RSTSent, q.RSTRcvd, q.GridSquare,
		q.Country, q.Name, q.QTH, wl, q.Comment,
	}
}

func (le *LogbookEditor) buildTable() {
	var rows []table.Row
	for _, q := range le.qsos {
		rows = append(rows, editorRow(&q))
	}
	t := table.New(
		table.WithColumns(editorCols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(le.height-8),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	le.table = t
}

func NewLogbookEditor(db *sql.DB, wlURL, wlKey, wlStationID, wlStationCall, logStationOp, logStationGrid string) *LogbookEditor {
	le := &LogbookEditor{db: db, mode: edModeList, wlURL: wlURL, wlKey: wlKey, wlStationID: wlStationID, wlStationCall: wlStationCall, logStationOp: logStationOp, logStationGrid: logStationGrid}
	for i := qsoEditField(0); i < qefCount; i++ {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 40
		switch i {
		case qefCall:
			ti.CharLimit = 20
		case qefDate:
			ti.CharLimit = 10
		case qefTimeOn, qefTimeOff:
			ti.CharLimit = 8
		case qefBand, qefGrid, qefMyGrid, qefTXPower, qefRSTSent, qefRSTRcvd:
			ti.CharLimit = 8
		case qefFreq, qefFreqRx:
			ti.CharLimit = 16
		case qefMode:
			ti.CharLimit = 12
		case qefSubmode:
			ti.CharLimit = 16
		case qefSource:
			ti.CharLimit = 10
		case qefDistance, qefBearing:
			ti.CharLimit = 10
		case qefComment:
			ti.CharLimit = 80
		case qefNotes:
			ti.CharLimit = 200
		case qefSOTA, qefPOTA, qefWWFF, qefIOTA, qefMySOTA, qefMyPOTA, qefMyWWFF:
			ti.CharLimit = 20
		case qefWLStatus:
			ti.CharLimit = 8
		}
		le.fields[i] = ti
	}
	return le
}

func (le *LogbookEditor) Init() tea.Cmd { return nil }

func (le *LogbookEditor) SetQSOS(qsos []qso.QSO) { le.qsos = qsos }

// UpdateWLStatus updates the WL status field in the currently editing form.
func (le *LogbookEditor) UpdateWLStatus(qID int64, status string) {
	if le.editing != nil && le.editing.ID == qID {
		le.fields[qefWLStatus].SetValue(status)
		le.editing.WavelogUploaded = status
	}
}

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
	le.focus = (le.focus + 1) % qefCount
	le.fields[le.focus].Focus()
}

func (le *LogbookEditor) prevField() {
	le.fields[le.focus].Blur()
	if le.focus == 0 {
		le.focus = qefCount - 1
	} else {
		le.focus--
	}
	le.fields[le.focus].Focus()
}

type editorMsg struct {
	deleted    int64
	saved      int64
	purged     bool
	wlQSOID    int64
	wlCall     string
	wlOK       bool
	normalized int
	err        error
}

func (le *LogbookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		le.width = msg.Width
		le.height = msg.Height

	case editorMsg:
		if msg.err != nil {
			// error handled by caller via toast
		}
		if msg.deleted != 0 || msg.saved != 0 || msg.purged || msg.wlCall != "" {
			le.mode = edModeList
			le.needsReload = true
		}
		if msg.normalized > 0 {
			// Normalization done, now upload all unsent QSOs
			var unsent []qso.QSO
			for _, q := range le.qsos {
				if q.WavelogUploaded != "yes" {
					unsent = append(unsent, q)
				}
			}
			return le, le.uploadBatch(unsent)
		}

	case tea.KeyPressMsg:
		k := msg.String()

		if le.mode == edModeConfirmDelete || le.mode == edModeConfirmPurge || le.mode == edModeConfirmWLSend || le.mode == edModeConfirmNormalize {
			switch k {
			case "y", "Y":
				return le, le.doConfirm()
			default:
				le.mode = edModeList
			}
			return le, nil
		}

		if le.mode == edModeEdit {
			switch k {
			case "ctrl+s":
				return le, le.doSave()
			case "ctrl+w":
				return le, le.doUploadToWavelog()
			case "esc", "f5":
				le.mode = edModeList
			case "tab", "down":
				le.nextField()
			case "shift+tab", "up":
				le.prevField()
			default:
				if le.focus != qefWLStatus {
					le.fields[le.focus], _ = le.fields[le.focus].Update(msg)
				}
			}
			return le, nil
		}

		// modeList — table handles navigation
		switch k {
		case "f5", "esc":
			le.done = true
		case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
			var cmd tea.Cmd
			le.table, cmd = le.table.Update(msg)
			return le, cmd
		case "delete":
			if len(le.qsos) > 0 {
				le.mode = edModeConfirmDelete
			}
		case "ctrl+w":
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" && len(le.qsos) > 0 {
				le.mode = edModeConfirmWLSend
			}
		case "e", "enter":
			if len(le.qsos) > 0 {
				idx := le.table.Cursor()
				if idx >= len(le.qsos) {
					idx = 0
				}
				q := le.qsos[idx]
				le.editing = &q
				le.fillEditForm(&q)
				le.focus = qefCall
				le.fields[le.focus].Focus()
				le.mode = edModeEdit
			}
		case "p":
			le.mode = edModeConfirmPurge
		}
	}

	return le, nil
}

func (le *LogbookEditor) doConfirm() tea.Cmd {
	if le.mode == edModeConfirmWLSend {
		return le.doBatchUpload()
	}
	if le.mode == edModeConfirmNormalize {
		return le.doNormalizeAndUpload()
	}
	if le.mode == edModeConfirmDelete {
		id := le.qsos[le.table.Cursor()].ID
		return func() tea.Msg {
			return editorMsg{deleted: id, err: store.DeleteQSO(le.db, id)}
		}
	}
	return func() tea.Msg {
		return editorMsg{purged: true, err: store.PurgeQSOs(le.db)}
	}
}

func (le *LogbookEditor) doSave() tea.Cmd {
	q := le.readEditForm()
	return func() tea.Msg {
		return editorMsg{saved: q.ID, err: store.UpdateQSO(le.db, q)}
	}
}

// Upload methods moved to editor_upload.go.

func (le *LogbookEditor) FooterText() string {
	hasWL := le.wlURL != "" && le.wlKey != "" && le.wlStationID != ""
	switch le.mode {
	case edModeConfirmDelete:
		return "Delete this QSO? (y/N)"
	case edModeConfirmPurge:
		return "Purge ALL QSOs? This cannot be undone. (y/N)"
	case edModeConfirmWLSend:
		return "Send all unsent QSOs to Wavelog? (y/N)"
	case edModeConfirmNormalize:
		return fmt.Sprintf("Normalize %d QSOs (station %s) then upload? (y/N)", len(le.mismatchQSOs), strings.Join(le.mismatchFields, "/"))
	case edModeEdit:
		if hasWL {
			return "ctrl+s to save  ctrl+w Upload to Wavelog  Tab/Up/Dn to navigate  Esc to discard"
		}
		return "ctrl+s to save  Tab/Up/Dn to navigate  Esc to discard"
	default:
		if hasWL {
			return "Up/Dn scroll  Enter/e edit  Del delete  ctrl+w send all to Wavelog  p purge  F5/Esc close"
		}
		return "Up/Dn scroll  Enter/e edit  Del delete  p purge  F5/Esc close"
	}
}

func (le *LogbookEditor) View() tea.View {
	if le.done {
		return tea.NewView("")
	}
	bodyW := ContentWidth(le.width)

	switch le.mode {
	case edModeConfirmDelete:
		return tea.NewView(le.viewConfirm("Delete QSO", bodyW))
	case edModeConfirmPurge:
		return tea.NewView(le.viewConfirm("Purge Logbook", bodyW))
	case edModeConfirmWLSend:
		unsent := 0
		for _, q := range le.qsos {
			if q.WavelogUploaded != "yes" {
				unsent++
			}
		}
		return tea.NewView(le.viewConfirm(fmt.Sprintf("Send %d unsent QSOs to Wavelog", unsent), bodyW))
	case edModeConfirmNormalize:
		return tea.NewView(le.viewNormalizeConfirm(bodyW))
	case edModeEdit:
		return tea.NewView(le.viewEdit(bodyW))
	default:
		le.buildTable()
		return tea.NewView(section("── Logbook Editor ", bodyW) + "\n" + le.table.View())
	}
}

func (le *LogbookEditor) viewConfirm(action string, bodyW int) string {
	var b strings.Builder
	b.WriteString(section("── "+action+" ", bodyW))
	b.WriteString("\n\n")
	b.WriteString("  Are you sure? (y/N)")
	return b.String()
}

func (le *LogbookEditor) viewNormalizeConfirm(bodyW int) string {
	var b strings.Builder
	fields := strings.Join(le.mismatchFields, ", ")
	b.WriteString(section(fmt.Sprintf("── Normalize %d QSOs ", len(le.mismatchQSOs)), bodyW))
	b.WriteString(fmt.Sprintf("\n\n  %d unsent QSOs have mismatched station %s.", len(le.mismatchQSOs), fields))
	b.WriteString("\n  Normalize them to match the Wavelog station profile,")
	b.WriteString("\n  update the local database, then upload?")
	b.WriteString("\n\n  Are you sure? (y/N)")
	return b.String()
}

func (le *LogbookEditor) viewEdit(bodyW int) string {
	var b strings.Builder
	b.WriteString(section("── Edit QSO ", bodyW))
	b.WriteString("\n\n")

	colW := (bodyW - 4) / 2
	if colW < 28 {
		colW = bodyW - 2
	}
	half := (qefCount + 1) / 2

	for i := qsoEditField(0); i < half; i++ {
		left := le.renderEditField(i, colW)
		rightIdx := i + half
		if rightIdx < qefCount {
			right := le.renderEditField(rightIdx, colW)
			b.WriteString(left)
			b.WriteString("  ")
			b.WriteString(right)
			b.WriteString("\n")
		} else {
			b.WriteString(left)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (le *LogbookEditor) renderEditField(f qsoEditField, colW int) string {
	label := qefLabels[f]
	val := le.fields[f].View()
	focused := f == le.focus
	prefix := " "
	lbl := fit(label, 13)
	if focused {
		prefix = CursorStyle.Render(">")
		lbl = CursorStyle.Render(lbl)
	} else {
		lbl = LabelStyle.Render(lbl)
	}
	valStyle := InputStyle
	if f == qefWLStatus {
		valStyle = DimStyle // read-only indicator
	}
	field := prefix + lbl + " " + valStyle.Render(val)
	return lipgloss.NewStyle().Width(colW).Render(field)
}
