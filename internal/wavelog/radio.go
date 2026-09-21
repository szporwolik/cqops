package wavelog

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/szporwolik/cqops/internal/applog"
)

// DefaultRadioName is the Wavelog radio device name CQOps pushes its live
// rig state to (frequency, mode, power).
const DefaultRadioName = "CQOps"

// Radio is one radio as returned by the API v2 radio endpoints.
// Frequencies are in Hz.
type Radio struct {
	ID          int64  `json:"id"`
	Name        string `json:"radio"`
	Frequency   int64  `json:"frequency"`
	FrequencyRx int64  `json:"frequency_rx"`
	Mode        string `json:"mode"`
	ModeRx      string `json:"mode_rx"`
	Power       int64  `json:"power"`
	PropMode    string `json:"prop_mode"`
	SatName     string `json:"sat_name"`
	UpdatedAt   string `json:"updated_at"`
}

// ListRadios returns all radios owned by the token. The list endpoint is
// not paginated.
func ListRadios(baseURL, apiKey string) ([]Radio, error) {
	_, body, err := v2Request(http.MethodGet, baseURL, apiKey, "/radio", nil, nil)
	if err != nil {
		return nil, err
	}
	var radios []Radio
	if _, err := v2DecodeData(body, &radios); err != nil {
		return nil, err
	}
	return radios, nil
}

// FindRadio returns the radio with the given name, or nil when there is none.
func FindRadio(baseURL, apiKey, name string) (*Radio, error) {
	radios, err := ListRadios(baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	for i := range radios {
		if radios[i].Name == name {
			return &radios[i], nil
		}
	}
	return nil, nil
}

// EnsureRadio returns the radio with the given name, creating it on Wavelog
// when it does not exist yet. The v2 radio resource uses an upsert model
// (POST creates the radio the first time a state is pushed for its name).
func EnsureRadio(baseURL, apiKey, name string) (*Radio, error) {
	r, err := FindRadio(baseURL, apiKey, name)
	if err != nil {
		return nil, err
	}
	if r != nil {
		return r, nil
	}
	return PushRadio(baseURL, apiKey, name, RadioState{})
}

// RadioState is one snapshot of live rig state pushed to Wavelog.
// Frequencies are in Hz, power in watts.
type RadioState struct {
	FrequencyHz   int64
	FrequencyRxHz int64
	Mode          string
	ModeRx        string
	PowerWatts    int64
}

// PushRadio pushes a rig-state snapshot to Wavelog (POST /api/v2/radio —
// upsert by name) and returns the stored radio, including its id.
// The API validates strictly: frequencies and power must be whole numbers
// and modes are capped at 10 characters.
func PushRadio(baseURL, apiKey, name string, s RadioState) (*Radio, error) {
	applog.Debug("Wavelog: pushing radio state", "radio", name,
		"freq_hz", s.FrequencyHz, "mode", s.Mode)
	payload := map[string]any{"radio": name}
	if s.FrequencyHz > 0 {
		payload["frequency"] = s.FrequencyHz
	}
	if s.FrequencyRxHz > 0 {
		payload["frequency_rx"] = s.FrequencyRxHz
	}
	if m := trimModeField(s.Mode); m != "" {
		payload["mode"] = m
	}
	if m := trimModeField(s.ModeRx); m != "" {
		payload["mode_rx"] = m
	}
	if s.PowerWatts > 0 {
		payload["power"] = s.PowerWatts
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	_, respBody, err := v2Request(http.MethodPost, baseURL, apiKey, "/radio", nil, body)
	if err != nil {
		return nil, err
	}
	var r Radio
	if _, err := v2DecodeData(respBody, &r); err != nil {
		return nil, fmt.Errorf("parse radio response: %w", err)
	}
	return &r, nil
}

// trimModeField caps a mode field at the API limit of 10 characters.
func trimModeField(m string) string {
	if len(m) > 10 {
		return m[:10]
	}
	return m
}
