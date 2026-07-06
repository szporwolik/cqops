package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/secrets"
	"github.com/szporwolik/cqops/internal/version"
)

// Default host/port values for rig and rotor backends.
const (
	DefaultFlrigHost  = "localhost"
	DefaultFlrigPort  = "12345"
	DefaultHamlibHost = "127.0.0.1"
	DefaultHamlibPort = "4532"
	DefaultRotorHost  = "127.0.0.1"
	DefaultRotorPort  = "4533"
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

	secrets      *secrets.Store `yaml:"-"`
	savedSecrets *savedSecrets  `yaml:"-"`
}

// SetSecretsStore attaches a secrets store for encrypted persistence of
// passwords and API keys. Call before Save.
func (c *Config) SetSecretsStore(s *secrets.Store) { c.secrets = s }

// SecretsStore returns the attached secrets store, or nil.
func (c *Config) SecretsStore() *secrets.Store { return c.secrets }

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
	DXC        DXCConfig        `yaml:"dxc,omitempty"`
	QRZ        QRZConfig        `yaml:"qrz,omitempty"`
	HTTPServer HTTPServerConfig `yaml:"http_server,omitempty"`
	GPS        GPSConfig        `yaml:"gps,omitempty"`
	APRS       APRSGlobalConfig `yaml:"aprs,omitempty"`

	// Legacy key — migrated to APRS on load. Remove after v0.9.x.
	APRSLegacy APRSGlobalConfig `yaml:"aprs_is,omitempty"`
}

// Normalize migrates legacy config keys and values to their current names.
// Called after every config load.
func (c *Config) Normalize() {
	// Legacy integration key: aprs_is → aprs
	if !c.Integrations.APRS.Enabled && c.Integrations.APRSLegacy.Enabled {
		c.Integrations.APRS = c.Integrations.APRSLegacy
		c.Integrations.APRSLegacy = APRSGlobalConfig{}
	}

	// Legacy key: distance_unit → units
	if c.General.Units == "" && c.General.UnitsLegacy != "" {
		c.General.Units = c.General.UnitsLegacy
		c.General.UnitsLegacy = ""
	}
	// Legacy values: km → metric, mi → imperial
	switch c.General.Units {
	case "km":
		c.General.Units = "metric"
	case "mi":
		c.General.Units = "imperial"
	}
}

// APRSGlobalConfig holds global APRS service settings.
type APRSGlobalConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Service        string `yaml:"service,omitempty"`          // "aprs_is", "kiss", "kiss_server", or "" (none/default)
	Server         string `yaml:"server,omitempty"`           // APRS-IS server host:port
	KISSServerHost string `yaml:"kiss_server_host,omitempty"` // KISS Server TCP host (default 127.0.0.1)
	KISSServerPort string `yaml:"kiss_server_port,omitempty"` // KISS Server TCP port (default 8001)
	Port           string `yaml:"port,omitempty"`             // KISS serial port
	BaudRate       int    `yaml:"baud_rate,omitempty"`        // KISS serial baud rate
	DataBits       int    `yaml:"data_bits,omitempty"`        // KISS: 5,6,7,8 (default 8)
	Parity         string `yaml:"parity,omitempty"`           // KISS: "none","odd","even","mark","space"
	StopBits       string `yaml:"stop_bits,omitempty"`        // KISS: "1","1.5","2" (default "1")
	DTR            bool   `yaml:"dtr,omitempty"`              // KISS: enable DTR
	RTS            bool   `yaml:"rts,omitempty"`              // KISS: enable RTS
}

type DXCConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host,omitempty"`
	Port    string `yaml:"port,omitempty"`
	Login   string `yaml:"login,omitempty"`
}

