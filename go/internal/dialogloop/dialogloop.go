// Package dialogloop は端末 A↔B の往復対話 policy を実装する。
// Mode 2 (CCH live-messaging) 上で動作し、livemsg store を transport に使う。
//
// 終了条件:
//   - peer から GO 応答が返り、相手が「収束」(NextRound=false) を返したとき
//   - 最大往復回数 MaxRounds に到達
//   - 5 カテゴリ floor に該当（HumanStopReason=floor:<cat>）
//   - peer agent 判断で StopReason="human-decision-needed"
package dialogloop

import (
	"context"
	"fmt"
	"strings"
)

type StopReason string

const (
	StopReasonConverged     StopReason = "converged"
	StopReasonMaxRounds     StopReason = "max-rounds"
	StopReasonFloor         StopReason = "floor"
	StopReasonHumanDecision StopReason = "human-decision-needed"
)

const ledgerReasonInFlight = "in-flight"

// PeerReply は 1 ターン分の応答。peer が次往復が必要と判断したら NextRound=true。
// peer が「人間判断要」と判断したら HumanStop=true（loop 即終了）。
type PeerReply struct {
	Body          string
	NextRound     bool
	HumanStop     bool
	FloorCategory string // 空文字以外で floor 該当扱い（runtimefloor 連動）
}

// Peer は livemsg 経由で相手端末に投げて応答を受け取る関数型。
// production は livemsg.Send + Inbox poll を使う実装、テストでは fake を注入。
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

// Run は initialBody を Peer に投げて応答 → 次ターンへ、を最大 MaxRounds 回まで
// 繰り返す。各ターン後に LedgerEmit を呼ぶ。floor 該当 / HumanStop / 収束で打ち切り。
func Run(ctx context.Context, cfg Config, initialBody string) (Outcome, error) {
	if err := validateConfig(cfg); err != nil {
		return Outcome{}, err
	}

	body := initialBody
	var outcome Outcome

	for round := 1; round <= cfg.MaxRounds; round++ {
		if err := ctx.Err(); err != nil {
			return Outcome{}, err
		}

		reply, err := cfg.Peer(ctx, cfg.Team, cfg.FromAgent, cfg.ToAgent, body)
		if err != nil {
			return Outcome{}, err
		}

		outcome.Rounds = round
		emitLedger(cfg, round, ledgerReasonInFlight, reply.Body)

		if stopReason, note, stop := evaluateStop(cfg, reply); stop {
			outcome.StopReason = stopReason
			outcome.HumanStopNote = note
			emitLedger(cfg, round, string(stopReason), reply.Body)
			return outcome, nil
		}

		body = reply.Body
	}

	outcome.StopReason = StopReasonMaxRounds
	emitLedger(cfg, outcome.Rounds, string(StopReasonMaxRounds), body)
	return outcome, nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Team) == "" {
		return fmt.Errorf("dialogloop: Team is required")
	}
	if cfg.MaxRounds <= 0 {
		return fmt.Errorf("dialogloop: MaxRounds must be > 0, got %d", cfg.MaxRounds)
	}
	if cfg.Peer == nil {
		return fmt.Errorf("dialogloop: Peer is required")
	}
	return nil
}

func evaluateStop(cfg Config, reply PeerReply) (StopReason, string, bool) {
	if reply.HumanStop {
		return StopReasonHumanDecision, excerpt(reply.Body), true
	}
	if cat := strings.TrimSpace(reply.FloorCategory); cat != "" && floorApplies(cfg, cat) {
		return StopReasonFloor, cat, true
	}
	if !reply.NextRound {
		return StopReasonConverged, "", true
	}
	return "", "", false
}

func floorApplies(cfg Config, category string) bool {
	if cfg.FloorCheck == nil {
		return true
	}
	return cfg.FloorCheck(category)
}

func emitLedger(cfg Config, round int, reason, body string) {
	if cfg.LedgerEmit == nil {
		return
	}
	cfg.LedgerEmit(round, reason, cfg.FromAgent, cfg.ToAgent, body)
}

func excerpt(body string) string {
	body = strings.TrimSpace(body)
	const maxLen = 200
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "…"
}
