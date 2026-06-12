// Package dialogloop は端末 A↔B の往復対話 policy を実装する。
package dialogloop

import (
	"context"
	"errors"
)

type StopReason string

const (
	StopReasonConverged     StopReason = "converged"
	StopReasonMaxRounds     StopReason = "max-rounds"
	StopReasonFloor         StopReason = "floor"
	StopReasonHumanDecision StopReason = "human-decision-needed"
)

type PeerReply struct {
	Body          string
	NextRound     bool
	HumanStop     bool
	FloorCategory string
}

type Peer func(ctx context.Context, team, fromAgent, toAgent, body string) (PeerReply, error)

type Config struct {
	Team       string
	FromAgent  string
	ToAgent    string
	MaxRounds  int
	Peer       Peer
	FloorCheck func(category string) bool
	LedgerEmit func(round int, reason string, fromAgent, toAgent, body string)
}

type Outcome struct {
	Rounds        int
	StopReason    StopReason
	HumanStopNote string
}

// Run is not yet implemented (RED phase).
func Run(ctx context.Context, cfg Config, initialBody string) (Outcome, error) {
	return Outcome{}, errors.New("dialogloop: not implemented")
}
