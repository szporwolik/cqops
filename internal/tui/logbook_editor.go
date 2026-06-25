package tui

import (
	"database/sql"
	"os"

	"charm.land/bubbles/v2/filepicker"
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
	edModeConfirmWLDownload
	edModeWLDownloading
	edModeWLDownloadResult
	edModeEdit
	edModeExport
	edModeExporting
	edModeExportResult
	edModeImport
	edModeImporting
	edModeImportResult
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
	qefMyCQZone
	qefMyITUZone
	qefMyDXCC
	qefMySIG
	qefMySIGInfo
	qefDistance
	qefBearing
	qefIOTA
	qefSOTA
	qefPOTA
	qefWWFF
	qefSIG
	qefSIGInfo
	qefMySOTA
	qefMyPOTA
	qefMyWWFF
	qefCQZone
	qefITUZone
	qefExchSent
	qefExchRcvd
	qefSTX
	qefSRX
	qefSTXString
	qefSRXString
	qefContestID
	qefWLStatus // non-focusable read-only
	qefSource   // non-focusable read-only — last real field
	qefCount
)

var qefLabels = []string{
	"Call", "Date", "Time On", "Time Off", "Band", "Frequency", "Freq RX",
	"Mode", "Submode", "RST Sent", "RST Rcvd", "Grid", "Name",
	"QTH", "Country", "Comment", "Notes", "TX Power",
	"Station Call", "Operator", "My Grid", "My Rig", "My Antenna",
	"My CQ Zone", "My ITU Zone", "My DXCC",
	"My SIG", "My SIG Info",
	"Distance km", "Bearing",
	"IOTA", "SOTA Ref", "POTA Ref", "WWFF Ref", "SIG", "SIG Info",
	"My SOTA", "My POTA", "My WWFF",
	"CQ Zone", "ITU Zone",
	"Exch Sent", "Exch Rcvd", "STX", "SRX", "STX String", "SRX String", "Contest ID",
	"WL Upload (RO)",
	"Source (RO)",
}

type LogbookEditor struct {
	db               *sql.DB
	qsos             []qso.QSO
	table            table.Model
	mode             editorMode
	dialog           *DialogModel // confirm dialog with left/right navigation
	editing          *qso.QSO
	fields           [qefCount]textinput.Model
	focus            qsoEditField
	done             bool
	needsReload      bool
	built            bool
	wlSkipped        int
	wlSkipDetail     string
	wlUnsentCount    int // cached unsent count from full DB, used by confirm dialog
	width            int
	height           int
	wlURL            string
	wlKey            string
	wlStationID      string
	wlLastFetchedID  int64
	logStationOp     string
	logStationGrid   string
	logStationCall   string // station callsign, for export filename
	contestID        string // active contest hash for filtering, "" = no filter
	contestName      string // display name for the contest info line
	contestAdifID    string // ADIF Contest-ID for the contest info line
	mismatchQSOs     []qso.QSO
	mismatchFields   []string
	wlDownloadCount  int
	wlDownloadDupes  int
	wlDownloadFailed int
	wlDownloadErr    string
	Offline          bool // when true, Wavelog upload/download is blocked

	// Pagination — only the current page is loaded from DB.
	currentPage int
	totalCount  int
	pageSize    int

	// Batch download/import progress (shared infrastructure, one active at a time).
	dlActive   bool // true while download goroutine is running
	dlProgress int
	dlTotal    int
	dlCurrent  int // QSOs processed so far
	dlCancel   chan struct{}
	dlMsgCh    chan editorMsg

	// Cached download progress message — rebuilt only when numbers change.
	dlCachedMsg string
	dlLastCur   int
	dlLastTot   int

	// ADIF import results.
	impInserted int
	impDupes    int
	impFailed   int
	impErr      string

	// View cache — avoids rebuilding the table on every frame with large QSO sets.
	cachedView string
	cachedSig  string
	builtW     int // width at last buildTable call
	builtH     int // height at last buildTable call

	// Cached spacer style — avoids lipgloss.NewStyle() on every cache miss.
	cachedSpacerStyle  lipgloss.Style
	cachedSpacerStyleW int

	// Cached dialog table style — avoids lipgloss.NewStyle() every frame.
	cachedTablePartStyle lipgloss.Style
	cachedTablePartW     int
	cachedTablePartH     int

	// Cached edit form column style — rebuilt only on colW change.
	cachedEditColStyle lipgloss.Style
	cachedEditColW     int

	// File export.
	filePicker filepicker.Model
	exportPath string
}

// =============================================================================
// Constructor
// =============================================================================

