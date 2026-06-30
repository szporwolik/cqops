package app

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/aprs"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ref"
	"github.com/szporwolik/cqops/internal/secrets"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wsjtx"
)

type App struct {
	Config       *config.Config
	ConfigPath   string
	LogbookName  string
	Logbook      *config.Logbook
	DB           *sql.DB
	DBPath       string
	WSJTX        *wsjtx.Listener
	WSJTXUpdated chan struct{}
	DXCC         *dxcc.Prefixes // in-memory DXCC prefix→country lookup
	SCP          *scp.Database  // in-memory Super Check Partial database
	RefDB        *ref.DB        // reference database (SOTA/POTA/WWFF)
	Secrets      *secrets.Store // encrypted secrets (passwords, API keys)
	APRSClient   *aprs.Client   // APRS-IS persistent connection
	APRSCache    *aprs.CacheDB  // APRS station cache database
	aprsStatusCB func(connected bool, err error)
	pruneStopCh  chan struct{} // stops the APRS cache pruning goroutine

	// lastWSJTX tracks the effective WSJT-X config last applied to the
	// listener. Used to avoid unnecessary Stop/Start cycles when config
	// is saved but the WSJT-X settings haven't changed.
	lastWSJTX struct {
		enabled bool
		host    string
		port    int
	}
}

func Init() (*App, error) {
	cfg, configPath, err := config.EnsureConfig()
	if err != nil {
		applog.Error("Config is corrupted or missing — cannot start", "error", err.Error())
		return nil, fmt.Errorf("config: %w", err)
	}
	applog.Info("Config OK", "path", configPath)

	// Secrets are already loaded and applied by EnsureConfig — just grab
	// the store reference for later use (e.g. corruption toast).
	sec := cfg.SecretsStore()

	name, lb, err := config.ResolveLogbook(cfg, "")
	if err != nil {
		applog.Error("Cannot resolve logbook", "error", err.Error())
		return nil, fmt.Errorf("logbook: %w", err)
	}

	dbPath, err := config.DBPath(name, lb)
	if err != nil {
		applog.Error("Cannot determine database path", "logbook", name, "error", err.Error())
		return nil, fmt.Errorf("db path: %w", err)
	}

	db, err := store.InitDB(dbPath)
	if err != nil {
		applog.Error("Database is corrupted or cannot be opened — cannot start", "path", dbPath, "error", err.Error())
		return nil, fmt.Errorf("database: %w", err)
	}
	applog.Info("Database OK", "path", dbPath)

	app := &App{
		Config:       cfg,
		ConfigPath:   configPath,
		LogbookName:  name,
		Logbook:      lb,
		DB:           db,
		DBPath:       dbPath,
		WSJTX:        wsjtx.NewListener(),
		WSJTXUpdated: make(chan struct{}, 10),
		Secrets:      sec,
	}

	// WSJT-X will be started later by the TUI model Init() with per-rig settings.
	// Don't start here — we don't know which rig is active yet.

	// Load cached data files concurrently — no mutual dependencies,
	// independent I/O. On slow storage (SD card on Pi), this cuts
	// startup time by loading all three files in parallel.
	var wg sync.WaitGroup
	if app.Config.General.UseCTY {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cacheDir, _ := config.CacheDir()
			ctyPath := filepath.Join(cacheDir, "cty.dat")
			if prefixes, err := dxcc.LoadLocal(ctyPath); err == nil {
				app.DXCC = prefixes
				applog.Info("DXCC: prefix data loaded from cache")
			} else {
				applog.Info("DXCC: no cached data yet — will fetch when online")
			}
		}()
	}
	if app.Config.General.UseSCP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cacheDir, _ := config.CacheDir()
			scpPath := filepath.Join(cacheDir, "MASTER.SCP")
			if db, err := scp.LoadLocal(scpPath); err == nil {
				app.SCP = db
				applog.Info("SCP: callsign database loaded from cache")
			} else {
				applog.Info("SCP: no cached data yet — will fetch when online")
			}
		}()
	}
	if app.Config.General.UseRef {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cacheDir, _ := config.CacheDir()
			refPath := filepath.Join(cacheDir, "ref.db")
			if rdb, err := ref.Open(refPath); err == nil {
				app.RefDB = rdb
				applog.Info("REF: database opened")
			} else {
				applog.Info("REF: cannot open database — will rebuild when online")
			}
		}()
	}
	wg.Wait()

	return app, nil
}

func (a *App) Close() {
	applog.Info("Shutting down — stopping WSJT-X listener")
	a.WSJTX.Stop()
	// Stop APRS cache pruner.
	a.stopAPRSPruner()
	if a.APRSClient != nil {
		applog.Debug("Stopping APRS client")
		a.APRSClient.Stop()
	}
	if a.APRSCache != nil {
		applog.Debug("Closing APRS cache database")
		a.APRSCache.Close()
	}
	if a.DB != nil {
		applog.Debug("Closing database")
		a.DB.Close()
	}
	if a.RefDB != nil {
		applog.Debug("Closing reference database")
		a.RefDB.Close()
	}
	applog.Info("CQOps shutdown complete")
}

