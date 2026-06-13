package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

// qsoCols is the shared list of QSO columns used across INSERT, SELECT, and UPDATE.
const qsoCols = `call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		wavelog_uploaded`

func InsertQSO(db *sql.DB, q *qso.QSO) (int64, error) {
	now := time.Now().UTC()
	q.CreatedAt = now
	q.UpdatedAt = now

	if q.Source == "" {
		q.Source = "manual"
	}

	res, err := db.Exec(
		`INSERT INTO qsos (` + qsoCols + `, created_at, updated_at)
		VALUES (` + placeholders(36) + `)`,
		q.Call, q.QSODate, q.TimeOn, q.TimeOff,
		q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
		q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
		q.Distance, q.Bearing,
		q.SOTARef, q.POTARef, q.WWFFRef, q.IOTA,
		q.MySOTARef, q.MyPOTARef, q.MyWWFFRef,
		q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
		q.WavelogUploaded,
		q.CreatedAt.Format(time.RFC3339), q.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("insert qso: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}

	q.ID = id
	return id, nil
}

// placeholders returns a string of n comma-separated "?" placeholders.
func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func ListQSOs(db *sql.DB, limit int) ([]qso.QSO, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(
		`SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		wavelog_uploaded,
		created_at, updated_at
		FROM qsos
		ORDER BY id DESC
		LIMIT ?`, limit,
	)
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
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.WavelogUploaded,
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

func GetQSOByID(db *sql.DB, id int64) (*qso.QSO, error) {
	var q qso.QSO
	var createdAt, updatedAt string

	err := db.QueryRow(
		`SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		wavelog_uploaded,
		created_at, updated_at
		FROM qsos WHERE id = ?`, id,
	).Scan(
		&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
		&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
		&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
		&q.Distance, &q.Bearing,
		&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA,
		&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
		&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
		&q.WavelogUploaded,
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

func DeleteQSO(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM qsos WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete qso: %w", err)
	}
	return nil
}

func ListAllQSOs(db *sql.DB) ([]qso.QSO, error) {
	return ListQSOs(db, 0)
}

func UpdateQSO(db *sql.DB, q *qso.QSO) error {
	q.UpdatedAt = time.Now().UTC()
	_, err := db.Exec(
		`UPDATE qsos SET call=?, qso_date=?, time_on=?, time_off=?, band=?, freq=?, freq_rx=?, mode=?, submode=?,
		rst_sent=?, rst_rcvd=?, gridsquare=?, name=?, qth=?, country=?, comment=?, notes=?, tx_pwr=?,
		distance=?, bearing=?,
		sota_ref=?, pota_ref=?, wwff_ref=?, iota=?,
		my_sota_ref=?, my_pota_ref=?, my_wwff_ref=?,
		station_callsign=?, operator=?, my_gridsquare=?, my_rig=?, my_antenna=?, source=?,
		wavelog_uploaded=?,
		updated_at=?
		WHERE id=?`,
		q.Call, q.QSODate, q.TimeOn, q.TimeOff,
		q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
		q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
		q.Distance, q.Bearing,
		q.SOTARef, q.POTARef, q.WWFFRef, q.IOTA,
		q.MySOTARef, q.MyPOTARef, q.MyWWFFRef,
		q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
		q.WavelogUploaded,
		q.UpdatedAt.Format(time.RFC3339),
		q.ID,
	)
	if err != nil {
		return fmt.Errorf("update qso: %w", err)
	}
	return nil
}

func PurgeQSOs(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM qsos`)
	if err != nil {
		return fmt.Errorf("purge qsos: %w", err)
	}
	return nil
}

// UpdateWavelogStatus sets the wavelog_uploaded status for a QSO.
func UpdateWavelogStatus(db *sql.DB, id int64, status string) error {
	_, err := db.Exec(`UPDATE qsos SET wavelog_uploaded=? WHERE id=?`, status, id)
	if err != nil {
		return fmt.Errorf("update wavelog status: %w", err)
	}
	return nil
}

// NormalizeStationFields updates station_callsign, operator and my_gridsquare
// for a set of QSOs. Used before batch upload to Wavelog to align QSOs with
// the target station profile.
func NormalizeStationFields(db *sql.DB, ids []int64, stationCall, operator, grid string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE qsos SET station_callsign=?, operator=?, my_gridsquare=?, updated_at=? WHERE id=?`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, id := range ids {
		if _, err := stmt.Exec(stationCall, operator, grid, now, id); err != nil {
			return fmt.Errorf("update qso %d: %w", id, err)
		}
	}
	return tx.Commit()
}
