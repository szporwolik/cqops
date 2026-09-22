package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

// qsoCols is the shared list of QSO columns used across INSERT, SELECT, and UPDATE.
const qsoCols = `call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		dxcc`

// qsoSelectCols is the SELECT-side column list (id + data columns +
// timestamps) shared by the list queries.
const qsoSelectCols = `id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		dxcc,
		created_at, updated_at`

// placeholders52 is a pre-computed string of 52 comma-separated "?" markers,
// used by InsertQSO to avoid a per-insert []string allocation.
const placeholders52 = "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?"

// InsertQSO persists a QSO and sets its ID on success. Retries on SQLITE_BUSY.
func InsertQSO(db *sql.DB, q *qso.QSO) (int64, error) {
	now := time.Now().UTC()
	q.CreatedAt = now
	q.UpdatedAt = now

	if q.Source == "" {
		q.Source = "manual"
	}

	var id int64
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		var res sql.Result
		res, err = db.Exec(
			`INSERT INTO qsos (`+qsoCols+`, base_call, created_at, updated_at)
			VALUES (`+placeholders52+`, ?, ?, ?)`,
			q.Call, q.QSODate, q.TimeOn, q.TimeOff,
			q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
			q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
			q.Distance, q.Bearing,
			q.SOTARef, q.POTARef, q.WWFFRef, q.IOTA, q.SIG, q.SIGInfo,
			q.MySOTARef, q.MyPOTARef, q.MyWWFFRef,
			q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
			q.CQZone, q.ITUZone, q.MyCQZone, q.MyITUZone, q.MyDXCC, q.MySIG, q.MySIGInfo, q.WavelogID, q.ContestID, q.ExchSent, q.ExchRcvd, q.STX, q.SRX, q.STXString, q.SRXString, q.ContestADIFID, q.DXCC,
			qso.DeriveBaseCall(q.Call),
			q.CreatedAt.Format(time.RFC3339), q.UpdatedAt.Format(time.RFC3339),
		)
		if err == nil {
			id, err = res.LastInsertId()
			if err == nil {
				q.ID = id
				return id, nil
			}
			return 0, fmt.Errorf("last insert id: %w", err)
		}
		if !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, fmt.Errorf("insert qso: %w", err)
}