func NewLogbookEditor(db *sql.DB, wlURL, wlKey, wlStationID string, wlLastFetchedID int64, logStationOp, logStationGrid, logStationCall string) *LogbookEditor {
	le := &LogbookEditor{db: db, mode: edModeList, wlURL: wlURL, wlKey: wlKey, wlStationID: wlStationID, wlLastFetchedID: wlLastFetchedID, logStationOp: logStationOp, logStationGrid: logStationGrid, logStationCall: logStationCall}
	le.filePicker = filepicker.New()
	le.filePicker.FileAllowed = false
	le.filePicker.DirAllowed = true
	le.filePicker.AutoHeight = false
	le.filePicker.ShowHidden = false
	if home, err := os.UserHomeDir(); err == nil {
		le.filePicker.CurrentDirectory = home
	}
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
		case qefMyCQZone, qefMyITUZone, qefMyDXCC:
			ti.CharLimit = 6
		case qefWLStatus:
			ti.CharLimit = 8
		case qefExchSent, qefExchRcvd, qefSTXString, qefSRXString:
			ti.CharLimit = 40
		case qefSTX, qefSRX:
			ti.CharLimit = 8
		case qefContestID:
			ti.CharLimit = 30
		}
		le.fields[i] = ti
	}
	return le
}

func (le *LogbookEditor) Init() tea.Cmd {
	if !le.built && len(le.qsos) > 0 {
		le.buildTable()
	}
	return nil
}

func (le *LogbookEditor) SetQSOS(qsos []qso.QSO) {
	le.qsos = qsos
	le.cachedSig = "" // invalidate view cache
	le.buildTable()
}

// SetContestID sets the active contest filter for the editor.
// Pass "" to clear the filter and show all QSOs.
func (le *LogbookEditor) SetContestID(id, name, adifID string) {
	if le.contestID != id {
		le.contestID = id
		le.contestName = name
		le.contestAdifID = adifID
		le.cachedSig = ""
		le.currentPage = 1
		le.loadPage()
	}
}

// loadPage fetches the current page of QSOs from the database.
func (le *LogbookEditor) loadPage() {
	if le.db == nil {
		return
	}
	// Determine page size from current terminal height.
	h := le.height
	if h < 10 {
		h = 24
	}
	// pageSize = data rows visible = tableH - 1 header row.
	le.pageSize = h - 7
	if le.pageSize < 5 {
		le.pageSize = 5
	}

	// Fetch page data and total count in a single query via COUNT(*) OVER().
	totalPages := le.totalPages()
	if totalPages < 1 {
		totalPages = 1
	}
	if le.currentPage < 1 {
		le.currentPage = 1
	}
	if le.currentPage > totalPages {
		le.currentPage = totalPages
	}

	offset := (le.currentPage - 1) * le.pageSize
	qsos, total, err := store.ListQSOsPageWithCount(le.db, le.pageSize, offset, le.contestID)
	if err != nil {
		le.qsos = nil
	} else {
		le.qsos = qsos
		le.totalCount = total
	}
	le.cachedSig = ""
	le.buildTable()
}

// isDownloadActive returns true when a Wavelog download is in progress.
func (le *LogbookEditor) isDownloadActive() bool {
	return le.dlActive
}
func (le *LogbookEditor) totalPages() int {
	if le.pageSize < 1 || le.totalCount < 1 {
		return 1
	}
	p := le.totalCount / le.pageSize
	if le.totalCount%le.pageSize != 0 {
		p++
	}
	return p
}

// goToPage navigates to the given page (1-indexed) and reloads data.
func (le *LogbookEditor) goToPage(p int) {
	tp := le.totalPages()
	if p < 1 {
		p = 1
	}
	if p > tp {
		p = tp
	}
	if p == le.currentPage && len(le.qsos) > 0 {
		return
	}
	le.currentPage = p
	le.loadPage()
	// Reset cursor to top of the new page.
	le.table.SetCursor(0)
}

func (le *LogbookEditor) CursorPos() int {
	if le.built {
		return le.table.Cursor()
	}
	return 0
}

func (le *LogbookEditor) QSOCount() int { return len(le.qsos) }

func (le *LogbookEditor) IsEditing() bool { return le.mode == edModeEdit }

func (le *LogbookEditor) IsExporting() bool {
	return le.mode == edModeExport || le.mode == edModeExporting || le.mode == edModeExportResult
}
func (le *LogbookEditor) IsImporting() bool {
	return le.mode == edModeImport || le.mode == edModeImporting || le.mode == edModeImportResult
}

// FilePicker returns the filepicker model for external use (help suffix).
func (le *LogbookEditor) FilePicker() filepicker.Model { return le.filePicker }

func (le *LogbookEditor) isModalMode() bool {
	switch le.mode {
	case edModeConfirmDelete, edModeConfirmPurge, edModeConfirmWLSend, edModeConfirmWLDownload,
		edModeConfirmNormalize, edModeWLDownloading, edModeWLDownloadResult,
		edModeExporting, edModeExportResult,
		edModeImporting, edModeImportResult:
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
