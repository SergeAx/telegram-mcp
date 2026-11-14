package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/chaindead/telegram-mcp/internal/config"
	"github.com/chaindead/telegram-mcp/internal/sessionstore"
	"github.com/chaindead/telegram-mcp/internal/tg"
	"github.com/invopop/jsonschema"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func serve(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load(cmd.String("config-dir"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	appID := cmp.Or(int(cmd.Int("api-id")), cfg.APIID)
	if appID == 0 {
		return errors.New("API ID is not set: run telegram-mcp init, or pass --api-id or TG_API_ID")
	}
	sessionPath := legacySessionPath(cmd)
	dryRun := cmd.Bool("dry")

	if schemaURL := cmd.String("schema-version"); schemaURL != "" {
		jsonschema.Version = schemaURL
	}

	_, err = os.Stat(sessionPath)
	if err != nil {
		return fmt.Errorf("session file not found(%s): %w", sessionPath, err)
	}

	server := mcp.NewServer(stdio.NewStdioServerTransport())
	client := tg.New(appID, "", sessionstore.Open(sessionPath), cmd.String("download-dir"))

	if dryRun {
		return runChecks(client)
	}

	err = server.RegisterTool("tg_me", "Get current telegram account info", client.GetMe)
	if err != nil {
		return fmt.Errorf("register tool: %w", err)
	}

	err = server.RegisterTool("tg_dialogs", "Get list of telegram dialogs (chats, channels, users)", client.GetDialogs)
	if err != nil {
		return fmt.Errorf("register dialogs tool: %w", err)
	}

	err = server.RegisterTool("tg_dialog", "Get messages of telegram dialog", client.GetHistory)
	if err != nil {
		return fmt.Errorf("register dialogs tool: %w", err)
	}

	err = server.RegisterTool("tg_send", "Send draft message to dialog", client.SendDraft)
	if err != nil {
		return fmt.Errorf("register dialogs tool: %w", err)
	}

	err = server.RegisterTool("tg_download", "Download the document attached to a message, returning the local path", client.Download)
	if err != nil {
		return fmt.Errorf("register download tool: %w", err)
	}

	err = server.RegisterTool("tg_read", "Mark dialog messages as read", client.ReadHistory)
	if err != nil {
		return fmt.Errorf("register read tool: %w", err)
	}

	if err := server.Serve(); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	<-ctx.Done()

	return nil
}

func runChecks(client *tg.Client) error {
	answer, err := client.GetMe(tg.EmptyArguments{})
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	data, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	log.Info().RawJSON("answer", data).Msg("Check GetMe: OK")

	answer, err = client.GetDialogs(tg.DialogsArguments{Offset: "", OnlyUnread: true})
	if err != nil {
		return fmt.Errorf("get dialogs: %w", err)
	}

	log.Info().RawJSON("answer", []byte(answer.Content[0].TextContent.Text)).Msg("Check GetDialogs: OK")

	// The public @telegram news channel, not the verified "Telegram" service
	// chat: that one holds login codes, which must not reach the log.
	answer, err = client.GetHistory(tg.HistoryArguments{Name: "telegram"})
	if err != nil {
		return fmt.Errorf("get channel history: %w", err)
	}

	log.Info().RawJSON("answer", []byte(answer.Content[0].TextContent.Text)).Msg("Check GetHistory: OK")

	testUser := os.Getenv("TG_TEST_USERNAME")
	if testUser == "" {
		log.Info().Msg("TG_TEST_USERNAME is not set, skipping SendDraft and ReadHistory checks")

		return nil
	}

	answer, err = client.SendDraft(tg.DraftArguments{Name: testUser, Text: "test draft"})
	if err != nil {
		log.Err(err).Msg("Check SendDraft: FAIL")
	} else {
		log.Info().RawJSON("answer", []byte(answer.Content[0].TextContent.Text)).Msg("Check SendDraft: OK")
	}

	answer, err = client.ReadHistory(tg.ReadArguments{Name: testUser})
	if err != nil {
		log.Err(err).Msg("Check ReadHistory: FAIL")
	} else {
		log.Info().RawJSON("answer", []byte(answer.Content[0].TextContent.Text)).Msg("Check ReadHistory: OK")
	}

	return nil
}