// ListQSOs returns recent QSOs ordered by QSO date/time descending.
// If contestID is non-empty, only QSOs matching that contest are returned.
func ListQSOs(db *sql.DB, limit int, contestID string) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos`
	var args []any
	if contestID != "" {
		query += ` WHERE contest_id = ? OR contest_adif_id = ?`
		args = append(args, contestID, contestID)
	}
	query += ` ORDER BY qso_date DESC, time_on DESC, id DESC`
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = db.Query(query+" LIMIT ?", append(args, limit)...)
	} else {
		rows, err = db.Query(query, args...)
	}
	if err != nil {
		return nil, fmt.Errorf("list qsos: %w", err)
	}
	defer rows.Close()

	var qsos []qso.QSO
	for rows.Next() {
		var q qso.QSO
		var createdAt, updatedAt string
		err := rows.Scan(
			&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
			&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
			&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
			&q.Distance, &q.Bearing,
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG, &q.SIGInfo,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogID, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan qso: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			q.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			q.UpdatedAt = t
		}
		qsos = append(qsos, q)
	}

	return qsos, rows.Err()
}

// LastContestExchange returns the exchange fields from the most recent QSO
// with the given callsign in the given contest. Used to prefill the received
// exchange when working the same station on another band/mode.
func LastContestExchange(db *sql.DB, call, contestID string) (exchRcvd, rstRcvd, exchSent string, err error) {
	query := `SELECT exch_rcvd, rst_rcvd, exch_sent FROM qsos
		WHERE call = ? AND (contest_id = ? OR contest_adif_id = ?)
		ORDER BY qso_date DESC, time_on DESC LIMIT 1`
	err = db.QueryRow(query, call, contestID, contestID).Scan(&exchRcvd, &rstRcvd, &exchSent)
	if err != nil {
		return "", "", "", err
	}
	return exchRcvd, rstRcvd, exchSent, nil
}

// ListQSOsPageWithCount returns a page of QSOs and the total count in a single
// query using COUNT(*) OVER(), avoiding the need for a separate CountQSOs call.
func ListQSOsPageWithCount(db *sql.DB, limit, offset int, contestID string) ([]qso.QSO, int, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone, dxcc,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at,
		COUNT(*) OVER() as total_count
		FROM qsos`
	var args []any
	if contestID != "" {
		query += ` WHERE contest_id = ? OR contest_adif_id = ?`
		args = append(args, contestID, contestID)
	}
	query += `
		ORDER BY qso_date DESC, time_on DESC, id DESC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list qsos page with count: %w", err)
	}
	defer rows.Close()

	var qsos []qso.QSO
	total := 0
	for rows.Next() {
		var q qso.QSO
		var createdAt, updatedAt string
		err := rows.Scan(
			&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
			&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
			&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
			&q.Distance, &q.Bearing,
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG, &q.SIGInfo,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone, &q.DXCC,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogID, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
			&createdAt, &updatedAt,
			&total,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan qso with count: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			q.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			q.UpdatedAt = t
		}
		qsos = append(qsos, q)
	}

	return qsos, total, rows.Err()
}

// ListQSOsPage returns a page of QSOs ordered by QSO date/time descending.
// If contestID is non-empty, only QSOs matching that contest are returned.
// When orderAsc is true, the order is chronological (ASC) — used for contest
// ADIF export so the output maps directly to Cabrillo.
func ListQSOsPage(db *sql.DB, limit, offset int, contestID string, orderAsc bool) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone, dxcc,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos`
	var args []any
	if contestID != "" {
		query += ` WHERE contest_id = ? OR contest_adif_id = ?`
		args = append(args, contestID, contestID)
	}
	query += `
		ORDER BY qso_date `
	if orderAsc {
		query += `ASC, time_on ASC, id ASC`
	} else {
		query += `DESC, time_on DESC, id DESC`
	}
	query += `
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list qsos page: %w", err)
	}
	defer rows.Close()

	var qsos []qso.QSO
	for rows.Next() {
		var q qso.QSO
		var createdAt, updatedAt string
		err := rows.Scan(
			&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
			&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
			&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
			&q.Distance, &q.Bearing,
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG, &q.SIGInfo,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone, &q.DXCC,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogID, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan qso: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			q.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			q.UpdatedAt = t
		}
		qsos = append(qsos, q)
	}

	return qsos, rows.Err()
}

// SearchQSOsByCall returns QSOs matching a callsign by base callsign match.
// Uses the indexed base_call column (extracted core callsign) for fast lookups.
// Matches: "SP9MOA", "SP9MOA/P", "9A/SP9MOA", "9A/SP9MOA/P" all via base_call="SP9MOA".
func SearchQSOsByCall(db *sql.DB, call string, limit int) ([]qso.QSO, error) {
	baseCall := qso.DeriveBaseCall(call)
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos
		WHERE base_call = ?
		ORDER BY id DESC
		LIMIT ?`

	rows, err := db.Query(query, baseCall, limit)
	if err != nil {
		return nil, fmt.Errorf("search qsos by call: %w", err)
	}
	defer rows.Close()

	var qsos []qso.QSO
	for rows.Next() {
		var q qso.QSO
		var createdAt, updatedAt string
		err := rows.Scan(
			&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
			&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
			&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
			&q.Distance, &q.Bearing,
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG, &q.SIGInfo,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogID, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan qso: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			q.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			q.UpdatedAt = t
		}
		qsos = append(qsos, q)
	}
	return qsos, rows.Err()
}

// SearchQSOs returns QSOs whose callsign, name, or country contains the
// query, optionally scoped to a contest. Newest first, capped at limit.
// Used by the logbook editor search so results cover the whole logbook,
// not just the currently displayed page.
func SearchQSOs(db *sql.DB, query, contestID string, limit int) ([]qso.QSO, error) {
	like := "%" + query + "%"
	sql := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos
		WHERE (call LIKE ? OR name LIKE ? OR country LIKE ?)`
	var args []any = []any{like, like, like}
	if contestID != "" {
		sql += ` AND (contest_id = ? OR contest_adif_id = ?)`
		args = append(args, contestID, contestID)
	}
	sql += `
		ORDER BY qso_date DESC, time_on DESC, id DESC
		LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("search qsos: %w", err)
	}
	defer rows.Close()

	var qsos []qso.QSO
	for rows.Next() {
		var q qso.QSO
		var createdAt, updatedAt string
		err := rows.Scan(
			&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
			&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
			&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
			&q.Distance, &q.Bearing,
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG, &q.SIGInfo,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogID, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan qso: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			q.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			q.UpdatedAt = t
		}
		qsos = append(qsos, q)
	}
	return qsos, rows.Err()
}

