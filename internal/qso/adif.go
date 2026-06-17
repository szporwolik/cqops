package qso

import (
	"fmt"
	"strings"

	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
)

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
	r := adif.NewRecord()

	set := func(f adifield.Field, v string) {
		if v != "" {
			r[f] = v
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
	set(adifield.MODE, q.Mode)
	set(adifield.SUBMODE, q.Submode)
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
	set(adifield.IOTA, q.IOTA)
	set(adifield.MY_SOTA_REF, q.MySOTARef)
	set(adifield.MY_POTA_REF, q.MyPOTARef)
	set(adifield.MY_WWFF_REF, q.MyWWFFRef)

	return r.String() + "<EOR>"
}
