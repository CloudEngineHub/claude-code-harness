package mailbox

import (
	"context"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridge"
)

// MemIngestor は harness-mem への lane 別 ingest を表す。
type MemIngestor interface {
	// Record は mem への記録のみ。失敗で error を返してよいが、Store.Append は
	// それを log に吐いて握りつぶす（fail-open / 3 状態テスト規約）。
	Record(ctx context.Context, lane Lane, ev bridge.Event) error
	Audit(ctx context.Context, lane Lane, ev bridge.Event) error
	Alert(ctx context.Context, lane Lane, ev bridge.Event) error
}

// NoopIngestor は mem 未導入時の default。全メソッドが nil 返し。
type NoopIngestor struct{}

func (NoopIngestor) Record(context.Context, Lane, bridge.Event) error { return nil }
func (NoopIngestor) Audit(context.Context, Lane, bridge.Event) error  { return nil }
func (NoopIngestor) Alert(context.Context, Lane, bridge.Event) error  { return nil }
