package qso

import (
	"strconv"
	"strings"
)

type StationInfo struct {
	StationCallsign string
	Operator        string
	MyGridSquare    string
	MyRig           string
	MyAntenna       string
	TXPower         string
	MySOTARef       string
	MyPOTARef       string
	MyWWFFRef       string
	MyCQZone        string
	MyITUZone       string
	MyDXCC          string
	MySIG           string
	MySIGInfo       string
}

// FillSSBSubmode gives an SSB contact its sideband when nothing set one:
// LSB below 10 MHz, USB above — the convention every logger stores and the
// same rule the QSO form applies when the frequency is entered. Rows that
// arrive without a submode (ADIF imports that only carry MODE=SSB, spot
// fills, hand-edited rows) would otherwise show "Sub —" next to an SSB
// contact that plainly has a sideband. Anything explicit is left alone.
func FillSSBSubmode(q *QSO) {
	if q == nil || strings.TrimSpace(q.Submode) != "" {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(q.Mode), "SSB") {
		return
	}
	if q.Freq <= 0 {
		return
	}
	if q.Freq < 10 {
		q.Submode = "LSB"
		return
	}
	q.Submode = "USB"
}

func ApplyStationDefaults(q *QSO, s StationInfo) {
	if q.StationCallsign == "" && s.StationCallsign != "" {
		q.StationCallsign = s.StationCallsign
	}
	if q.Operator == "" && s.Operator != "" {
		q.Operator = s.Operator
	}
	if q.MyGridSquare == "" && s.MyGridSquare != "" {
		q.MyGridSquare = s.MyGridSquare
	}
	if q.MyRig == "" && s.MyRig != "" {
		q.MyRig = s.MyRig
	}
	if q.MyAntenna == "" && s.MyAntenna != "" {
		q.MyAntenna = s.MyAntenna
	}
	if q.TXPower == "" && s.TXPower != "" {
		q.TXPower = s.TXPower
	}
	if q.MySOTARef == "" && s.MySOTARef != "" {
		q.MySOTARef = s.MySOTARef
	}
	if q.MyPOTARef == "" && s.MyPOTARef != "" {
		q.MyPOTARef = s.MyPOTARef
	}
	if q.MyWWFFRef == "" && s.MyWWFFRef != "" {
		q.MyWWFFRef = s.MyWWFFRef
	}
	if q.MyCQZone == "" && s.MyCQZone != "" {
		q.MyCQZone = s.MyCQZone
	}
	if q.MyITUZone == "" && s.MyITUZone != "" {
		q.MyITUZone = s.MyITUZone
	}
	if q.MyDXCC == "" && s.MyDXCC != "" {
		q.MyDXCC = s.MyDXCC
	}
	if q.MySIG == "" && s.MySIG != "" {
		q.MySIG = s.MySIG
	}
	if q.MySIGInfo == "" && s.MySIGInfo != "" {
		q.MySIGInfo = s.MySIGInfo
	}

	if q.Band == "" && q.Freq > 0 {
		q.Band = DeriveBand(q.Freq)
	}
	if q.Band != "" {
		q.Band = NormalizeBand(q.Band)
	}

	if q.Mode != "" {
		q.Mode, q.Submode = NormalizeMode(q.Mode, q.Submode)
	}
}

// ItoaOrEmpty returns the decimal string of n, or "" if n is zero.
func ItoaOrEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}
