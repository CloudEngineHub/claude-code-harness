package dialogloop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/livemsg"
)

const peerReplySubject = "dialogloop-peer-reply.v1"

// PeerReplySubject is the livemsg subject for dialogloop peer replies.
const PeerReplySubject = peerReplySubject

// peerReplyPayload is the JSON envelope for livemsg transport bodies.
type peerReplyPayload struct {
	Body          string `json:"body"`
	NextRound     bool   `json:"next_round"`
	HumanStop     bool   `json:"human_stop"`
	FloorCategory string `json:"floor_category,omitempty"`
}

// LivemsgPeerOpts configures NewLivemsgPeer polling behaviour.
type LivemsgPeerOpts struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
}

// NewLivemsgPeer returns a Peer that sends via store and polls the recipient inbox
// for a JSON-encoded peerReplyPayload reply. Fail-open: poll timeout returns an
// empty converged reply rather than failing the loop.
func NewLivemsgPeer(store *livemsg.Store, opts LivemsgPeerOpts) Peer {
	interval := opts.PollInterval
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}
	timeout := opts.PollTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	return func(ctx context.Context, team, fromAgent, toAgent, body string) (PeerReply, error) {
		if store == nil {
			return PeerReply{NextRound: false}, nil
		}
		if _, err := store.Send(ctx, team, fromAgent, toAgent, peerReplySubject, body); err != nil {
			return PeerReply{}, err
		}

		deadline := time.Now().Add(timeout)
		for {
			if err := ctx.Err(); err != nil {
				return PeerReply{}, err
			}
			msgs, err := store.Inbox(ctx, team, fromAgent)
			if err != nil {
				return PeerReply{}, err
			}
			for _, msg := range msgs {
				if msg.FromAgent != toAgent {
					continue
				}
				var payload peerReplyPayload
				if err := json.Unmarshal([]byte(msg.Body), &payload); err != nil {
					payload = peerReplyPayload{Body: msg.Body}
				}
				_ = store.MarkRead(ctx, team, msg.ID, fromAgent)
				return PeerReply{
					Body:          payload.Body,
					NextRound:     payload.NextRound,
					HumanStop:     payload.HumanStop,
					FloorCategory: payload.FloorCategory,
				}, nil
			}
			if time.Now().After(deadline) {
				return PeerReply{NextRound: false}, nil
			}
			select {
			case <-ctx.Done():
				return PeerReply{}, ctx.Err()
			case <-time.After(interval):
			}
		}
	}
}

// EncodePeerReply serializes a PeerReply for livemsg transport.
func EncodePeerReply(reply PeerReply) (string, error) {
	payload := peerReplyPayload{
		Body:          reply.Body,
		NextRound:     reply.NextRound,
		HumanStop:     reply.HumanStop,
		FloorCategory: reply.FloorCategory,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("dialogloop: encode peer reply: %w", err)
	}
	return string(b), nil
}
