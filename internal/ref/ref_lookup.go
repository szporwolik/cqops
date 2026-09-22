package ref

import (
	"fmt"
	"strings"
)

// Lookup returns the reference row for the given type and reference code,
// or nil when not found.
func (rdb *DB) Lookup(rt RefType, ref string) (*Row, bool) {
	// is_group=1 rows (IOTA group entries) are preferred over island entries.
	row := rdb.db.QueryRow(
		`SELECT ref_type, ref, name, grid, height FROM refs WHERE ref_type=? AND ref=? ORDER BY is_group DESC LIMIT 1`,
		string(rt), strings.ToUpper(ref),
	)
	var r Row
	var rtStr string
	err := row.Scan(&rtStr, &r.Ref, &r.Name, &r.Grid, &r.Height)
	if err != nil {
		return nil, false
	}
	r.RefType = RefType(rtStr)
	return &r, true
}

// Count returns the total number of references in the database.
func (rdb *DB) Count() (int, error) {
	var n int
	err := rdb.db.QueryRow(`SELECT COUNT(*) FROM refs`).Scan(&n)
	return n, err
}

// CountByType returns the number of references for a given type.
func (rdb *DB) CountByType(rt RefType) (int, error) {
	var n int
	err := rdb.db.QueryRow(`SELECT COUNT(*) FROM refs WHERE ref_type=?`, string(rt)).Scan(&n)
	return n, err
}

// NeedsSearchBackfill returns true when the search column is empty for some
// rows — meaning the database predates the diacritic-insensitive search
// feature and should be rebuilt.
func (rdb *DB) NeedsSearchBackfill() (bool, error) {
	var n int
	err := rdb.db.QueryRow(`SELECT COUNT(*) FROM refs WHERE search = '' LIMIT 1`).Scan(&n)
	return n > 0, err
}

// Search returns all reference rows whose ref or name contains the query string
// (case-insensitive and diacritic-insensitive substring match) or whose grid
// contains it. Results are ordered by ref_type then ref, limited to 500 rows.
//
// Queries of three characters or more use the FTS5 trigram index — a
// ~200k-row table can be searched without a scan. Shorter queries (and
// databases that predate the normalized search column) fall back to the
// legacy LIKE path.
func (rdb *DB) Search(query string) ([]Row, error) {
	q := normalizeForSearch(query)
	if q == "" {
		return nil, nil
	}
	if needs, err := rdb.NeedsSearchBackfill(); err == nil && needs {
		return rdb.searchLegacy(query, q)
	}
	if len(q) >= 3 && !strings.Contains(q, `"`) {
		return rdb.searchFTS(q)
	}
	return rdb.searchLegacy(query, q)
}

// searchFTS resolves the query through the trigram index over the normalized
// search text and the grid. MATCH with a quoted phrase is a substring match.
func (rdb *DB) searchFTS(q string) ([]Row, error) {
	match := `"` + q + `"`
	rows, err := rdb.db.Query(
		`SELECT ref_type, ref, name, grid, height FROM refs
		 WHERE rowid IN (SELECT rowid FROM refs_fts WHERE refs_fts MATCH ?)
		 ORDER BY ref_type, ref
		 LIMIT 500`,
		match,
	)
	if err != nil {
		return nil, fmt.Errorf("ref search %q: %w", q, err)
	}
	defer rows.Close()
	return scanSearchRows(rows)
}

// searchLegacy is the pre-FTS path: raw LIKE over the normalized search
// column, the original ref/name text (for databases predating the search
// column), and the grid.
func (rdb *DB) searchLegacy(query, q string) ([]Row, error) {
	like := "%" + q + "%"
	rawLike := "%" + query + "%"
	rows, err := rdb.db.Query(
		`SELECT ref_type, ref, name, grid, height FROM refs
		 WHERE search LIKE ? ESCAPE '\'
		    OR (search = '' AND (ref LIKE ? ESCAPE '\' OR name LIKE ? ESCAPE '\'))
		    OR grid LIKE ? ESCAPE '\'
		 ORDER BY ref_type, ref
		 LIMIT 500`,
		like, rawLike, rawLike, like+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("ref search %q: %w", query, err)
	}
	defer rows.Close()
	return scanSearchRows(rows)
}

func scanSearchRows(rows interface {
	Next() bool
	Scan(dest ...any) error
}) ([]Row, error) {
	var results []Row
	for rows.Next() {
		var r Row
		var rt string
		if err := rows.Scan(&rt, &r.Ref, &r.Name, &r.Grid, &r.Height); err != nil {
			continue
		}
		r.RefType = RefType(rt)
		results = append(results, r)
	}
	return results, nil
}

// NameForRef looks up the human-readable name for a single reference.
// Returns the name if found, or the reference itself as fallback.
func (rdb *DB) NameForRef(rt RefType, ref string) string {
	r, ok := rdb.Lookup(rt, strings.ToUpper(ref))
	if !ok || r.Name == "" {
		return ref
	}
	return r.Name
}

// ResolveRefNames takes a comma-separated list of reference codes and returns
// a human-readable string with names resolved from the database. Unknown refs
// are kept as-is. The typePrefix, if non-empty, is prepended (e.g. "SOTA: ").
func (rdb *DB) ResolveRefNames(rt RefType, refsCSV, typePrefix string) string {
	if refsCSV == "" {
		return ""
	}
	parts := strings.Split(refsCSV, ",")
	var names []string
	for _, ref := range parts {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		name := rdb.NameForRef(rt, ref)
		names = append(names, name)
	}
	if len(names) == 0 {
		return ""
	}
	result := strings.Join(names, ", ")
	if typePrefix != "" {
		result = typePrefix + result
	}
	return result
}
