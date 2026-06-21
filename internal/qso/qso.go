package qso

import (
	"strings"
	"time"
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
	MyCQZone        string // station CQ zone (1-40), from logbook station config
	MyITUZone       string // station ITU zone (1-90), from logbook station config
	MyDXCC          string // station DXCC entity number, from logbook station config
	MySIG           string // station Special Interest Group (e.g. "SOTA", "POTA")
	MySIGInfo       string // station Special Interest Group info (e.g. summit/park ref)
	WavelogUploaded string // "" = not attempted, "yes" = uploaded, "no" = failed
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

// NormalizeExchange re-derives STX, SRX, STXString, and SRXString from
// ExchSent and ExchRcvd. Call this after reading exchange fields from the
// logbook editor to keep them consistent with the main QSO form save path.
func (q *QSO) NormalizeExchange() {
	q.STXString = StripRSTPrefix(q.ExchSent, q.RSTSent)
	q.SRXString = StripRSTPrefix(q.ExchRcvd, q.RSTRcvd)
	q.STX = ParseSerial(q.ExchSent)
	q.SRX = ParseSerial(q.ExchRcvd)
}

// ParseSerial extracts the last integer from a contest exchange string.
// Returns 0 if no integer is found. In contest exchanges, the sequence
// number is the trailing number (e.g. "599 001" → 001, "5NN 042" → 42).
func ParseSerial(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Find the last integer by scanning from the end.
	end := len(s)
	// Skip trailing non-digits.
	for end > 0 && (s[end-1] < '0' || s[end-1] > '9') {
		end--
	}
	if end == 0 {
		return 0
	}
	// Find start of the last number.
	start := end
	for start > 0 && s[start-1] >= '0' && s[start-1] <= '9' {
		start--
	}
	n := 0
	for i := start; i < end; i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
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
