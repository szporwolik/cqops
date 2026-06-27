package secrets

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

// setTestKey overrides KeyFunc for the duration of the test.
func setTestKey(t *testing.T) []byte {
	t.Helper()
	key := testKey(t)
	orig := KeyFunc
	KeyFunc = func() []byte { return key }
	t.Cleanup(func() { KeyFunc = orig })
	return key
}

func TestStore_EmptyLoad(t *testing.T) {
	_ = setTestKey(t)
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Has() {
		t.Error("empty store should have no secrets")
	}
}

func TestStore_SetGet(t *testing.T) {
	_ = setTestKey(t)
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	s.Set("qrz.pass", "secret123")
	v, ok := s.Get("qrz.pass")
	if !ok {
		t.Fatal("secret not found after Set")
	}
	if v != "secret123" {
		t.Errorf("got %q, want secret123", v)
	}
}

func TestStore_SaveLoadRoundtrip(t *testing.T) {
	key := setTestKey(t)
	dir := t.TempDir()

	// Create and save.
	s1, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	s1.Set("qrz.pass", "hunter2")
	s1.Set("wavelog.logbook1.apikey", "wl-key-abc")
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload.
	s2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = key

	for _, tc := range []struct{ key, want string }{
		{"qrz.pass", "hunter2"},
		{"wavelog.logbook1.apikey", "wl-key-abc"},
	} {
		v, ok := s2.Get(tc.key)
		if !ok {
			t.Errorf("key %q not found after reload", tc.key)
			continue
		}
		if v != tc.want {
			t.Errorf("key %q: got %q, want %q", tc.key, v, tc.want)
		}
	}
}

func TestStore_Delete(t *testing.T) {
	_ = setTestKey(t)
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	s.Set("qrz.pass", "secret")
	s.Delete("qrz.pass")
	if _, ok := s.Get("qrz.pass"); ok {
		t.Error("secret should be deleted")
	}
}

func TestStore_GetMissing(t *testing.T) {
	_ = setTestKey(t)
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get("nonexistent"); ok {
		t.Error("missing key should return false")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	_ = setTestKey(t)
	dir := t.TempDir()

	s1, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	s1.Set("k", "v")
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	// Override with a different key for the reload.
	_ = setTestKey(t) // different key

	s2, err := Load(dir)
	if err != nil {
		t.Fatal("Load should not error on decrypt failure:", err)
	}

	// Decryption should fail silently → empty map, Corrupted flag set.
	if s2.Has() {
		t.Error("decryption with wrong key should result in empty store")
	}
	if !s2.Corrupted {
		t.Error("Corrupted flag should be set when decrypt fails")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := testKey(t)
	plain := []byte("hello, world — this is a secret message!")

	ct, err := encrypt(key, plain)
	if err != nil {
		t.Fatal(err)
	}

	got, err := decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Errorf("decrypt: got %q, want %q", got, plain)
	}
}

func TestEncryptDecrypt_Empty(t *testing.T) {
	key := testKey(t)
	plain := []byte{}

	ct, err := encrypt(key, plain)
	if err != nil {
		t.Fatal(err)
	}

	got, err := decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d bytes", len(got))
	}
}

func TestSave_WritesWithCorrectPermissions(t *testing.T) {
	_ = setTestKey(t)
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.Set("k", "v")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, "secrets.enc"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("secrets.enc permissions: got %04o, want 0600", mode)
	}
}
