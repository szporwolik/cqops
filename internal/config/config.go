package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/version"
)

type Config struct {
	General           GeneralConfig        `yaml:"general"`
	State             StateConfig          `yaml:"state"`
	Integrations      IntegrationsConfig   `yaml:"integrations,omitempty"`
	Favorites         map[int]Favorite     `yaml:"favorites,omitempty"`
	Logbooks          map[string]Logbook   `yaml:"logbooks"`
	Rigs              map[string]RigPreset `yaml:"rigs,omitempty"`
	Contests          map[string]Contest   `yaml:"contests,omitempty"`
	Operators         map[string]Operator  `yaml:"operators,omitempty"`
	BroadcastStations []BroadcastStation   `yaml:"-"`
}

// BroadcastStation represents a broadcast radio station preset.
type BroadcastStation struct {
	Radio        string `yaml:"radio"`
	Country      string `yaml:"country"`
	FrequencyKHz int    `yaml:"frequency_khz"`
}

// BroadcastBand returns "LW", "MW", or "SW" based on the frequency in kHz.
func (bs BroadcastStation) BroadcastBand() string {
	if bs.FrequencyKHz < 300 {
		return "LW"
	}
	if bs.FrequencyKHz < 1700 {
		return "MW"
	}
	return "SW"
}

type IntegrationsConfig struct {
	DXC DXCConfig `yaml:"dxc,omitempty"`
	QRZ QRZConfig `yaml:"qrz,omitempty"`
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
	Debug            bool                `yaml:"debug,omitempty"`   // verbose debug logging
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
	RetainComment   bool   `yaml:"retain_comment,omitempty"`
	RetainedComment string `yaml:"retained_comment,omitempty"`
	Version         string `yaml:"version,omitempty"`
}

type QRZConfig struct {
	Enabled bool   `yaml:"enabled"`
	User    string `yaml:"user,omitempty"`
	Pass    string `yaml:"pass,omitempty"`
}

// Favorite stores a mode/freq/submode/band snapshot for quick recall.
// Slots are 0-9, stored under alt+shift+N and recalled with alt+N.
type Favorite struct {
	Mode    string  `yaml:"mode,omitempty"`
	Freq    float64 `yaml:"freq,omitempty"`
	Submode string  `yaml:"submode,omitempty"`
	Band    string  `yaml:"band,omitempty"`
}

type Logbook struct {
	ID             string         `yaml:"-"`
	Name           string         `yaml:"name"`
	ActiveContest  string         `yaml:"active_contest,omitempty"`
	ActiveOperator string         `yaml:"active_operator,omitempty"`
	DatabasePath   string         `yaml:"database_path,omitempty"`
	Station        Station        `yaml:"station"`
	ADIF           ADIFConfig     `yaml:"adif,omitempty"`
	Wavelog        *WavelogConfig `yaml:"wavelog,omitempty"`
}

type Station struct {
	Callsign   string `yaml:"callsign"`
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
	Continent  string `yaml:"continent,omitempty"`   // station continent (EU, NA, SA, AS, AF, OC, AN)
}

// Contest represents a contest configuration.
type Contest struct {
	ID                  string `yaml:"-"`
	LogbookID           string `yaml:"logbook_id,omitempty"`
	Name                string `yaml:"name"`
	Date                string `yaml:"contest_date,omitempty"`
	NextQSO             int    `yaml:"next_qso,omitempty"`
	ContestID           string `yaml:"contest_id,omitempty"`
	ContestIDName       string `yaml:"contest_id_name,omitempty"`
	PrefillExchange     bool   `yaml:"prefill_exchange,omitempty"`
	ExchangeSent        string `yaml:"exchange_sent,omitempty"`
	PrefillExchangeRcvd bool   `yaml:"prefill_exchange_rcvd,omitempty"`
	ExchangeRcvd        string `yaml:"exchange_rcvd,omitempty"`
	InUse               *bool  `yaml:"in_use,omitempty"` // nil or true = in use, false = excluded from cycling
}

// Operator represents a multi-operator station callsign profile.
type Operator struct {
	ID       string `yaml:"-"`
	Callsign string `yaml:"callsign"`
	Name     string `yaml:"name,omitempty"`
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
	return rp.RadioBackend == "flrig", rp.FlrigHost, rp.FlrigPort
}

// RigHamlib returns the hamlib settings from the referenced preset.
func (s Station) RigHamlib(rgs map[string]RigPreset) (enabled bool, host, port string) {
	rp, ok := s.Rig(rgs)
	if !ok {
		return false, "127.0.0.1", "4532"
	}
	return rp.RadioBackend == "hamlib", rp.HamlibRadioHost, rp.HamlibRadioPort
}

// RigRotor returns the rotor settings from the referenced preset.
func (s Station) RigRotor(rgs map[string]RigPreset) (enabled bool, host, port string) {
	rp, ok := s.Rig(rgs)
	if !ok {
		return false, "127.0.0.1", "4533"
	}
	return rp.RotorBackend == "hamlib", rp.RotorHamlibHost, rp.RotorHamlibPort
}

