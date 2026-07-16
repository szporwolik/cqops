// Package hamqth implements the callbook.Provider interface for the
// HamQTH free callsign database (https://www.hamqth.com).
//
// HamQTH provides a simple XML API:
//   - Authenticate once to obtain a session_id (valid ~1 hour).
//   - Perform callsign lookups using the session_id.
//
// There are no rate limits beyond the natural HTTP latency.
package hamqth

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/callbook"
)

// --- XML types ---

// hamqthRoot is the root element returned by every HamQTH API call.
// All responses (auth, search, error) are wrapped in <HamQTH>.
type hamqthRoot struct {
	XMLName xml.Name       `xml:"HamQTH"`
	Session *hamqthSession `xml:"session"`
	Search  *hamqthSearch  `xml:"search"`
}

type hamqthSession struct {
	SessionID string `xml:"session_id"`
	Error     string `xml:"error"`
}

type hamqthSearch struct {
	Callsign  string `xml:"callsign"`
	Nick      string `xml:"nick"`
	QTH       string `xml:"qth"`
	Grid      string `xml:"grid"`
	Country   string `xml:"country"`
	AdrName   string `xml:"adr_name"`
	AdrStreet string `xml:"adr_street1"`
	AdrCity   string `xml:"adr_city"`
	AdrZip    string `xml:"adr_zip"`
	ITUZone   string `xml:"itu"`
	CQZone    string `xml:"cq"`
	Lat       string `xml:"latitude"`
	Lon       string `xml:"longitude"`
	Adif      string `xml:"adif"`    // DXCC ADIF ID (e.g. 503 for Czech Republic)
	Picture   string `xml:"picture"` // link to user's picture
	Lotw      string `xml:"lotw"`
	EQSL      string `xml:"eqsl"`
	QSL       string `xml:"qsl"`
}

// --- Client ---

// Client implements callbook.Provider for HamQTH.
// It is safe for concurrent use.
type Client struct {
	user     string
	pass     string
	priority int
	httpFn   func(string) ([]byte, error) // test seam

	mu    sync.Mutex
	sID   string // cached session ID
	sUser string // user the cached session belongs to
	sPass string // pass the cached session belongs to
	sAt   time.Time
}

// NewClient creates a HamQTH callbook client with default priority (45).
func NewClient(user, pass string) *Client {
	return &Client{user: user, pass: pass, priority: 45, httpFn: httpGetFn}
}

// NewClientWithPriority creates a HamQTH client with explicit priority.
func NewClientWithPriority(user, pass string, priority int) *Client {
	return &Client{user: user, pass: pass, priority: priority, httpFn: httpGetFn}
}

// Lookup is the package-level convenience function for testing and
// one-off lookups. New code should prefer constructing a Client.
func Lookup(user, pass, callsign string) (*SearchData, error) {
	return NewClient(user, pass).lookup(callsign)
}

// Name returns the provider identifier.
func (c *Client) Name() string { return "HamQTH" }

// Priority returns the lookup priority (default 45).
func (c *Client) Priority() int { return c.priority }

// Lookup queries HamQTH for a callsign. Returns nil, nil when the
// callsign is not found (normal condition, not an error).
func (c *Client) Lookup(callsign string) (*callbook.Result, error) {
	if callsign == "" || c.user == "" {
		return nil, nil
	}
	d, err := c.lookup(callsign)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}
	// HamQTH returns relative image paths (e.g. "user_img/SP9SPM/img.jpg")
	// or full URLs. Normalize to absolute URLs for the photo viewer.
	imgURL := normalizeImageURL(d.Picture)

	// HamQTH returns default placeholder images (e.g. paddle_and_notebook.jpg)
	// when the operator has no photo.  Treat those as "no image" so lower-
	// priority callbooks (QRZ, Callook) can supply a real photo instead.
	if isHamQTHDefaultImage(imgURL) {
		imgURL = ""
	}

	return &callbook.Result{
		Callsign: d.Callsign, Name: d.Name, Grid: d.Grid,
		Country: d.Country, QTH: d.QTH,
		Lat: d.Lat, Lon: d.Lon,
		DXCC: d.DXCC, CQZone: d.CQZone, ITUZone: d.ITUZone,
		ImageURL: imgURL, Provider: "hamqth",
	}, nil
}

