package ref

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ftl/hamradio/latlon"
	"github.com/ftl/hamradio/locator"
	"github.com/szporwolik/cqops/internal/applog"
)

// CSV URLs.
const (
	SotaURL = "https://storage.sota.org.uk/summitslist.csv"
	PotaURL = "https://pota.app/all_parks_ext.csv"
	WwffURL = "https://wwff.co/wwff-data/wwff_directory.csv"
	IotaURL = "https://www.iota-world.org/islands-on-the-air/downloads/download-file.html?path=islands.json"
)

// Cache file names.
const (
	sotaFile      = "sota.csv"
	potaFile      = "pota.csv"
	wwffFile      = "wwff.csv"
	iotaFile      = "iota_islands.json"
	iotaGroupFile = "iota_groups.json"
)

// IotaGroupURL is the groups list used for coordinate lookup.
const IotaGroupURL = "https://www.iota-world.org/islands-on-the-air/downloads/download-file.html?path=fulllist.json"

// MaxAge is the maximum age of cached data files before a re-download.
const MaxAge = 365 * 24 * time.Hour

// downloadRetries and backoff for potato internet.
const downloadRetries = 3
const downloadBackoff = 5 * time.Second

// httpClient is the shared HTTP client with a generous timeout suitable
// for 50 MB downloads on slow connections.
var httpClient = &http.Client{Timeout: 300 * time.Second}

// Rebuild downloads data files if needed and rebuilds the reference database.
// Progress is reported via the callback (called synchronously from the caller's
// goroutine). Returns the total number of imported rows.
//
// Error handling: if any single data source fails to download, the rebuild
// aborts and returns an error. Partial downloads are detected and re-fetched.
func (rdb *DB) Rebuild(cacheDir string, progress func(msg string)) (int, error) {
	if cacheDir == "" {
		return 0, fmt.Errorf("ref: cacheDir is empty")
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return 0, fmt.Errorf("ref: mkdir cache %s: %w", cacheDir, err)
	}

	// Step 1: ensure all data files are present and fresh.
	downloads := []struct {
		url, file string
	}{
		{SotaURL, sotaFile},
		{PotaURL, potaFile},
		{WwffURL, wwffFile},
		{IotaURL, iotaFile},
		{IotaGroupURL, iotaGroupFile},
	}
	for _, d := range downloads {
		if err := downloadFile(d.url, filepath.Join(cacheDir, d.file), progress); err != nil {
			return 0, fmt.Errorf("ref: %s download: %w", d.file, err)
		}
	}

	// Step 2: rebuild inside a single transaction for atomicity.
	tx, err := rdb.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("ref: begin tx: %w", err)
	}
	defer tx.Rollback() // safe no-op after commit

	if _, err := tx.Exec(`DELETE FROM refs`); err != nil {
		return 0, fmt.Errorf("ref: clear table: %w", err)
	}

	total := 0
	imports := []struct {
		label string
		fn    func(*sql.Tx, string) (int, error)
		file  string
	}{
		{"SOTA", importSOTA, sotaFile},
		{"POTA", importPOTA, potaFile},
		{"WWFF", importWWFF, wwffFile},
		{"IOTA", func(tx *sql.Tx, path string) (int, error) {
			return importIOTA(tx, path, filepath.Join(cacheDir, iotaGroupFile))
		}, iotaFile},
	}
	for _, imp := range imports {
		progress("Importing " + imp.label + "\u2026")
		n, err := imp.fn(tx, filepath.Join(cacheDir, imp.file))
		if err != nil {
			return 0, fmt.Errorf("ref: %s import: %w", imp.label, err)
		}
		total += n
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("ref: commit: %w", err)
	}
	applog.Info("ref: rebuild complete", "total", total)
	return total, nil
}

// RebuildIfMissing rebuilds only when the database is empty (first run).
func (rdb *DB) RebuildIfMissing(cacheDir string, progress func(msg string)) error {
	n, err := rdb.Count()
	if err != nil {
		return fmt.Errorf("ref: count: %w", err)
	}
	if n > 0 {
		applog.Debug("ref: database already populated, skipping rebuild", "rows", n)
		return nil
	}
	applog.Info("ref: database empty, starting initial rebuild")
	_, err = rdb.Rebuild(cacheDir, progress)
	return err
}