// MaybeRestartWSJTX restarts the WSJT-X listener only when the effective
// configuration (enabled, host, port) has changed since the last apply.
// The UDP socket is properly closed and reopened, so switching between
// rigs with different ports works correctly.
// Settings are passed from the active rig preset (per-rig config).
func (a *App) MaybeRestartWSJTX(enabled bool, host string, port int) {
	if enabled == a.lastWSJTX.enabled &&
		host == a.lastWSJTX.host &&
		port == a.lastWSJTX.port {
		return
	}

	a.WSJTX.Stop()
	if enabled {
		if err := a.WSJTX.Start(host, port); err != nil {
			applog.Error("WSJT-X restart failed", "error", err.Error())
			return
		}
	}

	a.lastWSJTX.enabled = enabled
	a.lastWSJTX.host = host
	a.lastWSJTX.port = port

	select {
	case a.WSJTXUpdated <- struct{}{}:
	default:
	}
}

// MaybeRestartAPRS starts or stops the APRS-IS client based on the
// active logbook's APRS configuration. Non-blocking — connection runs
// asynchronously. Call SetAPRSStatusCallback to receive toast updates.
func (a *App) MaybeRestartAPRS() {
	aprsCfg := a.Logbook.APRS
	enabled := aprsCfg != nil && aprsCfg.Enabled

	if !enabled {
		a.stopAPRSPruner()
		if a.APRSClient != nil {
			applog.Info("APRS: disabled, stopping client")
			a.APRSClient.Stop()
			a.APRSClient = nil
			// Only notify if we actually stopped a running client.
			if a.aprsStatusCB != nil {
				a.aprsStatusCB(false, nil)
			}
		}
		if a.APRSCache != nil {
			a.APRSCache.Close()
			a.APRSCache = nil
		}
		return
	}

	// Open cache database if needed.
	if a.APRSCache == nil {
		cacheDir, err := config.CacheDir()
		if err != nil {
			applog.Error("APRS: cannot determine cache directory", "error", err)
			return
		}
		cachePath := filepath.Join(cacheDir, "aprs.db")
		cache, err := aprs.OpenCacheDB(cachePath)
		if err != nil {
			applog.Error("APRS: cannot open cache database", "error", err)
			return
		}
		a.APRSCache = cache
	}

	server := aprsCfg.Server
	if server == "" {
		server = "euro.aprs2.net:14580"
	}
	callsign := aprsCfg.Callsign
	if callsign == "" {
		// Derive from station callsign: strip portable/test suffixes, add -10 SSID.
		base := a.Logbook.Station.Callsign
		if idx := strings.IndexAny(base, "/"); idx >= 0 {
			base = base[:idx]
		}
		if base != "" {
			callsign = base + "-10"
		}
	}

	// Build range filter from station position.
	var filter string
	if aprsCfg.RadiusKm > 0 {
		g := a.Logbook.Station.Grid
		if g != "" {
			lat, lon, err := gridToLatLon(g)
			if err == nil {
				filter = aprs.BuildRangeFilter(lat, lon, aprsCfg.RadiusKm)
			}
		}
	}

	// Start or restart client.
	if a.APRSClient != nil {
		applog.Debug("APRS: stopping previous client")
		a.APRSClient.Stop()
	}
	applog.Info("APRS: starting client", "server", server, "callsign", callsign)
	a.APRSClient = aprs.NewClient(server, callsign, aprsCfg.Passcode, filter)
	a.APRSClient.OnStatus = func(connected bool, err error) {
		if connected {
			applog.Info("APRS: connected", "server", server, "callsign", callsign)
		}
		// On error: connect() already logged the detail; just forward to TUI for toast.
		if a.aprsStatusCB != nil {
			a.aprsStatusCB(connected, err)
		}
	}
	a.APRSClient.OnPacket = func(raw string) {
		sr, ok := aprs.ParsePositionPacket(raw)
		if !ok {
			return
		}
		sr.RawPacket = raw
		sr.LastHeard = time.Now()
		applog.Debug("APRS: position parsed", "callsign", sr.Callsign, "lat", sr.Lat, "lon", sr.Lon)
		if a.APRSCache != nil {
			if err := a.APRSCache.UpsertStation(sr); err != nil {
				applog.Debug("APRS: cache upsert failed", "error", err)
			}
		}
	}

	// Start is non-blocking — connects in a goroutine.
	a.APRSClient.Start()

	// Start periodic cache pruning (every 5 min, removes stations >60 min old).
	a.startAPRSPruner()
}

// SetAPRSStatusCallback registers a callback for APRS connection state changes.
// Called from the TUI model to enable toast notifications.
func (a *App) SetAPRSStatusCallback(cb func(connected bool, err error)) {
	a.aprsStatusCB = cb
}

