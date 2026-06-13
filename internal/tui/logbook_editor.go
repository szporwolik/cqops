package tui

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
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
	cursor         int
	offset         int
	mode           editorMode
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

	case tea.KeyMsg:
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
		case "ctrl+w":
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" && len(le.qsos) > 0 {
				le.mode = edModeConfirmWLSend
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
	if le.mode == edModeConfirmWLSend {
		return le.doBatchUpload()
	}
	if le.mode == edModeConfirmNormalize {
		return le.doNormalizeAndUpload()
	}
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

func (le *LogbookEditor) doBatchUpload() tea.Cmd {
	wlCall := le.wlStationCall
	logOp := le.logStationOp
	logGrid := le.logStationGrid

	// Collect unsent QSOs
	var unsent []qso.QSO
	for _, q := range le.qsos {
		if q.WavelogUploaded != "yes" {
			unsent = append(unsent, q)
		}
	}
	if len(unsent) == 0 {
		return func() tea.Msg {
			return editorMsg{wlOK: true, wlCall: "all sent", err: nil}
		}
	}

	// Detect mismatches against Wavelog station / logbook station defaults
	var mismatch []qso.QSO
	var fields []string
	hasCallMismatch := false
	hasOpMismatch := false
	hasGridMismatch := false
	for _, q := range unsent {
		callDiff := wlCall != "" && q.StationCallsign != "" && !strings.EqualFold(q.StationCallsign, wlCall)
		opDiff := logOp != "" && q.Operator != "" && !strings.EqualFold(q.Operator, logOp)
		gridDiff := logGrid != "" && q.MyGridSquare != "" && !strings.EqualFold(q.MyGridSquare, logGrid)
		if callDiff || opDiff || gridDiff {
			mismatch = append(mismatch, q)
			if callDiff {
				hasCallMismatch = true
			}
			if opDiff {
				hasOpMismatch = true
			}
			if gridDiff {
				hasGridMismatch = true
			}
		}
	}
	if hasCallMismatch {
		fields = append(fields, "callsign")
	}
	if hasOpMismatch {
		fields = append(fields, "operator")
	}
	if hasGridMismatch {
		fields = append(fields, "grid")
	}

	if len(mismatch) > 0 {
		le.mismatchQSOs = mismatch
		le.mismatchFields = fields
		le.mode = edModeConfirmNormalize
		return nil
	}

	return le.uploadBatch(unsent)
}

func (le *LogbookEditor) doNormalizeAndUpload() tea.Cmd {
	db := le.db
	mismatch := le.mismatchQSOs
	wlCall := le.wlStationCall
	logOp := le.logStationOp
	logGrid := le.logStationGrid

	// Collect all unsent QSOs (some may not be mismatched but still unsent)
	var unsent []qso.QSO
	for _, q := range le.qsos {
		if q.WavelogUploaded != "yes" {
			unsent = append(unsent, q)
		}
	}

	// Build list of IDs to normalize
	var normIDs []int64
	for _, q := range mismatch {
		normIDs = append(normIDs, q.ID)
	}

	applog.InfoDetail("Wavelog: normalizing station fields", fmt.Sprintf("count=%d call=%s op=%s grid=%s", len(normIDs), wlCall, logOp, logGrid))

	return func() tea.Msg {
		if err := store.NormalizeStationFields(db, normIDs, wlCall, logOp, logGrid); err != nil {
			applog.Error("Wavelog: normalization failed", "error", err)
			return editorMsg{wlOK: false, err: fmt.Errorf("normalize: %w", err)}
		}
		// Also update in-memory QSO list so the list view reflects changes
		for i := range le.qsos {
			for _, mid := range normIDs {
				if le.qsos[i].ID == mid {
					le.qsos[i].StationCallsign = wlCall
					le.qsos[i].Operator = logOp
					le.qsos[i].MyGridSquare = logGrid
					break
				}
			}
		}
		return editorMsg{normalized: len(normIDs)}
	}
}

func (le *LogbookEditor) uploadBatch(unsent []qso.QSO) tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	db := le.db
	wlCall := le.wlStationCall

	// Build batch ADIF (override station callsign to match Wavelog station)
	var adifStr string
	for _, q := range unsent {
		adifStr += q.ToADIFWithStation(wlCall)
	}

	applog.InfoDetail("Wavelog: batch upload", fmt.Sprintf("count=%d", len(unsent)))

	return func() tea.Msg {
		result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
		if err != nil {
			// If the error contains duplicates, fall back to individual uploads
			// so that duplicates don't block new QSOs.
			errStr := err.Error()
			if strings.Contains(errStr, "Duplicate for") {
				applog.Warn("Wavelog: batch had duplicates, falling back to individual uploads", "count", len(unsent))
				return le.uploadIndividual(unsent)()
			}
			applog.Error("Wavelog: batch upload failed", "count", len(unsent), "error", err)
			return editorMsg{wlOK: false, err: err, wlCall: fmt.Sprintf("%d QSOs", len(unsent))}
		}
		if result != nil && result.AllDuplicates {
			for _, q := range unsent {
				store.UpdateWavelogStatus(db, q.ID, "yes")
			}
			applog.InfoDetail("Wavelog: batch already present (duplicates)", fmt.Sprintf("count=%d", len(unsent)))
			return editorMsg{wlQSOID: unsent[0].ID, wlOK: true, wlCall: fmt.Sprintf("%d QSOs (already on Wavelog)", len(unsent))}
		}
		for _, q := range unsent {
			store.UpdateWavelogStatus(db, q.ID, "yes")
		}
		applog.InfoDetail("Wavelog: batch upload OK", fmt.Sprintf("count=%d", len(unsent)))
		return editorMsg{wlQSOID: unsent[0].ID, wlOK: true, wlCall: fmt.Sprintf("%d QSOs", len(unsent))}
	}
}

