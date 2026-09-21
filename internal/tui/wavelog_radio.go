package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// wlRadioPushInterval is how often the live rig state (freq/mode/power from
// the QSO form) is pushed to the Wavelog radio resource.
const wlRadioPushInterval = 15 * time.Second

type wlRadioEnsureMsg struct {
	radio *wavelog.Radio
	err   error
}

type wlRadioPushMsg struct {
	radio   *wavelog.Radio
	err     error
	retried bool // true when the push recreated the radio first
}

// ensureWavelogRadioCmd makes sure the "CQOps" radio exists on Wavelog and
// stores its id. Dispatched after a successful connection check.
func (m *Model) ensureWavelogRadioCmd() tea.Cmd {
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled {
		return nil
	}
	url, key := wl.URL, wl.APIKey
	return func() tea.Msg {
		r, err := wavelog.EnsureRadio(url, key, wavelog.DefaultRadioName)
		if err != nil {
			return wlRadioEnsureMsg{err: err}
		}
		return wlRadioEnsureMsg{radio: r}
	}
}

// radioStateFromForm reads the live rig state from the QSO form fields.
// ok is false when there is nothing useful to push (no frequency/mode).
func (m *Model) radioStateFromForm() (wavelog.RadioState, bool) {
	freqMHz, err := strconv.ParseFloat(strings.TrimSpace(m.fields[fieldFreq].Value()), 64)
	if err != nil || freqMHz <= 0 {
		return wavelog.RadioState{}, false
	}
	mode := qso.NormalizeRigMode(strings.TrimSpace(m.fields[fieldMode].Value()))
	if mode == "" {
		return wavelog.RadioState{}, false
	}
	st := wavelog.RadioState{
		FrequencyHz: int64(math.Round(freqMHz * 1e6)),
		Mode:        mode,
	}
	if rx := strings.TrimSpace(m.fields[fieldFreqRx].Value()); rx != "" {
		if rxMHz, rerr := strconv.ParseFloat(rx, 64); rerr == nil && rxMHz > 0 {
			st.FrequencyRxHz = int64(math.Round(rxMHz * 1e6))
		}
	}
	if pw := strings.TrimSpace(m.fields[fieldTXPower].Value()); pw != "" {
		if watts, werr := strconv.Atoi(pw); werr == nil && watts > 0 {
			st.PowerWatts = int64(watts)
		}
	}
	return st, true
}

// pushWavelogRadioCmd pushes the current form state to the CQOps radio on
// Wavelog. When the radio no longer exists (deleted elsewhere), it is
// recreated and the push retried once.
func (m *Model) pushWavelogRadioCmd() tea.Cmd {
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled {
		return nil
	}
	url, key := wl.URL, wl.APIKey
	state, ok := m.radioStateFromForm()
	if !ok {
		return nil
	}
	return func() tea.Msg {
		r, err := wavelog.PushRadio(url, key, wavelog.DefaultRadioName, state)
		if err == nil {
			return wlRadioPushMsg{radio: r}
		}
		if apiErr, ok := err.(*wavelog.APIError); ok && apiErr.Code == "not_found" {
			// The radio vanished (e.g. deleted on the server) — recreate it
			// and retry the push once.
			applog.Warn("Wavelog: radio missing — recreating", "name", wavelog.DefaultRadioName)
			if _, nerr := wavelog.EnsureRadio(url, key, wavelog.DefaultRadioName); nerr == nil {
				r2, err2 := wavelog.PushRadio(url, key, wavelog.DefaultRadioName, state)
				return wlRadioPushMsg{radio: r2, err: err2, retried: true}
			}
		}
		return wlRadioPushMsg{err: err}
	}
}

// handleWavelogRadioMsg applies radio ensure/push results. Returns the
// message plus the model changes (id updates) — callers apply m changes.
func (m *Model) handleWavelogRadioMsg(msg tea.Msg) bool {
	switch r := msg.(type) {
	case wlRadioEnsureMsg:
		if r.err != nil {
			m.lookup.wlRadioID = 0
			applog.Warn("Wavelog: radio ensure failed", "error", r.err)
		} else if r.radio != nil {
			m.lookup.wlRadioID = r.radio.ID
			applog.InfoDetail("Wavelog: radio ready",
				fmt.Sprintf("name=%s id=%d", wavelog.DefaultRadioName, r.radio.ID))
		}
		return true
	case wlRadioPushMsg:
		if r.err != nil {
			applog.Warn("Wavelog: radio state push failed", "error", r.err, "retried", r.retried)
		} else if r.radio != nil && r.radio.ID != m.lookup.wlRadioID {
			// A recreated radio has a new id.
			m.lookup.wlRadioID = r.radio.ID
		}
		return true
	}
	return false
}
