package wavelog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

// httpClient is shared across all regular Wavelog API v2 calls.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// downloadClient has a very long timeout — used only for the paginated
// ADIF export (format=adif) which streams large responses.
var downloadClient = &http.Client{Timeout: 5 * time.Minute}

// syncClient is the short-timeout client for config-save station syncs —
// a hung server must not freeze the wizard or the logbook menu for long.
var syncClient = &http.Client{Timeout: 4 * time.Second}

// StationProfile represents a Wavelog station location.
// Values come from GET /api/v2/station; the v1 API field names are kept
// so downstream call sites did not need renaming.
type StationProfile struct {
	ID         string
	Name       string
	Gridsquare string
	Callsign   string
	Active     bool
}

// V1KeyRequiredMsg is shown to users still using a legacy v1 API key.
// Since CQOps 0.11.0 only v2 tokens (wl2_ prefix) are accepted.
const V1KeyRequiredMsg = "Wavelog API v2 token required since CQOps 0.11.0 — create a wl2_ token in Wavelog"

// TestConnection validates that the Wavelog instance is reachable and the
// API v2 token is valid. Since CQOps 0.11.0 only v2 tokens (wl2_ prefix)
// are accepted — legacy v1 keys are rejected up front with an actionable
// message instead of a confusing HTTP error.
func TestConnection(baseURL, apiKey string) error {
	applog.Debug("Wavelog: testing connection")
	if baseURL == "" || apiKey == "" {
		return fmt.Errorf("URL and API key required")
	}
	if !IsV2Token(apiKey) {
		return fmt.Errorf("%s", V1KeyRequiredMsg)
	}
	if err := StatusOK(baseURL); err != nil {
		applog.Error("Wavelog: status check failed", "url", baseURL, "error", err)
		return err
	}
	info, err := TokenWhoami(baseURL, apiKey)
	if err != nil {
		applog.Error("Wavelog: token validation failed", "url", baseURL, "error", err)
		return err
	}
	applog.InfoDetail("Wavelog: connected",
		fmt.Sprintf("token=%s owner=%s scopes=%d", info.Name, info.Owner, len(info.Scopes)))
	return nil
}

// FetchStations retrieves station profiles from the Wavelog API v2.
func FetchStations(baseURL, apiKey string) ([]StationProfile, error) {
	applog.Debug("Wavelog: fetching stations")
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("URL and API key required")
	}
	if !IsV2Token(apiKey) {
		return nil, fmt.Errorf("%s", V1KeyRequiredMsg)
	}
	stations, err := fetchStationsV2(baseURL, apiKey)
	if err != nil {
		applog.Error("Wavelog: fetch stations failed", "url", baseURL, "error", err)
		return nil, err
	}
	if len(stations) == 0 {
		applog.Warn("Wavelog: no station profiles found")
		return nil, fmt.Errorf("no station profiles found")
	}
	applog.InfoDetail("Wavelog: stations fetched", fmt.Sprintf("count=%d", len(stations)))
	return stations, nil
}

// PrivateLookupResult holds the callsign information returned by
// GET /api/v2/lookup (detail=full). v2 returns typed booleans, so there
// is no more true/"true"/1 guessing — fields are stored directly.
type PrivateLookupResult struct {
	callsign          string
	dxcc              string
	dxccID            string
	dxccCQZ           string
	dxccITUZ          string
	name              string
	gridsquare        string
	location          string
	state             string
	worked            bool
	workedBand        bool
	workedBandMode    bool
	lotw              bool
	dxccConfirmed     bool
	confirmedBand     bool
	confirmedBandMode bool
}

// newLookupResult converts a v2 lookup data block into the result type.
func newLookupResult(d *lookupData) *PrivateLookupResult {
	return &PrivateLookupResult{
		callsign:          d.Callsign,
		dxcc:              d.DXCC,
		dxccID:            d.DXCCID,
		dxccCQZ:           d.DXCCCQZ,
		dxccITUZ:          strconv.Itoa(d.DXCCITUZ),
		name:              d.Name,
		gridsquare:        d.Gridsquare,
		location:          d.Location,
		state:             d.State,
		worked:            d.CallWorked,
		workedBand:        d.CallWorkedBand,
		workedBandMode:    d.CallWorkedBandMode,
		lotw:              truthy(d.LotwMember),
		dxccConfirmed:     d.DXCCConfirmed,
		confirmedBand:     d.CallConfirmedBand,
		confirmedBandMode: d.CallConfirmedBandMode,
	}
}

// truthy interprets Wavelog's lotw_member value. It is usually a member
// number string; empty and explicit falsy values mean "not a member".
func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "false", "0", "no", "n":
		return false
	}
	return true
}

// Callsign returns the looked-up callsign.
func (r *PrivateLookupResult) Callsign() string { return r.callsign }

// Name returns the operator name.
func (r *PrivateLookupResult) Name() string { return r.name }

// Worked returns call_worked.
func (r *PrivateLookupResult) Worked() bool { return r.worked }

