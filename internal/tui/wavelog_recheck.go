package tui

import (
	"database/sql"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Deferred recheck of uploads whose outcome is unknown.
//
// A degraded Wavelog can commit a write long after the client gave up: on
// 2026-10-29 the server stored an insert past the write deadline, and the
// contact was on the server while CQOps still showed it as unsent. The check
// made right after the timeout is usually too early — it runs while the server
// is still busy with the same insert. Rechecking a little later resolves the
// state honestly: the contact is linked as soon as the server confirms it, and
// only a recheck that never gets an answer leaves the row unsent.
// =============================================================================

// uploadRecheckDelays is how long to wait before each confirmation attempt,
// measured from the previous one. A var (not a const) so tests can shrink it.
var uploadRecheckDelays = []time.Duration{20 * time.Second, 60 * time.Second, 3 * time.Minute}

// wlRecheckTarget is one contact whose upload outcome is unknown. The lookup
// must use the identity of the SNAPSHOT that was sent — the row may have been
// edited since — and the id is attached at the snapshot's revision so an edit
// made meanwhile stays durably dirty.
type wlRecheckTarget struct {
	qID  int64
	call string
	snap qso.QSO
	rev  int64
}

// wlRecheckLinked reports a contact the delayed recheck found on the server
// and linked locally.
type wlRecheckLinked struct {
	qID      int64
	call     string
	remoteID int64
	// changed: the row was edited after the snapshot reached the server, so
	// its current revision still needs a PATCH.
	changed bool
}

type wlUploadRecheckMsg struct {
	logbook string // logbook ID the contacts belong to; UI updates are scoped to it
	db      *sql.DB
	url     string
	key     string
	sid     string
	attempt int // index into uploadRecheckDelays of the attempt that produced this
	linked  []wlRecheckLinked
	// left are the contacts the server still does not confirm (or could not
	// be asked about). unanswered counts those whose lookup itself failed —
	// a failed lookup is not proof of absence and must not be reported as one.
	left       []wlRecheckTarget
	unanswered int
	// release transfers the database lease to the next attempt or to the
	// caller: the originating database must stay usable for the whole
	// recheck sequence even if the operator switched logbooks meanwhile.
	release func()
}

// queueUploadRecheck returns a tea.Cmd that waits, asks the server whether the
// unknown-outcome uploads landed, links the contacts it finds and reports the
// remainder for another attempt. Nothing is ever written to the server here:
// the contacts were already sent once, so this only reads and links.
//
// release, when non-nil, is the database lease transferred from the upload
// worker whose completion triggered the recheck; it is held across the wait and
// transferred onward with the result.
func (m *Model) queueUploadRecheck(db *sql.DB, url, key, sid, logbook string, targets []wlRecheckTarget, attempt int, release func()) tea.Cmd {
	if db == nil || url == "" || key == "" || len(targets) == 0 || attempt < 0 || attempt >= len(uploadRecheckDelays) {
		if release != nil {
			release()
		}
		return nil
	}
	if release == nil {
		release = func() {}
		if m.App != nil {
			release = m.App.KeepDBAlive(db)
		}
	}
	delay := uploadRecheckDelays[attempt]
	return func() tea.Msg {
		time.Sleep(delay)
		msg := wlUploadRecheckMsg{
			logbook: logbook, db: db, url: url, key: key, sid: sid,
			attempt: attempt, release: release,
		}
		for _, t := range targets {
			row, rerr := store.GetQSOByID(db, t.qID)
			if rerr != nil || row == nil {
				// Logbook switched away and the database is gone, or the row
				// was deleted: there is nothing left to link.
				applog.Warn("Wavelog: recheck skipped — the contact is no longer readable",
					"qso_id", t.qID, "call", t.call, "error", rerr)
				continue
			}
			if row.WavelogID > 0 {
				// Linked meanwhile (batch upload, download, another recheck).
				continue
			}
			rid, asked := lookupRemoteID(url, key, sid, &t.snap)
			if !asked {
				msg.left = append(msg.left, t)
				msg.unanswered++
				continue
			}
			if rid <= 0 {
				msg.left = append(msg.left, t)
				continue
			}
			changed, serr := store.SetWavelogIDChecked(db, t.qID, rid, t.rev)
			if serr != nil {
				applog.Error("Wavelog: recheck could not store the remote id",
					"qso_id", t.qID, "error", serr)
				msg.left = append(msg.left, t)
				continue
			}
			msg.linked = append(msg.linked, wlRecheckLinked{qID: t.qID, call: t.call, remoteID: rid, changed: changed})
		}
		return msg
	}
}

// handleUploadRecheck applies a recheck result: linked contacts are reported as
// confirmed (never as a late failure), and the remainder is either rechecked
// again or left unsent with an honest message.
func (m *Model) handleUploadRecheck(r wlUploadRecheckMsg) tea.Cmd {
	var cmds []tea.Cmd
	currentLogbook := m.App != nil && (r.logbook == "" || r.logbook == m.App.LogbookName)

	for _, l := range r.linked {
		applog.Warn("Wavelog: delayed upload confirmed on the server — contact linked",
			"qso_id", l.qID, "call", l.call, "remote_id", l.remoteID, "attempt", r.attempt+1)
		if l.changed {
			// The server holds the snapshot that was sent, the row has newer
			// edits: push the current revision through the regular chain.
			cmds = append(cmds, m.queueContactReconcile(r.db, r.url, r.key, r.logbook, l.qID, nil, nil))
		}
	}

	if currentLogbook && len(r.linked) > 0 {
		switch len(r.linked) {
		case 1:
			m.toasts.Success(fmt.Sprintf("Wavelog: %s — confirmed on the server", r.linked[0].call))
		default:
			m.toasts.Success(fmt.Sprintf("Wavelog: %d contacts confirmed on the server", len(r.linked)))
		}
		if m.ui.logbookEditor != nil {
			for _, l := range r.linked {
				m.ui.logbookEditor.UpdateWLStatus(l.qID, true, l.remoteID)
				m.ui.logbookEditor.needsReload = true
			}
		}
		m.pushDashboardRecentAndToday()
		m.needRefresh = true
		cmds = append(cmds, m.refreshQSOS())
	}

	if len(r.left) > 0 {
		if next := r.attempt + 1; next < len(uploadRecheckDelays) {
			applog.Info("Wavelog: upload still unconfirmed — checking again",
				"count", len(r.left), "attempt", next+1, "next_in", uploadRecheckDelays[next])
			cmds = append(cmds, m.queueUploadRecheck(r.db, r.url, r.key, r.sid, r.logbook, r.left, next, r.release))
			r.release = nil
		} else {
			applog.Warn("Wavelog: server never confirmed the upload — leaving the contacts unsent",
				"count", len(r.left), "attempts", len(uploadRecheckDelays), "unanswered", r.unanswered)
			if currentLogbook {
				m.toasts.Warn(recheckGiveUpText(r.left, r.unanswered))
				m.needRefresh = true
				cmds = append(cmds, m.refreshQSOS())
				m.notifyRecheckGaveUp(r)
			}
		}
	}

	if r.release != nil {
		r.release()
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// recheckGiveUpText words the end of the recheck sequence. It never claims the
// contacts are absent from the server: an unanswered lookup cannot prove that,
// and a rejected write is the only case that could.
func recheckGiveUpText(left []wlRecheckTarget, unanswered int) string {
	if len(left) == 1 {
		if unanswered > 0 {
			return fmt.Sprintf("Wavelog: %s — still unsent, cannot ask the server", left[0].call)
		}
		return fmt.Sprintf("Wavelog: %s — still unsent, not seen on the server yet", left[0].call)
	}
	if unanswered > 0 {
		return fmt.Sprintf("Wavelog: %d contacts still unsent, cannot ask the server", len(left))
	}
	return fmt.Sprintf("Wavelog: %d contacts still unsent", len(left))
}

// notifyRecheckGaveUp sends the desktop notification for contacts that stayed
// unconfirmed, honouring the "all errors" preference. It runs once the recheck
// sequence is exhausted, so a slow-but-working server never raises an alert.
func (m *Model) notifyRecheckGaveUp(r wlUploadRecheckMsg) {
	if m.App == nil {
		return
	}
	n := m.App.Config.General.Notifications
	if !n.Enabled || !n.AllErrors || !desktopAvailable() {
		return
	}
	applog.Info("Sending Wavelog notification", "calls", len(r.left), "reason", "upload unconfirmed")
	text := fmt.Sprintf("%d QSOs still unsent — Wavelog did not confirm them", len(r.left))
	if len(r.left) == 1 {
		text = fmt.Sprintf("QSO %s still unsent — Wavelog did not confirm it", r.left[0].call)
	}
	if err := beeep.Alert("CQOps — Wavelog", text, ""); err != nil {
		applog.Info("Wavelog notification failed", "error", err.Error())
	}
}
