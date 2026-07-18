// Package qrzru provides a callbook.Provider for the QRZ.RU XML API.
// QRZ.RU is a free amateur radio callbook focused on Russia and surrounding
// countries. It requires a dedicated API login/password (separate from the
// website credentials) obtained from the QRZ.RU personal cabinet.
//
// API docs: https://m.qrz.ru/help/api/xml
package qrzru

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/callbook"
)

// Client implements callbook.Provider for the QRZ.RU XML API.
type Client struct {
	user     string
	pass     string
	priority int
	httpFn   func(string) ([]byte, error) // test seam

	mu        sync.Mutex
	sessionID string
	sessUser  string
	sessPass  string
	lastReq   time.Time // rate-limit: 1 req per 3 s
}

// NewClientWithPriority creates a QRZ.RU client with explicit priority.
func NewClientWithPriority(user, pass string, priority int) *Client {
	return &Client{user: user, pass: pass, priority: priority, httpFn: httpGetFn}
}

// Name returns the provider identifier.
func (c *Client) Name() string { return "QRZ.RU" }

// Priority returns the lookup priority.
func (c *Client) Priority() int { return c.priority }

// Lookup queries QRZ.RU for a callsign.
func (c *Client) Lookup(callsign string) (*callbook.Result, error) {
	if callsign == "" || c.user == "" {
		return nil, nil
	}
	cd, err := c.lookup(callsign)
	if err != nil {
		return nil, err
	}
	if cd == nil {
		return nil, nil
	}
	return &callbook.Result{
		Callsign: cd.Callsign,
		Name:     cd.Name,
		QTH:      cd.QTH,
		// Country intentionally omitted — QRZ.RU returns Cyrillic names.
		// CTY provider (Big CTY) fills the English entity name.
		Grid:     cd.Grid,
		Lat:      cd.Lat,
		Lon:      cd.Lon,
		Zip:      cd.Zip,
		URL:      cd.URL,
		Class:    cd.Class,
		ImageURL: cd.ImageURL,
		LoTW:     cd.LoTW,
		EQSL:     cd.EQSL,
		Provider: "qrzru",
	}, nil
}

// TestConnection validates QRZ.RU credentials.
func (c *Client) TestConnection() error {
	return c.testConnection()
}

// --- internals ---------------------------------------------------------------

type callData struct {
	Callsign string
	Name     string
	QTH      string
	Grid     string
	Lat      string
	Lon      string
	Zip      string
	URL      string
	Class    string
	ImageURL string
	LoTW     bool
	EQSL     bool
}

func (c *Client) lookup(callsign string) (*callData, error) {
	if callsign == "" || c.user == "" {
		return nil, nil
	}

	// Reuse cached session if credentials still match.
	c.mu.Lock()
	sid := c.sessionID
	su := c.sessUser
	sp := c.sessPass
	c.mu.Unlock()

	if sid != "" && su == c.user && sp == c.pass {
		cd, err := c.doLookup(sid, callsign)
		if err == nil {
			return cd, nil
		}
		applog.Debug("QRZ.RU: cached session expired, re-authenticating")
		c.mu.Lock()
		c.sessionID = ""
		c.mu.Unlock()
	}
	return c.loginAndLookup(callsign)
}

func (c *Client) testConnection() error {
	applog.Debug("QRZ.RU: testing connection")
	if c.user == "" || c.pass == "" {
		return fmt.Errorf("QRZ.RU API login and password required")
	}
	sid, err := c.doLogin()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.sessionID = sid
	c.sessUser = c.user
	c.sessPass = c.pass
	c.mu.Unlock()
	applog.InfoDetail("QRZ.RU: connected", fmt.Sprintf("user=%s url=api.qrz.ru", c.user))
	return nil
}

func (c *Client) loginAndLookup(callsign string) (*callData, error) {
	sid, err := c.doLogin()
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.sessionID = sid
	c.sessUser = c.user
	c.sessPass = c.pass
	c.mu.Unlock()
	return c.doLookup(sid, callsign)
}

func (c *Client) doLogin() (string, error) {
	u := "https://api.qrz.ru/login?u=" + url.QueryEscape(c.user) +
		"&p=" + url.QueryEscape(c.pass) +
		"&agent=CQOps"
	data, err := c.httpFn(u)
	if err != nil {
		return "", fmt.Errorf("QRZ.RU login: %w", err)
	}
	var db qrzDB
	if err := xml.Unmarshal(data, &db); err != nil {
		return "", fmt.Errorf("QRZ.RU xml: %w", err)
	}
	if db.Session.ErrorCode != 0 || (db.Session.Error != "" && !strings.EqualFold(db.Session.Error, "ok")) {
		if db.Session.Error != "" {
			return "", fmt.Errorf("QRZ.RU: %s", db.Session.Error)
		}
		return "", fmt.Errorf("QRZ.RU: error code %d", db.Session.ErrorCode)
	}
	if db.Session.SessionID == "" {
		return "", fmt.Errorf("QRZ.RU: no session id")
	}
	return db.Session.SessionID, nil
}