// GetQSOByID returns a single QSO by primary key.
func GetQSOByID(db *sql.DB, id int64) (*qso.QSO, error) {
	var q qso.QSO
	var createdAt, updatedAt string

	err := db.QueryRow(
		`SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone, dxcc,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, wavelog_dirty, wavelog_dirty_rev, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos WHERE id = ?`, id,
	).Scan(
		&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
		&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
		&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
		&q.Distance, &q.Bearing,
		&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG, &q.SIGInfo,
		&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
		&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
		&q.CQZone, &q.ITUZone, &q.DXCC,
		&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
		&q.MySIG, &q.MySIGInfo,
		&q.WavelogID, &q.WavelogDirty, &q.WavelogDirtyRev, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get qso by id: %w", err)
	}

	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		q.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		q.UpdatedAt = t
	}

	return &q, nil
}

// DeleteQSO removes a QSO by primary key.
func DeleteQSO(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM qsos WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete qso: %w", err)
	}
	return nil
}

// FindQSOByKey returns the ID of a QSO matching call, band, mode, date and time_on.
// Returns 0 if no match is found.
func FindQSOByKey(db *sql.DB, call, band, mode, qsoDate, timeOn string) int64 {
	var id int64
	err := db.QueryRow(
		`SELECT id FROM qsos WHERE call = ? AND band = ? AND mode = ? AND qso_date = ? AND time_on = ? LIMIT 1`,
		call, band, mode, qsoDate, timeOn,
	).Scan(&id)
	if err != nil {
		return 0
	}
	return id
}

// DupeCheckResult holds the reference fields of a potential duplicate QSO.
type DupeCheckResult struct {
	ID   int64
	SOTA string
	POTA string
	WWFF string
	IOTA string
}

// IsDuplicateQSO returns (true, nil) if a QSO with the same call, band, mode
// and date already exists. If a match is found but has different reference
// data (SOTA/POTA/WWFF/IOTA), it is not considered a duplicate — the result
// will be (false, &result) with the existing QSO's ref fields for comparison.
func IsDuplicateQSO(db *sql.DB, call, band, mode, qsoDate string) (bool, *DupeCheckResult) {
	var r DupeCheckResult
	// Match by normalized call OR base_call (portable calls like DL/SP9MOA/P
	// match the base SP9MOA). The normalised form is used first as it's indexed;
	// the base_call fallback handles portable-to-base dupe detection.
	baseCall := qso.DeriveBaseCall(call)
	var err error
	if baseCall != "" && baseCall != qso.NormalizeCall(call) {
		err = db.QueryRow(
			`SELECT id, sota_ref, pota_ref, wwff_ref, iota FROM qsos
			 WHERE (call = ? OR base_call = ?) AND band = ? AND mode = ? AND qso_date = ?
			 LIMIT 1`,
			qso.NormalizeCall(call), baseCall, band, mode, qsoDate,
		).Scan(&r.ID, &r.SOTA, &r.POTA, &r.WWFF, &r.IOTA)
	} else {
		err = db.QueryRow(
			`SELECT id, sota_ref, pota_ref, wwff_ref, iota FROM qsos
			 WHERE call = ? AND band = ? AND mode = ? AND qso_date = ?
			 LIMIT 1`,
			qso.NormalizeCall(call), band, mode, qsoDate,
		).Scan(&r.ID, &r.SOTA, &r.POTA, &r.WWFF, &r.IOTA)
	}
	if err != nil {
		return false, nil
	}
	return true, &r
}

