// Package config reads and writes config.toml in the telegram-mcp directory.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/chaindead/telegram-mcp/internal/atomicfile"
	"github.com/chaindead/telegram-mcp/internal/secret"
)

const fileName = "config.toml"

const header = "# Written by telegram-mcp init and auth: edited values are kept, comments are not.\n\n"

type Config struct {
	APIID   int            `toml:"api_id,omitzero"`
	APIHash secret.Armored `toml:"api_hash,omitempty"`
	// DefaultAccount is the Telegram user ID of the account the server uses.
	DefaultAccount int64 `toml:"default_account,omitzero"`
}

func Path(dir string) string {
	return filepath.Join(dir, fileName)
}

func SessionPath(dir string, userID int64) string {
	return filepath.Join(dir, fmt.Sprintf("%d.session.age", userID))
}

// Load returns an empty Config if there is no config file yet.
func Load(dir string) (*Config, error) {
	var c Config
	md, err := toml.DecodeFile(Path(dir), &c)
	if errors.Is(err, fs.ErrNotExist) {
		return &c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", Path(dir), err)
	}

	if unknown := md.Undecoded(); len(unknown) > 0 {
		keys := make([]string, 0, len(unknown))
		for _, k := range unknown {
			keys = append(keys, k.String())
		}
		slog.Warn("Ignoring unknown config keys", "keys", keys, "file", Path(dir))
	}

	return &c, nil
}

func (c *Config) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	buf := bytes.NewBufferString(header)
	if err := toml.NewEncoder(buf).Encode(c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := atomicfile.Write(Path(dir), buf.Bytes()); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
