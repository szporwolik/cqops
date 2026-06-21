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
		sota_ref, pota_ref, wwff_ref, iota, sig,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_uploaded, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id`

// placeholders returns a string of n comma-separated "?" placeholders.
func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

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
			`INSERT INTO qsos (`+qsoCols+`, created_at, updated_at)
			VALUES (`+placeholders(52)+`)`,
			q.Call, q.QSODate, q.TimeOn, q.TimeOff,
			q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
			q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
			q.Distance, q.Bearing,
			q.SOTARef, q.POTARef, q.WWFFRef, q.IOTA, q.SIG,
			q.MySOTARef, q.MyPOTARef, q.MyWWFFRef,
			q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
			q.CQZone, q.ITUZone, q.MyCQZone, q.MyITUZone, q.MyDXCC, q.MySIG, q.MySIGInfo, q.WavelogUploaded, q.ContestID, q.ExchSent, q.ExchRcvd, q.STX, q.SRX, q.STXString, q.SRXString, q.ContestADIFID,
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
// only QSOs matching that contest are returned.
func ListQSOs(db *sql.DB, limit int, contestID string) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_uploaded, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos`
	var args []any
	if contestID != "" {
		query += ` WHERE contest_id = ?`
		args = append(args, contestID)
	}
	query += ` ORDER BY qso_date DESC, time_on DESC`
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
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogUploaded, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
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

// ListQSOsPage returns a page of QSOs ordered by QSO date/time descending.
// If contestID is non-empty, only QSOs matching that contest are returned.
func ListQSOsPage(db *sql.DB, limit, offset int, contestID string) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_uploaded, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos`
	var args []any
	if contestID != "" {
		query += ` WHERE contest_id = ?`
		args = append(args, contestID)
	}
	query += `
		ORDER BY qso_date DESC, time_on DESC
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
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogUploaded, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
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

// SearchQSOsByCall returns QSOs matching a callsign by exact or prefix match.
// "SP9SPM" matches "SP9SPM", "SP9SPM/P", "9A/SP9SPM", "9A/SP9SPM/P", etc.
func SearchQSOsByCall(db *sql.DB, call string, limit int) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota, sig,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_uploaded, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos
		WHERE call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ?
		ORDER BY id DESC
		LIMIT ?`

	rows, err := db.Query(query, call, call+"/%", "%/"+call, "%/"+call+"/%", limit)
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
			&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG,
			&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
			&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
			&q.CQZone, &q.ITUZone,
			&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
			&q.MySIG, &q.MySIGInfo,
			&q.WavelogUploaded, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
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
		sota_ref, pota_ref, wwff_ref, iota, sig,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		cq_zone, itu_zone,
		my_cq_zone, my_itu_zone, my_dxcc,
		my_sig, my_sig_info,
		wavelog_uploaded, contest_id, exch_sent, exch_rcvd, stx, srx, stx_string, srx_string, contest_adif_id,
		created_at, updated_at
		FROM qsos WHERE id = ?`, id,
	).Scan(
		&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &q.TimeOff,
		&q.Band, &q.Freq, &q.FreqRx, &q.Mode, &q.Submode,
		&q.RSTSent, &q.RSTRcvd, &q.GridSquare, &q.Name, &q.QTH, &q.Country, &q.Comment, &q.Notes, &q.TXPower,
		&q.Distance, &q.Bearing,
		&q.SOTARef, &q.POTARef, &q.WWFFRef, &q.IOTA, &q.SIG,
		&q.MySOTARef, &q.MyPOTARef, &q.MyWWFFRef,
		&q.StationCallsign, &q.Operator, &q.MyGridSquare, &q.MyRig, &q.MyAntenna, &q.Source,
		&q.CQZone, &q.ITUZone,
		&q.MyCQZone, &q.MyITUZone, &q.MyDXCC,
		&q.MySIG, &q.MySIGInfo,
		&q.WavelogUploaded, &q.ContestID, &q.ExchSent, &q.ExchRcvd, &q.STX, &q.SRX, &q.STXString, &q.SRXString, &q.ContestADIFID,
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
	err := db.QueryRow(
		`SELECT id, sota_ref, pota_ref, wwff_ref, iota FROM qsos
		 WHERE call = ? AND band = ? AND mode = ? AND qso_date = ?
		 LIMIT 1`,
		call, band, mode, qsoDate,
	).Scan(&r.ID, &r.SOTA, &r.POTA, &r.WWFF, &r.IOTA)
	if err != nil {
		return false, nil
	}
	return true, &r
}

