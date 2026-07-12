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
// (case-insensitive and diacritic-insensitive substring match). Also matches
// by prefix for grid searches. Results are ordered by ref_type then ref,
// limited to 500 rows.
func (rdb *DB) Search(query string) ([]Row, error) {
	q := normalizeForSearch(query)
	like := "%" + q + "%"
	// Raw LIKE for databases that predate the search column — preserves
	// ASCII case-insensitive matching on the original ref/name text.
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
	return results, rows.Err()
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
