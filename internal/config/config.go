package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	General  GeneralConfig          `yaml:"general"`
	State    StateConfig            `yaml:"state"`
	QRZ      QRZConfig              `yaml:"qrz,omitempty"`
	Logbooks map[string]Logbook     `yaml:"logbooks"`
	Rigs     map[string]RigPreset   `yaml:"rigs,omitempty"`
	WSJTX    WSJTXConfig            `yaml:"wsjtx,omitempty"`
}

type GeneralConfig struct {
	Timezone      string              `yaml:"timezone"`
	DistanceUnit  string              `yaml:"distance_unit,omitempty"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

type NotificationsConfig struct {
	Enabled       bool `yaml:"enabled"`
	QSO           bool `yaml:"qso_logged"`
	Wavelog       bool `yaml:"wavelog_sent"`
	WavelogErrors bool `yaml:"wavelog_errors"`
}

type StateConfig struct {
	ActiveLogbook string `yaml:"active_logbook"`
}

type QRZConfig struct {
	Enabled bool   `yaml:"enabled"`
	User    string `yaml:"user,omitempty"`
	Pass    string `yaml:"pass,omitempty"`
}

type Logbook struct {
	ID           string          `yaml:"-"`
	Description  string          `yaml:"description"`
	DatabasePath string          `yaml:"database_path,omitempty"`
	Station      Station         `yaml:"station"`
	ADIF         ADIFConfig      `yaml:"adif,omitempty"`
	Wavelog      *WavelogConfig  `yaml:"wavelog,omitempty"`
}

type Station struct {
	Callsign string `yaml:"callsign"`
	Operator string `yaml:"operator"`
	Grid     string `yaml:"grid"`
	RigName  string `yaml:"rig_name,omitempty"`
	SOTARef  string `yaml:"sota_ref,omitempty"`
	POTARef  string `yaml:"pota_ref,omitempty"`
	WWFFRef  string `yaml:"wwff_ref,omitempty"`
}

// Rig resolves the RigPreset referenced by RigName. Returns the preset and
// true if found, or zero value and false.
func (s Station) Rig(rgs map[string]RigPreset) (RigPreset, bool) {
	if s.RigName == "" {
		return RigPreset{}, false
	}
	rp, ok := rgs[s.RigName]
	return rp, ok
}

// RigModel returns the rig model from the referenced preset, or "".
func (s Station) RigModel(rgs map[string]RigPreset) string {
	rp, ok := s.Rig(rgs)
	if !ok {
		return ""
	}
	return rp.Model
}

// RigAntenna returns the antenna from the referenced preset, or "".
func (s Station) RigAntenna(rgs map[string]RigPreset) string {
	rp, ok := s.Rig(rgs)
	if !ok {
		return ""
	}
	return rp.Antenna
}

// RigPower returns the power from the referenced preset, or "".
func (s Station) RigPower(rgs map[string]RigPreset) string {
	rp, ok := s.Rig(rgs)
	if !ok {
		return ""
	}
	return rp.Power
}

// RigFlrig returns the flrig settings from the referenced preset.
func (s Station) RigFlrig(rgs map[string]RigPreset) (enabled bool, host, port string) {
	rp, ok := s.Rig(rgs)
	if !ok {
		return false, "localhost", "12345"
	}
	return rp.FlrigEnabled, rp.FlrigHost, rp.FlrigPort
}

type RigPreset struct {
	ID           string `yaml:"-"`
	Model        string `yaml:"model"`
	Antenna      string `yaml:"antenna"`
	Power        string `yaml:"power"`
	FlrigEnabled bool   `yaml:"flrig_enabled"`
	FlrigHost    string `yaml:"flrig_host"`
	FlrigPort    string `yaml:"flrig_port"`
}

type ADIFConfig struct {
	DefaultExportPath string `yaml:"default_export_path"`
}

type RigConfig struct {
	Provider     string `yaml:"provider"`
	AutoFill     bool   `yaml:"auto_fill"`
	FailSilently bool   `yaml:"fail_silently"`

	Flrig struct {
		Enabled   bool   `yaml:"enabled"`
		URL       string `yaml:"url"`
		TimeoutMS int    `yaml:"timeout_ms"`
	} `yaml:"flrig"`

	Rigctld struct {
		Enabled   bool   `yaml:"enabled"`
		Host      string `yaml:"host"`
		Port      int    `yaml:"port"`
		TimeoutMS int    `yaml:"timeout_ms"`
	} `yaml:"rigctld,omitempty"`
}

type WavelogConfig struct {
	Enabled          bool   `yaml:"enabled"`
	URL              string `yaml:"url"`
	APIKey           string `yaml:"api_key"`
	StationProfileID string `yaml:"station_profile_id"`
}

type WSJTXConfig struct {
	Enabled bool   `yaml:"enabled"`
	UDPHost string `yaml:"udp_host"`
	UDPPort int    `yaml:"udp_port"`
}

// Load reads and parses a YAML configuration file from path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// Save marshals cfg as YAML and writes it to path.

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Validate checks the config for structural integrity. Returns an error
// describing the first problem found, or nil if the config is valid.
func (c *Config) Validate() error {
	if len(c.Logbooks) == 0 {
		return fmt.Errorf("no logbooks configured")
	}
	if c.State.ActiveLogbook == "" {
		return fmt.Errorf("no active logbook set")
	}
	if _, ok := c.Logbooks[c.State.ActiveLogbook]; !ok {
		return fmt.Errorf("active logbook %q not found in logbooks", c.State.ActiveLogbook)
	}
	return nil
}
