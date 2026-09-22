package tui

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	adif "github.com/farmergreg/adif/v5"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/version"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Editor messages and update logic
// =============================================================================

type editorMsg struct {
	deleted    int64
	delCall    string
	delDate    string
	delSyncOK  bool
	delSyncErr string
	saved      int64
	saveCall   string
	saveDate   string
	purged     bool
	wlQSOID    int64
	wlCall     string
	wlOK       bool
	wlDup      bool
	// Batch/individual upload tallies. wlSentCount counts QSOs the remote
	// accepted whose remote id was persisted locally; wlDupCount remote
	// duplicates with a persisted id; wlUnresolvedCount remotely accepted
	// but local remote-id persistence incomplete (re-offered next upload);
	// wlFailCount requests that failed outright and remain fully unsent.
	wlSentCount       int
	wlDupCount        int
	wlFailCount       int
	wlUnresolvedCount int
	normalized        int
	// norm* carry the station fields a normalize worker changed plus the
	// affected local ids. They are immutable result data: the worker never
	// touches the in-memory list, and the owner loop applies them after
	// validating the generation.
	normIDs  []int64
	normCall string
	normOp   string
	normGrid string
	gen      uint64 // editor generation for async operation results (0 = not generation-bound)
	err      error
	dlCount  int
	dlDupes  int
	dlFailed int
	dlLastID int64
	dlErr    string
	// Batch download progress
	dlProgress int
	dlTotal    int
	dlDone     bool
	dlAborted  bool
	// dlDownload marks a Wavelog download completion (as opposed to ADIF
	// import/export, which reuses the same message shape). Only download
	// completions may advance the persisted Wavelog last_fetched_id cursor.
	dlDownload bool
	// dlTransient counts insert failures deferred for retry: the persisted
	// cursor stops before the first such record so the next incremental
	// download re-fetches it.
	dlTransient int
	// dlUnresolved counts imported records whose remote identity could not
	// be resolved (identity sidecar failure). They are re-fetched on the
	// next download too — the cursor must not advance past them or the
	// remote link would be lost forever.
	dlUnresolved int
	// Simple toast from the editor.
	toastWarn string
	// Remote-copy refresh of the QSO being edited.
	wlFetchQSOID   int64 // local id the fetch was issued for
	wlFetchQSO     *wavelog.QSOData
	wlFetchErr     string
	wlFetchRev     uint64  // edit revision when the fetch started — a later value means the operator typed since
	wlFetchGen     uint64  // editor generation that issued the fetch (0 = not generation-bound)
	wlFetchDB      *sql.DB // database the fetch ran against (nil = not database-bound)
	wlFetchSession uint64  // edit session when the fetch started — reopened contacts reject older sessions
	// Wavelog PATCH result after a save of a synced QSO.
	wlSyncOK      bool
	wlSyncGone    bool
	wlSyncErr     string
	wlSyncPending bool // offline save of a synced QSO — sync deferred
	// wlSyncIncomplete marks a PATCH that reached the remote server but
	// whose local acknowledgement could not be persisted: the row stays
	// pending (durably dirty) instead of falsely reporting success.
	wlSyncIncomplete bool
	// saveSession binds a save completion to the edit session that
	// initiated it: a completion for a superseded session must not close
	// the form of a newer session.
	saveSession uint64
	// saveRev is the form revision (le.editRev) captured when the save was
	// dispatched. The form stays editable while the PATCH runs; a
	// completion must only close it when BOTH the session and the revision
	// still match — otherwise the newer unsaved input is preserved.
	saveRev uint64
	// opSession binds delete/upload/purge completions to the edit session
	// they were initiated in. A generation match alone does not mean the
	// operator is still performing the original action: the completion may
	// arrive while another contact's form is open, and must then refresh
	// the list without closing the unrelated form. Zero is a legitimate
	// session (a fresh editor), so opSessionSet marks that a real session
	// was captured — unbound/legacy messages leave it false.
	opSession    uint64
	opSessionSet bool
	// wlSyncFollowUp marks a serialized follow-up PATCH result — it must
	// never close an open form (the form closed at the original save).
	wlSyncFollowUp bool
	// syncCtx carries the originating per-contact serialization context for
	// PATCH completions: the originating database, remote endpoint, editor
	// generation and the chain database lease. The follow-up dispatch must
	// use exactly this context — never the currently active database or the
	// current editor's Wavelog credentials, which may belong to another
	// logbook after a switch.
	syncCtx *contactSyncContext
	// lbID is the originating logbook identity of the operation that
	// produced this message. Model-level side effects must persist results
	// against THIS logbook (e.g. the Wavelog download cursor or the purge
	// cursor reset), independently of which logbook or editor is currently
	// visible.
	lbID string
	// wlUp* carry the originating database/endpoint of a single upload and
	// whether the row changed while the upload was on the wire — the model
	// queues a follow-up PATCH of the latest revision in that case. When the
	// server accepted the contact but the local id write failed, the
	// completion is marked wlUpUnresolved and the model retries the id
	// attach instead of presenting it as synchronized.
	wlUpDB         *sql.DB
	wlUpURL        string
	wlUpKey        string
	wlUpSID        string
	wlUpChanged    bool
	wlUpUnresolved bool
	// wlUpRelease transfers the database lease from an editor upload worker
	// to the global completion handler: ONE lease covers the whole upload →
	// id retry → reconciliation chain, so a logbook switch while the upload
	// is on the wire can never close the originating database before the
	// follow-up chain starts. The global handler owns it and releases it
	// when the chain finishes (or immediately with no follow-up).
	wlUpRelease func()
	// wlUpSnap/wlUpRev carry the SNAPSHOT that reached the server and the
	// row revision it was taken at. The id-attach retry must use this pair,
	// never the current row: an edit made after the failed attach belongs
	// to a follow-up PATCH, not to the accepted snapshot.
	wlUpSnap qso.QSO
	wlUpRev  int64
	// wlReconcileIDs lists rows whose revision changed between upload
	// preparation (data + revision captured together) and id attach: the
	// remote copy was built from an older snapshot, so the owner loop
	// queues a PATCH of each latest revision. Batch uploads carry many
	// rows, so the single-row wlUpChanged flag is not enough.
	wlReconcileIDs []int64
	// normRelease transfers the normalization worker's database lease to
	// the post-normalize batch upload launched from its completion — the
	// upload consumes it (or releases it when no upload follows), so a
	// logbook switch during normalization cannot close the retired database
	// before the upload starts.
	normRelease func()
	// dlOpID binds download/import/export messages to the operation that
	// produced them: a result from a replaced operation must never finalize
	// (or advance) a newer one.
	dlOpID uint64
	// dlChannelClosed marks the synthesized completion from a channel close
	// that never delivered a terminal message — it must never be treated as
	// an independent success; the editor finalizes honestly instead.
	dlChannelClosed bool
}

