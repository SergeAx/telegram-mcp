package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/chaindead/telegram-mcp/internal/config"
)

func initCommand(ctx context.Context, cmd *cli.Command) error {
	dir := cmd.Root().String("config-dir")
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	creds, err := resolveCredentials(cmd, cfg, true)
	if err != nil {
		return err
	}
	if err := creds.apply(cfg); err != nil {
		return err
	}
	if err := cfg.Save(dir); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}
	if err := migrateLegacySession(ctx, cmd, cfg, creds.apiID); err != nil {
		return err
	}

	key, _ := secretKey()
	ev := log.Info().Str("config", config.Path(dir)).Str("secret_key", key.Source())
	if cfg.DefaultAccount == 0 {
		ev.Msg("Saved the API credentials, now run: telegram-mcp auth --qr (or --phone <number>)")
	} else {
		ev.Msg("Saved the API credentials")
	}

	return nil
}
