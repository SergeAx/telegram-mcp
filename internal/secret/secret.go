// Package secret encrypts what telegram-mcp keeps on disk with an age
// identity from the OS keychain, or from an environment variable where
// there is no keychain.
package secret

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/zalando/go-keyring"
)

const (
	EnvVar = "TG_SECRET_KEY"

	keyringService = "telegram-mcp"
	keyringUser    = "age-identity"
)

var ErrNoKey = errors.New("no secret key")

// Armored is age output in its ASCII armor, which TOML keeps as a
// multi-line string.
type Armored string

func (a Armored) MarshalTOML() ([]byte, error) {
	return []byte("'''\n" + string(a) + "'''"), nil
}

type Key struct {
	identity *age.X25519Identity
	source   string
}

// Load finds the key without creating one.
func Load() (*Key, error) {
	if s := os.Getenv(EnvVar); s != "" {
		return parse(s, EnvVar)
	}

	s, err := keyring.Get(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, ErrNoKey
	}
	if err != nil {
		return nil, fmt.Errorf("%w: OS keychain: %w", ErrNoKey, err)
	}

	return parse(s, "OS keychain")
}

func Generate() (*Key, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generate secret key: %w", err)
	}

	return &Key{identity: identity, source: EnvVar}, nil
}

// Store saves the key in the OS keychain.
func (k *Key) Store() error {
	if err := keyring.Set(keyringService, keyringUser, k.identity.String()); err != nil {
		return fmt.Errorf("save secret key to OS keychain: %w", err)
	}
	k.source = "OS keychain"

	return nil
}

func (k *Key) String() string {
	return k.identity.String()
}

func (k *Key) Source() string {
	return k.source
}

func (k *Key) Encrypt(plain []byte) (Armored, error) {
	var b strings.Builder
	a := armor.NewWriter(&b)
	w, err := age.Encrypt(a, k.identity.Recipient())
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}
	if _, err := w.Write(plain); err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}
	if err := a.Close(); err != nil {
		return "", fmt.Errorf("armor: %w", err)
	}

	return Armored(b.String()), nil
}

func (k *Key) Decrypt(text Armored) ([]byte, error) {
	r, err := age.Decrypt(armor.NewReader(strings.NewReader(string(text))), k.identity)
	if err != nil {
		return nil, fmt.Errorf("decrypt with the key from %s: %w", k.source, err)
	}

	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decrypt with the key from %s: %w", k.source, err)
	}

	return plain, nil
}

func parse(s, source string) (*Key, error) {
	identity, err := age.ParseX25519Identity(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("parse secret key from %s: %w", source, err)
	}

	return &Key{identity: identity, source: source}, nil
}
