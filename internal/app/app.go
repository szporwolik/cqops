package app

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

type App struct {
	Config      *config.Config
	ConfigPath  string
	LogbookName string
	Logbook     *config.Logbook
	DB          *sql.DB
	DBPath      string
}

func Init(logbookFlag string) (*App, error) {
	cfg, configPath, err := config.EnsureConfig()
	if err != nil {
		return nil, fmt.Errorf("ensure config: %w", err)
	}

	name, lb, err := config.ResolveLogbook(cfg, logbookFlag)
	if err != nil {
		return nil, fmt.Errorf("resolve logbook: %w", err)
	}

	dbPath, err := config.DBPath(name, lb)
	if err != nil {
		return nil, fmt.Errorf("db path: %w", err)
	}

	db, err := store.InitDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	return &App{
		Config:      cfg,
		ConfigPath:  configPath,
		LogbookName: name,
		Logbook:     lb,
		DB:          db,
		DBPath:      dbPath,
	}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}

func (a *App) SwitchLogbook(name string) error {
	if _, ok := a.Config.Logbooks[name]; !ok {
		return fmt.Errorf("logbook %q not found", name)
	}

	if a.DB != nil {
		a.DB.Close()
	}

	lbVal := a.Config.Logbooks[name]
	dbPath, err := config.DBPath(name, &lbVal)
	if err != nil {
		return fmt.Errorf("db path: %w", err)
	}

	db, err := store.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	a.Config.ActiveLogbook = name
	lb := a.Config.Logbooks[name]
	a.LogbookName = name
	a.Logbook = &lb
	a.DB = db
	a.DBPath = dbPath

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
