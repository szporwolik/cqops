package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureConfig() (*Config, string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return nil, "", fmt.Errorf("config dir: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, "", fmt.Errorf("create config dir: %w", err)
	}

	configPath, err := ConfigPath()
	if err != nil {
		return nil, "", fmt.Errorf("config path: %w", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		cfg = DefaultConfig()
		if saveErr := Save(configPath, cfg); saveErr != nil {
			return nil, "", fmt.Errorf("save default config: %w", saveErr)
		}
	}

	if cfg.Logbooks == nil {
		cfg.Logbooks = make(map[string]Logbook)
	}

	// Validate structural integrity.
	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("config is corrupted: %w", err)
	}

	return cfg, configPath, nil
}

func ResolveLogbook(cfg *Config, cliFlag string) (string, *Logbook, error) {
	name := cfg.State.ActiveLogbook
	if cliFlag != "" {
		name = cliFlag
	}
	if env := os.Getenv("CQOPS_LOGBOOK"); env != "" && cliFlag == "" {
		name = env
	}
	if name == "" {
		name = "default"
	}

	lb, ok := cfg.Logbooks[name]
	if !ok {
		lb = Logbook{}
		cfg.Logbooks[name] = lb
	}

	return name, &lb, nil
}

func DBPath(logbookName string, lb *Logbook) (string, error) {
	if lb.DatabasePath != "" {
		return lb.DatabasePath, nil
	}

	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}

	return filepath.Join(dataDir, logbookName+".db"), nil
}

func IsFirstRun(cfg *Config) bool {
	if cfg.State.ActiveLogbook != "default" {
		return false
	}
	lb, ok := cfg.Logbooks["default"]
	if !ok {
		return true
	}
	return lb.Station.Callsign == "" && lb.Station.Operator == "" && lb.Station.Grid == ""
}
