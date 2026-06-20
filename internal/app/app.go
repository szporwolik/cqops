package app

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ref"
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

	// lastWSJTX tracks the effective WSJT-X config last applied to the
	// listener. Used to avoid unnecessary Stop/Start cycles when config
	// is saved but the WSJT-X settings haven't changed.
	lastWSJTX struct {
		enabled bool
		host    string
		port    int
	}
}

func Init(logbookFlag string) (*App, error) {
	cfg, configPath, err := config.EnsureConfig()
	if err != nil {
		applog.Error("Config is corrupted or missing — cannot start", "error", err.Error())
		return nil, fmt.Errorf("config: %w", err)
	}
	applog.Info("Config OK", "path", configPath)

	name, lb, err := config.ResolveLogbook(cfg, logbookFlag)
	if err != nil {
		applog.Error("Cannot resolve logbook", "name", logbookFlag, "error", err.Error())
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
		WSJTXUpdated: make(chan struct{}, 1),
	}

	app.MaybeRestartWSJTX()

	// Load cached data files — download/update happens later in the TUI
	// tick after internet availability is confirmed.
	if app.Config.General.UseCTY {
		cacheDir, _ := config.CacheDir()
		ctyPath := filepath.Join(cacheDir, "cty.dat")
		if prefixes, err := dxcc.LoadLocal(ctyPath); err == nil {
			app.DXCC = prefixes
			applog.Info("DXCC: prefix data loaded from cache")
		} else {
			applog.Info("DXCC: no cached data yet — will fetch when online")
		}
	}
	if app.Config.General.UseSCP {
		cacheDir, _ := config.CacheDir()
		scpPath := filepath.Join(cacheDir, "MASTER.SCP")
		if db, err := scp.LoadLocal(scpPath); err == nil {
			app.SCP = db
			applog.Info("SCP: callsign database loaded from cache")
		} else {
			applog.Info("SCP: no cached data yet — will fetch when online")
		}
	}
	if app.Config.General.UseRef {
		cacheDir, _ := config.CacheDir()
		refPath := filepath.Join(cacheDir, "ref.db")
		if rdb, err := ref.Open(refPath); err == nil {
			app.RefDB = rdb
			applog.Info("REF: database opened")
		} else {
			applog.Info("REF: cannot open database — will rebuild when online")
		}
	}

	return app, nil
}

func (a *App) Close() {
	applog.Info("Shutting down — stopping WSJT-X listener")
	a.WSJTX.Stop()
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
// This avoids leaking UDP goroutines/sockets from unnecessary restarts.
func (a *App) MaybeRestartWSJTX() {
	cur := a.Config.WSJTX

	// If nothing changed, skip the restart entirely.
	if cur.Enabled == a.lastWSJTX.enabled &&
		cur.UDPHost == a.lastWSJTX.host &&
		cur.UDPPort == a.lastWSJTX.port {
		return
	}

	a.WSJTX.Stop()
	if cur.Enabled {
		if err := a.WSJTX.Start(cur.UDPHost, cur.UDPPort); err != nil {
			applog.Error("WSJT-X restart failed", "error", err.Error())
			// Don't update last-applied on failure — next call will retry.
			return
		}
	}

	// Record as applied only after successful start or stop.
	a.lastWSJTX.enabled = cur.Enabled
	a.lastWSJTX.host = cur.UDPHost
	a.lastWSJTX.port = cur.UDPPort

	select {
	case a.WSJTXUpdated <- struct{}{}:
	default:
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

	a.Config.State.ActiveLogbook = name
	a.LogbookName = name
	a.Logbook = &lb
	a.DB = db
	a.DBPath = dbPath

	// Persist the active logbook choice so it survives restarts.
	if err := config.Save(a.ConfigPath, a.Config); err != nil {
		applog.Warn("Failed to save active logbook", "error", err)
	}

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
	if s.Operator != "" && s.Operator != s.Callsign {
		parts = append(parts, "op:"+s.Operator)
	}
	return strings.Join(parts, " ")
}

// LogbookDisplayName returns the human-readable name for the active logbook.
func (a *App) LogbookDisplayName() string {
	return config.LogbookDisplayName(a.Logbook)
}
