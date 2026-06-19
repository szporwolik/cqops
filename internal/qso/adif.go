package qso

import (
	"fmt"
	"strings"
	"unicode"

	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
	"golang.org/x/text/unicode/norm"
)

// =============================================================================
// ASCII sanitization — ADIF requires 7-bit ASCII in all fields.
// =============================================================================

// multiCharMappings maps Unicode characters that expand to multiple ASCII
// characters. These must be handled before NFD normalization because they
// don't decompose into base+combining form.
var multiCharMappings = strings.NewReplacer(
	"ß", "ss", "Æ", "AE", "æ", "ae",
	"Œ", "OE", "œ", "oe",
	"Ĳ", "IJ", "ĳ", "ij",
	"Þ", "TH", "þ", "th",
	"Đ", "DJ", "đ", "dj",
)

// sanitizeASCII converts s to ASCII using Unicode NFD normalization:
//  1. Expand multi-character ligatures (ß→ss, æ→ae, etc.)
//  2. NFD decompose (é → e + combining acute)
//  3. Strip non-spacing marks (accents, cedillas, ogoneks, etc.)
//
// This handles virtually all Latin-script languages (German, French, Spanish,
// Portuguese, Polish, Czech, Turkish, Nordic languages, etc.) correctly.
func sanitizeASCII(s string) string {
	// Fast path: already ASCII.
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}

	// Step 1: expand multi-character ligatures.
	s = multiCharMappings.Replace(s)

	// Step 2: NFD normalization decomposes base + combining mark.
	s = norm.NFD.String(s)

	// Step 3: strip non-spacing marks (category Mn).
	// Also strip any remaining non-ASCII characters.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r <= 127 {
			b.WriteRune(r)
		} else if unicode.Is(unicode.Mn, r) {
			// Non-spacing mark — drop it (already stripped base from NFD).
		} else {
			// Non-decomposable non-ASCII character (e.g. ł, Ł) —
			// map to closest ASCII via small lookup, or drop.
			if sub, ok := stubbornChars[r]; ok {
				b.WriteString(sub)
			}
			// Otherwise drop.
		}
	}
	return b.String()
}

// stubbornChars maps characters that don't decompose under NFD (e.g. stroke
// letters like ł, Ł, ø) to their closest ASCII equivalents.
var stubbornChars = map[rune]string{
	'ł': "l", 'Ł': "L",
	'ø': "o", 'Ø': "O",
	'ð': "d", 'Ð': "D", // eth — NFD does not decompose
	'đ': "dj", 'Đ': "DJ", // d with stroke (Croatian)
	'ı': "i", // Turkish dotless i — NFD does not decompose
}

// =============================================================================
// ADIF encoding
// =============================================================================

