package tui

// Test scaffolding for the rig loop. Kept in a _test.go file so it is not
// compiled into the production binary.

import (
	"context"

	"github.com/szporwolik/cqops/internal/rig"
)

// fakeRigClient implements RigClient with controllable results.
type fakeRigClient struct {
	status      rig.RigStatus
	err         error
	setFreqErr  error
	lastSetFreq int64
}

func (f *fakeRigClient) Status(ctx context.Context) (rig.RigStatus, error) {
	if f.err != nil {
		return rig.RigStatus{}, f.err
	}
	return f.status, nil
}

func (f *fakeRigClient) Power(ctx context.Context) (float64, error) {
	return f.status.Power, nil
}

func (f *fakeRigClient) SetFrequency(ctx context.Context, freqHz int64) error {
	f.lastSetFreq = freqHz
	return f.setFreqErr
}

func (f *fakeRigClient) SetMode(ctx context.Context, mode string) error {
	return nil
}

func (f *fakeRigClient) GetModes(ctx context.Context) ([]string, error) {
	return []string{"USB", "LSB", "CW-L", "CW-U", "RTTY", "AM", "FM", "DATA-U", "DATA-L"}, nil
}

func (f *fakeRigClient) GetName(ctx context.Context) (string, error) {
	return "FT-DX10", nil
}

// connectedFakeRig returns a fake client with the given frequency, mode, and band.
func connectedFakeRig(freq float64, mode, band string) *fakeRigClient {
	return &fakeRigClient{
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

// Ensure fakeRigClient implements RigClient.
var _ RigClient = (*fakeRigClient)(nil)
