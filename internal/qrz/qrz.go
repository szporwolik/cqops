package qrz

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

type CallData struct {
	Callsign string
	Name     string
	Grid     string
	Country  string
	QTH      string
	State    string
	Zip      string
	County   string
	Class    string
	Email    string
	URL      string
	Lat      string
	Lon      string
	DXCC     string
	CQZone   string
	ITUZone  string
	ImageURL string
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
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Session cache to avoid re-authenticating on every lookup.
var (
	cachedSessionKey  string
	cachedSessionUser string
	cachedSessionPass string
)

func Lookup(qrzUser, qrzPass, callsign string) (*CallData, error) {
	if callsign == "" || qrzUser == "" {
		return nil, nil
	}

	// Reuse cached session key if credentials match
	if cachedSessionKey != "" && cachedSessionUser == qrzUser && cachedSessionPass == qrzPass {
		data, err := qrzLookup(cachedSessionKey, callsign)
		if err == nil {
			return data, nil
		}
		// Session expired or failed — clear cache and fall through to re-auth
		applog.Debug("QRZ cached session failed, re-authenticating")
		cachedSessionKey = ""
	}

	return qrzLoginLookup(qrzUser, qrzPass, callsign)
}

func qrzLookup(sessionKey, callsign string) (*CallData, error) {
	u := "https://xmldata.qrz.com/xml/current/?s=" + url.QueryEscape(sessionKey) + ";callsign=" + url.QueryEscape(callsign)
	data, err := httpGet(u)
	if err != nil {
		applog.Error("QRZ lookup failed", "error", err)
		return nil, err
	}

	var db qrzDatabase
	if err := xml.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("qrz xml: %w", err)
	}
	if db.Session.Error != "" {
		// "Not found" is a normal result, not an error
		if strings.Contains(db.Session.Error, "Not found") {
			applog.Info("QRZ: not found", "callsign", callsign)
			return nil, nil
		}
		applog.Error("QRZ lookup error", "msg", db.Session.Error)
		return nil, fmt.Errorf("QRZ: %s", db.Session.Error)
	}

	c := db.Callsign
	if strings.TrimSpace(c.Call) == "" {
		applog.Debug("QRZ no data", "callsign", callsign)
		return nil, nil
	}
	applog.InfoDetail("QRZ lookup ok", fmt.Sprintf("%s — %s", c.Call, coalesce(c.Fname, c.Name)))
	return &CallData{
		Callsign: strings.TrimSpace(c.Call),
		Name:     strings.TrimSpace(coalesce(c.Fname, c.Name)),
		Grid:     strings.TrimSpace(c.Grid),
		Country:  strings.TrimSpace(c.Country),
		State:    strings.TrimSpace(c.State),
		QTH:      strings.TrimSpace(c.Addr2),
		Zip:      strings.TrimSpace(c.Zip),
		County:   strings.TrimSpace(c.County),
		Class:    strings.TrimSpace(c.Class),
		Email:    strings.TrimSpace(c.Email),
		URL:      strings.TrimSpace(c.URL),
		Lat:      strings.TrimSpace(c.Lat),
		Lon:      strings.TrimSpace(c.Lon),
		DXCC:     strings.TrimSpace(c.DXCC),
		CQZone:   strings.TrimSpace(c.CQZone),
		ITUZone:  strings.TrimSpace(c.ITUZone),
		ImageURL: strings.TrimSpace(c.Image),
	}, nil
}

func qrzLoginLookup(user, pass, callsign string) (*CallData, error) {
	applog.Debug("QRZ lookup", "callsign", callsign)
	u := "https://xmldata.qrz.com/xml/current/?username=" + url.QueryEscape(user) + ";password=" + url.QueryEscape(pass) + ";agent=CQOps"
	data, err := httpGet(u)
	if err != nil {
		applog.Error("QRZ auth failed", "error", err)
		return nil, err
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

	// Cache the session key
	cachedSessionKey = authDB.Session.Key
	cachedSessionUser = user
	cachedSessionPass = pass

	return qrzLookup(authDB.Session.Key, callsign)
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

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
