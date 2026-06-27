package config

// --- Secret key constants ---
const (
	secretQRZPass  = "qrz.pass"
	secretDXCLogin = "dxc.login"
)

func wavelogSecretKey(logbookID string) string {
	return "wavelog." + logbookID + ".apikey"
}

// savedSecrets holds plaintext copies of secrets that were extracted
// during Save, so they can be restored to the in-memory Config struct
// after YAML marshaling.
type savedSecrets struct {
	QRZPass     string
	DXCLogin    string
	WavelogKeys map[string]string // logbookID → apikey
}

// extractAndSaveSecrets copies secret values from the Config struct to
// the secrets store, clears them from the struct, and stashes copies for
// later restoration.
func (c *Config) extractAndSaveSecrets() {
	var saved savedSecrets
	saved.WavelogKeys = make(map[string]string)

	// QRZ password.
	if c.Integrations.QRZ.Pass != "" {
		saved.QRZPass = c.Integrations.QRZ.Pass
		c.secrets.Set(secretQRZPass, c.Integrations.QRZ.Pass)
		c.Integrations.QRZ.Pass = ""
	}

	// DXC login.
	if c.Integrations.DXC.Login != "" {
		saved.DXCLogin = c.Integrations.DXC.Login
		c.secrets.Set(secretDXCLogin, c.Integrations.DXC.Login)
		c.Integrations.DXC.Login = ""
	}

	// Wavelog API keys (per logbook).
	for id, lb := range c.Logbooks {
		if lb.Wavelog != nil && lb.Wavelog.APIKey != "" {
			saved.WavelogKeys[id] = lb.Wavelog.APIKey
			c.secrets.Set(wavelogSecretKey(id), lb.Wavelog.APIKey)
			lb.Wavelog.APIKey = ""
			c.Logbooks[id] = lb
		}
	}

	// Persist to disk immediately.
	_ = c.secrets.Save()

	// Stash copies for post-marshal restoration.
	c.savedSecrets = &saved
}

// restoreSecrets puts secret values back into the Config struct from the
// stashed copies. Call after YAML marshaling is complete.
func (c *Config) restoreSecrets() {
	if c.savedSecrets == nil {
		return
	}
	s := c.savedSecrets
	c.savedSecrets = nil

	if s.QRZPass != "" {
		c.Integrations.QRZ.Pass = s.QRZPass
	}
	if s.DXCLogin != "" {
		c.Integrations.DXC.Login = s.DXCLogin
	}
	for id, key := range s.WavelogKeys {
		if lb, ok := c.Logbooks[id]; ok {
			if lb.Wavelog == nil {
				lb.Wavelog = &WavelogConfig{}
			}
			lb.Wavelog.APIKey = key
			c.Logbooks[id] = lb
		}
	}
}

// ApplySecrets overlays secrets from the store onto the Config struct.
// Call after Load() when a secrets store is available. If a field already
// has a non-empty value (e.g. from a plaintext config that hasn't been
// migrated yet), the store takes precedence and the plaintext value is
// migrated into the store.
func (c *Config) ApplySecrets() {
	if c.secrets == nil {
		return
	}

	// QRZ password.
	if v, ok := c.secrets.Get(secretQRZPass); ok {
		if c.Integrations.QRZ.Pass != "" && c.Integrations.QRZ.Pass != v {
			// Plaintext value differs — migrate it.
			c.secrets.Set(secretQRZPass, c.Integrations.QRZ.Pass)
			_ = c.secrets.Save()
		}
		c.Integrations.QRZ.Pass = v
	} else if c.Integrations.QRZ.Pass != "" {
		// Plaintext exists but no store entry — first migration.
		c.secrets.Set(secretQRZPass, c.Integrations.QRZ.Pass)
	}

	// DXC login.
	if v, ok := c.secrets.Get(secretDXCLogin); ok {
		if c.Integrations.DXC.Login != "" && c.Integrations.DXC.Login != v {
			c.secrets.Set(secretDXCLogin, c.Integrations.DXC.Login)
			_ = c.secrets.Save()
		}
		c.Integrations.DXC.Login = v
	} else if c.Integrations.DXC.Login != "" {
		c.secrets.Set(secretDXCLogin, c.Integrations.DXC.Login)
	}

	// Wavelog API keys.
	for id, lb := range c.Logbooks {
		if lb.Wavelog == nil {
			continue
		}
		key := wavelogSecretKey(id)
		if v, ok := c.secrets.Get(key); ok {
			if lb.Wavelog.APIKey != "" && lb.Wavelog.APIKey != v {
				c.secrets.Set(key, lb.Wavelog.APIKey)
				_ = c.secrets.Save()
			}
			lb.Wavelog.APIKey = v
		} else if lb.Wavelog.APIKey != "" {
			c.secrets.Set(key, lb.Wavelog.APIKey)
		}
		c.Logbooks[id] = lb
	}

	// Persist any newly migrated secrets.
	_ = c.secrets.Save()
}
