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

	var id int64
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		res, err := db.Exec(
			`INSERT INTO qsos (`+qsoCols+`, created_at, updated_at)
			VALUES (`+placeholders(36)+`)`,
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

// placeholders returns a string of n comma-separated "?" placeholders.
func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func ListQSOs(db *sql.DB, limit int) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		wavelog_uploaded,
		created_at, updated_at
		FROM qsos
		ORDER BY id DESC`
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = db.Query(query+" LIMIT ?", limit)
	} else {
		rows, err = db.Query(query)
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

// ListQSOsPage returns a page of QSOs ordered by id DESC.
func ListQSOsPage(db *sql.DB, limit, offset int) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		wavelog_uploaded,
		created_at, updated_at
		FROM qsos
		ORDER BY id DESC
		LIMIT ? OFFSET ?`
	rows, err := db.Query(query, limit, offset)
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

// SearchQSOsByCall returns QSOs matching a callsign by exact or prefix match.
// "SP9SPM" matches "SP9SPM", "SP9SPM/P", "9A/SP9SPM", "9A/SP9SPM/P", etc.
func SearchQSOsByCall(db *sql.DB, call string, limit int) ([]qso.QSO, error) {
	query := `SELECT id, call, qso_date, time_on, time_off, band, freq, freq_rx, mode, submode,
		rst_sent, rst_rcvd, gridsquare, name, qth, country, comment, notes, tx_pwr,
		distance, bearing,
		sota_ref, pota_ref, wwff_ref, iota,
		my_sota_ref, my_pota_ref, my_wwff_ref,
		station_callsign, operator, my_gridsquare, my_rig, my_antenna, source,
		wavelog_uploaded,
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

func ListAllQSOs(db *sql.DB) ([]qso.QSO, error) {
	return ListQSOs(db, 0)
}

func UpdateQSO(db *sql.DB, q *qso.QSO) error {
	q.UpdatedAt = time.Now().UTC()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		_, err = db.Exec(
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

func PurgeQSOs(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM qsos`)
	if err != nil {
		return fmt.Errorf("purge qsos: %w", err)
	}
	return nil
}

