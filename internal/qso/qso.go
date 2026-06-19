package qso

import "time"

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
	MySOTARef       string
	MyPOTARef       string
	MyWWFFRef       string
	CQZone          string // DXCC CQ zone (1-40), enriched from CTY.DAT or QRZ
	ITUZone         string // DXCC ITU zone (1-90), enriched from CTY.DAT or QRZ
	WavelogUploaded string // "" = not attempted, "yes" = uploaded, "no" = failed
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
