package tui

import (
	"database/sql"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
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
	dialog         *DialogModel // confirm dialog with left/right navigation
	editing        *qso.QSO
	fields         [qefCount]textinput.Model
	focus          qsoEditField
	done           bool
	needsReload    bool
	built          bool
	wlSkipped      int
	wlSkipDetail   string
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

// editorColTiers defines which columns to show at each available width.
var editorColTiers = []struct {
	names []string
}{
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "RSTs", "RSTr"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "RSTs", "RSTr", "DXCC"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment"}},
	{[]string{"Date", "Time", "Call", "WL", "How", "Band", "Mode", "Sub", "RSTs", "RSTr", "DXCC", "Name", "Grid", "QTH", "Comment", "Dist", "Pwr"}},
}

// editorColWidths maps column titles to minimum widths.
var editorColWidths = map[string]int{
	"Date": 10, "Time": 8, "Call": 10, "WL": 3, "How": 4,
	"Band": 5, "Mode": 5, "Sub": 4, "RSTs": 4, "RSTr": 4,
	"DXCC": 6, "Name": 8, "Grid": 6, "QTH": 8,
	"Comment": 12, "Dist": 5, "Pwr": 4,
}

// editorColValue returns the display value for a column and QSO.
func editorColValue(col string, q *qso.QSO) string {
	switch col {
	case "Date":
		return formatDate(q.QSODate)
	case "Time":
		return formatTime(q.TimeOn)
	case "Call":
		return q.Call
	case "WL":
		if q.WavelogUploaded == "yes" {
			return "Y"
		} else if q.WavelogUploaded == "no" {
			return "N"
		}
		return ""
	case "How":
		switch q.Source {
		case "wsjtx":
			return "FTx"
		case "manual":
			return "Man"
		default:
			return q.Source
		}
	case "Band":
		b := qso.NormalizeBand(q.Band)
		if b == "" && q.Freq > 0 {
			return fmt.Sprintf("%.1f", q.Freq)
		}
		return b
	case "Mode":
		return q.Mode
	case "Sub":
		return q.Submode
	case "RSTs":
		return q.RSTSent
	case "RSTr":
		return q.RSTRcvd
	case "DXCC":
		return q.Country
	case "Name":
		return q.Name
	case "Grid":
		return q.GridSquare
	case "QTH":
		return q.QTH
	case "Comment":
		return q.Comment
	case "Dist":
		if q.Distance > 0 {
			return fmt.Sprintf("%.0f", q.Distance)
		}
		return ""
	case "Pwr":
		return q.TXPower
	}
	return ""
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

func (le *LogbookEditor) SetQSOS(qsos []qso.QSO) { le.qsos = qsos; le.buildTable() }

func (le *LogbookEditor) CursorPos() int {
	if le.built {
		return le.table.Cursor()
	}
	return 0
}

func (le *LogbookEditor) QSOCount() int { return len(le.qsos) }

func (le *LogbookEditor) IsEditing() bool { return le.mode == edModeEdit }

func (le *LogbookEditor) isConfirmMode() bool {
	switch le.mode {
	case edModeConfirmDelete, edModeConfirmPurge, edModeConfirmWLSend, edModeConfirmNormalize:
		return true
	}
	return false
}

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
	delCall    string
	delDate    string
	saved      int64
	saveCall   string
	saveDate   string
	purged     bool
	wlQSOID    int64
	wlCall     string
	wlOK       bool
	normalized int
	skipped    int
	skipReason string
	err        error
}

func (le *LogbookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		le.width = msg.Width
		le.height = msg.Height
		le.buildTable()

	case editorMsg:
		if msg.err != nil {
			// error handled by caller via toast
		}
		if msg.deleted != 0 || msg.saved != 0 || msg.purged || msg.wlCall != "" {
			le.mode = edModeList
			le.needsReload = true
		}
		if msg.normalized > 0 {
			// Normalization done, now upload all unsent QSOs (skip invalid).
			var unsent []qso.QSO
			for _, q := range le.qsos {
				if q.WavelogUploaded != "yes" {
					if q.Band == "" || q.Mode == "" || q.QSODate == "" {
						continue
					}
					unsent = append(unsent, q)
				}
			}
			return le, le.uploadBatch(unsent)
		}

	case tea.KeyPressMsg:
		k := msg.String()

		// Confirm modes — route keys to the dialog with left/right navigation.
		if le.isConfirmMode() && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d := updated.(DialogModel)
			*le.dialog = d
			if d.Done() {
				if d.Result.Confirmed {
					return le, le.doConfirm()
				}
				le.dialog = nil
				le.mode = edModeList
			}
			return le, nil
		}

		if le.mode == edModeEdit {
			switch k {
			case "ctrl+s":
				return le, le.doSave()
			case "w":
				return le, le.doUploadToWavelog()
			case "esc", "f6":
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

		// modeList — table handles all navigation
		switch k {
		case "f6", "esc":
			le.done = true
		case "up", "down", "left", "right", "pgup", "pgdown", "home", "end", "k", "j", "h", "l":
			var cmd tea.Cmd
			le.table, cmd = le.table.Update(msg)
			return le, cmd
		case "delete":
			if len(le.qsos) > 0 {
				le.mode = edModeConfirmDelete
			}
		case "w":
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
	if le.dialog == nil {
		return nil
	}
	val := le.dialog.Result.Value
	le.dialog = nil
	switch val {
	case "wlsend":
		le.mode = edModeList
		return le.doBatchUpload()
	case "purge":
		le.mode = edModeList
		applog.Warn("LogbookEditor: purging all QSOs")
		return func() tea.Msg {
			err := store.PurgeQSOs(le.db)
			if err != nil {
				applog.Error("LogbookEditor: purge failed", "error", err.Error())
			} else {
				applog.Info("LogbookEditor: all QSOs purged")
			}
			return editorMsg{purged: true, err: err}
		}
	case "delete":
		q := le.qsos[le.table.Cursor()]
		call := q.Call
		date := formatDate(q.QSODate)
		id := q.ID
		le.mode = edModeList
		applog.Info("LogbookEditor: deleting QSO", "id", id, "call", call, "date", date)
		return func() tea.Msg {
			err := store.DeleteQSO(le.db, id)
			if err != nil {
				applog.Error("LogbookEditor: delete failed", "id", id, "call", call, "error", err.Error())
			} else {
				applog.Info("LogbookEditor: QSO deleted", "id", id, "call", call)
			}
			return editorMsg{deleted: id, delCall: call, delDate: date, err: err}
		}
	}
	le.mode = edModeList
	return nil
}

func (le *LogbookEditor) doSave() tea.Cmd {
	q := le.readEditForm()
	call := q.Call
	date := formatDate(q.QSODate)
	id := q.ID
	applog.Info("LogbookEditor: saving QSO", "id", id, "call", call, "date", date)
	return func() tea.Msg {
		err := store.UpdateQSO(le.db, q)
		if err != nil {
			applog.Error("LogbookEditor: save failed", "id", id, "call", call, "error", err.Error())
		} else {
			applog.Info("LogbookEditor: QSO saved", "id", id, "call", call)
		}
		return editorMsg{saved: id, saveCall: call, saveDate: date, err: err}
	}
}

// Upload methods moved to editor_upload.go.

func (le *LogbookEditor) View() tea.View {
	if le.done {
		return tea.NewView("")
	}
	bodyW := le.width // full terminal width for wider table
	if bodyW < 30 {
		bodyW = 30
	}

	switch le.mode {
	case edModeConfirmDelete:
		if le.dialog == nil {
			q := le.qsos[le.table.Cursor()]
			d := NewDialog("Delete QSO", q.Call+" from "+formatDate(q.QSODate),
				DangerOption("Delete", "delete"),
				Option{Label: "Cancel", Value: "cancel"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeConfirmPurge:
		if le.dialog == nil {
			d := NewDialog("Purge Logbook", "All QSOs will be permanently deleted.",
				DangerOption("Purge", "purge"),
				Option{Label: "Cancel", Value: "cancel"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeConfirmWLSend:
		if le.dialog == nil {
			unsent := 0
			for _, q := range le.qsos {
				if q.WavelogUploaded != "yes" {
					unsent++
				}
			}
			d := NewDialog("Send to Wavelog", fmt.Sprintf("%d unsent QSOs", unsent),
				Option{Label: "Send", Value: "wlsend"},
				Option{Label: "Cancel", Value: "cancel"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeConfirmNormalize:
		return tea.NewView(le.viewNormalizeConfirm(bodyW))
	case edModeEdit:
		contentH := contentHeight(le.height)
		if contentH < 10 {
			contentH = 10
		}
		return tea.NewView(le.viewEdit(bodyW, contentH))
	default:
		if !le.built && len(le.qsos) > 0 {
			le.buildTable()
		}
		contentH := contentHeight(le.height)
		return tea.NewView(lipgloss.NewStyle().MaxWidth(bodyW).MaxHeight(contentH).Render(
			S.RecentQSOsBox.Width(bodyW).Render(
				lipgloss.NewStyle().MaxWidth(bodyW - 2).MaxHeight(contentH - 2).Render(le.table.View()),
			),
		))
	}
}

// viewWithDialog renders the list view with the confirm dialog composited on top.
func (le *LogbookEditor) viewWithDialog(bodyW int) string {
	// Build the base list view
	if !le.built && len(le.qsos) > 0 {
		le.buildTable()
	}
	contentH := contentHeight(le.height)
	if contentH < 5 {
		contentH = 5
	}
	body := lipgloss.NewStyle().MaxWidth(bodyW).MaxHeight(contentH).Render(
		S.RecentQSOsBox.Width(bodyW).Render(
			lipgloss.NewStyle().MaxWidth(bodyW - 2).MaxHeight(contentH - 2).Render(le.table.View()),
		),
	)
	if le.dialog != nil {
		return RenderDialogOverlay(body, *le.dialog, bodyW, le.height)
	}
	return body
}

func (le *LogbookEditor) buildTable() {
	w := le.width
	if w < 40 {
		w = 80
	}
	h := le.height
	if h < 10 {
		h = 24
	}
	// Dynamic viewport: fill available vertical space. Terminal height minus
	// status(1), tabs(1), help(1), border(2), spacer(1) = h-6.
	tableH := h - 7
	if tableH < 5 {
		tableH = 5
	}
	// Select the widest tier that fits within bodyW.
	bodyW := w - 4
	if bodyW < 20 {
		bodyW = 20
	}
	var names []string
	for _, t := range editorColTiers {
		total := 0
		for _, n := range t.names {
			total += editorColWidths[n]
		}
		total += len(t.names) - 1
		if total <= bodyW {
			names = t.names
		}
	}

	// Build columns from selected tier.
	var cols []table.Column
	minTotal := 0
	for _, n := range names {
		w := editorColWidths[n]
		minTotal += w
		cols = append(cols, table.Column{Title: n, Width: w})
	}
	gaps := len(cols) - 1
	extra := bodyW - gaps - minTotal
	if extra > 0 {
		dist := 0
		for i := range cols {
			var share int
			switch cols[i].Title {
			case "Comment":
				share = extra * 5 / 10
			case "Name":
				share = extra * 2 / 10
			case "QTH":
				share = extra / 10
			case "Call":
				share = extra / 10
			}
			cols[i].Width += share
			dist += share
		}
		if leftover := extra - dist; leftover > 0 {
			cols[len(cols)-1].Width += leftover
		}
	}

	// Rebuild rows with only the selected columns.
	var trimmedRows []table.Row
	for _, q := range le.qsos {
		var row table.Row
		for _, n := range names {
			row = append(row, editorColValue(n, &q))
		}
		trimmedRows = append(trimmedRows, row)
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(trimmedRows),
		table.WithFocused(true),
		table.WithHeight(tableH),
		table.WithWidth(bodyW),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(false).Foreground(P.Text).Background(P.Surface)
	s.Cell = s.Cell.Background(P.Surface)
	s.Selected = s.Selected.Foreground(P.SelectedFg).Background(P.SelectedBg).Bold(false)
	t.SetStyles(s)
	le.table = t
	le.built = true
}

func (le *LogbookEditor) viewConfirm(action string, bodyW int) string {
	// Centered modal matching the quit‑dialog style.
	modalW := bodyW - 12
	if modalW > 56 {
		modalW = 56
	}
	if modalW < 30 {
		modalW = 30
	}
	innerW := modalW - 4 // inside border + padding

	title := S.ConfirmTitle.Width(innerW).Align(lipgloss.Center).Render(action)
	msg := S.ConfirmMsg.Width(innerW).Align(lipgloss.Center).Render("Are you sure?")
	yesBtn := S.ConfirmDanger.Width(innerW).Align(lipgloss.Center).Render(" Y = yes ")
	noBtn := S.ConfirmBtnDim.Width(innerW).Align(lipgloss.Center).Render(" any other key = cancel ")
	hint := S.ConfirmHelp.Width(innerW).Align(lipgloss.Center).Render("Press Y to confirm, any other key to go back")

	modal := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		msg,
		"",
		yesBtn,
		"",
		noBtn,
		"",
		hint,
	)

	return lipgloss.Place(bodyW, lipgloss.Height(modal)+4,
		lipgloss.Center, lipgloss.Center,
		S.ConfirmBox.Width(modalW).Render(modal),
	)
}

func (le *LogbookEditor) viewNormalizeConfirm(bodyW int) string {
	return S.ConfirmBox.Width(bodyW).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			S.ConfirmTitle.Render(fmt.Sprintf("Normalize %d QSOs", len(le.mismatchQSOs))),
			"",
			S.ConfirmMsg.Render(fmt.Sprintf("%d unsent QSOs will be normalised.", len(le.mismatchQSOs))),
			S.ConfirmHelp.Render("y = yes  ·  any other key = cancel"),
		),
	)
}

func (le *LogbookEditor) viewEdit(bodyW int, contentH int) string {
	header := S.Title.Render("Edit QSO")

	// Form fields in a bordered box.
	innerW := bodyW - 4
	if innerW < 20 {
		innerW = 20
	}
	colW := innerW / 2
	if colW < 28 {
		colW = innerW
	}
	half := (qefCount + 1) / 2

	var lines []string
	for i := qsoEditField(0); i < half; i++ {
		left := le.renderEditField(i, colW)
		rightIdx := i + half
		if rightIdx < qefCount {
			right := le.renderEditField(rightIdx, colW)
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
		} else {
			lines = append(lines, left)
		}
	}
	formBox := S.QSOFormBox.Width(bodyW).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	editH := lipgloss.Height(header) + lipgloss.Height(formBox)
	fillerH := contentH - editH
	if fillerH < 0 {
		fillerH = 0
	}
	filler := lipgloss.NewStyle().Height(fillerH).Render("")

	return lipgloss.JoinVertical(lipgloss.Left, header, formBox, filler)
}

func (le *LogbookEditor) renderEditField(f qsoEditField, colW int) string {
	label := qefLabels[f]
	val := le.fields[f].View()
	focused := f == le.focus

	lbl := LabelStyle.Render(fit(label, 13))
	if focused {
		lbl = CursorStyle.Render("> " + fit(label, 13))
	}
	valStyle := InputStyle
	if f == qefWLStatus {
		valStyle = DimStyle
	}
	return lipgloss.NewStyle().Width(colW).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			lbl,
			lipgloss.NewStyle().Width(1).Render(" "),
			valStyle.Render(val),
		),
	)
}
