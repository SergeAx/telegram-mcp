package main

import (
	"context"
	"fmt"

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
	notice("Saved the API credentials to %s, encrypted with the key from %s", config.Path(dir), key.Source())
	if cfg.DefaultAccount == 0 {
		notice("Now log in: telegram-mcp auth --qr, or --phone <number>")
	}

	return nil
}
