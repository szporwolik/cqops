package app

import (
	"bytes"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/aprs"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ctybig"
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
	BigCTY           *ctybig.DB     // Big CTY prefix lookup with DXCC entity numbers (from cty.csv)
	SCP              *scp.Database  // in-memory Super Check Partial database
	RefDB            *ref.DB        // reference database (SOTA/POTA/WWFF)
	Secrets          *secrets.Store // encrypted secrets (passwords, API keys)
	APRSClient       aprs.Client    // APRS connection (TCP APRS-IS or KISS serial)
	APRSCache        *aprs.CacheDB  // APRS station cache database
	lock             *lockFile      // single-instance guard
	aprsStatusCB     func(connected bool, err error)
	aprsBeaconCB     func(callsign string) // called after each successful beacon
	aprsRefresh      bool                  // set by RequestAPRSRefresh, cleared by dashboard
	pruneStopCh      chan struct{}         // stops the APRS cache pruning goroutine
	beaconStopCh     chan struct{}         // stops the APRS beacon goroutine
	aprsRestartTimer *time.Timer           // debounces rapid logbook switches for APRS restart

	// APRS lifecycle is single-owner: every transition (start/stop/restart)
	// runs on the TUI main loop. Workers and the debounce timer hand work
	// back through these channels instead of touching shared state.
	aprsRestartCh chan struct{}  // debounce timer → owner
	aprsEvents    chan aprsEvent // APRS workers → owner (status, beacon)
	aprsGen       uint64         // bumped on each client replacement; stale events dropped
	aprsPrunerWG  sync.WaitGroup // joins the pruner before its cache is replaced
	aprsBeaconWG  sync.WaitGroup // joins the beacon before its resources are replaced

	gpsMu      sync.RWMutex // protects gpsGrid, gpsHasFix and beaconGrid
	gpsGrid    string       // last known GPS grid (set by TUI model)
	gpsHasFix  bool         // true when GPS has a valid fix
	beaconGrid string       // cached effective grid for APRS beacon workers (owner-refreshed)
	Offline    bool         // when true, skip all network operations
	InetOnline bool         // true when internet connectivity check succeeded

	// dbMu guards dbHolders: retired logbook databases stay open while
	// background operations captured them via KeepDBAlive, so in-flight
	// work can finish on the old logbook after a switch.
	dbMu      sync.Mutex
	dbHolders map[*sql.DB]*dbHolder

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
	// Single-instance guard acquired FIRST — before any config is read or
	// written. Two instances racing at startup must never both proceed to
	// the first-run wizard or open the same SQLite database. The lock is
	// created atomically (O_EXCL), so exactly one instance can own it.
	configDir, err := config.ConfigDir()
	if err != nil {
		applog.Error("Cannot determine config directory", "error", err.Error())
		return nil, fmt.Errorf("config dir: %w", err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		applog.Error("Cannot create config directory", "error", err.Error())
		return nil, fmt.Errorf("config dir: %w", err)
	}
	lk, err := acquireLock(configDir)
	if err != nil {
		applog.Error("Lock acquisition failed", "error", err.Error())
		return nil, err
	}

	cfg, configPath, err := config.EnsureConfig()
	if err != nil {
		applog.Error("Config is corrupted or missing — cannot start", "error", err.Error())
		lk.release()
		return nil, fmt.Errorf("config: %w", err)
	}
	applog.Info("Config OK", "path", configPath)

	// Secrets are already loaded and applied by EnsureConfig — just grab
	// the store reference for later use (e.g. corruption toast).
	sec := cfg.SecretsStore()

	name, lb, err := config.ResolveLogbook(cfg, "")
	if err != nil {
		applog.Error("Cannot resolve logbook", "error", err.Error())
		lk.release()
		return nil, fmt.Errorf("logbook: %w", err)
	}

	dbPath, err := config.DBPath(name, lb)
	if err != nil {
		applog.Error("Cannot determine database path", "logbook", name, "error", err.Error())
		lk.release()
		return nil, fmt.Errorf("db path: %w", err)
	}

	db, err := store.InitDB(dbPath)
	if err != nil {
		applog.Error("Database is corrupted or cannot be opened — cannot start", "path", dbPath, "error", err.Error())
		lk.release()
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
		lock:         lk,
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
			csvPath := filepath.Join(cacheDir, "cty.csv")
			if data, err := os.ReadFile(csvPath); err == nil && len(data) > 0 {
				if db, err := ctybig.ParseCSV(bytes.NewReader(data)); err == nil {
					app.BigCTY = db
					applog.Info("DXCC: Big CTY loaded from cache", "entries", db.Prefixes())
				}
			} else {
				applog.Info("DXCC: no cached Big CTY yet — will fetch when online")
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
	if a.lock != nil {
		a.lock.release()
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
// Must run on the owner goroutine (the TUI main loop): workers and the
// debounce timer hand their work back here via RunPendingAPRS.
func (a *App) MaybeRestartAPRS() {
	a.ensureAPRSChannels()
	aprsGlobal := a.Config.Integrations.APRS
	aprsCfg := a.Logbook.APRS
	logbookEnabled := aprsCfg != nil && aprsCfg.Enabled

	// Fully disabled — tear down client, beacon, pruner, and cache.
	if !aprsGlobal.Enabled {
		a.stopAPRS()
		return
	}

	// Receive-only: the integration is enabled at the CQOps level, but the
	// active logbook has no APRS config. Stations are still cached for the
	// F3 pane and dashboard map; nothing is ever transmitted.
	if !logbookEnabled {
		a.startAPRS(aprsGlobal, aprsCfg, true)
		return
	}

	a.startAPRS(aprsGlobal, aprsCfg, false)
}

// stopAPRS tears down the APRS client, beacon goroutine, pruner, and cache.
// The workers are joined before their resources are replaced or closed.
func (a *App) stopAPRS() {
	a.stopAPRSPruner()
	a.stopAPRSBeacon()
	a.aprsGen++ // invalidate in-flight events from the client about to stop
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
}

// startAPRS starts (or restarts) the APRS client for the given config.
// receiveOnly runs a listener without the beacon goroutine and falls back
// to a fixed default receive radius from the station grid. The APRS-IS
// passcode is always computed from the login callsign.
func (a *App) startAPRS(aprsGlobal config.APRSGlobalConfig, aprsCfg *config.APRSConfig, receiveOnly bool) {
	// APRS is enabled — but don't start the network client if we're
	// known to be offline (--offline flag or failed health checks).
	if a.Offline || !a.InetOnline {
		applog.Debug("APRS: start skipped — offline", "forced", a.Offline, "inet", a.InetOnline)
		return
	}

	// APRS-IS requires a login callsign — validate it before opening the
	// cache database so invalid configs fail cleanly.
	callsign := ""
	if aprsCfg != nil {
		callsign = aprsCfg.Callsign
	}
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
	if callsign == "" && (aprsGlobal.Service == "" || aprsGlobal.Service == "aprs_is") {
		// Cannot log in without a callsign — nothing to receive.
		applog.Warn("APRS: cannot start — no callsign for login")
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

		// New client generation — in-flight events from the previous client
		// are dropped when they reach the owner.
		a.aprsGen++
		gen := a.aprsGen
		kiss := aprs.NewKISSClient(port, baud, dataBits, par, stop, aprsGlobal.DTR, aprsGlobal.RTS)
		kiss.OnStatus = func(connected bool, err error) {
			a.pushAPRSStatus(gen, connected, err)
		}
		cache := a.APRSCache // immutable snapshot for the worker
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
			// LastHeard is already set from the embedded packet timestamp
			// when present — arrival time is only a fallback.
			if sr.LastHeard.IsZero() {
				sr.LastHeard = time.Now()
			}
			applog.Debug("APRS: position parsed (KISS)", "callsign", sr.Callsign, "lat", sr.Lat, "lon", sr.Lon)
			if cache != nil {
				if err := cache.UpsertStation(sr); err != nil {
					applog.Debug("APRS: cache upsert failed", "error", err)
				}
			}
		}
		a.APRSClient = kiss
		kiss.Start()

		// Start periodic cache pruning and, in full mode, the beacon goroutine.
		a.startAPRSPruner()
		if !receiveOnly {
			a.startAPRSBeacon()
		}
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

		a.aprsGen++
		gen := a.aprsGen
		kc := aprs.NewKISSServerClient(addr)
		kc.OnStatus = func(connected bool, err error) {
			a.pushAPRSStatus(gen, connected, err)
		}
		cache := a.APRSCache
		kc.OnPacket = func(raw string) {
			sr, ok := aprs.ParsePositionPacket(raw)
			if !ok {
				preview := raw
				if len(preview) > 100 {
					preview = preview[:100]
				}
				applog.Debug("KISS server: unparsed frame", "preview", preview)
				return
			}
			sr.RawPacket = raw
			sr.Source = "kiss"
			// LastHeard is already set from the embedded packet timestamp
			// when present — arrival time is only a fallback.
			if sr.LastHeard.IsZero() {
				sr.LastHeard = time.Now()
			}
			applog.Debug("APRS: position parsed (KISS server)", "callsign", sr.Callsign, "lat", sr.Lat, "lon", sr.Lon)
			if cache != nil {
				if err := cache.UpsertStation(sr); err != nil {
					applog.Debug("APRS: cache upsert failed", "error", err)
				}
			}
		}
		a.APRSClient = kc
		kc.Start()

		a.startAPRSPruner()
		if !receiveOnly {
			a.startAPRSBeacon()
		}
		return
	}

	server := aprsGlobal.Server
	if server == "" {
		server = aprsDefaultServer
	}
	// The standard APRS-IS passcode is computed from the login callsign —
	// accepted by all servers, so no credential needs to be stored.
	passcode := aprs.Passcode(callsign)

	// Build range filter from station position.
	// Prefer GPS-derived grid when available, fall back to configured grid.
	// Receive-only mode uses a fixed default radius so the cache only
	// holds nearby traffic (no worldwide APRS-IS flood).
	var filter string
	radiusKm := 0
	if aprsCfg != nil && aprsCfg.RadiusKm > 0 {
		radiusKm = aprsCfg.RadiusKm
	} else if receiveOnly {
		radiusKm = aprsDefaultReceiveRadiusKm
	}
	if radiusKm > 0 {
		g := a.EffectiveGrid()
		if g != "" {
			lat, lon, err := geo.GridToLatLon(g)
			if err == nil {
				filter = aprs.BuildRangeFilter(lat, lon, radiusKm)
			}
		}
	}

	// Stop previous client asynchronously — the 3-second Stop() timeout
	// would freeze the TUI if called from the Update path. Its in-flight
	// events are dropped via the generation bump.
	if a.APRSClient != nil {
		old := a.APRSClient
		a.APRSClient = nil
		go old.Stop()
	}
	a.aprsGen++
	gen := a.aprsGen
	applog.Info("APRS: starting client", "server", server, "callsign", callsign)
	tcp := aprs.NewTCPClient(server, callsign, passcode, filter)
	tcp.OnStatus = func(connected bool, err error) {
		if connected {
			applog.Info("APRS: connected", "server", server, "callsign", callsign)
		}
		a.pushAPRSStatus(gen, connected, err)
	}
	cache := a.APRSCache
	tcp.OnPacket = func(raw string) {
		sr, ok := aprs.ParsePositionPacket(raw)
		if !ok {
			return
		}
		sr.RawPacket = raw
		sr.Source = "aprs_is"
		// LastHeard is already set from the embedded packet timestamp
		// when present — arrival time is only a fallback.
		if sr.LastHeard.IsZero() {
			sr.LastHeard = time.Now()
		}
		applog.Debug("APRS: position parsed", "callsign", sr.Callsign, "lat", sr.Lat, "lon", sr.Lon)
		if cache != nil {
			if err := cache.UpsertStation(sr); err != nil {
				applog.Debug("APRS: cache upsert failed", "error", err)
			}
		}
	}
	a.APRSClient = tcp
	tcp.Start()

	// Start periodic cache pruning (every 5 min, removes stations >60 min old).
	a.startAPRSPruner()

	// Start beacon goroutine if TX is enabled.
	if !receiveOnly {
		a.startAPRSBeacon()
	}
}

// SetAPRSStatusCallback registers a callback for APRS connection state
// changes. The callback is invoked on the owner goroutine (RunPendingAPRS),
// never from client goroutines.
func (a *App) SetAPRSStatusCallback(cb func(connected bool, err error)) {
	a.aprsStatusCB = cb
}

// pushAPRSStatus hands a client status change to the owner goroutine.
// Safe to call from any goroutine; the owner drops events whose generation
// no longer matches (replaced clients fire a stale disconnect after Stop).
func (a *App) pushAPRSStatus(gen uint64, connected bool, err error) {
	select {
	case a.aprsEvents <- aprsEvent{kind: aprsEvStatus, gen: gen, connected: connected, err: err}:
	default:
	}
}

// SetAPRSBeaconCallback registers a callback invoked after each successful
// APRS position beacon. Invoked on the owner goroutine (RunPendingAPRS).
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
// The timer never runs the transition itself: it hands the work back to
// the owner goroutine (RunPendingAPRS), so every APRS lifecycle
// transition stays on the TUI main loop.
func (a *App) ScheduleAPRSRestart() {
	a.ensureAPRSChannels()
	if a.aprsRestartTimer != nil {
		a.aprsRestartTimer.Stop()
	}
	a.aprsRestartTimer = time.AfterFunc(3*time.Second, func() {
		select {
		case a.aprsRestartCh <- struct{}{}:
		default:
		}
	})
}

// ensureAPRSChannels lazily initializes the owner-handoff channels so App
// values built directly by tests keep working.
func (a *App) ensureAPRSChannels() {
	if a.aprsRestartCh == nil {
		a.aprsRestartCh = make(chan struct{}, 1)
	}
	if a.aprsEvents == nil {
		a.aprsEvents = make(chan aprsEvent, 16)
	}
}

// RunPendingAPRS processes APRS lifecycle work handed back by background
// workers. Must run on the owner goroutine (the TUI main loop) — this is
// what makes every APRS lifecycle transition single-owner: workers only
// publish events, and all configuration, callbacks, and lifecycle changes
// happen here.
func (a *App) RunPendingAPRS() {
	a.ensureAPRSChannels()
	a.refreshBeaconGrid()

	// Drain events first — they may describe state about to change.
	for i := 0; i < 64; i++ {
		select {
		case ev := <-a.aprsEvents:
			a.handleAPRSEvent(ev)
		default:
			goto eventsDone
		}
	}
eventsDone:

	select {
	case <-a.aprsRestartCh:
		a.MaybeRestartAPRS()
	default:
	}
}

// handleAPRSEvent acts on one worker event. Owner goroutine only.
func (a *App) handleAPRSEvent(ev aprsEvent) {
	switch ev.kind {
	case aprsEvStatus:
		if ev.gen != a.aprsGen {
			return // event from a replaced client
		}
		if a.aprsStatusCB != nil {
			a.aprsStatusCB(ev.connected, ev.err)
		}
	case aprsEvBeacon:
		if ev.gen != a.aprsGen {
			return // beacon from a replaced run
		}
		if a.aprsBeaconCB != nil {
			a.aprsBeaconCB(ev.callsign)
		}
		// Persist the beacon timestamp here — the worker never touches
		// the live config.
		if a.Logbook != nil {
			if aprsCfg := a.Logbook.APRS; aprsCfg != nil {
				aprsCfg.LastBeaconAt = time.Now().UTC().Format(time.RFC3339)
				if err := config.Save(a.ConfigPath, a.Config); err != nil {
					applog.Warn("APRS: failed to persist beacon timestamp", "error", err)
				}
			}
		}
	}
}

// aprsEvent carries APRS worker results back to the owner goroutine.
// Workers never touch UI state or live configuration directly.
type aprsEventKind int

const (
	aprsEvStatus aprsEventKind = iota
	aprsEvBeacon
)

type aprsEvent struct {
	kind      aprsEventKind
	gen       uint64
	connected bool
	err       error
	callsign  string
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
	aprsDefaultServer  = "euro.aprs2.net:14580" // default APRS-IS server
	aprsDefaultSSID    = "-10"                  // default APRS SSID suffix

	aprsDefaultReceiveRadiusKm = 100 // receive-only range filter
)

// startAPRSPruner launches a background goroutine that periodically deletes
// cached APRS stations older than the retention window. Runs every 5 minutes.
// Stops when stopAPRSPruner is called or the app shuts down.
func (a *App) startAPRSPruner() {
	a.stopAPRSPruner() // joins any previous pruner
	cache := a.APRSCache
	if cache == nil {
		return
	}
	stopCh := make(chan struct{})
	a.pruneStopCh = stopCh
	// The worker receives an immutable cache snapshot — it never reads the
	// APRSCache field, which the owner may replace or close.
	a.aprsPrunerWG.Add(1)
	go func() {
		defer a.aprsPrunerWG.Done()
		a.aprsPruneLoop(stopCh, cache)
	}()
	applog.Debug("APRS: cache pruner started", "interval", aprsPruneInterval, "retain", aprsRetainDuration)
}

// aprsPruneLoop periodically deletes cached stations older than the
// retention window until the given stop channel closes.
func (a *App) aprsPruneLoop(stopCh chan struct{}, cache *aprs.CacheDB) {
	ticker := time.NewTicker(aprsPruneInterval)
	defer ticker.Stop()

	// Prune once at startup to clean up stale entries from a previous run.
	a.pruneOnce(cache, aprsRetainDuration)

	for {
		select {
		case <-ticker.C:
			a.pruneOnce(cache, aprsRetainDuration)
		case <-stopCh:
			return
		}
	}
}

func (a *App) stopAPRSPruner() {
	if a.pruneStopCh != nil {
		close(a.pruneStopCh)
		a.pruneStopCh = nil
		a.aprsPrunerWG.Wait() // join before the cache can be replaced/closed
		applog.Debug("APRS: cache pruner stopped")
	}
}

func (a *App) pruneOnce(cache *aprs.CacheDB, retainDuration time.Duration) {
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

// aprsBeaconSnap is an immutable snapshot handed to a beacon worker. The
// worker must never read live logbook/client state — restarts and logbook
// switches replace those; the worker only knows its own snapshot and the
// cached beacon grid (gpsMu-protected).
type aprsBeaconSnap struct {
	cfg      config.APRSConfig
	callsign string
	client   aprs.Client
	gen      uint64
}

// startAPRSBeacon launches a goroutine that periodically sends the station's
// position to APRS-IS, using an immutable snapshot of the logbook APRS
// config and the active client. Stops when stopAPRSBeacon is called or the
// app shuts down.
func (a *App) startAPRSBeacon() {
	a.stopAPRSBeacon() // joins any previous beacon
	a.ensureAPRSChannels()
	client := a.APRSClient
	aprsCfg := a.Logbook.APRS
	if client == nil || aprsCfg == nil {
		return
	}

	snap := aprsBeaconSnap{
		cfg:      *aprsCfg, // value copy — the worker mutates only this
		client:   client,
		callsign: a.aprsBeaconCallsign(aprsCfg),
		gen:      a.aprsGen,
	}
	stopCh := make(chan struct{})
	a.beaconStopCh = stopCh
	a.aprsBeaconWG.Add(1)
	go func() {
		defer a.aprsBeaconWG.Done()
		a.aprsBeaconLoop(stopCh, snap)
	}()
	applog.Debug("APRS: beacon goroutine started")
}

// aprsBeaconCallsign derives the login callsign used for beacons: the
// configured APRS callsign, else the station callsign without a portable
// suffix plus the default SSID.
func (a *App) aprsBeaconCallsign(aprsCfg *config.APRSConfig) string {
	if aprsCfg != nil && aprsCfg.Callsign != "" {
		return aprsCfg.Callsign
	}
	if a.Logbook == nil {
		return ""
	}
	base := a.Logbook.Station.Callsign
	if idx := strings.IndexAny(base, "/"); idx >= 0 {
		base = base[:idx]
	}
	if base != "" {
		return base + aprsDefaultSSID
	}
	return ""
}

// aprsBeaconLoop sends position beacons until the given stop channel closes.
func (a *App) aprsBeaconLoop(stopCh chan struct{}, snap aprsBeaconSnap) {
	// Wait 10s for the APRS client to connect.
	select {
	case <-time.After(10 * time.Second):
	case <-stopCh:
		return
	}

	for {
		if !snap.cfg.Enabled || !snap.cfg.SendLocation {
			select {
			case <-time.After(30 * time.Second):
				continue
			case <-stopCh:
				return
			}
		}

		intervalMin := snap.cfg.IntervalMin
		if intervalMin < 5 {
			intervalMin = 5
		}
		if intervalMin > 180 {
			intervalMin = 180
		}
		interval := time.Duration(intervalMin) * time.Minute

		// Wait until next scheduled beacon based on LastBeaconAt.
		if snap.cfg.LastBeaconAt != "" {
			last, err := time.Parse(time.RFC3339, snap.cfg.LastBeaconAt)
			if err == nil {
				elapsed := time.Since(last)
				if elapsed < interval {
					remaining := interval - elapsed
					applog.Debug("APRS: beacon waiting", "remaining", remaining.Round(time.Second), "lastBeacon", snap.cfg.LastBeaconAt)
					select {
					case <-time.After(remaining):
					case <-stopCh:
						return
					}
				}
			}
		}

		if a.sendAPRSBeacon(&snap.cfg, snap.callsign, snap.client) {
			// Local scheduling state only — the live config is updated by
			// the owner when the event below is processed.
			snap.cfg.LastBeaconAt = time.Now().UTC().Format(time.RFC3339)
			select {
			case a.aprsEvents <- aprsEvent{kind: aprsEvBeacon, gen: snap.gen, callsign: snap.callsign}:
			default:
			}
		}

		// Wait for next interval.
		select {
		case <-time.After(interval):
		case <-stopCh:
			return
		}
	}
}

func (a *App) stopAPRSBeacon() {
	if a.beaconStopCh != nil {
		close(a.beaconStopCh)
		a.beaconStopCh = nil
		a.aprsBeaconWG.Wait() // join before its resources can be replaced
		applog.Debug("APRS: beacon goroutine stopped")
	}
}

// sendAPRSBeacon assembles and transmits one position beacon from the given
// snapshot values. The grid comes from the owner-refreshed cache, never from
// live config. Returns true when the beacon was actually transmitted.
func (a *App) sendAPRSBeacon(cfg *config.APRSConfig, callsign string, client aprs.Client) bool {
	// Guard against nil/disconnected client — the worker's snapshot may
	// predate a client replacement.
	if client == nil || !client.IsConnected() {
		applog.Debug("APRS: beacon skipped — not connected")
		return false
	}

	grid := a.beaconGridSafe()
	if grid == "" {
		applog.Debug("APRS: beacon skipped — no station grid")
		return false
	}
	lat, lon, err := geo.GridToLatLon(grid)
	if err != nil {
		applog.Debug("APRS: beacon skipped — grid error", "error", err)
		return false
	}

	symbol := cfg.Symbol
	if symbol == "" {
		symbol = "/-"
	}

	switch c := client.(type) {
	case *aprs.TCPClient:
		if err := c.SendPosition(callsign, lat, lon, symbol, cfg.Comment); err != nil {
			applog.Warn("APRS: beacon failed", "error", err)
			return false
		}
	case *aprs.KISSClient:
		if err := c.SendPosition(callsign, lat, lon, symbol, cfg.Comment); err != nil {
			applog.Warn("KISS: beacon failed", "error", err)
			return false
		}
	case *aprs.KISSServerClient:
		if err := c.SendPosition(callsign, lat, lon, symbol, cfg.Comment); err != nil {
			applog.Warn("KISS server: beacon failed", "error", err)
			return false
		}
	default:
		applog.Debug("APRS: beacon skipped — unsupported client")
		return false
	}
	return true
}

// SendAPRSBeaconNow transmits the station position immediately — the manual
// beacon shortcut. Runs on the owner goroutine, so the beacon event is
// handled inline (toast callback + timestamp persist). Returns an error when
// TX is not configured or the client is not connected.
func (a *App) SendAPRSBeaconNow() error {
	aprsCfg := a.Logbook.APRS
	if aprsCfg == nil || !aprsCfg.Enabled || !aprsCfg.SendLocation {
		return fmt.Errorf("beaconing is not configured — enable Send Location in the logbook APRS settings")
	}
	if a.APRSClient == nil || !a.APRSClient.IsConnected() {
		return fmt.Errorf("not connected")
	}
	a.refreshBeaconGrid()
	callsign := a.aprsBeaconCallsign(aprsCfg)
	if a.sendAPRSBeacon(aprsCfg, callsign, a.APRSClient) {
		a.handleAPRSEvent(aprsEvent{kind: aprsEvBeacon, gen: a.aprsGen, callsign: callsign})
	}
	return nil
}

func (a *App) SwitchLogbook(name string) error {
	if _, ok := a.Config.Logbooks[name]; !ok {
		return fmt.Errorf("logbook %q not found", name)
	}

	// Open and validate the replacement FIRST. On failure the current
	// database stays open and active — a failed switch must never leave
	// the running logbook closed.
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

	// Stop and JOIN APRS workers before replacing the logbook so the old
	// beacon/pruner can never read the new logbook's config.
	a.stopAPRSPruner()
	a.stopAPRSBeacon()

	// Commit the state transition, then retire the previous database
	// (deferred until its background holders finish — see KeepDBAlive).
	if a.DB != nil {
		a.retireDB(a.DB)
	}
	a.Config.State.ActiveLogbook = name
	a.LogbookName = name
	a.Logbook = &lb
	a.DB = db
	a.DBPath = dbPath
	a.refreshBeaconGrid()

	// Persist the active logbook choice so it survives restarts.
	if err := config.Save(a.ConfigPath, a.Config); err != nil {
		applog.Warn("Failed to save active logbook", "error", err)
	}

	// Restart APRS for the new logbook config (debounced — won't fire
	// until rapid switching settles).
	a.ScheduleAPRSRestart()

	return nil
}

// dbHolder tracks background operations that reference a logbook database
// after it was replaced by SwitchLogbook. A retired database is closed only
// when the last holder releases it.
type dbHolder struct {
	db      *sql.DB
	refs    int
	retired bool
}

// KeepDBAlive marks db as in use by a background operation. The returned
// release func must be called exactly once when the operation stops using db.
// SwitchLogbook retires replaced databases instead of closing them outright,
// so commands that captured the old logbook can finish without touching or
// corrupting the new one.
func (a *App) KeepDBAlive(db *sql.DB) func() {
	if db == nil {
		return func() {}
	}
	a.dbMu.Lock()
	if a.dbHolders == nil {
		a.dbHolders = make(map[*sql.DB]*dbHolder)
	}
	h := a.dbHolders[db]
	if h == nil {
		h = &dbHolder{db: db}
		a.dbHolders[db] = h
	}
	h.refs++
	a.dbMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			a.dbMu.Lock()
			h.refs--
			if h.refs == 0 {
				delete(a.dbHolders, db)
				if h.retired {
					h.db.Close()
				}
			}
			a.dbMu.Unlock()
		})
	}
}

// retireDB closes db unless background operations still hold it via
// KeepDBAlive — in that case the close is deferred to the last release.
func (a *App) retireDB(db *sql.DB) {
	a.dbMu.Lock()
	defer a.dbMu.Unlock()
	if h := a.dbHolders[db]; h != nil {
		h.retired = true
		if h.refs == 0 {
			delete(a.dbHolders, db)
			db.Close()
		}
		return
	}
	db.Close()
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

// SetGPSGrid is called by the TUI model when GPS position updates. It also
// refreshes the cached beacon grid, so APRS beacon workers follow GPS
// movement without ever reading live config.
func (a *App) SetGPSGrid(grid string, hasFix bool) {
	a.gpsMu.Lock()
	a.gpsGrid = grid
	a.gpsHasFix = hasFix
	a.beaconGrid = a.effectiveGridUnlocked()
	a.gpsMu.Unlock()
}

// refreshBeaconGrid recomputes the cached grid used by APRS beacon workers.
// Runs on the owner goroutine (RunPendingAPRS, SwitchLogbook, manual beacon);
// workers only read the cached value under gpsMu.
func (a *App) refreshBeaconGrid() {
	a.gpsMu.Lock()
	a.beaconGrid = a.effectiveGridUnlocked()
	a.gpsMu.Unlock()
}

// beaconGridSafe returns the cached effective grid for beacon workers.
// Safe to call from any goroutine.
func (a *App) beaconGridSafe() string {
	a.gpsMu.RLock()
	defer a.gpsMu.RUnlock()
	return a.beaconGrid
}

// EffectiveGrid returns the GPS-derived grid when GPS is enabled, has a fix,
// and the logbook has gps_grid enabled. Falls back to the configured station
// grid otherwise. The grid is truncated to the configured GPS precision
// (6, 8, or 10 chars) to avoid leaking more-accurate position data than
// the user intended. Call on the owner goroutine.
func (a *App) EffectiveGrid() string {
	a.gpsMu.RLock()
	defer a.gpsMu.RUnlock()
	return a.effectiveGridUnlocked()
}

// effectiveGridUnlocked computes the effective grid. Caller must hold gpsMu;
// it reads owner-owned config fields, so call only from the owner goroutine.
func (a *App) effectiveGridUnlocked() string {
	var raw string
	if a.Config != nil && a.Config.Integrations.GPS.Enabled && a.gpsHasFix && a.gpsGrid != "" &&
		a.Logbook != nil && a.Logbook.Station.GPSGrid {
		raw = a.gpsGrid
	} else if a.Logbook != nil {
		raw = strings.TrimSpace(strings.ToUpper(a.Logbook.Station.Grid))
	}
	if raw == "" {
		return ""
	}
	// Truncate to configured GPS grid precision.
	prec := 10
	if a.Config != nil {
		if p := a.Config.Integrations.GPS.GridPrecision; p == 6 || p == 8 {
			prec = p
		}
	}
	if len(raw) > prec {
		raw = raw[:prec]
	}
	return raw
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
