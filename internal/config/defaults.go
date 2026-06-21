package config

func DefaultConfig() *Config {
	tz := SystemTimezone()
	defaultRigID := NewID("default-rig")
	defaultLogbookID := NewID("default-logbook")

	return &Config{
		General: GeneralConfig{
			Timezone:         tz,
			DistanceUnit:     "km",
			PictureAtQRZPane: false,
			SolarAtQSOPane:   true,
			UseCTY:           true,
			UseSCP:           true,
			UseRef:           true,
			Debug:            true,
			Notifications: NotificationsConfig{
				Enabled:       true,
				QSO:           false,
				Wavelog:       false,
				WavelogErrors: true,
			},
		},
		State: StateConfig{
			ActiveLogbook: defaultLogbookID,
		},
		Logbooks: map[string]Logbook{
			defaultLogbookID: {
				ID:      defaultLogbookID,
				Name:    "Default",
				Station: Station{},
				ADIF:    ADIFConfig{},
			},
		},
		Rigs: map[string]RigPreset{
			defaultRigID: {
				ID: defaultRigID,
			},
		},
	}
}
