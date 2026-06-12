package bridgedelivery_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridgedelivery"
)

type recordingDeliverer struct {
	target bridgedelivery.Target
	err    error

	mu       sync.Mutex
	called   bool
	lastNote bridgedelivery.Notice
}

func (d *recordingDeliverer) Target() bridgedelivery.Target { return d.target }

func (d *recordingDeliverer) Deliver(_ context.Context, n bridgedelivery.Notice) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.called = true
	d.lastNote = n
	return d.err
}

func (d *recordingDeliverer) snapshot() (called bool, note bridgedelivery.Notice) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.called, d.lastNote
}

func sampleNotice() bridgedelivery.Notice {
	return bridgedelivery.Notice{
		ToTeam:  "team-a",
		ToAgent: "agent-b",
		Subject: "notice subject",
		Body:    "notice body",
		TS:      1718000000000000001,
	}
}

func TestRegistry_DispatchesByTarget(t *testing.T) {
	cc := &recordingDeliverer{target: bridgedelivery.TargetCC}
	cursor := &recordingDeliverer{target: bridgedelivery.TargetCursor}
	codex := &recordingDeliverer{target: bridgedelivery.TargetCodex}

	reg := bridgedelivery.NewRegistry()
	reg.Register(cc)
	reg.Register(cursor)
	reg.Register(codex)

	ctx := context.Background()
	n := sampleNotice()

	cases := []struct {
		target bridgedelivery.Target
		del    *recordingDeliverer
	}{
		{bridgedelivery.TargetCC, cc},
		{bridgedelivery.TargetCursor, cursor},
		{bridgedelivery.TargetCodex, codex},
	}

	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			res := reg.Deliver(ctx, tc.target, n, bridgedelivery.DeliverOpts{})
			if !res.Delivered || res.Fallback {
				t.Fatalf("Deliver(%s) = %+v, want delivered without fallback", tc.target, res)
			}
			if res.Target != tc.target {
				t.Fatalf("Target = %q, want %q", res.Target, tc.target)
			}

			called, got := tc.del.snapshot()
			if !called {
				t.Fatal("expected deliverer to be called")
			}
			if got != n {
				t.Fatalf("notice = %+v, want %+v", got, n)
			}

			for _, other := range cases {
				if other.del == tc.del {
					continue
				}
				otherCalled, _ := other.del.snapshot()
				if otherCalled {
					t.Fatalf("deliverer %s should not have been called for target %s", other.target, tc.target)
				}
			}
		})
	}
}

func TestRegistry_FallbackOnError_LogsWarning(t *testing.T) {
	del := &recordingDeliverer{
		target: bridgedelivery.TargetCursor,
		err:    errors.New("stop hook unavailable"),
	}
	reg := bridgedelivery.NewRegistry()
	reg.Register(del)

	var logBuf bytes.Buffer
	res := reg.Deliver(context.Background(), bridgedelivery.TargetCursor, sampleNotice(), bridgedelivery.DeliverOpts{
		Logger: &logBuf,
	})

	if res.Delivered {
		t.Fatal("expected Delivered=false on error")
	}
	if !res.Fallback {
		t.Fatal("expected Fallback=true on error")
	}
	if res.ErrorReason == "" {
		t.Fatal("expected ErrorReason to be set")
	}

	logLine := strings.TrimSpace(logBuf.String())
	if !strings.Contains(logLine, "bridgedelivery: fallback to next turn") {
		t.Fatalf("log missing fallback warning: %q", logLine)
	}
	if !strings.Contains(logLine, "target=cursor") {
		t.Fatalf("log missing target: %q", logLine)
	}
}

func TestRegistry_LedgerEmitOnEachDeliver(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		reg := bridgedelivery.NewRegistry()
		reg.Register(&recordingDeliverer{target: bridgedelivery.TargetCC})

		var calls []bridgedelivery.DeliveryResult
		res := reg.Deliver(context.Background(), bridgedelivery.TargetCC, sampleNotice(), bridgedelivery.DeliverOpts{
			LedgerEmit: func(r bridgedelivery.DeliveryResult) { calls = append(calls, r) },
		})
		if !res.Delivered {
			t.Fatalf("unexpected failure: %+v", res)
		}
		if len(calls) != 1 {
			t.Fatalf("LedgerEmit calls = %d, want 1", len(calls))
		}
		if !calls[0].Delivered || calls[0].Fallback {
			t.Fatalf("ledger result = %+v, want success", calls[0])
		}
	})

	t.Run("failure", func(t *testing.T) {
		reg := bridgedelivery.NewRegistry()
		reg.Register(&recordingDeliverer{
			target: bridgedelivery.TargetCodex,
			err:    errors.New("inbox write failed"),
		})

		var calls []bridgedelivery.DeliveryResult
		res := reg.Deliver(context.Background(), bridgedelivery.TargetCodex, sampleNotice(), bridgedelivery.DeliverOpts{
			LedgerEmit: func(r bridgedelivery.DeliveryResult) { calls = append(calls, r) },
		})
		if res.Delivered || !res.Fallback {
			t.Fatalf("unexpected success: %+v", res)
		}
		if len(calls) != 1 {
			t.Fatalf("LedgerEmit calls = %d, want 1", len(calls))
		}
		if calls[0].Delivered || !calls[0].Fallback {
			t.Fatalf("ledger result = %+v, want fallback", calls[0])
		}
	})
}

func TestRegistry_UnregisteredTarget_FailOpenSkip(t *testing.T) {
	reg := bridgedelivery.NewRegistry()
	var logBuf bytes.Buffer

	res := reg.Deliver(context.Background(), bridgedelivery.Target("unknown"), sampleNotice(), bridgedelivery.DeliverOpts{
		Logger: &logBuf,
	})

	if res.Delivered {
		t.Fatal("expected Delivered=false for unregistered target")
	}
	if !res.Fallback {
		t.Fatal("expected Fallback=true for unregistered target")
	}

	logLine := strings.TrimSpace(logBuf.String())
	if !strings.Contains(logLine, "bridgedelivery: fallback to next turn") {
		t.Fatalf("log missing skip/fallback warning: %q", logLine)
	}
}
