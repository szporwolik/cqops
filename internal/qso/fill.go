package qso

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
