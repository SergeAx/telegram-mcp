package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/chaindead/telegram-mcp/internal/config"
	"github.com/chaindead/telegram-mcp/internal/prompt"
	"github.com/chaindead/telegram-mcp/internal/secret"
)

var apiHashFormat = regexp.MustCompile(`^[0-9a-f]{32}$`)

// credentials come from flags and env vars first, then config.toml, then
// questions; reask turns config.toml values into the questions' defaults.
type credentials struct {
	apiID   int
	apiHash string
	asked   bool
}

var secretKey = sync.OnceValues(loadOrCreateKey)

func resolveCredentials(cmd *cli.Command, cfg *config.Config, reask bool) (credentials, error) {
	var c credentials
	var err error
	if c.apiID, err = resolveAPIID(cmd, cfg, reask, &c.asked); err != nil {
		return c, err
	}
	if c.apiHash, err = resolveAPIHash(cmd, cfg, reask, &c.asked); err != nil {
		return c, err
	}

	return c, nil
}

func (c credentials) apply(cfg *config.Config) error {
	key, err := secretKey()
	if err != nil {
		return err
	}

	cfg.APIID = c.apiID
	if cfg.APIHash, err = key.Encrypt([]byte(c.apiHash)); err != nil {
		return fmt.Errorf("encrypt API hash: %w", err)
	}

	return nil
}

func resolveAPIID(cmd *cli.Command, cfg *config.Config, reask bool, asked *bool) (int, error) {
	if cmd.Root().IsSet("api-id") {
		return int(cmd.Root().Int("api-id")), nil
	}
	if cfg.APIID != 0 && !reask {
		return cfg.APIID, nil
	}

	question := "API ID: "
	if cfg.APIID != 0 {
		question = fmt.Sprintf("API ID [%d]: ", cfg.APIID)
	}
	for {
		answer, err := prompt.Line(question)
		if err != nil {
			return 0, fmt.Errorf("API ID is not set (--api-id, TG_API_ID or init): %w", err)
		}
		if answer == "" && cfg.APIID != 0 {
			return cfg.APIID, nil
		}
		if id, err := strconv.Atoi(answer); err == nil && id > 0 {
			*asked = true

			return id, nil
		}
		fmt.Fprintln(os.Stderr, "The API ID is the number shown at https://my.telegram.org/apps")
	}
}

func resolveAPIHash(cmd *cli.Command, cfg *config.Config, reask bool, asked *bool) (string, error) {
	if cmd.IsSet("api-hash") {
		hash := strings.ToLower(cmd.String("api-hash"))
		if !apiHashFormat.MatchString(hash) {
			return "", errors.New("--api-hash must be 32 hex digits")
		}

		return hash, nil
	}
	if cfg.APIHash != "" && !reask {
		return storedAPIHash(cfg)
	}

	question := "API hash: "
	if cfg.APIHash != "" {
		question = "API hash [keep the stored one]: "
	}
	for {
		answer, err := prompt.Masked(question)
		if err != nil {
			return "", fmt.Errorf("API hash is not set (--api-hash, TG_API_HASH or init): %w", err)
		}
		if answer == "" && cfg.APIHash != "" {
			return storedAPIHash(cfg)
		}
		if hash := strings.ToLower(answer); apiHashFormat.MatchString(hash) {
			*asked = true

			return hash, nil
		}
		fmt.Fprintln(os.Stderr, "The API hash is 32 hex digits, shown at https://my.telegram.org/apps")
	}
}

func storedAPIHash(cfg *config.Config) (string, error) {
	key, err := secretKey()
	if err != nil {
		return "", err
	}

	hash, err := key.Decrypt(cfg.APIHash)
	if err != nil {
		return "", fmt.Errorf("stored API hash: %w", err)
	}

	return string(hash), nil
}

func loadOrCreateKey() (*secret.Key, error) {
	key, err := secret.Load()
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, secret.ErrNoKey) {
		return nil, fmt.Errorf("load secret key: %w", err)
	}

	if key, err = secret.Generate(); err != nil {
		return nil, fmt.Errorf("create secret key: %w", err)
	}
	if err := key.Store(); err != nil {
		notice("No OS keychain to keep the secret key in: %v", err)
		notice("Set this variable wherever telegram-mcp runs, in the MCP client config too:\n\n  %s=%s\n", secret.EnvVar, key)
	}

	return key, nil
}
