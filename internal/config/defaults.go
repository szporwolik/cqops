package config

func DefaultConfig() *Config {
	tz := SystemTimezone()
	defaultRigID := NewID("default-rig")
	defaultLogbookID := NewID("default-logbook")

	return &Config{
		General: GeneralConfig{
			Timezone:     tz,
			DistanceUnit: "km",
		},
		State: StateConfig{
			ActiveLogbook: defaultLogbookID,
		},
		Logbooks: map[string]Logbook{
			defaultLogbookID: {
				ID:          defaultLogbookID,
				Description: "Default station logbook",
				Station:     Station{},
				ADIF:        ADIFConfig{},
			},
		},
		Rigs: map[string]RigPreset{
			defaultRigID: {
				ID: defaultRigID,
			},
		},
		WSJTX: WSJTXConfig{
			Enabled: false,
			UDPHost: "127.0.0.1",
			UDPPort: 2233,
		},
	}
}
