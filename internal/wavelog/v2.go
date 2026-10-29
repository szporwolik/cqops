package wavelog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	return v2RequestWithClient(clientForMethod(method), method, baseURL, token, path, query, body)
}

// clientForMethod picks the deadline by what the request does. A write may be
// committed by the server long after the client gave up, so it gets the longer
// writeClient deadline; reads stay on the short one so the typing path never
// waits half a minute for a callsign lookup.
func clientForMethod(method string) *http.Client {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return writeClient
	}
	return httpClient
}

// v2RequestRadio is the live rig-state path: every few seconds, cosmetic data,
// so its own short deadline (radioClient) applies.
func v2RequestRadio(method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	return v2RequestWithClient(radioClient, method, baseURL, token, path, query, body)
}

// v2RequestDownloadCtx is v2RequestDownload with request cancellation — a
// user aborting a download cancels in-flight page fetches immediately.
func v2RequestDownloadCtx(ctx context.Context, method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	return v2RequestWithClientCtx(ctx, downloadClient, method, baseURL, token, path, query, body)
}

// v2RequestSync performs a v2 request on the short-timeout client, used by
// config-save paths (wizard / logbook menu) that must not block the UI for
// long when the server is unreachable.
func v2RequestSync(method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	return v2RequestWithClient(syncClient, method, baseURL, token, path, query, body)
}

func v2RequestWithClient(client *http.Client, method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	return v2RequestWithClientCtx(context.Background(), client, method, baseURL, token, path, query, body)
}

func v2RequestWithClientCtx(ctx context.Context, client *http.Client, method, baseURL, token, path string, query url.Values, body []byte) (int, []byte, error) {
	u := v2BaseURL(baseURL) + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
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
		// Our own deadline firing is not the same as a refused connection: the
		// server may have received and committed the request. Callers decide
		// what to do with that, so it gets its own error type.
		if d, ok := clientDeadline(client, err); ok {
			return 0, nil, &TimeoutError{Timeout: d, err: err}
		}
		return 0, nil, FriendlyError(err)
	}
	defer resp.Body.Close()

	// Bound the response so a runaway or hostile body cannot exhaust memory
	// on low-end hardware. Reject rather than truncate, so an oversized body
	// surfaces as a clear error instead of a confusing parse failure.
	const maxRespBytes = 64 << 20
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes+1))
	if err != nil {
		return 0, nil, FriendlyError(err)
	}
	if len(respBody) > maxRespBytes {
		return resp.StatusCode, nil, fmt.Errorf("response too large (over %d MB)", maxRespBytes>>20)
	}

	if resp.StatusCode >= 400 {
		err := v2Error(resp.StatusCode, respBody)
		// A 5xx is not proof that nothing happened: a proxy or an upstream
		// timeout in front of the database can answer 502/504 for a write that
		// was committed, so it is typed too and reconciled before a retry.
		if resp.StatusCode >= 500 {
			return resp.StatusCode, respBody, &ServerError{Status: resp.StatusCode, err: err}
		}
		return resp.StatusCode, respBody, err
	}
	return resp.StatusCode, respBody, nil
}

// clientDeadline reports whether err is the client's own deadline firing (as
// opposed to a refused or reset connection), together with that deadline.
func clientDeadline(client *http.Client, err error) (time.Duration, bool) {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return client.Timeout, true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return client.Timeout, true
	}
	return 0, false
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
	Callsign string     `json:"callsign"`
	DXCC     flexString `json:"dxcc"`
	DXCCID   flexString `json:"dxcc_id"`
	DXCCLat  flexString `json:"dxcc_lat"`
	DXCCLong flexString `json:"dxcc_long"`
	DXCCCQZ  flexString `json:"dxcc_cqz"`
	DXCCFlag string     `json:"dxcc_flag"`
	Cont     string     `json:"cont"`

	Name       string     `json:"name"`
	Gridsquare string     `json:"gridsquare"`
	Location   string     `json:"location"`
	IotaRef    string     `json:"iota_ref"`
	State      string     `json:"state"`
	USCounty   string     `json:"us_county"`
	QSLManager string     `json:"qsl_manager"`
	Bearing    flexString `json:"bearing"`

	WorkedBefore bool `json:"workedBefore"`

	CallWorked         bool `json:"call_worked"`
	CallWorkedBand     bool `json:"call_worked_band"`
	CallWorkedBandMode bool `json:"call_worked_band_mode"`

	LotwMember lotwMemberValue `json:"lotw_member"`

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

// lotwMemberValue tolerates both shapes the API returns for lotw_member:
// a member-number string ("14") and a plain boolean (true/false), plus null
// and numeric 0/1. A bool-only field broke the whole lookup for calls whose
// record carries the boolean form.
type lotwMemberValue struct {
	Value bool
}

func (l *lotwMemberValue) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	switch {
	case raw == "null":
		l.Value = false
		return nil
	case strings.HasPrefix(raw, `"`):
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		l.Value = truthy(s)
		return nil
	case raw == "true":
		l.Value = true
		return nil
	case raw == "false":
		l.Value = false
		return nil
	default:
		var n float64
		if err := json.Unmarshal(b, &n); err != nil {
			return err
		}
		l.Value = n != 0
		return nil
	}
}

// flexString is a JSON string that also accepts a bare number, a boolean or
// null, keeping the text as it arrived. Wavelog is not consistent here: the
// lookup block documents `dxcc_id` as a string, but a real server answers
// `"dxcc_id": 269`. Decoding that into a string failed the WHOLE lookup
// ("cannot unmarshal number into Go struct field lookupData.dxcc_id of type
// string"), so the Wavelog provider returned nothing and the callbook chain
// still paid its timeout.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	switch {
	case raw == "" || raw == "null":
		*f = ""
		return nil
	case strings.HasPrefix(raw, `"`):
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	case raw == "true":
		*f = "1"
		return nil
	case raw == "false":
		*f = ""
		return nil
	default:
		var n json.Number
		if err := json.Unmarshal(b, &n); err != nil {
			// Anything else is kept verbatim rather than failing the lookup.
			*f = flexString(raw)
			return nil
		}
		*f = flexString(n.String())
		return nil
	}
}

func (f flexString) String() string { return string(f) }

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
