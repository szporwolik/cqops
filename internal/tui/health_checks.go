package tui

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ctybig"
	"github.com/szporwolik/cqops/internal/ref"
	"github.com/szporwolik/cqops/internal/version"
)

// bigCTYCatalog is the page listing all Big CTY releases.
const bigCTYCatalog = "https://www.country-files.com/category/big-cty/"

// bigCTYCatalogPage is the catalog URL the fetcher reads. A variable so tests
// can point it at a local server.
var bigCTYCatalogPage = bigCTYCatalog

// refHTTPTimeout bounds every reference-data HTTP request: a server that
// accepts a connection and then stalls fails the request instead of hanging
// the refresh. Kept a variable for tests.
var refHTTPTimeout = 30 * time.Second

// refHTTPClient is the client for reference-data downloads.
var refHTTPClient = &http.Client{Timeout: refHTTPTimeout}

// maxBigCTYCSVBytes caps the EXPANDED cty.csv so a hostile or unexpected
// archive cannot exhaust memory — the compressed download cap alone does not
// bound the expansion. A variable so tests can lower the cap.
var maxBigCTYCSVBytes uint64 = 64 * 1024 * 1024

// bigCTYZipRe extracts the first Big CTY ZIP download URL from the catalog page.
var bigCTYZipRe = regexp.MustCompile(`https://www\.country-files\.com/bigcty/download/\d{4}/bigcty-\d{8}\.zip`)

// findBigCTYURL fetches the Big CTY catalog page and returns the latest
// ZIP download URL. The request is bounded by ctx and refHTTPClient — a
// stalled server yields an empty string instead of a hang.
func findBigCTYURL(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bigCTYCatalogPage, nil)
	if err != nil {
		return ""
	}
	resp, err := refHTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return ""
	}
	return bigCTYZipRe.FindString(string(body))
}

// bigCTYFiles holds the extracted Big CTY CSV data.
type bigCTYFiles struct {
	ctyCSV []byte
}

// downloadBigCTY downloads the Big CTY ZIP from url and extracts cty.csv.
// The request is bounded by ctx and refHTTPClient; the compressed download
// and the EXPANDED cty.csv are both capped.
func downloadBigCTY(ctx context.Context, url string) (*bigCTYFiles, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("download bigcty: %w", err)
	}
	resp, err := refHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download bigcty: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download bigcty: HTTP %s", resp.Status)
	}

	zipBytes, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read bigcty zip: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open bigcty zip: %w", err)
	}

	var bf bigCTYFiles
	for _, f := range zr.File {
		if !strings.EqualFold(f.Name, "cty.csv") {
			continue
		}
		if f.UncompressedSize64 > maxBigCTYCSVBytes {
			return nil, fmt.Errorf("cty.csv expands to %d bytes, exceeding the %d byte cap", f.UncompressedSize64, maxBigCTYCSVBytes)
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(rc, int64(maxBigCTYCSVBytes)+1))
		rc.Close()
		if err != nil {
			continue
		}
		if uint64(len(data)) > maxBigCTYCSVBytes {
			return nil, fmt.Errorf("cty.csv exceeds %d bytes when expanded", maxBigCTYCSVBytes)
		}
		bf.ctyCSV = data
		break
	}
	if len(bf.ctyCSV) == 0 {
		return nil, fmt.Errorf("cty.csv not found in bigcty zip")
	}
	return &bf, nil
}

// bigCTYDateRe extracts the YYYYMMDD date embedded in the Big CTY filename.
var bigCTYDateRe = regexp.MustCompile(`bigcty-(\d{8})\.zip`)

// isBigCTYNewer returns true when the remote Big CTY at url is newer than
// the local file at path. The URL embeds a date (bigcty-YYYYMMDD.zip);
// if the date cannot be extracted we conservatively return true so a
// download is attempted.
func isBigCTYNewer(url, localPath string) (bool, error) {
	m := bigCTYDateRe.FindStringSubmatch(url)
	if len(m) < 2 {
		return true, nil // can't determine date — try download
	}
	remoteDate, err := time.Parse("20060102", m[1])
	if err != nil {
		return true, nil
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return true, nil
	}
	return remoteDate.After(info.ModTime()), nil
}

// backfillMissingDXCC scans the QSO table for rows with an empty dxcc
// column and fills them using the Big CTY prefix lookup. Processes at
// most maxRows per call (0 = unlimited). Returns the number of QSOs updated.
func backfillMissingDXCC(database *sql.DB, bdb *ctybig.DB) (int, error) {
	return backfillMissingDXCCLimit(database, bdb, 0)
}

