package qrz

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CallData struct {
	Callsign string
	Name     string
	Grid     string
	Country  string
	QTH      string
	State    string
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
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func Lookup(qrzUser, qrzPass, callsign string) (*CallData, error) {
	if callsign == "" || qrzUser == "" { return nil, nil }

	return qrzLoginLookup(qrzUser, qrzPass, callsign)
}

func qrzLoginLookup(user, pass, callsign string) (*CallData, error) {
	u := "https://xmldata.qrz.com/xml/current/?username=" + url.QueryEscape(user) + ";password=" + url.QueryEscape(pass) + ";agent=CQOps"
	data, err := httpGet(u)
	if err != nil { return nil, err }

	var authDB qrzDatabase
	if err := xml.Unmarshal(data, &authDB); err != nil { return nil, err }
	if authDB.Session.Error != "" { return nil, fmt.Errorf("QRZ: %s", authDB.Session.Error) }
	if authDB.Session.Key == "" { return nil, fmt.Errorf("QRZ: no session key") }

	u2 := "https://xmldata.qrz.com/xml/current/?s=" + url.QueryEscape(authDB.Session.Key) + ";callsign=" + url.QueryEscape(callsign)
	data, err = httpGet(u2)
	if err != nil { return nil, err }

	var db qrzDatabase
	if err := xml.Unmarshal(data, &db); err != nil { return nil, fmt.Errorf("qrz xml: %w", err) }
	if db.Session.Error != "" { return nil, fmt.Errorf("QRZ: %s", db.Session.Error) }

	c := db.Callsign
	if strings.TrimSpace(c.Call) == "" { return nil, nil }
	return &CallData{
		Callsign: strings.TrimSpace(c.Call),
		Name:     strings.TrimSpace(coalesce(c.Fname, c.Name)),
		Grid:     strings.TrimSpace(c.Grid),
		Country:  strings.TrimSpace(c.Country),
		State:    strings.TrimSpace(c.State),
		QTH:      strings.TrimSpace(c.Addr2),
	}, nil
}

func httpGet(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil { return nil, err }
	req.Header.Set("User-Agent", "CQOps/1.0")
	resp, err := httpClient.Do(req)
	if err != nil { return nil, fmt.Errorf("request: %w", err) }
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func coalesce(a, b string) string { if a != "" { return a }; return b }
