package config

import (
	"fmt"
	"os"
)

// --- Secret key constants ---
const (
	secretQRZPass    = "qrz.pass"
	secretHamQTHPass = "hamqth.pass"
	secretDXCLogin   = "dxc.login"
)

func wavelogSecretKey(logbookID string) string {
	return "wavelog." + logbookID + ".apikey"
}

// savedSecrets holds plaintext copies of secrets that were extracted
// during Save, so they can be restored to the in-memory Config struct
// after YAML marshaling.
type savedSecrets struct {
	QRZPass     string
	HamQTHPass  string
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
	if c.Integrations.Callbook.QRZ.Pass != "" {
		saved.QRZPass = c.Integrations.Callbook.QRZ.Pass
		c.secrets.Set(secretQRZPass, c.Integrations.Callbook.QRZ.Pass)
		c.Integrations.Callbook.QRZ.Pass = ""
	}

	// HamQTH password.
	if c.Integrations.Callbook.HamQTH.Pass != "" {
		saved.HamQTHPass = c.Integrations.Callbook.HamQTH.Pass
		c.secrets.Set(secretHamQTHPass, c.Integrations.Callbook.HamQTH.Pass)
		c.Integrations.Callbook.HamQTH.Pass = ""
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
	if err := c.secrets.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "CQOps: secrets save failed: %v\n", err)
	}

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
		c.Integrations.Callbook.QRZ.Pass = s.QRZPass
	}
	if s.HamQTHPass != "" {
		c.Integrations.Callbook.HamQTH.Pass = s.HamQTHPass
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
		if c.Integrations.Callbook.QRZ.Pass != "" && c.Integrations.Callbook.QRZ.Pass != v {
			// Plaintext value differs — migrate it.
			c.secrets.Set(secretQRZPass, c.Integrations.Callbook.QRZ.Pass)
			if err := c.secrets.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "CQOps: secrets save failed (qrz migration): %v\n", err)
			}
		}
		c.Integrations.Callbook.QRZ.Pass = v
	} else if c.Integrations.Callbook.QRZ.Pass != "" {
		// Plaintext exists but no store entry — first migration.
		c.secrets.Set(secretQRZPass, c.Integrations.Callbook.QRZ.Pass)
	}

	// HamQTH password.
	if v, ok := c.secrets.Get(secretHamQTHPass); ok {
		if c.Integrations.Callbook.HamQTH.Pass != "" && c.Integrations.Callbook.HamQTH.Pass != v {
			c.secrets.Set(secretHamQTHPass, c.Integrations.Callbook.HamQTH.Pass)
			if err := c.secrets.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "CQOps: secrets save failed (hamqth migration): %v\n", err)
			}
		}
		c.Integrations.Callbook.HamQTH.Pass = v
	} else if c.Integrations.Callbook.HamQTH.Pass != "" {
		c.secrets.Set(secretHamQTHPass, c.Integrations.Callbook.HamQTH.Pass)
	}

	// DXC login.
	if v, ok := c.secrets.Get(secretDXCLogin); ok {
		if c.Integrations.DXC.Login != "" && c.Integrations.DXC.Login != v {
			c.secrets.Set(secretDXCLogin, c.Integrations.DXC.Login)
			if err := c.secrets.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "CQOps: secrets save failed (dxc migration): %v\n", err)
			}
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
				if err := c.secrets.Save(); err != nil {
					fmt.Fprintf(os.Stderr, "CQOps: secrets save failed (wavelog migration): %v\n", err)
				}
			}
			lb.Wavelog.APIKey = v
		} else if lb.Wavelog.APIKey != "" {
			c.secrets.Set(key, lb.Wavelog.APIKey)
		}
		c.Logbooks[id] = lb
	}

	// Persist any newly migrated secrets.
	if err := c.secrets.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "CQOps: secrets save failed (migration): %v\n", err)
	}
}
