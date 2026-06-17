package qso

import (
	"math"
	"strings"
)

type bandRange struct {
	low, high float64
	name      string
}

var bandRanges = []bandRange{
	{0.1357, 0.1378, "2190m"},
	{0.472, 0.479, "630m"},
	{0.501, 0.504, "560m"},
	{1.8, 2.0, "160m"},
	{3.5, 4.0, "80m"},
	{5.06, 5.45, "60m"},
	{7.0, 7.3, "40m"},
	{10.1, 10.15, "30m"},
	{14.0, 14.35, "20m"},
	{18.068, 18.168, "17m"},
	{21.0, 21.45, "15m"},
	{24.890, 24.99, "12m"},
	{28.0, 29.7, "10m"},
	{40.0, 45.0, "8m"},
	{50.0, 54.0, "6m"},
	{54.000001, 69.9, "5m"},
	{70.0, 71.0, "4m"},
	{144.0, 148.0, "2m"},
	{222.0, 225.0, "1.25m"},
	{420.0, 450.0, "70cm"},
	{902.0, 928.0, "33cm"},
	{1240.0, 1300.0, "23cm"},
	{2300.0, 2450.0, "13cm"},
	{3300.0, 3500.0, "9cm"},
	{5650.0, 5925.0, "6cm"},
	{10000.0, 10500.0, "3cm"},
	{24000.0, 24250.0, "1.25cm"},
	{47000.0, 47200.0, "6mm"},
	{75500.0, 81000.0, "4mm"},
	{119980.0, 123000.0, "2.5mm"},
	{134000.0, 149000.0, "2mm"},
	{241000.0, 250000.0, "1mm"},
	{300000.0, 7500000.0, "submm"},
}

var bandIndex map[string]int

func init() {
	bandIndex = make(map[string]int, len(bandRanges))
	for i, r := range bandRanges {
		bandIndex[strings.ToLower(r.name)] = i
	}
}

func DeriveBand(freqMHz float64) string {
	for _, r := range bandRanges {
		if freqMHz >= r.low && freqMHz <= r.high {
			return r.name
		}
	}
	return ""
}

func IsValidBand(band string) bool {
	_, ok := bandIndex[strings.ToLower(strings.TrimSpace(band))]
	return ok
}

func NormalizeBand(band string) string {
	key := strings.ToLower(strings.TrimSpace(band))
	if idx, ok := bandIndex[key]; ok {
		return bandRanges[idx].name
	}
	return band
}

func BandRange(band string) (low, high float64, ok bool) {
	key := strings.ToLower(strings.TrimSpace(band))
	idx, found := bandIndex[key]
	if !found {
		return 0, 0, false
	}
	r := bandRanges[idx]
	return r.low, r.high, true
}

func AllBands() []string {
	result := make([]string, len(bandRanges))
	for i, r := range bandRanges {
		result[i] = r.name
	}
	return result
}

func FreqToHz(mhz float64) int64 {
	return int64(math.Round(mhz * 1_000_000))
}