// WorkedBand returns call_worked_band.
func (r *PrivateLookupResult) WorkedBand() bool { return r.workedBand }

// WorkedBandMode returns call_worked_band_mode.
func (r *PrivateLookupResult) WorkedBandMode() bool { return r.workedBandMode }

// LoTW returns lotw_member.
func (r *PrivateLookupResult) LoTW() bool { return r.lotw }

// DXCCConfirmed returns dxcc_confirmed.
func (r *PrivateLookupResult) DXCCConfirmed() bool { return r.dxccConfirmed }

// ConfirmedBand returns call_confirmed_band.
func (r *PrivateLookupResult) ConfirmedBand() bool { return r.confirmedBand }

// ConfirmedBandMode returns call_confirmed_band_mode.
func (r *PrivateLookupResult) ConfirmedBandMode() bool { return r.confirmedBandMode }

// Grid returns the gridsquare.
func (r *PrivateLookupResult) Grid() string { return r.gridsquare }

// DXCCID returns the numeric DXCC entity ID (e.g. "269" for Poland).
func (r *PrivateLookupResult) DXCCID() string { return r.dxccID }

// DXCCName returns the DXCC entity name (e.g. "POLAND").
func (r *PrivateLookupResult) DXCCName() string { return r.dxcc }

// QTH returns the location (city/address).
func (r *PrivateLookupResult) QTH() string { return r.location }

// Country returns the DXCC country name.
func (r *PrivateLookupResult) Country() string { return r.dxcc }

// CQZone returns the DXCC CQ zone.
func (r *PrivateLookupResult) CQZone() string { return r.dxccCQZ }

// ITUZone returns the DXCC ITU zone.
func (r *PrivateLookupResult) ITUZone() string { return r.dxccITUZ }

// State returns the state.
func (r *PrivateLookupResult) State() string { return r.state }

// PrivateLookup queries the Wavelog API v2 for callsign worked/confirmed
// data. Optional band, mode and stationProfileID scope the per-band/mode
// flags the legacy private_lookup endpoint provided.
func PrivateLookup(baseURL, apiKey, callsign, band, mode, stationProfileID string) (*PrivateLookupResult, error) {
	if baseURL == "" || apiKey == "" || callsign == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("callsign", callsign)
	q.Set("detail", "full")
	if band != "" {
		q.Set("band", band)
	}
	if mode != "" {
		q.Set("mode", mode)
	}
	if stationProfileID != "" {
		if idInt, err := ParseStationID(stationProfileID); err == nil {
			q.Set("station_ids", strconv.Itoa(idInt))
		}
	}

	_, body, err := v2Request(http.MethodGet, baseURL, apiKey, "/lookup", q, nil)
	if err != nil {
		return nil, err
	}
	var data lookupData
	if _, err := v2DecodeData(body, &data); err != nil {
		return nil, err
	}
	return newLookupResult(&data), nil
}

// ParseStationID extracts the numeric station profile id from a value that
// may also carry a display label (the station picker renders e.g.
// "1 — SP9SPM (Home) JO90" into the read-only field). Returns an error when
// no leading integer is present.
func ParseStationID(s string) (int, error) {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, fmt.Errorf("invalid station_profile_id %q", s)
	}
	return strconv.Atoi(s[:end])
}

// SanitizeStationID strips a display label from a station profile value,
// returning just the numeric id. Non-numeric input is returned trimmed.
func SanitizeStationID(s string) string {
	if id, err := ParseStationID(s); err == nil {
		return strconv.Itoa(id)
	}
	return strings.TrimSpace(s)
}

// TestStation validates that a specific station profile is reachable.
func TestStation(baseURL, apiKey, stationID string) error {
	applog.Debug("Wavelog: testing station", "station_id", stationID)
	if stationID == "" {
		return fmt.Errorf("no station selected")
	}
	idInt, err := ParseStationID(stationID)
	if err != nil {
		return err
	}
	stationID = strconv.Itoa(idInt)
	_, _, err = v2Request(http.MethodGet, baseURL, apiKey, "/station/"+stationID, nil, nil)
	if err != nil {
		applog.Error("Wavelog: station test failed", "station_id", stationID, "error", err)
		return err
	}
	applog.InfoDetail("Wavelog: station test OK", fmt.Sprintf("station_id=%s", stationID))
	return nil
}

// QSOUploadResult carries structured info about a Wavelog upload response.
// v2 reports {parsed, imported, skipped, messages}; AllDuplicates is
// derived: nothing imported and everything skipped means the QSOs were
// already present.
type QSOUploadResult struct {
	Status        string
	ADIFCount     int
	ADIFErrors    int
	Messages      []string
	AllDuplicates bool
}

