package config

func DefaultConfig() *Config {
	tz := SystemTimezone()

	return &Config{
		ActiveLogbook: "default",
		Timezone:      tz,
		Logbooks: map[string]Logbook{
			"default": {
				Description: "Default station logbook",
				Station:     Station{},
				ADIF:        ADIFConfig{},
			},
		},
		Rigs: map[string]RigPreset{
			"default": {},
		},
		Rig: RigConfig{
			Provider:     "",
			AutoFill:     true,
			FailSilently: true,
			Flrig: struct {
				Enabled   bool   `yaml:"enabled"`
				URL       string `yaml:"url"`
				TimeoutMS int    `yaml:"timeout_ms"`
			}{
				Enabled:   false,
				URL:       "http://localhost:12345",
				TimeoutMS: 1000,
			},
		},
	}
}
