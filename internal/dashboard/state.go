package dashboard

import (
	"sync"
	"time"
)

// =============================================================================
// Snapshot model — the full dashboard state served as JSON
// =============================================================================

// Snapshot is the complete read-model served to browsers.
type Snapshot struct {
	App       AppInfo       `json:"app"`
	Station   StationInfo   `json:"station"`
	Operator  OperatorInfo  `json:"operator"`
	Logbook   LogbookInfo   `json:"logbook"`
	Rig       RigInfo       `json:"rig"`
	WSJTX     WSJTXInfo     `json:"wsjtx"`
	ActiveQSO *ActiveQSO    `json:"activeQso,omitempty"`
	LastQSO   *QSOView      `json:"lastQso,omitempty"`
	Recent    []QSOView     `json:"recent"`
	Today     []QSOView     `json:"today,omitempty"`
	Stats     Stats         `json:"stats"`
	Map       MapState      `json:"map"`
	Partner   *PartnerInfo  `json:"partner,omitempty"`
	Display   DisplayConfig `json:"display"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type StationInfo struct {
	Callsign string  `json:"callsign"`
	Locator  string  `json:"locator,omitempty"`
	Lat      float64 `json:"lat,omitempty"`
	Lon      float64 `json:"lon,omitempty"`
	QTH      string  `json:"qth,omitempty"`
	Radio    string  `json:"radio,omitempty"`
	Antenna  string  `json:"antenna,omitempty"`
	PowerW   int     `json:"powerW,omitempty"`
}

type OperatorInfo struct {
	Callsign string `json:"callsign,omitempty"`
	Name     string `json:"name,omitempty"`
}

type LogbookInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type RigInfo struct {
	Enabled      bool      `json:"enabled"`
	Connected    bool      `json:"connected"`
	Name         string    `json:"name,omitempty"`
	FrequencyHz  int64     `json:"frequencyHz,omitempty"`
	Frequency    string    `json:"frequency,omitempty"`
	Band         string    `json:"band,omitempty"`
	Mode         string    `json:"mode,omitempty"`
	PowerW       int       `json:"powerW,omitempty"`
	UpdatedAtUTC time.Time `json:"updatedAtUtc,omitempty"`
}

type WSJTXInfo struct {
	Enabled      bool      `json:"enabled"`
	Connected    bool      `json:"connected"`
	LastMessage  string    `json:"lastMessage,omitempty"`
	UpdatedAtUTC time.Time `json:"updatedAtUtc,omitempty"`
}

type ActiveQSO struct {
	State        string    `json:"state"`
	Source       string    `json:"source"`
	Call         string    `json:"call"`
	Band         string    `json:"band,omitempty"`
	Mode         string    `json:"mode,omitempty"`
	Submode      string    `json:"submode,omitempty"`
	FrequencyHz  int64     `json:"frequencyHz,omitempty"`
	Frequency    string    `json:"frequency,omitempty"`
	RSTSent      string    `json:"rstSent,omitempty"`
	RSTRcvd      string    `json:"rstRcvd,omitempty"`
	Grid         string    `json:"grid,omitempty"`
	Name         string    `json:"name,omitempty"`
	QTH          string    `json:"qth,omitempty"`
	Country      string    `json:"country,omitempty"`
	DXCC         int       `json:"dxcc,omitempty"`
	IsDupe       bool      `json:"isDupe,omitempty"`
	IsNewCall    bool      `json:"isNewCall,omitempty"`
	IsNewDXCC    bool      `json:"isNewDxcc,omitempty"`
	RefNames     string    `json:"refNames,omitempty"`
	UpdatedAtUTC time.Time `json:"updatedAtUtc"`
}

type QSOView struct {
	ID          string    `json:"id,omitempty"`
	TimeUTC     time.Time `json:"timeUtc"`
	Call        string    `json:"call"`
	Band        string    `json:"band,omitempty"`
	Mode        string    `json:"mode,omitempty"`
	Submode     string    `json:"submode,omitempty"`
	FrequencyHz int64     `json:"frequencyHz,omitempty"`
	Frequency   string    `json:"frequency,omitempty"`
	RSTSent     string    `json:"rstSent,omitempty"`
	RSTRcvd     string    `json:"rstRcvd,omitempty"`
	Grid        string    `json:"grid,omitempty"`
	Country     string    `json:"country,omitempty"`
	DXCC        int       `json:"dxcc,omitempty"`
	Operator    string    `json:"operator,omitempty"`
	Lat         float64   `json:"lat,omitempty"`
	Lon         float64   `json:"lon,omitempty"`
}

type Stats struct {
	QSOsToday   int     `json:"qsosToday"`
	QSOsSession int     `json:"qsosSession"`
	UniqueCalls int     `json:"uniqueCalls"`
	DXCC        int     `json:"dxcc"`
	Grids       int     `json:"grids"`
	Bands       int     `json:"bands"`
	Modes       int     `json:"modes"`
	LastQSOAgoS int     `json:"lastQsoAgoS,omitempty"`
	RatePerHour float64 `json:"ratePerHour,omitempty"`
}

type MapState struct {
	Station *MapPoint  `json:"station,omitempty"`
	Recent  []MapPoint `json:"recent"`
}

type MapPoint struct {
	Call      string    `json:"call,omitempty"`
	Label     string    `json:"label,omitempty"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Grid      string    `json:"grid,omitempty"`
	Band      string    `json:"band,omitempty"`
	Mode      string    `json:"mode,omitempty"`
	TimeUTC   time.Time `json:"timeUtc,omitempty"`
	IsLastQSO bool      `json:"isLastQso,omitempty"`
	IsStation bool      `json:"isStation,omitempty"`
}

