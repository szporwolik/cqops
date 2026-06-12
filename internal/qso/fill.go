package qso

type FillSource struct {
	Call            string
	Band            string
	Freq            float64
	Mode            string
	Submode         string
	RSTSent         string
	RSTRcvd         string
	GridSquare      string
	Name            string
	QTH             string
	Comment         string
	StationCallsign string
	Operator        string
	MyGridSquare    string
	MyRig           string
	MyAntenna       string
}

func Fill(q *QSO, rig RigProvider, station FillSource) {
	fromRig := false
	if rig != nil {
		status, err := rig.Status(nil)
		if err == nil && status.Connected {
			fromRig = true
		}
	}

	if q.StationCallsign == "" && station.StationCallsign != "" {
		q.StationCallsign = station.StationCallsign
	}
	if q.Operator == "" && station.Operator != "" {
		q.Operator = station.Operator
	}
	if q.MyGridSquare == "" && station.MyGridSquare != "" {
		q.MyGridSquare = station.MyGridSquare
	}
	if q.MyRig == "" && station.MyRig != "" {
		q.MyRig = station.MyRig
	}
	if q.MyAntenna == "" && station.MyAntenna != "" {
		q.MyAntenna = station.MyAntenna
	}

	if fromRig {
		status, _ := rig.Status(nil)
		if q.Freq == 0 && status.FrequencyMHz > 0 {
			q.Freq = status.FrequencyMHz
		}
		if q.Band == "" && status.Band != "" {
			q.Band = status.Band
		}
		if q.Mode == "" && status.Mode != "" {
			q.Mode = status.Mode
		}
	}

	if q.Band == "" && q.Freq > 0 {
		q.Band = DeriveBand(q.Freq)
	}
}

type RigProvider interface {
	Status(ctx interface{}) (RigStatus, error)
}

type RigStatus struct {
	Provider      string
	Connected     bool
	FrequencyHz   int64
	FrequencyMHz  float64
	Band          string
	Mode          string
	RawMode       string
}