// startAPRSPruner launches a background goroutine that periodically deletes
// cached APRS stations older than the retention window (60 min). Runs every
// 5 minutes. Stops when stopAPRSPruner is called or the app shuts down.
func (a *App) startAPRSPruner() {
	a.stopAPRSPruner() // ensure no duplicate
	a.pruneStopCh = make(chan struct{})
	go func() {
		const pruneInterval = 5 * time.Minute
		const retainDuration = 60 * time.Minute
		ticker := time.NewTicker(pruneInterval)
		defer ticker.Stop()

		// Prune once at startup to clean up stale entries from a previous run.
		a.pruneOnce(retainDuration)

		for {
			select {
			case <-ticker.C:
				a.pruneOnce(retainDuration)
			case <-a.pruneStopCh:
				return
			}
		}
	}()
	applog.Debug("APRS: cache pruner started", "interval", "5m", "retain", "60m")
}

func (a *App) stopAPRSPruner() {
	if a.pruneStopCh != nil {
		close(a.pruneStopCh)
		a.pruneStopCh = nil
		applog.Debug("APRS: cache pruner stopped")
	}
}

func (a *App) pruneOnce(retainDuration time.Duration) {
	if a.APRSCache == nil {
		return
	}
	cutoff := time.Now().Add(-retainDuration)
	n, err := a.APRSCache.PruneOlderThan(cutoff)
	if err != nil {
		applog.Debug("APRS: cache prune failed", "error", err)
		return
	}
	if n > 0 {
		applog.Debug("APRS: cache pruned", "removed", n)
	}
}

// gridToLatLon converts a Maidenhead grid square (4/6/8/10 char) to decimal lat/lon.
// Simplified local version to avoid import cycle. Returns the center of the grid cell.
func gridToLatLon(grid string) (float64, float64, error) {
	grid = strings.ToUpper(strings.TrimSpace(grid))
	if len(grid) < 4 {
		return 0, 0, fmt.Errorf("grid too short: %s", grid)
	}
	lon := float64(grid[0]-'A')*20 - 180
	lat := float64(grid[1]-'A')*10 - 90
	lon += float64(grid[2]-'0') * 2
	lat += float64(grid[3]-'0') * 1
	if len(grid) >= 6 {
		lon += float64(grid[4]-'A') * (5.0 / 60.0)
		lat += float64(grid[5]-'A') * (2.5 / 60.0)
		lon += 2.5 / 60.0  // center of sub-square
		lat += 1.25 / 60.0 // center of sub-square
		if len(grid) >= 8 {
			lon += float64(grid[6]-'0') * (0.5 / 60.0)
			lat += float64(grid[7]-'0') * (0.25 / 60.0)
			lon += 0.25 / 60.0  // center of extended cell
			lat += 0.125 / 60.0 // center of extended cell
			if len(grid) >= 10 {
				lon += float64(grid[8]-'A') * (0.5 / 60.0 / 24.0)
				lat += float64(grid[9]-'A') * (0.25 / 60.0 / 24.0)
				lon += 0.5 / 60.0 / 48.0  // center
				lat += 0.25 / 60.0 / 48.0 // center
			}
		}
	} else {
		lon += 1.0 // center of 2° square
		lat += 0.5 // center of 1° square
	}
	return lat, lon, nil
}

func (a *App) SwitchLogbook(name string) error {
	if _, ok := a.Config.Logbooks[name]; !ok {
		return fmt.Errorf("logbook %q not found", name)
	}

	if a.DB != nil {
		a.DB.Close()
	}

	lb := a.Config.Logbooks[name]
	dbPath, err := config.DBPath(name, &lb)
	if err != nil {
		return fmt.Errorf("db path: %w", err)
	}

	db, err := store.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	applog.Info("Database OK", "path", dbPath)

	a.Config.State.ActiveLogbook = name
	a.LogbookName = name
	a.Logbook = &lb
	a.DB = db
	a.DBPath = dbPath

	// Persist the active logbook choice so it survives restarts.
	if err := config.Save(a.ConfigPath, a.Config); err != nil {
		applog.Warn("Failed to save active logbook", "error", err)
	}

	// Restart APRS for the new logbook config.
	a.MaybeRestartAPRS()

	return nil
}

func (a *App) StationSummary() string {
	s := a.Logbook.Station
	parts := []string{}
	if s.Callsign != "" {
		parts = append(parts, s.Callsign)
	}
	if s.Grid != "" {
		parts = append(parts, s.Grid)
	}

	return strings.Join(parts, " ")
}

// LogbookDisplayName returns the human-readable name for the active logbook.
func (a *App) LogbookDisplayName() string {
	return config.LogbookDisplayName(a.Logbook)
}

// SetActiveContest sets the active contest for the current logbook, updating
// both the in-memory pointer and the config map so the change survives saves.
func (a *App) SetActiveContest(id string) {
	a.Logbook.ActiveContest = id
	lb := a.Config.Logbooks[a.LogbookName]
	lb.ActiveContest = id
	a.Config.Logbooks[a.LogbookName] = lb
}

// SetActiveOperator sets the active operator for the current logbook.
func (a *App) SetActiveOperator(id string) {
	a.Logbook.ActiveOperator = id
	lb := a.Config.Logbooks[a.LogbookName]
	lb.ActiveOperator = id
	a.Config.Logbooks[a.LogbookName] = lb
}