// backfillMissingDXCCLimit is the implementation with a row limit.
func backfillMissingDXCCLimit(database *sql.DB, bdb *ctybig.DB, maxRows int) (int, error) {
	if database == nil || bdb == nil {
		return 0, nil
	}

	query := `SELECT id, call FROM qsos WHERE COALESCE(dxcc,'') = ''`
	if maxRows > 0 {
		query += fmt.Sprintf(" LIMIT %d", maxRows)
	}
	rows, err := database.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type update struct {
		id   int64
		dxcc int
	}
	var updates []update
	for rows.Next() {
		var id int64
		var call string
		if err := rows.Scan(&id, &call); err != nil {
			continue
		}
		e := bdb.Find(call)
		if e != nil && e.DXCC > 0 {
			updates = append(updates, update{id: id, dxcc: e.DXCC})
		}
	}
	rows.Close()

	count := 0
	if err := rows.Err(); err != nil {
		return count, err
	}
	for _, u := range updates {
		_, err := database.Exec(`UPDATE qsos SET dxcc = ? WHERE id = ?`,
			fmt.Sprintf("%d", u.dxcc), u.id)
		if err == nil {
			count++
		}
	}
	return count, nil
}

// =============================================================================
// Health checks — periodic internet connectivity and date/time management.
// =============================================================================

// autoUpdateDateTime keeps the QSO form date/time fields current.
// Below 85 chars terminal width, separators are dropped to save space.
func (m *Model) autoUpdateDateTime() {
	if !m.dateTimeAuto {
		return
	}
	now := time.Now().UTC()
	dateFmt, timeFmt := dateTimeFormats(m.width)
	m.fields[fieldDate].SetValue(now.Format(dateFmt))
	m.fields[fieldTime].SetValue(now.Format(timeFmt))
}

// dateTimeFormats returns the format strings for date and time fields.
// Seconds are omitted from default display; the user can type them manually.
// DB saves and ADIF/Wavelog exports always include full seconds.
func dateTimeFormats(width int) (dateFmt, timeFmt string) {
	if width < 85 {
		return "20060102", "1504"
	}
	return "2006-01-02", "15:04"
}

// maybeCheckInet returns a tea.Cmd to check internet connectivity at intervals.
// Uses adaptive polling: every 60 s when online AND confirmed, every 15 s when
// offline or before the first successful confirmation. This avoids a long wait
// on cold start when the first check fails transiently (WiFi still connecting,
// DNS not ready). Requires 2 consecutive failures before marking offline
// (avoids transient blips); a single success marks online immediately.
func (m *Model) maybeCheckInet() tea.Cmd {
	if m.Offline {
		return nil
	}
	interval := healthCheckTicks
	if !m.inetOnline || !m.inetConfirmed {
		interval = healthCheckTicksFast
	}
	if m.tickCount%interval == 0 {
		return checkInetCmd()
	}
	return nil
}

// checkInetCmd returns a tea.Cmd that checks internet connectivity by
// attempting to reach two independent endpoints. If either responds, we
// consider the internet reachable. This avoids false negatives when a
// single provider (e.g. Google) is temporarily blocked.
func checkInetCmd() tea.Cmd {
	return func() tea.Msg {
		applog.Debug("Internet: testing connectivity")
		client := &http.Client{Timeout: 3 * time.Second}
		// Primary: Google 204 endpoint (fast, no body).
		resp, err := client.Get("https://clients3.google.com/generate_204")
		if err == nil {
			resp.Body.Close()
			applog.Info("Internet: reachable")
			return inetResultMsg(true)
		}
		// Fallback: Cloudflare DNS — a simple TCP dial to 1.1.1.1:53
		// is lighter than a full HTTP request and works behind most
		// firewalls.
		applog.Debug("Internet: primary check failed, trying fallback", "error", err)
		client.Timeout = 2 * time.Second
		resp2, err2 := client.Get("https://1.1.1.1")
		if err2 == nil {
			resp2.Body.Close()
			applog.Info("Internet: reachable (fallback)")
			return inetResultMsg(true)
		}
		applog.Warn("Internet: unreachable", "primary_err", err, "fallback_err", err2)
		return inetResultMsg(false)
	}
}

// isNetworkError returns true when err indicates a definitive network-layer
// failure (DNS resolution, routing) rather than an application-level issue.
// Used by service failure handlers (DXC, APRS) to decide whether to fire
// an immediate health check instead of waiting for the scheduled poll.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "no such host") ||
		strings.Contains(s, "network is unreachable")
}