func (le *LogbookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width != le.width || msg.Height != le.height {
			le.width = msg.Width
			le.height = msg.Height
			le.buildTable()
		}

	case uploadPrepMsg:
		return le.handleUploadPrep(msg)

	case editorMsg:
		// Accept the read: the operation's single pending read is released
		// ONLY here, after the owner loop received its result. Clearing it
		// in the reader goroutine would open a gap in which a tick could
		// dispatch a second reader that observes the channel closure and
		// overtakes the real terminal.
		if msg.dlOpID != 0 && le.dlOp != nil && msg.dlOpID == le.dlOp.id {
			le.dlOp.readPending.Store(false)
		}

		// Bind download/import/export messages to their operation: a stale
		// result from a replaced operation must never finalize or advance a
		// newer one.
		if msg.dlOpID != 0 && (le.dlOp == nil || le.dlOp.id != msg.dlOpID) {
			return le, nil
		}

		// The synthesized channel-close completion is not an independent
		// success: finalize honestly — with the captured error, or with an
		// explicit did-not-finish error — never a silent zero-count result.
		if msg.dlChannelClosed {
			if le.dlActive {
				le.dlActive = false
				le.dlOp = nil
				le.dialog = nil
				switch le.mode {
				case edModeImporting, edModeImport, edModeExporting, edModeExport:
					if strings.TrimSpace(le.impErr) == "" {
						le.impErr = "operation did not finish"
					}
					if le.mode == edModeExporting || le.mode == edModeExport {
						le.mode = edModeExportResult
					} else {
						le.mode = edModeImportResult
					}
				default:
					if strings.TrimSpace(le.wlDownloadErr) == "" {
						le.wlDownloadErr = "operation did not finish"
					}
					le.mode = edModeWLDownloadResult
				}
				le.needsReload = true
			}
			return le, nil
		}

		// A normalization completion is a normalization completion even while
		// a download is active — the progress branches below must not
		// swallow it, which would leak its transferred database lease.
		if msg.normalized > 0 {
			return le.handleNormalizeResult(msg)
		}

		// Batch download/import/export progress — only when a download is actually active.
		if le.dlActive && !msg.dlDone && msg.dlErr == "" {
			le.dlProgress = msg.dlProgress
			le.dlTotal = msg.dlTotal
			le.dlCurrent = msg.dlCount
			if le.mode == edModeImport {
				le.mode = edModeImporting
			} else if le.mode == edModeExport {
				le.mode = edModeExporting
			} else if le.mode != edModeWLDownloading && le.mode != edModeImporting && le.mode != edModeExporting {
				le.mode = edModeWLDownloading
			}
			return le, le.readDownloadMsg()
		}

		// Download/import/export error received before done signal.
		// Capture the error immediately so it's not lost before the channel closes.
		if le.dlActive && !msg.dlDone {
			if errText := strings.TrimSpace(msg.dlErr); errText != "" {
				switch le.mode {
				case edModeImporting, edModeImport, edModeExporting, edModeExport:
					le.impErr = errText
				default:
					le.wlDownloadErr = errText
				}
				return le, le.readDownloadMsg()
			}
		}

		// Download/import/export complete (or aborted).
		if le.dlActive && msg.dlDone {
			le.dlActive = false
			le.dlOp = nil
			le.dialog = nil // clear stale dialog so results can render

			// ADIF export result.
			if le.mode == edModeExporting || le.mode == edModeExport {
				le.impInserted = msg.dlCount // reuse as exported count
				if msg.dlAborted {
					le.impErr = ""
				} else if msg.dlErr != "" {
					le.impErr = msg.dlErr
				}
				// A terminal without an error must not wipe a mid-stream
				// error captured earlier — the bare dlDone synthesized from
				// channel closure carries neither counts nor error.
				le.mode = edModeExportResult
				le.needsReload = true
				return le, nil
			}

			// ADIF import result.
			if le.mode == edModeImporting || le.mode == edModeImport {
				le.impInserted = msg.dlCount
				le.impDupes = msg.dlDupes
				le.impFailed = msg.dlFailed
				if msg.dlAborted {
					le.impErr = ""
				} else if msg.dlErr != "" {
					le.impErr = msg.dlErr
				}
				// Same as export: a bare terminal must not clear a
				// previously captured error.
				le.mode = edModeImportResult
				le.needsReload = true
				return le, nil
			}

			// Wavelog download result.
			if msg.dlAborted {
				le.wlDownloadCount = msg.dlCount
				le.wlDownloadDupes = msg.dlDupes
				le.wlDownloadErr = ""
				le.wlDownloadHold = 0
				le.wlDownloadAbort = true
				le.mode = edModeWLDownloadResult
				le.needsReload = true
			} else if msg.dlErr != "" {
				le.wlDownloadErr = msg.dlErr
				le.mode = edModeWLDownloadResult
				le.needsReload = true
			} else if le.wlDownloadErr != "" {
				// Error was already captured from a previous dlErr message;
				// keep it and transition to result screen.
				le.mode = edModeWLDownloadResult
				le.needsReload = true
			} else {
				le.wlDownloadCount = msg.dlCount
				le.wlDownloadDupes = msg.dlDupes
				le.wlDownloadFailed = msg.dlFailed
				le.wlDownloadHold = msg.dlTransient + msg.dlUnresolved
				le.wlDownloadErr = ""
				le.wlDownloadAbort = false
				le.mode = edModeWLDownloadResult
				le.needsReload = true
			}
			return le, nil
		}

		if msg.deleted != 0 || msg.saved != 0 || msg.purged || msg.wlCall != "" {
			// A completion from a replaced editor (logbook switched), a
			// serialized follow-up PATCH, or a superseded edit session must
			// never close the form being edited now — only the unchanged
			// session that initiated the operation may transition back to
			// the list.
			closeForm := true
			if msg.gen != 0 && msg.gen != le.gen {
				closeForm = false // completion from a replaced editor
			}
			if msg.wlSyncFollowUp {
				closeForm = false // follow-up PATCH — the form closed at the original save
			}
			if msg.saved != 0 && msg.saveSession != 0 && msg.saveSession != le.editSession {
				closeForm = false // the contact was reopened — a newer session owns the form
			}
			if msg.saved != 0 && msg.saveRev != le.editRev {
				// The operator typed after the save was captured — the form
				// holds newer unsaved input that must not be abandoned.
				closeForm = false
			}
			if msg.deleted != 0 && msg.opSessionSet && msg.opSession != le.editSession {
				closeForm = false // another contact was opened while the remote delete was pending
			}
			if msg.wlCall != "" && msg.opSessionSet && msg.opSession != le.editSession {
				closeForm = false // another contact was opened while the upload was pending
			}
			if msg.purged && msg.opSessionSet && msg.opSession != le.editSession {
				closeForm = false // a contact was opened while the purge ran
			}
			if closeForm {
				le.mode = edModeList
				le.needsReload = true
			} else if (msg.deleted != 0 || msg.wlCall != "") && (msg.gen == 0 || msg.gen == le.gen) {
				// A delete/upload of THIS logbook finished while the operator
				// edits another contact: refresh the list behind the open
				// form without closing it.
				le.needsReload = true
			}
		}

	case tea.PasteMsg:
		// Forward clipboard paste to the focused text input during
		// inline editing. Non-focusable fields (WLStatus, Source) are
		// skipped — they're read-only display fields.
		if le.mode == edModeEdit && le.focus != qefWLStatus && le.focus != qefSource {
			prev := le.fields[le.focus].Value()
			le.fields[le.focus], _ = le.fields[le.focus].Update(msg)
			if le.fields[le.focus].Value() != prev {
				le.editRev++
			}
		}
		return le, nil

	case tea.KeyPressMsg:
		k := msg.String()

		// File export/import mode — route ALL keys to the filepicker handler.
		if le.mode == edModeExport || le.mode == edModeImport {
			return le.handleFilePickerUpdate(msg)
		}

		// In list mode, forward non-navigation keys to search input.
		// Navigation keys (up/down/pgup/pgdn/home/end) still go to the table.
		if le.mode == edModeList {
			switch k {
			case "backspace":
				if le.searchQuery != "" {
					le.searchInput.SetValue("")
					le.searchQuery = ""
					le.applySearchFilter()
				}
				return le, nil
			case "up", "down", "left", "right", "home", "end",
				"pgup", "pgdown",
				"esc", "f8",
				"delete", "enter",
				"ctrl+w", "alt+w", "ctrl+e", "ctrl+p", "ctrl+i":
				// Navigation and action keys — handled below.
			default:
				// Forward to search input.
				prev := le.searchInput.Value()
				le.searchInput, _ = le.searchInput.Update(msg)
				if le.searchInput.Value() != prev {
					le.searchQuery = strings.TrimSpace(le.searchInput.Value())
					le.applySearchFilter()
				}
				return le, nil
			}
		}

		// Download progress — route keys to the dialog (Abort button).
		if le.dlActive && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d, ok := updated.(DialogModel)
			if !ok {
				return le, le.readDownloadMsg()
			}
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				// The operation object stays alive: the worker needs it to
				// send the final dlDone message. cancelDownload is idempotent
				// and never nils the channels the worker captured.
				le.cancelDownload()
			}
			return le, le.readDownloadMsg()
		}

		// Download/import result — route keys to the dialog (OK button).
		if le.mode == edModeWLDownloadResult && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d, ok := updated.(DialogModel)
			if !ok {
				return le, nil
			}
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				le.mode = edModeList
				le.needsReload = true
			}
			return le, nil
		}
		if le.mode == edModeImportResult && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d, ok := updated.(DialogModel)
			if !ok {
				return le, nil
			}
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				le.mode = edModeList
				le.needsReload = true
			}
			return le, nil
		}
		if le.mode == edModeExportResult && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d, ok := updated.(DialogModel)
			if !ok {
				return le, nil
			}
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				le.mode = edModeList
				le.needsReload = true
			}
			return le, nil
		}

		// Confirm modes — route keys to the dialog with left/right navigation.
		if le.isModalMode() && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d, ok := updated.(DialogModel)
			if !ok {
				return le, nil
			}
			*le.dialog = d
			if d.Done() {
				if d.Result.Confirmed && d.Result.Value != "cancel" {
					return le, le.doConfirm()
				}
				le.dialog = nil
				le.mode = edModeList
			}
			return le, nil
		}

		if le.mode == edModeEdit {
			// Shared navigation: Tab/Down, Shift+Tab/Up, and the Save & Back
			// button (Space/Enter opens the save confirmation).
			if handled, cmd := le.fm.onKey(msg, le, func() tea.Cmd {
				le.mode = edModeConfirmSave
				return nil
			}); handled {
				return le, cmd
			}
			switch k {
			case "enter":
				// Enter opens the save confirmation — same dialog flow
				// as deleting a QSO.
				le.mode = edModeConfirmSave
				return le, nil
			case "esc":
				le.mode = edModeList
			case "pgup", "pgdown", "home", "end":
				le.editVP, _ = le.editVP.Update(msg)
				return le, nil
			default:
				if le.focus != qefWLStatus && le.focus != qefSource {
					prev := le.fields[le.focus].Value()
					le.fields[le.focus], _ = le.fields[le.focus].Update(msg)
					if le.fields[le.focus].Value() != prev {
						le.editRev++
					}
				}
			}
			return le, nil
		}

		// modeList — table handles navigation; we intercept page transitions.
		switch k {
		case "esc":
			le.done = true
		case "pgup":
			le.goToPage(le.currentPage - 1)
		case "pgdown":
			le.goToPage(le.currentPage + 1)
		case "up", "down", "left", "right", "home", "end":
			// Before passing to table, check for page boundary overflow.
			cursor := le.table.Cursor()
			if k == "down" {
				if cursor >= len(le.qsos)-1 && le.currentPage < le.totalPages() {
					le.goToPage(le.currentPage + 1)
					return le, nil
				}
			}
			if k == "up" {
				if cursor <= 0 && le.currentPage > 1 {
					le.goToPage(le.currentPage - 1)
					// Set cursor to last row of the new page.
					if len(le.qsos) > 0 {
						le.table.SetCursor(len(le.qsos) - 1)
					}
					return le, nil
				}
			}
			var cmd tea.Cmd
			le.table, cmd = le.table.Update(msg)
			return le, cmd
		case "delete":
			if len(le.qsos) > 0 {
				le.dialog = nil
				le.mode = edModeConfirmDelete
			}
		case "ctrl+w":
			if le.Offline {
				return le, func() tea.Msg { return editorMsg{toastWarn: "Wavelog: network not available — cannot upload"} }
			}
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" {
				if len(le.qsos) == 0 {
					return le, func() tea.Msg { return editorMsg{toastWarn: "Wavelog: logbook is empty — nothing to upload"} }
				}
				// Count unsent QSOs from the full database with a single
				// SQL COUNT — never a full-log scan.
				le.wlUnsentCount = 0
				if le.db != nil {
					if n, cntErr := store.CountUnsentQSOs(le.db); cntErr == nil {
						le.wlUnsentCount = n
					}
				}
				if le.wlUnsentCount == 0 {
					// Fallback: count from current page if DB query failed.
					for _, q := range le.qsos {
						if q.WavelogID == 0 {
							le.wlUnsentCount++
						}
					}
				}
				le.dialog = nil
				le.mode = edModeConfirmWLSend
			}
		case "alt+w":
			if le.Offline {
				return le, func() tea.Msg { return editorMsg{toastWarn: "Wavelog: network not available — cannot download"} }
			}
			if le.contestID != "" {
				return le, func() tea.Msg {
					return editorMsg{toastWarn: "Wavelog download is not available in contest-filtered view"}
				}
			}
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" {
				le.dialog = nil
				le.mode = edModeConfirmWLDownload
			}
		case "enter":
			if len(le.qsos) > 0 {
				idx := le.table.Cursor()
				if idx >= len(le.qsos) {
					idx = 0
				}
				q := le.qsos[idx]
				le.editing = &q
				le.editRev = 0
				// A new edit session must never share a refresh identity
				// with a previous session for the same contact — otherwise
				// a stale refresh from the earlier session could pass every
				// check after the contact is reopened.
				le.editSession++
				le.fillEditForm(&q)
				le.fm.reset()
				le.focusRow(int(qefCall))
				le.mode = edModeEdit
				// Synced QSOs: refresh the form with the server's current copy —
				// unless the local row carries changes that never reached the
				// server (offline save, failed PATCH). Those must not be
				// overwritten by the stale remote copy.
				if q.WavelogID > 0 && !le.Offline && le.wlURL != "" && le.wlKey != "" {
					if dirty, err := store.QSOHasPendingSync(le.db, q.ID); err == nil && dirty {
						return le, func() tea.Msg {
							return editorMsg{toastWarn: "Wavelog: local changes not synced yet — editing the local copy"}
						}
					}
					return le, le.fetchRemoteCopy(q.WavelogID, q.ID)
				}
			}
		case "ctrl+p":
			le.dialog = nil
			le.mode = edModeConfirmPurge
		case "ctrl+e":
			le.filePicker = filepicker.New()
			le.filePicker.FileAllowed = false
			le.filePicker.DirAllowed = true
			le.filePicker.AutoHeight = false
			le.filePicker.ShowHidden = false
			if home, err := os.UserHomeDir(); err == nil {
				le.filePicker.CurrentDirectory = home
			}
			le.mode = edModeExport
			return le, le.filePicker.Init()
		case "ctrl+i":
			// Only trigger import in list mode.
			if le.mode == edModeList {
				le.filePicker = filepicker.New()
				le.filePicker.AllowedTypes = []string{".adi", ".adif"}
				le.filePicker.FileAllowed = true
				le.filePicker.DirAllowed = true
				le.filePicker.AutoHeight = false
				le.filePicker.ShowHidden = false
				if home, err := os.UserHomeDir(); err == nil {
					le.filePicker.CurrentDirectory = home
				}
				le.mode = edModeImport
				return le, le.filePicker.Init()
			}
		}
	}

	// During download, always keep the channel reader alive.  Other messages
	// (ticks, flrig polls, etc.) would otherwise replace readDownloadMsg.
	if le.dlActive {
		return le, le.readDownloadMsg()
	}

	// File export/import mode — route non-key messages to the filepicker.
	if le.mode == edModeExport || le.mode == edModeImport {
		if _, isKey := msg.(tea.KeyPressMsg); !isKey {
			return le.handleFilePickerUpdate(msg)
		}
		return le, nil
	}

	// Reload QSO list from DB after purge, import, download, or upload
	// operations that set the needsReload flag. Without this, le.qsos
	// holds stale data and batch uploads operate on outdated QSOs.
	if le.needsReload && le.mode == edModeList && le.height > 0 {
		le.needsReload = false
		le.loadPage()
	}

	return le, nil
}

