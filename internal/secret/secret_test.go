package secret

import (
	"errors"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestRoundTrip(t *testing.T) {
	key, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	text, err := key.Encrypt([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(text), "-----BEGIN AGE ENCRYPTED FILE-----\n") {
		t.Errorf("Encrypt() = %q, want age armor", text)
	}

	plain, err := key.Decrypt(text)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "0123456789abcdef0123456789abcdef" {
		t.Errorf("Decrypt() = %q", plain)
	}
}

func TestDecryptWithOtherKey(t *testing.T) {
	a, _ := Generate()
	b, _ := Generate()
	text, err := a.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Decrypt(text); err == nil {
		t.Error("Decrypt() with another key succeeded")
	}
}

func TestLoad(t *testing.T) {
	keyring.MockInit()
	t.Setenv(EnvVar, "")

	if _, err := Load(); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Load() from empty keychain error = %v, want ErrNoKey", err)
	}

	stored, _ := Generate()
	if err := stored.Store(); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.String() != stored.String() || loaded.Source() != "OS keychain" {
		t.Errorf("Load() = key from %s, want the stored one", loaded.Source())
	}

	fromEnv, _ := Generate()
	t.Setenv(EnvVar, fromEnv.String())
	if loaded, _ := Load(); loaded.String() != fromEnv.String() {
		t.Error("Load() ignored " + EnvVar)
	}
}
