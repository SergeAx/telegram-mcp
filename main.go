package main

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

const (
	dir = ".telegram-mcp"
)

func main() {
	if err := setupLogging(); err != nil {
		exit(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		exit(fmt.Errorf("find the home directory: %w", err))
	}

	app := &cli.Command{
		Name:  "telegram-mcp",
		Usage: "Telegram MCP server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config-dir",
				Usage:   "Directory with config.toml and the session",
				Value:   filepath.Join(homeDir, dir),
				Sources: cli.EnvVars("TG_CONFIG_DIR"),
			},
			&cli.IntFlag{
				Name:    "api-id",
				Aliases: []string{"app-id"},
				Usage:   "Telegram API ID, overrides config.toml",
				Sources: cli.EnvVars("TG_API_ID", "TG_APP_ID"),
			},
			&cli.StringFlag{
				Name:    "session",
				Usage:   "Path to session file (default: session.json in --config-dir)",
				Sources: cli.EnvVars("TG_SESSION_PATH"),
			},
			&cli.StringFlag{
				Name:    "download-dir",
				Usage:   "Directory tg_download saves documents to, one subdirectory per dialog",
				Value:   filepath.Join(os.TempDir(), "telegram-mcp"),
				Sources: cli.EnvVars("TG_DOWNLOAD_DIR"),
			},
			&cli.StringFlag{
				Name:    "schema-version",
				Usage:   "JSON Schema version URL (e.g. https://json-schema.org/draft-07/schema#)",
				Sources: cli.EnvVars("TG_SCHEMA_VERSION"),
			},
			&cli.BoolFlag{
				Name:        "dry",
				Usage:       "Test configuration",
				Local:       true,
				HideDefault: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "init",
				Usage:  "Store the API ID and the encrypted API hash in config.toml",
				Flags:  []cli.Flag{apiHashFlag()},
				Action: initCommand,
			},
			{
				Name:  "auth",
				Usage: "Authenticate with Telegram",
				Flags: []cli.Flag{
					apiHashFlag(),
					&cli.StringFlag{
						Name:    "phone",
						Usage:   "Phone number to authenticate with (required unless --qr)",
						Aliases: []string{"p"},
					},
					&cli.BoolFlag{
						Name:        "qr",
						Usage:       "Log in by scanning a QR code from a logged-in phone",
						HideDefault: true,
					},
					&cli.StringFlag{
						Name:        "password",
						Usage:       "2FA password, asked for when needed if omitted",
						HideDefault: true,
					},
					&cli.BoolFlag{
						Name:        "new",
						Usage:       "Log in again even if the current session still works",
						HideDefault: true,
					},
				},
				Action: authCommand,
			},
		},
		Action: serve,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		exit(err)
	}
}

// setupLogging writes to stderr, which MCP clients keep as the server's
// log, or with TG_DEBUG_LOG to that file in JSON, debug messages included.
func setupLogging() error {
	path := os.Getenv("TG_DEBUG_LOG")
	if path == "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

		return nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666) //nolint:gosec // the operator chooses the log path
	if err != nil {
		return fmt.Errorf("open the debug log: %w", err)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})))

	return nil
}

func exit(err error) {
	notice("Error: %v", err)
	if os.Getenv("TG_DEBUG_LOG") != "" {
		slog.Error("Exiting", "err", err)
	}
	os.Exit(1)
}

// notice talks to the person at the terminal; logs are for diagnostics.
func notice(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func apiHashFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    "api-hash",
		Usage:   "Telegram API hash, overrides config.toml",
		Sources: cli.EnvVars("TG_API_HASH"),
	}
}

func legacySessionPath(cmd *cli.Command) string {
	return cmp.Or(cmd.Root().String("session"), filepath.Join(cmd.Root().String("config-dir"), "session.json"))
}
