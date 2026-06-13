package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

func InsertQSO(db *sql.DB, q *qso.QSO) (int64, error) {
	now := time.Now().UTC()
	q.CreatedAt = now
	q.UpdatedAt = now

	if q.Source == "" {
		q.Source = "manual"
	}

	res, err := db.Exec(
		`INSERT INTO qsos (call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		q.Call, q.QSODate, q.TimeOn, q.TimeOff,
		q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
		q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
		q.Distance, q.Bearing,
		q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
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

func ListQSOs(db *sql.DB, limit int) ([]qso.QSO, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(
		`SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
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
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
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
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		created_at, updated_at
		FROM qsos WHERE id = ?`, id,
	).Scan(
		&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
		&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
		&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
		&q.Distance, &q.Bearing,
		&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
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
		station_callsign=?, operator=?, my_gridsquare=?, my_rig=?, my_antenna=?, source=?,
		updated_at=?
		WHERE id=?`,
		q.Call, q.QSODate, q.TimeOn, q.TimeOff,
		q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
		q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
		q.Distance, q.Bearing,
		q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
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