// noteNetworkError is called when a network-dependent service (DXC, APRS)
// fails with a hard network error.  It sets a flag so that the next tick
// dispatches an immediate health check, avoiding the up-to-60 s delay
// of the normal polling cycle.  The health-check streak logic is unchanged
// — two consecutive failures are still required before marking offline.
func (m *Model) noteNetworkError() {
	if !m.inetOnline {
		return // already offline; nothing to accelerate
	}
	m.triggerRapidCheck = true
}

// =============================================================================
// Version check — GitHub latest release.
// =============================================================================

// maybeCheckVersion returns a tea.Cmd to check GitHub for a newer release.
// Runs once when internet is first confirmed reachable.
func (m *Model) maybeCheckVersion() tea.Cmd {
	if m.versionChecked {
		return nil
	}
	if !m.inetOnline || m.Offline {
		return nil
	}
	m.versionChecked = true
	return checkVersionCmd()
}

// checkVersionCmd returns a tea.Cmd that queries the GitHub API for the
// latest release tag and compares it to the running version.
func checkVersionCmd() tea.Cmd {
	return func() tea.Msg {
		applog.Debug("Version: checking GitHub for latest release")
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("GET", "https://api.github.com/repos/szporwolik/cqops/releases/latest", nil)
		if err != nil {
			applog.Debug("Version: failed to build request", "error", err)
			return versionCheckMsg{}
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "CQOps-version-check")
		resp, err := client.Do(req)
		if err != nil {
			applog.Debug("Version: GitHub API unreachable", "error", err)
			return versionCheckMsg{}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			applog.Debug("Version: GitHub API returned non-OK status", "status", resp.StatusCode)
			return versionCheckMsg{}
		}

		var release struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			applog.Debug("Version: failed to parse GitHub response", "error", err)
			return versionCheckMsg{}
		}

		latest := strings.TrimPrefix(release.TagName, "v")
		if latest == "" {
			applog.Debug("Version: empty tag in GitHub response")
			return versionCheckMsg{}
		}

		current := version.Resolved()
		applog.Info("Version: check complete",
			"current", current,
			"latest", latest,
		)

		return versionCheckMsg{latest: latest}
	}
}

// versionNewer returns true if a is a newer semver-like version than b.
// Handles "0.8.0" style dotted versions with simple string comparison
// (works for equal-length segments).
func versionNewer(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	// Compare dotted segments lexicographically after zero-padding.
	segA := strings.Split(strings.TrimPrefix(a, "v"), ".")
	segB := strings.Split(strings.TrimPrefix(b, "v"), ".")
	maxLen := len(segA)
	if len(segB) > maxLen {
		maxLen = len(segB)
	}
	for i := 0; i < maxLen; i++ {
		var va, vb string
		if i < len(segA) {
			va = fmt.Sprintf("%010s", segA[i])
		} else {
			va = "0000000000"
		}
		if i < len(segB) {
			vb = fmt.Sprintf("%010s", segB[i])
		} else {
			vb = "0000000000"
		}
		if va > vb {
			return true
		}
		if va < vb {
			return false
		}
	}
	return false
}

// refDataMsg carries freshly loaded reference databases from the background
// refresh worker to the main loop, which installs them on App. The worker
// itself never touches App — it only returns data.
type refDataMsg struct {
	bigCTY *ctybig.DB
	scp    *scp.Database
	refDB  *ref.DB
}

// dxccBackfillMsg reports the result of a post-refresh DXCC backfill.
type dxccBackfillMsg struct {
	count int
}

// refDataTimeout bounds the whole reference-data refresh; every HTTP request
// inside it is additionally bounded by refHTTPClient.
const refDataTimeout = 2 * time.Minute

// dxccBackfillCmd runs the full missing-DXCC backfill off the UI loop after
// a fresh Big CTY database was installed.
func dxccBackfillCmd(database *sql.DB, bdb *ctybig.DB) tea.Cmd {
	return func() tea.Msg {
		n, err := backfillMissingDXCC(database, bdb)
		if err != nil {
			applog.Warn("DXCC: backfill failed", "error", err)
		}
		return dxccBackfillMsg{count: n}
	}
}

