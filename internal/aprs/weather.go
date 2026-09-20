package aprs

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Weather holds a decoded APRS weather block. Fields are only meaningful
// when their Ok flag is set.
type Weather struct {
	WindDir  int // degrees
	WindMph  int // sustained wind speed
	WindOk   bool
	TempF    int
	TempOk   bool
	Humidity int // percent; h00 encodes 100%
	HumOk    bool
	Pressure float64 // hPa (tenths of hPa on the wire)
	PressOk  bool
	RainIn   float64 // last hour, inches (hundredths on the wire)
	RainOk   bool
	Rest     string // free text after the weather block
}

var errNoNum = fmt.Errorf("no number")

// takeNum reads an optional minus sign and up to max digits starting at
// off. It returns the value and the remaining string.
func takeNum(s string, off, max int) (int, string, error) {
	body := s[off:]
	end := 0
	if len(body) > 0 && body[0] == '-' {
		end = 1
	}
	for end < len(body) && end < max && body[end] >= '0' && body[end] <= '9' {
		end++
	}
	if end == 0 || (end == 1 && body[0] == '-') {
		return 0, s, errNoNum
	}
	v, err := strconv.Atoi(body[:end])
	if err != nil {
		return 0, s, err
	}
	return v, body[end:], nil
}

// ParseWeather decodes the standard APRS weather block that starts a
// weather-station comment: "DDD/SSSgGGGtTTTrRRRpPPPPhHHbBBBBB…" where DDD
// is wind direction, SSS wind speed in mph, t temperature in °F, h humidity
// (00 = 100%), b pressure in tenths of hPa. Returns false when the comment
// does not start with a weather block.
func ParseWeather(comment string) (Weather, bool) {
	var w Weather
	c := strings.TrimSpace(comment)
	// Wind direction/speed: DDD/SSS.
	if len(c) < 7 || c[3] != '/' {
		return w, false
	}
	dir, err1 := strconv.Atoi(c[0:3])
	spd, err2 := strconv.Atoi(c[4:7])
	if err1 != nil || err2 != nil || dir < 0 || dir > 360 {
		return w, false
	}
	w.WindDir, w.WindMph, w.WindOk = dir, spd, true
	c = c[7:]

	// Optional fields in spec order.
	for len(c) > 0 {
		var v int
		var err error
		switch c[0] {
		case 'g':
			// Gust — parsed and skipped so the loop does not treat it as
			// free text; not displayed to keep the line compact.
			if _, c, err = takeNum(c, 1, 3); err != nil {
				goto done
			}
		case 't':
			if v, c, err = takeNum(c, 1, 3); err != nil {
				goto done
			}
			w.TempF, w.TempOk = v, true
		case 'r', 'p', 'P':
			if v, c, err = takeNum(c, 1, 3); err != nil {
				goto done
			}
			w.RainIn, w.RainOk = float64(v)/100, true
		case 'h':
			if v, c, err = takeNum(c, 1, 2); err != nil {
				goto done
			}
			if v == 0 {
				v = 100 // APRS encodes 100% humidity as h00
			}
			w.Humidity, w.HumOk = v, true
		case 'b':
			if v, c, err = takeNum(c, 1, 5); err != nil {
				goto done
			}
			w.Pressure, w.PressOk = float64(v)/10, true
		default:
			goto done
		}
	}

done:
	w.Rest = strings.TrimSpace(c)
	return w, true
}

// Format renders the decoded weather compactly:
// "236° 35 km/h · 24°C · 79% · 1017 hPa". metric selects °C and km/h,
// otherwise °F and mph. Any free text after the block is appended.
func (w Weather) Format(metric bool) string {
	var parts []string
	if w.WindOk && w.WindMph > 0 {
		speed := w.WindMph
		unit := "mph"
		if metric {
			speed = int(math.Round(float64(w.WindMph) * 1.609344))
			unit = "km/h"
		}
		parts = append(parts, fmt.Sprintf("%03d\u00b0 %d %s", w.WindDir, speed, unit))
	}
	if w.TempOk {
		t := float64(w.TempF)
		unit := "\u00b0F"
		if metric {
			t = math.Round((t - 32) * 5 / 9)
			unit = "\u00b0C"
		}
		parts = append(parts, fmt.Sprintf("%d%s", int(t), unit))
	}
	if w.HumOk {
		parts = append(parts, fmt.Sprintf("%d%%", w.Humidity))
	}
	if w.PressOk {
		parts = append(parts, fmt.Sprintf("%.0f hPa", w.Pressure))
	}
	if w.RainOk && w.RainIn > 0 {
		if metric {
			parts = append(parts, fmt.Sprintf("%.1f mm rain", w.RainIn*25.4))
		} else {
			parts = append(parts, fmt.Sprintf("%.2f in rain", w.RainIn))
		}
	}
	out := strings.Join(parts, " \u00b7 ")
	if w.Rest != "" {
		if out == "" {
			out = w.Rest
		} else {
			out += " \u00b7 " + w.Rest
		}
	}
	return out
}
