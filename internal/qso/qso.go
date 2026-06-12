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
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

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