// uploadIndividual sends each unsent QSO one at a time, silently handling
// duplicates. Used as fallback when a batch upload encounters mixed results.
func (le *LogbookEditor) uploadIndividual(unsent []qso.QSO) tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	db := le.db
	wlCall := le.wlStationCall

	return func() tea.Msg {
		okCount := 0
		dupCount := 0
		failCount := 0
		var lastErr error

		for _, q := range unsent {
			adifStr := q.ToADIFWithStation(wlCall)
			result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
			if err != nil {
				applog.Warn("Wavelog: individual upload failed", "qso_id", q.ID, "call", q.Call, "error", err)
				store.UpdateWavelogStatus(db, q.ID, "no")
				failCount++
				lastErr = err
				continue
			}
			if result != nil && result.AllDuplicates {
				applog.Debug("Wavelog: duplicate (silent)", "qso_id", q.ID, "call", q.Call)
				store.UpdateWavelogStatus(db, q.ID, "yes")
				dupCount++
				continue
			}
			store.UpdateWavelogStatus(db, q.ID, "yes")
			okCount++
		}

		applog.InfoDetail("Wavelog: individual upload complete",
			fmt.Sprintf("ok=%d dup=%d fail=%d", okCount, dupCount, failCount))

		if failCount > 0 && okCount == 0 && dupCount == 0 {
			return editorMsg{wlOK: false, err: lastErr, wlCall: fmt.Sprintf("%d failed", failCount)}
		}
		msg := fmt.Sprintf("%d ok", okCount)
		if dupCount > 0 {
			msg += fmt.Sprintf(", %d already present", dupCount)
		}
		if failCount > 0 {
			msg += fmt.Sprintf(", %d failed", failCount)
		}
		return editorMsg{wlQSOID: unsent[0].ID, wlOK: true, wlCall: msg}
	}
}