// PartnerInfo holds lookup data for the currently entered callsign
// (from QRZ, Wavelog, or manual form entry). Aimed at non-ham visitors.
type PartnerInfo struct {
	Call       string  `json:"call,omitempty"`
	Name       string  `json:"name,omitempty"`
	QTH        string  `json:"qth,omitempty"`
	Country    string  `json:"country,omitempty"`
	Continent  string  `json:"continent,omitempty"`
	Grid       string  `json:"grid,omitempty"`
	DistanceKm float64 `json:"distanceKm,omitempty"`
	Bearing    float64 `json:"bearing,omitempty"`
	Lat        float64 `json:"lat,omitempty"`
	Lon        float64 `json:"lon,omitempty"`
	ImageURL   string  `json:"imageUrl,omitempty"`
	Source     string  `json:"source,omitempty"` // "qrz", "wavelog", "form"
}

// DisplayConfig holds dashboard display settings pushed from server config.
type DisplayConfig struct {
	Header1           string `json:"header1,omitempty"`
	Header2           string `json:"header2,omitempty"`
	ClubLogo          string `json:"clubLogo,omitempty"`
	MapTileURL        string `json:"mapTileUrl,omitempty"`
	MapAttrib         string `json:"mapAttrib,omitempty"`
	DrawLines         bool   `json:"drawLines"`
	MaxLines          int    `json:"maxLines,omitempty"`
	HighlightLastQSO  bool   `json:"highlightLastQso"`
	AnimateActivePath bool   `json:"animateActivePath"`
}

// =============================================================================
// State — thread-safe dashboard snapshot with event publishing
// =============================================================================

// maxRecent is the maximum number of recent QSOs kept in the snapshot.
const maxRecent = 20

// State holds the authoritative dashboard snapshot and publishes
// change events to the SSE hub.
type State struct {
	mu       sync.RWMutex
	snapshot Snapshot
	hub      *Hub

	// Change detection — avoids publishing redundant events.
	lastRig        RigInfo
	lastWSJTX      WSJTXInfo
	lastActiveCall string

	// Session counter — incremented on AddLoggedQSO.
	sessionQSOs int
}

