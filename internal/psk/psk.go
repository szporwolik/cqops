package psk

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

// httpClient is shared across all PSK Reporter API calls.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// Report holds a single PSK Reporter reception record.
type Report struct {
	ReceiverCallsign string
	ReceiverLocator  string
	Frequency        float64 // Hz
	SNR              int     // dB
	Mode             string
	FlowStartSeconds int64 // Unix timestamp
}

// receptionReports is the XML root element.
type receptionReports struct {
	XMLName xml.Name          `xml:"receptionReports"`
	Reports []receptionReport `xml:"receptionReport"`
}

type receptionReport struct {
	ReceiverCallsign string `xml:"receiverCallsign,attr"`
	ReceiverLocator  string `xml:"receiverLocator,attr"`
	Frequency        string `xml:"frequency,attr"`
	SNR              string `xml:"sNR,attr"`
	Mode             string `xml:"mode,attr"`
	FlowStartSeconds string `xml:"flowStartSeconds,attr"`
}

// FetchReports queries the PSK Reporter API for the given callsign.
// Results are cached to cacheDir for 5 minutes.
// Returns the parsed reports and the time of the fetch.
func FetchReports(callsign, cacheDir string) ([]Report, time.Time, error) {
	if callsign == "" {
		return nil, time.Time{}, fmt.Errorf("no callsign provided")
	}

	cacheFile := filepath.Join(cacheDir, "psk_"+strings.ToUpper(callsign)+".xml")

	// Check cache — valid for 5 minutes.
	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < 5*time.Minute {
			data, err := os.ReadFile(cacheFile)
			if err == nil {
				reports, err := parseReports(data)
				if err == nil {
					applog.Debug("PSK Reporter: using cached data", "callsign", callsign, "age", time.Since(info.ModTime()).Round(time.Second))
					return reports, info.ModTime(), nil
				}
			}
		}
	}

	// Fetch from API.
	url := fmt.Sprintf("https://retrieve.pskreporter.info/query?senderCallsign=%s&rptlimit=100",
		strings.ToUpper(callsign))

	applog.Info("PSK Reporter: fetching", "callsign", callsign)
	resp, err := httpClient.Get(url)
	if err != nil {
		applog.Error("PSK Reporter: fetch failed", "callsign", callsign, "error", err)
		// Return cached data even if expired — better than nothing.
		if data, rerr := os.ReadFile(cacheFile); rerr == nil {
			if reports, perr := parseReports(data); perr == nil {
				info, _ := os.Stat(cacheFile)
				modTime := time.Now()
				if info != nil {
					modTime = info.ModTime()
				}
				applog.Info("PSK Reporter: using stale cache after fetch failure", "callsign", callsign)
				return reports, modTime, nil
			}
		}
		return nil, time.Time{}, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		applog.Error("PSK Reporter: server error", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return nil, time.Time{}, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	// Cache the response.
	if err := os.MkdirAll(cacheDir, 0755); err == nil {
		if err := os.WriteFile(cacheFile, body, 0644); err != nil {
			applog.Warn("PSK Reporter: failed to write cache", "file", cacheFile, "error", err)
		}
	}

	reports, err := parseReports(body)
	if err != nil {
		return nil, time.Time{}, err
	}

	applog.Info("PSK Reporter: fetched OK", "callsign", callsign, "reports", len(reports))
	return reports, time.Now(), nil
}

func parseReports(data []byte) ([]Report, error) {
	var rr receptionReports
	if err := xml.Unmarshal(data, &rr); err != nil {
		return nil, fmt.Errorf("parse XML: %w", err)
	}

	var reports []Report
	for _, r := range rr.Reports {
		freq, err := strconv.ParseFloat(r.Frequency, 64)
		if err != nil && r.Frequency != "" {
			applog.Warn("PSK Reporter: bad frequency in report", "freq", r.Frequency, "error", err)
		}
		snr, err := strconv.Atoi(r.SNR)
		if err != nil && r.SNR != "" {
			applog.Warn("PSK Reporter: bad SNR in report", "snr", r.SNR, "error", err)
		}
		ts, err := strconv.ParseInt(r.FlowStartSeconds, 10, 64)
		if err != nil && r.FlowStartSeconds != "" {
			applog.Warn("PSK Reporter: bad timestamp in report", "ts", r.FlowStartSeconds, "error", err)
		}
		reports = append(reports, Report{
			ReceiverCallsign: strings.ToUpper(r.ReceiverCallsign),
			ReceiverLocator:  strings.ToUpper(r.ReceiverLocator),
			Frequency:        freq,
			SNR:              snr,
			Mode:             strings.ToUpper(r.Mode),
			FlowStartSeconds: ts,
		})
	}
	return reports, nil
}
