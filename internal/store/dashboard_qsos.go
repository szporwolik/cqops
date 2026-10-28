package store

import (
	"database/sql"
	"fmt"
)

// DashboardQSO is the dashboard's own projection of a contact: exactly the
// columns the Today and Recent panels render.
//
// The general-purpose loader (ListQSOsFromDate → listQSOsByQuery) scans the
// whole record — notes, all five references plus their "my" counterparts,
// station fields, the Wavelog bookkeeping and the timestamps — and the
// dashboard reads only fourteen of those columns. It loads up to 5000 rows on
// the TUI update path, so the unused columns were pure allocation churn:
// measured 56 ms and 25.7 MiB of allocations per refresh at 5000 rows, on the
// goroutine that also handles keystrokes.
//
// This is deliberately a separate type and scanner: the general QSO scanner is
// shared by the editor, exports, imports, the dupe check and Wavelog sync, and
// must keep reading everything.
type DashboardQSO struct {
	ID           int64
	Call         string
	QSODate      string
	TimeOn       string
	Band         string
	Freq         float64 // MHz; 0 when the row carries no frequency
	Mode         string
	Submode      string
	RSTSent      string
	RSTRcvd      string
	GridSquare   string
	Country      string
	Operator     string
	MyGridSquare string
}

// dashboardQSOCols mirrors DashboardQSO field for field. Keep the two in the
// same order: the scanner below reads positionally.
const dashboardQSOCols = `id, call, qso_date, time_on, band, freq, mode, submode,
	rst_sent, rst_rcvd, gridsquare, country, operator, my_gridsquare`

// ListDashboardQSOsFromDate returns the dashboard projection of the contacts
// with qso_date >= date, newest-first, capped at limit. Same ordering and the
// same limit semantics as ListQSOsFromDate — it is the same query with a
// narrower SELECT, so paging ties (date, time, id) break identically.
func ListDashboardQSOsFromDate(db *sql.DB, date string, limit int) ([]DashboardQSO, error) {
	if db == nil {
		return nil, nil
	}
	query := `SELECT ` + dashboardQSOCols + `
		FROM qsos WHERE qso_date >= ? ORDER BY ` + qsoNewestFirst + ` LIMIT ?`
	rows, err := db.Query(query, date, limit)
	if err != nil {
		return nil, fmt.Errorf("list dashboard qsos: %w", err)
	}
	defer rows.Close()

	var qsos []DashboardQSO
	for rows.Next() {
		var q DashboardQSO
		// Nullable columns are read through sql.Null* so a row an older import
		// left with NULLs still renders (as empty) instead of failing the whole
		// panel; the general scanner would reject such a row outright.
		var (
			band, submode, rstSent, rstRcvd, grid, country, operator, myGrid sql.NullString
			freq                                                             sql.NullFloat64
		)
		if err := rows.Scan(
			&q.ID, &q.Call, &q.QSODate, &q.TimeOn, &band, &freq, &q.Mode, &submode,
			&rstSent, &rstRcvd, &grid, &country, &operator, &myGrid,
		); err != nil {
			return nil, fmt.Errorf("scan dashboard qso: %w", err)
		}
		q.Band = band.String
		q.Freq = freq.Float64
		q.Submode = submode.String
		q.RSTSent = rstSent.String
		q.RSTRcvd = rstRcvd.String
		q.GridSquare = grid.String
		q.Country = country.String
		q.Operator = operator.String
		q.MyGridSquare = myGrid.String
		qsos = append(qsos, q)
	}
	return qsos, rows.Err()
}
