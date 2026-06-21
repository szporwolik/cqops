package tui

import (
	"context"

	"github.com/szporwolik/cqops/internal/rig"
)

// =============================================================================
// Flrig client interface — allows testing without a live flrig server.
// =============================================================================

// FlrigClient abstracts the flrig HTTP client for testability.
// The production implementation is flrig.Client.
type FlrigClient interface {
	Status(ctx context.Context) (rig.RigStatus, error)
	SetFrequency(ctx context.Context, freqHz int64) error
	GetModes(ctx context.Context) ([]string, error)
	SetMode(ctx context.Context, mode string) error
	GetName(ctx context.Context) (string, error)
}

// =============================================================================
// Fake flrig client for tests.
// =============================================================================

// fakeFlrigClient implements FlrigClient with controllable results.
type fakeFlrigClient struct {
	status      rig.RigStatus
	err         error
	setFreqErr  error
	lastSetFreq int64
}

func (f *fakeFlrigClient) Status(ctx context.Context) (rig.RigStatus, error) {
	if f.err != nil {
		return rig.RigStatus{}, f.err
	}
	return f.status, nil
}

func (f *fakeFlrigClient) SetFrequency(ctx context.Context, freqHz int64) error {
	f.lastSetFreq = freqHz
	return f.setFreqErr
}

func (f *fakeFlrigClient) SetMode(ctx context.Context, mode string) error {
	return nil
}

func (f *fakeFlrigClient) GetModes(ctx context.Context) ([]string, error) {
	return []string{"USB", "LSB", "CW-L", "CW-U", "RTTY", "AM", "FM", "DATA-U", "DATA-L"}, nil
}

func (f *fakeFlrigClient) GetName(ctx context.Context) (string, error) {
	return "FT-DX10", nil
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
		},
	}
}

// Ensure fakeFlrigClient implements FlrigClient.
var _ FlrigClient = (*fakeFlrigClient)(nil)
