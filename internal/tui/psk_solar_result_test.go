package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/psk"
	"github.com/szporwolik/cqops/internal/solar"
)

// =============================================================================
// PSK Reporter result-message tests (Pass 14)
// =============================================================================

func TestPSKFetchResult_Success(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.psk.fetching = true
	m.psk.fetched = false

	now := time.Now().UTC()
	msg := pskFetchMsg{
		reports: []psk.Report{
			{ReceiverCallsign: "K1ABC", ReceiverLocator: "FN42", Frequency: 14074000, SNR: 12, Mode: "FT8", FlowStartSeconds: now.Unix()},
			{ReceiverCallsign: "W1AW", ReceiverLocator: "FN31", Frequency: 21074000, SNR: -3, Mode: "FT4", FlowStartSeconds: now.Unix()},
		},
		fetchTime: now,
		err:       nil,
	}

	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("pskFetchMsg should be consumed")
	}
	if m.psk.fetching {
		t.Error("psk.fetching should be false after result")
	}
	if !m.psk.fetched {
		t.Error("psk.fetched should be true after successful fetch")
	}
	if m.psk.lastFetch.IsZero() {
		t.Error("psk.lastFetch should be set")
	}
}

func TestPSKFetchResult_Error(t *testing.T) {
	m := newTestModel()
	m.psk.fetching = true
	m.psk.fetched = true

	msg := pskFetchMsg{
		reports:   nil,
		fetchTime: time.Time{},
		err:       errors.New("connection refused"),
	}

	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("pskFetchMsg with error should be consumed")
	}
	if m.psk.fetching {
		t.Error("psk.fetching should be false after error result")
	}
	if !m.psk.fetched {
		t.Error("psk.fetched should stay true after fetch error")
	}
}

func TestPSKFetchResult_Empty(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.psk.fetching = true
	m.psk.fetched = false

	msg := pskFetchMsg{
		reports:   nil,
		fetchTime: time.Now().UTC(),
		err:       nil,
	}

	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("pskFetchMsg with empty reports should be consumed")
	}
	if m.psk.fetching {
		t.Error("psk.fetching should be false")
	}
	if !m.psk.fetched {
		t.Error("psk.fetched should be true even with empty result")
	}
}

func TestPSKFetchResult_NoPanicOnNil(t *testing.T) {
	// Use lifecycle model so the DB is not nil when handling empty success.
	m := newLifecycleTestModel(t)
	msg := pskFetchMsg{}
	// Should not panic — empty reports → no DB insert attempted beyond empty slice.
	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("zero pskFetchMsg should be consumed without panic")
	}
}

// =============================================================================
// Solar result-message tests (Pass 14)
// =============================================================================

func TestSolarFetchResult_Success(t *testing.T) {
	m := newTestModel()
	m.solar.fetching = true
	m.solar.failed = false

	msg := solarFetchMsg{
		data: &solar.Data{
			FetchedAt: time.Now().UTC(),
			SolarFlux: 150,
			AIndex:    5,
			KIndex:    2.0,
			Updated:   "18 Jun 2026 1200 UTC",
		},
		err:      nil,
		attempts: 1,
	}

	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("solarFetchMsg should be consumed")
	}
	if m.solar.fetching {
		t.Error("solar.fetching should be false after result")
	}
	if m.solar.failed {
		t.Error("solar.failed should be false on success")
	}
	if m.solar.data == nil {
		t.Fatal("solar.data should be set")
	}
	if m.solar.data.SolarFlux != 150 {
		t.Errorf("SolarFlux = %d, want 150", m.solar.data.SolarFlux)
	}
}

func TestSolarFetchResult_Error(t *testing.T) {
	m := newTestModel()
	m.solar.fetching = true
	m.solar.failed = false

	msg := solarFetchMsg{
		data:     nil,
		err:      errors.New("timeout"),
		attempts: 3,
	}

	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("solarFetchMsg with error should be consumed")
	}
	if m.solar.fetching {
		t.Error("solar.fetching should be false after error")
	}
	if !m.solar.failed {
		t.Error("solar.failed should be true after max retries exhausted")
	}
}

func TestSolarFetchResult_ErrorWithRetriesLeft(t *testing.T) {
	m := newTestModel()
	m.solar.fetching = true
	m.solar.failed = false

	msg := solarFetchMsg{
		data:     nil,
		err:      errors.New("timeout"),
		attempts: 1, // more retries available
	}

	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("solarFetchMsg with retries left should be consumed")
	}
	// handleSolarResult sets failed=true on ANY error (retries handled
	// by the fetch command itself, not the result handler).
	if !m.solar.failed {
		t.Error("solar.failed should be true after fetch error")
	}
	if m.solar.fetching {
		t.Error("solar.fetching should be false after result")
	}
}

func TestSolarFetchResult_EmptyDataSafe(t *testing.T) {
	m := newTestModel()
	msg := solarFetchMsg{
		data: &solar.Data{},
		err:  nil,
	}
	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("solarFetchMsg with empty Data should be consumed")
	}
}

func TestSolarFetchResult_NoPanicOnNil(t *testing.T) {
	m := newTestModel()
	msg := solarFetchMsg{}
	consumed, _ := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("zero solarFetchMsg should be consumed without panic")
	}
}
