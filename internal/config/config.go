package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/szporwolik/cqops/internal/qso"
)

type Config struct {
	General  GeneralConfig        `yaml:"general"`
	State    StateConfig          `yaml:"state"`
	QRZ      QRZConfig            `yaml:"qrz,omitempty"`
	Logbooks map[string]Logbook   `yaml:"logbooks"`
	Rigs     map[string]RigPreset `yaml:"rigs,omitempty"`
	Contests map[string]Contest   `yaml:"contests,omitempty"`
	WSJTX    WSJTXConfig          `yaml:"wsjtx,omitempty"`
	DXC      DXCConfig            `yaml:"dxc,omitempty"`
}

type DXCConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host,omitempty"`
	Port    string `yaml:"port,omitempty"`
	Login   string `yaml:"login,omitempty"`
}

type GeneralConfig struct {
	Timezone         string              `yaml:"timezone"`
	DistanceUnit     string              `yaml:"distance_unit,omitempty"`
	RenderMap        bool                `yaml:"render_map,omitempty"`
	DrawGrayline     bool                `yaml:"draw_grayline,omitempty"`
	PictureAtQRZPane bool                `yaml:"picture_at_qrz_pane,omitempty"`
	SolarAtQSOPane   bool                `yaml:"solar_at_qso_pane,omitempty"`
	UseCTY           bool                `yaml:"use_cty,omitempty"` // CTY.DAT DXCC country file
	UseSCP           bool                `yaml:"use_scp,omitempty"` // Super Check Partial callsign database
	UseRef           bool                `yaml:"use_ref,omitempty"` // REF database
	Notifications    NotificationsConfig `yaml:"notifications"`
}

type NotificationsConfig struct {
	Enabled       bool `yaml:"enabled"`
	QSO           bool `yaml:"qso_logged"`
	Wavelog       bool `yaml:"wavelog_sent"`
	WavelogErrors bool `yaml:"wavelog_errors"`
	BeepOnError   bool `yaml:"beep_on_error"`
}

type StateConfig struct {
	ActiveLogbook   string `yaml:"active_logbook"`
	ActiveContest   string `yaml:"active_contest,omitempty"`
	RetainComment   bool   `yaml:"retain_comment,omitempty"`
	RetainedComment string `yaml:"retained_comment,omitempty"`
}

type QRZConfig struct {
	Enabled bool   `yaml:"enabled"`
	User    string `yaml:"user,omitempty"`
	Pass    string `yaml:"pass,omitempty"`
}

type Logbook struct {
	ID           string         `yaml:"-"`
	Description  string         `yaml:"description"`
	DatabasePath string         `yaml:"database_path,omitempty"`
	Station      Station        `yaml:"station"`
	ADIF         ADIFConfig     `yaml:"adif,omitempty"`
	Wavelog      *WavelogConfig `yaml:"wavelog,omitempty"`
}

type Station struct {
	Callsign   string `yaml:"callsign"`
	Operator   string `yaml:"operator"`
	Grid       string `yaml:"grid"`
	RigName    string `yaml:"rig_name,omitempty"`
	SOTARef    string `yaml:"sota_ref,omitempty"`
	POTARef    string `yaml:"pota_ref,omitempty"`
	WWFFRef    string `yaml:"wwff_ref,omitempty"`
	IARURegion int    `yaml:"iaru_region,omitempty"` // 1, 2, or 3
	CQZone     int    `yaml:"cq_zone,omitempty"`     // station CQ zone (1-40)
	ITUZone    int    `yaml:"itu_zone,omitempty"`    // station ITU zone (1-90)
	DXCC       int    `yaml:"dxcc,omitempty"`        // station DXCC entity number
	SIG        string `yaml:"sig,omitempty"`         // station Special Interest Group (e.g. SOTA, POTA)
	SIGInfo    string `yaml:"sig_info,omitempty"`    // station SIG info (e.g. summit/park reference)
}

