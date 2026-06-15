package tui

import (
	"database/sql"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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
	db              *sql.DB
	qsos            []qso.QSO
	table           table.Model
	mode            editorMode
	dialog          *DialogModel // confirm dialog with left/right navigation
	editing         *qso.QSO
	fields          [qefCount]textinput.Model
	focus           qsoEditField
	done            bool
	needsReload     bool
	built           bool
	wlSkipped       int
	wlSkipDetail    string
	width           int
	height          int
	wlURL           string
	wlKey           string
	wlStationID     string
	wlLastFetchedID int64
	logStationOp    string
	logStationGrid  string
	mismatchQSOs    []qso.QSO
	mismatchFields  []string
	wlDownloadCount int
	wlDownloadDupes int
	wlDownloadErr   string

	// Pagination — only the current page is loaded from DB.
	currentPage int
	totalCount  int
	pageSize    int

	// Batch download progress
	dlProgress  int
	dlTotal     int
	dlCurrent   int // QSOs processed so far
	dlCancel    chan struct{}
	dlMsgCh     chan editorMsg
	dlSpinner   spinner.Model
	dlSpinnerOn bool // true when spinner is active (download in progress)

	// View cache — avoids rebuilding the table on every frame with large QSO sets.
	cachedView string
	cachedSig  string
	builtW     int // width at last buildTable call
	builtH     int // height at last buildTable call
}

// =============================================================================
// Constructor
// =============================================================================

func NewLogbookEditor(db *sql.DB, wlURL, wlKey, wlStationID string, wlLastFetchedID int64, logStationOp, logStationGrid string) *LogbookEditor {
	le := &LogbookEditor{db: db, mode: edModeList, wlURL: wlURL, wlKey: wlKey, wlStationID: wlStationID, wlLastFetchedID: wlLastFetchedID, logStationOp: logStationOp, logStationGrid: logStationGrid}
	le.dlSpinner = spinner.New(spinner.WithSpinner(spinner.Dot))
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

	// Refresh total count.
	counts, err := store.CountQSOs(le.db)
	if err == nil {
		le.totalCount = counts.Total
	}

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
	qsos, err := store.ListQSOsPage(le.db, le.pageSize, offset)
	if err != nil {
		le.qsos = nil
	} else {
		le.qsos = qsos
	}
	le.cachedSig = ""
	le.buildTable()
}

// isDownloadActive returns true when a Wavelog download is in progress.
func (le *LogbookEditor) isDownloadActive() bool {
	return le.mode == edModeWLDownloading
}

// spinCmd returns a spinner.Tick command when a download is active.
func (le *LogbookEditor) spinCmd() tea.Cmd {
	if le.mode == edModeWLDownloading {
		return le.dlSpinner.Tick
	}
	return nil
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

func (le *LogbookEditor) isConfirmMode() bool {
	switch le.mode {
	case edModeConfirmDelete, edModeConfirmPurge, edModeConfirmWLSend, edModeConfirmWLDownload,
		edModeConfirmNormalize, edModeWLDownloading, edModeWLDownloadResult:
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
