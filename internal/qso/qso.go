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
