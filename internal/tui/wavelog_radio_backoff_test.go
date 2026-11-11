package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/wavelog"
)

// TestRadioPushBackoff pins the schedule. A degraded Wavelog answered 60
// consecutive timeouts in 46 minutes — one attempt every 15 s, each blocking for
// the whole client deadline, for data that is only informational.
func TestRadioPushBackoff(t *testing.T) {
	cases := []struct {
		fails int
		want  time.Duration
	}{
		{0, 15 * time.Second},
		{1, 15 * time.Second},
		{2, 30 * time.Second},
		{3, 60 * time.Second},
		{4, 120 * time.Second},
		{5, 240 * time.Second},
		{6, 5 * time.Minute},
		{50, 5 * time.Minute},
	}
	for _, tc := range cases {
		if got := wlRadioPushBackoff(tc.fails); got != tc.want {
			t.Errorf("after %d failures the interval = %s, want %s", tc.fails, got, tc.want)
		}
	}
}

// TestRadioPushFailuresDriveTheBackoff: failures accumulate (so the interval
// grows) and a single success puts the push straight back on the normal cadence.
func TestRadioPushFailuresDriveTheBackoff(t *testing.T) {
	m := newLifecycleTestModel(t)

	fail := func() {
		if !m.handleWavelogRadioMsg(wlRadioPushMsg{err: errors.New("no answer from Wavelog within 5s")}) {
			t.Fatal("a push failure was not handled")
		}
	}

	fail()
	if m.lookup.wlRadioFails != 1 {
		t.Fatalf("failures = %d after one failure, want 1", m.lookup.wlRadioFails)
	}
	if got := wlRadioPushBackoff(m.lookup.wlRadioFails); got != wlRadioPushInterval {
		t.Errorf("interval after the first failure = %s, want the normal %s", got, wlRadioPushInterval)
	}

	fail()
	fail()
	if got := wlRadioPushBackoff(m.lookup.wlRadioFails); got != 60*time.Second {
		t.Errorf("interval after %d failures = %s, want 60s", m.lookup.wlRadioFails, got)
	}

	m.handleWavelogRadioMsg(wlRadioPushMsg{radio: &wavelog.Radio{ID: 7}})
	if m.lookup.wlRadioFails != 0 {
		t.Errorf("a success left %d failures behind — the backoff would never reset", m.lookup.wlRadioFails)
	}
	if m.lookup.wlRadioID != 7 {
		t.Errorf("radio id = %d after a successful push, want 7", m.lookup.wlRadioID)
	}
	if got := wlRadioPushBackoff(m.lookup.wlRadioFails); got != wlRadioPushInterval {
		t.Errorf("interval after recovery = %s, want %s", got, wlRadioPushInterval)
	}
}
