package tg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"
	mcp "github.com/metoro-io/mcp-golang"
)

type SearchArguments struct {
	Query  string `json:"query" jsonschema:"required,description=Text to search for"`
	Name   string `json:"name,omitempty" jsonschema:"description=Name of the dialog to search in; all dialogs when empty"`
	Offset string `json:"offset,omitempty" jsonschema:"description=Offset for continuation"`
}

type SearchResult struct {
	Dialog string `json:"dialog,omitempty"`
	Title  string `json:"title,omitempty"`
	MessageInfo
}

type SearchResponse struct {
	Messages []SearchResult `json:"messages"`
	Offset   string         `json:"offset,omitempty"`
}

const searchPageSize = 50

func (c *Client) Search(args SearchArguments) (*mcp.ToolResponse, error) {
	if strings.TrimSpace(args.Query) == "" {
		return nil, errors.New("query is empty")
	}

	offset, err := parseSearchOffset(args.Offset)
	if err != nil {
		return nil, fmt.Errorf("parse offset: %w", err)
	}

	var raw tg.MessagesMessagesClass
	client := c.T()
	if err := client.Run(context.Background(), func(ctx context.Context) (err error) {
		api := client.API()

		if args.Name == "" {
			raw, err = api.MessagesSearchGlobal(ctx, &tg.MessagesSearchGlobalRequest{
				Q:          args.Query,
				Filter:     &tg.InputMessagesFilterEmpty{},
				OffsetRate: offset.Rate,
				OffsetPeer: offset.Peer,
				OffsetID:   offset.MsgID,
				Limit:      searchPageSize,
			})
		} else {
			var peer tg.InputPeerClass
			peer, err = getInputPeerFromName(ctx, api, args.Name)
			if err != nil {
				return fmt.Errorf("get inputPeer from name: %w", err)
			}

			raw, err = api.MessagesSearch(ctx, &tg.MessagesSearchRequest{
				Peer:     peer,
				Q:        args.Query,
				Filter:   &tg.InputMessagesFilterEmpty{},
				OffsetID: offset.MsgID,
				Limit:    searchPageSize,
			})
		}
		if err != nil {
			return fmt.Errorf("search messages: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	h, err := newHistory(raw)
	if err != nil {
		return nil, fmt.Errorf("process search results: %w", err)
	}

	var nextRate int
	if slice, ok := raw.(*tg.MessagesMessagesSlice); ok {
		nextRate, _ = slice.GetNextRate()
	}

	rsp := h.searchResponse(nextRate)
	jsonData, err := json.Marshal(rsp)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonData))), nil
}

func (h *history) searchResponse(nextRate int) SearchResponse {
	chats := make(map[int64]any)
	for _, c := range h.Chats {
		switch v := c.(type) {
		case *tg.Chat:
			chats[v.ID] = v
		case *tg.Channel:
			chats[v.ID] = v
		}
	}

	peerEntity := func(p tg.PeerClass) any {
		if u, ok := p.(*tg.PeerUser); ok {
			if user, ok := h.users[u.UserID]; ok {
				return user
			}

			return nil
		}

		return chats[getPeerID(p)]
	}

	rsp := SearchResponse{Messages: make([]SearchResult, 0, len(h.Messages))}
	var last *tg.Message
	for _, msg := range h.Messages {
		m, ok := msg.(*tg.Message)
		if !ok {
			continue
		}
		last = m

		entity := peerEntity(m.PeerID)
		rsp.Messages = append(rsp.Messages, SearchResult{
			Dialog:      getUsername(entity),
			Title:       getTitle(entity),
			MessageInfo: h.info(m),
		})
	}

	if last != nil && len(h.Messages) >= searchPageSize {
		next := searchOffset{Rate: nextRate, MsgID: last.ID, Peer: inputPeerOf(peerEntity(last.PeerID))}
		rsp.Offset = next.String()
	}

	return rsp
}

func inputPeerOf(entity any) tg.InputPeerClass {
	switch v := entity.(type) {
	case *tg.User:
		return v.AsInputPeer()
	case *tg.Chat:
		return v.AsInputPeer()
	case *tg.Channel:
		return v.AsInputPeer()
	default:
		return &tg.InputPeerEmpty{}
	}
}

// searchOffset is colon-separated because access hashes can be negative.
type searchOffset struct {
	Rate  int
	MsgID int
	Peer  tg.InputPeerClass
}

func (o searchOffset) String() string {
	kind, id, hash := "none", int64(0), int64(0)
	switch p := o.Peer.(type) {
	case *tg.InputPeerUser:
		kind, id, hash = peerKindUser, p.UserID, p.AccessHash
	case *tg.InputPeerChat:
		kind, id = peerKindChat, p.ChatID
	case *tg.InputPeerChannel:
		kind, id, hash = peerKindChannel, p.ChannelID, p.AccessHash
	}

	return fmt.Sprintf("%d:%s:%d:%d:%d", o.Rate, kind, id, hash, o.MsgID)
}

func parseSearchOffset(s string) (searchOffset, error) {
	o := searchOffset{Peer: &tg.InputPeerEmpty{}}
	if s == "" {
		return o, nil
	}

	parts := strings.Split(s, ":")
	if len(parts) != 5 {
		return o, fmt.Errorf("want 5 colon-separated fields, got %q", s)
	}

	nums := make([]int64, 0, 4)
	for _, p := range []string{parts[0], parts[2], parts[3], parts[4]} {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return o, fmt.Errorf("invalid number %q: %w", p, err)
		}
		nums = append(nums, n)
	}
	o.Rate, o.MsgID = int(nums[0]), int(nums[3])
	id, hash := nums[1], nums[2]

	switch parts[1] {
	case peerKindUser:
		o.Peer = &tg.InputPeerUser{UserID: id, AccessHash: hash}
	case peerKindChat:
		o.Peer = &tg.InputPeerChat{ChatID: id}
	case peerKindChannel:
		o.Peer = &tg.InputPeerChannel{ChannelID: id, AccessHash: hash}
	case "none":
	default:
		return o, fmt.Errorf("unknown peer kind %q", parts[1])
	}

	return o, nil
}
