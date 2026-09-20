package wavelog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/szporwolik/cqops/internal/applog"
)

// =============================================================================
// Wavelog API v2 client.
//
// CQOps speaks only the v2 API (since CQOps 0.11.0): Bearer-token auth,
// resource paths under /api/v2, and the {data,meta}/{error} envelopes.
// =============================================================================

// IsV2Token reports whether the API key is a Wavelog API v2 token.
// v2 tokens are prefixed wl2_; legacy v1 keys start with wl and are
// rejected by every v2 endpoint with 401 invalid_token.
func IsV2Token(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), "wl2_")
}

// APIError is a Wavelog API v2 error envelope.
type APIError struct {
	StatusCode int            // HTTP status code
	Code       string         // error.code, e.g. "invalid_token"
	Message    string         // error.message
	Details    map[string]any // error.details, e.g. {"required_scope": ...}
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Code
}

// RetryAfterSeconds returns details.retry_after for rate_limited errors.
func (e *APIError) RetryAfterSeconds() int {
	return intFromDetails(e.Details, "retry_after")
}

// RequiredScope returns details.required_scope for insufficient_scope errors.
func (e *APIError) RequiredScope() string {
	if v, ok := e.Details["required_scope"].(string); ok {
		return v
	}
	return ""
}

// intFromDetails extracts an int value from an error details map, tolerating
// both float64 (JSON numbers) and string encodings.
func intFromDetails(d map[string]any, key string) int {
	if d == nil {
		return 0
	}
	switch v := d[key].(type) {
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

// v2Meta is the meta block shared by every v2 success response.
type v2Meta struct {
	Timestamp  string `json:"timestamp"`
	Resource   string `json:"resource"`
	Method     string `json:"method"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	Count      int    `json:"count"`
	Total      int    `json:"total"`
	TotalPages int    `json:"total_pages"`
	HasMore    bool   `json:"has_more"`
}

// v2BaseURL appends the API v2 prefix to the instance URL. Wavelog 3.1+
// serves /api/v2 directly; the index.php form works too but is redundant.
// If the configured URL already ends with /api/v2 (or /index.php/api/v2),
// it is used as-is — so both "https://host" and "https://host/api/v2" work.
func v2BaseURL(baseURL string) string {
	b := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	for _, suffix := range []string{"/api/v2", "/index.php/api/v2"} {
		if strings.HasSuffix(b, suffix) {
			return b
		}
	}
	return b + "/api/v2"
}

// v2Request performs a regular API v2 request on the default client.
// The status endpoint passes an empty token (it is public).
func v2Request(method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	return v2RequestWithClient(httpClient, method, baseURL, token, path, query, body)
}

// v2RequestDownload performs a v2 request on the long-timeout download
// client (paginated ADIF exports).
func v2RequestDownload(method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	return v2RequestWithClient(downloadClient, method, baseURL, token, path, query, body)
}

func v2RequestWithClient(client *http.Client, method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	u := v2BaseURL(baseURL) + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return 0, nil, FriendlyError(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, FriendlyError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, FriendlyError(err)
	}

	if resp.StatusCode >= 400 {
		return resp.StatusCode, respBody, v2Error(resp.StatusCode, respBody)
	}
	return resp.StatusCode, respBody, nil
}

// v2Error decodes the v2 error envelope. Falls back to a plain HTTP error
// when the body is not a valid envelope.
func v2Error(status int, body []byte) error {
	var env struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Code != "" {
		return &APIError{
			StatusCode: status,
			Code:       env.Error.Code,
			Message:    env.Error.Message,
			Details:    env.Error.Details,
		}
	}
	return FriendlyError(fmt.Errorf("HTTP %d", status))
}

// v2DecodeData unmarshals the data block of a success envelope.
func v2DecodeData(body []byte, target any) (v2Meta, error) {
	var env struct {
		Data json.RawMessage `json:"data"`
		Meta v2Meta          `json:"meta"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return v2Meta{}, fmt.Errorf("parse response: %w", err)
	}
	if err := json.Unmarshal(env.Data, target); err != nil {
		return v2Meta{}, fmt.Errorf("parse response data: %w", err)
	}
	return env.Meta, nil
}

// StatusOK checks the public v2 status endpoint. No token is required.
func StatusOK(baseURL string) error {
	applog.Debug("Wavelog: checking v2 status")
	_, body, err := v2Request(http.MethodGet, baseURL, "", "/status", nil, nil)
	if err != nil {
		return err
	}
	var data struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if _, err := v2DecodeData(body, &data); err != nil {
		return err
	}
	if data.Status != "ok" {
		return fmt.Errorf("wavelog API returned status %q", data.Status)
	}
	return nil
}

// TokenInfo describes the API v2 token (whoami).
type TokenInfo struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Owner     string   `json:"owner"`
	UserID    int      `json:"user_id"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at"`
}

