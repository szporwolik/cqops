package qso

import (
	"strconv"
	"strings"
	"time"
)

// DXCC resolution provenance, stored in QSO.DXCCSource. It records HOW the
// entity number was obtained so a future reconfirmation or re-resolution can
// tell a provider value from a guess.
const (
	DXCCSourceExplicit = "explicit" // carried by the record or entered locally
	DXCCSourceImported = "imported" // supplied by an ADIF file / remote logbook
	DXCCSourcePrefix   = "prefix"   // deterministic callsign-prefix lookup (CTY)
	DXCCSourceCallbook = "callbook" // callbook (QRZ/HamQTH/…) result
	DXCCSourceUnknown  = "unknown"  // still unresolved
)

type QSO struct {
	ID              int64
	Call            string
	QSODate         string
	TimeOn          string
	TimeOff         string
	Band            string
	Freq            float64
	FreqRx          float64
	Mode            string
	Submode         string
	RSTSent         string
	RSTRcvd         string
	GridSquare      string
	Name            string
	QTH             string
	Country         string
	Comment         string
	Notes           string
	TXPower         string
	StationCallsign string
	Operator        string
	MyGridSquare    string
	MyRig           string
	MyAntenna       string
	Source          string
	Distance        float64
	Bearing         float64
	SOTARef         string
	POTARef         string
	WWFFRef         string
	IOTA            string
	SIG             string // Special Interest Group name (e.g. "SOTA", "POTA")
	SIGInfo         string // Special Interest Group info (e.g. summit/park reference)
	ExchSent        string // contest exchange sent (e.g. "599 001")
	ExchRcvd        string // contest exchange received
	MySOTARef       string
	MyPOTARef       string
	MyWWFFRef       string
	STX             int    // contest QSO transmitted serial number
	SRX             int    // contest QSO received serial number
	STXString       string // contest QSO transmitted exchange
	SRXString       string // contest QSO received exchange
	CQZone          string // DXCC CQ zone (1-40), enriched from CTY.DAT or QRZ
	ITUZone         string // DXCC ITU zone (1-90), enriched from CTY.DAT or QRZ
	DXCC            string // DXCC entity number (e.g. "269" for Poland), from QRZ
	DXCCSource      string // how DXCC was resolved (DXCCSource* constant)
	// Confirmation status as ADIF defines it: "Y" received, "N" not received,
	// "" not provided/unknown. Empty is preserved on update, so a caller that
	// does not manage confirmations can never wipe a stored one.
	QSLRcvd         string // paper QSL received (QSL_RCVD)
	LOTWQSLRcvd     string // LoTW confirmation (LOTW_QSL_RCVD)
	EQSLQSLRcvd     string // eQSL confirmation (EQSL_QSL_RCVD)
	MyCQZone        string // station CQ zone (1-40), from logbook station config
	MyITUZone       string // station ITU zone (1-90), from logbook station config
	MyDXCC          string // station DXCC entity number, from logbook station config
	MySIG           string // station Special Interest Group (e.g. "SOTA", "POTA")
	MySIGInfo       string // station Special Interest Group info (e.g. summit/park ref)
	WavelogID       int64  // remote QSO id from Wavelog (0 = not in Wavelog); the single source of truth for uploaded state
	WavelogDirty    bool   // local row diverged from the Wavelog copy (PATCH deferred or failed) — remote refreshes must not overwrite it
	WavelogDirtyRev int64  // pending-sync revision (bumped on every SaveQSOForSync); acks clear dirty only for their exact revision
	ContestID       string // internal contest hash (e.g. "7e0644bc0522"), for DB filtering
	ContestADIFID   string // ADIF Contest ID (e.g. "CQ-WPX-CW"), for ADIF export
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewQSO returns a QSO pre-filled with the current UTC date/time and
// source set to "manual".
func NewQSO() *QSO {
	now := time.Now().UTC()
	return &QSO{
		QSODate:   now.Format("20060102"),
		TimeOn:    now.Format("150405"),
		Source:    "manual",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ADIFTime combines an ADIF QSO_DATE (YYYYMMDD) and TIME_ON (HHMMSS, HHMM or
// empty) into a UTC time. It reports false when the date is unusable, so the
// caller can fall back to another timestamp (created_at).
//
// Separators are tolerated like StripNonDigits tolerates them, but the common
// case — an all-digit date and time — is parsed in place without allocating.
// That matters where this runs once per row over a whole log (the contest rate
// panel, ADIF import validation): time.Parse would allocate for every contact.
func ADIFTime(date, timeOn string) (time.Time, bool) {
	d := date
	if len(d) != 8 || !isAllDigits(d) {
		d = StripNonDigits(d) // tolerates "2026-05-01" and friends
		if len(d) != 8 {
			return time.Time{}, false
		}
	}
	t := timeOn
	if len(t) != 6 || !isAllDigits(t) {
		t = StripNonDigits(t)
		if t == "" {
			t = "000000" // ADIF allows a contact without TIME_ON
		} else if len(t) < 6 {
			t += "000000"[:6-len(t)] // HHMM is a valid ADIF TIME_ON
		}
		if len(t) != 6 {
			return time.Time{}, false
		}
	}
	year, month, day := num(d[0:4]), num(d[4:6]), num(d[6:8])
	hour, min, sec := num(t[0:2]), num(t[2:4]), num(t[4:6])
	// Check every field before time.Date: it normalises out-of-range values
	// instead of rejecting them, so a bogus TIME_ON would silently become a
	// different moment (12:60:00 → 13:00:00, 12:00:99 → 12:01:39) and feed an
	// invented time into the rate, peak and session numbers. The old
	// time.Parse-based parser rejected these and fell back to created_at; this
	// keeps that behaviour.
	if month < 1 || month > 12 || day < 1 || day > 31 ||
		hour > 23 || min > 59 || sec > 59 {
		return time.Time{}, false
	}
	out := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
	// The remaining normalisation is the calendar itself (Feb 30 becomes Mar 2),
	// so the date round-trip is what rejects a malformed day.
	if out.Year() != year || int(out.Month()) != month || out.Day() != day {
		return time.Time{}, false
	}
	return out, true
}

// num parses a short run of digits that isAllDigits has already validated.
func num(s string) int {
	v := 0
	for i := 0; i < len(s); i++ {
		v = v*10 + int(s[i]-'0')
	}
	return v
}

// Confirmed reports whether any confirmation was received for the contact
// (paper QSL, LoTW or eQSL).
func (q *QSO) Confirmed() bool {
	return IsQSLReceived(q.QSLRcvd) || IsQSLReceived(q.LOTWQSLRcvd) || IsQSLReceived(q.EQSLQSLRcvd)
}

// IsQSLReceived reports whether an ADIF QSL-received status means "received".
// ADIF allows Y (yes) and V (verified, import-only).
func IsQSLReceived(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "Y", "V":
		return true
	}
	return false
}

// NormalizeQSLStatus maps an ADIF QSL-received status to the value CQOps
// stores: "Y", "N" or "" (not provided). Anything else is treated as unknown
// rather than inventing a confirmation.
func NormalizeQSLStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "Y", "V":
		return "Y"
	case "N":
		return "N"
	}
	return ""
}

// NormalizeExchange re-derives STX, SRX, STXString, and SRXString from
// ExchSent and ExchRcvd. Call this after reading exchange fields from the
// logbook editor to keep them consistent with the main QSO form save path.
func (q *QSO) NormalizeExchange() {
	q.STXString = StripRSTPrefix(q.ExchSent, q.RSTSent)
	q.SRXString = StripRSTPrefix(q.ExchRcvd, q.RSTRcvd)
	q.STX = SerialFromExchange(q.ExchSent, q.RSTSent)
	q.SRX = SerialFromExchange(q.ExchRcvd, q.RSTRcvd)
}

// SerialFromExchange derives the contest serial from an exchange as typed or
// imported ("59 001" → 1, "5NN 042" → 42).
//
// Two rules keep it from inventing a serial out of unrelated digits, which an
// ordinary save would then push to Wavelog:
//   - the RST prefix is removed first, so the report can never become the
//     serial ("55 MA" is a state exchange, not serial 55);
//   - only a standalone number counts, so digits inside a state, grid or code
//     are ignored ("JO90" → 0, not 90) — and a bare report left in front of a
//     non-numeric exchange is not a serial either.
//
// Returns 0 when the exchange carries no serial at all.
func SerialFromExchange(exchange, rst string) int {
	tokens := strings.FieldsFunc(StripRSTPrefix(exchange, rst), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '/' || r == ',' || r == ';'
	})
	idx := -1
	for i, tok := range tokens {
		if isAllDigits(tok) {
			idx = i // the trailing number is the serial ("599 001" → 1)
		}
	}
	if idx < 0 {
		return 0
	}
	// The RST is a form field and may be empty, leaving the report in the text.
	// A report in front of a non-numeric exchange carries no serial at all.
	if idx == 0 && len(tokens) > 1 && looksLikeReport(tokens[0]) {
		return 0
	}
	n, err := strconv.Atoi(tokens[idx])
	if err != nil {
		return 0 // more digits than a serial can hold
	}
	return n
}

