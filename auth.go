package main

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

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

	ev := log.Info().
		Str("session", sessionPath).
		Int64("app-id", appID)
	if phone != "" {
		ev = ev.Str("phone", maskPhone(phone))
	}
	ev.Msg("Authenticate with Telegram")

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

var twoDigitCountryCodes = strings.Fields("20 27 30 31 32 33 34 36 39 40 41 43 44 45 46 47 48 49 " +
	"51 52 53 54 55 56 57 58 60 61 62 63 64 65 66 81 82 84 86 90 91 92 93 94 95 98")

// Country codes are prefix-free: 1 and 7 are the only one-digit codes,
// the listed ones are two digits long, and all the rest are three.
func countryCodeLen(digits string) int {
	switch {
	case digits[0] == '1' || digits[0] == '7':
		return 1
	case slices.Contains(twoDigitCountryCodes, digits[:2]):
		return 2
	default:
		return 3
	}
}

// A fixed run of dots hides the number's length too.
func maskPhone(phone string) string {
	digits := strings.Map(func(r rune) rune {
		if r < '0' || r > '9' {
			return -1
		}

		return r
	}, phone)
	if len(digits) < 8 {
		return "+....."
	}

	return "+" + digits[:countryCodeLen(digits)] + "....." + digits[len(digits)-4:]
}