func (c *Client) doLookup(sid, callsign string) (*callData, error) {
	// Rate limit: 1 request per 3 seconds after initial burst.
	c.mu.Lock()
	if !c.lastReq.IsZero() {
		elapsed := time.Since(c.lastReq)
		if elapsed < 3*time.Second {
			c.mu.Unlock()
			time.Sleep(3*time.Second - elapsed)
			c.mu.Lock()
		}
	}
	c.lastReq = time.Now()
	c.mu.Unlock()

	u := "https://api.qrz.ru/callsign?id=" + url.QueryEscape(sid) +
		"&callsign=" + url.QueryEscape(callsign)
	data, err := c.httpFn(u)
	if err != nil {
		return nil, fmt.Errorf("QRZ.RU lookup: %w", err)
	}
	var db qrzDB
	if err := xml.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("QRZ.RU xml: %w", err)
	}
	if db.Session.ErrorCode != 0 || (db.Session.Error != "" && !strings.EqualFold(db.Session.Error, "ok")) {
		if strings.Contains(db.Session.Error, "not found") || strings.Contains(db.Session.Error, "Not found") {
			applog.Debug("QRZ.RU: not found", "callsign", callsign)
			return nil, nil
		}
		if db.Session.Error != "" {
			return nil, fmt.Errorf("QRZ.RU: %s", db.Session.Error)
		}
		return nil, fmt.Errorf("QRZ.RU: error code %d", db.Session.ErrorCode)
	}
	call := db.Callsign
	if strings.TrimSpace(call.Call) == "" {
		applog.Debug("QRZ.RU: no data", "callsign", callsign)
		return nil, nil
	}

	name := buildCallName(call)
	qth := buildQTH(call)
	img := imageURL(call, db.Files)

	applog.Debug("QRZ.RU: lookup ok", "callsign", call.Call, "name", name,
		"grid", call.QTHLoc, "hasPhoto", img != "")
	return &callData{
		Callsign: call.Call,
		Name:     name,
		QTH:      qth,
		Grid:     strings.ToUpper(strings.TrimSpace(call.QTHLoc)),
		Lat:      strings.TrimSpace(call.Latitude),
		Lon:      strings.TrimSpace(call.Longitude),
		Zip:      call.Zip,
		URL:      call.URL,
		Class:    strings.TrimSpace(call.Class),
		ImageURL: img,
		LoTW:     strings.EqualFold(strings.TrimSpace(call.IsLoTW), "Y"),
		EQSL:     strings.EqualFold(strings.TrimSpace(call.IsEQSL), "Y"),
	}, nil
}

// --- XML types ---------------------------------------------------------------

// qrzDB is the root element returned by api.qrz.ru.
// Same schema as QRZ.com: <QRZDatabase> → <Session> + <Callsign> + <Files>.
type qrzDB struct {
	XMLName  xml.Name    `xml:"QRZDatabase"`
	Session  qrzSession  `xml:"Session"`
	Callsign qrzCallSign `xml:"Callsign"`
	Files    qrzFiles    `xml:"Files"`
}

type qrzSession struct {
	SessionID string `xml:"session_id"`
	ErrorCode int    `xml:"errorcode"`
	Error     string `xml:"error"`
	GMTime    string `xml:"GMTime"`
}

type qrzFiles struct {
	Files []string `xml:"file"`
}

type qrzCallSign struct {
	Call      string `xml:"call"`
	Type      string `xml:"type"`
	OtherCall string `xml:"othercall"`
	CountryID string `xml:"country_id"`
	Name      string `xml:"name"`
	Surname   string `xml:"surname"`
	Name2     string `xml:"name2"` // patronymic
	EName     string `xml:"ename"`
	ESurname  string `xml:"esurname"`
	EName2    string `xml:"ename2"` // English patronymic
	Birthday  string `xml:"birthday"`
	City      string `xml:"city"`
	Street    string `xml:"street"`
	Region    string `xml:"region"`
	Zip       string `xml:"zip"`
	Phone     string `xml:"phone"`
	Country   string `xml:"country"`
	URL       string `xml:"url"`
	ICQ       string `xml:"icq"`
	Skype     string `xml:"skype"`
	QTHLoc    string `xml:"qthloc"`
	Latitude  string `xml:"latitude"`
	Longitude string `xml:"longitude"`
	Class     string `xml:"class"`
	State     string `xml:"state"`
	IsEQSL    string `xml:"is_eqsl"`
	IsLoTW    string `xml:"is_lotw"`
	IsMailQSL string `xml:"is_mailqsl"`
	Image     string `xml:"image"`
	Photo     string `xml:"photo"`
	Created   string `xml:"created"`
	LastEdit  string `xml:"lastedit"`
	Lookup    string `xml:"lookup"`
}

// buildCallName combines EName + ESurname for display, falling back to
// native Name + Surname when English transliteration is unavailable.
func buildCallName(call qrzCallSign) string {
	name := strings.TrimSpace(call.EName)
	if name == "" {
		name = strings.TrimSpace(call.Name)
	}
	s := strings.TrimSpace(call.ESurname)
	if s == "" {
		s = strings.TrimSpace(call.Surname)
	}
	if s != "" {
		if name != "" {
			name = name + " " + s
		} else {
			name = s
		}
	}
	return name
}

// buildQTH combines city and street into a single location string.
func buildQTH(call qrzCallSign) string {
	city := strings.TrimSpace(call.City)
	street := strings.TrimSpace(call.Street)
	if city != "" && street != "" {
		return city + ", " + street
	}
	if city != "" {
		return city
	}
	return street
}

// imageURL returns the best available image URL from the QRZ.RU record.
// Checks: Files block → Image field → Photo field.
func imageURL(call qrzCallSign, files qrzFiles) string {
	for _, f := range files.Files {
		if f = strings.TrimSpace(f); f != "" {
			return f
		}
	}
	if img := strings.TrimSpace(call.Image); img != "" {
		return img
	}
	return strings.TrimSpace(call.Photo)
}

// --- HTTP transport (test seam) ---------------------------------------------

var httpGetFn = defaultHTTPGet

func defaultHTTPGet(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CQOps/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 256*1024))
}