type RigPreset struct {
	ID              string `yaml:"-"`
	Name            string `yaml:"name,omitempty"`
	Model           string `yaml:"model"`
	Antenna         string `yaml:"antenna"`
	Power           string `yaml:"power"`
	RadioBackend    string `yaml:"radio_backend,omitempty"` // "" | "flrig" | "hamlib"
	Backend         string `yaml:"backend,omitempty"`       // deprecated — migrated to RadioBackend
	FlrigEnabled    bool   `yaml:"flrig_enabled,omitempty"`
	FlrigHost       string `yaml:"flrig_host,omitempty"`
	FlrigPort       string `yaml:"flrig_port,omitempty"`
	HamlibRadioHost string `yaml:"hamlib_radio_host,omitempty"`
	HamlibRadioPort string `yaml:"hamlib_radio_port,omitempty"`
	RotorBackend    string `yaml:"rotor_backend,omitempty"` // "" | "hamlib"
	RotorHamlibHost string `yaml:"rotor_hamlib_host,omitempty"`
	RotorHamlibPort string `yaml:"rotor_hamlib_port,omitempty"`
	WsjtxEnabled    bool   `yaml:"wsjtx_enabled,omitempty"`
	WsjtxUDPHost    string `yaml:"wsjtx_udp_host,omitempty"`
	WsjtxUDPPort    int    `yaml:"wsjtx_udp_port,omitempty"`
}

type ADIFConfig struct {
	DefaultExportPath string `yaml:"default_export_path"`
}

type WavelogConfig struct {
	Enabled          bool   `yaml:"enabled"`
	URL              string `yaml:"url"`
	APIKey           string `yaml:"api_key"`
	StationProfileID string `yaml:"station_profile_id"`
	LastFetchedID    int64  `yaml:"last_fetched_id,omitempty"`
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

	// Backward compat: migrate old backend → radio_backend, FlrigEnabled → RadioBackend.
	for id, rp := range cfg.Rigs {
		if rp.RadioBackend == "" && rp.Backend != "" {
			rp.RadioBackend = rp.Backend
			rp.Backend = ""
		}
		if rp.RadioBackend == "" && rp.FlrigEnabled {
			rp.RadioBackend = "flrig"
		}
		rp.FlrigEnabled = false // no longer the source of truth
		rp.Backend = ""         // clear old key
		cfg.Rigs[id] = rp
	}

	cfg.BroadcastStations = DefaultBroadcastStations()

	return &cfg, nil
}

// Save marshals cfg as YAML and writes it to path.

func Save(path string, cfg *Config) error {
	cfg.State.Version = version.Resolved()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Upgrade runs version-gated migration steps based on the config's stored
// version. Call after Load() and before any other config-dependent init.
// Steps are idempotent — they only run when the stored version is older
// than the step's target version.
func (c *Config) Upgrade(currentVersion string) {
	if c == nil {
		return
	}
	stored := c.State.Version

	// Example future step:
	// if versionOlder(stored, "0.9.0") {
	//     // migration logic for 0.9.0
	// }

	_ = stored
	_ = currentVersion
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
	if c.Integrations.QRZ.Enabled {
		if strings.TrimSpace(c.Integrations.QRZ.User) == "" {
			return fmt.Errorf("qrz.user is required when qrz.enabled is true")
		}
		if strings.TrimSpace(c.Integrations.QRZ.Pass) == "" {
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

	// --- DXC ---
	if c.Integrations.DXC.Enabled {
		if strings.TrimSpace(c.Integrations.DXC.Host) == "" {
			return fmt.Errorf("dxc.host is required when enabled")
		}
		if c.Integrations.DXC.Port != "" {
			if p, err := strconv.Atoi(c.Integrations.DXC.Port); err != nil || p < 1 || p > 65535 {
				return fmt.Errorf("dxc.port must be 1-65535, got %q", c.Integrations.DXC.Port)
			}
		}
	}

	// --- Rigs ---
	for id, rig := range c.Rigs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("rig entry with empty id")
		}
		if rig.RadioBackend == "flrig" || rig.FlrigEnabled {
			if strings.TrimSpace(rig.FlrigHost) == "" {
				return fmt.Errorf("rig %q: flrig_host is required when radio_backend=flrig", id)
			}
			if rig.FlrigPort != "" {
				if p, err := strconv.Atoi(rig.FlrigPort); err != nil || p < 1 || p > 65535 {
					return fmt.Errorf("rig %q: flrig_port must be 1-65535, got %q", id, rig.FlrigPort)
				}
			}
		}
		if rig.RadioBackend == "hamlib" {
			if strings.TrimSpace(rig.HamlibRadioHost) == "" {
				return fmt.Errorf("rig %q: hamlib_radio_host is required when radio_backend=hamlib", id)
			}
			if rig.HamlibRadioPort != "" {
				if p, err := strconv.Atoi(rig.HamlibRadioPort); err != nil || p < 1 || p > 65535 {
					return fmt.Errorf("rig %q: hamlib_radio_port must be 1-65535, got %q", id, rig.HamlibRadioPort)
				}
			}
		}
	}

	// --- Operators ---
	seenCalls := make(map[string]string)
	for id, op := range c.Operators {
		call := strings.TrimSpace(op.Callsign)
		if call == "" {
			return fmt.Errorf("operator %q: callsign is required", id)
		}
		if !qso.IsValidCall(call) {
			return fmt.Errorf("operator %q: callsign %q is not valid", id, call)
		}
		lower := strings.ToLower(call)
		if dup, ok := seenCalls[lower]; ok {
			return fmt.Errorf("operator %q: callsign %q already used by operator %q", id, call, dup)
		}
		seenCalls[lower] = id
	}

	// --- ActiveOperator references ---
	for lbID, lb := range c.Logbooks {
		if lb.ActiveOperator == "" {
			continue
		}
		if _, ok := c.Operators[lb.ActiveOperator]; !ok {
			return fmt.Errorf("logbook %q: active_operator references unknown operator %q", lbID, lb.ActiveOperator)
		}
	}

	return nil
}