// normalizeImageURL converts a HamQTH image path to an absolute URL.
// HamQTH may return relative paths like "user_img/OK2CQR/OK2CQR.jpg"
// or full URLs. Empty/blank input returns empty string.
func normalizeImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	// Relative path — prepend HamQTH base URL.
	return "https://www.hamqth.com/" + strings.TrimLeft(raw, "/")
}

// isHamQTHDefaultImage returns true when url is a HamQTH default placeholder
// image (e.g. paddle_and_notebook.jpg).  Detected by the /images/default/
// path prefix — host-agnostic to handle mirrors and protocol variants.
func isHamQTHDefaultImage(url string) bool {
	return strings.Contains(url, "/images/default/")
}

// TestConnection validates HamQTH credentials and obtains a session.
func (c *Client) TestConnection() error {
	return c.testConnection()
}

// lookup returns the internal SearchData.
func (c *Client) lookup(callsign string) (*SearchData, error) {
	if callsign == "" || c.user == "" {
		return nil, nil
	}

	// Reuse cached session if still valid (< 50 minutes).
	c.mu.Lock()
	sID, sUser, sPass, sAt := c.sID, c.sUser, c.sPass, c.sAt
	c.mu.Unlock()

	if sID != "" && sUser == c.user && sPass == c.pass && time.Since(sAt) < 50*time.Minute {
		data, err := c.search(sID, callsign)
		if err == nil {
			return data, nil
		}
		applog.Debug("HamQTH: cached session failed, re-authenticating")
		c.mu.Lock()
		c.sID = ""
		c.mu.Unlock()
	}
	return c.loginAndSearch(callsign)
}

// testConnection obtains a session to verify credentials.
func (c *Client) testConnection() error {
	applog.Debug("HamQTH: testing connection")
	if c.user == "" || c.pass == "" {
		return fmt.Errorf("HamQTH username and password required")
	}
	u := "https://www.hamqth.com/xml.php?u=" + url.QueryEscape(c.user) + "&p=" + url.QueryEscape(c.pass)
	data, err := c.httpFn(u)
	if err != nil {
		return fmt.Errorf("HamQTH connection failed: %w", err)
	}
	var root hamqthRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("HamQTH invalid response: %w", err)
	}
	if root.Session == nil {
		return fmt.Errorf("HamQTH: unexpected response (no session element)")
	}
	if root.Session.Error != "" {
		return fmt.Errorf("HamQTH: %s", root.Session.Error)
	}
	if root.Session.SessionID == "" {
		return fmt.Errorf("HamQTH: empty session ID")
	}
	c.mu.Lock()
	c.sID = root.Session.SessionID
	c.sUser = c.user
	c.sPass = c.pass
	c.sAt = time.Now()
	c.mu.Unlock()
	applog.InfoDetail("HamQTH: connected", fmt.Sprintf("user=%s", c.user))
	return nil
}

// search performs a callsign lookup with a valid session.
func (c *Client) search(sessionID, callsign string) (*SearchData, error) {
	u := "https://www.hamqth.com/xml.php?id=" + url.QueryEscape(sessionID) +
		"&callsign=" + url.QueryEscape(callsign) +
		"&prg=CQOps"
	data, err := c.httpFn(u)
	if err != nil {
		return nil, fmt.Errorf("HamQTH lookup: %w", err)
	}

	var root hamqthRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("HamQTH xml: %w", err)
	}

	// Check for session-level errors first (wrong session, expired, not found).
	if root.Session != nil && root.Session.Error != "" {
		e := root.Session.Error
		if strings.Contains(e, "not found") || strings.Contains(e, "Not found") {
			applog.Info("HamQTH: not found", "callsign", callsign)
			return nil, nil
		}
		if strings.Contains(e, "Session") || strings.Contains(e, "session") {
			return nil, fmt.Errorf("HamQTH session expired: %s", e)
		}
		applog.Warn("HamQTH: error", "msg", e)
		return nil, fmt.Errorf("HamQTH: %s", e)
	}

	// Check for search results.
	if root.Search != nil && root.Search.Callsign != "" {
		return toSearchData(root.Search), nil
	}

	// Empty result — callsign not in database.
	applog.Debug("HamQTH: no data", "callsign", callsign)
	return nil, nil
}