// handleNormalizeResult applies a completed normalization on the owner loop:
// the worker's changed fields are applied to the in-memory list and the
// post-normalize upload is launched with the transferred database lease. It
// runs FIRST in the editorMsg handling — a download in progress would
// otherwise swallow the result into its progress branches and leak
// normRelease.
func (le *LogbookEditor) handleNormalizeResult(msg editorMsg) (tea.Model, tea.Cmd) {
	// The normalize worker belongs to the editor that launched it; a result
	// delivered to a replacement editor (F8 / logbook switched
	// mid-operation) must not trigger an upload there — and must release
	// its lease.
	if msg.gen != 0 && msg.gen != le.gen {
		applog.Warn("Wavelog: discarding normalize result for a replaced editor",
			fmt.Sprintf("msg_gen=%d editor_gen=%d", msg.gen, le.gen))
		if msg.normRelease != nil {
			msg.normRelease()
		}
		return le, nil
	}
	// Apply the worker's returned changes on the owner loop only — the
	// worker is limited to database work and must never mutate le.qsos
	// from its goroutine (the owner loop reads it during table rebuilds).
	for i := range le.qsos {
		for _, mid := range msg.normIDs {
			if le.qsos[i].ID == mid {
				if msg.normCall != "" {
					le.qsos[i].StationCallsign = msg.normCall
				}
				if msg.normOp != "" {
					le.qsos[i].Operator = msg.normOp
				}
				if msg.normGrid != "" {
					le.qsos[i].MyGridSquare = msg.normGrid
				}
				break
			}
		}
	}
	// Normalization done, now upload all unsent QSOs — fetched as eligible
	// rows only, never a full-log scan.
	var unsent []qso.QSO
	if le.db != nil {
		rows, listErr := store.ListUnsentQSOs(le.db)
		if listErr != nil {
			applog.Error("Wavelog: post-normalize upload — cannot list QSOs", "error", listErr)
			if msg.normRelease != nil {
				msg.normRelease()
			}
			return le, func() tea.Msg {
				return editorMsg{wlOK: false, err: fmt.Errorf("cannot read logbook: %w", listErr)}
			}
		}
		for _, q := range rows {
			if q.Band == "" || q.Mode == "" || q.QSODate == "" {
				continue
			}
			unsent = append(unsent, q)
		}
	} else {
		for _, q := range le.qsos {
			if q.WavelogID == 0 {
				if q.Band == "" || q.Mode == "" || q.QSODate == "" {
					continue
				}
				unsent = append(unsent, q)
			}
		}
	}
	return le, le.uploadBatchLeased(unsent, msg.normRelease)
}