// NewState creates a dashboard state with the given hub.
func NewState(hub *Hub) *State {
	return &State{
		hub: hub,
	}
}

// Snapshot returns a copy of the current dashboard state.
// Safe for concurrent readers.
func (s *State) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

// SetApp sets the application metadata. No event is published.
func (s *State) SetApp(name, version string) {
	s.mu.Lock()
	s.snapshot.App = AppInfo{Name: name, Version: version}
	s.mu.Unlock()
}

// SetStation updates station info and publishes if it changed.
func (s *State) SetStation(info StationInfo) {
	s.mu.Lock()
	changed := s.snapshot.Station != info
	s.snapshot.Station = info
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	if changed {
		s.hub.Publish(EventStation, info)
	}
}

// SetOperator updates operator info and publishes if it changed.
func (s *State) SetOperator(info OperatorInfo) {
	s.mu.Lock()
	s.snapshot.Operator = info
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	// Always publish — cheap, called ≤1×/tick, and change detection
	// would block the initial fill when the SSE handshake beats the
	// first pushDashboardFast call.
	s.hub.Publish(EventOperator, info)
}

// SetLogbook updates logbook info and publishes if it changed.
func (s *State) SetLogbook(info LogbookInfo) {
	s.mu.Lock()
	s.snapshot.Logbook = info
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	// Always publish — see SetOperator for rationale.
	s.hub.Publish(EventLogbook, info)
}

// SetRig updates rig info and publishes only when meaningful fields change.
func (s *State) SetRig(info RigInfo) {
	s.mu.Lock()
	changed := info.Connected != s.lastRig.Connected ||
		info.FrequencyHz != s.lastRig.FrequencyHz ||
		info.Mode != s.lastRig.Mode ||
		info.Band != s.lastRig.Band
	s.lastRig = info
	if changed {
		s.snapshot.Rig = info
		s.snapshot.UpdatedAt = timeNow()
		s.mu.Unlock()
		s.hub.Publish(EventRig, info)
	} else {
		s.mu.Unlock()
	}
}

// SetWSJTX updates WSJT-X info and publishes when meaningful fields change.
func (s *State) SetWSJTX(info WSJTXInfo) {
	s.mu.Lock()
	changed := info.Connected != s.lastWSJTX.Connected ||
		info.LastMessage != s.lastWSJTX.LastMessage
	s.lastWSJTX = info
	if changed {
		s.snapshot.WSJTX = info
		s.snapshot.UpdatedAt = timeNow()
		s.mu.Unlock()
		s.hub.Publish(EventWSJTX, info)
	} else {
		s.mu.Unlock()
	}
}

// SetActiveQSO updates the active QSO and publishes when it changed.
// Pass nil to clear. Change detection compares Call + flags so that
// late-arriving dupe/new flags trigger a publish even when the call
// itself hasn't changed.
func (s *State) SetActiveQSO(qso *ActiveQSO) {
	s.mu.Lock()
	if qso != nil {
		s.lastActiveCall = qso.Call
	} else {
		s.lastActiveCall = ""
	}
	// Detect meaningful changes: different call, new flags, or any
	// form field the hero panel displays (grid, country, band, mode, etc.).
	prev := s.snapshot.ActiveQSO
	changed := (qso == nil) != (prev == nil)
	if !changed && qso != nil && prev != nil {
		changed = qso.Call != prev.Call ||
			qso.Band != prev.Band ||
			qso.Mode != prev.Mode ||
			qso.Frequency != prev.Frequency ||
			qso.Grid != prev.Grid ||
			qso.Country != prev.Country ||
			qso.Name != prev.Name ||
			qso.QTH != prev.QTH ||
			qso.RSTSent != prev.RSTSent ||
			qso.RSTRcvd != prev.RSTRcvd ||
			qso.IsDupe != prev.IsDupe ||
			qso.IsNewCall != prev.IsNewCall ||
			qso.IsNewDXCC != prev.IsNewDXCC ||
			qso.RefNames != prev.RefNames
	}
	s.snapshot.ActiveQSO = qso
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	if changed {
		if qso != nil {
			s.hub.Publish(EventActiveQSO, qso)
		} else {
			s.hub.Publish(EventActiveQSO, nil)
		}
	}
}