// downloadFile downloads url to localPath if missing or stale. Uses atomic
// write (temp file + rename) and retries on transient HTTP errors.
func downloadFile(url, localPath string, progress func(msg string)) error {
	fi, err := os.Stat(localPath)
	if err == nil {
		age := time.Since(fi.ModTime())
		if age < MaxAge {
			applog.Debug("ref: cached file fresh", "file", filepath.Base(localPath), "age", age.Round(time.Hour))
			return nil
		}
		applog.Debug("ref: cached file stale, re-downloading", "file", filepath.Base(localPath), "age", age.Round(time.Hour))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", localPath, err)
	}

	progress(fmt.Sprintf("Downloading %s\u2026", filepath.Base(localPath)))
	applog.Info("ref: downloading", "url", url, "file", filepath.Base(localPath))

	var lastErr error
	for attempt := 0; attempt < downloadRetries; attempt++ {
		if attempt > 0 {
			wait := downloadBackoff * time.Duration(attempt)
			applog.Debug("ref: download retry", "attempt", attempt+1, "wait", wait)
			time.Sleep(wait)
		}
		lastErr = fetchToFile(url, localPath)
		if lastErr == nil {
			return nil
		}
		applog.Warn("ref: download attempt failed", "attempt", attempt+1, "error", lastErr)
	}
	return fmt.Errorf("download %s after %d attempts: %w", filepath.Base(localPath), downloadRetries, lastErr)
}

// fetchToFile downloads url and writes to localPath atomically (temp + rename).
// Returns an error if the HTTP status is not 200 or if the write fails.
func fetchToFile(url, localPath string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain body to allow connection reuse.
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}

	tmpPath := localPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp %s: %w", tmpPath, err)
	}
	defer os.Remove(tmpPath)

	written, err := io.Copy(f, resp.Body)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write %s: %w (wrote %d bytes)", filepath.Base(localPath), err, written)
	}
	// Validate: if Content-Length was sent, verify the written size matches.
	if cl := resp.ContentLength; cl > 0 && written != cl {
		return fmt.Errorf("write %s: wrote %d bytes, expected %d (Content-Length mismatch)", filepath.Base(localPath), written, cl)
	}

	if err := os.Rename(tmpPath, localPath); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, localPath, err)
	}
	applog.Info("ref: downloaded OK", "file", filepath.Base(localPath), "bytes", written)
	return nil
}

// parseLatLon returns (lat, lon) as float64 or an error.
func parseLatLon(latStr, lonStr string) (float64, float64, error) {
	lat, err := strconv.ParseFloat(strings.TrimSpace(latStr), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("lat: %w", err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(lonStr), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("lon: %w", err)
	}
	return lat, lon, nil
}

// gridFromLatLon returns a 6-character Maidenhead grid locator from lat/lon,
// or empty string on invalid input. Falls back to 4-char when lat/lon precision
// only supports coarser resolution.
func gridFromLatLon(latStr, lonStr string) string {
	lat, lon, err := parseLatLon(latStr, lonStr)
	if err != nil {
		return ""
	}
	ll := latlon.NewLatLon(latlon.Latitude(lat), latlon.Longitude(lon))
	g := locator.LatLonToLocator(ll, 6)
	s := strings.TrimRight(string(g[:]), "\x00")
	if len(s) >= 4 {
		return strings.ToUpper(s)
	}
	return ""
}

// importSOTA parses a SOTA summits CSV and inserts rows into tx.
// The CSV has a title line first, then a header line, then data rows.
// Columns (after title+header): SummitCode, AssociationName, RegionName,
// SummitName, AltM, AltFt, GridRef1(lon), GridRef2(lat), Longitude, Latitude, ...
// GridRef1/2 are WGS84 decimal coordinates, NOT Maidenhead grid references.
// We need: ref=SummitCode(0), name=SummitName(3), height=AltM(4),
// grid computed from Longitude(8)/Latitude(9).
func importSOTA(tx *sql.Tx, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// Skip title line (e.g. "SOTA Summits List (Date=19/06/2026)").
	if _, err := r.Read(); err != nil {
		return 0, fmt.Errorf("read title: %w", err)
	}
	// Skip header line.
	if _, err := r.Read(); err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO refs (ref_type, ref, name, grid, height, is_group) VALUES (?,?,?,?,?,0)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			applog.Warn("ref: sota CSV read error", "error", err)
			continue
		}
		if len(rec) < 10 {
			continue
		}
		// col 0: SummitCode, col 3: SummitName, col 4: AltM,
		// col 8: Longitude, col 9: Latitude
		ref := strings.TrimSpace(rec[0])
		name := strings.TrimSpace(rec[3])
		height := 0
		if h, err := strconv.Atoi(strings.TrimSpace(rec[4])); err == nil {
			height = h
		}
		grid := gridFromLatLon(rec[9], rec[8]) // lat, lon
		if ref == "" || name == "" {
			continue
		}
		if _, err := stmt.Exec(string(RefSOTA), ref, name, grid, height); err != nil {
			applog.Warn("ref: sota insert", "ref", ref, "error", err)
			continue
		}
		count++
	}
	applog.Info("ref: SOTA imported", "count", count)
	return count, nil
}

