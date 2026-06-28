// Package solar fetches and caches solar/geomagnetic propagation data
// from hamqsl.com. Data is cached for 15 minutes and used to display
// current HF/VHF conditions to the operator.
package solar

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

// httpClient is shared across all solar data API calls.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// Data holds the parsed solar/geomagnetic conditions used by CQOps.
// Only the fields relevant to HF/VHF operation are captured.
type Data struct {
	FetchedAt time.Time `json:"fetched_at"`

	SolarFlux    int     `json:"solarflux"`     // SFI — 10.7 cm solar radio flux
	AIndex       int     `json:"aindex"`        // Geomagnetic A-index
	KIndex       float64 `json:"kindex"`        // Geomagnetic K-index (current)
	KIndexNT     string  `json:"kindexnt"`      // K-index no-time value (can be "No Report")
	Sunspots     int     `json:"sunspots"`      // Sunspot number
	XRay         string  `json:"xray"`          // X-ray class e.g. B6.0, C1.2
	HeliumLine   float64 `json:"heliumline"`    // 30.4 nm helium line
	ProtonFlux   float64 `json:"protonflux"`    // Proton flux
	ElectronFlux float64 `json:"electronflux"`  // Electron flux
	Aurora       float64 `json:"aurora"`        // Aurora index
	SolarWind    float64 `json:"solarwind"`     // Solar wind speed (km/s)
	MagField     float64 `json:"magneticfield"` // Interplanetary magnetic field (nT)
	GeomagField  string  `json:"geomagfield"`   // Geomagnetic field status (QUIET, ACTIVE, STORM, etc.)
	SignalNoise  string  `json:"signalnoise"`   // Signal noise level (S1-S2, S3-S4, etc.)
	Updated      string  `json:"updated"`       // Raw update string from source

	// Band conditions: key = "80m-40m_day", value = "Good"/"Fair"/"Poor"
	Bands map[string]string `json:"bands"`
}

// BandOrder lists the HF bands in the order they appear in the solar panel table.
var BandOrder = []string{"80m-40m", "30m-20m", "17m-15m", "12m-10m"}

// BandShort maps a band key to its compact display label.
var BandShort = map[string]string{
	"80m-40m": "80-40",
	"30m-20m": "30-20",
	"17m-15m": "17-15",
	"12m-10m": "12-10",
}

// cachedFilePath returns the path to the cached solar XML file.
func cachedFilePath(cacheDir string) string {
	return filepath.Join(cacheDir, "solar.xml")
}

// Fetch retrieves solar data from hamqsl.com. Results are cached to cacheDir
// for 15 minutes. Returns the parsed data. Errors are non-fatal — the caller should
// simply skip displaying solar data.
func Fetch(cacheDir string) (*Data, error) {
	if cacheDir == "" {
		return nil, fmt.Errorf("no cache directory")
	}

	cacheFile := cachedFilePath(cacheDir)

	// Check cache — valid for 15 minutes.
	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < 15*time.Minute {
			data, err := os.ReadFile(cacheFile)
			if err == nil {
				d, err := Parse(data)
				if err == nil {
					applog.Debug("Solar: using cached data", "age", time.Since(info.ModTime()).Round(time.Second))
					return d, nil
				}
			}
		}
	}

	// Fetch fresh data.
	applog.Info("Solar: fetching hamqsl.com")
	resp, err := httpClient.Get("https://www.hamqsl.com/solarxml.php")
	if err != nil {
		applog.Warn("Solar: fetch failed", "error", err)
		// Return stale cache if available.
		if data, rerr := os.ReadFile(cacheFile); rerr == nil {
			if d, perr := Parse(data); perr == nil {
				info, _ := os.Stat(cacheFile)
				modTime := time.Now()
				if info != nil {
					modTime = info.ModTime()
				}
				d.FetchedAt = modTime
				applog.Info("Solar: using stale cache after fetch failure")
				return d, nil
			}
		}
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Cache the raw XML to disk — write before parsing so we always have
	// a fallback on disk even if parsing somehow fails.
	if err := os.MkdirAll(cacheDir, 0755); err == nil {
		if err := os.WriteFile(cacheFile, body, 0644); err != nil {
			applog.Warn("Solar: failed to write cache", "error", err)
		}
	}

	d, err := Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	applog.Info("Solar: fetched OK", "solarflux", d.SolarFlux, "aindex", d.AIndex, "kindex", d.KIndex)
	return d, nil
}

// Cached returns the cached solar data and a boolean indicating whether it is
// still fresh (within the last hour). Returns nil, false if no cached data exists.
func Cached(cacheDir string) (*Data, bool) {
	if cacheDir == "" {
		return nil, false
	}
	cacheFile := cachedFilePath(cacheDir)
	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}
	d, err := Parse(data)
	if err != nil {
		return nil, false
	}
	d.FetchedAt = info.ModTime()
	fresh := time.Since(info.ModTime()) < 15*time.Minute
	return d, fresh
}

