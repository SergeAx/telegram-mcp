package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/chaindead/telegram-mcp/internal/config"
	"github.com/chaindead/telegram-mcp/internal/tg"
	"github.com/invopop/jsonschema"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
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
	storage, err := serverStorage(cmd, cfg)
	if err != nil {
		return err
	}
	dryRun := cmd.Bool("dry")

	if schemaURL := cmd.String("schema-version"); schemaURL != "" {
		jsonschema.Version = schemaURL
	}

	if _, err := os.Stat(storage.Path()); err != nil {
		return fmt.Errorf("session file not found, run telegram-mcp auth: %w", err)
	}

	server := mcp.NewServer(stdio.NewStdioServerTransport())
	client := tg.New(appID, "", storage, cmd.String("download-dir"))

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

	reportCheck("GetMe", answer)

	answer, err = client.GetDialogs(tg.DialogsArguments{Offset: "", OnlyUnread: true})
	if err != nil {
		return fmt.Errorf("get dialogs: %w", err)
	}

	reportCheck("GetDialogs", answer)

	// The public @telegram news channel, not the verified "Telegram" service
	// chat: that one holds login codes, which must not reach the log.
	answer, err = client.GetHistory(tg.HistoryArguments{Name: "telegram"})
	if err != nil {
		return fmt.Errorf("get channel history: %w", err)
	}

	reportCheck("GetHistory", answer)

	testUser := os.Getenv("TG_TEST_USERNAME")
	if testUser == "" {
		notice("TG_TEST_USERNAME is not set, skipping the SendDraft and ReadHistory checks")

		return nil
	}

	answer, err = client.SendDraft(tg.DraftArguments{Name: testUser, Text: "test draft"})
	if err != nil {
		notice("Check SendDraft: FAIL: %v", err)
	} else {
		reportCheck("SendDraft", answer)
	}

	answer, err = client.ReadHistory(tg.ReadArguments{Name: testUser})
	if err != nil {
		notice("Check ReadHistory: FAIL: %v", err)
	} else {
		reportCheck("ReadHistory", answer)
	}

	return nil
}

// reportCheck prints the answer to stdout, which carries nothing else
// without the MCP server.
func reportCheck(name string, answer *mcp.ToolResponse) {
	notice("Check %s: OK", name)
	fmt.Println(answer.Content[0].TextContent.Text)
}
