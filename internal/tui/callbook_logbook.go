package tui

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/ftl/hamradio/latlon"
	"github.com/ftl/hamradio/locator"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/ctybig"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// LogbookCallbookProvider searches past local QSOs for callsign data.
type LogbookCallbookProvider struct {
	db       *sql.DB
	priority int
}

// NewLogbookCallbookProvider creates a provider that queries the local logbook.
func NewLogbookCallbookProvider(db *sql.DB, priority int) *LogbookCallbookProvider {
	return &LogbookCallbookProvider{db: db, priority: priority}
}

func (p *LogbookCallbookProvider) Name() string          { return "Local Logbook" }
func (p *LogbookCallbookProvider) Priority() int         { return p.priority }
func (p *LogbookCallbookProvider) TestConnection() error { return nil }

func (p *LogbookCallbookProvider) Lookup(callsign string) (*callbook.Result, error) {
	if p.db == nil || callsign == "" {
		return nil, nil
	}
	applog.Debug("Logbook provider: searching past QSOs", "call", callsign)
	qsos, err := store.SearchQSOsByCall(p.db, callsign, 20)
	if err != nil {
		applog.Debug("Logbook provider: query failed", "call", callsign, "error", err)
		return nil, nil
	}
	if len(qsos) == 0 {
		applog.Debug("Logbook provider: no past QSOs found", "call", callsign)
		return nil, nil
	}
	applog.Debug("Logbook provider: found QSOs", "call", callsign, "count", len(qsos))

	var name, qth, country, grid, dxcc, cqZone, ituZone string
	for _, q := range qsos {
		if name == "" {
			name = q.Name
		}
		if qth == "" && q.QTH != "" {
			qth = q.QTH
		}
		if country == "" && q.Country != "" {
			country = q.Country
		}
		if dxcc == "" {
			dxcc = q.DXCC
		}
		if cqZone == "" {
			cqZone = q.CQZone
		}
		if ituZone == "" {
			ituZone = q.ITUZone
		}
		if grid == "" && q.GridSquare != "" &&
			q.SOTARef == "" && q.POTARef == "" && q.WWFFRef == "" {
			grid = q.GridSquare
		}
		if name != "" && qth != "" && country != "" && grid != "" {
			break
		}
	}

	if name == "" && qth == "" && country == "" && grid == "" {
		applog.Debug("Logbook provider: all fields empty in past QSOs", "call", callsign)
		return nil, nil
	}

	applog.Debug("Logbook provider: returning data", "call", callsign,
		"name", name, "qth", qth, "country", country, "grid", grid,
		"dxcc", dxcc, "cq", cqZone, "itu", ituZone)
	return &callbook.Result{
		Callsign: callsign,
		Name:     name, QTH: qth, Country: country, Grid: grid,
		DXCC: dxcc, CQZone: cqZone, ITUZone: ituZone,
		Provider: "logbook",
	}, nil
}

// CTYProvider uses the Big CTY DXCC prefix database to enrich callsigns.
type CTYProvider struct {
	db       *ctybig.DB
	priority int
}

// NewCTYProvider creates a provider backed by the Big CTY prefix database.
func NewCTYProvider(db *ctybig.DB, priority int) *CTYProvider {
	return &CTYProvider{db: db, priority: priority}
}

func (p *CTYProvider) Name() string          { return "CTY.DAT" }
func (p *CTYProvider) Priority() int         { return p.priority }
func (p *CTYProvider) TestConnection() error { return nil }

