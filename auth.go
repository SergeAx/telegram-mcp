package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/gotd/td/session"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/chaindead/telegram-mcp/internal/config"
	"github.com/chaindead/telegram-mcp/internal/secret"
	"github.com/chaindead/telegram-mcp/internal/sessionstore"
	"github.com/chaindead/telegram-mcp/internal/tg"
)

func authCommand(ctx context.Context, cmd *cli.Command) error {
	phone := cmd.String("phone")
	useQR := cmd.Bool("qr")
	if phone == "" && !useQR {
		return errors.New("either --phone or --qr is required")
	}
	dir := cmd.Root().String("config-dir")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	creds, err := resolveCredentials(cmd, cfg, false)
	if err != nil {
		return err
	}
	if err := migrateLegacySession(ctx, cmd, cfg, creds.apiID); err != nil {
		return err
	}
	key, err := secretKey()
	if err != nil {
		return err
	}

	login, err := currentSession(ctx, dir, cfg, key, cmd.Bool("new"))
	if err != nil {
		return err
	}

	ev := log.Info().Int("api_id", creds.apiID)
	if phone != "" {
		ev = ev.Str("phone", maskPhone(phone))
	}
	ev.Msg("Authenticate with Telegram")

	userID, err := tg.Auth(phone, creds.apiID, creds.apiHash, login, cmd.String("password"), useQR)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to authenticate with Telegram")
	}

	if err := saveAccount(ctx, dir, cfg, creds, key, login, userID); err != nil {
		return err
	}

	server := map[string]any{"command": "telegram-mcp"}
	if creds.apiID != cfg.APIID {
		server["env"] = map[string]string{"TG_API_ID": strconv.Itoa(creds.apiID)}
	}

	snippet, _ := json.MarshalIndent(map[string]any{"telegram": server}, "", "\t")
	log.Info().RawJSON("config", snippet).Msg("Successfully authenticated with Telegram")

	return nil
}

// currentSession starts the login from the default account's session, so
// that auth logs in only if that one no longer works.
func currentSession(ctx context.Context, dir string, cfg *config.Config, key *secret.Key, fresh bool) (*session.StorageMemory, error) {
	login := new(session.StorageMemory)
	if fresh || cfg.DefaultAccount == 0 {
		return login, nil
	}

	data, err := sessionstore.Open(config.SessionPath(dir, cfg.DefaultAccount), key).LoadSession(ctx)
	switch {
	case err == nil:
		_ = login.StoreSession(ctx, data)
	case !errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("load the current session (--new logs in afresh): %w", err)
	}

	return login, nil
}

func saveAccount(
	ctx context.Context, dir string, cfg *config.Config, creds credentials, key *secret.Key, login *session.StorageMemory, userID int64,
) error {
	data, err := login.Bytes(nil)
	if err != nil {
		return fmt.Errorf("read the new session: %w", err)
	}
	store := sessionstore.Open(config.SessionPath(dir, userID), key)
	if err := store.StoreSession(ctx, data); err != nil {
		return fmt.Errorf("save the new session: %w", err)
	}
	if prev := cfg.DefaultAccount; prev != 0 && prev != userID {
		log.Info().Str("session", config.SessionPath(dir, prev)).Msg("Kept the session of the previous account")
	}

	cfg.DefaultAccount = userID
	if creds.asked {
		if err := creds.apply(cfg); err != nil {
			return err
		}
	}
	if err := cfg.Save(dir); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	log.Info().Str("session", store.Path()).Msg("Saved the encrypted session")

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
