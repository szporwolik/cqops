package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// wlStationDetailMsg carries the full Wavelog station profile fetched after a
// station selection change. stationID guards against stale results.
type wlStationDetailMsg struct {
	stationID string
	station   *wavelog.Station
	err       error
}

// fetchWavelogStationDetailCmd fetches the full profile of one station.
func fetchWavelogStationDetailCmd(url, key, stationID string) tea.Cmd {
	u := strings.TrimRight(strings.TrimSpace(url), "/")
	k := strings.TrimSpace(key)
	if u == "" || k == "" || stationID == "" {
		return nil
	}
	return func() tea.Msg {
		st, err := wavelog.GetStation(u, k, stationID)
		return wlStationDetailMsg{stationID: stationID, station: st, err: err}
	}
}

// fillStationFormFromWavelog mirrors a Wavelog station profile into the
// station form. Callsign and grid are only overwritten when the profile
// provides them; optional reference/zone fields mirror the profile exactly,
// so values from a previously selected station do not linger.
func fillStationFormFromWavelog(f *StationForm, st *wavelog.Station) {
	if st.Callsign != "" {
		f.Callsign.SetValue(st.Callsign)
	}
	if st.Gridsquare != "" {
		f.Locator.SetValue(st.Gridsquare)
	}
	setNum := func(t *textinput.Model, n int) {
		if n > 0 {
			t.SetValue(strconv.Itoa(n))
		} else {
			t.SetValue("")
		}
	}
	setNum(&f.DXCC, st.DXCC)
	setNum(&f.CQZone, st.CQ)
	setNum(&f.ITUZone, st.ITU)
	f.SOTARef.SetValue(st.SOTA)
	f.POTARef.SetValue(st.POTA)
	f.WWFFRef.SetValue(st.WWFF)
	f.SIG.SetValue(st.SIG)
	f.SIGInfo.SetValue(st.SIGInfo)
}

// applyWavelogStation mirrors the Wavelog station profile values into a
// logbook station: grid, callsign, DXCC entity, CQ/ITU zones and the
// SOTA/POTA/WWFF/SIG reference fields. Returns true when anything changed.
func applyWavelogStation(st *wavelog.Station, s *config.Station) bool {
	changed := false
	set := func(dst *string, v string) {
		if *dst != v {
			*dst = v
			changed = true
		}
	}
	set(&s.Callsign, st.Callsign)
	set(&s.Grid, st.Gridsquare)
	if st.DXCC > 0 && s.DXCC != st.DXCC {
		s.DXCC = st.DXCC
		changed = true
	}
	if st.CQ > 0 && s.CQZone != st.CQ {
		s.CQZone = st.CQ
		changed = true
	}
	if st.ITU > 0 && s.ITUZone != st.ITU {
		s.ITUZone = st.ITU
		changed = true
	}
	set(&s.SOTARef, st.SOTA)
	set(&s.POTARef, st.POTA)
	set(&s.WWFFRef, st.WWFF)
	set(&s.SIG, st.SIG)
	set(&s.SIGInfo, st.SIGInfo)
	return changed
}
