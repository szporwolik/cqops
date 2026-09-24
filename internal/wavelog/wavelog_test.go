package wavelog

import (
	"errors"
	"strings"
	"testing"
)

// =============================================================================
// FriendlyError tests
// =============================================================================

func TestFriendlyError_Nil(t *testing.T) {
	if err := FriendlyError(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestFriendlyError_DNS(t *testing.T) {
	cases := []struct{ msg, want string }{
		{"lookup wavelog.example.com: no such host", "cannot reach wavelog.example.com — check the URL"},
		{"dial tcp: lookup invalid.host: no such host", "cannot reach invalid.host — check the URL"},
		{"Name or service not known", "cannot reach server — check the URL"},
		{"lookup api.example.com: Name or service not known", "cannot reach api.example.com — check the URL"},
	}
	for _, c := range cases {
		err := FriendlyError(errors.New(c.msg))
		if err == nil || err.Error() != c.want {
			t.Errorf("msg=%q: got %q, want %q", c.msg, friendlyMsg(err), c.want)
		}
	}
}

func TestFriendlyError_ConnectionRefused(t *testing.T) {
	err := FriendlyError(errors.New("dial tcp 1.2.3.4:443: connect: connection refused"))
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("got %q", friendlyMsg(err))
	}
}

func TestFriendlyError_Timeout(t *testing.T) {
	cases := []string{
		"context deadline exceeded (Client.Timeout exceeded)",
		"Client.Timeout exceeded while awaiting headers",
		"dial tcp: i/o timeout",
		"Timeout",
	}
	for _, msg := range cases {
		err := FriendlyError(errors.New(msg))
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("msg=%q: got %q", friendlyMsg(err), msg)
		}
	}
}

func TestFriendlyError_HTTP(t *testing.T) {
	cases := []struct{ msg, want string }{
		{"HTTP 401", "invalid API key"},
		{"HTTP 401 \u2014 Station ID not accessible", "station profile not accessible"},
		{"HTTP 403", "access denied"},
		{"HTTP 404", "server not found"},
		{"HTTP 500", "server error"},
		{"HTTP 503", "server error"},
		{"HTTP 400", "request failed"},
		{"HTTP 429", "request failed"},
	}
	for _, c := range cases {
		err := FriendlyError(errors.New(c.msg))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("msg=%q: got %q, want substring %q", c.msg, friendlyMsg(err), c.want)
		}
	}
}

func TestFriendlyError_TLS(t *testing.T) {
	cases := []string{
		"x509: certificate signed by unknown authority",
		"tls: failed to verify certificate",
		"certificate is valid for example.com",
	}
	for _, msg := range cases {
		err := FriendlyError(errors.New(msg))
		if err == nil || !strings.Contains(err.Error(), "secure connection failed") {
			t.Errorf("msg=%q: got %q", msg, friendlyMsg(err))
		}
	}
}

func TestFriendlyError_ConnectionLost(t *testing.T) {
	cases := []string{"connection reset by peer", "EOF"}
	for _, msg := range cases {
		err := FriendlyError(errors.New(msg))
		if err == nil || !strings.Contains(err.Error(), "connection lost") {
			t.Errorf("msg=%q: got %q", msg, friendlyMsg(err))
		}
	}
}

func TestFriendlyError_Passthrough(t *testing.T) {
	cases := []string{
		"URL and API key required",
		"no station profiles found",
	}
	for _, msg := range cases {
		err := FriendlyError(errors.New(msg))
		if err == nil || !strings.Contains(err.Error(), msg) {
			t.Errorf("msg=%q: got %q", msg, friendlyMsg(err))
		}
	}
}

func TestFriendlyError_Unknown(t *testing.T) {
	err := FriendlyError(errors.New("some random error"))
	if err == nil || err.Error() != "some random error" {
		t.Errorf("got %q", friendlyMsg(err))
	}
}

// =============================================================================
// APIError / FriendlyError v2 tests
// =============================================================================

func TestFriendlyError_APIError(t *testing.T) {
	cases := []struct {
		apiErr *APIError
		want   string
	}{
		{&APIError{StatusCode: 401, Code: "invalid_token"}, "wl2_"},
		{&APIError{StatusCode: 401, Code: "unauthorized"}, "wl2_"},
		{&APIError{StatusCode: 401, Code: "token_expired"}, "expired"},
		{&APIError{StatusCode: 403, Code: "insufficient_scope", Details: map[string]any{"required_scope": "qso:write"}}, "qso:write"},
		{&APIError{StatusCode: 429, Code: "rate_limited", Details: map[string]any{"retry_after": float64(7)}}, "7s"},
		{&APIError{StatusCode: 404, Code: "not_found"}, "not found"},
		{&APIError{StatusCode: 409, Code: "conflict"}, "duplicate"},
	}
	for _, c := range cases {
		err := FriendlyError(c.apiErr)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("code=%q: got %q, want substring %q", c.apiErr.Code, friendlyMsg(err), c.want)
		}
	}
}

