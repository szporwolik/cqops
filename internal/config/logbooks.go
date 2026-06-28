package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/szporwolik/cqops/internal/secrets"
	"github.com/szporwolik/cqops/internal/version"
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

	// Run version-gated upgrade steps before anything else touches config.
	cfg.Upgrade(version.Resolved())

	// Load encrypted secrets (passwords, API keys) and overlay them onto
	// the config before validation. Without this, Validate() would reject
	// the config because QRZ pass / Wavelog API key appear empty in YAML.
	if sec, err := secrets.Load(configDir); err == nil {
		cfg.SetSecretsStore(sec)
		cfg.ApplySecrets()
	}

	// Populate in-memory ID fields from map keys (id is not serialized).
	PopulateIDs(cfg)
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
	id := cfg.State.ActiveLogbook
	if cliFlag != "" {
		// cliFlag could be an ID or a callsign — try ID first, then search by callsign.
		if lb, ok := cfg.Logbooks[cliFlag]; ok {
			return cliFlag, &lb, nil
		}
		if foundID, foundLB, ok := FindLogbookByCallsign(cfg, cliFlag); ok {
			return foundID, foundLB, nil
		}
		return "", nil, fmt.Errorf("logbook %q not found", cliFlag)
	}
	if env := os.Getenv("CQOPS_LOGBOOK"); env != "" && cliFlag == "" {
		if lb, ok := cfg.Logbooks[env]; ok {
			return env, &lb, nil
		}
		if foundID, foundLB, ok := FindLogbookByCallsign(cfg, env); ok {
			return foundID, foundLB, nil
		}
	}
	if id == "" {
		return "", nil, fmt.Errorf("no active logbook set")
	}

	lb, ok := cfg.Logbooks[id]
	if !ok {
		// Active logbook ID not found — try to find any logbook.
		for firstID, firstLB := range cfg.Logbooks {
			cfg.State.ActiveLogbook = firstID
			return firstID, &firstLB, nil
		}
		return "", nil, fmt.Errorf("active logbook %q not found and no logbooks configured", id)
	}

	return id, &lb, nil
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
	// First run: exactly one logbook with no callsign, operator, or grid set.
	// Handles edge cases: nil/empty logbook map, missing station data.
	if cfg == nil || len(cfg.Logbooks) == 0 {
		return true
	}
	if len(cfg.Logbooks) != 1 {
		return false
	}
	for _, lb := range cfg.Logbooks {
		return lb.Station.Callsign == "" && lb.Station.Grid == ""
	}
	return true
}
