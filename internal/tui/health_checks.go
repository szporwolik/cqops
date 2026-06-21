package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ref"
	"github.com/szporwolik/cqops/internal/version"
)

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

// dateTimeFormats returns the format strings for date and time fields
// based on terminal width. Below 85 chars, dashes and colons are omitted.
func dateTimeFormats(width int) (dateFmt, timeFmt string) {
	if width < 85 {
		return "20060102", "150405"
	}
	return "2006-01-02", "15:04:05"
}

// maybeCheckInet returns a tea.Cmd to check internet connectivity at intervals.
func (m *Model) maybeCheckInet() tea.Cmd {
	if m.Offline {
		return nil
	}
	if m.tickCount%healthCheckTicks == 0 {
		return checkInetCmd()
	}
	return nil
}

// checkInetCmd returns a tea.Cmd that checks internet connectivity
// by attempting to reach Google's generate_204 endpoint.
func checkInetCmd() tea.Cmd {
	return func() tea.Msg {
		applog.Debug("Internet: testing connectivity")
		client := &http.Client{Timeout: 1 * time.Second}
		resp, err := client.Get("https://clients3.google.com/generate_204")
		if err != nil {
			applog.Warn("Internet: unreachable", "error", err)
			return inetResultMsg(false)
		}
		defer resp.Body.Close()
		applog.Info("Internet: reachable")
		return inetResultMsg(true)
	}
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

// maybeRefreshDataFiles returns a command to download or update CTY.DAT and
// MASTER.SCP data files. Runs once at startup (if cache is missing) and then
// at most once per 24 hours. Only triggers when internet is confirmed
// reachable and the respective config flags are on.
func (m *Model) maybeRefreshDataFiles() tea.Cmd {
	if !m.inetOnline {
		return nil
	}
	// Don't recalculate on every tick — check at most once per 24 hours.
	if time.Since(m.lastDataCheck) < 24*time.Hour {
		return nil
	}
	m.lastDataCheck = time.Now()
	return func() tea.Msg {
		cacheDir, err := config.CacheDir()
		if err != nil {
			return nil
		}

		if m.App.Config.General.UseCTY {
			localFile := filepath.Join(cacheDir, "cty.dat")
			if _, statErr := os.Stat(localFile); os.IsNotExist(statErr) {
				applog.Info("DXCC: downloading on first run")
				if dlErr := dxcc.Download(dxcc.DefaultURL, localFile); dlErr != nil {
					applog.Warn("DXCC: download failed", "error", dlErr.Error())
				} else if prefixes, loadErr := dxcc.LoadLocal(localFile); loadErr == nil {
					m.App.DXCC = prefixes
					applog.Info("DXCC: prefix data loaded after download")
				}
			} else {
				if updated, _ := dxcc.Update(dxcc.DefaultURL, localFile); updated {
					if prefixes, loadErr := dxcc.LoadLocal(localFile); loadErr == nil {
						m.App.DXCC = prefixes
						applog.Info("DXCC: prefix data updated")
					}
				}
			}
		}

		if m.App.Config.General.UseSCP {
			localFile := filepath.Join(cacheDir, "MASTER.SCP")
			if _, statErr := os.Stat(localFile); os.IsNotExist(statErr) {
				applog.Info("SCP: downloading on first run")
				if dlErr := scp.Download(scp.DefaultURL, localFile); dlErr != nil {
					applog.Warn("SCP: download failed", "error", dlErr.Error())
				} else if db, loadErr := scp.LoadLocal(localFile); loadErr == nil {
					m.App.SCP = db
					applog.Info("SCP: database loaded after download")
				}
			} else {
				if updated, _ := scp.Update(scp.DefaultURL, localFile); updated {
					if db, loadErr := scp.LoadLocal(localFile); loadErr == nil {
						m.App.SCP = db
						applog.Info("SCP: database updated")
					}
				}
			}
		}

		if m.App.Config.General.UseRef && m.App.RefDB == nil {
			refPath := filepath.Join(cacheDir, "ref.db")
			if rdb, openErr := ref.Open(refPath); openErr == nil {
				m.App.RefDB = rdb
				applog.Info("REF: database opened on demand")
			}
		}
		return nil
	}
}
