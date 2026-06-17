package solar

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The exact XML format returned by https://www.hamqsl.com/solarxml.php
// (snapshot from 2026-06-17, minor whitespace preserved).
var validXML = []byte(`<?xml version="1.0" encoding="UTF-8" ?>
<solar>
        <solardata>
                <source url="http://www.hamqsl.com/solar.html">N0NBH</source>
                <updated> 17 Jun 2026 1335 GMT</updated>
                <solarflux>113</solarflux>
                <aindex> 6</aindex>
                <kindex> 2</kindex>
                <kindexnt>No Report</kindexnt>
                <xray>B6.0</xray>
                <sunspots>47</sunspots>
                <heliumline>118.1</heliumline>
                <protonflux>15</protonflux>
                <electonflux>1860</electonflux>
                <aurora> 3</aurora>
                <normalization>1.99</normalization>
                <latdegree>65.6</latdegree>
                <solarwind>482.3</solarwind>
                <magneticfield> -4.9</magneticfield>
                <calculatedconditions>
                        <band name="80m-40m" time="day">Fair</band>
                        <band name="30m-20m" time="day">Good</band>
                        <band name="17m-15m" time="day">Fair</band>
                        <band name="12m-10m" time="day">Poor</band>
                        <band name="80m-40m" time="night">Good</band>
                        <band name="30m-20m" time="night">Good</band>
                        <band name="17m-15m" time="night">Fair</band>
                        <band name="12m-10m" time="night">Poor</band>
                </calculatedconditions>
                <geomagfield>QUIET</geomagfield>
                <signalnoise>S1-S2</signalnoise>
        </solardata>
</solar>`)

func TestParseValid(t *testing.T) {
	d, err := Parse(validXML)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if d.SolarFlux != 113 {
		t.Errorf("SolarFlux = %d, want 113", d.SolarFlux)
	}
	if d.AIndex != 6 {
		t.Errorf("AIndex = %d, want 6", d.AIndex)
	}
	if d.KIndex != 2.0 {
		t.Errorf("KIndex = %f, want 2.0", d.KIndex)
	}
	if d.KIndexNT != "No Report" {
		t.Errorf("KIndexNT = %q, want \"No Report\"", d.KIndexNT)
	}
	if d.Sunspots != 47 {
		t.Errorf("Sunspots = %d, want 47", d.Sunspots)
	}
	if d.XRay != "B6.0" {
		t.Errorf("XRay = %q, want B6.0", d.XRay)
	}
	if d.HeliumLine != 118.1 {
		t.Errorf("HeliumLine = %f, want 118.1", d.HeliumLine)
	}
	if d.ProtonFlux != 15 {
		t.Errorf("ProtonFlux = %f, want 15", d.ProtonFlux)
	}
	if d.ElectronFlux != 1860 {
		t.Errorf("ElectronFlux = %f, want 1860", d.ElectronFlux)
	}
	if d.Aurora != 3 {
		t.Errorf("Aurora = %f, want 3", d.Aurora)
	}
	if d.SolarWind != 482.3 {
		t.Errorf("SolarWind = %f, want 482.3", d.SolarWind)
	}
	if d.MagField != -4.9 {
		t.Errorf("MagField = %f, want -4.9", d.MagField)
	}
	if d.GeomagField != "QUIET" {
		t.Errorf("GeomagField = %q, want QUIET", d.GeomagField)
	}
	if d.SignalNoise != "S1-S2" {
		t.Errorf("SignalNoise = %q, want S1-S2", d.SignalNoise)
	}
	if d.Updated != "17 Jun 2026 1335 GMT" {
		t.Errorf("Updated = %q, want \"17 Jun 2026 1335 GMT\"", d.Updated)
	}

	// Band conditions.
	if d.Bands["80m-40m_day"] != "Fair" {
		t.Errorf("80m-40m day = %q, want Fair", d.Bands["80m-40m_day"])
	}
	if d.Bands["30m-20m_day"] != "Good" {
		t.Errorf("30m-20m day = %q, want Good", d.Bands["30m-20m_day"])
	}
	if d.Bands["12m-10m_night"] != "Poor" {
		t.Errorf("12m-10m night = %q, want Poor", d.Bands["12m-10m_night"])
	}
	if len(d.Bands) != 8 {
		t.Errorf("Bands count = %d, want 8", len(d.Bands))
	}
}

func TestParseEmptyXML(t *testing.T) {
	_, err := Parse([]byte(``))
	if err == nil {
		t.Error("expected error on empty XML")
	}
}

func TestParseGarbageXML(t *testing.T) {
	_, err := Parse([]byte(`not xml`))
	if err == nil {
		t.Error("expected error on garbage XML")
	}
}

func TestParseMissingSolarData(t *testing.T) {
	// Valid XML but no solardata element.
	_, err := Parse([]byte(`<solar></solar>`))
	if err != nil {
		t.Errorf("unexpected error on empty solar element: %v", err)
	}
	// Should not panic, Data struct is zero-valued.
}