func (le *LogbookEditor) doUploadToWavelog() tea.Cmd {
	if le.wlURL == "" || le.wlKey == "" || le.wlStationID == "" {
		return func() tea.Msg {
			return editorMsg{wlOK: false, err: fmt.Errorf("Wavelog not configured")}
		}
	}
	q := le.readEditForm()
	adifStr := q.ToADIFWithStation(le.wlStationCall)
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	qID := q.ID
	call := q.Call

	return func() tea.Msg {
		applog.InfoDetail("Wavelog: uploading from editor", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
		if err != nil {
			applog.Error("Wavelog: editor upload failed", "qso_id", qID, "call", call, "error", err)
			store.UpdateWavelogStatus(le.db, qID, "no")
			return editorMsg{wlQSOID: qID, wlCall: call, wlOK: false, err: err}
		}
		if result != nil && result.AllDuplicates {
			store.UpdateWavelogStatus(le.db, qID, "yes")
			applog.InfoDetail("Wavelog: editor QSO already present (duplicate)", fmt.Sprintf("qso_id=%d call=%s", qID, call))
			return editorMsg{wlQSOID: qID, wlCall: call, wlOK: true}
		}
		store.UpdateWavelogStatus(le.db, qID, "yes")
		applog.InfoDetail("Wavelog: editor upload OK", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		return editorMsg{wlQSOID: qID, wlCall: call, wlOK: true}
	}
}

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
	case edModeConfirmWLSend:
		unsent := 0
		for _, q := range le.qsos {
			if q.WavelogUploaded != "yes" {
				unsent++
			}
		}
		return le.viewConfirm(fmt.Sprintf("Send %d unsent QSOs to Wavelog", unsent), bodyW)
	case edModeConfirmNormalize:
		return le.viewNormalizeConfirm(bodyW)
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
	b.WriteString("\n")

	if len(le.qsos) == 0 {
		b.WriteString("\n  No QSOs in logbook.")
		return b.String()
	}

	cols := selectEditorCols(bodyW - 2) // -2 for the "  " prefix
	if len(cols) == 0 {
		return b.String()
	}

	// Build header
	var headerParts []string
	for _, c := range cols {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", c.width, c.header))
	}
	headerLine := headerStyle.Render("  " + strings.Join(headerParts, " "))
	b.WriteString("\n" + headerLine + "\n\n")

	// Build rows
	for i := le.offset; i < le.offset+maxRows && i < len(le.qsos); i++ {
		q := le.qsos[i]
		prefix := "  "
		if i == le.cursor {
			prefix = CursorStyle.Render("> ")
		}
		var vals []string
		for _, c := range cols {
			v := c.value(&q)
			if v == "" {
				v = "—"
			}
			v = trunc(v, c.width)
			vals = append(vals, fmt.Sprintf("%-*s", c.width, v))
		}
		line := prefix + strings.Join(vals, " ")
		if i == le.cursor {
			line = InputStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

type editorCol struct {
	header   string
	minWidth int
	width    int // computed at selection time
	grow     bool
	priority int // higher = more important
	value    func(q *qso.QSO) string
}

var editorAllCols = []editorCol{
	{"ID", 3, 0, false, 100, func(q *qso.QSO) string { return fmt.Sprintf("%d", q.ID) }},
	{"Date", 10, 0, false, 95, func(q *qso.QSO) string { return formatDate(q.QSODate) }},
	{"Time", 8, 0, false, 90, func(q *qso.QSO) string { return formatTime(q.TimeOn) }},
	{"Call", 7, 0, true, 85, func(q *qso.QSO) string { return q.Call }},
	{"Band", 5, 0, false, 80, func(q *qso.QSO) string {
		b := qso.NormalizeBand(q.Band)
		if b == "" && q.Freq > 0 {
			b = fmt.Sprintf("%.0f", q.Freq)
		}
		return b
	}},
	{"Mode", 5, 0, false, 75, func(q *qso.QSO) string { return q.Mode }},
	{"RSTs", 4, 0, false, 70, func(q *qso.QSO) string { return q.RSTSent }},
	{"RSTr", 4, 0, false, 65, func(q *qso.QSO) string { return q.RSTRcvd }},
	{"Grid", 6, 0, false, 60, func(q *qso.QSO) string { return q.GridSquare }},
	{"DXCC", 6, 0, true, 55, func(q *qso.QSO) string { return q.Country }},
	{"Name", 7, 0, true, 50, func(q *qso.QSO) string { return q.Name }},
	{"QTH", 8, 0, true, 45, func(q *qso.QSO) string { return q.QTH }},
	{"WL", 8, 0, false, 42, func(q *qso.QSO) string {
		switch q.WavelogUploaded {
		case "yes":
			return "yes"
		case "no":
			return "no"
		default:
			return ""
		}
	}},
	{"Comment", 10, 0, true, 40, func(q *qso.QSO) string { return q.Comment }},
	{"Rig", 8, 0, true, 5, func(q *qso.QSO) string { return q.MyRig }},
	{"Sub", 4, 0, false, 25, func(q *qso.QSO) string { return q.Submode }},
	{"SOTA", 8, 0, false, 25, func(q *qso.QSO) string { return q.SOTARef }},
	{"POTA", 8, 0, false, 20, func(q *qso.QSO) string { return q.POTARef }},
	{"IOTA", 7, 0, false, 15, func(q *qso.QSO) string { return q.IOTA }},
	{"WWFF", 9, 0, false, 10, func(q *qso.QSO) string { return q.WWFFRef }},
}

func selectEditorCols(availW int) []editorCol {
	// Sort by priority descending
	sorted := make([]editorCol, len(editorAllCols))
	copy(sorted, editorAllCols)
	// Bubble sort by priority (stable, small N)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].priority > sorted[i].priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Greedy: pick columns by priority until we run out of width
	var cols []editorCol
	usedW := 0
	for _, c := range sorted {
		needed := c.minWidth
		if len(cols) > 0 {
			needed++ // space between columns
		}
		if usedW+needed <= availW {
			cols = append(cols, c)
			usedW += needed
		}
	}

	if len(cols) == 0 {
		return nil
	}

	// Distribute remaining width to growable columns
	extra := availW - usedW
	if extra > 0 {
		growCount := 0
		for _, c := range cols {
			if c.grow {
				growCount++
			}
		}
		if growCount > 0 {
			perGrow := extra / growCount
			for i := range cols {
				if cols[i].grow {
					cols[i].width = cols[i].minWidth + perGrow
					extra -= perGrow
				} else {
					cols[i].width = cols[i].minWidth
				}
			}
			// Give leftover to first grow column
			for i := range cols {
				if cols[i].grow && extra > 0 {
					cols[i].width += extra
					break
				}
			}
		} else {
			for i := range cols {
				cols[i].width = cols[i].minWidth
			}
		}
	} else {
		for i := range cols {
			cols[i].width = cols[i].minWidth
		}
	}

	// Sort back by priority descending for display order
	for i := 0; i < len(cols)-1; i++ {
		for j := i + 1; j < len(cols); j++ {
			if cols[j].priority > cols[i].priority {
				cols[i], cols[j] = cols[j], cols[i]
			}
		}
	}

	return cols
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
	valStyle := InputStyle
	if f == qefWLStatus {
		valStyle = DimStyle // read-only indicator
	}
	field := prefix + lbl + " " + valStyle.Render(val)
	for lipgloss.Width(field) < colW {
		field += " "
	}
	return field
}
