// Package bridgedelivery は非 CC peer への notice delivery を host hook 経由で
// 実装する。Cursor / Codex / CC それぞれの delivery payload shape を組み立て、
// 配達失敗時は次 turn fallback として 1 行警告 + ledger 記録する。
//
// 設計境界:
//   - store = mailbox / livemsg（別系統）。本パッケージは delivery transport のみ
//   - notice の中身（message 内容）は呼び出し側が決める。本パッケージは封筒
package bridgedelivery

import (
	"context"
	"fmt"
	"io"
)

type Target string

const (
	TargetCC     Target = "cc"
	TargetCursor Target = "cursor"
	TargetCodex  Target = "codex"
)

type Notice struct {
	ToTeam  string
	ToAgent string
	Subject string
	Body    string
	TS      int64
}

type DeliveryResult struct {
	Target      Target
	Delivered   bool
	Fallback    bool
	ErrorReason string
}

// Deliverer は host 固有の delivery 経路を実装するインターフェース。
// production 実装は host hook payload を組み立てる、テストでは fake を注入。
type Deliverer interface {
	Target() Target
	// Deliver は notice を host 固有の delivery payload に成形し、配達経路に
	// 投げる。Cursor は stop hook の followup_message、Codex は Bash PreToolUse
	// 連動のための inbox 経由 marker、CC は livemsg Send（92.6.1）+ Monitor。
	Deliver(ctx context.Context, n Notice) error
}

// Registry はターゲットごとに Deliverer を保持する。
type Registry struct {
	deliverers map[Target]Deliverer
}

func NewRegistry() *Registry {
	return &Registry{deliverers: make(map[Target]Deliverer)}
}

func (r *Registry) Register(d Deliverer) {
	if r == nil || d == nil {
		return
	}
	r.deliverers[d.Target()] = d
}

type DeliverOpts struct {
	Logger     io.Writer
	LedgerEmit func(res DeliveryResult)
}

// Deliver は target を見て該当 Deliverer で配達する。
// 失敗 (Deliverer が error 返却) → DeliveryResult{Delivered:false, Fallback:true,
// ErrorReason} を返し、Logger に「bridgedelivery: fallback to next turn (target=...,
// reason=...)」を 1 行 emit。エラー伝播はしない（fail-open）。
// LedgerEmit が nil でなければ毎 deliver で呼ぶ。
func (r *Registry) Deliver(ctx context.Context, target Target, n Notice, opts DeliverOpts) DeliveryResult {
	res := DeliveryResult{Target: target}

	if r == nil {
		res.Fallback = true
		res.ErrorReason = "nil registry"
		emitFallback(opts.Logger, target, res.ErrorReason)
		emitLedger(opts, res)
		return res
	}

	d, ok := r.deliverers[target]
	if !ok {
		res.Fallback = true
		res.ErrorReason = "unregistered deliverer"
		emitFallback(opts.Logger, target, res.ErrorReason)
		emitLedger(opts, res)
		return res
	}

	if err := d.Deliver(ctx, n); err != nil {
		res.Fallback = true
		res.ErrorReason = err.Error()
		emitFallback(opts.Logger, target, res.ErrorReason)
		emitLedger(opts, res)
		return res
	}

	res.Delivered = true
	emitLedger(opts, res)
	return res
}

func emitFallback(w io.Writer, target Target, reason string) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "bridgedelivery: fallback to next turn (target=%s, reason=%s)\n", target, reason)
}

func emitLedger(opts DeliverOpts, res DeliveryResult) {
	if opts.LedgerEmit != nil {
		opts.LedgerEmit(res)
	}
}
