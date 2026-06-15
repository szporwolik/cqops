package tui

import (
	"database/sql"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

type editorMode int

const (
	edModeList editorMode = iota
	edModeConfirmDelete
	edModeConfirmPurge
	edModeConfirmWLSend
	edModeConfirmNormalize
	edModeConfirmWLDownload
	edModeWLDownloading
	edModeWLDownloadResult
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
	qefWLStatus // last field, non-focusable read-only
	qefCount
)

var qefLabels = []string{
	"Call", "Date", "Time On", "Time Off", "Band", "Frequency", "Freq RX",
	"Mode", "Submode", "RST Sent", "RST Rcvd", "Grid", "Name",
	"QTH", "Country", "Comment", "Notes", "TX Power",
	"Station Call", "Operator", "My Grid", "My Rig", "My Antenna",
	"Source", "Distance km", "Bearing",
	"IOTA", "SOTA Ref", "POTA Ref", "WWFF Ref",
	"My SOTA", "My POTA", "My WWFF", "WL Upload (RO)",
}

type LogbookEditor struct {
	db                *sql.DB
	qsos              []qso.QSO
	table             table.Model
	mode              editorMode
	dialog            *DialogModel // confirm dialog with left/right navigation
	editing           *qso.QSO
	fields            [qefCount]textinput.Model
	focus             qsoEditField
	done              bool
	needsReload       bool
	built             bool
	wlSkipped         int
	wlSkipDetail      string
	width             int
	height            int
	wlURL             string
	wlKey             string
	wlStationID       string
	wlLastFetchedID   int64
	logStationOp      string
	logStationGrid    string
	mismatchQSOs      []qso.QSO
	mismatchFields    []string
	wlDownloadCount   int
	wlDownloadDupes   int
	wlDownloadErr     string

	// Batch download progress
	dlProgress int
	dlTotal    int
	dlCancel   chan struct{}
	dlMsgCh    chan editorMsg
}

// =============================================================================
// Constructor
// =============================================================================

func NewLogbookEditor(db *sql.DB, wlURL, wlKey, wlStationID string, wlLastFetchedID int64, logStationOp, logStationGrid string) *LogbookEditor {
	le := &LogbookEditor{db: db, mode: edModeList, wlURL: wlURL, wlKey: wlKey, wlStationID: wlStationID, wlLastFetchedID: wlLastFetchedID, logStationOp: logStationOp, logStationGrid: logStationGrid}
	for i := qsoEditField(0); i < qefCount; i++ {
		ti := newTextinput()
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
	// Apply Surface background to all textinput style states.
	for i := qsoEditField(0); i < qefCount; i++ {
		applyTextinputSurfaceStyle(&le.fields[i])
	}
	return le
}

func (le *LogbookEditor) Init() tea.Cmd {
	if !le.built && len(le.qsos) > 0 {
		le.buildTable()
	}
	return nil
}

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
	case edModeConfirmDelete, edModeConfirmPurge, edModeConfirmWLSend, edModeConfirmWLDownload:
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
