package qrzcom

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/callbook"
)

// Client implements callbook.Provider for QRZ.com XML API.
// It is safe for concurrent use.
type Client struct {
	user     string
	pass     string
	priority int
	httpFn   func(string) ([]byte, error) // test seam

	mu    sync.Mutex
	key   string // cached session key
	kUser string // user the cached key belongs to
	kPass string // pass the cached key belongs to
}

// NewClient creates a QRZ.com callbook client.
// It uses the package-level httpGetFn (the test seam) as its HTTP transport.
func NewClient(user, pass string) *Client {
	return &Client{user: user, pass: pass, priority: 50, httpFn: httpGetFn}
}

// NewClientWithPriority creates a QRZ.com client with explicit priority.
func NewClientWithPriority(user, pass string, priority int) *Client {
	return &Client{user: user, pass: pass, priority: priority, httpFn: httpGetFn}
}

// Name returns the provider identifier.
func (c *Client) Name() string { return "QRZ.com" }

// Priority returns the lookup priority (default 50).
func (c *Client) Priority() int { return c.priority }

// Lookup queries QRZ.com for a callsign. It returns nil, nil when the
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
	return &callbook.Result{
		Callsign: d.Callsign, Name: d.Name, Grid: d.Grid, Country: d.Country,
		QTH: d.QTH, State: d.State, Zip: d.Zip, County: d.County,
		Class: d.Class, Email: d.Email, URL: d.URL,
		Lat: d.Lat, Lon: d.Lon, DXCC: d.DXCC, CQZone: d.CQZone, ITUZone: d.ITUZone,
		ImageURL: d.ImageURL, LoTW: d.LoTW, EQSL: d.EQSL, QSLManager: d.QSLManager,
		Provider: "qrz",
	}, nil
}

// TestConnection validates QRZ credentials.
func (c *Client) TestConnection() error {
	return c.testConnection()
}

// lookup is the internal method returning the legacy CallData (shared with
// the deprecated package-level Lookup function during migration).
func (c *Client) lookup(callsign string) (*CallData, error) {
	if callsign == "" || c.user == "" {
		return nil, nil
	}
	// Reuse cached session key if credentials match.
	c.mu.Lock()
	key := c.key
	kUser := c.kUser
	kPass := c.kPass
	c.mu.Unlock()

	if key != "" && kUser == c.user && kPass == c.pass {
		data, err := c.qrzLookup(key, callsign)
		if err == nil {
			return data, nil
		}
		applog.Debug("QRZ cached session failed, re-authenticating")
		c.mu.Lock()
		c.key = ""
		c.mu.Unlock()
	}
	return c.qrzLoginLookup(callsign)
}

func (c *Client) testConnection() error {
	applog.Debug("QRZ: testing connection")
	if c.user == "" || c.pass == "" {
		return fmt.Errorf("QRZ username and password required")
	}
	u := "https://xmldata.qrz.com/xml/current/?username=" + url.QueryEscape(c.user) + ";password=" + url.QueryEscape(c.pass) + ";agent=CQOps"
	data, err := c.httpFn(u)
	if err != nil {
		applog.Error("QRZ: connection failed", "error", sanitizeQRError(err))
		return fmt.Errorf("connection failed: %s", sanitizeQRError(err))
	}
	var authDB qrzDatabase
	if err := xml.Unmarshal(data, &authDB); err != nil {
		applog.Error("QRZ: invalid xml response", "error", err)
		return fmt.Errorf("invalid response: %w", err)
	}
	if authDB.Session.Error != "" {
		applog.Error("QRZ: auth error", "msg", authDB.Session.Error)
		return fmt.Errorf("QRZ: %s", authDB.Session.Error)
	}
	if authDB.Session.Key == "" {
		return fmt.Errorf("QRZ: no session key")
	}
	c.mu.Lock()
	c.key = authDB.Session.Key
	c.kUser = c.user
	c.kPass = c.pass
	c.mu.Unlock()
	applog.InfoDetail("QRZ: connected", fmt.Sprintf("user=%s url=xmldata.qrz.com", c.user))
	return nil
}

func (c *Client) qrzLookup(sessionKey, callsign string) (*CallData, error) {
	u := "https://xmldata.qrz.com/xml/current/?s=" + url.QueryEscape(sessionKey) + ";callsign=" + url.QueryEscape(callsign)
	data, err := c.httpFn(u)
	if err != nil {
		applog.Error("QRZ lookup failed", "error", err)
		return nil, err
	}
	var db qrzDatabase
	if err := xml.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("qrz xml: %w", err)
	}
	if db.Session.Error != "" {
		if strings.Contains(db.Session.Error, "Not found") {
			applog.Info("QRZ: not found", "callsign", callsign)
			return nil, nil
		}
		applog.Error("QRZ lookup error", "msg", db.Session.Error)
		return nil, fmt.Errorf("QRZ: %s", db.Session.Error)
	}
	call := db.Callsign
	if strings.TrimSpace(call.Call) == "" {
		applog.Debug("QRZ no data", "callsign", callsign)
		return nil, nil
	}
	applog.InfoDetail("QRZ lookup ok", fmt.Sprintf("%s — %s", call.Call, coalesce(call.Fname, call.Name)))
	return &CallData{
		Callsign:   strings.TrimSpace(call.Call),
		Name:       strings.TrimSpace(coalesce(call.Fname, call.Name)),
		Grid:       strings.TrimSpace(call.Grid),
		Country:    strings.TrimSpace(call.Country),
		State:      strings.TrimSpace(call.State),
		QTH:        strings.TrimSpace(call.Addr2),
		Zip:        strings.TrimSpace(call.Zip),
		County:     strings.TrimSpace(call.County),
		Class:      strings.TrimSpace(call.Class),
		Email:      strings.TrimSpace(call.Email),
		URL:        strings.TrimSpace(call.URL),
		Lat:        strings.TrimSpace(call.Lat),
		Lon:        strings.TrimSpace(call.Lon),
		DXCC:       strings.TrimSpace(call.DXCC),
		CQZone:     strings.TrimSpace(call.CQZone),
		ITUZone:    strings.TrimSpace(call.ITUZone),
		ImageURL:   strings.TrimSpace(call.Image),
		LoTW:       strings.TrimSpace(call.LoTW) == "1",
		EQSL:       strings.TrimSpace(call.EQSL) == "1",
		QSLManager: strings.TrimSpace(call.QSLMgr),
	}, nil
}