// runRefDataRefresh performs one bounded refresh pass over the reference
// data files in cacheDir and returns the loaded resources. It must not touch
// App — the result is installed by the main loop from refDataMsg.
func runRefDataRefresh(cacheDir string, useCTY, useSCP, useRef, bigCTYLoaded bool) refDataMsg {
	ctx, cancel := context.WithTimeout(context.Background(), refDataTimeout)
	defer cancel()

	var out refDataMsg
	if useCTY {
		out.bigCTY = refreshBigCTY(ctx, cacheDir, bigCTYLoaded)
	}
	if useSCP {
		out.scp = refreshSCP(cacheDir)
	}
	if useRef {
		if rdb, openErr := ref.Open(filepath.Join(cacheDir, "ref.db")); openErr == nil {
			out.refDB = rdb
			applog.Info("REF: database opened on demand")
		}
	}
	return out
}

// refreshBigCTY loads the cached cty.csv when not yet loaded, downloads a
// newer copy when the catalog says one exists, and returns the best
// available database (nil when nothing usable was found).
func refreshBigCTY(ctx context.Context, cacheDir string, loaded bool) *ctybig.DB {
	csvFile := filepath.Join(cacheDir, "cty.csv")

	var cached *ctybig.DB
	if !loaded {
		if data, err := os.ReadFile(csvFile); err == nil && len(data) > 0 {
			if db, err := ctybig.ParseCSV(bytes.NewReader(data)); err == nil {
				applog.Info("DXCC: Big CTY loaded from cache", "entries", db.Prefixes())
				cached = db
			}
		}
	}

	needDownload := false
	if _, statErr := os.Stat(csvFile); os.IsNotExist(statErr) {
		needDownload = true
	} else if url := findBigCTYURL(ctx); url != "" {
		if newer, _ := isBigCTYNewer(url, csvFile); newer {
			needDownload = true
		}
	}

	if needDownload {
		if url := findBigCTYURL(ctx); url != "" {
			applog.Info("DXCC: downloading Big CTY", "url", url)
			bf, dlErr := downloadBigCTY(ctx, url)
			if dlErr != nil {
				applog.Warn("DXCC: Big CTY download failed", "error", dlErr.Error())
			} else if len(bf.ctyCSV) > 0 {
				if werr := os.WriteFile(csvFile, bf.ctyCSV, 0644); werr != nil {
					applog.Warn("DXCC: cannot cache Big CTY CSV", "error", werr)
				}
				if db, err := ctybig.ParseCSV(bytes.NewReader(bf.ctyCSV)); err == nil {
					applog.Info("DXCC: Big CTY CSV loaded", "entries", db.Prefixes())
					return db
				}
				applog.Warn("DXCC: Big CTY CSV parse failed")
			}
		}
	}
	return cached
}

// refreshSCP downloads or updates MASTER.SCP and loads it. The scp library
// bounds its own HTTP requests; this runs off the UI loop regardless.
func refreshSCP(cacheDir string) *scp.Database {
	localFile := filepath.Join(cacheDir, "MASTER.SCP")
	if _, statErr := os.Stat(localFile); os.IsNotExist(statErr) {
		applog.Info("SCP: downloading on first run")
		if dlErr := scp.Download(scp.DefaultURL, localFile); dlErr != nil {
			applog.Warn("SCP: download failed", "error", dlErr.Error())
			return nil
		}
	} else {
		if updated, _ := scp.Update(scp.DefaultURL, localFile); updated {
			applog.Info("SCP: database updated")
		}
	}
	if db, loadErr := scp.LoadLocal(localFile); loadErr == nil {
		applog.Info("SCP: database loaded")
		return db
	}
	return nil
}

// maybeRefreshDataFiles returns a command to download or update CTY.DAT and
// MASTER.SCP data files. Runs once at startup (if cache is missing) and then
// at most once per 24 hours. Only triggers when internet is confirmed
// reachable and the respective config flags are on. The worker runs on a
// background goroutine (Bubble Tea v2 commands) and returns loaded resources
// as a message — it never assigns App fields itself.
func (m *Model) maybeRefreshDataFiles() tea.Cmd {
	if !m.inetOnline {
		return nil
	}
	// Don't recalculate on every tick — check at most once per 24 hours.
	if time.Since(m.lastDataCheck) < 24*time.Hour {
		return nil
	}
	m.lastDataCheck = time.Now()

	cacheDir, err := config.CacheDir()
	if err != nil {
		return nil
	}
	useCTY := m.App.Config.General.UseCTY
	useSCP := m.App.Config.General.UseSCP
	useRef := m.App.Config.General.UseRef && m.App.RefDB == nil
	bigCTYLoaded := m.App.BigCTY != nil

	return func() tea.Msg {
		return runRefDataRefresh(cacheDir, useCTY, useSCP, useRef, bigCTYLoaded)
	}
}
