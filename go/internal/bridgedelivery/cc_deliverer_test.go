package bridgedelivery_test

import (
	"context"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridgedelivery"
)

func TestCCDeliverer_UsesLivemsgSend(t *testing.T) {
	var (
		gotTeam    string
		gotFrom    string
		gotTo      string
		gotSubject string
		gotBody    string
	)

	del := bridgedelivery.NewCCDeliverer(func(_ context.Context, team, from, to, subject, body string) (string, error) {
		gotTeam, gotFrom, gotTo, gotSubject, gotBody = team, from, to, subject, body
		return "msg-123", nil
	})

	n := sampleNotice()
	if err := del.Deliver(context.Background(), n); err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if gotTeam != n.ToTeam || gotTo != n.ToAgent {
		t.Fatalf("team/to = (%q,%q), want (%q,%q)", gotTeam, gotTo, n.ToTeam, n.ToAgent)
	}
	if gotFrom == "" {
		t.Fatal("expected non-empty from agent")
	}
	if gotSubject != n.Subject || gotBody != n.Body {
		t.Fatalf("subject/body = (%q,%q), want (%q,%q)", gotSubject, gotBody, n.Subject, n.Body)
	}
}