// HasScope reports whether the token carries the given scope.
func (t *TokenInfo) HasScope(scope string) bool {
	for _, s := range t.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// TokenWhoami fetches metadata about the current token and validates it.
func TokenWhoami(baseURL, token string) (*TokenInfo, error) {
	applog.Debug("Wavelog: fetching token info")
	_, body, err := v2Request(http.MethodGet, baseURL, token, "/token", nil, nil)
	if err != nil {
		return nil, err
	}
	var info TokenInfo
	if _, err := v2DecodeData(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// v2Station is one entry of GET /api/v2/station.
type v2Station struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Callsign   string `json:"callsign"`
	Gridsquare string `json:"gridsquare"`
	Active     bool   `json:"active"`
}

// fetchStationsV2 retrieves station locations from the v2 API.
func fetchStationsV2(baseURL, token string) ([]StationProfile, error) {
	_, body, err := v2Request(http.MethodGet, baseURL, token, "/station", nil, nil)
	if err != nil {
		return nil, err
	}
	var raw []v2Station
	if _, err := v2DecodeData(body, &raw); err != nil {
		return nil, err
	}
	stations := make([]StationProfile, 0, len(raw))
	for _, s := range raw {
		stations = append(stations, StationProfile{
			ID:         strconv.Itoa(s.ID),
			Name:       s.Name,
			Gridsquare: s.Gridsquare,
			Callsign:   s.Callsign,
			Active:     s.Active,
		})
	}
	return stations, nil
}

// lookupData is the data block of GET /api/v2/lookup (detail=full and
// detail=basic share these fields; basic simply omits the per-band/mode
// flags). Types match the v2 API: booleans are real booleans.
type lookupData struct {
	Callsign string `json:"callsign"`
	DXCC     string `json:"dxcc"`
	DXCCID   string `json:"dxcc_id"`
	DXCCLat  string `json:"dxcc_lat"`
	DXCCLong string `json:"dxcc_long"`
	DXCCCQZ  string `json:"dxcc_cqz"`
	DXCCFlag string `json:"dxcc_flag"`
	Cont     string `json:"cont"`

	Name       string `json:"name"`
	Gridsquare string `json:"gridsquare"`
	Location   string `json:"location"`
	IotaRef    string `json:"iota_ref"`
	State      string `json:"state"`
	USCounty   string `json:"us_county"`
	QSLManager string `json:"qsl_manager"`
	Bearing    string `json:"bearing"`

	WorkedBefore bool `json:"workedBefore"`

	CallWorked         bool `json:"call_worked"`
	CallWorkedBand     bool `json:"call_worked_band"`
	CallWorkedBandMode bool `json:"call_worked_band_mode"`

	LotwMember string `json:"lotw_member"`

	DXCCConfirmedOnBand     bool `json:"dxcc_confirmed_on_band"`
	DXCCConfirmedOnBandMode bool `json:"dxcc_confirmed_on_band_mode"`
	DXCCConfirmed           bool `json:"dxcc_confirmed"`

	CallConfirmed         bool `json:"call_confirmed"`
	CallConfirmedBand     bool `json:"call_confirmed_band"`
	CallConfirmedBandMode bool `json:"call_confirmed_band_mode"`

	SuffixSlash string    `json:"suffix_slash"`
	DXCCITUZ    int       `json:"dxcc_ituz"`
	LatLng      []float64 `json:"latlng"`
}

// v2QSOImport is the data block of a successful POST /api/v2/qso
// (both single JSON creates and bulk ADIF imports).
type v2QSOImport struct {
	Parsed   int      `json:"parsed"`
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Messages []string `json:"messages"`
}

// v2ADIFExport is the data block of GET /api/v2/qso?format=adif.
type v2ADIFExport struct {
	Exported      int     `json:"exported"`
	LastFetchedID int64   `json:"lastfetchedid"`
	ADIF          *string `json:"adif"` // null when nothing new
}
