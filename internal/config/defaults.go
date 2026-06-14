package config

func DefaultConfig() *Config {
	tz := SystemTimezone()

	return &Config{
		General: GeneralConfig{
			Timezone:     tz,
			DistanceUnit: "km",
		},
		State: StateConfig{
			ActiveLogbook: "default",
		},
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
		WSJTX: WSJTXConfig{
			Enabled: false,
			UDPHost: "127.0.0.1",
			UDPPort: 2233,
		},
	}
}