// PostQSOWithResult uploads a QSO in ADIF format to the Wavelog API v2 and
// returns structured result info. When all parsed QSOs were skipped as
// duplicates, the returned error is nil and AllDuplicates is set.
func PostQSOWithResult(baseURL, apiKey, stationID, adifStr string) (*QSOUploadResult, error) {
	applog.Debug("Wavelog: posting QSO")
	if baseURL == "" || apiKey == "" || stationID == "" || adifStr == "" {
		return nil, fmt.Errorf("missing required parameters")
	}
	stationIDInt, err := ParseStationID(stationID)
	if err != nil {
		return nil, fmt.Errorf("invalid station_profile_id %q: %w", stationID, err)
	}

	payload := struct {
		ImportType       string `json:"import_type"`
		StationProfileID int    `json:"station_profile_id"`
		ADIF             string `json:"adif"`
	}{
		ImportType:       "adif",
		StationProfileID: stationIDInt,
		ADIF:             adifStr,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		applog.Error("Wavelog: marshal QSO payload failed", "error", err)
		return nil, err
	}

	_, respBody, err := v2Request(http.MethodPost, baseURL, apiKey, "/qso", nil, body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusBadRequest {
			result := &QSOUploadResult{Status: "validation_error", Messages: messagesFromError(apiErr)}
			return result, err
		}
		applog.Error("Wavelog: QSO upload failed", "error", err)
		return nil, FriendlyError(err)
	}

	var imported v2QSOImport
	if _, err := v2DecodeData(respBody, &imported); err != nil {
		applog.Error("Wavelog: read QSO response failed", "error", err)
		return nil, FriendlyError(fmt.Errorf("read response: %w", err))
	}

	result := &QSOUploadResult{
		Status:     "ok",
		ADIFCount:  imported.Imported,
		ADIFErrors: maxInt(0, imported.Parsed-imported.Imported-imported.Skipped),
		Messages:   imported.Messages,
	}
	if imported.Parsed > 0 && imported.Imported == 0 && imported.Skipped == imported.Parsed {
		result.AllDuplicates = true
		applog.InfoDetail("Wavelog: all QSOs already present (duplicates)", fmt.Sprintf("count=%d", imported.Parsed))
	}
	applog.InfoDetail("Wavelog: QSO upload finished",
		fmt.Sprintf("parsed=%d imported=%d skipped=%d", imported.Parsed, imported.Imported, imported.Skipped))
	return result, nil
}

// messagesFromError extracts user-facing messages from a v2 validation
// error envelope (details may carry an array of messages).
func messagesFromError(apiErr *APIError) []string {
	if apiErr.Details == nil {
		return []string{apiErr.Message}
	}
	if msgs, ok := apiErr.Details["messages"].([]any); ok {
		out := make([]string, 0, len(msgs))
		for _, m := range msgs {
			out = append(out, fmt.Sprint(m))
		}
		if len(out) > 0 {
			return out
		}
	}
	if apiErr.Message != "" {
		return []string{apiErr.Message}
	}
	return nil
}

// maxInt returns the larger of two ints (math.Max exists only for floats).
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CreateQSOInput carries the fields for a single-QSO JSON create.
// Frequencies are in Hz (the v2 JSON convention, unlike ADIF's MHz).
type CreateQSOInput struct {
	StationProfileID int
	Call             string
	Band             string
	Mode             string // mode or submode, e.g. "SSB" or "FT8"
	QSODate          string // YYYY-MM-DD
	TimeOn           string // HHMM or HHMMSS
	TimeOff          string
	RSTSent          string
	RSTRcvd          string
	Gridsquare       string
	Name             string
	QTH              string
	Comment          string
	Notes            string
	FreqHz           int64
	FreqRxHz         int64
}

// CreateQSO creates a single QSO via POST /api/v2/qso (import_type=json).
// Unlike the bulk ADIF import, the response carries the created object
// including the remote id — which CQOps stores locally so future edits can
// target PATCH /api/v2/qso/{id}.
// Returns isDuplicate=true when Wavelog rejected the QSO as a duplicate.
func CreateQSO(baseURL, apiKey string, in CreateQSOInput) (remoteID int64, isDuplicate bool, err error) {
	payload := map[string]any{
		"station_profile_id": in.StationProfileID,
		"call":               in.Call,
		"band":               in.Band,
		"mode":               in.Mode,
		"qso_date":           in.QSODate,
		"time_on":            in.TimeOn,
	}
	addStr := func(key, v string) {
		if v != "" {
			payload[key] = v
		}
	}
	addStr("time_off", in.TimeOff)
	addStr("rst_sent", in.RSTSent)
	addStr("rst_rcvd", in.RSTRcvd)
	addStr("gridsquare", in.Gridsquare)
	addStr("name", in.Name)
	addStr("qth", in.QTH)
	addStr("comment", in.Comment)
	addStr("notes", in.Notes)
	if in.FreqHz > 0 {
		payload["freq"] = in.FreqHz
	}
	if in.FreqRxHz > 0 {
		payload["freq_rx"] = in.FreqRxHz
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, false, err
	}

	_, respBody, err := v2Request(http.MethodPost, baseURL, apiKey, "/qso", nil, body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			// Live duplicate behavior: a conflicting QSO is reported as 409
			// conflict or as 400 validation_error with a "Duplicate" message.
			if apiErr.Code == "conflict" ||
				(apiErr.Code == "validation_error" && strings.Contains(strings.ToLower(apiErr.Message), "duplicate")) {
				return 0, true, nil
			}
		}
		return 0, false, err
	}

	// Success is the created object; be defensive about bulk-summary shaped
	// responses some server versions may return.
	var data struct {
		ID       int64 `json:"id"`
		Imported int   `json:"imported"`
		Skipped  int   `json:"skipped"`
	}
	if _, err := v2DecodeData(respBody, &data); err != nil {
		return 0, false, err
	}
	if data.Imported == 0 && data.Skipped > 0 {
		return 0, true, nil
	}
	return data.ID, false, nil
}