func (le *LogbookEditor) handleFilePickerUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+e", "ctrl+i", "esc":
			le.mode = edModeList
			return le, nil
		case "enter":
			if le.mode == edModeExport {
				// Export: confirm current directory, start async export with progress.
				dir := le.filePicker.CurrentDirectory
				if p := le.filePicker.Path; p != "" {
					dir = p
				}
				ts := time.Now().UTC().Format("20060102_150405")
				name := "cqops"
				if le.logStationCall != "" {
					n := strings.ReplaceAll(le.logStationCall, "/", "-")
					name = strings.ToLower(strings.ReplaceAll(n, " ", "_"))
				}
				// Append contest info when a contest filter is active.
				if le.contestID != "" && le.contestAdifID != "" {
					cid := sanitizeFilename(le.contestAdifID)
					name += "_" + strings.ToLower(cid)
					if le.contestDate != "" {
						name += "_" + strings.ReplaceAll(le.contestDate, "-", "")
					}
				}
				path := filepath.Join(dir, fmt.Sprintf("%s_%s.adi", ts, name))
				le.exportPath = path
				// Start async export with progress dialog.
				le.dlProgress = 0
				le.dlTotal = 0
				le.dlCurrent = 0
				le.impInserted = 0
				le.impErr = ""
				le.dlActive = true
				op := newDownloadOp()
				le.dlOp = op
				le.mode = edModeExporting
				release := le.dbLease(le.db)
				go le.runExport(op, path, release)
				return le, le.readDownloadMsg()
			}
			// Import: let filepicker handle selection via DidSelectFile below.
		}
	}

	var cmd tea.Cmd
	le.filePicker, cmd = le.filePicker.Update(msg)

	// Import mode: check if a file was selected.
	if le.mode == edModeImport {
		if didSelect, path := le.filePicker.DidSelectFile(msg); didSelect && path != "" {
			// Validate that the selected file is an ADIF file.
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".adi" && ext != ".adif" {
				return le, func() tea.Msg { return editorMsg{toastWarn: "ADIF: only .adi / .adif files can be imported"} }
			}
			// Start async import with progress dialog.
			le.dlProgress = 0
			le.dlTotal = 0
			le.dlCurrent = 0
			le.dlActive = true
			op := newDownloadOp()
			le.dlOp = op
			le.impInserted = 0
			le.impDupes = 0
			le.impFailed = 0
			le.impErr = ""
			le.mode = edModeImporting
			release := le.dbLease(le.db)
			go le.runImport(op, path, release)
			return le, le.readDownloadMsg()
		}
	}

	return le, cmd
}