// loginAndSearch authenticates and then performs the lookup in one go.
func (c *Client) loginAndSearch(callsign string) (*SearchData, error) {
	applog.Debug("HamQTH: authenticating", "callsign", callsign)
	authURL := "https://www.hamqth.com/xml.php?u=" + url.QueryEscape(c.user) + "&p=" + url.QueryEscape(c.pass)
	data, err := c.httpFn(authURL)
	if err != nil {
		return nil, fmt.Errorf("HamQTH auth: %w", err)
	}
	var root hamqthRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("HamQTH auth xml: %w", err)
	}
	if root.Session == nil {
		return nil, fmt.Errorf("HamQTH: unexpected auth response (no session element)")
	}
	if root.Session.Error != "" {
		return nil, fmt.Errorf("HamQTH: %s", root.Session.Error)
	}
	if root.Session.SessionID == "" {
		return nil, fmt.Errorf("HamQTH: empty session ID")
	}

	c.mu.Lock()
	c.sID = root.Session.SessionID
	c.sUser = c.user
	c.sPass = c.pass
	c.sAt = time.Now()
	c.mu.Unlock()

	return c.search(root.Session.SessionID, callsign)
}

// --- HTTP transport and test seam ---

// httpGetFn is the package-level HTTP transport. Tests replace it via the
// test seam to avoid real network calls.
var httpGetFn = defaultHTTPGet

func defaultHTTPGet(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 256*1024))
}

// SetHTTPFn sets the HTTP transport function (test seam).
// Callers should restore the original via defer.
func SetHTTPFn(fn func(string) ([]byte, error)) func() {
	prev := httpGetFn
	httpGetFn = fn
	return func() { httpGetFn = prev }
}

// --- Result type ---

// SearchData holds the raw HamQTH search result fields.
// Kept package-exported so tests can inspect it.
type SearchData struct {
	Callsign string
	Name     string
	Grid     string
	Country  string
	QTH      string
	Lat      string
	Lon      string
	DXCC     string
	CQZone   string
	ITUZone  string
	Picture  string // full URL from <picture> element
}

func toSearchData(s *hamqthSearch) *SearchData {
	// HamQTH's "nick" field is the operator name, and "qth" is the city.
	name := strings.TrimSpace(s.Nick)
	if name == "" {
		name = strings.TrimSpace(s.AdrName)
	}
	qth := strings.TrimSpace(s.QTH)
	if qth == "" {
		qth = strings.TrimSpace(s.AdrCity)
	}

	sd := &SearchData{
		Callsign: strings.TrimSpace(s.Callsign),
		Name:     name,
		Grid:     strings.TrimSpace(s.Grid),
		Country:  strings.TrimSpace(s.Country),
		QTH:      qth,
		Lat:      normalizeCoord(s.Lat),
		Lon:      normalizeCoord(s.Lon),
		DXCC:     strings.TrimSpace(s.Adif),
		CQZone:   strings.TrimSpace(s.CQZone),
		ITUZone:  strings.TrimSpace(s.ITUZone),
		Picture:  strings.TrimSpace(s.Picture),
	}
	applog.InfoDetail("HamQTH: lookup ok", fmt.Sprintf("%s — %s", sd.Callsign, sd.Name))
	return sd
}

// normalizeCoord trims and rounds a coordinate string to 5 decimal places,
// removing floating-point noise from the HamQTH API (e.g. 50.25344610000001 → 50.25345).
func normalizeCoord(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw // unparseable, return as-is
	}
	// Round to 5 decimal places (~1 m precision, plenty for ham radio).
	return strconv.FormatFloat(f, 'f', 5, 64)
}
