package app

import (
	"database/sql"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/aprs"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/geo"
	"github.com/szporwolik/cqops/internal/ref"
	"github.com/szporwolik/cqops/internal/secrets"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wsjtx"
	"go.bug.st/serial"
)

type App struct {
	Config           *config.Config
	ConfigPath       string
	LogbookName      string
	Logbook          *config.Logbook
	DB               *sql.DB
	DBPath           string
	WSJTX            *wsjtx.Listener
	WSJTXUpdated     chan struct{}
	DXCC             *dxcc.Prefixes // in-memory DXCC prefix→country lookup
	SCP              *scp.Database  // in-memory Super Check Partial database
	RefDB            *ref.DB        // reference database (SOTA/POTA/WWFF)
	Secrets          *secrets.Store // encrypted secrets (passwords, API keys)
	APRSClient       aprs.Client    // APRS connection (TCP APRS-IS or KISS serial)
	APRSCache        *aprs.CacheDB  // APRS station cache database
	aprsStatusCB     func(connected bool, err error)
	aprsBeaconCB     func(callsign string) // called after each successful beacon
	aprsRefresh      bool                  // set by RequestAPRSRefresh, cleared by dashboard
	pruneStopCh      chan struct{}         // stops the APRS cache pruning goroutine
	beaconStopCh     chan struct{}         // stops the APRS beacon goroutine
	aprsRestartTimer *time.Timer           // debounces rapid logbook switches for APRS restart
	gpsGrid          string                // last known GPS grid (set by TUI model)
	gpsHasFix        bool                  // true when GPS has a valid fix

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
	// Cancel any pending debounced APRS restart.
	a.StopAPRSTimer()
	// Stop APRS cache pruner and beacon.
	a.stopAPRSPruner()
	a.stopAPRSBeacon()
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

// MaybeRestartAPRS starts or stops the APRS client based on the
// active logbook's APRS configuration and global APRS service settings.
// Non-blocking — connection runs asynchronously.
// Call SetAPRSStatusCallback to receive toast updates.
func (a *App) MaybeRestartAPRS() {
	aprsGlobal := a.Config.Integrations.APRS
	aprsCfg := a.Logbook.APRS
	enabled := aprsGlobal.Enabled && aprsCfg != nil && aprsCfg.Enabled

	if !enabled {
		a.stopAPRSPruner()
		a.stopAPRSBeacon()
		if a.APRSClient != nil {
			applog.Info("APRS: disabled, stopping client")
			a.APRSClient.Stop()
			a.APRSClient = nil
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

	// Open cache database if needed (shared by APRS-IS and KISS).
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

	// KISS service — start KISS TNC client.
	if aprsGlobal.Service == "kiss" {
		port := aprsGlobal.Port
		baud := aprsGlobal.BaudRate
		if port == "" || baud == 0 {
			applog.Debug("APRS: KISS not configured (missing port/baud)")
			return
		}
		dataBits := aprsGlobal.DataBits
		if dataBits < 5 || dataBits > 8 {
			dataBits = 8
		}
		par := serialParity(aprsGlobal.Parity)
		stop := serialStopBits(aprsGlobal.StopBits)

		if a.APRSClient != nil {
			a.APRSClient.Stop()
			a.APRSClient = nil
			// Let the OS release the serial port handle before reconnecting.
			time.Sleep(200 * time.Millisecond)
		}

		kiss := aprs.NewKISSClient(port, baud, dataBits, par, stop, aprsGlobal.DTR, aprsGlobal.RTS)
		kiss.OnStatus = func(connected bool, err error) {
			if connected {
				applog.Info("KISS: connected", "port", port)
			}
			if a.aprsStatusCB != nil {
				a.aprsStatusCB(connected, err)
			}
		}
		kiss.OnPacket = func(raw string) {
			sr, ok := aprs.ParsePositionPacket(raw)
			if !ok {
				preview := raw
				if len(preview) > 100 {
					preview = preview[:100]
				}
				applog.Debug("KISS: unparsed frame", "preview", preview)
				return
			}
			sr.RawPacket = raw
			sr.Source = "kiss"
			sr.LastHeard = time.Now()
			applog.Debug("APRS: position parsed (KISS)", "callsign", sr.Callsign, "lat", sr.Lat, "lon", sr.Lon)
			if a.APRSCache != nil {
				if err := a.APRSCache.UpsertStation(sr); err != nil {
					applog.Debug("APRS: cache upsert failed", "error", err)
				}
			}
		}
		a.APRSClient = kiss
		kiss.Start()

		// Start periodic cache pruning and beacon goroutine.
		a.startAPRSPruner()
		a.startAPRSBeacon()
		return
	}

	// KISS Server service — connect to a KISS TNC over TCP.
	if aprsGlobal.Service == "kiss_server" {
		host := aprsGlobal.KISSServerHost
		if host == "" {
			host = "127.0.0.1"
		}
		port := aprsGlobal.KISSServerPort
		if port == "" {
			port = "8001"
		}
		addr := net.JoinHostPort(host, port)

		if a.APRSClient != nil {
			a.APRSClient.Stop()
			a.APRSClient = nil
		}

		kc := aprs.NewKISSServerClient(addr)
		kc.OnStatus = func(connected bool, err error) {
			if connected {
				applog.Info("KISS server: connected", "addr", addr)
			}
			if a.aprsStatusCB != nil {
				a.aprsStatusCB(connected, err)
			}
		}
		kc.OnPacket = func(raw string) {
			sr, ok := aprs.ParsePositionPacket(raw)
			if !ok {
				return
			}
			sr.RawPacket = raw
			sr.Source = "kiss"
			sr.LastHeard = time.Now()
			if a.APRSCache != nil {
				if err := a.APRSCache.UpsertStation(sr); err != nil {
					applog.Debug("APRS: cache upsert failed", "error", err)
				}
			}
		}
		a.APRSClient = kc
		kc.Start()

		a.startAPRSPruner()
		a.startAPRSBeacon()
		return
	}

	server := a.Config.Integrations.APRS.Server
	if server == "" {
		server = aprsDefaultServer
	}
	passcode := aprsCfg.Passcode
	callsign := aprsCfg.Callsign
	if callsign == "" {
		// Derive from station callsign: strip portable/test suffixes, add SSID.
		base := a.Logbook.Station.Callsign
		if idx := strings.IndexAny(base, "/"); idx >= 0 {
			base = base[:idx]
		}
		if base != "" {
			callsign = base + aprsDefaultSSID
		}
	}

	// Build range filter from station position.
	var filter string
	if aprsCfg.RadiusKm > 0 {
		g := a.Logbook.Station.Grid
		if g != "" {
			lat, lon, err := geo.GridToLatLon(g)
			if err == nil {
				filter = aprs.BuildRangeFilter(lat, lon, aprsCfg.RadiusKm)
			}
		}
	}

	// Stop previous client asynchronously — the 3-second Stop() timeout
	// would freeze the TUI if called from the Update path.
	if a.APRSClient != nil {
		old := a.APRSClient
		a.APRSClient = nil
		go old.Stop()
	}
	applog.Info("APRS: starting client", "server", server, "callsign", callsign)
	tcp := aprs.NewTCPClient(server, callsign, passcode, filter)
	tcp.OnStatus = func(connected bool, err error) {
		if connected {
			applog.Info("APRS: connected", "server", server, "callsign", callsign)
		}
		if a.aprsStatusCB != nil {
			a.aprsStatusCB(connected, err)
		}
	}
	tcp.OnPacket = func(raw string) {
		sr, ok := aprs.ParsePositionPacket(raw)
		if !ok {
			return
		}
		sr.RawPacket = raw
		sr.Source = "aprs_is"
		sr.LastHeard = time.Now()
		applog.Debug("APRS: position parsed", "callsign", sr.Callsign, "lat", sr.Lat, "lon", sr.Lon)
		if a.APRSCache != nil {
			if err := a.APRSCache.UpsertStation(sr); err != nil {
				applog.Debug("APRS: cache upsert failed", "error", err)
			}
		}
	}
	a.APRSClient = tcp
	tcp.Start()

	// Start periodic cache pruning (every 5 min, removes stations >60 min old).
	a.startAPRSPruner()

	// Start beacon goroutine if TX is enabled.
	a.startAPRSBeacon()
}

// SetAPRSStatusCallback registers a callback for APRS connection state changes.
// Called from the TUI model to enable toast notifications.
func (a *App) SetAPRSStatusCallback(cb func(connected bool, err error)) {
	a.aprsStatusCB = cb
}

// SetAPRSBeaconCallback registers a callback invoked after each successful
// APRS position beacon. Called from the TUI model for toast notifications.
func (a *App) SetAPRSBeaconCallback(cb func(callsign string)) {
	a.aprsBeaconCB = cb
}

// RequestAPRSRefresh flags that the dashboard should push APRS data on the
// next tick. Called when logbook radius changes so the map updates
// immediately instead of waiting for the periodic timer.
func (a *App) RequestAPRSRefresh() {
	a.aprsRefresh = true
}

// ConsumeAPRSRefresh returns true and clears the flag if a refresh was
// requested. Used by the dashboard tick to trigger an immediate push.
func (a *App) ConsumeAPRSRefresh() bool {
	if a.aprsRefresh {
		a.aprsRefresh = false
		return true
	}
	return false
}

// ScheduleAPRSRestart debounces APRS client restarts so rapid logbook
// switching doesn't hammer the serial port or APRS-IS server with
// repeated stop/start cycles. The restart fires 3 seconds after the
// last call — if another call arrives before then, the timer resets.
func (a *App) ScheduleAPRSRestart() {
	if a.aprsRestartTimer != nil {
		a.aprsRestartTimer.Stop()
	}
	a.aprsRestartTimer = time.AfterFunc(3*time.Second, func() {
		a.MaybeRestartAPRS()
	})
}

// StopAPRSTimer cancels any pending debounced APRS restart. Call during
// app shutdown to avoid goroutine leaks.
func (a *App) StopAPRSTimer() {
	if a.aprsRestartTimer != nil {
		a.aprsRestartTimer.Stop()
		a.aprsRestartTimer = nil
	}
}

// APRS cache retention and pruning intervals.
const (
	aprsPruneInterval  = 5 * time.Minute
	aprsRetainDuration = 60 * time.Minute
	aprsBeaconMin      = 1                      // minimum beacon interval in minutes
	aprsDefaultServer  = "euro.aprs2.net:14580" // default APRS-IS server
	aprsDefaultSSID    = "-10"                  // default APRS SSID suffix
)

// startAPRSPruner launches a background goroutine that periodically deletes
// cached APRS stations older than the retention window. Runs every 5 minutes.
// Stops when stopAPRSPruner is called or the app shuts down.
func (a *App) startAPRSPruner() {
	a.stopAPRSPruner() // ensure no duplicate
	a.pruneStopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(aprsPruneInterval)
		defer ticker.Stop()

		// Prune once at startup to clean up stale entries from a previous run.
		a.pruneOnce(aprsRetainDuration)

		for {
			select {
			case <-ticker.C:
				a.pruneOnce(aprsRetainDuration)
			case <-a.pruneStopCh:
				return
			}
		}
	}()
	applog.Debug("APRS: cache pruner started", "interval", aprsPruneInterval, "retain", aprsRetainDuration)
}

func (a *App) stopAPRSPruner() {
	if a.pruneStopCh != nil {
		close(a.pruneStopCh)
		a.pruneStopCh = nil
		applog.Debug("APRS: cache pruner stopped")
	}
}

func (a *App) pruneOnce(retainDuration time.Duration) {
	cache := a.APRSCache
	if cache == nil {
		return
	}
	cutoff := time.Now().Add(-retainDuration)
	n, err := cache.PruneOlderThan(cutoff)
	if err != nil {
		applog.Debug("APRS: cache prune failed", "error", err)
		return
	}
	if n > 0 {
		applog.Debug("APRS: cache pruned", "removed", n)
	}
}

// startAPRSBeacon launches a goroutine that periodically sends the station's
// position to APRS-IS. The interval is read from the active logbook's APRS
// config on each tick (it can change when the user edits settings).
// Respects LastBeaconAt on startup — won't send if a beacon was already
// sent recently (e.g. after an app restart).
func (a *App) startAPRSBeacon() {
	a.stopAPRSBeacon()
	a.beaconStopCh = make(chan struct{})
	go func() {
		defer func() { a.beaconStopCh = nil }()
		// Wait 10s for the APRS client to connect.
		select {
		case <-time.After(10 * time.Second):
		case <-a.beaconStopCh:
			return
		}

		for {
			aprsCfg := a.Logbook.APRS
			if aprsCfg == nil || !aprsCfg.Enabled || !aprsCfg.SendLocation {
				select {
				case <-time.After(30 * time.Second):
					continue
				case <-a.beaconStopCh:
					return
				}
			}

			intervalMin := aprsCfg.IntervalMin
			if intervalMin < 1 {
				intervalMin = 1
			}
			interval := time.Duration(intervalMin) * time.Minute

			// Wait until next scheduled beacon based on LastBeaconAt.
			if aprsCfg.LastBeaconAt != "" {
				last, err := time.Parse(time.RFC3339, aprsCfg.LastBeaconAt)
				if err == nil {
					elapsed := time.Since(last)
					if elapsed < interval {
						remaining := interval - elapsed
						applog.Debug("APRS: beacon waiting", "remaining", remaining.Round(time.Second), "lastBeacon", aprsCfg.LastBeaconAt)
						select {
						case <-time.After(remaining):
						case <-a.beaconStopCh:
							return
						}
					}
				}
			}

			a.sendAPRSBeacon(aprsCfg)
			a.persistBeaconTimestamp(aprsCfg)

			// Wait for next interval.
			select {
			case <-time.After(interval):
			case <-a.beaconStopCh:
				return
			}
		}
	}()
	applog.Debug("APRS: beacon goroutine started")
}

func (a *App) stopAPRSBeacon() {
	if a.beaconStopCh != nil {
		close(a.beaconStopCh)
		a.beaconStopCh = nil
		applog.Debug("APRS: beacon goroutine stopped")
	}
}

func (a *App) sendAPRSBeacon(aprsCfg *config.APRSConfig) {
	// Guard against nil client — the beacon goroutine may fire between
	// MaybeRestartAPRS stopping the old client and starting a new one.
	client := a.APRSClient
	if client == nil || !client.IsConnected() {
		applog.Debug("APRS: beacon skipped — not connected")
		return
	}

	callsign := aprsCfg.Callsign
	if callsign == "" {
		callsign = a.Logbook.Station.Callsign
		if idx := strings.IndexAny(callsign, "/"); idx >= 0 {
			callsign = callsign[:idx]
		}
		if callsign != "" {
			callsign += "-10"
		}
	}

	grid := a.EffectiveGrid()
	if grid == "" {
		applog.Debug("APRS: beacon skipped — no station grid")
		return
	}
	lat, lon, err := geo.GridToLatLon(grid)
	if err != nil {
		applog.Debug("APRS: beacon skipped — grid error", "error", err)
		return
	}

	symbol := aprsCfg.Symbol
	if symbol == "" {
		symbol = "/-"
	}

	if tcp, ok := a.APRSClient.(*aprs.TCPClient); ok {
		if err := tcp.SendPosition(callsign, lat, lon, symbol, aprsCfg.Comment); err != nil {
			applog.Warn("APRS: beacon failed", "error", err)
		} else if a.aprsBeaconCB != nil {
			a.aprsBeaconCB(callsign)
		}
	} else if kiss, ok := a.APRSClient.(*aprs.KISSClient); ok {
		if err := kiss.SendPosition(callsign, lat, lon, symbol, aprsCfg.Comment); err != nil {
			applog.Warn("KISS: beacon failed", "error", err)
		} else if a.aprsBeaconCB != nil {
			a.aprsBeaconCB(callsign)
		}
	} else if ks, ok := a.APRSClient.(*aprs.KISSServerClient); ok {
		if err := ks.SendPosition(callsign, lat, lon, symbol, aprsCfg.Comment); err != nil {
			applog.Warn("KISS server: beacon failed", "error", err)
		} else if a.aprsBeaconCB != nil {
			a.aprsBeaconCB(callsign)
		}
	}
}

// persistBeaconTimestamp writes the current time as LastBeaconAt and saves
// the config. Errors are logged but not surfaced — beaconing continues.
func (a *App) persistBeaconTimestamp(aprsCfg *config.APRSConfig) {
	aprsCfg.LastBeaconAt = time.Now().UTC().Format(time.RFC3339)
	if err := config.Save(a.ConfigPath, a.Config); err != nil {
		applog.Warn("APRS: failed to persist beacon timestamp", "error", err)
	}
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

	// Stop APRS goroutines BEFORE replacing the logbook to prevent the
	// old beacon/pruner from reading the new logbook's config.
	a.stopAPRSPruner()
	a.stopAPRSBeacon()

	a.Config.State.ActiveLogbook = name
	a.LogbookName = name
	a.Logbook = &lb
	a.DB = db
	a.DBPath = dbPath

	// Persist the active logbook choice so it survives restarts.
	if err := config.Save(a.ConfigPath, a.Config); err != nil {
		applog.Warn("Failed to save active logbook", "error", err)
	}

	// Restart APRS for the new logbook config (debounced — won't fire
	// until rapid switching settles).
	a.ScheduleAPRSRestart()

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

// SetGPSGrid is called by the TUI model when GPS position updates.
func (a *App) SetGPSGrid(grid string, hasFix bool) {
	a.gpsGrid = grid
	a.gpsHasFix = hasFix
}

// EffectiveGrid returns the GPS-derived grid when GPS is enabled, has a fix,
// and the logbook has gps_grid enabled. Falls back to the configured station
// grid otherwise. Safe to call from any goroutine.
func (a *App) EffectiveGrid() string {
	if a.Config.Integrations.GPS.Enabled && a.gpsHasFix && a.gpsGrid != "" &&
		a.Logbook != nil && a.Logbook.Station.GPSGrid {
		return a.gpsGrid
	}
	if a.Logbook != nil {
		return strings.TrimSpace(strings.ToUpper(a.Logbook.Station.Grid))
	}
	return ""
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

// serialParity converts a config parity string to a serial.Parity value.
func serialParity(s string) serial.Parity {
	switch s {
	case "odd":
		return serial.OddParity
	case "even":
		return serial.EvenParity
	case "mark":
		return serial.MarkParity
	case "space":
		return serial.SpaceParity
	default:
		return serial.NoParity
	}
}

// serialStopBits converts a config stop bits string to a serial.StopBits value.
func serialStopBits(s string) serial.StopBits {
	switch s {
	case "1.5":
		return serial.OnePointFiveStopBits
	case "2":
		return serial.TwoStopBits
	default:
		return serial.OneStopBit
	}
}
