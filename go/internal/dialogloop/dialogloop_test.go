package dialogloop_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/dialogloop"
)

func baseConfig(peer dialogloop.Peer) dialogloop.Config {
	return dialogloop.Config{
		Team:      "team-a",
		FromAgent: "agent-a",
		ToAgent:   "agent-b",
		MaxRounds: 5,
		Peer:      peer,
	}
}

func TestRun_ConvergedFirstRound(t *testing.T) {
	cfg := baseConfig(func(_ context.Context, team, from, to, body string) (dialogloop.PeerReply, error) {
		if team != "team-a" || from != "agent-a" || to != "agent-b" || body != "hello" {
			t.Fatalf("unexpected peer args: team=%q from=%q to=%q body=%q", team, from, to, body)
		}
		return dialogloop.PeerReply{Body: "done", NextRound: false}, nil
	})

	out, err := dialogloop.Run(context.Background(), cfg, "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Rounds != 1 {
		t.Fatalf("Rounds = %d, want 1", out.Rounds)
	}
	if out.StopReason != dialogloop.StopReasonConverged {
		t.Fatalf("StopReason = %q, want %q", out.StopReason, dialogloop.StopReasonConverged)
	}
}

func TestRun_PingPongUntilConverge(t *testing.T) {
	var round int
	cfg := baseConfig(func(_ context.Context, _, _, _, body string) (dialogloop.PeerReply, error) {
		round++
		if round < 3 {
			return dialogloop.PeerReply{Body: "pong-" + body, NextRound: true}, nil
		}
		return dialogloop.PeerReply{Body: "final", NextRound: false}, nil
	})

	out, err := dialogloop.Run(context.Background(), cfg, "start")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Rounds != 3 {
		t.Fatalf("Rounds = %d, want 3", out.Rounds)
	}
	if out.StopReason != dialogloop.StopReasonConverged {
		t.Fatalf("StopReason = %q, want %q", out.StopReason, dialogloop.StopReasonConverged)
	}
}

func TestRun_MaxRoundsReached(t *testing.T) {
	cfg := baseConfig(func(_ context.Context, _, _, _, _ string) (dialogloop.PeerReply, error) {
		return dialogloop.PeerReply{Body: "again", NextRound: true}, nil
	})
	cfg.MaxRounds = 3

	out, err := dialogloop.Run(context.Background(), cfg, "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Rounds != 3 {
		t.Fatalf("Rounds = %d, want 3", out.Rounds)
	}
	if out.StopReason != dialogloop.StopReasonMaxRounds {
		t.Fatalf("StopReason = %q, want %q", out.StopReason, dialogloop.StopReasonMaxRounds)
	}
}

func TestRun_FloorCategoryStops(t *testing.T) {
	cfg := baseConfig(func(_ context.Context, _, _, _, _ string) (dialogloop.PeerReply, error) {
		return dialogloop.PeerReply{
			Body:          "blocked egress",
			FloorCategory: "egress",
		}, nil
	})
	cfg.FloorCheck = func(category string) bool {
		return category == "egress"
	}

	out, err := dialogloop.Run(context.Background(), cfg, "curl example.com")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Rounds != 1 {
		t.Fatalf("Rounds = %d, want 1", out.Rounds)
	}
	if out.StopReason != dialogloop.StopReasonFloor {
		t.Fatalf("StopReason = %q, want %q", out.StopReason, dialogloop.StopReasonFloor)
	}
	if !strings.Contains(out.HumanStopNote, "egress") {
		t.Fatalf("HumanStopNote = %q, want egress category", out.HumanStopNote)
	}
}