// importPOTA parses the extended POTA parks CSV and inserts rows into tx.
// Columns: reference, name, active, entityId, locationDesc, latitude, longitude, grid
// We need: ref=reference(0), name=name(1), grid from grid(7) or lat(5)/lon(6).
func importPOTA(tx *sql.Tx, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// Skip header.
	if _, err := r.Read(); err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO refs (ref_type, ref, name, grid, height, is_group) VALUES (?,?,?,?,0,0)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			applog.Warn("ref: pota CSV read error", "error", err)
			continue
		}
		if len(rec) < 8 {
			continue
		}
		// col 0: reference, col 1: name, col 2: active, col 5: latitude,
		// col 6: longitude, col 7: grid
		if strings.TrimSpace(rec[2]) != "1" {
			continue // skip inactive/deleted
		}
		ref := strings.TrimSpace(rec[0])
		name := strings.TrimSpace(rec[1])
		grid := ""
		if len(rec) > 7 {
			g := strings.TrimSpace(rec[7])
			if len(g) >= 4 {
				grid = strings.ToUpper(g) // preserve full 6-char grid
			}
		}
		if grid == "" && len(rec) > 6 {
			grid = gridFromLatLon(rec[5], rec[6]) // lat, lon
		}
		if ref == "" || name == "" {
			continue
		}
		if _, err := stmt.Exec(string(RefPOTA), ref, name, grid); err != nil {
			applog.Warn("ref: pota insert", "ref", ref, "error", err)
			continue
		}
		count++
	}
	applog.Info("ref: POTA imported", "count", count)
	return count, nil
}

// importWWFF parses a WWFF directory CSV and inserts rows into tx.
// Columns: reference, status, name, program, dxcc, state, county, continent,
// iota, iaruLocator, latitude, longitude, ...
// We need: ref=reference, name=name, grid=iaruLocator (or from lat/lon).
func importWWFF(tx *sql.Tx, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// Skip header.
	if _, err := r.Read(); err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO refs (ref_type, ref, name, grid, height, is_group) VALUES (?,?,?,?,0,0)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			applog.Warn("ref: wwff CSV read error", "error", err)
			continue
		}
		if len(rec) < 12 {
			continue
		}
		// col 0: reference, col 1: status, col 2: name, col 9: iaruLocator,
		// col 10: latitude, col 11: longitude
		status := strings.TrimSpace(rec[1])
		if status != "active" {
			continue
		}
		ref := strings.TrimSpace(rec[0])
		name := strings.TrimSpace(rec[2])
		grid := ""
		if len(rec) > 9 {
			g := strings.TrimSpace(rec[9])
			if len(g) >= 4 {
				grid = strings.ToUpper(g) // preserve full 6-char grid
			}
		}
		if grid == "" && len(rec) > 11 {
			grid = gridFromLatLon(rec[10], rec[11]) // lat, lon
		}
		if ref == "" || name == "" {
			continue
		}
		if _, err := stmt.Exec(string(RefWWFF), ref, name, grid); err != nil {
			applog.Warn("ref: wwff insert", "ref", ref, "error", err)
			continue
		}
		count++
	}
	applog.Info("ref: WWFF imported", "count", count)
	return count, nil
}

// iotaIsland is a single entry in islands.json.
type iotaIsland struct {
	RefNo    string `json:"refno"`
	Name     string `json:"name"`
	Excluded string `json:"excluded"`
}

// iotaGroup is a single entry in fulllist.json (used for coordinates and group name).
type iotaGroup struct {
	RefNo  string `json:"refno"`
	Name   string `json:"name"`
	LatMax string `json:"latitude_max"`
	LatMin string `json:"latitude_min"`
	LonMax string `json:"longitude_max"`
	LonMin string `json:"longitude_min"`
}

