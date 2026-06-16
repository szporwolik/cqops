package tui

import (
	"context"
	"errors"

	"github.com/szporwolik/cqops/internal/rig"
)

// =============================================================================
// Flrig client interface — allows testing without a live flrig server.
// =============================================================================

// FlrigClient abstracts the flrig HTTP client for testability.
// The production implementation is flrig.Client.
type FlrigClient interface {
	Status(ctx context.Context) (rig.RigStatus, error)
}

// =============================================================================
// Fake flrig client for tests.
// =============================================================================

// fakeFlrigClient implements FlrigClient with controllable results.
type fakeFlrigClient struct {
	status rig.RigStatus
	err    error
}

func (f *fakeFlrigClient) Status(ctx context.Context) (rig.RigStatus, error) {
	if f.err != nil {
		return rig.RigStatus{}, f.err
	}
	return f.status, nil
}

// disconnectedFakeFlrig returns a fake client that reports not connected.
func disconnectedFakeFlrig() *fakeFlrigClient {
	return &fakeFlrigClient{
		status: rig.RigStatus{Provider: "flrig", Connected: false},
	}
}

// connectedFakeFlrig returns a fake client with the given frequency, mode, and band.
func connectedFakeFlrig(freq float64, mode, band string) *fakeFlrigClient {
	return &fakeFlrigClient{
		status: rig.RigStatus{
			Provider:     "flrig",
			Connected:    true,
			FrequencyMHz: freq,
			Mode:         mode,
			Band:         band,
			Power:        50,
			PTT:          false,
		},
	}
}

// transmittingFakeFlrig returns a fake client with PTT active.
func transmittingFakeFlrig(freq float64, mode, band string) *fakeFlrigClient {
	return &fakeFlrigClient{
		status: rig.RigStatus{
			Provider:     "flrig",
			Connected:    true,
			FrequencyMHz: freq,
			Mode:         mode,
			Band:         band,
			Power:        50,
			PTT:          true,
		},
	}
}

// errorFakeFlrig returns a fake client that always returns an error.
func errorFakeFlrig() *fakeFlrigClient {
	return &fakeFlrigClient{err: errors.New("connection refused")}
}

// Ensure fakeFlrigClient implements FlrigClient.
var _ FlrigClient = (*fakeFlrigClient)(nil)