func TestAPIError_RetryAfter(t *testing.T) {
	e := &APIError{Details: map[string]any{"retry_after": "12"}}
	if got := e.RetryAfterSeconds(); got != 12 {
		t.Errorf("got %d, want 12", got)
	}
}

// =============================================================================
// PrivateLookupResult tests
// =============================================================================

func TestPrivateLookupResult_Accessors(t *testing.T) {
	r := &PrivateLookupResult{
		callsign:          "SP9ABC",
		dxcc:              "POLAND",
		dxccID:            "269",
		dxccCQZ:           "15",
		dxccITUZ:          "28",
		name:              "Jan",
		gridsquare:        "KO00ca",
		location:          "Niepolomice",
		state:             "M",
		worked:            true,
		workedBand:        true,
		workedBandMode:    false,
		lotw:              true,
		dxccConfirmed:     true,
		confirmedBand:     false,
		confirmedBandMode: false,
	}
	if r.Callsign() != "SP9ABC" {
		t.Errorf("Callsign = %q", r.Callsign())
	}
	if r.Name() != "Jan" {
		t.Errorf("Name = %q", r.Name())
	}
	if !r.Worked() || !r.WorkedBand() || r.WorkedBandMode() {
		t.Error("worked flags wrong")
	}
	if !r.LoTW() {
		t.Error("expected LoTW=true")
	}
	if !r.DXCCConfirmed() || r.ConfirmedBand() || r.ConfirmedBandMode() {
		t.Error("confirmation flags wrong")
	}
	if r.Grid() != "KO00ca" {
		t.Errorf("Grid = %q", r.Grid())
	}
	if r.DXCCID() != "269" || r.DXCCName() != "POLAND" || r.Country() != "POLAND" {
		t.Error("dxcc fields wrong")
	}
	if r.QTH() != "Niepolomice" || r.State() != "M" {
		t.Error("qth/state wrong")
	}
	if r.CQZone() != "15" || r.ITUZone() != "28" {
		t.Error("zone fields wrong")
	}
}

func TestPrivateLookupResult_Empty(t *testing.T) {
	r := &PrivateLookupResult{}
	if r.Callsign() != "" || r.Name() != "" || r.Worked() || r.LoTW() {
		t.Error("expected empty result to report no data")
	}
}

func TestNewLookupResult(t *testing.T) {
	d := &lookupData{
		Callsign:              "DL1ABC",
		DXCC:                  "GERMANY",
		DXCCID:                "230",
		DXCCCQZ:               "14",
		DXCCITUZ:              28,
		Name:                  "Marty",
		Gridsquare:            "JO31",
		CallWorked:            true,
		CallWorkedBand:        false,
		CallWorkedBandMode:    false,
		LotwMember:            "14",
		DXCCConfirmed:         true,
		CallConfirmedBand:     false,
		CallConfirmedBandMode: false,
	}
	r := newLookupResult(d)
	if r.Worked() != true || r.WorkedBand() != false {
		t.Error("worked flags wrong")
	}
	if !r.LoTW() {
		t.Error("expected LoTW=true for member number")
	}
	if r.ITUZone() != "28" {
		t.Errorf("ITUZone = %q, want 28", r.ITUZone())
	}
	if r.Grid() != "JO31" || r.Name() != "Marty" {
		t.Error("grid/name wrong")
	}
}

func TestTruthy(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"12", true},
		{"yes", true},
		{"", false},
		{"0", false},
		{"false", false},
		{"no", false},
	}
	for _, c := range cases {
		if got := truthy(c.val); got != c.want {
			t.Errorf("truthy(%q) = %v, want %v", c.val, got, c.want)
		}
	}
}

func TestStripADIFHeader(t *testing.T) {
	in := "Wavelog ADIF export\n<ADIF_VER:5>3.1.7\n<EOH>\n<CALL:5>F5MXH<EOR>"
	got := stripADIFHeader(in)
	if strings.Contains(got, "<EOH>") {
		t.Errorf("header not stripped: %q", got)
	}
	if !strings.Contains(got, "<CALL:5>F5MXH") {
		t.Errorf("records lost: %q", got)
	}
	if stripADIFHeader("no header here") != "no header here" {
		t.Error("expected passthrough without <EOH>")
	}
}

// =============================================================================
// Helpers
// =============================================================================

func friendlyMsg(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

// =============================================================================
// v2 token enforcement
// =============================================================================

func TestFetchStations_RejectsV1Key(t *testing.T) {
	// A legacy v1 key must fail fast with the migration message,
	// without any network call.
	_, err := FetchStations("http://127.0.0.1:1", "wl123_not_v2")
	if err == nil {
		t.Fatal("FetchStations with v1 key should fail")
	}
	if !strings.Contains(err.Error(), V1KeyRequiredMsg) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), V1KeyRequiredMsg)
	}
}

func TestTestConnection_RejectsV1Key(t *testing.T) {
	err := TestConnection("http://127.0.0.1:1", "wl123_not_v2")
	if err == nil {
		t.Fatal("TestConnection with v1 key should fail")
	}
	if !strings.Contains(err.Error(), V1KeyRequiredMsg) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), V1KeyRequiredMsg)
	}
}