func TestRun_HumanDecisionStops(t *testing.T) {
	cfg := baseConfig(func(_ context.Context, _, _, _, _ string) (dialogloop.PeerReply, error) {
		return dialogloop.PeerReply{
			Body:      "please decide: ship or wait",
			HumanStop: true,
		}, nil
	})

	out, err := dialogloop.Run(context.Background(), cfg, "status?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Rounds != 1 {
		t.Fatalf("Rounds = %d, want 1", out.Rounds)
	}
	if out.StopReason != dialogloop.StopReasonHumanDecision {
		t.Fatalf("StopReason = %q, want %q", out.StopReason, dialogloop.StopReasonHumanDecision)
	}
	if !strings.Contains(out.HumanStopNote, "ship or wait") {
		t.Fatalf("HumanStopNote = %q, want body excerpt", out.HumanStopNote)
	}
}

func TestRun_LedgerEmittedEachRound(t *testing.T) {
	var round int
	cfg := baseConfig(func(_ context.Context, _, _, _, _ string) (dialogloop.PeerReply, error) {
		round++
		if round < 3 {
			return dialogloop.PeerReply{Body: "continue", NextRound: true}, nil
		}
		return dialogloop.PeerReply{Body: "done", NextRound: false}, nil
	})
	cfg.MaxRounds = 5

	type emission struct {
		round  int
		reason string
	}
	var emissions []emission
	cfg.LedgerEmit = func(r int, reason, from, to, body string) {
		emissions = append(emissions, emission{round: r, reason: reason})
		if from != "agent-a" || to != "agent-b" {
			t.Errorf("LedgerEmit agents: from=%q to=%q", from, to)
		}
		_ = body
	}

	out, err := dialogloop.Run(context.Background(), cfg, "start")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Rounds != 3 {
		t.Fatalf("Rounds = %d, want 3", out.Rounds)
	}
	if len(emissions) != 4 {
		t.Fatalf("ledger emissions = %d, want 4 (3 in-flight + 1 stop)", len(emissions))
	}
	for i := 0; i < 3; i++ {
		if emissions[i].round != i+1 {
			t.Errorf("emission[%d].round = %d, want %d", i, emissions[i].round, i+1)
		}
		if emissions[i].reason != "in-flight" {
			t.Errorf("emission[%d].reason = %q, want in-flight", i, emissions[i].reason)
		}
	}
	if emissions[3].reason != string(dialogloop.StopReasonConverged) {
		t.Errorf("final emission reason = %q, want converged", emissions[3].reason)
	}
}

func TestRun_ConfigValidation(t *testing.T) {
	valid := baseConfig(func(_ context.Context, _, _, _, _ string) (dialogloop.PeerReply, error) {
		return dialogloop.PeerReply{}, nil
	})

	cases := []struct {
		name string
		cfg  dialogloop.Config
	}{
		{
			name: "MaxRounds zero",
			cfg: func() dialogloop.Config {
				c := valid
				c.MaxRounds = 0
				return c
			}(),
		},
		{
			name: "MaxRounds negative",
			cfg: func() dialogloop.Config {
				c := valid
				c.MaxRounds = -1
				return c
			}(),
		},
		{
			name: "Peer nil",
			cfg: func() dialogloop.Config {
				c := valid
				c.Peer = nil
				return c
			}(),
		},
		{
			name: "Team empty",
			cfg: func() dialogloop.Config {
				c := valid
				c.Team = ""
				return c
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dialogloop.Run(context.Background(), tc.cfg, "x")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestRun_PeerErrorPropagates(t *testing.T) {
	cfg := baseConfig(func(_ context.Context, _, _, _, _ string) (dialogloop.PeerReply, error) {
		return dialogloop.PeerReply{}, errors.New("transport down")
	})

	_, err := dialogloop.Run(context.Background(), cfg, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRun_FloorCheckNilTreatsNonEmptyCategory(t *testing.T) {
	cfg := baseConfig(func(_ context.Context, _, _, _, _ string) (dialogloop.PeerReply, error) {
		return dialogloop.PeerReply{FloorCategory: "prod-deploy"}, nil
	})
	cfg.FloorCheck = nil

	out, err := dialogloop.Run(context.Background(), cfg, "deploy")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.StopReason != dialogloop.StopReasonFloor {
		t.Fatalf("StopReason = %q, want floor", out.StopReason)
	}
}