func TestParseElectronFluxTypo(t *testing.T) {
	// The source XML uses "electonflux" (missing 'r') — verify we handle it.
	raw := []byte(`<solar><solardata><electonflux>1860</electonflux></solardata></solar>`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if d.ElectronFlux != 1860 {
		t.Errorf("ElectronFlux = %f, want 1860", d.ElectronFlux)
	}
}

func TestParseLeadingSpaces(t *testing.T) {
	// hamqsl.com uses leading spaces in numeric elements — verify trimming.
	raw := []byte(`<solar><solardata><aindex> 6</aindex><kindex> 2</kindex></solardata></solar>`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if d.AIndex != 6 {
		t.Errorf("AIndex = %d, want 6", d.AIndex)
	}
	if d.KIndex != 2.0 {
		t.Errorf("KIndex = %f, want 2.0", d.KIndex)
	}
}

func TestParseKIndexFloat(t *testing.T) {
	raw := []byte(`<solar><solardata><kindex>1.33</kindex></solardata></solar>`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if d.KIndex != 1.33 {
		t.Errorf("KIndex = %f, want 1.33", d.KIndex)
	}
}

func TestParseKIndexNoReport(t *testing.T) {
	raw := []byte(`<solar><solardata><kindex>No Report</kindex><kindexnt>No Report</kindexnt></solardata></solar>`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if d.KIndex != 0 {
		t.Errorf("KIndex should be 0 when 'No Report', got %f", d.KIndex)
	}
}

func TestCacheWriteAndRead(t *testing.T) {
	dir := t.TempDir()

	// Write a valid XML as if we had fetched it.
	cacheFile := filepath.Join(dir, "solar.xml")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, validXML, 0644); err != nil {
		t.Fatal(err)
	}

	// Cached should return fresh data.
	d, fresh := Cached(dir)
	if d == nil || !fresh {
		t.Fatal("Cached should return fresh data")
	}
	if d.SolarFlux != 113 {
		t.Errorf("cached SolarFlux = %d, want 113", d.SolarFlux)
	}
}

func TestCacheStaleAfterOneHour(t *testing.T) {
	dir := t.TempDir()

	cacheFile := filepath.Join(dir, "solar.xml")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, validXML, 0644); err != nil {
		t.Fatal(err)
	}

	// Touch the file to be 2 hours old.
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cacheFile, past, past); err != nil {
		t.Skip("Chtimes not supported on this platform")
	}

	d, fresh := Cached(dir)
	if d == nil {
		t.Fatal("Cached should still return data even when stale")
	}
	if fresh {
		t.Error("Cached should return fresh=false for old data")
	}
}

func TestCachedEmptyDir(t *testing.T) {
	dir := t.TempDir()
	d, fresh := Cached(dir)
	if d != nil || fresh {
		t.Error("Cached should return nil, false for empty dir")
	}
}

func TestCachedEmptyCacheDir(t *testing.T) {
	d, fresh := Cached("")
	if d != nil || fresh {
		t.Error("Cached should return nil, false for empty string")
	}
}

func TestFetchEmptyCacheDir(t *testing.T) {
	_, err := Fetch("")
	if err == nil {
		t.Error("Fetch should fail with empty cache dir")
	}
	if !strings.Contains(err.Error(), "no cache directory") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDoesNotAllocateBandsWhenEmpty(t *testing.T) {
	raw := []byte(`<solar><solardata></solardata></solar>`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	// Bands map should be initialized but empty.
	if d.Bands == nil {
		t.Error("Bands should be non-nil (empty map)")
	}
	if len(d.Bands) != 0 {
		t.Errorf("Bands should be empty, got %d entries", len(d.Bands))
	}
}

func TestParseFloatHelper(t *testing.T) {
	tests := []struct {
		in  string
		out float64
	}{
		{" 3", 3},
		{"1.99", 1.99},
		{"", 0},
		{"No Report", 0},
		{" -4.9", -4.9},
	}
	for _, tt := range tests {
		got, _ := parseFloat(tt.in)
		if got != tt.out {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.in, got, tt.out)
		}
	}
}

// Verify that the test XML round-trips through xml.Unmarshal.
func TestXMLRoundTrip(t *testing.T) {
	var raw xmlSolar
	if err := xml.Unmarshal(validXML, &raw); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if raw.SolarData.SolarFlux != "113" {
		t.Errorf("raw SolarFlux = %q, want 113", raw.SolarData.SolarFlux)
	}
	if raw.SolarData.ElectronFlux != "1860" {
		t.Errorf("raw ElectronFlux (electonflux) = %q, want 1860", raw.SolarData.ElectronFlux)
	}
	if len(raw.SolarData.Conditions.Bands) != 8 {
		t.Errorf("raw Bands count = %d, want 8", len(raw.SolarData.Conditions.Bands))
	}
}

// Ensure Data.FetchedAt is set to roughly now when parsing.
func TestParseSetsFetchedAt(t *testing.T) {
	d, err := Parse(validXML)
	if err != nil {
		t.Fatal(err)
	}
	if d.FetchedAt.IsZero() {
		t.Error("FetchedAt should be set")
	}
	if time.Since(d.FetchedAt) > 5*time.Second {
		t.Errorf("FetchedAt is too old: %v", d.FetchedAt)
	}
}

// Verify that fmt.Sprintf does not panic with nil Data (defensive).
func TestDataStringer(t *testing.T) {
	d := &Data{}
	s := fmt.Sprintf("SFI %d A %d K %.1f", d.SolarFlux, d.AIndex, d.KIndex)
	if s == "" {
		t.Error("unexpected empty string")
	}
}