// Contest represents a contest configuration.
type Contest struct {
	ID                  string `yaml:"-"`
	Name                string `yaml:"name"`
	Date                string `yaml:"contest_date,omitempty"`
	NextQSO             int    `yaml:"next_qso,omitempty"`
	ContestID           string `yaml:"contest_id,omitempty"`
	ContestIDName       string `yaml:"contest_id_name,omitempty"`
	PrefillExchange     bool   `yaml:"prefill_exchange,omitempty"`
	ExchangeSent        string `yaml:"exchange_sent,omitempty"`
	PrefillExchangeRcvd bool   `yaml:"prefill_exchange_rcvd,omitempty"`
	ExchangeRcvd        string `yaml:"exchange_rcvd,omitempty"`
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
	LastFetchedID    int64  `yaml:"last_fetched_id,omitempty"`
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
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Validate checks the config for structural integrity. Returns an error
// describing the first problem found, or nil if the config is valid.
// Validate checks the config for structural integrity and safe values.
// Returns an error describing the first problem found, or nil if valid.
func (c *Config) Validate() error {
	if len(c.Logbooks) == 0 {
		return fmt.Errorf("no logbooks configured")
	}
	if c.State.ActiveLogbook == "" {
		return fmt.Errorf("no active logbook set")
	}
	lb, ok := c.Logbooks[c.State.ActiveLogbook]
	if !ok {
		return fmt.Errorf("active logbook %q not found in logbooks", c.State.ActiveLogbook)
	}

	// --- General settings ---
	tz := strings.TrimSpace(c.General.Timezone)
	if tz == "" {
		return fmt.Errorf("general.timezone is required")
	}
	switch c.General.DistanceUnit {
	case "", "km", "mi":
	default:
		return fmt.Errorf("general.distance_unit must be 'km' or 'mi', got %q", c.General.DistanceUnit)
	}

	// --- Active logbook station ---
	call := strings.TrimSpace(lb.Station.Callsign)
	if call != "" && !qso.IsValidCall(call) {
		return fmt.Errorf("station callsign %q is not a valid callsign", call)
	}
	grid := strings.TrimSpace(lb.Station.Grid)
	if grid != "" && !qso.IsValidLocator(grid) {
		return fmt.Errorf("station grid %q is not a valid Maidenhead locator", grid)
	}

	// --- QRZ ---
	if c.QRZ.Enabled {
		if strings.TrimSpace(c.QRZ.User) == "" {
			return fmt.Errorf("qrz.user is required when qrz.enabled is true")
		}
		if strings.TrimSpace(c.QRZ.Pass) == "" {
			return fmt.Errorf("qrz.pass is required when qrz.enabled is true")
		}
	}

	// --- Wavelog ---
	if lb.Wavelog != nil && lb.Wavelog.Enabled {
		wl := lb.Wavelog
		if strings.TrimSpace(wl.URL) == "" {
			return fmt.Errorf("wavelog.url is required when enabled")
		}
		u, err := url.Parse(wl.URL)
		if err != nil {
			return fmt.Errorf("wavelog.url %q is not a valid URL: %w", wl.URL, err)
		}
		if u.Scheme != "https" {
			return fmt.Errorf("wavelog.url must use HTTPS, got %q", wl.URL)
		}
		if strings.TrimSpace(wl.APIKey) == "" {
			return fmt.Errorf("wavelog.api_key is required when enabled")
		}
	}

	// --- WSJT-X ---
	if c.WSJTX.Enabled {
		if strings.TrimSpace(c.WSJTX.UDPHost) == "" {
			return fmt.Errorf("wsjtx.udp_host is required when enabled")
		}
		if c.WSJTX.UDPPort < 1 || c.WSJTX.UDPPort > 65535 {
			return fmt.Errorf("wsjtx.udp_port must be 1-65535, got %d", c.WSJTX.UDPPort)
		}
	}

	// --- DXC ---
	if c.DXC.Enabled {
		if strings.TrimSpace(c.DXC.Host) == "" {
			return fmt.Errorf("dxc.host is required when enabled")
		}
		if c.DXC.Port != "" {
			if p, err := strconv.Atoi(c.DXC.Port); err != nil || p < 1 || p > 65535 {
				return fmt.Errorf("dxc.port must be 1-65535, got %q", c.DXC.Port)
			}
		}
	}

	// --- Rigs ---
	for id, rig := range c.Rigs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("rig entry with empty id")
		}
		if rig.FlrigEnabled {
			if strings.TrimSpace(rig.FlrigHost) == "" {
				return fmt.Errorf("rig %q: flrig_host is required when flrig_enabled", id)
			}
			if rig.FlrigPort != "" {
				if p, err := strconv.Atoi(rig.FlrigPort); err != nil || p < 1 || p > 65535 {
					return fmt.Errorf("rig %q: flrig_port must be 1-65535, got %q", id, rig.FlrigPort)
				}
			}
		}
	}

	return nil
}
