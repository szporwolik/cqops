// Package secrets provides encrypted storage for sensitive configuration
// values (passwords, API keys). Secrets are encrypted with an AES-256-GCM
// key derived from the machine identity and stored separately from the
// plaintext config.yaml.
//
// The encrypted file lives at <configDir>/secrets.enc. After decryption at
// startup, values are cached in memory — subsequent reads are zero-overhead
// map lookups.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Store holds decrypted secrets in memory and persists them to an
// AES-256-GCM encrypted file.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
	path string
	key  []byte // derived 256-bit key, cached after first derivation

	// Corrupted is true when the secrets file exists but could not be
	// decrypted (wrong machine, corrupted file, old key). The app
	// starts normally but secrets must be re-entered by the user.
	Corrupted bool
}

// KeyFunc returns the 256-bit AES key. Override in tests to use a fixed key.
var KeyFunc = deriveKey

// Load reads and decrypts the secrets file at <dir>/secrets.enc.
// If the file does not exist, an empty Store is returned (no error).
// The 256-bit AES key is derived from the machine identity.
func Load(dir string) (*Store, error) {
	s := &Store{
		data: make(map[string]string),
		path: filepath.Join(dir, "secrets.enc"),
		key:  KeyFunc(),
	}

	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil // first run — no secrets yet
		}
		return nil, fmt.Errorf("secrets: open: %w", err)
	}
	defer f.Close()

	ciphertext, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("secrets: read: %w", err)
	}

	plaintext, err := decrypt(s.key, ciphertext)
	if err != nil {
		// Decryption failed — the secrets file may be from a different
		// machine, corrupted, or encrypted with an old key. Return an
		// empty store so the app can still start; secrets will be
		// re-entered by the user.
		s.Corrupted = true
		return s, nil
	}

	if err := json.Unmarshal(plaintext, &s.data); err != nil {
		return nil, fmt.Errorf("secrets: unmarshal: %w", err)
	}

	return s, nil
}

// Save encrypts and writes all secrets to disk. Atomic: writes to a temp
// file then renames.
func (s *Store) Save() error {
	s.mu.RLock()
	plaintext, err := json.Marshal(s.data)
	s.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("secrets: marshal: %w", err)
	}

	ciphertext, err := encrypt(s.key, plaintext)
	if err != nil {
		return fmt.Errorf("secrets: encrypt: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, ciphertext, 0600); err != nil {
		return fmt.Errorf("secrets: write: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("secrets: rename: %w", err)
	}
	return nil
}

// Get returns the secret value for key, and whether it was found.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	v, ok := s.data[key]
	s.mu.RUnlock()
	return v, ok
}

// Set stores a secret value in memory. Call Save() to persist.
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
}

// Delete removes a secret. Call Save() to persist.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
}

// Has reports whether any secrets are stored.
func (s *Store) Has() bool {
	s.mu.RLock()
	n := len(s.data)
	s.mu.RUnlock()
	return n > 0
}

// --- crypto ---

// deriveKey produces a stable 256-bit AES key from the machine identity.
// Uses /etc/machine-id on Linux (systemd), /var/lib/dbus/machine-id
// fallback, or hostname as last resort. The result is NOT intended to be
// a strong cryptographic secret — it protects against casual exposure
// (config file leaks, backups, dotfile sharing), not a determined local
// attacker with root access.
func deriveKey() []byte {
	id := machineID()
	h := sha256.Sum256([]byte("cqops/secrets/v1:" + id))
	return h[:]
}

func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	// ciphertext = nonce || gcm-seal(plaintext)
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, errors.New("secrets: ciphertext too short")
	}
	nonce, ct := ciphertext[:ns], ciphertext[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

// --- machine identity (production) ---

func machineID() string {
	// Linux: prefer /etc/machine-id (systemd), fall back to dbus machine-id.
	if id, err := os.ReadFile("/etc/machine-id"); err == nil {
		if s := trim(string(id)); s != "" {
			return s
		}
	}
	if id, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		if s := trim(string(id)); s != "" {
			return s
		}
	}
	// macOS: try system profiling.
	if id, err := os.ReadFile("/Library/Preferences/SystemConfiguration/com.apple.wifi.plist"); false {
		_ = id
		_ = err
	}
	// Windows: MachineGuid would require syscall, skip for now.
	// Fallback: hostname.
	if host, err := os.Hostname(); err == nil {
		return host
	}
	return "cqops-default-machine"
}

func trim(s string) string {
	// Strip trailing newline and whitespace common in system files.
	n := len(s)
	for n > 0 && (s[n-1] == '\n' || s[n-1] == '\r' || s[n-1] == ' ') {
		n--
	}
	return s[:n]
}

// --- test seams ---