// Parse decodes the hamqsl.com solar XML into a Data struct.
// Returns an error only if the XML is fundamentally malformed (missing root,
// unparseable numbers on critical fields). Minor field parse failures are
// logged and the field is left at its zero value.
func Parse(xmlBytes []byte) (*Data, error) {
	var raw xmlSolar
	if err := xml.Unmarshal(xmlBytes, &raw); err != nil {
		return nil, fmt.Errorf("xml unmarshal: %w", err)
	}

	sd := raw.SolarData
	d := &Data{
		FetchedAt: time.Now().UTC(),
		Updated:   strings.TrimSpace(sd.Updated),
	}

	// Numeric fields — trim spaces because hamqsl.com uses leading spaces
	// (e.g. <aindex> 6</aindex>).
	d.SolarFlux, _ = strconv.Atoi(strings.TrimSpace(sd.SolarFlux))
	a, err := strconv.Atoi(strings.TrimSpace(sd.AIndex))
	if err != nil && strings.TrimSpace(sd.AIndex) != "" {
		applog.Warn("Solar: bad aindex", "value", sd.AIndex, "error", err)
	}
	d.AIndex = a
	k, err := strconv.ParseFloat(strings.TrimSpace(sd.KIndex), 64)
	if err != nil && strings.TrimSpace(sd.KIndex) != "" {
		applog.Warn("Solar: bad kindex", "value", sd.KIndex, "error", err)
	}
	d.KIndex = k
	d.KIndexNT = strings.TrimSpace(sd.KIndexNT)
	ssn, ssnErr := strconv.Atoi(strings.TrimSpace(sd.Sunspots))
	if ssnErr != nil && strings.TrimSpace(sd.Sunspots) != "" {
		applog.Warn("Solar: bad sunspots", "value", sd.Sunspots, "error", ssnErr)
	}
	d.Sunspots = ssn
	d.XRay = strings.TrimSpace(sd.XRay)
	d.HeliumLine, _ = parseFloat(sd.HeliumLine)
	// hamqsl.com has a typo in the XML: "electonflux" instead of "electronflux".
	d.ProtonFlux, _ = parseFloat(sd.ProtonFlux)
	d.ElectronFlux, _ = parseFloat(sd.ElectronFlux)
	d.Aurora, _ = parseFloat(sd.Aurora)
	d.SolarWind, _ = parseFloat(sd.SolarWind)
	d.MagField, _ = parseFloat(sd.MagneticField)
	d.GeomagField = strings.TrimSpace(sd.GeomagField)
	d.SignalNoise = strings.TrimSpace(sd.SignalNoise)

	// Band conditions.
	d.Bands = make(map[string]string)
	for _, b := range sd.Conditions.Bands {
		key := b.Name + "_" + b.Time
		d.Bands[key] = strings.TrimSpace(b.Value)
	}

	return d, nil
}

// parseFloat trims and parses a float64, returning 0 on failure.
func parseFloat(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

// ---------------------------------------------------------------------------
// XML mapping types (unexported)
// ---------------------------------------------------------------------------

type xmlSolar struct {
	SolarData xmlSolarData `xml:"solardata"`
}

type xmlSolarData struct {
	Source        xmlSource     `xml:"source"`
	Updated       string        `xml:"updated"`
	SolarFlux     string        `xml:"solarflux"`
	AIndex        string        `xml:"aindex"`
	KIndex        string        `xml:"kindex"`
	KIndexNT      string        `xml:"kindexnt"`
	XRay          string        `xml:"xray"`
	Sunspots      string        `xml:"sunspots"`
	HeliumLine    string        `xml:"heliumline"`
	ProtonFlux    string        `xml:"protonflux"`
	ElectronFlux  string        `xml:"electonflux"` // intentional typo in source XML
	Aurora        string        `xml:"aurora"`
	SolarWind     string        `xml:"solarwind"`
	MagneticField string        `xml:"magneticfield"`
	GeomagField   string        `xml:"geomagfield"`
	SignalNoise   string        `xml:"signalnoise"`
	Conditions    xmlConditions `xml:"calculatedconditions"`
}

type xmlSource struct {
	URL  string `xml:"url,attr"`
	Name string `xml:",chardata"`
}

type xmlConditions struct {
	Bands []xmlBand `xml:"band"`
}

type xmlBand struct {
	Name  string `xml:"name,attr"`
	Time  string `xml:"time,attr"`
	Value string `xml:",chardata"`
}
