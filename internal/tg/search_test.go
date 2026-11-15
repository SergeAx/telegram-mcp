package tg

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestSearchOffsetRoundTrip(t *testing.T) {
	for _, o := range []searchOffset{
		{Rate: 1731700000, MsgID: 42, Peer: &tg.InputPeerChannel{ChannelID: 1536467031, AccessHash: -7686665012531382339}},
		{Rate: 7, MsgID: 9, Peer: &tg.InputPeerUser{UserID: 560043329, AccessHash: 1234567890123}},
		{MsgID: 607738, Peer: &tg.InputPeerChat{ChatID: 700944713}},
		{MsgID: 5, Peer: &tg.InputPeerEmpty{}},
	} {
		got, err := parseSearchOffset(o.String())
		if err != nil {
			t.Fatalf("parseSearchOffset(%q): %v", o.String(), err)
		}
		if got.String() != o.String() {
			t.Errorf("round trip: got %q, want %q", got.String(), o.String())
		}
	}
}

func TestParseSearchOffsetEmpty(t *testing.T) {
	o, err := parseSearchOffset("")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := o.Peer.(*tg.InputPeerEmpty); !ok || o.Rate != 0 || o.MsgID != 0 {
		t.Errorf("parseSearchOffset(\"\") = %+v, want the zero offset with an empty peer", o)
	}
}

func TestParseSearchOffsetInvalid(t *testing.T) {
	for _, s := range []string{"1:user:2:3", "x:user:2:3:4", "1:group:2:3:4"} {
		if _, err := parseSearchOffset(s); err == nil {
			t.Errorf("parseSearchOffset(%q) succeeded, want an error", s)
		}
	}
}
