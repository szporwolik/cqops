package wavelog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

// httpClient is shared across all Wavelog API calls.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// downloadClient has a very long timeout — used only for get_contacts_adif
// which streams large ADIF responses (9342 QSOs ≈ 20-60s).
var downloadClient = &http.Client{Timeout: 5 * time.Minute}

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
		return FriendlyError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("Wavelog: read response failed", "error", err)
		return FriendlyError(err)
	}

	if resp.StatusCode >= 400 {
		applog.Error("Wavelog: server error", "status", resp.StatusCode, "body", strings.TrimSpace(string(respBody)))
		return FriendlyError(fmt.Errorf("HTTP %d", resp.StatusCode))
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
		return nil, FriendlyError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("Wavelog: read stations response failed", "error", err)
		return nil, FriendlyError(err)
	}

	if resp.StatusCode >= 400 {
		applog.Error("Wavelog: stations server error", "status", resp.StatusCode, "body", strings.TrimSpace(string(respBody)))
		return nil, FriendlyError(fmt.Errorf("HTTP %d", resp.StatusCode))
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

// PrivateLookupResult holds the callsign information returned by api/private_lookup.
// Uses a generic map internally because Wavelog instances vary in how they
// serialise booleans (true/false vs "true"/"false"/1/0) and numbers.
type PrivateLookupResult struct {
	raw map[string]interface{}
}

// str returns the string value for key, or "" if missing/wrong type/nil.
func (r *PrivateLookupResult) str(key string) string {
	v, ok := r.raw[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	return s
}

// IsTrue returns true if the field value indicates a positive result.
// Handles bool true, string "true"/"1"/"yes", float64 non-zero, and any
// non-empty string that isn't explicitly falsy.
func (r *PrivateLookupResult) IsTrue(key string) bool {
	v, ok := r.raw[key]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(t) {
		case "true", "1", "yes", "y":
			return true
		case "false", "0", "no", "n", "":
			return false
		default:
			return true // assume truthy for unexpected values
		}
	case float64:
		return t != 0
	}
	return false
}

// Callsign returns the looked-up callsign.
func (r *PrivateLookupResult) Callsign() string { return r.str("callsign") }

// Name returns the operator name.
func (r *PrivateLookupResult) Name() string { return r.str("name") }

// Worked returns call_worked.
func (r *PrivateLookupResult) Worked() bool { return r.IsTrue("call_worked") }

// WorkedBand returns call_worked_band.
func (r *PrivateLookupResult) WorkedBand() bool { return r.IsTrue("call_worked_band") }

// WorkedBandMode returns call_worked_band_mode.
func (r *PrivateLookupResult) WorkedBandMode() bool { return r.IsTrue("call_worked_band_mode") }

// LoTW returns lotw_member.
func (r *PrivateLookupResult) LoTW() bool { return r.IsTrue("lotw_member") }

// DXCCConfirmed returns dxcc_confirmed.
func (r *PrivateLookupResult) DXCCConfirmed() bool { return r.IsTrue("dxcc_confirmed") }

// ConfirmedBand returns call_confirmed_band.
func (r *PrivateLookupResult) ConfirmedBand() bool { return r.IsTrue("call_confirmed_band") }

// ConfirmedBandMode returns call_confirmed_band_mode.
func (r *PrivateLookupResult) ConfirmedBandMode() bool { return r.IsTrue("call_confirmed_band_mode") }

// PrivateLookup queries the Wavelog API for callsign confirmation/worked data.
func PrivateLookup(baseURL, apiKey, callsign, band, mode string) (*PrivateLookupResult, error) {
	if baseURL == "" || apiKey == "" || callsign == "" {
		return nil, nil
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/api/private_lookup"

	payload := map[string]string{
		"key":      apiKey,
		"callsign": callsign,
	}
	if band != "" {
		payload["band"] = band
	}
	if mode != "" {
		payload["mode"] = mode
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, FriendlyError(fmt.Errorf("private_lookup: %w", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, FriendlyError(fmt.Errorf("read response: %w", err))
	}

	if resp.StatusCode >= 400 {
		return nil, FriendlyError(fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	applog.Debug("Wavelog: private_lookup raw response", "body", strings.TrimSpace(string(respBody)))

	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &PrivateLookupResult{raw: raw}, nil
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
		return FriendlyError(err)
	}
	defer resp.Body.Close()

	// Drain body for connection reuse; we only care about the status code.
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		applog.Error("Wavelog: station test server error", "station_id", stationID, "status", resp.StatusCode)
		return FriendlyError(fmt.Errorf("HTTP %d", resp.StatusCode))
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
		return nil, FriendlyError(fmt.Errorf("upload failed: %w", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.Error("Wavelog: read QSO response failed", "error", err)
		return nil, FriendlyError(fmt.Errorf("read response: %w", err))
	}
	bodyStr := strings.TrimSpace(string(respBody))

	// Try to parse structured response
	var result QSOUploadResult
	if jsonErr := json.Unmarshal(respBody, &result); jsonErr == nil {
		// Check if all errors are duplicates.
		if result.Status == "abort" && result.ADIFErrors > 0 {
			allDup := len(result.Messages) > 0
			for _, m := range result.Messages {
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
		return &result, fmt.Errorf("%s", uploadErrorDetail(&result, bodyStr))
	}

	applog.InfoDetail("Wavelog: QSO uploaded", fmt.Sprintf("status=%d", resp.StatusCode))
	return &result, nil
}

// ContactsResponse from api/get_contacts_adif.
// ADIFPath contains the path to a temporary file with the raw ADIF data.
// The caller is responsible for removing the file when done.
type ContactsResponse struct {
	ExportedQSOs     int         `json:"exported_qsos"`
	LastFetchedIDRaw json.Number `json:"lastfetchedid"`
	Message          string      `json:"message"`
	ADIFPath         string      `json:"-"` // temp file path
	ADIFSize         int64       `json:"-"` // file size in bytes for progress
}

// LastFetchedID returns the last fetched ID as int64, defaulting to 0 on parse failure.
func (r *ContactsResponse) LastFetchedID() int64 {
	v, err := r.LastFetchedIDRaw.Int64()
	if err != nil {
		return 0
	}
	return v
}

// FetchContacts pulls QSOs from Wavelog as ADIF since the given fetchFromID.
// The ADIF is saved to a temporary file; the caller must remove it.
func FetchContacts(baseURL, apiKey, stationID string, fetchFromID int64) (*ContactsResponse, error) {
	applog.DebugDetail("Wavelog: fetching contacts",
		fmt.Sprintf("url=%s station_id=%s from_id=%d", baseURL, stationID, fetchFromID))
	if baseURL == "" || apiKey == "" || stationID == "" {
		return nil, fmt.Errorf("missing required parameters")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/index.php/api/get_contacts_adif"

	var stationIDInt int
	if _, err := fmt.Sscanf(stationID, "%d", &stationIDInt); err != nil {
		return nil, fmt.Errorf("invalid station_id %q: %w", stationID, err)
	}

	payload := map[string]interface{}{
		"key":         apiKey,
		"station_id":  stationIDInt,
		"fetchfromid": fetchFromID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		applog.ErrorDetail("Wavelog: marshal fetch payload failed",
			fmt.Sprintf("error=%v", err))
		return nil, err
	}

	// Use the long-timeout client for this heavy endpoint.
	resp, err := downloadClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		applog.ErrorDetail("Wavelog: fetch contacts HTTP error",
			fmt.Sprintf("url=%s station_id=%s error=%v", url, stationID, err))
		return nil, FriendlyError(fmt.Errorf("fetch failed: %w", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		applog.ErrorDetail("Wavelog: read fetch response failed",
			fmt.Sprintf("url=%s error=%v", url, err))
		return nil, FriendlyError(fmt.Errorf("read response: %w", err))
	}

	if resp.StatusCode >= 400 {
		bodyStr := strings.TrimSpace(string(respBody))
		reason := extractAPIReason(bodyStr)
		if reason == "" {
			reason = bodyStr
		}
		applog.ErrorDetail("Wavelog: fetch contacts server error",
			fmt.Sprintf("url=%s station_id=%s status=%d reason=%s", url, stationID, resp.StatusCode, reason))
		return nil, fmt.Errorf("server error: HTTP %d — %s", resp.StatusCode, reason)
	}

	// Parse just the metadata fields; extract the ADIF string.
	var raw struct {
		ExportedQSOs     int         `json:"exported_qsos"`
		LastFetchedIDRaw json.Number `json:"lastfetchedid"`
		Message          string      `json:"message"`
		ADIF             string      `json:"adif"`
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		applog.ErrorDetail("Wavelog: unmarshal fetch response failed",
			fmt.Sprintf("url=%s error=%v", url, err))
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Write ADIF to temp file for line-by-line processing.
	f, err := os.CreateTemp("", "cqops-wl-download-*.adif")
	if err != nil {
		applog.Error("Wavelog: failed to create temp file", "error", err)
		return nil, fmt.Errorf("temp file: %w", err)
	}
	if _, err := f.WriteString(raw.ADIF); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	// Get file size for progress tracking.
	fi, err := os.Stat(f.Name())
	fileSize := int64(0)
	if err == nil {
		fileSize = fi.Size()
	}

	result := &ContactsResponse{
		ExportedQSOs:     raw.ExportedQSOs,
		LastFetchedIDRaw: raw.LastFetchedIDRaw,
		Message:          raw.Message,
		ADIFPath:         f.Name(),
		ADIFSize:         fileSize,
	}

	applog.InfoDetail("Wavelog: contacts fetched to file",
		fmt.Sprintf("path=%s size=%d exported=%d last_id=%d",
			result.ADIFPath, result.ADIFSize, result.ExportedQSOs, result.LastFetchedID()))
	return result, nil
}
