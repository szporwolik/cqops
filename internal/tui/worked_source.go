package tui

import (
	"net/url"
	"strings"
)

// CompactSourceName returns a compact display name for a log source.
// If displayName is non-empty (explicitly configured), it is used verbatim.
// Otherwise the URL is parsed and only the hostname (without default port,
// scheme, credentials, path, query, or fragment) is returned.
//
// Non-default ports (anything other than 80/443) are preserved.
// If the URL is malformed, the raw URL string is returned unmodified.
func CompactSourceName(displayName, rawURL string) string {
	if displayName != "" {
		return displayName
	}
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		// Unparseable — return raw, but strip credentials if present.
		return stripCredentials(rawURL)
	}
	return hostname(u)
}

// hostname extracts the host portion from a parsed URL, stripping default
// ports (80 for http, 443 for https) and preserving non-default ports.
func hostname(u *url.URL) string {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		return host
	}
	// Strip default ports.
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		return host
	}
	// Non-default port — preserve it.
	return host + ":" + port
}

// stripCredentials removes userinfo from a URL string when normal parsing
// fails or returns empty host. Best-effort; does not handle every edge case.
func stripCredentials(raw string) string {
	// Remove scheme://user:pass@ → scheme://
	if idx := strings.Index(raw, "://"); idx != -1 {
		rest := raw[idx+3:]
		if atIdx := strings.Index(rest, "@"); atIdx != -1 {
			raw = raw[:idx+3] + rest[atIdx+1:]
		}
	}
	// Also handle bare user:pass@host
	if idx := strings.Index(raw, "@"); idx != -1 && !strings.Contains(raw[:idx], "://") {
		raw = raw[idx+1:]
	}
	// Remove path/query/fragment after the host.
	if idx := strings.Index(raw, "/"); idx != -1 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, "?"); idx != -1 {
		if idx < len(raw) && (strings.Index(raw, "/") == -1 || strings.Index(raw, "/") > idx) {
			raw = raw[:idx]
		}
	}
	return raw
}
