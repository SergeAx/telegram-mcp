package tg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/pkg/errors"
)

type DownloadArguments struct {
	Name      string `json:"name" jsonschema:"required,description=Name of the dialog"`
	MessageID int    `json:"message_id" jsonschema:"required,description=ID of the message with the document as tg_dialog reports it"`
}

type DownloadResponse struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size"`
}

func (c *Client) Download(args DownloadArguments) (*mcp.ToolResponse, error) {
	var rsp DownloadResponse
	client := c.T()
	if err := client.Run(context.Background(), func(ctx context.Context) error {
		api := client.API()

		inputPeer, err := getInputPeerFromName(ctx, api, args.Name)
		if err != nil {
			return fmt.Errorf("get inputPeer from name: %w", err)
		}

		msg, err := getMessage(ctx, api, inputPeer, args.MessageID)
		if err != nil {
			return err
		}

		media, ok := msg.Media.(*tg.MessageMediaDocument)
		if !ok {
			return fmt.Errorf("message %d has no document", args.MessageID)
		}
		doc, ok := messageDocument(media)
		if !ok {
			return fmt.Errorf("document of message %d is no longer available", args.MessageID)
		}

		dir := filepath.Join(c.downloadDir, safeFileName(args.Name))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create download dir: %w", err)
		}

		rsp = DownloadResponse{
			Path:     filepath.Join(dir, fmt.Sprintf("%d-%s", args.MessageID, safeFileName(documentFileName(doc)))),
			MimeType: doc.MimeType,
			Size:     doc.Size,
		}
		if _, err := downloader.NewDownloader().Download(api, doc.AsInputDocumentFileLocation("")).ToPath(ctx, rsp.Path); err != nil {
			return fmt.Errorf("download document: %w", err)
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to download")
	}

	jsonData, err := json.Marshal(rsp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal response")
	}

	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonData))), nil
}

func getMessage(ctx context.Context, api *tg.Client, peer tg.InputPeerClass, id int) (*tg.Message, error) {
	ids := []tg.InputMessageClass{&tg.InputMessageID{ID: id}}

	var (
		raw tg.MessagesMessagesClass
		err error
	)
	if ch, ok := peer.(*tg.InputPeerChannel); ok {
		raw, err = api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash},
			ID:      ids,
		})
	} else {
		raw, err = api.MessagesGetMessages(ctx, ids)
	}
	if err != nil {
		return nil, fmt.Errorf("get message %d: %w", id, err)
	}

	h, err := newHistory(raw)
	if err != nil {
		return nil, err
	}
	for _, m := range h.Messages {
		if msg, ok := m.(*tg.Message); ok && msg.ID == id && inDialog(peer, msg) {
			return msg, nil
		}
	}

	return nil, fmt.Errorf("message %d not found in this dialog", id)
}

// inDialog guards messages.getMessages, which looks the ID up across all
// private chats and basic groups, not just the one asked for.
func inDialog(peer tg.InputPeerClass, msg *tg.Message) bool {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		u, ok := msg.PeerID.(*tg.PeerUser)

		return ok && u.UserID == p.UserID
	case *tg.InputPeerChat:
		c, ok := msg.PeerID.(*tg.PeerChat)

		return ok && c.ChatID == p.ChatID
	default:
		return true
	}
}

func safeFileName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}

		return r
	}, name)
	name = strings.Trim(name, ". ")
	if name == "" {
		return "file"
	}

	return name
}
