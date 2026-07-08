package dialogloop_test

import (
	"context"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/dialogloop"
	"github.com/Chachamaru127/claude-code-harness/go/internal/livemsg"
)

func TestNewLivemsgPeer_ReceivesJSONReply(t *testing.T) {
	store, err := livemsg.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	team := "team-a"
	from := "agent-a"
	to := "agent-b"

	replyBody, err := dialogloop.EncodePeerReply(dialogloop.PeerReply{
		Body:      "ack",
		NextRound: false,
	})
	if err != nil {
		t.Fatalf("EncodePeerReply: %v", err)
	}
	if _, err := store.Send(ctx, team, to, from, dialogloop.PeerReplySubject, replyBody); err != nil {
		t.Fatalf("Send reply: %v", err)
	}

	peer := dialogloop.NewLivemsgPeer(store, dialogloop.LivemsgPeerOpts{
		PollInterval: 1,
		PollTimeout:  500,
	})
	got, err := peer(ctx, team, from, to, "hello")
	if err != nil {
		t.Fatalf("peer: %v", err)
	}
	if got.Body != "ack" {
		t.Fatalf("Body = %q, want ack", got.Body)
	}
	if got.NextRound {
		t.Fatal("NextRound = true, want false")
	}
}
