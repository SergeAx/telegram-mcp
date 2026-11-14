package tg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/chaindead/telegram-mcp/internal/prompt"
	"github.com/chaindead/telegram-mcp/internal/sessionstore"
)

func Auth(phone string, appID int64, appHash string, sessionPath string, password string, newSession, useQR bool) error {
	if newSession {
		_ = os.Remove(sessionPath)
	}

	dispatcher := tg.NewUpdateDispatcher()
	loggedIn := qrlogin.OnLoginToken(dispatcher)

	opts := telegram.Options{
		SessionStorage: sessionstore.Open(sessionPath),
		UpdateHandler:  dispatcher,
	}
	if newOpts, err := telegram.OptionsFromEnvironment(opts); err != nil {
		log.Warn().Err(err).Msg("Failed to load OptionsFromEnvironment, skipping")
	} else {
		opts = newOpts
	}

	client := telegram.NewClient(int(appID), appHash, opts)

	sessionDir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return fmt.Errorf("mkdir(%s): %w", sessionDir, err)
	}

	if err := client.Run(context.Background(), func(ctx context.Context) error {
		var err error
		if useQR {
			err = qrAuth(ctx, client, int(appID), appHash, password, loggedIn)
		} else {
			err = codeAuth(ctx, client, phone, password)
		}
		if err != nil {
			return fmt.Errorf("auth: %w", err)
		}

		self, err := client.Self(ctx)
		if err != nil {
			return fmt.Errorf("get self info: %w", err)
		}

		log.Info().
			Str("first_name", self.FirstName).
			Str("last_name", self.LastName).
			Str("username", self.Username).
			Int64("id", self.ID).
			Msg("Logged in as")

		return nil
	}); err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	return nil
}

// passwordPrompt asks for the 2FA password only if the account has one and
// it was not given upfront.
type passwordPrompt struct {
	auth.UserAuthenticator
	password string
}

func (p passwordPrompt) Password(context.Context) (string, error) {
	return askPassword(p.password)
}

func askPassword(password string) (string, error) {
	if password != "" {
		return password, nil
	}

	answer, err := prompt.Masked("🔑 2FA password: ")
	if err != nil {
		return "", fmt.Errorf("read 2FA password: %w", err)
	}

	return answer, nil
}

func codeAuth(ctx context.Context, client *telegram.Client, phone, password string) error {
	code := auth.CodeAuthenticatorFunc(func(_ context.Context, sent *tg.AuthSentCode) (string, error) {
		ev := log.Info().Stringer("via", sent.Type)
		if next, ok := sent.GetNextType(); ok {
			ev = ev.Stringer("next", next)
		}
		if timeout, ok := sent.GetTimeout(); ok {
			ev = ev.Int("resend_after_sec", timeout)
		}
		ev.Msg("Code sent")

		answer, err := prompt.Line("📩 Enter code: ")
		if err != nil {
			return "", fmt.Errorf("read code: %w", err)
		}

		return answer, nil
	})
	flow := auth.NewFlow(passwordPrompt{auth.Constant(phone, "", code), password}, auth.SendCodeOptions{})

	return errors.Wrap(client.Auth().IfNecessary(ctx, flow), "code login")
}
