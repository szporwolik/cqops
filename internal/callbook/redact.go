package callbook

import (
	"errors"
	"net/url"
	"regexp"
)

// secretQueryRe matches query parameters whose values are credentials or
// session tokens. Providers pass these in the URL, and a failed request
// returns a *url.Error carrying the whole URL, so the value would otherwise
// reach the log file. Both "&" and ";" separators are matched because
// QRZ.com uses ";".
var secretQueryRe = regexp.MustCompile(`(?i)(^|[?&;])(password|passwd|pwd|pass|p|api_key|apikey|key|token|s|id)=[^&;#]*`)

// RedactURL replaces credential and session values in a URL query string
// with "****". Non-secret parameters (user, callsign, agent) are preserved so
// logs stay useful for diagnosis.
func RedactURL(raw string) string {
	return secretQueryRe.ReplaceAllString(raw, "${1}${2}=****")
}

// RedactURLError rewrites the URL inside a *url.Error so callbook passwords
// and session ids never reach logs. Errors of any other type are returned
// unchanged. Call this on the error from an HTTP request before wrapping it.
func RedactURLError(err error) error {
	if err == nil {
		return nil
	}
	var ue *url.Error
	if !errors.As(err, &ue) {
		return err
	}
	redacted := *ue
	redacted.URL = RedactURL(ue.URL)
	return &redacted
}
