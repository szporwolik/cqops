package tui

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

type editorMode int

const (
	edModeList editorMode = iota
	edModeConfirmDelete
	edModeConfirmPurge
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
	qefMyRig
	qefMyAntenna
	qefSource
	qefDistance
	qefBearing
	qefCount
)

var qefLabels = []string{
	"Call", "Date", "Time On", "Time Off", "Band", "Frequency",
	"Mode", "Submode", "RST Sent", "RST Rcvd", "Grid", "Name",
	"QTH", "Country", "Comment", "Notes", "TX Power",
	"Station Call", "Operator", "My Grid", "My Rig", "My Antenna",
	"Source", "Distance km", "Bearing",
}

type LogbookEditor struct {
	db          *sql.DB
	qsos        []qso.QSO
	cursor      int
	offset      int
	mode        editorMode
	editing     *qso.QSO
	fields      [qefCount]textinput.Model
	focus       qsoEditField
	done        bool
	needsReload bool
	width       int
	height      int
}

func NewLogbookEditor(db *sql.DB) *LogbookEditor {
	le := &LogbookEditor{db: db, mode: edModeList}
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
		case qefFreq:
			ti.CharLimit = 16
		case qefMode:
			ti.CharLimit = 12
		case qefSubmode:
			ti.CharLimit = 16
		case qefSource:
			ti.CharLimit = 10
		case qefDistance, qefBearing:
			ti.CharLimit = 10
		}
		le.fields[i] = ti
	}
	return le
}

func (le *LogbookEditor) Init() tea.Cmd { return nil }

func (le *LogbookEditor) SetQSOS(qsos []qso.QSO) { le.qsos = qsos }

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
		Freq: gf(qefFreq), Mode: g(qefMode), Submode: g(qefSubmode),
		RSTSent: g(qefRSTSent), RSTRcvd: g(qefRSTRcvd),
		GridSquare: g(qefGrid), Name: g(qefName), QTH: g(qefQTH),
		Country: g(qefCountry), Comment: g(qefComment), Notes: g(qefNotes),
		TXPower: g(qefTXPower), StationCallsign: g(qefStationCall),
		Operator: g(qefOperator), MyGridSquare: g(qefMyGrid),
		MyRig: g(qefMyRig), MyAntenna: g(qefMyAntenna), Source: g(qefSource),
		Distance: gf(qefDistance), Bearing: gf(qefBearing),
		CreatedAt: le.editing.CreatedAt,
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
	deleted int64
	saved   int64
	purged  bool
	err     error
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
		if msg.deleted != 0 || msg.saved != 0 || msg.purged {
			le.mode = edModeList
			le.needsReload = true
		}

	case tea.KeyMsg:
		k := msg.String()

		if le.mode == edModeConfirmDelete || le.mode == edModeConfirmPurge {
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
			case "esc", "f5":
				le.mode = edModeList
			case "tab", "down":
				le.nextField()
			case "shift+tab", "up":
				le.prevField()
			default:
				le.fields[le.focus], _ = le.fields[le.focus].Update(msg)
			}
			return le, nil
		}

		// modeList
		switch k {
		case "f5", "esc":
			le.done = true
		case "up", "k":
			if le.cursor > 0 {
				le.cursor--
			}
		case "down", "j":
			if le.cursor < len(le.qsos)-1 {
				le.cursor++
			}
		case "delete":
			if len(le.qsos) > 0 {
				le.mode = edModeConfirmDelete
			}
		case "e", "enter":
			if len(le.qsos) > 0 {
				q := le.qsos[le.cursor]
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
	if le.mode == edModeConfirmDelete {
		id := le.qsos[le.cursor].ID
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

func (le *LogbookEditor) FooterText() string {
	switch le.mode {
	case edModeConfirmDelete:
		return "Delete this QSO? (y/N)"
	case edModeConfirmPurge:
		return "Purge ALL QSOs? This cannot be undone. (y/N)"
	case edModeEdit:
		return "Ctrl+S to save  Tab/Up/Dn to navigate  Esc to discard"
	default:
		return "Up/Dn scroll  Enter/e edit  Del delete  p purge  F5/Esc close"
	}
}

func (le *LogbookEditor) View() string {
	if le.done {
		return ""
	}
	w := le.width
	if w < 40 {
		w = 80
	}
	bodyW := w - 2

	switch le.mode {
	case edModeConfirmDelete:
		return le.viewConfirm("Delete QSO", bodyW)
	case edModeConfirmPurge:
		return le.viewConfirm("Purge Logbook", bodyW)
	case edModeEdit:
		return le.viewEdit(bodyW)
	default:
		return le.viewList(bodyW)
	}
}

func (le *LogbookEditor) viewConfirm(action string, bodyW int) string {
	var b strings.Builder
	b.WriteString(section("── "+action+" ", bodyW))
	b.WriteString("\n\n")
	b.WriteString("  Are you sure? (y/N)")
	return b.String()
}

func (le *LogbookEditor) viewList(bodyW int) string {
	h := le.height
	if h < 10 {
		h = 24
	}
	maxRows := h - 8
	if maxRows < 3 {
		maxRows = 3
	}

	if le.cursor < le.offset {
		le.offset = le.cursor
	}
	if le.cursor >= le.offset+maxRows {
		le.offset = le.cursor - maxRows + 1
	}
	if le.offset < 0 {
		le.offset = 0
	}

	var b strings.Builder
	b.WriteString(section("── Logbook Editor ", bodyW))
	b.WriteString("\n\n")

	if len(le.qsos) == 0 {
		b.WriteString("  No QSOs in logbook.")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  %-4s %-10s %-8s %-10s %-5s %-4s %-4s %-4s %-6s %s",
		"ID", "Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "Grid", "Comment"))
	b.WriteString("\n\n")

	for i := le.offset; i < le.offset+maxRows && i < len(le.qsos); i++ {
		q := le.qsos[i]
		prefix := "  "
		if i == le.cursor {
			prefix = CursorStyle.Render("> ")
		}
		band := q.Band
		if band == "" && q.Freq > 0 {
			band = fmt.Sprintf("%.0f", q.Freq)
		}
		line := fmt.Sprintf("%s%-4d %-10s %-8s %-10s %-5s %-4s %-4s %-4s %-6s %s",
			prefix, q.ID, q.QSODate, q.TimeOn,
			truncate(q.Call, 10), truncate(band, 5), truncate(q.Mode, 4),
			truncate(q.RSTSent, 4), truncate(q.RSTRcvd, 4),
			truncate(q.GridSquare, 6), truncate(q.Comment, 20))
		if i == le.cursor {
			line = InputStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
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
			b.WriteString(left + "  " + right + "\n")
		} else {
			b.WriteString(left + "\n")
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
	field := prefix + lbl + " " + InputStyle.Render(val)
	for lipgloss.Width(field) < colW {
		field += " "
	}
	return field
}
