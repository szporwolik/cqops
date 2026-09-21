package config

import (
	"fmt"
	"os"
)

// --- Secret key constants ---
const (
	secretQRZPass    = "qrz.pass"
	secretHamQTHPass = "hamqth.pass"
	secretQRZRuPass  = "qrzru.pass"
	secretDXCLogin   = "dxc.login"
)

func wavelogSecretKey(logbookID string) string {
	return "wavelog." + logbookID + ".apikey"
}

// syncSecretsToStore persists secret values to the encrypted store and
// deletes entries whose live fields are now empty, so cleared credentials
// do not return on restart. This is a read-only pass over the live config —
// the YAML scrub happens on a copy in Save. It returns the store's Save
// error: the caller must abort the whole save on failure, because the
// scrubbed YAML no longer contains the credentials and a failed secrets
// write would otherwise leave no persisted copy at all.
func (c *Config) syncSecretsToStore() error {
	changed := false
	sync := func(key, val string) {
		cur, ok := c.secrets.Get(key)
		if val == "" {
			if ok {
				c.secrets.Delete(key)
				changed = true
			}
			return
		}
		if !ok || cur != val {
			c.secrets.Set(key, val)
			changed = true
		}
	}

	sync(secretQRZPass, c.Integrations.Callbook.QRZ.Pass)
	sync(secretHamQTHPass, c.Integrations.Callbook.HamQTH.Pass)
	sync(secretQRZRuPass, c.Integrations.Callbook.QRZRu.Pass)
	sync(secretDXCLogin, c.Integrations.DXC.Login)
	for id, lb := range c.Logbooks {
		val := ""
		if lb.Wavelog != nil {
			val = lb.Wavelog.APIKey
		}
		sync(wavelogSecretKey(id), val)
	}

	if !changed {
		return nil
	}
	return c.secrets.Save()
}

// scrubbedCopy returns a copy of the config that is safe to marshal:
// secret values are blanked on the copy when a secrets store is attached,
// and the logbook map plus its pointer fields are copied, so Save never
// writes shared state while other goroutines may read it.
func (c *Config) scrubbedCopy() *Config {
	cp := *c
	cp.secrets = nil
	cp.Logbooks = make(map[string]Logbook, len(c.Logbooks))
	for id, lb := range c.Logbooks {
		if lb.Wavelog != nil {
			wl := *lb.Wavelog
			if c.secrets != nil {
				wl.APIKey = ""
			}
			lb.Wavelog = &wl
		}
		if lb.APRS != nil {
			ap := *lb.APRS
			lb.APRS = &ap
		}
		cp.Logbooks[id] = lb
	}
	if c.secrets != nil {
		cp.Integrations = c.Integrations
		cp.Integrations.Callbook.QRZ.Pass = ""
		cp.Integrations.Callbook.HamQTH.Pass = ""
		cp.Integrations.Callbook.QRZRu.Pass = ""
		cp.Integrations.DXC.Login = ""
	}
	return &cp
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

	// QRZ.ru password.
	if v, ok := c.secrets.Get(secretQRZRuPass); ok {
		if c.Integrations.Callbook.QRZRu.Pass != "" && c.Integrations.Callbook.QRZRu.Pass != v {
			c.secrets.Set(secretQRZRuPass, c.Integrations.Callbook.QRZRu.Pass)
			if err := c.secrets.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "CQOps: secrets save failed (qrzru migration): %v\n", err)
			}
		}
		c.Integrations.Callbook.QRZRu.Pass = v
	} else if c.Integrations.Callbook.QRZRu.Pass != "" {
		c.secrets.Set(secretQRZRuPass, c.Integrations.Callbook.QRZRu.Pass)
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
