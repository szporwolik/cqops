package tui

import (
	"testing"
)

func TestCompactSourceName_ExplicitDisplayName(t *testing.T) {
	got := CompactSourceName("MyWavelog", "https://qso.cqops.com/api/")
	if got != "MyWavelog" {
		t.Errorf("expected explicit display name, got %q", got)
	}
}

func TestCompactSourceName_HTTPS(t *testing.T) {
	got := CompactSourceName("", "https://qso.cqops.com/")
	if got != "qso.cqops.com" {
		t.Errorf("expected qso.cqops.com, got %q", got)
	}
}

func TestCompactSourceName_HTTP(t *testing.T) {
	got := CompactSourceName("", "http://qso.cqops.com/")
	if got != "qso.cqops.com" {
		t.Errorf("expected qso.cqops.com, got %q", got)
	}
}

func TestCompactSourceName_StripPath(t *testing.T) {
	got := CompactSourceName("", "https://qso.cqops.com/api/")
	if got != "qso.cqops.com" {
		t.Errorf("expected qso.cqops.com, got %q", got)
	}
}

func TestCompactSourceName_StripCredentials(t *testing.T) {
	got := CompactSourceName("", "https://user:password@qso.cqops.com/")
	if got != "qso.cqops.com" {
		t.Errorf("expected qso.cqops.com, got %q", got)
	}
}

func TestCompactSourceName_NonDefaultPort(t *testing.T) {
	got := CompactSourceName("", "https://log.example.net:8443/api/")
	if got != "log.example.net:8443" {
		t.Errorf("expected log.example.net:8443, got %q", got)
	}
}

func TestCompactSourceName_DefaultPort80(t *testing.T) {
	got := CompactSourceName("", "http://example.com:80/path")
	if got != "example.com" {
		t.Errorf("expected example.com, got %q", got)
	}
}

func TestCompactSourceName_DefaultPort443(t *testing.T) {
	got := CompactSourceName("", "https://example.com:443/path")
	if got != "example.com" {
		t.Errorf("expected example.com, got %q", got)
	}
}

func TestCompactSourceName_IPv4(t *testing.T) {
	got := CompactSourceName("", "http://192.168.1.100:8080/api")
	if got != "192.168.1.100:8080" {
		t.Errorf("expected 192.168.1.100:8080, got %q", got)
	}
}

func TestCompactSourceName_IPv6(t *testing.T) {
	got := CompactSourceName("", "https://[::1]:8443/path")
	if got != "::1:8443" {
		t.Errorf("expected ::1:8443, got %q", got)
	}
}

func TestCompactSourceName_MalformedURL(t *testing.T) {
	got := CompactSourceName("", "not-a-url")
	if got != "not-a-url" {
		t.Errorf("expected raw string for malformed URL, got %q", got)
	}
}

func TestCompactSourceName_EmptyURL(t *testing.T) {
	got := CompactSourceName("", "")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestCompactSourceName_DisplayNameAndURL(t *testing.T) {
	got := CompactSourceName("ClubLog", "https://clublog.org/")
	if got != "ClubLog" {
		t.Errorf("expected ClubLog, got %q", got)
	}
}

func TestCompactSourceName_NoScheme(t *testing.T) {
	// net/url.Parse treats "host:port" as opaque (scheme-less) URIs,
	// returning no Host. This is fine — we return it as-is.
	got := CompactSourceName("", "log.example.net")
	if got != "log.example.net" {
		t.Errorf("expected log.example.net, got %q", got)
	}
}

func TestCompactSourceName_CredentialsOnly(t *testing.T) {
	got := CompactSourceName("", "https://admin@example.com/")
	if got != "example.com" {
		t.Errorf("expected example.com, got %q", got)
	}
}

func TestCompactSourceName_QueryAndFragment(t *testing.T) {
	got := CompactSourceName("", "https://example.com/path?key=val#section")
	if got != "example.com" {
		t.Errorf("expected example.com, got %q", got)
	}
}

func TestCompactSourceName_EmptyDisplayName_EmptyURL(t *testing.T) {
	got := CompactSourceName("", "")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
