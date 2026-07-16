// Package ctybig parses the Big CTY cty.csv file from country-files.com.
// The CSV format carries the ADIF DXCC entity number, which the standard
// cty.dat format does not.
package ctybig

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Entry holds DXCC information for a single callsign prefix or exact call.
type Entry struct {
	Prefix     string  // primary prefix (e.g. "K")
	Name       string  // country name (e.g. "United States")
	DXCC       int     // ADIF DXCC entity number (e.g. 291)
	Continent  string  // 2-letter continent (e.g. "NA")
	CQZone     int     // CQ zone
	ITUZone    int     // ITU zone
	Lat        float64 // latitude
	Lon        float64 // longitude (negative = west)
	TZOffset   float64 // timezone offset from UTC (e.g. -5.0 = EST, 1.0 = CET)
	ExactMatch bool    // true when this entry requires exact callsign match (=)
}

// DB holds the parsed Big CTY data and provides prefix lookups.
type DB struct {
	// exact holds entries that require exact callsign match (keyed by
	// uppercase callsign, e.g. "3D2CR").
	exact map[string]*Entry
	// prefixes is a sorted slice for longest-prefix-match search.
	prefixes []prefixEntry
	// defaultEntry is the fallback when no prefix matches.
	defaultEntry *Entry
}

type prefixEntry struct {
	prefix string
	entry  *Entry
}

// ParseCSV reads the Big CTY CSV format and returns a populated DB.
// The CSV has no header; each line is:
//
//	PREFIX,NAME,DXCC,CONT,CQ,ITU,LAT,LON,TZ,ALIASES;
//
// Aliases is a space-separated list of additional prefixes ending with
// a semicolon. Prefixes starting with '=' require exact callsign match.
// Zone overrides (#) and [#] on individual aliases are stripped; the
// parent entry's zone values take precedence.
func ParseCSV(r io.Reader) (*DB, error) {
	db := &DB{
		exact: make(map[string]*Entry),
	}

	scanner := bufio.NewScanner(r)
	// Some lines (US entry) can be very long due to many aliases.
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "*") {
			continue // skip empty lines and header markers
		}

		entry, aliases, err := parseLine(line)
		if err != nil {
			continue // skip malformed lines
		}

		storePrefix(&db.prefixes, entry.Prefix, entry)

		// Process the alias list (semicolon-terminated).
		for _, alias := range strings.Fields(strings.TrimSuffix(aliases, ";")) {
			alias = strings.TrimSpace(alias)
			if alias == "" || alias == entry.Prefix {
				continue
			}

			isExact := strings.HasPrefix(alias, "=")
			clean := strings.TrimPrefix(alias, "=")

			// Strip zone override suffixes: AA0(4)[7] → AA0
			clean = stripZoneOverrides(clean)

			if clean == "" {
				continue
			}

			if isExact {
				a := *entry
				a.ExactMatch = true
				a.Prefix = clean
				db.exact[strings.ToUpper(clean)] = &a
			} else {
				storePrefix(&db.prefixes, clean, entry)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ctybig: scan: %w", err)
	}

	// Sort prefixes longest-first for longest-match lookup.
	sortPrefixes(db.prefixes)

	return db, nil
}

// Find looks up a callsign and returns the best matching Entry.
// Exact-match entries (callsign-level) are checked first, then
// longest-prefix match on the callsign.
func (db *DB) Find(callsign string) *Entry {
	call := strings.ToUpper(strings.TrimSpace(callsign))
	if call == "" {
		return nil
	}

	// 1. Exact callsign match (e.g. "3D2CR" → Conway Reef).
	if e, ok := db.exact[call]; ok {
		return e
	}

	// 2. Longest prefix match.
	for _, pe := range db.prefixes {
		if strings.HasPrefix(call, pe.prefix) {
			return pe.entry
		}
	}

	return db.defaultEntry
}

// Prefixes returns the number of prefix entries in the DB (for logging).
func (db *DB) Prefixes() int {
	return len(db.prefixes) + len(db.exact)
}

// --- internals ---------------------------------------------------------------

func parseLine(line string) (*Entry, string, error) {
	// Split into 10 fields: first 9 are comma-delimited, 10th is
	// the rest (aliases up to the terminating ';').
	parts := strings.SplitN(line, ",", 10)
	if len(parts) < 10 {
		return nil, "", fmt.Errorf("short line: %d fields", len(parts))
	}

	e := &Entry{Prefix: strings.ToUpper(strings.TrimSpace(parts[0]))}
	e.Name = strings.TrimSpace(parts[1])
	e.DXCC, _ = strconv.Atoi(strings.TrimSpace(parts[2]))
	e.Continent = strings.ToUpper(strings.TrimSpace(parts[3]))
	e.CQZone, _ = strconv.Atoi(strings.TrimSpace(parts[4]))
	e.ITUZone, _ = strconv.Atoi(strings.TrimSpace(parts[5]))
	e.Lat, _ = strconv.ParseFloat(strings.TrimSpace(parts[6]), 64)
	e.Lon, _ = strconv.ParseFloat(strings.TrimSpace(parts[7]), 64)
	e.TZOffset, _ = strconv.ParseFloat(strings.TrimSpace(parts[8]), 64)

	return e, parts[9], nil
}

func storePrefix(prefixes *[]prefixEntry, prefix string, entry *Entry) {
	p := strings.ToUpper(strings.TrimSpace(prefix))
	if p == "" {
		return
	}
	*prefixes = append(*prefixes, prefixEntry{prefix: p, entry: entry})
}

func sortPrefixes(prefixes []prefixEntry) {
	// Sort by length descending, then alphabetically.
	for i := 0; i < len(prefixes); i++ {
		for j := i + 1; j < len(prefixes); j++ {
			if len(prefixes[j].prefix) > len(prefixes[i].prefix) ||
				(len(prefixes[j].prefix) == len(prefixes[i].prefix) &&
					prefixes[j].prefix < prefixes[i].prefix) {
				prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
			}
		}
	}
}

// stripZoneOverrides removes (N) and [N] suffix annotations from an alias.
// "AA0(4)[7]" → "AA0", "=N2NL/MM(7)" → "N2NL/MM".
func stripZoneOverrides(s string) string {
	// Remove [N] first (longer pattern), then (N).
	for {
		before := s
		s = removeBracket(s)
		s = removeParen(s)
		if s == before {
			break
		}
	}
	return s
}

func removeBracket(s string) string {
	if idx := strings.Index(s, "["); idx >= 0 {
		if end := strings.Index(s[idx:], "]"); end >= 0 {
			return s[:idx] + s[idx+end+1:]
		}
	}
	return s
}

func removeParen(s string) string {
	if idx := strings.Index(s, "("); idx >= 0 {
		if end := strings.Index(s[idx:], ")"); end >= 0 {
			return s[:idx] + s[idx+end+1:]
		}
	}
	return s
}
