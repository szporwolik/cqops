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
// PrivateLookupResult tests
// =============================================================================

func TestPrivateLookupResult_Callsign(t *testing.T) {
	r := &PrivateLookupResult{raw: map[string]interface{}{"callsign": "SP9ABC"}}
	if r.Callsign() != "SP9ABC" {
		t.Errorf("got %q", r.Callsign())
	}
}

func TestPrivateLookupResult_Name(t *testing.T) {
	r := &PrivateLookupResult{raw: map[string]interface{}{"name": "Jan"}}
	if r.Name() != "Jan" {
		t.Errorf("got %q", r.Name())
	}
}

func TestPrivateLookupResult_Str_Missing(t *testing.T) {
	r := &PrivateLookupResult{raw: map[string]interface{}{}}
	if r.Callsign() != "" {
		t.Errorf("expected empty, got %q", r.Callsign())
	}
}

func TestPrivateLookupResult_Str_Nil(t *testing.T) {
	r := &PrivateLookupResult{raw: map[string]interface{}{"name": nil}}
	if r.Name() != "" {
		t.Errorf("expected empty for nil value, got %q", r.Name())
	}
}

func TestPrivateLookupResult_IsTrue(t *testing.T) {
	cases := []struct {
		val  interface{}
		want bool
	}{
		{true, true},
		{false, false},
		{"true", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"Y", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"N", false},
		{"", false},
		{float64(1), true},
		{float64(0), false},
		{"something", true}, // unknown non-empty → truthy
	}
	for _, c := range cases {
		r := &PrivateLookupResult{raw: map[string]interface{}{"x": c.val}}
		got := r.IsTrue("x")
		if got != c.want {
			t.Errorf("IsTrue(%v) = %v, want %v", c.val, got, c.want)
		}
	}
}

func TestPrivateLookupResult_IsTrue_Missing(t *testing.T) {
	r := &PrivateLookupResult{raw: map[string]interface{}{}}
	if r.IsTrue("nonexistent") {
		t.Error("expected false for missing key")
	}
}

func TestPrivateLookupResult_Worked(t *testing.T) {
	r := &PrivateLookupResult{raw: map[string]interface{}{"call_worked": true}}
	if !r.Worked() {
		t.Error("expected worked=true")
	}
	r2 := &PrivateLookupResult{raw: map[string]interface{}{"call_worked": "1"}}
	if !r2.Worked() {
		t.Error("expected worked=true for string 1")
	}
}

func TestPrivateLookupResult_LoTW(t *testing.T) {
	r := &PrivateLookupResult{raw: map[string]interface{}{"lotw_member": "12"}}
	if !r.LoTW() {
		t.Error("expected LoTW=true for non-empty string")
	}
	r2 := &PrivateLookupResult{raw: map[string]interface{}{"lotw_member": false}}
	if r2.LoTW() {
		t.Error("expected LoTW=false")
	}
}

// =============================================================================
// extractAPIReason and stripHTML
// =============================================================================

func TestExtractAPIReason(t *testing.T) {
	if r := extractAPIReason(`{"reason":"Invalid key"}`); r != "Invalid key" {
		t.Errorf("got %q", r)
	}
	if r := extractAPIReason("plain text"); r != "" {
		t.Errorf("expected empty, got %q", r)
	}
	if r := extractAPIReason(`{}`); r != "" {
		t.Errorf("expected empty, got %q", r)
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<b>bold</b>", "bold"},
		{"no html", "no html"},
		{"<a href='x'>link</a>", "link"},
		{"<br>", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := stripHTML(c.in); got != c.want {
			t.Errorf("stripHTML(%q) = %q, want %q", c.in, got, c.want)
		}
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