type GeneralConfig struct {
	Timezone         string              `yaml:"timezone"`
	Units            string              `yaml:"units,omitempty"`         // "metric" or "imperial"
	UnitsLegacy      string              `yaml:"distance_unit,omitempty"` // legacy key, migrated on load
	RenderMap        bool                `yaml:"render_map,omitempty"`
	DrawGrayline     bool                `yaml:"draw_grayline,omitempty"`
	PictureAtQRZPane bool                `yaml:"picture_at_qrz_pane,omitempty"`
	SolarAtQSOPane   bool                `yaml:"solar_at_qso_pane,omitempty"`
	UseCTY           bool                `yaml:"use_cty,omitempty"`        // CTY.DAT DXCC country file
	UseSCP           bool                `yaml:"use_scp,omitempty"`        // Super Check Partial callsign database
	UseRef           bool                `yaml:"use_ref,omitempty"`        // REF database
	Debug            bool                `yaml:"debug,omitempty"`          // verbose debug logging
	KittyGraphics    bool                `yaml:"kitty_graphics,omitempty"` // experimental Kitty terminal graphics
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

// HTTPServerConfig holds the optional built-in HTTP server configuration.
type HTTPServerConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Address    string `yaml:"address,omitempty"`      // e.g. "0.0.0.0" (LAN) or "localhost"
	Port       string `yaml:"port,omitempty"`         // e.g. "8073"
	Header1    string `yaml:"header_1,omitempty"`     // club name for dashboard
	Header2    string `yaml:"header_2,omitempty"`     // event name for dashboard
	ClubLogo   string `yaml:"club_logo,omitempty"`    // file path or URL to club logo
	EventStart string `yaml:"event_start,omitempty"`  // YYYY-MM-DD, filter stats from this date
	MapTileURL string `yaml:"map_tile_url,omitempty"` // Leaflet tile server URL
	MapAttrib  string `yaml:"map_attrib,omitempty"`   // tile attribution text
}

// GPSConfig holds GPS receiver serial port configuration.
type GPSConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Service       string `yaml:"service,omitempty"`        // "serial", "gpsd", or "" (none)
	Port          string `yaml:"port,omitempty"`           // serial port e.g. "COM6"
	BaudRate      int    `yaml:"baud_rate,omitempty"`      // serial baud rate
	DTR           bool   `yaml:"dtr,omitempty"`            // enable DTR
	RTS           bool   `yaml:"rts,omitempty"`            // enable RTS
	GPSDHost      string `yaml:"gpsd_host,omitempty"`      // GPSD server address
	GPSDPort      string `yaml:"gpsd_port,omitempty"`      // GPSD server port
	GridPrecision int    `yaml:"grid_precision,omitempty"` // 6, 8, or 10 (default 10)
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
	APRS           *APRSConfig    `yaml:"aprs,omitempty"`
}

type Station struct {
	Callsign   string `yaml:"callsign"`
	Grid       string `yaml:"grid"`
	GPSGrid    bool   `yaml:"gps_grid,omitempty"` // use GPS grid when available
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
		return false, DefaultFlrigHost, DefaultFlrigPort
	}
	return rp.RadioBackend == "flrig", rp.FlrigHost, rp.FlrigPort
}

// RigHamlib returns the hamlib settings from the referenced preset.
func (s Station) RigHamlib(rgs map[string]RigPreset) (enabled bool, host, port string) {
	rp, ok := s.Rig(rgs)
	if !ok {
		return false, DefaultHamlibHost, DefaultHamlibPort
	}
	return rp.RadioBackend == "hamlib", rp.HamlibRadioHost, rp.HamlibRadioPort
}

