package rig

import "context"

type Rig interface {
	Status(ctx context.Context) (RigStatus, error)
	Power(ctx context.Context) (float64, error)
}

type RigStatus struct {
	Provider       string
	Connected      bool
	FrequencyHz    int64
	FrequencyMHz   float64
	FrequencyRxHz  int64 // VFO B frequency when split (0 = not available)
	FrequencyRxMHz float64
	Split          bool // true when radio is in split mode
	Band           string
	Mode           string
	RawMode        string
	Power          float64
}