// ClearActiveQSO clears the active QSO and publishes so the browser
// switches back to overview mode immediately.
func (s *State) ClearActiveQSO() {
	s.mu.Lock()
	s.snapshot.ActiveQSO = nil
	s.lastActiveCall = ""
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	s.hub.Publish(EventActiveQSO, nil)
}

// AddLoggedQSO prepends a QSO to the recent list, updates the last QSO,
// increments the session counter, and publishes events. The recent list is
// capped at maxRecent entries.
func (s *State) AddLoggedQSO(view QSOView) {
	s.mu.Lock()
	s.snapshot.LastQSO = &view
	// Prepend — keep a copy to avoid aliasing the caller's slice.
	recent := make([]QSOView, 0, maxRecent)
	recent = append(recent, view)
	recent = append(recent, s.snapshot.Recent...)
	if len(recent) > maxRecent {
		recent = recent[:maxRecent]
	}
	s.snapshot.Recent = recent
	s.sessionQSOs++
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()

	s.hub.Publish(EventQSOLogged, view)
	s.hub.Publish(EventRecentQSOs, recent)
}

// SetRecent replaces the entire recent QSO list and publishes.
// Use for bulk updates (initial seed, delete refresh) instead of
// many AddLoggedQSO calls.
func (s *State) SetRecent(views []QSOView) {
	s.mu.Lock()
	recent := make([]QSOView, 0, maxRecent)
	for i := 0; i < len(views) && len(recent) < maxRecent; i++ {
		recent = append(recent, views[i])
	}
	s.snapshot.Recent = recent
	if len(recent) > 0 {
		last := recent[0]
		s.snapshot.LastQSO = &last
	}
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	s.hub.Publish(EventRecentQSOs, recent)
}

// SetToday replaces the full today QSO list (for map display, uncapped).
func (s *State) SetToday(views []QSOView) {
	s.mu.Lock()
	s.snapshot.Today = views
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	s.hub.Publish(EventToday, views)
}

// SetStats updates statistics and publishes.
func (s *State) SetStats(stats Stats) {
	s.mu.Lock()
	stats.QSOsSession = s.sessionQSOs
	s.snapshot.Stats = stats
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	s.hub.Publish(EventStats, stats)
}

// LastActiveCall returns the most recently set active callsign.
func (s *State) LastActiveCall() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastActiveCall
}

// SessionQSOs returns the number of QSOs logged this session.
func (s *State) SessionQSOs() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionQSOs
}

// SetPartner updates the partner lookup info and publishes when changed.
// Pass nil to clear.
func (s *State) SetPartner(p *PartnerInfo) {
	s.mu.Lock()
	prev := s.snapshot.Partner
	changed := (p == nil) != (prev == nil)
	if !changed && p != nil && prev != nil {
		changed = p.Call != prev.Call ||
			p.Name != prev.Name ||
			p.QTH != prev.QTH ||
			p.Country != prev.Country ||
			p.Grid != prev.Grid ||
			p.ImageURL != prev.ImageURL
	}
	if p != nil {
		copy := *p
		s.snapshot.Partner = &copy
	} else {
		s.snapshot.Partner = nil
	}
	s.snapshot.UpdatedAt = timeNow()
	s.mu.Unlock()
	if changed {
		s.hub.Publish(EventPartner, s.snapshot.Partner)
	}
}

// SetDisplay updates the dashboard display config. No event needed
// (only changes on server restart or config save).
func (s *State) SetDisplay(d DisplayConfig) {
	s.mu.Lock()
	s.snapshot.Display = d
	s.mu.Unlock()
}