// looksLikeReport reports whether tok could be an RST value (55, 57, 59, 599).
func looksLikeReport(tok string) bool {
	return len(tok) <= 3 && tok[0] >= '5' && tok[0] <= '9'
}

// isAllDigits reports whether s is non-empty and holds nothing but ASCII
// digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// StripRSTPrefix removes the leading RST value from an exchange string.
// For example, StripRSTPrefix("599 001 KO00", "599") returns "001 KO00".
// StripRSTPrefix("599 001 KO00", "59") also returns "001 KO00" — it
// handles the common case where RST "59" prefixes the longer "599".
// If the RST is empty or the exchange is empty, returns exchange unchanged.
func StripRSTPrefix(exchange, rst string) string {
	exchange = strings.TrimSpace(exchange)
	rst = strings.TrimSpace(rst)
	if rst == "" || exchange == "" {
		return exchange
	}
	upper := strings.ToUpper(exchange)
	upperRST := strings.ToUpper(rst)
	// Try exact RST match first.
	if strings.HasPrefix(upper, upperRST) {
		rest := strings.TrimSpace(exchange[len(rst):])
		// If the rest starts with a digit, the RST we stripped was too short
		// (e.g. RST="59" but exchange="599 001"). Try one more digit.
		if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			longer := rst + string(rest[0])
			if strings.HasPrefix(upper, strings.ToUpper(longer)) {
				return strings.TrimSpace(exchange[len(longer):])
			}
		}
		return rest
	}
	return exchange
}
