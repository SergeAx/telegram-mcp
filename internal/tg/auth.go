package tg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/chaindead/telegram-mcp/internal/prompt"
)

// Auth logs in unless the session in storage is already authorized, and
// returns the user ID.
func Auth(phone string, appID int, appHash string, storage telegram.SessionStorage, password string, useQR bool) (int64, error) {
	dispatcher := tg.NewUpdateDispatcher()
	loggedIn := qrlogin.OnLoginToken(dispatcher)

	opts := telegram.Options{
		SessionStorage: storage,
		UpdateHandler:  dispatcher,
	}
	if newOpts, err := telegram.OptionsFromEnvironment(opts); err != nil {
		log.Warn().Err(err).Msg("Failed to load OptionsFromEnvironment, skipping")
	} else {
		opts = newOpts
	}

	client := telegram.NewClient(appID, appHash, opts)

	var selfID int64
	if err := client.Run(context.Background(), func(ctx context.Context) error {
		var err error
		if useQR {
			err = qrAuth(ctx, client, appID, appHash, password, loggedIn)
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

		name := strings.TrimSpace(self.FirstName + " " + self.LastName)
		if self.Username != "" {
			name += " @" + self.Username
		}
		fmt.Fprintf(os.Stderr, "Logged in as %s, user ID %d\n", name, self.ID)
		selfID = self.ID

		return nil
	}); err != nil {
		return 0, fmt.Errorf("client error: %w", err)
	}

	return selfID, nil
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
		sentVia := "Telegram sent the code via " + strings.TrimPrefix(sent.Type.TypeName(), "auth.sentCodeType")
		if next, ok := sent.GetNextType(); ok {
			sentVia += ", next time via " + strings.TrimPrefix(next.TypeName(), "auth.codeType")
		}
		if timeout, ok := sent.GetTimeout(); ok {
			sentVia += fmt.Sprintf(", a new one after %d s", timeout)
		}
		fmt.Fprintln(os.Stderr, sentVia)

		answer, err := prompt.Line("📩 Enter code: ")
		if err != nil {
			return "", fmt.Errorf("read code: %w", err)
		}

		return answer, nil
	})
	flow := auth.NewFlow(passwordPrompt{auth.Constant(phone, "", code), password}, auth.SendCodeOptions{})

	return errors.Wrap(client.Auth().IfNecessary(ctx, flow), "code login")
}
