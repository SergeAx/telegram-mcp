package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/chaindead/telegram-mcp/internal/config"
)

func initCommand(_ context.Context, cmd *cli.Command) error {
	dir := cmd.Root().String("config-dir")
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	creds, err := resolveCredentials(cmd, cfg, true)
	if err != nil {
		return err
	}
	if err := creds.save(dir, cfg); err != nil {
		return err
	}

	key, _ := secretKey()
	log.Info().
		Str("config", config.Path(dir)).
		Str("secret_key", key.Source()).
		Msg("Saved the API credentials, now run: telegram-mcp auth --qr (or --phone <number>)")

	return nil
}
