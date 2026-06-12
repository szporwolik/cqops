package qso

import (
	"math"
)

func DeriveBand(freqMHz float64) string {
	for _, r := range bandRanges {
		if freqMHz >= r.low && freqMHz <= r.high {
			return r.name
		}
	}
	return ""
}

type bandRange struct {
	low, high float64
	name      string
}

var bandRanges = []bandRange{
	{1.8, 2.0, "160M"},
	{3.5, 4.0, "80M"},
	{5.3305, 5.4065, "60M"},
	{7.0, 7.3, "40M"},
	{10.1, 10.15, "30M"},
	{14.0, 14.35, "20M"},
	{18.068, 18.168, "17M"},
	{21.0, 21.45, "15M"},
	{24.89, 24.99, "12M"},
	{28.0, 29.7, "10M"},
	{50.0, 54.0, "6M"},
	{144.0, 148.0, "2M"},
	{430.0, 440.0, "70CM"},
}

func FreqToHz(mhz float64) int64 {
	return int64(math.Round(mhz * 1_000_000))
}