func (le *LogbookEditor) doConfirm() tea.Cmd {
	if le.dialog == nil {
		return nil
	}
	mode := le.mode // capture before clearing
	applog.Debug("LogEditor: dialog response", "mode", mode, "value", le.dialog.Result.Value)
	le.dialog = nil
	switch mode {
	case edModeConfirmNormalize:
		return le.doNormalizeAndUpload()
	case edModeConfirmWLSend:
		le.mode = edModeList
		return le.doBatchUpload()
	case edModeConfirmWLDownload:
		le.mode = edModeWLDownloading
		le.dlProgress = 0
		le.dlTotal = 0
		return le.doWavelogDownload()
	case edModeConfirmPurge:
		le.mode = edModeList
		le.wlLastFetchedID = 0
		// Clear any lingering download state.
		le.dlProgress = 0
		le.dlTotal = 0
		le.cancelDownload()
		le.dlOp = nil
		gen := le.gen
		lbID := le.logbookID
		opSession := le.editSession
		db := le.db
		release := le.dbLease(db)
		applog.Warn("LogbookEditor: purging all QSOs")
		return func() tea.Msg {
			defer release()
			err := store.PurgeQSOs(db)
			if err != nil {
				applog.Error("LogbookEditor: purge failed", "error", err.Error())
			} else {
				applog.Info("LogbookEditor: all QSOs purged")
			}
			return editorMsg{purged: true, err: err, gen: gen, lbID: lbID, opSession: opSession, opSessionSet: true}
		}
	case edModeConfirmSave:
		// Keep the edit form visible until the async save result arrives;
		// the editorMsg{saved} handler switches back to the list.
		le.mode = edModeEdit
		return le.doSave()
	case edModeConfirmDelete:
		if len(le.qsos) == 0 {
			le.mode = edModeList
			return nil
		}
		idx := le.table.Cursor()
		if idx >= len(le.qsos) {
			idx = 0
		}
		q := le.qsos[idx]
		call := q.Call
		date := formatDate(q.QSODate)
		id := q.ID
		remoteID := q.WavelogID
		url, key := le.wlURL, le.wlKey
		gen := le.gen
		lbID := le.logbookID
		opSession := le.editSession
		db := le.db
		release := le.dbLease(db)
		le.mode = edModeList
		applog.Info("LogbookEditor: deleting QSO", "id", id, "call", call, "date", date)
		return func() tea.Msg {
			defer release()
			err := store.DeleteQSO(db, id)
			if err != nil {
				applog.Error("LogbookEditor: delete failed", "id", id, "call", call, "error", err.Error())
				// No deleted id on failure — the handler shows the error toast only.
				return editorMsg{err: err}
			}
			applog.Info("LogbookEditor: QSO deleted", "id", id, "call", call)

			// Synced QSOs: remove the Wavelog copy too. Best-effort — the
			// local delete must never depend on this succeeding.
			em := editorMsg{deleted: id, delCall: call, delDate: date, gen: gen, lbID: lbID, opSession: opSession, opSessionSet: true}
			if remoteID > 0 && url != "" && key != "" && !le.Offline {
				derr := wavelog.DeleteQSO(url, key, remoteID)
				if derr != nil {
					if apiErr, ok := derr.(*wavelog.APIError); ok && apiErr.Code == "not_found" {
						em.delSyncOK = true // already gone remotely — fine
					} else {
						em.delSyncErr = wavelog.FriendlyError(derr).Error()
					}
				} else {
					em.delSyncOK = true
				}
			}
			return em
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
	synced := q.WavelogID > 0
	// Capture the form revision when the save is dispatched: the form
	// stays editable while the PATCH runs, and a completion must only
	// close it when the operator has not typed since.
	saveRev := le.editRev
	// Capture what the worker needs up front — it must not read editor
	// state from the command goroutine.
	db := le.db
	url, key := le.wlURL, le.wlKey
	offline := le.Offline
	gen := le.gen
	applog.Info("LogbookEditor: saving QSO", "id", id, "call", call, "date", date)

	// The local write happens HERE, on the owner loop: the bumped revision
	// must be durable before deciding whether to launch or queue the remote
	// PATCH, so a queued edit can never be read after its follow-up was
	// dispatched. A crash while the PATCH is in flight still leaves a
	// durably dirty row that a remote refresh cannot overwrite.
	var rev int64
	if synced {
		var serr error
		rev, serr = store.SaveQSOForSync(db, q)
		if serr != nil {
			applog.Error("LogbookEditor: save failed", "id", id, "call", call, "error", serr.Error())
			// No saved id on failure — the handler shows the error toast only
			// (the edit form stays open for a retry).
			return func() tea.Msg { return editorMsg{err: serr, gen: gen} }
		}
	} else {
		if uerr := store.UpdateQSO(db, q); uerr != nil {
			applog.Error("LogbookEditor: save failed", "id", id, "call", call, "error", uerr.Error())
			return func() tea.Msg { return editorMsg{err: uerr, gen: gen} }
		}
	}
	applog.Info("LogbookEditor: QSO saved", "id", id, "call", call)

	// Synced QSOs: push the edit to Wavelog. Local save must never depend
	// on this succeeding — and offline mode must never contact the server
	// at all (the local change is reported as pending sync).
	em := editorMsg{saved: id, saveCall: call, saveDate: date, gen: gen, saveSession: le.editSession, saveRev: saveRev, lbID: le.logbookID}
	if !synced {
		return func() tea.Msg { return em }
	}
	// The row is already durably marked pending. Without credentials
	// there is no way to push now — keep the pending state so a later
	// refresh cannot clobber the edit once credentials are configured.
	if url == "" || key == "" {
		return func() tea.Msg { return em }
	}
	if offline {
		em.wlSyncPending = true
		return func() tea.Msg { return em }
	}
	// Serialize remote updates per contact: only ONE PATCH per QSO may be
	// on the wire. A save arriving while one is pending is queued (the
	// newest revision is already stored locally); when the in-flight PATCH
	// completes, the owner loop dispatches a follow-up carrying the newest
	// revision. Without this an older stalled PATCH could complete after a
	// newer one and overwrite the server with stale values.
	//
	// The coordination lives on the model-owned coordinator and is keyed by
	// persistent logbook/contact identity and remote source — NOT by this
	// disposable editor instance. Reopening the editor (F8) while a PATCH
	// is on the wire must still queue the next save, never dispatch it in
	// parallel.
	if le.sync == nil {
		le.sync = &contactSyncCoord{}
	}
	syncKey := contactSyncKey{logbook: le.logbookID, localID: id, url: url}
	if le.sync.inFlight == nil {
		le.sync.inFlight = make(map[contactSyncKey]*contactSyncContext)
	}
	if le.sync.queued == nil {
		le.sync.queued = make(map[contactSyncKey]bool)
	}
	if le.sync.inFlight[syncKey] != nil {
		le.sync.queued[syncKey] = true
		em.wlSyncPending = true
		return func() tea.Msg { return em }
	}
	// Chain dispatch: acquire the database lease for the WHOLE queued
	// operation — the in-flight PATCH, every follow-up, and the interval
	// between workers. The owner loop releases it only when the chain is
	// fully drained or abandoned, so a logbook switch retires the
	// originating database instead of closing it mid-chain.
	ctx := &contactSyncContext{
		key:     syncKey,
		coord:   le.sync,
		db:      db,
		url:     url,
		apiKey:  key,
		gen:     gen,
		release: le.dbLease(db),
	}
	le.sync.inFlight[syncKey] = ctx
	em.syncCtx = ctx
	return func() tea.Msg {
		// The PATCH worker writes locally (ack persistence, gone handling)
		// against the ORIGINATING database only — the chain lease holds it
		// open, and the owner loop releases it at chain end.
		syncErr := wavelog.UpdateQSO(ctx.url, ctx.apiKey, q.WavelogID, buildUpdateInput(q))
		if syncErr != nil {
			if apiErr, ok := syncErr.(*wavelog.APIError); ok && apiErr.Code == "not_found" {
				// The remote copy was deleted elsewhere — the local id is
				// stale; clear it so the row is honest again.
				if serr := store.SetWavelogID(ctx.db, id, 0); serr == nil {
					em.wlSyncGone = true
					if derr := store.SetWavelogDirty(ctx.db, id, false); derr != nil {
						applog.Warn("Wavelog: failed to clear pending sync", "qso_id", id, "error", derr)
					}
				} else {
					em.wlSyncErr = wavelog.FriendlyError(syncErr).Error()
				}
			} else {
				// Transient PATCH failure — the row is already durably
				// marked pending by SaveQSOForSync.
				em.wlSyncErr = wavelog.FriendlyError(syncErr).Error()
			}
			return em
		}
		// Success — clear the pending flag only if the row still carries
		// THIS edit's revision. If the operator saved again while this
		// PATCH was in flight, an older acknowledgement must not clear
		// the newer pending edit.
		em.wlSyncOK = true
		if derr := store.ClearWavelogDirtyIfRevision(ctx.db, id, rev); derr != nil {
			applog.Warn("Wavelog: failed to clear pending sync", "qso_id", id, "error", derr)
			// The remote copy synced, but the local acknowledgement could
			// not be persisted — report an incomplete synchronization result
			// so the row stays pending instead of claiming full success.
			em.wlSyncOK = false
			em.wlSyncIncomplete = true
		}
		return em
	}
}

// downloadOp carries immutable per-operation state for a download/import/
// export: the progress channel and a cancellable context. It is captured by
// the worker goroutine at launch and never mutated afterward, so the UI can
// tear down editor state without losing cancellation or blocking sends.
type downloadOp struct {
	id     uint64
	msgCh  chan editorMsg
	ctx    context.Context
	cancel context.CancelFunc
	// readPending is true while the operation's single channel read is in
	// flight — ticks and progress messages must not dispatch a second
	// reader, which would receive the channel-close zero value and could
	// finalize the operation before the real result.
	readPending atomic.Bool
	// channelClosed records that the worker closed the channel; no further
	// reads are dispatched after that.
	channelClosed atomic.Bool
}

// downloadOpCounter hands out per-operation ids; download messages are
// stamped with them so a result from a replaced operation is discarded.
var downloadOpCounter atomic.Uint64

func newDownloadOp() *downloadOp {
	ctx, cancel := context.WithCancel(context.Background())
	return &downloadOp{id: downloadOpCounter.Add(1), msgCh: make(chan editorMsg, 4), ctx: ctx, cancel: cancel}
}

// stop requests cancellation. Safe to call multiple times.
func (op *downloadOp) stop() { op.cancel() }

// cancelled reports whether the operation was cancelled.
func (op *downloadOp) cancelled() bool {
	select {
	case <-op.ctx.Done():
		return true
	default:
		return false
	}
}

// send delivers a non-terminal progress message without ever blocking
// forever: once cancelled (or the consumer is gone), the send is dropped.
// Terminal messages (dlDone/dlErr) use the plain channel — the consumer is
// guaranteed to keep reading until the channel closes.
func (op *downloadOp) send(em editorMsg) {
	select {
	case op.msgCh <- em:
	case <-op.ctx.Done():
	}
}

// doWavelogDownload fetches contacts from Wavelog, deduplicates against local DB,
// and inserts new QSOs in batches. Progress is reported via editorMsg so the UI
// can show a live counter. The user can abort by pressing any key.
func (le *LogbookEditor) doWavelogDownload() tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	fetchFromID := le.wlLastFetchedID

	op := newDownloadOp()
	le.dlOp = op
	le.dlActive = true

	applog.InfoDetail("Wavelog: starting contacts download",
		fmt.Sprintf("url=%s station_id=%s from_id=%d", url, sid, fetchFromID))

	// The download writes the imported QSOs locally; the database is leased
	// now so a logbook switch retires — not closes — it for the duration.
	release := le.dbLease(le.db)

	// Start the download goroutine with the operation captured.
	go le.runDownload(op, url, key, sid, fetchFromID, release)

	// Return a Cmd that reads the first progress message.
	return le.readDownloadMsg()
}

// cancelDownload asks the active operation to stop. Idempotent — the
// operation object survives so the worker can deliver its final message.
func (le *LogbookEditor) cancelDownload() {
	if le.dlOp != nil {
		le.dlOp.stop()
	}
}

// readDownloadMsg reads the next message from the active operation's channel.
// Exactly ONE read may be pending per operation: extra dispatches (ticks,
// polls) return nil while a read is in flight — a second blocked reader
// would receive the channel-close zero value and could finalize the
// operation before the real result arrives. Messages are stamped with the
// operation id so a result from a replaced operation is discarded.
func (le *LogbookEditor) readDownloadMsg() tea.Cmd {
	op := le.dlOp
	if op == nil || op.channelClosed.Load() {
		return nil
	}
	if !op.readPending.CompareAndSwap(false, true) {
		return nil // a read is already pending
	}
	return func() tea.Msg {
		msg, ok := <-op.msgCh
		// readPending deliberately stays TRUE here: the owner loop clears it
		// only after accepting this result (see the editorMsg handling).
		// Clearing it in the reader goroutine would let a tick in the gap
		// before Update dispatch a second reader, which could observe the
		// channel closure and overtake the real terminal.
		if !ok {
			op.channelClosed.Store(true)
			// The channel closed without this read receiving a terminal
			// message. Never treat the close as an independent success: the
			// handler finalizes honestly (captured error, or an explicit
			// did-not-finish) instead of a silent zero-count result.
			return editorMsg{dlDone: true, dlChannelClosed: true, dlOpID: op.id}
		}
		msg.dlOpID = op.id
		return msg
	}
}

// runDownload performs the actual HTTP fetch, saves ADIF to a temp file,
// then processes it line-by-line. Progress is reported per batch (every 50 QSOs
// or when done). The goroutine checks op.ctx between records; on abort
// processing stops immediately. Downloads are idempotent — duplicates are
// detected and skipped.
func (le *LogbookEditor) runDownload(op *downloadOp, url, key, sid string, fetchFromID int64, release func()) {
	// The channel lives on the immutable operation object — the UI handler
	// can clear editor state without breaking sends, close, or cancellation.
	defer release()
	msgCh := op.msgCh
	defer close(msgCh)

	db := le.db

	// Send initial message so the dialog appears immediately.
	op.send(editorMsg{dlProgress: 0, dlTotal: 0})

	// Page-level progress: the v2 export is paginated; update the download
	// bar as each page arrives instead of waiting for the whole fetch.
	result, err := wavelog.FetchContactsProgressCtx(op.ctx, url, key, sid, fetchFromID,
		func(exported, total int) {
			op.send(editorMsg{dlProgress: exported, dlTotal: total})
		})
	if err != nil {
		if op.cancelled() {
			msgCh <- editorMsg{dlCount: 0, dlDupes: 0, dlAborted: true, dlDone: true, gen: le.gen, lbID: le.logbookID}
			return
		}
		applog.ErrorDetail("Wavelog: contacts download failed",
			fmt.Sprintf("url=%s station_id=%s from_id=%d error=%v", url, sid, fetchFromID, err))
		msgCh <- editorMsg{dlErr: err.Error(), gen: le.gen, lbID: le.logbookID}
		return
	}

	// Clean up temp file when done.
	defer os.Remove(result.ADIFPath)

	if result.ADIFPath == "" || result.ExportedQSOs == 0 {
		applog.Info("Wavelog: no new contacts to download")
		msgCh <- editorMsg{dlCount: 0, dlLastID: result.LastFetchedID(), dlDownload: true, dlDone: true, gen: le.gen, lbID: le.logbookID}
		return
	}

	// Open the temp ADIF file for line-by-line scanning.
	f, err := os.Open(result.ADIFPath)
	if err != nil {
		applog.Error("Wavelog: failed to open temp ADIF file", "error", err)
		msgCh <- editorMsg{dlErr: "failed to read downloaded data", gen: le.gen, lbID: le.logbookID}
		return
	}
	defer f.Close()

	var inserted, dupes, failed int
	totalExported := result.ExportedQSOs
	const batchInterval = 50 // report progress every 50 QSOs for smooth but efficient UI

	// Checkpoint discipline: the persisted Wavelog cursor may only advance
	// through records that were durably handled (inserted, duplicate, or
	// permanently invalid). The first record whose INSERT fails freezes the
	// cursor so the next incremental download re-fetches it — failed
	// contacts are never skipped past.
	checkpointID := int64(0)
	checkpointOK := false
	frozen := false
	transientFails := 0
	// idGap freezes the cursor at the first record whose remote identity
	// could not be resolved (WavelogID=0). Advancing past it would make
	// the next incremental download skip the record's id range — it would
	// never learn its remote link. unresolved counts those records for the
	// result dialog.
	idGap := false
	unresolved := 0

	applog.Info("Wavelog: scanning ADIF", "exported", totalExported, "size_bytes", result.ADIFSize)

	// Stream-parse ADIF records one at a time — never loads all QSOs into memory.
	scanner := adif.NewScanner(f)
	processed := 0 // total records seen (including skipped/dupes)
	for scanner.Scan() {
		// Check for abort between records.
		if op.cancelled() {
			msgCh <- editorMsg{dlCount: inserted, dlDupes: dupes, dlAborted: true, dlDone: true, gen: le.gen, lbID: le.logbookID}
			return
		}

		if scanner.IsHeader() {
			continue
		}
		r := scanner.Record()
		processed++

		qs := qso.ParseADIFRecord(r, "wavelog")

		// Remote ids are matched by verified identity (call/band/mode/
		// date/time), never by position: a failed or mismatched id page
		// leaves its records without ids, and the checkpoint freezes on
		// them instead of skipping past.
		if len(result.WavelogIDsByKey) > 0 {
			qs.WavelogID = result.WavelogIDsByKey[wavelog.MakeQSOIDKeyFromADIF(
				qs.Call, qs.Band, qs.Mode, qs.Submode, qs.QSODate, qs.TimeOn)]
		}

		// Enrich: compute distance/bearing if both grids are available.
		if myGrid := strings.TrimSpace(le.logStationGrid); myGrid != "" && qs.GridSquare != "" {
			qs.Distance = gridDistanceKm(myGrid, qs.GridSquare)
			qs.Bearing = gridBearingDeg(myGrid, qs.GridSquare)
		}

		if err := qso.ValidateImportRecord(qs); err != nil {
			applog.Warn("Wavelog: skipping invalid imported QSO", "call", qs.Call, "reason", err)
			failed++
			// Permanently invalid — but it may advance the checkpoint only
			// when every preceding record was durably handled. An earlier
			// unresolved identity (idGap) or failed insert (frozen) keeps
			// the cursor behind this record too, otherwise the next
			// incremental download would skip the gap's range.
			if !frozen && !idGap && qs.WavelogID > 0 {
				checkpointID = qs.WavelogID
				checkpointOK = true
			}
			continue
		}

		if existingID := store.FindQSOByKey(db, qs.Call, qs.Band, qs.Mode, qs.QSODate, qs.TimeOn); existingID != 0 {
			// The row already exists locally — still learn its remote id
			// when the local copy has none, so it counts as uploaded.
			if qs.WavelogID > 0 {
				if serr := store.SetWavelogID(db, existingID, qs.WavelogID); serr != nil {
					applog.Warn("Wavelog: failed to store remote id for dupe", "qso_id", existingID, "error", serr)
					// The recovered id could not be persisted — the row is
					// still locally unresolved. Freeze advancement so the
					// next download retries this recovery instead of
					// skipping the record's id range forever.
					unresolved++
					idGap = true
				} else if !frozen && !idGap {
					checkpointID = qs.WavelogID
					checkpointOK = true
				}
			} else {
				// The existing row's remote identity is still unknown —
				// the cursor must stay behind it so the next download can
				// resolve it.
				unresolved++
				idGap = true
			}
			continue
		}

		// Retry on SQLITE_BUSY — the main thread may briefly hold a write lock
		// (e.g. during loadPage after purge).  A short sleep + retry resolves
		// nearly all transient lock conflicts.
		var insertErr error
		for attempt := 0; attempt < 3; attempt++ {
			_, insertErr = store.InsertQSO(db, qs)
			if insertErr == nil {
				break
			}
			if !strings.Contains(insertErr.Error(), "database is locked") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if insertErr != nil {
			applog.Error("Wavelog: failed to insert downloaded QSO", "call", qs.Call, "error", insertErr)
			failed++
			transientFails++
			frozen = true // never advance the checkpoint past an unsaved record
			continue
		}
		inserted++
		if qs.WavelogID > 0 {
			if !frozen && !idGap {
				checkpointID = qs.WavelogID
				checkpointOK = true
			}
		} else {
			// Imported without a remote link — the cursor must not advance
			// past this record or its identity would never be reconciled.
			unresolved++
			idGap = true
		}

		// Report progress every batchInterval QSOs, and always for the first
		// one (so the UI transitions from "Downloading…" to "Processing QSO 1").
		if inserted == 1 || inserted%batchInterval == 0 {
			op.send(editorMsg{dlProgress: totalExported, dlTotal: totalExported, dlCount: inserted})
		}

		// Log milestone every 500 QSOs so we can trace progress in logs.
		if inserted%500 == 0 && inserted > 0 {
			applog.Info("Wavelog: download progress", "inserted", inserted, "dupes", dupes, "processed", processed)
		}
	}

	// Send final progress (catch remainder if last batch was partial).
	if inserted%batchInterval != 0 {
		op.send(editorMsg{dlProgress: totalExported, dlTotal: totalExported, dlCount: inserted})
	}

	scanFailed := false
	if err := scanner.Err(); err != nil {
		applog.Error("Wavelog: ADIF scanner error", "error", err, "inserted", inserted, "dupes", dupes)
		scanFailed = true
	}

	// Compute the persistable cursor. With a scanner error, a transient
	// insert failure, or an unresolved remote identity, records beyond the
	// durable prefix are uncertain and must be re-fetched — the cursor
	// stays behind them.
	lastID := result.LastFetchedID()
	if scanFailed || frozen || idGap {
		if checkpointOK {
			lastID = checkpointID
		} else {
			// No expressible checkpoint (remote ids unavailable) — stay at
			// the starting cursor so nothing is skipped.
			lastID = fetchFromID
		}
	} else if checkpointOK {
		lastID = checkpointID
	}

	applog.Info("Wavelog: contacts download complete",
		"inserted", inserted, "dupes", dupes, "failed", failed, "processed", processed,
		"last_id", lastID, "transient", transientFails, "unresolved", unresolved, "scan_failed", scanFailed)
	if dupes > 0 {
		applog.Warn("Wavelog: ADIF export contains duplicate QSO records — skipped during import",
			"dupe_count", dupes,
			"note", "These QSOs exist more than once in the Wavelog database and should be cleaned up at the source.")
	}

	msgCh <- editorMsg{
		dlCount:      inserted,
		dlDupes:      dupes,
		dlFailed:     failed,
		dlLastID:     lastID,
		dlDownload:   true,
		dlTransient:  transientFails,
		dlUnresolved: unresolved,
		dlDone:       true,
		gen:          le.gen,
		lbID:         le.logbookID,
	}
}

// runImport performs ADIF import from a local file with progress reporting.
// It mirrors runDownload but reads from a local file instead of Wavelog HTTP.
func (le *LogbookEditor) runImport(op *downloadOp, path string, release func()) {
	defer release()
	msgCh := op.msgCh
	defer close(msgCh)

	db := le.db

	// Count total records for progress estimation.
	totalRecords := countADIFRecords(path)
	op.send(editorMsg{dlProgress: totalRecords, dlTotal: totalRecords, dlCount: 0})

	f, err := os.Open(path)
	if err != nil {
		applog.Error("ADIF import: failed to open file", "path", path, "error", err)
		// Terminal result carrying the error — the import must never end
		// on a success-looking result screen.
		msgCh <- editorMsg{dlErr: "cannot open file: " + err.Error(), dlDone: true, gen: le.gen, lbID: le.logbookID}
		return
	}
	defer f.Close()

	var inserted, dupes, failed int
	const batchInterval = 50 // report every 50 QSOs for smooth but efficient UI

	applog.Info("ADIF import: scanning", "path", path, "estimated_records", totalRecords)

	scanner := adif.NewScanner(f)
	for scanner.Scan() {
		// Check for abort between records.
		if op.cancelled() {
			msgCh <- editorMsg{dlCount: inserted, dlDupes: dupes, dlAborted: true, dlDone: true, gen: le.gen, lbID: le.logbookID}
			return
		}

		if scanner.IsHeader() {
			continue
		}
		r := scanner.Record()
		qs := qso.ParseADIFRecord(r, "import")

		// Enrich: compute distance/bearing if both grids are available.
		if myGrid := strings.TrimSpace(le.logStationGrid); myGrid != "" && qs.GridSquare != "" {
			qs.Distance = gridDistanceKm(myGrid, qs.GridSquare)
			qs.Bearing = gridBearingDeg(myGrid, qs.GridSquare)
		}

		if err := qso.ValidateImportRecord(qs); err != nil {
			applog.Warn("ADIF import: skipping invalid QSO", "call", qs.Call, "reason", err)
			failed++
			continue
		}

		if existingID := store.FindQSOByKey(db, qs.Call, qs.Band, qs.Mode, qs.QSODate, qs.TimeOn); existingID != 0 {
			applog.Warn("ADIF import: duplicate QSO skipped",
				"local_id", existingID, "call", qs.Call, "band", qs.Band, "date", qs.QSODate)
			dupes++
			continue
		}

		var insertErr error
		for attempt := 0; attempt < 3; attempt++ {
			_, insertErr = store.InsertQSO(db, qs)
			if insertErr == nil {
				break
			}
			if !strings.Contains(insertErr.Error(), "database is locked") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if insertErr != nil {
			applog.Error("ADIF import: failed to insert QSO", "call", qs.Call, "error", insertErr)
			failed++
			continue
		}
		inserted++

		// Report progress every batchInterval QSOs, and always for the first
		// one so the UI transitions from "Importing…" to showing a count.
		if inserted == 1 || inserted%batchInterval == 0 {
			op.send(editorMsg{dlProgress: totalRecords, dlTotal: totalRecords, dlCount: inserted, dlDupes: dupes})
		}

		if inserted%500 == 0 && inserted > 0 {
			applog.Info("ADIF import: progress", "inserted", inserted, "dupes", dupes, "failed", failed)
		}
	}

	if err := scanner.Err(); err != nil {
		// A truncated or corrupt file must not end on a success-looking
		// result: the single terminal message carries the error together
		// with the counts of whatever was imported before the failure.
		applog.Error("ADIF import: scanner error", "error", err, "inserted", inserted, "dupes", dupes)
		msgCh <- editorMsg{
			dlCount:  inserted,
			dlDupes:  dupes,
			dlFailed: failed,
			dlErr:    "file could not be read completely: " + err.Error(),
			dlDone:   true,
			gen:      le.gen,
			lbID:     le.logbookID,
		}
		return
	}

	applog.Info("ADIF import: complete", "inserted", inserted, "dupes", dupes, "failed", failed, "path", path)

	msgCh <- editorMsg{
		dlCount:  inserted,
		dlDupes:  dupes,
		dlFailed: failed,
		dlDone:   true,
		gen:      le.gen,
		lbID:     le.logbookID,
	}
}

// countADIFRecords quickly estimates the number of ADIF records in a file
// by counting "<CALL:" occurrences. This is fast and doesn't parse the full ADIF.
func countADIFRecords(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Read first 1MB to estimate record count.
	buf := make([]byte, 1024*1024)
	n, _ := f.Read(buf)
	if n == 0 {
		return 0
	}

	count := 0
	data := string(buf[:n])
	// Count ADIF field markers to estimate records — rough but fast.
	for {
		idx := strings.Index(data, "<CALL:")
		if idx == -1 {
			break
		}
		count++
		data = data[idx+1:]
	}

	// If file is larger than 1MB, extrapolate.
	fi, err := f.Stat()
	if err == nil && fi.Size() > int64(n) {
		count = count * int(fi.Size()) / n
	}

	return count
}

// sanitizeFilename replaces characters that are unsafe in filenames across
// Windows, macOS, and Linux. Returns a string safe for use in file paths.
func sanitizeFilename(s string) string {
	repl := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-", "?", "-",
		"\"", "-", "<", "-", ">", "-", "|", "-",
		" ", "_",
	)
	return repl.Replace(s)
}

// errExportCancelled aborts the snapshot stream when the user cancels the
// export — distinguished from real errors so no failure toast is shown.
var errExportCancelled = errors.New("export cancelled")

// runExport performs ADIF export to a local file with progress reporting.
// The file is written to a sibling temp path and promoted to the target only
// after a fully successful write and close — a failed or aborted export never
// leaves a partial file at the requested path, and every failure is reported
// as an error result instead of a success-looking one.
func (le *LogbookEditor) runExport(op *downloadOp, path string, release func()) {
	defer release()
	msgCh := op.msgCh
	defer close(msgCh)

	db := le.db

	// Count QSOs — filtered by contest if active.
	var total int
	if le.contestID != "" {
		counts, err := store.CountQSOsForContest(db, le.contestID)
		if err == nil {
			total = counts.Total
		}
	} else {
		counts, err := store.CountQSOs(db)
		if err == nil {
			total = counts.Total
		}
	}
	op.send(editorMsg{dlProgress: total, dlTotal: total, dlCount: 0})

	if total == 0 {
		msgCh <- editorMsg{dlErr: "logbook is empty", dlDone: true}
		return
	}

	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		applog.Error("ADIF export: failed to create temp file", "path", tmpPath, "error", err)
		msgCh <- editorMsg{dlErr: "cannot create file: " + err.Error(), dlDone: true, gen: le.gen, lbID: le.logbookID}
		return
	}

	written := 0
	// fail discards the temp file and emits the single terminal result.
	fail := func(reason string) {
		f.Close()
		os.Remove(tmpPath)
		applog.Error("ADIF export: failed", "path", path, "reason", reason)
		msgCh <- editorMsg{dlCount: written, dlErr: reason, dlDone: true, gen: le.gen, lbID: le.logbookID}
	}

	// Write ADIF header. Per ADIF spec, the first character must not be '<'
	// or the file is treated as having no header.
	_, err = fmt.Fprintf(f, "CQOps ADIF Export\n<ADIF_VER:5>3.1.7<PROGRAMID:5>CQOps<PROGRAMVERSION:%d>%s<EOH>\n",
		len(version.Resolved()), version.Resolved())
	if err != nil {
		fail("write error: " + err.Error())
		return
	}

	// Stream QSOs through a single read-only snapshot transaction with a
	// keyset cursor: a QSO inserted while the export runs (e.g. WSJT-X
	// logging) can neither shift the pages nor duplicate/omit records.
	// Atomic temp-file promotion protects the destination file; only the
	// snapshot protects export consistency.
	orderAsc := le.contestID != ""
	streamErr := store.ExportQSOsSnapshot(db, le.contestID, orderAsc, func(q qso.QSO) error {
		if op.cancelled() {
			return errExportCancelled
		}
		if _, err := fmt.Fprintln(f, q.ToADIF()); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		written++
		if written%500 == 0 {
			op.send(editorMsg{dlProgress: total, dlTotal: total, dlCount: written})
		}
		if written%1000 == 0 {
			applog.Info("ADIF export: progress", "written", written, "total", total)
		}
		return nil
	})
	if streamErr == errExportCancelled {
		f.Close()
		os.Remove(tmpPath)
		msgCh <- editorMsg{dlCount: written, dlAborted: true, dlDone: true, gen: le.gen, lbID: le.logbookID}
		return
	}
	if streamErr != nil {
		fail(streamErr.Error())
		return
	}

	// Check the close error: buffered writes can fail only at Close, so the
	// export is not real until it succeeds.
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		applog.Error("ADIF export: failed to close file", "path", tmpPath, "error", err)
		msgCh <- editorMsg{dlCount: written, dlErr: "write error: " + err.Error(), dlDone: true, gen: le.gen, lbID: le.logbookID}
		return
	}
	// Promote only after successful completion.
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		applog.Error("ADIF export: failed to promote file", "path", path, "error", err)
		msgCh <- editorMsg{dlCount: written, dlErr: "cannot finalize file: " + err.Error(), dlDone: true, gen: le.gen, lbID: le.logbookID}
		return
	}

	applog.Info("ADIF export: complete", "path", path, "count", written)
	msgCh <- editorMsg{dlCount: written, dlDone: true, gen: le.gen, lbID: le.logbookID}
}
