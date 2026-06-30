package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// handleDXCUpdate routes messages to the DXC table for keyboard navigation
// and handles filter keybindings.
func (m *Model) handleDXCUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "f4":
			m.screen = screenQSO
			return m, cmd

		case "pgup":
			// Cycle time window forward — skip if same value.
			next := (m.dxc.timeIdx + 1) % len(dxcTimeWindows)
			if m.dxc.timeIdx == next {
				return m, cmd
			}
			m.dxc.timeIdx = next
			m.dxc.timeFilter = dxcTimeWindows[m.dxc.timeIdx]
			m.dxc.tableReady = false
			return m, cmd

		case "pgdown":
			// Cycle time window backward.
			next := m.dxc.timeIdx - 1
			if next < 0 {
				next = len(dxcTimeWindows) - 1
			}
			if m.dxc.timeIdx == next {
				return m, cmd
			}
			m.dxc.timeIdx = next
			m.dxc.timeFilter = dxcTimeWindows[m.dxc.timeIdx]
			m.dxc.tableReady = false
			return m, cmd

		case "home":
			m.dxcCycleFilter(&m.dxc.bandIdx, &m.dxc.bandFilter, m.dxcBandChoices())
			return m, cmd

		case "end":
			m.dxcCycleFilterBack(&m.dxc.bandIdx, &m.dxc.bandFilter, m.dxcBandChoices())
			return m, cmd

		case `\`:
			m.dxcCycleFilter(&m.dxc.contIdx, &m.dxc.contFilter, m.dxcContChoices())
			return m, cmd

		case "enter":
			// Fill QSO form with highlighted spot, tune rig, and jump to form.
			lookupCmd := m.dxcFillFromSelected()
			cmd = tea.Batch(cmd, m.dxcTuneCmd(), lookupCmd)
			m.screen = screenQSO
			return m, cmd

		case "insert":
			m.dxcCycleFilter(&m.dxc.modeIdx, &m.dxc.modeFilter, dxcFilterChoices(m.dxcAvailableModes()))
			return m, cmd

		case "delete":
			m.dxcCycleFilterBack(&m.dxc.modeIdx, &m.dxc.modeFilter, dxcFilterChoices(m.dxcAvailableModes()))
			return m, cmd

		case "backspace":
			// Clear all filters — only rebuild if filters were actually active.
			if m.dxc.timeFilter != 0 || m.dxc.bandFilter != "" || m.dxc.contFilter != "" || m.dxc.modeFilter != "" {
				m.dxc.timeFilter = 0
				m.dxc.timeIdx = 0
				m.dxc.bandFilter = ""
				m.dxc.bandIdx = 0
				m.dxc.contFilter = ""
				m.dxc.contIdx = 0
				m.dxc.modeFilter = ""
				m.dxc.modeIdx = 0
				m.dxc.tableReady = false
			}
			return m, cmd
		}
	}

	if m.dxc.tableReady {
		// Space is reserved for tune — don't forward to the table.
		if kp, ok := msg.(tea.KeyPressMsg); ok && (kp.String() == " " || kp.String() == "space" || kp.Code == ' ') {
			cmd = tea.Batch(cmd, m.dxcTuneCmd())
			return m, cmd
		}
		t, c := m.dxc.table.Update(msg)
		m.dxc.table = t
		m.updateDXCSelectedCall()
		if c != nil {
			cmd = tea.Batch(cmd, c)
		}
	}

	return m, cmd
}

// dxcFilterChoices prepends "" to a list of filter options so "all" (no filter)
// is the first choice, matching dxcBandChoices/dxcContChoices conventions.
func dxcFilterChoices(opts []string) []string {
	return append([]string{""}, opts...)
}

// dxcCycleFilter advances *idx to the next choice in opts and sets *filter.
// No-op if opts is empty or the new idx equals the current idx.
func (m *Model) dxcCycleFilter(idx *int, filter *string, opts []string) {
	if len(opts) == 0 {
		return
	}
	next := (*idx + 1) % len(opts)
	if *idx != next {
		*idx = next
		*filter = opts[*idx]
		m.dxc.tableReady = false
	}
}

// dxcCycleFilterBack decrements *idx to the previous choice in opts.
func (m *Model) dxcCycleFilterBack(idx *int, filter *string, opts []string) {
	if len(opts) == 0 {
		return
	}
	next := *idx - 1
	if next < 0 {
		next = len(opts) - 1
	}
	if *idx != next {
		*idx = next
		*filter = opts[*idx]
		m.dxc.tableReady = false
	}
}

// dxcFillFromSelected fills the QSO form with the currently highlighted DXC spot
// and returns a Cmd that triggers QRZ/Wavelog lookups for the populated callsign.
func (m *Model) dxcFillFromSelected() tea.Cmd {
	spot, ok := m.dxcSpotAtCursor()
	if !ok {
		return nil
	}

	// Clear call-dependent state only when the call actually changes.
	prevCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if !strings.EqualFold(spot.DXCall, prevCall) {
		m.lookup.partnerData = nil
		m.lookup.wlPrivateData = nil
		m.lookup.wlLookupDone = false
		m.invalidatePartnerMapCache()
	}
	// Always set the callsign field.
	m.fields[fieldCall].SetValue(spot.DXCall)
	// Clear QRZ-populated fields so old callsign data does not bleed.
	m.clearQRZFields()

	// Fill frequency: when WSJT-X is offline, use DXC spot frequency.
	// Always set band and mode from the spot — these are known regardless
	// of flrig/WSJT-X state.
	if spot.Band != "" {
		m.fields[fieldBand].SetValue(spot.Band)
	}
	if spot.Mode != "" {
		mode, subm := qso.NormalizeMode(spot.Mode, "")
		m.fields[fieldMode].SetValue(mode)
		if subm != "" {
			m.fields[fieldSubmode].SetValue(subm)
		}
	}
	applog.Debug("DXC: dxcFillFromSelected fields",
		"call", spot.DXCall,
		"band", spot.Band,
		"mode", spot.Mode,
		"freq_khz", spot.Frequency,
	)
	if !m.wsjtx.online {
		freqMHz := spot.Frequency / 1000
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.5f", freqMHz))
		// Don't call applyFreqDefaults — band and mode are already set from
		// the spot above. Only auto-fill SSB submode if needed.
		m.autoFillSSBSubmode()
		applog.Info("DXC: populated QSO form from spot",
			"call", spot.DXCall,
			"freq", fmt.Sprintf("%.1f kHz", spot.Frequency),
		)
	} else {
		applog.Info("DXC: populated QSO form call from spot",
			"call", spot.DXCall,
		)
	}

	// Parse spot comment for reference designators (SOTA, POTA, WWFF, IOTA)
	// and auto-fill the corresponding QSO form fields.
	m.parseSpotCommentForRefs(spot.Comment)

	// Commit the callsign and trigger lookups immediately — the user is
	// jumping to the QSO form, so background QRZ/Wavelog lookups should
	// start right away rather than waiting for a manual field exit.
	cur := m.commitCall()
	if cur != "" {
		m.autoFillRST()
		m.autoFillSSBSubmode()
		m.dxccAutoFill()
		m.scpMatches = nil
		m.scpCacheKey = ""
		m.prefillContestExchange()
		// Ensure date is set before dupe check — checkDupe bails if
		// the date field is empty, and the form may not have been in
		// focus long enough for autoUpdateDateTime to run yet.
		if strings.TrimSpace(m.fields[fieldDate].Value()) == "" {
			m.fields[fieldDate].SetValue(time.Now().UTC().Format("20060102"))
		}
		m.lookup.qrzCall = cur
		m.lookup.wlCall = cur
		m.checkDupe()
		applog.Debug("DXC: dupe result after populate",
			"call", cur,
			"dupe", m.dupe,
		)
		m.rc.pathSig = ""
		m.rc.logStatsSig = ""
		return m.lookupCallCmd(cur)
	}
	return nil
}

// dxcSpotAtCursor returns the DXC spot at the current table cursor position.
// Returns the spot data captured at the last cursor move — NOT a fresh DB query,
// so the frequency matches what the user sees in the table.
func (m *Model) dxcSpotAtCursor() (store.DXCSpot, bool) {
	if m.dxc.selectedCall == "" || !m.dxc.tableReady {
		return store.DXCSpot{}, false
	}
	if m.dxc.selectedSpot.DXCall == "" {
		return store.DXCSpot{}, false
	}
	applog.Debug("DXC: dxcSpotAtCursor",
		"call", m.dxc.selectedSpot.DXCall,
		"freq_khz", m.dxc.selectedSpot.Frequency,
	)
	return m.dxc.selectedSpot, true
}

// updateDXCSelectedCall updates the cached selected spot from the current
// table cursor position. Call after cursor movement or table rebuild.
// Stores the FULL spot data so frequency is locked at cursor-move time.
func (m *Model) updateDXCSelectedCall() {
	if !m.dxc.tableReady {
		m.dxc.selectedCall = ""
		m.dxc.selectedSpot = store.DXCSpot{}
		return
	}
	cursor := m.dxc.table.Cursor()
	spots := m.dxc.cachedSpots
	if len(spots) == 0 {
		spots = m.dxcFilteredSpots()
	}
	prev := m.dxc.selectedCall
	if cursor >= 0 && cursor < len(spots) {
		m.dxc.selectedSpot = spots[cursor]
		m.dxc.selectedCall = m.dxc.selectedSpot.DXCall
	} else {
		m.dxc.selectedCall = ""
		m.dxc.selectedSpot = store.DXCSpot{}
	}
	if m.dxc.selectedCall != prev {
		applog.Debug("DXC: selected call changed",
			"cursor", cursor,
			"prev", prev,
			"new", m.dxc.selectedCall,
			"freq_khz", m.dxc.selectedSpot.Frequency,
		)
	}
}

// wellKnownWWFFPrefixes lists the most common country-specific WWFF prefixes
// (top ~10 ham nations). The generic "WWFF" prefix is always checked first.
var wellKnownWWFFPrefixes = []string{
	"KFF", "DLFF", "SPFF", "FFF", "GFF", "JAFF",
	"IFF", "EAFF", "OZFF", "VKFF", "VEFF", "ONFF",
}

// Pre-compiled reference regexps — compiled once at init, not per spot fill.
var reIOTA = regexp.MustCompile(`\b(AF|AN|AS|EU|NA|OC|SA)-\d{3,4}\b`)
var reSOTA = regexp.MustCompile(`\b([A-Z0-9]+/[A-Z0-9]+-\d{3,4})\b`)
var rePOTA = regexp.MustCompile(`\b([A-Z0-9]{1,4}-\d{4,6})\b`)
var reWWFF = regexp.MustCompile(`\b((?:` + strings.Join(append([]string{"WWFF"}, wellKnownWWFFPrefixes...), "|") + `)-\d{3,5})\b`)

// parseSpotCommentForRefs scans a DXC spot comment for SOTA, POTA, WWFF, and
// IOTA reference designators and fills the corresponding QSO form fields.
// Only fills empty fields — existing values are never overwritten.
func (m *Model) parseSpotCommentForRefs(comment string) {
	if comment == "" {
		return
	}
	upper := strings.ToUpper(strings.TrimSpace(comment))

	// IOTA: two-letter continent code, dash, digits.
	if m.fields[fieldIOTA].Value() == "" {
		if match := reIOTA.FindStringSubmatch(upper); match != nil {
			m.fields[fieldIOTA].SetValue(match[0])
		}
	}

	// SOTA: country/association prefix, slash, summit code.
	if m.fields[fieldSOTA].Value() == "" {
		if match := reSOTA.FindStringSubmatch(upper); match != nil {
			m.fields[fieldSOTA].SetValue(match[1])
		}
	}

	// WWFF: check BEFORE POTA.
	if m.fields[fieldWWFF].Value() == "" {
		if match := reWWFF.FindStringSubmatch(upper); match != nil {
			m.fields[fieldWWFF].SetValue(match[1])
		}
	}

	// POTA: country prefix, dash, digits. Skip WWFF-like prefixes.
	if m.fields[fieldPOTA].Value() == "" {
		for _, match := range rePOTA.FindAllStringSubmatch(upper, -1) {
			ref := match[1]
			if strings.HasSuffix(strings.SplitN(ref, "-", 2)[0], "FF") {
				continue
			}
			if strings.Contains(ref, "KHZ") || strings.Contains(ref, "DB-") {
				continue
			}
			m.fields[fieldPOTA].SetValue(ref)
			break
		}
	}
}
