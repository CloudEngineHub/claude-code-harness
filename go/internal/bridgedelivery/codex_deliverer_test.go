package bridgedelivery_test

import (
	"context"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridgedelivery"
)

func TestCodexDeliverer_PayloadGoesToInbox(t *testing.T) {
	var (
		gotTeam    string
		gotAgent   string
		gotSubject string
		gotBody    string
	)

	del := bridgedelivery.NewCodexDeliverer(func(_ context.Context, team, agent, subject, body string) error {
		gotTeam, gotAgent, gotSubject, gotBody = team, agent, subject, body
		return nil
	})

	n := sampleNotice()
	if err := del.Deliver(context.Background(), n); err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if gotTeam != n.ToTeam || gotAgent != n.ToAgent {
		t.Fatalf("team/agent = (%q,%q), want (%q,%q)", gotTeam, gotAgent, n.ToTeam, n.ToAgent)
	}
	if gotSubject != n.Subject || gotBody != n.Body {
		t.Fatalf("subject/body = (%q,%q), want (%q,%q)", gotSubject, gotBody, n.Subject, n.Body)
	}
}
