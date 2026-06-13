package wavelog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// StationProfile represents a Wavelog station location.
type StationProfile struct {
	ID         string `json:"station_id"`
	Name       string `json:"station_profile_name"`
	Gridsquare string `json:"station_gridsquare"`
	Callsign   string `json:"station_callsign"`
	Active     string `json:"station_active"`
}

// VersionResponse from api/version.
type VersionResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// TestConnection validates that the Wavelog API URL and key are reachable.
func TestConnection(baseURL, apiKey string) error {
	applog.Debug("Wavelog: testing connection")
	if baseURL == "" || apiKey == "" {
		return fmt.Errorf("URL and API key required")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/api/version"

	payload := map[string]string{"key": apiKey}
	body, err := json.Marshal(payload)
	if err != nil {
		applog.Error("Wavelog: marshal payload failed", "error", err)
		return err
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		applog.Error("Wavelog: connection failed", "url", baseURL, "error", err)
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("Wavelog: read response failed", "error", err)
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		applog.Error("Wavelog: server error", "status", resp.StatusCode, "body", strings.TrimSpace(string(respBody)))
		return fmt.Errorf("server error: HTTP %d — %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var vr VersionResponse
	if err := json.Unmarshal(respBody, &vr); err != nil {
		applog.Error("Wavelog: invalid version response", "error", err)
		return fmt.Errorf("invalid response: %w", err)
	}
	if vr.Status != "ok" {
		applog.Error("Wavelog: API returned non-ok status", "status", vr.Status)
		return fmt.Errorf("API returned status: %s", vr.Status)
	}

	applog.InfoDetail("Wavelog: connected", fmt.Sprintf("version=%s url=%s", vr.Version, baseURL))
	return nil
}

// FetchStations retrieves station profiles from the Wavelog API.
func FetchStations(baseURL, apiKey string) ([]StationProfile, error) {
	applog.Debug("Wavelog: fetching stations")
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("URL and API key required")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/api/station_info/" + apiKey

	resp, err := httpClient.Get(url)
	if err != nil {
		applog.Error("Wavelog: fetch stations failed", "url", baseURL, "error", err)
		return nil, fmt.Errorf("fetch stations: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("Wavelog: read stations response failed", "error", err)
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		applog.Error("Wavelog: stations server error", "status", resp.StatusCode, "body", strings.TrimSpace(string(respBody)))
		return nil, fmt.Errorf("server error: HTTP %d — %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var stations []StationProfile
	if err := json.Unmarshal(respBody, &stations); err != nil {
		applog.Error("Wavelog: parse stations failed", "error", err)
		return nil, fmt.Errorf("parse stations: %w", err)
	}

	if len(stations) == 0 {
		applog.Warn("Wavelog: no station profiles found")
		return nil, fmt.Errorf("no station profiles found")
	}

	applog.InfoDetail("Wavelog: stations fetched", fmt.Sprintf("count=%d", len(stations)))
	return stations, nil
}

// TestStation validates that a specific station profile is reachable.
func TestStation(baseURL, apiKey, stationID string) error {
	applog.Debug("Wavelog: testing station", "station_id", stationID)
	if stationID == "" {
		return fmt.Errorf("no station selected")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/api/get_wp_stats"

	payload := map[string]string{
		"key":        apiKey,
		"station_id": stationID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		applog.Error("Wavelog: marshal station test failed", "error", err)
		return err
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		applog.Error("Wavelog: station test failed", "station_id", stationID, "error", err)
		return fmt.Errorf("station test failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("Wavelog: read station test response failed", "error", err)
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		applog.Error("Wavelog: station test server error", "station_id", stationID, "status", resp.StatusCode)
		return fmt.Errorf("station error: HTTP %d — %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	applog.InfoDetail("Wavelog: station test OK", fmt.Sprintf("station_id=%s", stationID))
	return nil
}

// PostQSO uploads a QSO in ADIF format to Wavelog.
func PostQSO(baseURL, apiKey, stationID, adifStr string) error {
	_, err := PostQSOWithResult(baseURL, apiKey, stationID, adifStr)
	return err
}

// QSOUploadResult carries structured info about a Wavelog upload response.
type QSOUploadResult struct {
	Status        string   `json:"status"`
	ADIFCount     int      `json:"adif_count"`
	ADIFErrors    int      `json:"adif_errors"`
	Messages      []string `json:"messages"`
	AllDuplicates bool
}

// PostQSOWithResult uploads a QSO in ADIF format to Wavelog and returns
// structured result info. When all rejected QSOs are duplicates, the
// returned error is nil and AllDuplicates is set to true.
func PostQSOWithResult(baseURL, apiKey, stationID, adifStr string) (*QSOUploadResult, error) {
	applog.Debug("Wavelog: posting QSO")
	if baseURL == "" || apiKey == "" || stationID == "" || adifStr == "" {
		return nil, fmt.Errorf("missing required parameters")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/index.php/api/qso"

	payload := map[string]string{
		"key":                apiKey,
		"station_profile_id": stationID,
		"type":               "adif",
		"string":             adifStr,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		applog.Error("Wavelog: marshal QSO payload failed", "error", err)
		return nil, err
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		applog.Error("Wavelog: QSO upload failed", "error", err)
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("Wavelog: read QSO response failed", "error", err)
		return nil, fmt.Errorf("read response: %w", err)
	}
	bodyStr := strings.TrimSpace(string(respBody))

	// Try to parse structured response
	var result QSOUploadResult
	if jsonErr := json.Unmarshal(respBody, &result); jsonErr == nil {
		// Check if all errors are duplicates
		if result.Status == "abort" && result.ADIFErrors > 0 && len(result.Messages) > 1 {
			allDup := true
			for _, m := range result.Messages[1:] { // messages[0] is usually empty
				if m != "" && !strings.Contains(m, "Duplicate for") {
					allDup = false
					break
				}
			}
			if allDup && result.ADIFErrors > 0 {
				result.AllDuplicates = true
				applog.InfoDetail("Wavelog: all QSOs already present (duplicates)", fmt.Sprintf("count=%d", result.ADIFCount))
				return &result, nil
			}
		}
	}

	if resp.StatusCode >= 400 {
		applog.Error("Wavelog: QSO upload server error", "status", resp.StatusCode, "body", bodyStr)
		return &result, fmt.Errorf("server error: HTTP %d — %s", resp.StatusCode, bodyStr)
	}

	applog.InfoDetail("Wavelog: QSO uploaded", fmt.Sprintf("status=%d", resp.StatusCode))
	return &result, nil
}
