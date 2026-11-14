package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	"github.com/chaindead/telegram-mcp/internal/sessionstore"
	"github.com/chaindead/telegram-mcp/internal/tg"
	"github.com/pkg/errors"
)

func main() {
	flag.Parse()

	appIDStr, apiHash, sessionPath := os.Getenv("TG_APP_ID"), os.Getenv("TG_API_HASH"), os.Getenv("TG_SESSION_PATH")
	if appIDStr == "" {
		fatal("TG_APP_ID is required")
	}
	if apiHash == "" {
		fatal("TG_API_HASH is required")
	}
	if sessionPath == "" {
		fatal("TG_SESSION_PATH is required")
	}

	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		fatal("TG_APP_ID is not a number", "err", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client := tg.New(appID, apiHash, sessionstore.Open(sessionPath, nil), "").T()

	if err := client.Run(ctx, func(ctx context.Context) error {
		self, err := client.Self(ctx)
		if err != nil {
			return errors.Wrap(err, "get self")
		}

		slog.Info("Logged in as",
			"first_name", self.FirstName,
			"last_name", self.LastName,
			"username", self.Username,
			"id", self.ID)

		messages, err := getUnreadMessages(ctx, client)
		if err != nil {
			return errors.Wrap(err, "get unread messages")
		}

		for _, msg := range messages {
			slog.Info("Unread message",
				"id", msg.ID,
				"text", msg.Text,
				"date", msg.Date,
				"from_id", msg.FromID,
				"from_name", msg.FromName,
				"chat_type", msg.ChatType,
				"chat_title", msg.ChatTitle)
		}

		return nil
	}); err != nil {
		fatal("client error", "err", err)
	}
}

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
