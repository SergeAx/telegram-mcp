package tg

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"
)

const (
	peerKindUser    = "user"
	peerKindChat    = "chat"
	peerKindChannel = "chan"
)

type DialogsOffset struct {
	MsgID int `json:"msg_id"`
	Date  int `json:"offset_date"`
	Peer  tg.InputPeerClass
}

func getInputPeerID(p tg.PeerClass) tg.InputPeerClass {
	switch v := p.(type) {
	case *tg.PeerUser:
		return &tg.InputPeerUser{UserID: v.UserID}
	case *tg.PeerChannel:
		return &tg.InputPeerChannel{ChannelID: v.ChannelID}
	case *tg.PeerChat:
		return &tg.InputPeerChat{ChatID: v.ChatID}
	default:
		return &tg.InputPeerEmpty{}
	}
}

func (o DialogsOffset) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(o.String())
	if err != nil {
		return nil, fmt.Errorf("marshal offset: %w", err)
	}

	return data, nil
}

func (o *DialogsOffset) String() string {
	var id int64
	var peerType string
	switch p := o.Peer.(type) {
	case *tg.InputPeerUser:
		peerType = peerKindUser
		id = p.UserID
	case *tg.InputPeerChannel:
		peerType = peerKindChannel
		id = p.ChannelID
	case *tg.InputPeerChat:
		peerType = peerKindChat
		id = p.ChatID
	default:
		peerType = "unknown"
	}

	if id == 0 {
		return "end"
	}

	return fmt.Sprintf("%s-%d-%d-%d", peerType, id, o.MsgID, o.Date)
}

func (o *DialogsOffset) UnmarshalJSON(data []byte) error {
	parts := strings.Split(string(data), "-")
	if len(parts) != 4 {
		return errors.New("invalid format")
	}

	var err error
	switch parts[0] {
	case peerKindUser:
		var userID int64
		userID, err = strconv.ParseInt(parts[1], 10, 64)
		o.Peer = &tg.InputPeerUser{UserID: userID}
	case peerKindChannel:
		var channelID int64
		channelID, err = strconv.ParseInt(parts[1], 10, 64)
		o.Peer = &tg.InputPeerChannel{ChannelID: channelID}
	case peerKindChat:
		var chatID int64
		chatID, err = strconv.ParseInt(parts[1], 10, 64)
		o.Peer = &tg.InputPeerChat{ChatID: chatID}
	default:
		return errors.New("unknown peer type")
	}

	if err != nil {
		return fmt.Errorf("invalid peer: %w", err)
	}

	o.MsgID, err = strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid message ID: %w", err)
	}

	o.Date, err = strconv.Atoi(parts[3])
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}

	return nil
}
