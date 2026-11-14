package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/chaindead/telegram-mcp/internal/config"
	"github.com/chaindead/telegram-mcp/internal/prompt"
	"github.com/chaindead/telegram-mcp/internal/secret"
	"github.com/chaindead/telegram-mcp/internal/sessionstore"
	"github.com/chaindead/telegram-mcp/internal/tg"
)

// serverStorage picks the default account's encrypted session, or the
// plaintext session.json of a setup from before init.
func serverStorage(cmd *cli.Command, cfg *config.Config) (*sessionstore.Store, error) {
	if cfg.DefaultAccount == 0 || cmd.IsSet("session") {
		path := legacySessionPath(cmd)
		slog.Warn("The session is not encrypted: run telegram-mcp init to encrypt it", "session", path)

		return sessionstore.Open(path, nil), nil
	}

	key, err := secret.Load()
	if err != nil {
		return nil, fmt.Errorf("load secret key (telegram-mcp init creates one, or set %s): %w", secret.EnvVar, err)
	}

	return sessionstore.Open(config.SessionPath(cmd.String("config-dir"), cfg.DefaultAccount), key), nil
}

// migrateLegacySession encrypts the session.json of a setup from before
// init into the account's session file, and makes that the default account.
func migrateLegacySession(ctx context.Context, cmd *cli.Command, cfg *config.Config, apiID int) error {
	path := legacySessionPath(cmd)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the old session: %w", err)
	}
	if cfg.DefaultAccount != 0 {
		notice("Delete %s: it holds a session unencrypted, and config.toml already names an account", path)

		return nil
	}

	userID, err := legacyUserID(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if userID == 0 {
		legacy := sessionstore.Open(path, nil)
		if userID, err = tg.New(apiID, "", legacy, "").SelfID(ctx); err != nil {
			return fmt.Errorf("find the account of %s: %w", path, err)
		}
		if data, err = legacy.LoadSession(ctx); err != nil {
			return fmt.Errorf("reread %s: %w", path, err)
		}
	}

	key, err := secretKey()
	if err != nil {
		return err
	}
	dir := cmd.Root().String("config-dir")
	encrypted := sessionstore.Open(config.SessionPath(dir, userID), key)
	if err := encrypted.StoreSession(ctx, data); err != nil {
		return fmt.Errorf("encrypt the old session: %w", err)
	}
	back, err := encrypted.LoadSession(ctx)
	if err != nil {
		return fmt.Errorf("verify the encrypted session: %w", err)
	}
	if !bytes.Equal(back, data) {
		return fmt.Errorf("verify the encrypted session: %s differs from %s", encrypted.Path(), path)
	}

	cfg.DefaultAccount = userID
	if err := cfg.Save(dir); err != nil {
		return fmt.Errorf("save the account: %w", err)
	}
	notice("Encrypted the session from %s into %s", path, encrypted.Path())

	removeLegacySession(path)

	return nil
}

// legacyUserID is zero in sessions saved by gotd versions that did not
// record it.
func legacyUserID(data []byte) (int64, error) {
	var s struct{ Data struct{ UserID int64 } }
	if err := json.Unmarshal(data, &s); err != nil {
		return 0, fmt.Errorf("decode session: %w", err)
	}

	return s.Data.UserID, nil
}

func removeLegacySession(path string) {
	if prompt.Interactive() {
		answer, err := prompt.Line(fmt.Sprintf("Delete the unencrypted %s? [Y/n] ", path))
		if err == nil && !strings.EqualFold(answer, "n") && !strings.EqualFold(answer, "no") {
			if err := os.Remove(path); err != nil {
				notice("Failed to delete the unencrypted session: %v", err)

				return
			}
			notice("Deleted %s", path)

			return
		}
	}

	notice("Delete %s: it holds the session unencrypted", path)
}