// ListAllQSOs returns all QSOs ordered by id DESC.
func ListAllQSOs(db *sql.DB) ([]qso.QSO, error) {
	return ListQSOs(db, 0, "")
}

// UpdateQSO updates an existing QSO. Retries on SQLITE_BUSY.
func UpdateQSO(db *sql.DB, q *qso.QSO) error {
	q.UpdatedAt = time.Now().UTC()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		_, err = db.Exec(
			`UPDATE qsos SET call=?, qso_date=?, time_on=?, time_off=?, band=?, freq=?, freq_rx=?, mode=?, submode=?,
			rst_sent=?, rst_rcvd=?, gridsquare=?, name=?, qth=?, country=?, comment=?, notes=?, tx_pwr=?,
			distance=?, bearing=?,
		sota_ref=?, pota_ref=?, wwff_ref=?, iota=?, sig=?,
			my_sota_ref=?, my_pota_ref=?, my_wwff_ref=?,
			station_callsign=?, operator=?, my_gridsquare=?, my_rig=?, my_antenna=?, source=?,
			cq_zone=?, itu_zone=?,
			my_cq_zone=?, my_itu_zone=?, my_dxcc=?,
			my_sig=?, my_sig_info=?,
			wavelog_uploaded=?, contest_id=?, exch_sent=?, exch_rcvd=?, stx=?, srx=?, stx_string=?, srx_string=?, contest_adif_id=?,
			updated_at=?
			WHERE id=?`,
			q.Call, q.QSODate, q.TimeOn, q.TimeOff,
			q.Band, q.Freq, q.FreqRx, q.Mode, q.Submode,
			q.RSTSent, q.RSTRcvd, q.GridSquare, q.Name, q.QTH, q.Country, q.Comment, q.Notes, q.TXPower,
			q.Distance, q.Bearing,
			q.SOTARef, q.POTARef, q.WWFFRef, q.IOTA, q.SIG,
			q.MySOTARef, q.MyPOTARef, q.MyWWFFRef,
			q.StationCallsign, q.Operator, q.MyGridSquare, q.MyRig, q.MyAntenna, q.Source,
			q.CQZone, q.ITUZone, q.MyCQZone, q.MyITUZone, q.MyDXCC, q.MySIG, q.MySIGInfo, q.WavelogUploaded, q.ContestID, q.ExchSent, q.ExchRcvd, q.STX, q.SRX, q.STXString, q.SRXString, q.ContestADIFID,
			q.UpdatedAt.Format(time.RFC3339),
			q.ID,
		)
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
}

// UpdateQSOEnrichment applies callbook enrichment to a QSO.
// Only fields that are currently empty in the database are updated —
// existing data is never overwritten by enrichment.
func UpdateQSOEnrichment(db *sql.DB, qsoID int64, e EnrichmentData) {
	if e.Name == "" && e.QTH == "" && e.Country == "" && e.GridSquare == "" && e.IOTA == "" && e.CQZone == "" && e.ITUZone == "" {
		return
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

	if len(sets) == 0 {
		return
	}

	args = append(args, qsoID)
	query := fmt.Sprintf("UPDATE qsos SET %s WHERE id = ?", strings.Join(sets, ", "))
	db.Exec(query, args...) // best-effort; errors logged by caller
}
