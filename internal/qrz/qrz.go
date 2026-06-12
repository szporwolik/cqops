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
	XMLName xml.Name   `xml:"QRZDatabase"`
	Version string     `xml:"version"`
	Session qrzKey     `xml:"Session"`
	Callsign qrzCall   `xml:"Callsign"`
}

type qrzKey struct {
	Key     string `xml:"Key"`
	Count   string `xml:"Count"`
	SubExp  string `xml:"SubExp"`
	Message string `xml:"Message"`
	Error   string `xml:"Error"`
}

type qrzCall struct {
	Call    string `xml:"call"`
	Fname   string `xml:"fname"`
	Name    string `xml:"name"`
	Grid    string `xml:"grid"`
	Country string `xml:"country"`
	State   string `xml:"state"`
	Addr2   string `xml:"addr2"`
	Error   string `xml:"Error"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func Lookup(apiKey, callsign string) (*CallData, error) {
	if apiKey == "" || callsign == "" {
		return nil, nil
	}

	baseURL := "https://xmldata.qrz.com/xml/current/"

	sessionKey, err := getSession(baseURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("qrz session: %w", err)
	}

	u := baseURL + "?s=" + url.QueryEscape(sessionKey+";callsign="+callsign)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CQOps/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qrz request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var db qrzDatabase
	if err := xml.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("qrz xml: %w", err)
	}

	if db.Callsign.Error != "" {
		return nil, fmt.Errorf("qrz: %s", db.Callsign.Error)
	}

	cd := &CallData{
		Callsign: strings.TrimSpace(db.Callsign.Call),
		Name:     strings.TrimSpace(coalesce(db.Callsign.Fname, db.Callsign.Name)),
		Grid:     strings.TrimSpace(db.Callsign.Grid),
		Country:  strings.TrimSpace(db.Callsign.Country),
		State:    strings.TrimSpace(db.Callsign.State),
		QTH:      strings.TrimSpace(db.Callsign.Addr2),
	}

	if cd.Callsign == "" {
		return nil, nil
	}

	return cd, nil
}

func getSession(baseURL, apiKey string) (string, error) {
	u := baseURL + "?s=" + url.QueryEscape(apiKey)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "CQOps/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var db qrzDatabase
	if err := xml.Unmarshal(data, &db); err != nil {
		return "", fmt.Errorf("qrz xml: %w", err)
	}

	if db.Session.Error != "" {
		return "", fmt.Errorf("qrz auth: %s", db.Session.Error)
	}

	if db.Session.Key == "" {
		return "", fmt.Errorf("qrz: no session key returned")
	}

	return db.Session.Key, nil
}

func coalesce(a, b string) string {
	if a != "" { return a }
	return b
}
