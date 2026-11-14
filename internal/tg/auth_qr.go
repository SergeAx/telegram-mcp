package tg

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/pkg/errors"
	"rsc.io/qr"
)

const qrQuietZone = 4

func qrAuth(ctx context.Context, client *telegram.Client, appID int, appHash, password string, loggedIn qrlogin.LoggedIn) error {
	status, err := client.Auth().Status(ctx)
	if err != nil {
		return errors.Wrap(err, "auth status")
	}
	if status.Authorized {
		return nil
	}

	opts := qrlogin.Options{Migrate: client.MigrateTo}
	if password != "" {
		opts.Password = func(context.Context) (string, error) { return password, nil }
	}

	_, err = qrlogin.NewQR(client.API(), appID, appHash, opts).Auth(ctx, loggedIn, func(_ context.Context, token qrlogin.Token) error {
		code, err := qr.Encode(token.URL(), qr.L)
		if err != nil {
			return errors.Wrap(err, "encode QR")
		}

		fmt.Println("Scan with Telegram on your phone: Settings → Devices → Link Desktop Device")
		fmt.Print(renderQR(code))
		fmt.Printf("Expires at %s, a new code will be shown then\n", token.Expires().Format("15:04:05"))

		return nil
	})

	return errors.Wrap(err, "QR login")
}

// Each "▀" cell packs two QR rows: foreground is the top module, background the bottom one.
func renderQR(code *qr.Code) string {
	color := func(black bool, dark, light int) int {
		if black {
			return dark
		}

		return light
	}

	var b strings.Builder
	for y := -qrQuietZone; y < code.Size+qrQuietZone; y += 2 {
		for x := -qrQuietZone; x < code.Size+qrQuietZone; x++ {
			fmt.Fprintf(&b, "\x1b[%d;%dm▀", color(code.Black(x, y), 30, 97), color(code.Black(x, y+1), 40, 107))
		}
		b.WriteString("\x1b[0m\n")
	}

	return b.String()
}