// RigRotor returns the rotor settings from the referenced preset.
func (s Station) RigRotor(rgs map[string]RigPreset) (enabled bool, host, port string) {
	rp, ok := s.Rig(rgs)
	if !ok {
		return false, DefaultRotorHost, DefaultRotorPort
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
	Backend         string `yaml:"backend,omitempty"`       // DEPRECATED: migrated to RadioBackend in Load() — remove after v1.0
	FlrigEnabled    bool   `yaml:"flrig_enabled,omitempty"` // DEPRECATED: migrated to RadioBackend in Load() — remove after v1.0
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

// APRSConfig holds per-logbook APRS beacon settings.
// Server is configured globally in Integrations → APRS-IS.
type APRSConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Callsign     string `yaml:"callsign"`
	Passcode     string `yaml:"passcode"`
	RadiusKm     int    `yaml:"radius_km"`
	SendLocation bool   `yaml:"send_location"`
	IntervalMin  int    `yaml:"interval_minutes"`
	Symbol       string `yaml:"symbol"`
	Comment      string `yaml:"comment"`
	LastBeaconAt string `yaml:"last_beacon_at,omitempty"` // RFC3339, per-logbook
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

	// Migrate legacy config keys to current names.
	cfg.Normalize()

	// Backward compat: migrate old backend → radio_backend, FlrigEnabled → RadioBackend.
	for id, rp := range cfg.Rigs {
		if rp.RadioBackend == "" && rp.Backend != "" {
			fmt.Fprintf(os.Stderr, "CQOps: rig %s uses deprecated 'backend' field — please update to 'radio_backend'\n", id)
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

// Save marshals cfg as YAML and writes it to path. If a secrets store is
// attached via SetSecretsStore, passwords and API keys are extracted and
// persisted to the encrypted store before the YAML is written.
func Save(path string, cfg *Config) error {
	cfg.State.Version = version.Resolved()

	// Extract and persist secrets before marshaling.
	if cfg.secrets != nil {
		cfg.extractAndSaveSecrets()
	}
	defer cfg.restoreSecrets() // restore in-memory values after YAML marshal

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
// At minimum, stamps the current version into state so future migrations
// can gate on it.
func (c *Config) Upgrade(currentVersion string) {
	if c == nil {
		return
	}
	// Stamp the running version so future migrations can use it.
	c.State.Version = currentVersion
}

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
	switch c.General.Units {
	case "", "metric", "imperial":
	default:
		return fmt.Errorf("general.units must be 'metric' or 'imperial', got %q", c.General.Units)
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
	if lb.Station.IARURegion != 0 && (lb.Station.IARURegion < 1 || lb.Station.IARURegion > 3) {
		return fmt.Errorf("station iaru_region must be 1, 2, or 3, got %d", lb.Station.IARURegion)
	}
	if lb.Station.CQZone != 0 && (lb.Station.CQZone < 1 || lb.Station.CQZone > 40) {
		return fmt.Errorf("station cq_zone must be 1-40, got %d", lb.Station.CQZone)
	}
	if lb.Station.ITUZone != 0 && (lb.Station.ITUZone < 1 || lb.Station.ITUZone > 90) {
		return fmt.Errorf("station itu_zone must be 1-90, got %d", lb.Station.ITUZone)
	}

	// --- HTTPServer ---
	if c.Integrations.HTTPServer.Enabled {
		if c.Integrations.HTTPServer.Port != "" {
			if p, err := strconv.Atoi(c.Integrations.HTTPServer.Port); err != nil || p < 1 || p > 65535 {
				return fmt.Errorf("http_server.port must be 1-65535, got %q", c.Integrations.HTTPServer.Port)
			}
		}
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
		login := strings.TrimSpace(c.Integrations.DXC.Login)
		if login != "" && !qso.IsValidCall(login) {
			return fmt.Errorf("dxc.login must be a valid callsign, got %q", login)
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

	// --- Station.RigName references ---
	for lbID, lb := range c.Logbooks {
		if lb.Station.RigName == "" {
			continue
		}
		if _, ok := c.Rigs[lb.Station.RigName]; !ok {
			return fmt.Errorf("logbook %q: rig_name references unknown rig %q", lbID, lb.Station.RigName)
		}
	}

	// --- ActiveContest references ---
	for lbID, lb := range c.Logbooks {
		if lb.ActiveContest == "" {
			continue
		}
		if _, ok := c.Contests[lb.ActiveContest]; !ok {
			return fmt.Errorf("logbook %q: active_contest references unknown contest %q", lbID, lb.ActiveContest)
		}
	}

	// --- Contest.LogbookID references ---
	for ctID, ct := range c.Contests {
		if ct.LogbookID == "" {
			continue
		}
		if _, ok := c.Logbooks[ct.LogbookID]; !ok {
			return fmt.Errorf("contest %q: logbook_id references unknown logbook %q", ctID, ct.LogbookID)
		}
	}

	return nil
}
