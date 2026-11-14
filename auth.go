package main

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/chaindead/telegram-mcp/internal/tg"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func authCommand(_ context.Context, cmd *cli.Command) error {
	phone := cmd.String("phone")
	useQR := cmd.Bool("qr")
	if phone == "" && !useQR {
		return errors.New("either --phone or --qr is required")
	}
	newSession := cmd.Bool("new")
	pass := cmd.String("password")
	appID := cmd.Root().Int("app-id")
	apiHash := cmd.Root().String("api-hash")
	if apiHash == "" {
		return errors.New("--api-hash (or TG_API_HASH) is required for auth")
	}
	sessionPath := cmd.Root().String("session")

	log.Info().
		Str("phone", phone).
		Str("api-hash", apiHash).
		Str("session", sessionPath).
		Int64("app-id", appID).
		Msg("Authenticate with Telegram")

	err := tg.Auth(phone, appID, apiHash, sessionPath, pass, newSession, useQR)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to authenticate with Telegram")
	}

	config := map[string]any{
		"telegram": map[string]any{
			"command": "telegram-mcp",
			"env":     map[string]string{"TG_APP_ID": strconv.FormatInt(appID, 10)},
		},
	}

	data, _ := json.MarshalIndent(config, "", "\t")
	log.Info().RawJSON("config", data).Msg("Successfully authenticated with Telegram")

	return nil
}