func (c *Client) qrzLoginLookup(callsign string) (*CallData, error) {
	applog.Debug("QRZ lookup", "callsign", callsign)
	u := "https://xmldata.qrz.com/xml/current/?username=" + url.QueryEscape(c.user) + ";password=" + url.QueryEscape(c.pass) + ";agent=CQOps"
	data, err := c.httpFn(u)
	if err != nil {
		applog.Error("QRZ auth failed", "error", sanitizeQRError(err))
		return nil, fmt.Errorf("QRZ: %s", sanitizeQRError(err))
	}
	var authDB qrzDatabase
	if err := xml.Unmarshal(data, &authDB); err != nil {
		return nil, err
	}
	if authDB.Session.Error != "" {
		applog.Error("QRZ auth error", "msg", authDB.Session.Error)
		return nil, fmt.Errorf("QRZ: %s", authDB.Session.Error)
	}
	if authDB.Session.Key == "" {
		return nil, fmt.Errorf("QRZ: no session key")
	}
	c.mu.Lock()
	c.key = authDB.Session.Key
	c.kUser = c.user
	c.kPass = c.pass
	c.mu.Unlock()
	return c.qrzLookup(authDB.Session.Key, callsign)
}

// ---------------------------------------------------------------------------
// Legacy package-level API — delegates to a transient Client.
// Prefer constructing a Client via NewClient and keeping it for the
// lifetime of the connection.
// ---------------------------------------------------------------------------

// clientFor returns a transient Client for the legacy API. The session
// cache is lost after the call because the Client is discarded.
func clientFor(user, pass string) *Client {
	return &Client{user: user, pass: pass, httpFn: httpGetFn}
}

// Lookup is the legacy package-level callbook lookup. New code should use
// NewClient(user, pass).Lookup(callsign) instead.
func Lookup(qrzUser, qrzPass, callsign string) (*CallData, error) {
	return clientFor(qrzUser, qrzPass).lookup(callsign)
}

// TestConnection is the legacy package-level connection test.
func TestConnection(user, pass string) error {
	return clientFor(user, pass).testConnection()
}

// LookupResult is the package-level callbook lookup returning a
// provider-neutral result. New code should prefer constructing a Client.
func LookupResult(user, pass, callsign string) (*callbook.Result, error) {
	return clientFor(user, pass).Lookup(callsign)
}

type CallData struct {
	Callsign   string
	Name       string
	Grid       string
	Country    string
	QTH        string
	State      string
	Zip        string
	County     string
	Class      string
	Email      string
	URL        string
	Lat        string
	Lon        string
	DXCC       string
	CQZone     string
	ITUZone    string
	ImageURL   string
	LoTW       bool
	EQSL       bool
	QSLManager string
}

type qrzDatabase struct {
	XMLName  xml.Name `xml:"QRZDatabase"`
	Session  qrzKey   `xml:"Session"`
	Callsign qrzCall  `xml:"Callsign"`
}

type qrzKey struct {
	Key   string `xml:"Key"`
	Error string `xml:"Error"`
}

type qrzCall struct {
	Call    string `xml:"call"`
	Fname   string `xml:"fname"`
	Name    string `xml:"name"`
	Grid    string `xml:"grid"`
	Country string `xml:"country"`
	State   string `xml:"state"`
	Addr2   string `xml:"addr2"`
	Zip     string `xml:"zip"`
	County  string `xml:"county"`
	Class   string `xml:"class"`
	Email   string `xml:"email"`
	URL     string `xml:"url"`
	Lat     string `xml:"lat"`
	Lon     string `xml:"lon"`
	DXCC    string `xml:"dxcc"`
	CQZone  string `xml:"ccol"`
	ITUZone string `xml:"wcol"`
	Image   string `xml:"image"`
	LoTW    string `xml:"lotw"`
	EQSL    string `xml:"eqsl"`
	QSLMgr  string `xml:"qslmgr"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// sanitizeQRError strips the QRZ password from error messages.
func sanitizeQRError(err error) string {
	msg := err.Error()
	re := regexp.MustCompile(`;password=[^;&?]+`)
	return re.ReplaceAllString(msg, ";password=****")
}

func httpGet(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CQOps/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// httpGetFn is the test seam — replaceable with httptest.Server.
var httpGetFn = httpGet

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
