package bridgedelivery

import (
	"context"
	"fmt"
)

const ccNoticeFromAgent = "bridgedelivery"

type ccDeliverer struct {
	livemsgSend func(ctx context.Context, team, from, to, subject, body string) (string, error)
}

func NewCCDeliverer(livemsgSend func(ctx context.Context, team, from, to, subject, body string) (string, error)) Deliverer {
	return &ccDeliverer{livemsgSend: livemsgSend}
}

func (d *ccDeliverer) Target() Target { return TargetCC }

func (d *ccDeliverer) Deliver(ctx context.Context, n Notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.livemsgSend == nil {
		return fmt.Errorf("cc deliverer not configured")
	}
	_, err := d.livemsgSend(ctx, n.ToTeam, ccNoticeFromAgent, n.ToAgent, n.Subject, n.Body)
	return err
}