func (p *CTYProvider) Lookup(callsign string) (*callbook.Result, error) {
	if p.db == nil || callsign == "" {
		return nil, nil
	}
	e := p.db.Find(callsign)
	if e == nil {
		applog.Debug("CTY provider: no prefix match", "call", callsign)
		return nil, nil
	}
	applog.Debug("CTY provider: matched prefix", "call", callsign,
		"country", e.Name, "cq", e.CQZone, "itu", e.ITUZone)
	r := &callbook.Result{
		Callsign: callsign,
		Country:  strings.TrimSpace(e.Name),
		Provider: "cty",
	}
	if e.DXCC > 0 {
		r.DXCC = strconv.Itoa(e.DXCC)
	}
	if e.CQZone != 0 {
		r.CQZone = strconv.Itoa(e.CQZone)
	}
	if e.ITUZone != 0 {
		r.ITUZone = strconv.Itoa(e.ITUZone)
	}
	// Grid from Big CTY coordinates — lower priority than QRZ/HamQTH,
	// but provides a reasonable fallback when no callbook data is available.
	if e.Lat != 0 || e.Lon != 0 {
		ll := latlon.NewLatLon(latlon.Latitude(e.Lat), latlon.Longitude(e.Lon))
		grid := locator.LatLonToLocator(ll, 4)
		gridStr := strings.TrimRight(string(grid[:]), "\x00")
		if len(gridStr) >= 4 {
			r.Grid = strings.ToUpper(gridStr[:4])
		}
	}
	applog.Debug("CTY: enriched from prefix DB", "call", callsign, "country", r.Country, "dxcc", r.DXCC, "grid", r.Grid)
	return r, nil
}

// WavelogCallbookProvider queries the Wavelog private lookup API for
// callsign data. Runs below QRZ (default priority 10) — fills gaps when
// QRZ is unavailable or has no data. The worked/confirmed status still
// runs via the separate Wavelog lookup for the partner view.
type WavelogCallbookProvider struct {
	url      string
	apiKey   string
	priority int
}

// NewWavelogCallbookProvider creates a Wavelog callbook provider.
func NewWavelogCallbookProvider(url, apiKey string, priority int) *WavelogCallbookProvider {
	return &WavelogCallbookProvider{url: url, apiKey: apiKey, priority: priority}
}

func (p *WavelogCallbookProvider) Name() string  { return "Wavelog" }
func (p *WavelogCallbookProvider) Priority() int { return p.priority }
func (p *WavelogCallbookProvider) TestConnection() error {
	if p.url == "" || p.apiKey == "" {
		return nil
	}
	return wavelog.TestConnection(p.url, p.apiKey)
}

func (p *WavelogCallbookProvider) Lookup(callsign string) (*callbook.Result, error) {
	if p.url == "" || p.apiKey == "" || callsign == "" {
		return nil, nil
	}
	applog.Debug("Wavelog provider: looking up", "call", callsign)
	// No band/mode — we just want name/QTH/grid/country/zones.
	data, err := wavelog.PrivateLookup(p.url, p.apiKey, callsign, "", "", "")
	if err != nil {
		applog.Debug("Wavelog provider: lookup failed", "call", callsign, "error", err)
		return nil, err
	}
	if data == nil || data.Callsign() == "" {
		applog.Debug("Wavelog provider: no data", "call", callsign)
		return nil, nil
	}
	r := &callbook.Result{
		Callsign: callsign,
		Name:     data.Name(),
		QTH:      data.QTH(),
		Country:  data.Country(),
		Grid:     data.Grid(),
		DXCC:     data.DXCCID(),
		CQZone:   data.CQZone(),
		ITUZone:  data.ITUZone(),
		State:    data.State(),
		Provider: "wavelog",
	}
	// When the only fields are country/zone (DXCC prefix data already
	// covered by CTY.DAT), treat this as "no meaningful data" — don't
	// mislead the user with a Wavelog badge for bare prefix info.
	if r.Name == "" && r.Grid == "" && r.QTH == "" {
		applog.Debug("Wavelog provider: only DXCC data, skipping", "call", callsign)
		return nil, nil
	}
	applog.Debug("Wavelog provider: returning data", "call", callsign,
		"name", r.Name, "qth", r.QTH, "country", r.Country, "grid", r.Grid)
	return r, nil
}