// isValidIOTA returns true if s looks like a valid IOTA reference.
// Valid format: continent code, hyphen, digits (e.g. "EU-005", "eu-005").
// Values like "BLANK", "NONE", "NULL", "16", etc. are rejected.
// Case-insensitive per ADIF spec.
func isValidIOTA(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Must contain a hyphen.
	idx := strings.IndexByte(s, '-')
	if idx < 1 || idx > len(s)-2 {
		return false
	}
	// Before hyphen: must be 1-3 letters (continent code, case-insensitive).
	for _, r := range s[:idx] {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	// After hyphen: must be digits and possibly letters (e.g. "005", "001S").
	rest := s[idx+1:]
	if len(rest) < 2 || len(rest) > 6 {
		return false
	}
	for _, r := range rest {
		if r >= '0' && r <= '9' {
			continue
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		return false
	}
	return true
}

// ToADIF returns the QSO as an ADIF string suitable for Wavelog upload.
func (q *QSO) ToADIF() string {
	return q.toADIFWithStation(q.StationCallsign)
}

// ToADIFWithStation returns the QSO as an ADIF string, overriding the
// station callsign (needed when uploading to a Wavelog station whose
// callsign differs from the operator's callsign recorded in the QSO).
func (q *QSO) ToADIFWithStation(stationCall string) string {
	return q.toADIFWithStation(stationCall)
}

func (q *QSO) toADIFWithStation(stationCall string) string {
	// Fix legacy data: FT8/FT2 are standalone modes, not MFSK submodes.
	// QSOs stored before this fix may have MODE=MFSK, SUBMODE=FT8 or FT2.
	mode, submode := q.Mode, q.Submode
	if strings.EqualFold(mode, "MFSK") && (strings.EqualFold(submode, "FT8") || strings.EqualFold(submode, "FT2")) {
		mode = strings.ToUpper(submode)
		submode = ""
	}

	r := adif.NewRecord()

	set := func(f adifield.Field, v string) {
		if v != "" {
			r[f] = sanitizeASCII(v)
		}
	}
	setf := func(f adifield.Field, v float64) {
		if v != 0 {
			r[f] = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", v), "0"), ".")
		}
	}

	set(adifield.CALL, q.Call)
	set(adifield.QSO_DATE, q.QSODate)
	set(adifield.TIME_ON, q.TimeOn)
	set(adifield.TIME_OFF, q.TimeOff)
	set(adifield.BAND, q.Band)
	setf(adifield.FREQ, q.Freq)
	setf(adifield.FREQ_RX, q.FreqRx)
	set(adifield.MODE, mode)
	set(adifield.SUBMODE, submode)
	set(adifield.RST_SENT, q.RSTSent)
	set(adifield.RST_RCVD, q.RSTRcvd)
	set(adifield.GRIDSQUARE, q.GridSquare)
	set(adifield.NAME, q.Name)
	set(adifield.QTH, q.QTH)
	set(adifield.COUNTRY, q.Country)
	set(adifield.COMMENT, q.Comment)
	set(adifield.NOTES, q.Notes)
	set(adifield.TX_PWR, q.TXPower)
	set(adifield.STATION_CALLSIGN, stationCall)
	set(adifield.OPERATOR, q.Operator)
	set(adifield.MY_GRIDSQUARE, q.MyGridSquare)
	set(adifield.MY_RIG, q.MyRig)
	set(adifield.MY_ANTENNA, q.MyAntenna)
	set(adifield.SOTA_REF, q.SOTARef)
	set(adifield.POTA_REF, q.POTARef)
	set(adifield.WWFF_REF, q.WWFFRef)
	// Only write IOTA if it looks like a valid IOTA reference (e.g. "EU-005").
	if isValidIOTA(q.IOTA) {
		set(adifield.IOTA, q.IOTA)
	}
	set(adifield.MY_SOTA_REF, q.MySOTARef)
	set(adifield.MY_POTA_REF, q.MyPOTARef)
	set(adifield.MY_WWFF_REF, q.MyWWFFRef)
	setf(adifield.DISTANCE, q.Distance)

	return r.String() + "<EOR>"
}

// ParseADIFRecord converts an ADIF record into a QSO.
// Used by both Wavelog download and local ADIF import to avoid duplication.
func ParseADIFRecord(r adif.Record, source string) *QSO {
	qs := NewQSO()

	get := func(f adifield.Field) string { return strings.TrimSpace(r[f]) }
	getFloat := func(f adifield.Field) float64 {
		var v float64
		if s := get(f); s != "" {
			fmt.Sscanf(s, "%f", &v)
		}
		return v
	}

	if v := get(adifield.CALL); v != "" {
		qs.Call = strings.ToUpper(v)
	}
	if v := get(adifield.BAND); v != "" {
		qs.Band = NormalizeBand(v)
	}
	if v := get(adifield.MODE); v != "" {
		qs.Mode = strings.ToUpper(v)
	}
	if v := get(adifield.SUBMODE); v != "" {
		qs.Submode = strings.ToUpper(v)
	}
	// Strip non-digit chars from date/time (WSJT-X includes punctuation).
	qs.QSODate = stripNonDigits(get(adifield.QSO_DATE))
	qs.TimeOn = stripNonDigits(get(adifield.TIME_ON))
	qs.TimeOff = stripNonDigits(get(adifield.TIME_OFF))
	qs.Freq = getFloat(adifield.FREQ)
	qs.FreqRx = getFloat(adifield.FREQ_RX)
	qs.RSTSent = get(adifield.RST_SENT)
	qs.RSTRcvd = get(adifield.RST_RCVD)
	qs.GridSquare = NormalizeLocator(get(adifield.GRIDSQUARE))
	qs.Name = get(adifield.NAME)
	qs.QTH = get(adifield.QTH)
	qs.Country = get(adifield.COUNTRY)
	// DXCC field may contain the entity name as a fallback for Country.
	if qs.Country == "" {
		qs.Country = get(adifield.DXCC)
	}
	qs.Comment = get(adifield.COMMENT)
	qs.Notes = get(adifield.NOTES)
	qs.TXPower = get(adifield.TX_PWR)
	qs.SOTARef = get(adifield.SOTA_REF)
	qs.POTARef = get(adifield.POTA_REF)
	qs.WWFFRef = get(adifield.WWFF_REF)
	qs.IOTA = get(adifield.IOTA)
	// Clear clearly invalid IOTA values so they don't pollute the DB.
	if qs.IOTA != "" && !isValidIOTA(qs.IOTA) {
		qs.IOTA = ""
	}
	qs.MySOTARef = get(adifield.MY_SOTA_REF)
	qs.MyPOTARef = get(adifield.MY_POTA_REF)
	qs.MyWWFFRef = get(adifield.MY_WWFF_REF)
	if v := get(adifield.STATION_CALLSIGN); v != "" {
		qs.StationCallsign = strings.ToUpper(v)
	}
	if v := get(adifield.OPERATOR); v != "" {
		qs.Operator = strings.ToUpper(v)
	}
	qs.MyGridSquare = NormalizeLocator(get(adifield.MY_GRIDSQUARE))
	qs.MyRig = get(adifield.MY_RIG)
	qs.MyAntenna = get(adifield.MY_ANTENNA)
	qs.Distance = getFloat(adifield.DISTANCE)
	qs.Source = source

	return qs
}

// stripNonDigits removes all non-digit characters from s.
func stripNonDigits(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