// UpdateWavelogStatus sets the wavelog_uploaded status for a QSO.
func UpdateWavelogStatus(db *sql.DB, id int64, status string) error {
	// Retry on SQLITE_BUSY — the main thread may briefly hold a write lock
	// during concurrent operations (e.g. WSJT-X insert + immediate upload).
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		_, err = db.Exec(`UPDATE qsos SET wavelog_uploaded=? WHERE id=?`, status, id)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("update wavelog status: %w", err)
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

// QSOCounts holds aggregate QSO statistics.
type QSOCounts struct {
	Total     int
	FromWSJTX int
	ToWavelog int
}

// CountQSOs returns aggregate statistics for the current logbook.
func CountQSOs(db *sql.DB) (QSOCounts, error) {
	var c QSOCounts
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos`).Scan(&c.Total); err != nil {
		return c, fmt.Errorf("count qsos: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos WHERE source='wsjtx'`).Scan(&c.FromWSJTX); err != nil {
		return c, fmt.Errorf("count wsjtx qsos: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos WHERE wavelog_uploaded='yes'`).Scan(&c.ToWavelog); err != nil {
		return c, fmt.Errorf("count wavelog qsos: %w", err)
	}
	return c, nil
}

// LogbookStats holds per-call aggregate statistics from the local logbook.
type LogbookStats struct {
	CallWorked  bool
	CallOnBand  bool
	CallOnMode  bool
	QSOCount    int
	LastQSODate string // YYYY-MM-DD or empty
}

// GetLogbookStats computes per-call statistics efficiently with a single
// multi-count query. band and mode use the same prefix-match logic as
// SearchQSOsByCall for consistency.
func GetLogbookStats(db *sql.DB, call, band, mode string) (LogbookStats, error) {
	var s LogbookStats

	// Match: exact, suffix (/P etc.), prefix (DL/ etc.), both (DL/.../P).
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM qsos WHERE call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ?`,
		call, call+"/%", "%/"+call, "%/"+call+"/%",
	).Scan(&s.QSOCount); err != nil {
		return s, fmt.Errorf("count call qsos: %w", err)
	}
	s.CallWorked = s.QSOCount > 0

	// On specific band.
	if band != "" {
		var n int
		db.QueryRow(
			`SELECT COUNT(*) FROM qsos WHERE (call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ?) AND band = ?`,
			call, call+"/%", "%/"+call, "%/"+call+"/%", band,
		).Scan(&n)
		s.CallOnBand = n > 0
	}

	// On specific mode.
	if mode != "" {
		var n int
		db.QueryRow(
			`SELECT COUNT(*) FROM qsos WHERE (call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ?) AND mode = ?`,
			call, call+"/%", "%/"+call, "%/"+call+"/%", mode,
		).Scan(&n)
		s.CallOnMode = n > 0
	}

	// Last QSO date.
	var lastDate string
	db.QueryRow(
		`SELECT qso_date FROM qsos WHERE call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ? ORDER BY id DESC LIMIT 1`,
		call, call+"/%", "%/"+call, "%/"+call+"/%",
	).Scan(&lastDate)
	if lastDate != "" && len(lastDate) == 8 {
		s.LastQSODate = lastDate[0:4] + "-" + lastDate[4:6] + "-" + lastDate[6:8]
	}

	return s, nil
}

// PSKSpot holds a single PSK Reporter reception record from the database.
type PSKSpot struct {
	ID           int64
	ReceiverCall string
	ReceiverLoc  string
	Frequency    float64
	SNR          int
	Mode         string
	FlowStart    int64
	FetchTime    int64
	StationCall  string
}

// InsertPSKSpots bulk-inserts PSK Reporter spots, skipping duplicates.
// Returns the number of newly inserted rows.
func InsertPSKSpots(db *sql.DB, spots []PSKSpot) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO psk_spots
		(receiver_call, receiver_loc, frequency, snr, mode, flow_start, fetch_time, station_call)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, s := range spots {
		res, err := stmt.Exec(s.ReceiverCall, s.ReceiverLoc, s.Frequency, s.SNR, s.Mode,
			s.FlowStart, s.FetchTime, s.StationCall)
		if err != nil {
			return inserted, fmt.Errorf("insert psk_spot: %w", err)
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

// QueryPSKSpots returns PSK Reporter spots for the given station callsign
// within the specified time window (flow_start >= since).
func QueryPSKSpots(db *sql.DB, stationCall string, since int64) ([]PSKSpot, error) {
	rows, err := db.Query(`SELECT id, receiver_call, receiver_loc, frequency, snr, mode,
		flow_start, fetch_time, station_call
		FROM psk_spots WHERE station_call=? AND flow_start >= ?
		ORDER BY flow_start DESC LIMIT 500`, stationCall, since)
	if err != nil {
		return nil, fmt.Errorf("query psk_spots: %w", err)
	}
	defer rows.Close()

	var spots []PSKSpot
	for rows.Next() {
		var s PSKSpot
		if err := rows.Scan(&s.ID, &s.ReceiverCall, &s.ReceiverLoc, &s.Frequency,
			&s.SNR, &s.Mode, &s.FlowStart, &s.FetchTime, &s.StationCall); err != nil {
			return spots, fmt.Errorf("scan psk_spot: %w", err)
		}
		spots = append(spots, s)
	}
	return spots, rows.Err()
}

// ---------------------------------------------------------------------------
// DXC spots
// ---------------------------------------------------------------------------

// DXCSpot holds a single DX Cluster spot from the database.
type DXCSpot struct {
	ID         int64
	DXCall     string
	Frequency  float64
	Band       string
	Mode       string
	ModeCat    string
	Comment    string
	Spotter    string
	ReceivedAt int64
}

// InsertDXCSpots bulk-inserts DX Cluster spots, skipping duplicates.
func InsertDXCSpots(db *sql.DB, spots []DXCSpot) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO dxc_spots
		(dx_call, frequency, band, mode, mode_cat, comment, spotter, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, s := range spots {
		res, err := stmt.Exec(s.DXCall, s.Frequency, s.Band, s.Mode, s.ModeCat, s.Comment, s.Spotter, s.ReceivedAt)
		if err != nil {
			return inserted, fmt.Errorf("insert dxc_spot: %w", err)
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

// PurgeOldDXCSpots removes DXC spots older than 120 minutes.
func PurgeOldDXCSpots(db *sql.DB) error {
	cutoff := time.Now().UTC().Add(-120 * time.Minute).Unix()
	_, err := db.Exec(`DELETE FROM dxc_spots WHERE received_at < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("purge dxc_spots: %w", err)
	}
	return nil
}

// QueryDXCSpots returns recent DXC spots ordered by time (newest first).
func QueryDXCSpots(db *sql.DB) ([]DXCSpot, error) {
	rows, err := db.Query(`SELECT id, dx_call, frequency, band, mode, mode_cat, comment, spotter, received_at
		FROM dxc_spots ORDER BY received_at DESC LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("query dxc_spots: %w", err)
	}
	defer rows.Close()
	var spots []DXCSpot
	for rows.Next() {
		var s DXCSpot
		if err := rows.Scan(&s.ID, &s.DXCall, &s.Frequency, &s.Band, &s.Mode, &s.ModeCat, &s.Comment, &s.Spotter, &s.ReceivedAt); err != nil {
			return spots, fmt.Errorf("scan dxc_spot: %w", err)
		}
		spots = append(spots, s)
	}
	return spots, rows.Err()
}

// QueryDXCSpotByCall returns the most recent DXC spot for a given callsign, if any.
func QueryDXCSpotByCall(db *sql.DB, call string) (*DXCSpot, error) {
	var s DXCSpot
	err := db.QueryRow(`SELECT id, dx_call, frequency, band, mode, mode_cat, comment, spotter, received_at
		FROM dxc_spots WHERE dx_call = ? ORDER BY received_at DESC LIMIT 1`, call).Scan(
		&s.ID, &s.DXCall, &s.Frequency, &s.Band, &s.Mode, &s.ModeCat, &s.Comment, &s.Spotter, &s.ReceivedAt)
	if err != nil {
		return nil, err // sql.ErrNoRows if not found
	}
	return &s, nil
}

// PurgeOldPSKSpots removes spots older than 7 days.
func PurgeOldPSKSpots(db *sql.DB) error {
	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour).Unix()
	_, err := db.Exec(`DELETE FROM psk_spots WHERE flow_start < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("purge psk_spots: %w", err)
	}
	return nil
}
