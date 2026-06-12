package bridgedelivery

import (
	"context"
	"fmt"
)

type codexDeliverer struct {
	inboxBackendFn func(ctx context.Context, team, agent, subject, body string) error
}

func NewCodexDeliverer(inboxBackendFn func(ctx context.Context, team, agent, subject, body string) error) Deliverer {
	return &codexDeliverer{inboxBackendFn: inboxBackendFn}
}

func (d *codexDeliverer) Target() Target { return TargetCodex }

func (d *codexDeliverer) Deliver(ctx context.Context, n Notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.inboxBackendFn == nil {
		return fmt.Errorf("codex deliverer not configured")
	}
	return d.inboxBackendFn(ctx, n.ToTeam, n.ToAgent, n.Subject, n.Body)
}