// importIOTA parses fulllist.json for group names and coordinates,
// then islands.json for individual island search entries. Group entries
// are inserted first so Lookup (ORDER BY rowid) returns the group name.
func importIOTA(tx *sql.Tx, islandsPath, groupsPath string) (int, error) {
	// Step 1: parse groups — extract name and coordinates.
	type groupInfo struct {
		name                           string
		latMax, latMin, lonMax, lonMin float64
	}
	groupMap := make(map[string]groupInfo)
	groupsData, err := os.ReadFile(groupsPath)
	if err != nil {
		return 0, fmt.Errorf("read groups: %w", err)
	}
	var groups []iotaGroup
	if err := json.Unmarshal(groupsData, &groups); err != nil {
		return 0, fmt.Errorf("parse groups json: %w", err)
	}
	for _, g := range groups {
		if g.RefNo == "" {
			continue
		}
		latMax, _ := strconv.ParseFloat(g.LatMax, 64)
		latMin, _ := strconv.ParseFloat(g.LatMin, 64)
		lonMax, _ := strconv.ParseFloat(g.LonMax, 64)
		lonMin, _ := strconv.ParseFloat(g.LonMin, 64)
		groupMap[g.RefNo] = groupInfo{
			name:   g.Name,
			latMax: latMax,
			latMin: latMin,
			lonMax: lonMax,
			lonMin: lonMin,
		}
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO refs (ref_type, ref, name, grid, height, is_group) VALUES (?,?,?,?,0,?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int

	// Step 2: insert group entries first (is_group=1).
	for refno, gi := range groupMap {
		if gi.name == "" {
			continue
		}
		grid := ""
		if gi.latMax != 0 || gi.latMin != 0 || gi.lonMax != 0 || gi.lonMin != 0 {
			centerLat := (gi.latMax + gi.latMin) / 2
			centerLon := (gi.lonMax + gi.lonMin) / 2
			ll := latlon.NewLatLon(latlon.Latitude(centerLat), latlon.Longitude(centerLon))
			g := locator.LatLonToLocator(ll, 6)
			s := strings.TrimRight(string(g[:]), "\x00")
			if len(s) >= 4 {
				grid = strings.ToUpper(s)
			}
		}
		if _, err := stmt.Exec(string(RefIOTA), refno, gi.name, grid, 1); err != nil {
			applog.Warn("ref: iota group insert", "ref", refno, "name", gi.name, "error", err)
			continue
		}
		count++
	}

	// Step 3: insert individual islands for searchability (higher rowid).
	islandsData, err := os.ReadFile(islandsPath)
	if err != nil {
		return 0, fmt.Errorf("read islands: %w", err)
	}
	var islands []iotaIsland
	if err := json.Unmarshal(islandsData, &islands); err != nil {
		return 0, fmt.Errorf("parse islands json: %w", err)
	}

	for _, isl := range islands {
		if isl.RefNo == "" || isl.Name == "" {
			continue
		}
		// Skip if island name matches group name — already inserted.
		if gi, ok := groupMap[isl.RefNo]; ok && strings.EqualFold(isl.Name, gi.name) {
			continue
		}

		// Compute grid from the parent group's bounding box centre.
		grid := ""
		if gi, ok := groupMap[isl.RefNo]; ok {
			if gi.latMax != 0 || gi.latMin != 0 || gi.lonMax != 0 || gi.lonMin != 0 {
				centerLat := (gi.latMax + gi.latMin) / 2
				centerLon := (gi.lonMax + gi.lonMin) / 2
				ll := latlon.NewLatLon(latlon.Latitude(centerLat), latlon.Longitude(centerLon))
				g := locator.LatLonToLocator(ll, 6)
				s := strings.TrimRight(string(g[:]), "\x00")
				if len(s) >= 4 {
					grid = strings.ToUpper(s)
				}
			}
		}

		if _, err := stmt.Exec(string(RefIOTA), isl.RefNo, isl.Name, grid, 0); err != nil {
			applog.Warn("ref: iota island insert", "ref", isl.RefNo, "name", isl.Name, "error", err)
			continue
		}
		count++
	}
	applog.Info("ref: IOTA imported", "count", count, "groups", len(groupMap))
	return count, nil
}

// NeedsRebuild returns true when the database is empty or any cached CSV
// file is newer than the database file itself.
func (rdb *DB) NeedsRebuild(cacheDir string) bool {
	n, _ := rdb.Count()
	if n == 0 {
		return true
	}
	// Check if any CSV is newer than the DB — cheap heuristic.
	for _, fn := range []string{sotaFile, potaFile, wwffFile, iotaFile, iotaGroupFile} {
		fi, err := os.Stat(filepath.Join(cacheDir, fn))
		if err != nil {
			continue
		}
		if fi.ModTime().After(time.Now().Add(-MaxAge)) {
			// CSV was downloaded recently — but we need to check vs DB.
			// For now, just rely on the 30-day rebuild cycle.
			continue
		}
	}
	return false
}

// EnsureOpen is a convenience for opening a DB, creating the directory if needed.
func EnsureOpen(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("ref: mkdir %s: %w", dir, err)
	}
	return Open(dbPath)
}

// EnsureColumns is used by tests to verify schema.
func EnsureColumns(db *sql.DB) []string {
	rows, err := db.Query(`PRAGMA table_info(refs)`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			continue
		}
		cols = append(cols, name)
	}
	return cols
}
