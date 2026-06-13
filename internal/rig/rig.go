package rig

import "context"

type Rig interface {
	Status(ctx context.Context) (RigStatus, error)
}

type RigStatus struct {
	Provider     string
	Connected    bool
	FrequencyHz  int64
	FrequencyMHz float64
	Band         string
	Mode         string
	RawMode      string
	Power        float64
}