// DeleteQSO deletes a QSO by remote id via DELETE /api/v2/qso/{id}.
// Returns nil on success (204). A QSO that does not exist or is not owned
// by the token reports an APIError with code not_found (404).
func DeleteQSO(baseURL, apiKey string, remoteID int64) error {
	if remoteID <= 0 {
		return fmt.Errorf("invalid remote id %d", remoteID)
	}
	_, _, err := v2Request(http.MethodDelete, baseURL, apiKey, "/qso/"+strconv.FormatInt(remoteID, 10), nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// QSOData is one QSO as returned by the v2 REST API (GET /api/v2/qso/{id} and
// JSON list rows). Strings that can be null decode to "" on JSON null.
type QSOData struct {
	ID         int64  `json:"id"`
	StationID  int64  `json:"station_id"`
	QSODate    string `json:"qso_date"` // "YYYY-MM-DD HH:MM:SS"
	Mode       string `json:"mode"`
	Submode    string `json:"submode"`
	Freq       string `json:"freq"` // Hz, string-encoded
	FreqRx     string `json:"freq_rx"`
	Call       string `json:"call"`
	Band       string `json:"band"`
	BandRx     string `json:"band_rx"`
	RSTSent    string `json:"rst_sent"`
	RSTRcvd    string `json:"rst_rcvd"`
	Gridsquare string `json:"gridsquare"`
	Name       string `json:"name"`
	Comment    string `json:"comment"`
	Notes      string `json:"notes"`
	QTH        string `json:"qth"`
	TXPower    string `json:"tx_pwr"`
	PropMode   string `json:"prop_mode"`
	SatName    string `json:"sat_name"`
	SatMode    string `json:"sat_mode"`
	SOTARef    string `json:"sota_ref"`
	POTARef    string `json:"pota_ref"`
	WWFFRef    string `json:"wwff_ref"`
	IOTA       string `json:"iota"`
	SIG        string `json:"sig"`
	SIGInfo    string `json:"sig_info"`
	DarcDok    string `json:"darc_dok"`
	State      string `json:"state"`
	Cnty       string `json:"cnty"`
	CQZ        int    `json:"cqz"`
	ITUZ       int    `json:"ituz"`
	QSLVia     string `json:"qsl_via"`
	SRX        *int   `json:"srx"`
	STX        *int   `json:"stx"`
	SRXString  string `json:"srx_string"`
	STXString  string `json:"stx_string"`
}

// GetQSO fetches a single QSO from the Wavelog API v2 by remote id. Defensive
// about the data block shape: the REST form returns the object directly, but a
// single-element array is accepted too.
func GetQSO(baseURL, apiKey string, remoteID int64) (*QSOData, error) {
	if remoteID <= 0 {
		return nil, fmt.Errorf("invalid remote id %d", remoteID)
	}
	_, body, err := v2Request(http.MethodGet, baseURL, apiKey, "/qso/"+strconv.FormatInt(remoteID, 10), nil, nil)
	if err != nil {
		return nil, err
	}
	var qs QSOData
	if _, err := v2DecodeData(body, &qs); err == nil && qs.ID != 0 {
		return &qs, nil
	}
	var rows []QSOData
	if _, err := v2DecodeData(body, &rows); err != nil || len(rows) == 0 {
		if err != nil {
			return nil, fmt.Errorf("parse QSO response: %w", err)
		}
		return nil, fmt.Errorf("QSO response carried no data")
	}
	return &rows[0], nil
}

// UpdateQSOInput carries the fields for a PATCH /api/v2/qso/{id} update.
// Frequencies are in Hz (string-encoded when set, the v2 convention).
//
// Presence is separated from value: a nil pointer leaves the field
// untouched, while a pointer to an empty string or a nonpositive frequency
// explicitly CLEARS the field remotely (sent as JSON null, the API's
// supported clearing representation). Call, Band, Mode, QSODate and TimeOn
// are identity fields and are never cleared.
type UpdateQSOInput struct {
	Call       string
	Band       string
	Mode       string // canonical mode or submode, e.g. "SSB" or "FT8"
	QSODate    string // YYYY-MM-DD — must be sent together with TimeOn
	TimeOn     string // HH:MM:SS — must be sent together with QSODate
	RSTSent    *string
	RSTRcvd    *string
	Gridsquare *string
	Name       *string
	QTH        *string
	Comment    *string
	Notes      *string
	TXPower    *string
	SOTARef    *string
	POTARef    *string
	WWFFRef    *string
	IOTA       *string
	SIG        *string
	SIGInfo    *string
	FreqHz     *int64
	FreqRxHz   *int64
}

// UpdateQSO updates a QSO on the Wavelog API v2 via PATCH /api/v2/qso/{id}.
// The v2 API accepts partial bodies; qso_date (YYYY-MM-DD) and time_on
// (HH:MM:SS) must be supplied together. A missing QSO reports an APIError
// with code not_found. Cleared fields are sent as JSON null.
func UpdateQSO(baseURL, apiKey string, remoteID int64, in UpdateQSOInput) error {
	if remoteID <= 0 {
		return fmt.Errorf("invalid remote id %d", remoteID)
	}
	payload := map[string]any{}
	addStr := func(key, v string) {
		if v != "" {
			payload[key] = v
		}
	}
	// Clearable fields: nil = leave untouched, empty = clear (JSON null).
	addClear := func(key string, v *string) {
		if v == nil {
			return
		}
		if *v == "" {
			payload[key] = nil
			return
		}
		payload[key] = *v
	}
	addClearHz := func(key string, v *int64) {
		if v == nil {
			return
		}
		if *v <= 0 {
			payload[key] = nil
			return
		}
		payload[key] = strconv.FormatInt(*v, 10)
	}
	addStr("call", in.Call)
	addStr("band", in.Band)
	addStr("mode", in.Mode)
	if in.QSODate != "" && in.TimeOn != "" {
		payload["qso_date"] = in.QSODate
		payload["time_on"] = in.TimeOn
	}
	addClear("rst_sent", in.RSTSent)
	addClear("rst_rcvd", in.RSTRcvd)
	addClear("gridsquare", in.Gridsquare)
	addClear("name", in.Name)
	addClear("qth", in.QTH)
	addClear("comment", in.Comment)
	addClear("notes", in.Notes)
	addClear("tx_pwr", in.TXPower)
	addClear("sota_ref", in.SOTARef)
	addClear("pota_ref", in.POTARef)
	addClear("wwff_ref", in.WWFFRef)
	addClear("iota", in.IOTA)
	addClear("sig", in.SIG)
	addClear("sig_info", in.SIGInfo)
	addClearHz("freq", in.FreqHz)
	addClearHz("freq_rx", in.FreqRxHz)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, _, err = v2Request(http.MethodPatch, baseURL, apiKey, "/qso/"+strconv.FormatInt(remoteID, 10), nil, body)
	return err
}

// QSOIDKey identifies a QSO by its dedupe fields. Used to match local QSOs
// against remote ids from the JSON list for backfilling wavelog_id.
type QSOIDKey struct {
	Call    string
	Band    string
	Mode    string
	QSODate string // YYYY-MM-DD
	TimeOn  string // HHMMSS
}

// MakeQSOIDKey normalizes local QSO fields into a QSOIDKey (ADIF date in,
// ISO date out; case-insensitive fields upper-cased).
func MakeQSOIDKey(call, band, mode, qsoDate, timeOn string) QSOIDKey {
	date := qsoDate
	if len(date) == 8 {
		if t, err := time.Parse("20060102", date); err == nil {
			date = t.Format("2006-01-02")
		}
	}
	return QSOIDKey{
		Call:    strings.ToUpper(strings.TrimSpace(call)),
		Band:    strings.ToUpper(strings.TrimSpace(band)),
		Mode:    strings.ToUpper(strings.TrimSpace(mode)),
		QSODate: date,
		TimeOn:  strings.TrimSpace(timeOn),
	}
}

// MakeQSOIDKeyFromADIF builds a QSOIDKey from parsed ADIF fields using the
// same canonicalization the JSON sidecar uses: MFSK + submode collapses to
// the submode, and 4-digit times (HHMM) are padded to HHMMSS so they match
// the JSON list's seconds-precision time. A 6-digit ADIF time with nonzero
// seconds never matches a padded key.
func MakeQSOIDKeyFromADIF(call, band, mode, submode, qsoDate, timeOn string) QSOIDKey {
	key := MakeQSOIDKey(call, band, mode, qsoDate, timeOn)
	if key.Mode == "MFSK" {
		if sub := strings.ToUpper(strings.TrimSpace(submode)); sub != "" {
			key.Mode = sub
		}
	}
	if len(key.TimeOn) == 4 {
		key.TimeOn += "00"
	}
	return key
}

// qsoIDRow is one row of the JSON list used for remote-id backfill.
type qsoIDRow struct {
	ID      int64  `json:"id"`
	Call    string `json:"call"`
	Band    string `json:"band"`
	Mode    string `json:"mode"`
	Submode string `json:"submode"`
	QSODate string `json:"qso_date"` // "YYYY-MM-DD HH:MM:SS"
}

func (r qsoIDRow) key() QSOIDKey {
	date := ""
	if len(r.QSODate) >= 10 {
		date = r.QSODate[:10]
	}
	timeOn := ""
	if len(r.QSODate) >= 19 {
		timeOn = strings.ReplaceAll(r.QSODate[11:19], ":", "")
	}
	mode := strings.ToUpper(strings.TrimSpace(r.Mode))
	// Canonicalize the same way as internal/qso: MFSK + submode collapses
	// to the submode (MFSK+FT8 → FT8), otherwise the key would never match
	// a locally canonicalized QSO.
	if mode == "MFSK" {
		if sub := strings.ToUpper(strings.TrimSpace(r.Submode)); sub != "" {
			mode = sub
		}
	}
	return QSOIDKey{
		Call:    strings.ToUpper(strings.TrimSpace(r.Call)),
		Band:    strings.ToUpper(strings.TrimSpace(r.Band)),
		Mode:    mode,
		QSODate: date,
		TimeOn:  timeOn,
	}
}

// FindQSOIDs returns the remote ids of the newest `limit` QSOs for a station,
// keyed by their dedupe fields. Used to backfill remote ids after bulk ADIF
// uploads, whose summary response does not carry ids.
func FindQSOIDs(baseURL, apiKey, stationID string, limit int) (map[QSOIDKey]int64, error) {
	if idInt, err := ParseStationID(stationID); err == nil {
		stationID = strconv.Itoa(idInt)
	}
	q := url.Values{}
	q.Set("station_id", stationID)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	_, body, err := v2Request(http.MethodGet, baseURL, apiKey, "/qso", q, nil)
	if err != nil {
		return nil, err
	}
	var rows []qsoIDRow
	if _, err := v2DecodeData(body, &rows); err != nil {
		return nil, err
	}
	out := make(map[QSOIDKey]int64, len(rows))
	for _, r := range rows {
		k := r.key()
		if _, ok := out[k]; !ok {
			out[k] = r.ID
		}
	}
	return out, nil
}

// v2QSOListPageSize is the page size for the full-list reconciliation scan.
// The v2 JSON list caps per_page at 5000 — one page for typical logs.
const v2QSOListPageSize = 5000

// FetchAllQSOIDs downloads the complete remote QSO list for a station as a
// dedupe-key → remote-id map. Used to reconcile migrated local logs before an
// upload: local rows that already exist remotely get their wavelog_id without
// ever being re-uploaded, and only genuinely new QSOs go to the server.
// The list is newest-first, so on duplicate keys the newest id wins.
func FetchAllQSOIDs(baseURL, apiKey, stationID string) (map[QSOIDKey]int64, error) {
	if idInt, err := ParseStationID(stationID); err == nil {
		stationID = strconv.Itoa(idInt)
	}
	out := make(map[QSOIDKey]int64)
	for page := 1; ; page++ {
		if page > 200 {
			return nil, fmt.Errorf("reconcile scan exceeded 200 pages — log is unexpectedly large")
		}
		q := url.Values{}
		q.Set("station_id", stationID)
		q.Set("per_page", strconv.Itoa(v2QSOListPageSize))
		q.Set("page", strconv.Itoa(page))
		_, body, err := v2RequestDownload(http.MethodGet, baseURL, apiKey, "/qso", q, nil)
		if err != nil {
			return nil, err
		}
		var rows []qsoIDRow
		meta, err := v2DecodeData(body, &rows)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			k := r.key()
			if _, ok := out[k]; !ok {
				out[k] = r.ID
			}
		}
		if !meta.HasMore || len(rows) == 0 {
			break
		}
	}
	applog.InfoDetail("Wavelog: reconciled remote QSO list",
		fmt.Sprintf("rows=%d", len(out)))
	return out, nil
}

// FindQSOID returns the remote id for a single QSO identified by its dedupe
// fields, or 0 when it is not on the server. Used after a duplicate-create
// response so the local row can still learn its remote id.
func FindQSOID(baseURL, apiKey, stationID, call, qsoDateISO, timeOn, band, mode string) (int64, error) {
	if idInt, err := ParseStationID(stationID); err == nil {
		stationID = strconv.Itoa(idInt)
	}
	q := url.Values{}
	q.Set("callsign", call)
	q.Set("station_id", stationID)
	_, body, err := v2Request(http.MethodGet, baseURL, apiKey, "/qso", q, nil)
	if err != nil {
		return 0, err
	}
	var rows []qsoIDRow
	if _, err := v2DecodeData(body, &rows); err != nil {
		return 0, err
	}
	want := QSOIDKey{
		Call:    strings.ToUpper(strings.TrimSpace(call)),
		Band:    strings.ToUpper(strings.TrimSpace(band)),
		Mode:    strings.ToUpper(strings.TrimSpace(mode)),
		QSODate: qsoDateISO,
		TimeOn:  strings.TrimSpace(timeOn),
	}
	for _, r := range rows {
		k := r.key()
		if k.Call == want.Call && k.QSODate == want.QSODate && k.TimeOn == want.TimeOn &&
			k.Band == want.Band && (k.Mode == want.Mode || strings.ToUpper(r.Submode) == want.Mode) {
			return r.ID, nil
		}
	}
	return 0, nil
}

// ContactsResponse from GET /api/v2/qso?format=adif.
// ADIFPath contains the path to a temporary file with the raw ADIF data.
// The caller is responsible for removing the file when done.
type ContactsResponse struct {
	ExportedQSOs     int
	LastFetchedIDRaw json.Number
	ADIFPath         string // temp file path
	ADIFSize         int64  // file size in bytes for progress
	TotalRows        int    // meta.total of the export (expected rows)

	// WavelogIDsByKey maps each exported row's dedupe identity to its
	// remote id. nil when ids could not be verified. Rows are matched by
	// identity, never by position, so a missing or mismatched id page
	// leaves its rows unassigned instead of shifting ids.
	WavelogIDsByKey map[QSOIDKey]int64
}

// LastFetchedID returns the last fetched ID as int64, defaulting to 0 on parse failure.
func (r *ContactsResponse) LastFetchedID() int64 {
	v, err := r.LastFetchedIDRaw.Int64()
	if err != nil {
		return 0
	}
	return v
}

// v2ADIFPageSize is the batch size for the paginated ADIF export. Small
// enough to keep the UI progress bar moving frequently (250 rows per page
// ≈ 7 pages for a 1600-QSO log), large enough to keep request overhead low.
const v2ADIFPageSize = 250

// fetchQSOIDsPage returns the remote ids of the QSOs in the same
// since_id/per_page window as an ADIF export page, keyed by their dedupe
// identity. The JSON list is ordered newest first; matching is by verified
// identity (call/band/mode/date/time), never by position, so a missing or
// mismatched page can never shift ids onto other records.
func fetchQSOIDsPage(ctx context.Context, baseURL, apiKey, stationID string, sinceID int64) (map[QSOIDKey]int64, error) {
	q := url.Values{}
	q.Set("since_id", strconv.FormatInt(sinceID, 10))
	q.Set("station_id", stationID)
	q.Set("per_page", strconv.Itoa(v2ADIFPageSize))

	_, body, err := v2RequestDownloadCtx(ctx, http.MethodGet, baseURL, apiKey, "/qso", q, nil)
	if err != nil {
		return nil, err
	}
	var rows []qsoIDRow
	if _, err := v2DecodeData(body, &rows); err != nil {
		return nil, err
	}
	out := make(map[QSOIDKey]int64, len(rows))
	for _, r := range rows {
		k := r.key()
		if _, ok := out[k]; !ok {
			out[k] = r.ID
		}
	}
	return out, nil
}

// FetchContacts pulls QSOs from the Wavelog API v2 as ADIF since the given
// fetchFromID. It streams every page straight into the temporary file and
// returns when the export is complete. See FetchContactsProgressCtx for the
// pagination details.
func FetchContacts(baseURL, apiKey, stationID string, fetchFromID int64) (*ContactsResponse, error) {
	return FetchContactsProgressCtx(context.Background(), baseURL, apiKey, stationID, fetchFromID, nil)
}

// FetchContactsProgress pulls QSOs from the Wavelog API v2 as ADIF since the
// given fetchFromID, reporting per-page progress. See FetchContactsProgressCtx.
func FetchContactsProgress(baseURL, apiKey, stationID string, fetchFromID int64, onPage func(exported, total int)) (*ContactsResponse, error) {
	return FetchContactsProgressCtx(context.Background(), baseURL, apiKey, stationID, fetchFromID, onPage)
}

// FetchContactsProgressCtx pulls QSOs from the Wavelog API v2 as ADIF since
// the given fetchFromID. The v2 export is paginated: each GET returns up to
// v2ADIFPageSize (250) rows with meta.has_more and meta.total. Pages are
// fetched with since_id=lastfetchedid until has_more is false or a page
// reports zero rows. Each page is written to the temporary file immediately
// — the whole log is never held in memory — and headers of pages after the
// first are stripped so the result stays a single valid ADIF document.
//
// Cancelling ctx aborts in-flight page requests immediately.
//
// onPage, when non-nil, is called after every page with the cumulative
// exported count and the expected total (meta.total).
func FetchContactsProgressCtx(ctx context.Context, baseURL, apiKey, stationID string, fetchFromID int64, onPage func(exported, total int)) (*ContactsResponse, error) {
	applog.DebugDetail("Wavelog: fetching contacts",
		fmt.Sprintf("url=%s station_id=%s from_id=%d", baseURL, stationID, fetchFromID))
	if baseURL == "" || apiKey == "" || stationID == "" {
		return nil, fmt.Errorf("missing required parameters")
	}
	sidInt, err := ParseStationID(stationID)
	if err != nil {
		return nil, fmt.Errorf("invalid station_id %q: %w", stationID, err)
	}
	stationID = strconv.Itoa(sidInt)

	// Create the temp file up front and stream pages into it.
	f, err := os.CreateTemp("", "cqops-wl-download-*.adif")
	if err != nil {
		applog.Error("Wavelog: failed to create temp file", "error", err)
		return nil, fmt.Errorf("temp file: %w", err)
	}
	fail := func(err error) (*ContactsResponse, error) {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}

	expectedTotal := 0
	totalExported := 0
	lastID := fetchFromID
	sinceID := fetchFromID
	firstPage := true
	var remoteIDs map[QSOIDKey]int64

	for {
		q := url.Values{}
		q.Set("format", "adif")
		q.Set("since_id", strconv.FormatInt(sinceID, 10))
		q.Set("station_id", stationID)
		q.Set("per_page", strconv.Itoa(v2ADIFPageSize))

		_, body, err := v2RequestDownloadCtx(ctx, http.MethodGet, baseURL, apiKey, "/qso", q, nil)
		if err != nil {
			applog.ErrorDetail("Wavelog: fetch contacts HTTP error",
				fmt.Sprintf("url=%s station_id=%s error=%v", baseURL, stationID, err))
			return fail(FriendlyError(fmt.Errorf("fetch failed: %w", err)))
		}

		var exported v2ADIFExport
		meta, err := v2DecodeData(body, &exported)
		if err != nil {
			applog.ErrorDetail("Wavelog: unmarshal fetch response failed",
				fmt.Sprintf("url=%s error=%v", baseURL, err))
			return fail(fmt.Errorf("parse response: %w", err))
		}

		if expectedTotal == 0 && meta.Total > 0 {
			expectedTotal = meta.Total
		}

		if exported.Exported == 0 {
			// Nothing new since since_id.
			break
		}

		// Loop safety: the server must advance lastfetchedid between pages.
		if !firstPage && exported.LastFetchedID <= sinceID {
			return fail(fmt.Errorf("server did not advance lastfetchedid (stuck at %d)", sinceID))
		}

		totalExported += exported.Exported
		if exported.LastFetchedID > lastID {
			lastID = exported.LastFetchedID
		}

		adif := ""
		if exported.ADIF != nil {
			adif = *exported.ADIF
		}
		if !firstPage {
			// Strip the document header so the result stays a single valid
			// ADIF document.
			adif = stripADIFHeader(adif)
		}
		if _, err := f.WriteString(adif); err != nil {
			applog.Error("Wavelog: write temp file failed", "error", err)
			return fail(fmt.Errorf("write temp file: %w", err))
		}
		firstPage = false

		// Capture remote ids for the same window. Best-effort sidecar:
		// a failure here must not abort the download. Rows are keyed by
		// verified identity and merged only when the JSON window holds
		// exactly the same number of rows as the ADIF page — a failed or
		// mismatched page simply leaves its rows without ids.
		pageIDs, idErr := fetchQSOIDsPage(ctx, baseURL, apiKey, stationID, sinceID)
		if idErr != nil {
			applog.Warn("Wavelog: remote id capture failed", "error", idErr)
		} else if len(pageIDs) != exported.Exported {
			applog.Warn("Wavelog: remote id page membership mismatch",
				fmt.Sprintf("ids=%d exported=%d", len(pageIDs), exported.Exported))
		} else {
			if remoteIDs == nil {
				remoteIDs = make(map[QSOIDKey]int64)
			}
			for k, id := range pageIDs {
				if _, ok := remoteIDs[k]; !ok {
					remoteIDs[k] = id
				}
			}
		}

		if onPage != nil {
			total := expectedTotal
			if total < totalExported {
				total = totalExported
			}
			onPage(totalExported, total)
		}

		if !meta.HasMore {
			break
		}
		sinceID = lastID
	}

	fi, err := f.Stat()
	fileSize := int64(0)
	if err == nil {
		fileSize = fi.Size()
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	result := &ContactsResponse{
		ExportedQSOs:     totalExported,
		LastFetchedIDRaw: json.Number(strconv.FormatInt(lastID, 10)),
		ADIFPath:         f.Name(),
		ADIFSize:         fileSize,
		TotalRows:        expectedTotal,
		WavelogIDsByKey:  remoteIDs,
	}

	applog.InfoDetail("Wavelog: contacts fetched to file",
		fmt.Sprintf("path=%s size=%d exported=%d total=%d last_id=%d",
			result.ADIFPath, result.ADIFSize, result.ExportedQSOs, result.TotalRows, result.LastFetchedID()))
	return result, nil
}

// stripADIFHeader removes the ADIF header block (everything up to and
// including <EOH>) from a page that is being appended to an existing
// document.
func stripADIFHeader(adif string) string {
	idx := strings.Index(adif, "<EOH>")
	if idx < 0 {
		idx = strings.Index(adif, "<eoh>")
	}
	if idx < 0 {
		return adif
	}
	return adif[idx+len("<EOH>"):]
}
