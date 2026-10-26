package tui

import (
	"context"

	"github.com/szporwolik/cqops/internal/rig"
	"github.com/szporwolik/cqops/internal/rotor"
)

// =============================================================================
// Rig client interface — allows testing without a live rig server.
// Both flrig and hamlib backends implement this interface.
// =============================================================================

// RigClient abstracts the rig control backend (flrig HTTP or hamlib TCP).
type RigClient interface {
	Status(ctx context.Context) (rig.RigStatus, error)
	Power(ctx context.Context) (float64, error)
	SetFrequency(ctx context.Context, freqHz int64) error
	GetModes(ctx context.Context) ([]string, error)
	SetMode(ctx context.Context, mode string) error
	GetName(ctx context.Context) (string, error)
}

// RotorClient abstracts a rotor control backend (hamlib rotctld TCP).
type RotorClient interface {
	Status(ctx context.Context) (rotor.Status, error)
	SetPosition(ctx context.Context, az, el float64) error
	Stop(ctx context.Context) error
	GetName(ctx context.Context) (string, error)
}