// WorkedCallsOnBandDate returns a set of (call,mode) pairs for QSOs
// logged on the given band and date. Used by the DXC path line to mark
// already-worked spots as dupes.
func WorkedCallsOnBandDate(db *sql.DB, band, qsoDate string) (map[string]bool, error) {
	rows, err := db.Query(
		`SELECT DISTINCT call, mode FROM qsos WHERE band = ? AND qso_date = ?`,
		band, qsoDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	worked := make(map[string]bool)
	for rows.Next() {
		var call, mode string
		if err := rows.Scan(&call, &mode); err != nil {
			continue
		}
		// Key: normalized call|mode so spot-mode (USB) matches logged-mode (SSB).
		worked[qso.NormalizeCall(call)+"|"+qso.NormalizeRigMode(mode)] = true
	}
	return worked, rows.Err()
}

// WorkedCallsInContest returns a set of (call,mode) pairs already logged
// on the given band within a specific contest (48h+ events span multiple
// dates, so the date filter from WorkedCallsOnBandDate is insufficient).
func WorkedCallsInContest(db *sql.DB, contestID, band string) (map[string]bool, error) {
	rows, err := db.Query(
		`SELECT DISTINCT call, mode FROM qsos WHERE (contest_id = ? OR contest_adif_id = ?) AND band = ?`,
		contestID, contestID, band,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	worked := make(map[string]bool)
	for rows.Next() {
		var call, mode string
		if err := rows.Scan(&call, &mode); err != nil {
			continue
		}
		worked[qso.NormalizeCall(call)+"|"+qso.NormalizeRigMode(mode)] = true
	}
	return worked, rows.Err()
}

// DXCDupeSet returns a set of (call,band,mode) triples for QSOs that
// mark DXC spots as dupes. Outside contests the scope is today's date;
// inside contests it spans the entire contest (48h+). Key format is
// "CALL|BAND|MODE" — the caller checks each spot's (call,band,mode)
// against this set with zero per-spot DB queries.

// DXCDXCCWorkedSets returns two maps for DXC spot highlighting:
//   - dxccBand: map["230|20m"]bool — DXCC+Band already worked (all-time).
//   - dxccBandMode: map["230|20m|FT8"]bool — DXCC+Band+Mode already worked (all-time).
//
// Used to colour new-DXCC/new-band/new-mode spots blue in the DXC table.
// Scoped to the entire logbook — rebuilt on logbook/profile change, not per-day.
// Runs as a single query per table rebuild with zero per-spot DB queries.
func DXCDXCCWorkedSets(db *sql.DB) (dxccBand, dxccBandMode map[string]bool, err error) {
	rows, err := db.Query(`SELECT DISTINCT dxcc, band, mode FROM qsos WHERE dxcc != ''`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	dxccBand = make(map[string]bool, 256)
	dxccBandMode = make(map[string]bool, 256)
	for rows.Next() {
		var dxcc, band, mode string
		if err := rows.Scan(&dxcc, &band, &mode); err != nil {
			continue
		}
		dxcc = strings.TrimSpace(dxcc)
		band = strings.TrimSpace(band)
		mode = strings.TrimSpace(mode)
		if dxcc == "" {
			continue
		}
		bk := dxcc + "|" + band
		dxccBand[bk] = true
		if mode != "" {
			dxccBandMode[bk+"|"+mode] = true
		}
	}
	return dxccBand, dxccBandMode, rows.Err()
}
func DXCDupeSet(db *sql.DB, qsoDate, contestID string) (map[string]bool, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if contestID != "" {
		rows, err = db.Query(
			`SELECT DISTINCT call, band, mode FROM qsos WHERE contest_id = ? OR contest_adif_id = ?`,
			contestID, contestID,
		)
	} else {
		rows, err = db.Query(
			`SELECT DISTINCT call, band, mode FROM qsos WHERE qso_date = ?`,
			qsoDate,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	worked := make(map[string]bool, 256)
	for rows.Next() {
		var call, band, mode string
		if err := rows.Scan(&call, &band, &mode); err != nil {
			continue
		}
		key := qso.NormalizeCall(call) + "|" + qso.NormalizeBand(band) + "|" + qso.NormalizeRigMode(mode)
		worked[key] = true
	}
	return worked, rows.Err()
}

// ListAllQSOs returns all QSOs ordered by id DESC. Uses keyset pagination
// so the scan cost stays linear in the log size — OFFSET pagination degrades
// quadratically on large tables.
func ListAllQSOs(db *sql.DB) ([]qso.QSO, error) {
	const pageSize = 500
	var all []qso.QSO
	lastID := int64(math.MaxInt64)
	for {
		page, err := listQSOsByQuery(db,
			`SELECT `+qsoSelectCols+` FROM qsos WHERE id < ? ORDER BY id DESC LIMIT ?`,
			lastID, pageSize)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
		lastID = page[len(page)-1].ID
	}
	return all, nil
}

// CountUnsentQSOs returns the number of QSOs without a remote id — a single
// SQL COUNT so upload preparation never walks the whole logbook.
func CountUnsentQSOs(db *sql.DB) (int, error) {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos WHERE COALESCE(wavelog_id, 0) = 0`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count unsent qsos: %w", err)
	}
	return n, nil
}

// ListUnsentQSOs returns the QSOs without a remote id (never uploaded),
// ordered by id DESC. Only the eligible rows are fetched, so preparing an
// upload stays cheap even for very large historical logs.
func ListUnsentQSOs(db *sql.DB) ([]qso.QSO, error) {
	return listQSOsByQuery(db,
		`SELECT `+qsoSelectCols+` FROM qsos WHERE COALESCE(wavelog_id, 0) = 0 ORDER BY id DESC`)
}

// ListQSOsPageAfterTx returns the next page of QSOs strictly after the
// cursor row, ordered by qso_date/time_on/id in the same order ListQSOsPage
// uses. A nil cursor returns the first page. Reads run on the caller's
// transaction, so a streaming export sees one consistent snapshot: a
// concurrently inserted QSO can never shift the pages and duplicate or omit
// records.
func ListQSOsPageAfterTx(tx *sql.Tx, limit int, contestID string, orderAsc bool, cursor *qso.QSO) ([]qso.QSO, error) {
	var b strings.Builder
	b.WriteString(`SELECT ` + qsoSelectCols + ` FROM qsos`)
	var args []any
	var conds []string
	if contestID != "" {
		conds = append(conds, `(contest_id = ? OR contest_adif_id = ?)`)
		args = append(args, contestID, contestID)
	}
	if cursor != nil {
		if orderAsc {
			conds = append(conds,
				`((qso_date > ?) OR (qso_date = ? AND time_on > ?) OR (qso_date = ? AND time_on = ? AND id > ?))`)
		} else {
			conds = append(conds,
				`((qso_date < ?) OR (qso_date = ? AND time_on < ?) OR (qso_date = ? AND time_on = ? AND id < ?))`)
		}
		args = append(args,
			cursor.QSODate, cursor.QSODate, cursor.TimeOn,
			cursor.QSODate, cursor.TimeOn, cursor.ID)
	}
	if len(conds) > 0 {
		b.WriteString(` WHERE ` + strings.Join(conds, ` AND `))
	}
	b.WriteString(` ORDER BY qso_date `)
	if orderAsc {
		b.WriteString(`ASC, time_on ASC, id ASC`)
	} else {
		b.WriteString(`DESC, time_on DESC, id DESC`)
	}
	b.WriteString(` LIMIT ?`)
	args = append(args, limit)
	return listQSOsByQuery(tx, b.String(), args...)
}

// ExportQSOsSnapshot streams all QSOs (optionally contest-filtered) through
// fn inside a single read-only transaction. Pages advance by keyset cursor,
// so concurrent inserts — e.g. WSJT-X logging a QSO mid-export — cannot
// shift offsets; every record appears exactly once. The callback is invoked
// once per QSO in export order; returning an error aborts the stream.
func ExportQSOsSnapshot(db *sql.DB, contestID string, orderAsc bool, fn func(q qso.QSO) error) error {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin export snapshot: %w", err)
	}
	defer tx.Rollback()

	const pageSize = 500
	var cursor *qso.QSO
	for {
		rows, err := ListQSOsPageAfterTx(tx, pageSize, contestID, orderAsc, cursor)
		if err != nil {
			return fmt.Errorf("export page: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		for i := range rows {
			if err := fn(rows[i]); err != nil {
				return err
			}
		}
		cursor = &rows[len(rows)-1]
		if len(rows) < pageSize {
			break
		}
	}
	return tx.Commit()
}

// ListQSOsFromDate returns QSOs with qso_date >= the given date, newest-first,
// capped at limit. Used by the dashboard to avoid loading unbounded history.
func ListQSOsFromDate(db *sql.DB, date string, limit int) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig, sig_info,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_id, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos WHERE qso_date >= ? ORDER BY qso_date DESC, time_on DESC, id DESC LIMIT ?`
	return listQSOsByQuery(db, query, date, limit)
}

// qsoQueryer abstracts *sql.DB and *sql.Tx for the shared row-scanning
// helpers. Both satisfy it.
type qsoQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// listQSOsByQuery is a helper that scans QSOs from a parameterized query.
func listQSOsByQuery(q qsoQueryer, query string, args ...any) ([]qso.QSO, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list qsos: %w", err)
	}
	defer rows.Close()
	var qsos []qso.QSO
	for rows.Next() {
		var q qso.QSO
		var createdAt, updatedAt string
		err := rows.Scan(
			&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
			&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
			&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
			&q.Distance, &q.Bearing,
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG, &q.SIGInfo,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogID, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
			&q.DXCC,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan qso: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			q.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			q.UpdatedAt = t
		}
		qsos = append(qsos, q)
	}
	return qsos, rows.Err()
}

// UpdateQSO updates an existing QSO. Retries on SQLITE_BUSY.
//
// buildUpdateQSOStatement renders the QSO row UPDATE with the caller's extra
// SET fragment appended before updated_at. SaveQSOForSync uses it to set the
// pending-sync flag and bump its revision in the same statement as the edit,
// so the row changes and the dirty state change together or not at all.
//
// Derived-field policy: base_call is always recomputed from the stored call.
// DXCC is written when the caller supplies a value; when the call CHANGED
// and no fresh DXCC was provided, the stale prefix-derived value is
// invalidated (cleared) so the DXCC backfill recomputes it — preserving it
// would leave worked-DXCC statistics disagreeing with the displayed contact.
// oldCall is the call stored before this update ("" when unknown).
func buildUpdateQSOStatement(q *qso.QSO, oldCall, extraSet string) (string, []any) {
	q.UpdatedAt = time.Now().UTC()
	callChanged := oldCall != "" && !strings.EqualFold(oldCall, q.Call)

	dxccSet := ""
	dxccArg := ""
	if q.DXCC != "" {
		dxccSet = ", dxcc=?"
		dxccArg = q.DXCC
	} else if callChanged {
		dxccSet = ", dxcc=?"
	}

	query := `UPDATE qsos SET call=?, qso_date=?, time_on=?, time_off=?, band=?, freq=?, freq_rx=?, mode=?, submode=?,
		rst_sent=?, rst_rcvd=?, gridsquare=?, name=?, qth=?, country=?, comment=?, notes=?, tx_pwr=?,
		distance=?, bearing=?,
	sota_ref=?, pota_ref=?, wwff_ref=?, iota=?, sig=?, sig_info=?,
		my_sota_ref=?, my_pota_ref=?, my_wwff_ref=?,
		station_callsign=?, operator=?, my_gridsquare=?, my_rig=?, my_antenna=?, source=?,
		cq_zone=?, itu_zone=?,
		my_cq_zone=?, my_itu_zone=?, my_dxcc=?,
		my_sig=?, my_sig_info=?,
		wavelog_id=?, contest_id=?, exch_sent=?, exch_rcvd=?, stx=?, srx=?, stx_string=?, srx_string=?, contest_adif_id=?,
		base_call=?` + dxccSet + extraSet + `,
		updated_at=?
		WHERE id=?`

	args := []any{
		q.Call, q.QSODate, q.TimeOn, q.TimeOff,
		q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
		q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
		q.Distance, q.Bearing,
		q.SOTARef, q.POTARef, q.WWFFRef, q.IOTA, q.SIG, q.SIGInfo,
		q.MySOTARef, q.MyPOTARef, q.MyWWFFRef,
		q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
		q.CQZone, q.ITUZone, q.MyCQZone, q.MyITUZone, q.MyDXCC, q.MySIG, q.MySIGInfo,
		q.WavelogID, q.ContestID, q.ExchSent, q.ExchRcvd, q.STX, q.SRX, q.STXString, q.SRXString, q.ContestADIFID,
		qso.DeriveBaseCall(q.Call),
	}
	if dxccSet != "" {
		args = append(args, dxccArg)
	}
	args = append(args, q.UpdatedAt.Format(time.RFC3339), q.ID)
	return query, args
}

// UpdateQSO persists an edited QSO. Every local write bumps the pending-sync
// revision (wavelog_dirty_rev), including non-synced rows: an initial upload
// in flight captures an older revision, and the bumped counter lets the
// upload completion detect that the row changed and keep it pending.
func UpdateQSO(db *sql.DB, q *qso.QSO) error {
	var oldCall string
	_ = db.QueryRow(`SELECT call FROM qsos WHERE id=?`, q.ID).Scan(&oldCall)

	query, args := buildUpdateQSOStatement(q, oldCall, ", wavelog_dirty_rev=wavelog_dirty_rev+1")

	var err error
	for attempt := 0; attempt < 3; attempt++ {
		_, err = db.Exec(query, args...)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("update qso: %w", err)
}

// SaveQSOForSync persists an edited QSO and durably marks it as having a
// pending remote update in ONE transaction, bumping the pending-sync
// revision. The returned revision identifies this exact edit — the dirty
// flag may only be cleared by an acknowledgement for the same revision (see
// ClearWavelogDirtyIfRevision). A crash between the local write and the
// PATCH therefore leaves a dirty row that a remote refresh will not
// overwrite, never a changed row that falsely looks synced.
func SaveQSOForSync(db *sql.DB, q *qso.QSO) (int64, error) {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, fmt.Errorf("begin save for sync: %w", err)
	}
	defer tx.Rollback()

	var oldCall string
	_ = tx.QueryRow(`SELECT call FROM qsos WHERE id=?`, q.ID).Scan(&oldCall)

	query, args := buildUpdateQSOStatement(q, oldCall, ", wavelog_dirty=1, wavelog_dirty_rev=wavelog_dirty_rev+1")
	if _, err := tx.Exec(query, args...); err != nil {
		return 0, fmt.Errorf("update qso for sync: %w", err)
	}
	// Read the bumped revision inside the same transaction — the write
	// lock is held until commit, so this is exactly our edit's revision.
	var rev int64
	if err := tx.QueryRow(`SELECT wavelog_dirty_rev FROM qsos WHERE id=?`, q.ID).Scan(&rev); err != nil {
		return 0, fmt.Errorf("read sync revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit save for sync: %w", err)
	}
	return rev, nil
}

// PurgeQSOs deletes all QSOs from the database.
func PurgeQSOs(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM qsos`)
	if err != nil {
		return fmt.Errorf("purge qsos: %w", err)
	}
	return nil
}

// EnrichmentData holds callbook-derived fields for non-destructive QSO enrichment.
// Only non-empty fields are applied; existing data is never overwritten.
type EnrichmentData struct {
	Name       string
	QTH        string
	Country    string
	GridSquare string
	IOTA       string
	CQZone     string
	ITUZone    string
	DXCC       string
}

// UpdateQSOEnrichment applies callbook enrichment to a QSO.
// Only fields that are currently empty in the database are updated —
// existing data is never overwritten by enrichment.
func UpdateQSOEnrichment(db *sql.DB, qsoID int64, e EnrichmentData) error {
	if e.Name == "" && e.QTH == "" && e.Country == "" && e.GridSquare == "" && e.IOTA == "" && e.CQZone == "" && e.ITUZone == "" && e.DXCC == "" {
		return nil
	}

	var sets []string
	var args []interface{}

	if e.Name != "" {
		sets = append(sets, "name = CASE WHEN COALESCE(name,'') = '' THEN ? ELSE name END")
		args = append(args, e.Name)
	}
	if e.QTH != "" {
		sets = append(sets, "qth = CASE WHEN COALESCE(qth,'') = '' THEN ? ELSE qth END")
		args = append(args, e.QTH)
	}
	if e.Country != "" {
		sets = append(sets, "country = CASE WHEN COALESCE(country,'') = '' THEN ? ELSE country END")
		args = append(args, e.Country)
	}
	if e.GridSquare != "" {
		sets = append(sets, "gridsquare = CASE WHEN COALESCE(gridsquare,'') = '' THEN ? ELSE gridsquare END")
		args = append(args, e.GridSquare)
	}
	if e.IOTA != "" {
		sets = append(sets, "iota = CASE WHEN COALESCE(iota,'') = '' THEN ? ELSE iota END")
		args = append(args, e.IOTA)
	}
	if e.CQZone != "" {
		sets = append(sets, "cq_zone = CASE WHEN COALESCE(cq_zone,'') = '' THEN ? ELSE cq_zone END")
		args = append(args, e.CQZone)
	}
	if e.ITUZone != "" {
		sets = append(sets, "itu_zone = CASE WHEN COALESCE(itu_zone,'') = '' THEN ? ELSE itu_zone END")
		args = append(args, e.ITUZone)
	}
	if e.DXCC != "" {
		sets = append(sets, "dxcc = CASE WHEN COALESCE(dxcc,'') = '' THEN ? ELSE dxcc END")
		args = append(args, e.DXCC)
	}

	if len(sets) == 0 {
		return nil
	}

	args = append(args, qsoID)
	query := fmt.Sprintf("UPDATE qsos SET %s WHERE id = ?", strings.Join(sets, ", "))
	if _, err := db.Exec(query, args...); err != nil {
		return fmt.Errorf("update qso enrichment: %w", err)
	}
	return nil
}
