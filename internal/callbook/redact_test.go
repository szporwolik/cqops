package callbook

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			// QRZ.com separates parameters with ";", which url.ParseQuery rejects.
			name: "qrzcom semicolon separated password",
			raw:  "https://xmldata.qrz.com/xml/current/?username=N0CALL;password=hunter2;agent=CQOps",
			want: "https://xmldata.qrz.com/xml/current/?username=N0CALL;password=****;agent=CQOps",
		},
		{
			name: "qrzru short password param",
			raw:  "https://api.qrz.ru/login?u=N0CALL&p=hunter2&agent=CQOps",
			want: "https://api.qrz.ru/login?u=N0CALL&p=****&agent=CQOps",
		},
		{
			name: "hamqth password param",
			raw:  "https://www.hamqth.com/xml.php?u=N0CALL&p=hunter2",
			want: "https://www.hamqth.com/xml.php?u=N0CALL&p=****",
		},
		{
			name: "qrzcom session key",
			raw:  "https://xmldata.qrz.com/xml/current/?s=SESSIONKEY;callsign=SP9MOA",
			want: "https://xmldata.qrz.com/xml/current/?s=****;callsign=SP9MOA",
		},
		{
			name: "session id",
			raw:  "https://api.qrz.ru/callsign?id=SESSIONID&callsign=SP9MOA",
			want: "https://api.qrz.ru/callsign?id=****&callsign=SP9MOA",
		},
		{
			name: "no secrets is unchanged",
			raw:  "https://callook.info/SP9MOA/json",
			want: "https://callook.info/SP9MOA/json",
		},
		{
			// "callsign" and "grid" must not be mistaken for "s" and "id".
			name: "similar param names preserved",
			raw:  "https://example.com/x?callsign=SP9MOA&grid=JO90",
			want: "https://example.com/x?callsign=SP9MOA&grid=JO90",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactURL(tt.raw); got != tt.want {
				t.Errorf("RedactURL()\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestRedactURLErrorHidesPassword(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "https://www.hamqth.com/xml.php?u=N0CALL&p=hunter2",
		Err: errors.New("context deadline exceeded"),
	}

	got := RedactURLError(err).Error()
	if strings.Contains(got, "hunter2") {
		t.Errorf("redacted error still contains the password: %s", got)
	}
	if !strings.Contains(got, "context deadline exceeded") {
		t.Errorf("redacted error lost the underlying cause: %s", got)
	}
}

func TestRedactURLErrorPassesThroughOtherErrors(t *testing.T) {
	plain := errors.New("connection refused")
	if got := RedactURLError(plain); got != plain {
		t.Errorf("non-URL error should be returned unchanged, got %v", got)
	}
	if RedactURLError(nil) != nil {
		t.Error("nil error should stay nil")
	}
}

// http.NewRequest surfaces url.Parse failures, and those also carry the raw
// URL — so the request-construction path needs redacting too, not just the
// transport path.
func TestRedactURLErrorCoversParseFailures(t *testing.T) {
	_, err := http.NewRequest(http.MethodGet, "https://example.com/x?u=N0CALL&p=hunter2\x7f", nil)
	if err == nil {
		t.Skip("URL unexpectedly parsed; nothing to redact")
	}

	if !strings.Contains(err.Error(), "hunter2") {
		t.Skip("parse error does not embed the URL on this Go version")
	}
	if got := RedactURLError(err).Error(); strings.Contains(got, "hunter2") {
		t.Errorf("redacted parse error still contains the password: %s", got)
	}
}
