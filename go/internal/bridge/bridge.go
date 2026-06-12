// Package bridge は 3 source（CC Mailbox / Cursor stop hook / Codex app-server）から
// 入る source-specific event を bridge-event.v1 envelope に normalize するための
// adapter 抽象を提供する。store 配線（mailbox 集約）は 95.1.2 で行う。
package bridge

import "errors"

type Source string

const (
	SourceCC     Source = "cc"
	SourceCursor Source = "cursor"
	SourceCodex  Source = "codex"
)

type Event struct {
	Source    Source                 `json:"source"`
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
	TS        int64                  `json:"ts"` // Unix nanos
}

// Adapter は source-specific 生 event JSON を Event 1 件に正規化する。
// 1 入力＝1 出力（バッチ化は呼び出し側責務）。
type Adapter interface {
	Source() Source
	Normalize(raw []byte) (Event, error)
}

var errNotImplemented = errors.New("bridge: not implemented")

type Registry struct {
	adapters map[Source]Adapter
}

func NewRegistry() *Registry {
	return &Registry{adapters: make(map[Source]Adapter)}
}

func (r *Registry) Register(a Adapter) {
	if r == nil || a == nil {
		return
	}
	r.adapters[a.Source()] = a
}

// Normalize は src を見て該当 adapter で正規化する。adapter 未登録 source は
// (Event{}, false, nil) を返し caller に skip させる（fail-open）。
// 1 行警告は呼び出し側で logger に出す（パッケージは stdout を汚さない）。
func (r *Registry) Normalize(src Source, raw []byte) (Event, bool, error) {
	a, ok := r.adapters[src]
	if !ok {
		return Event{}, false, nil
	}
	ev, err := a.Normalize(raw)
	return ev, true, err
}

type ccMailboxAdapter struct{}

func NewCCMailboxAdapter() Adapter { return &ccMailboxAdapter{} }

func (a *ccMailboxAdapter) Source() Source { return SourceCC }

func (a *ccMailboxAdapter) Normalize(raw []byte) (Event, error) {
	return Event{}, errNotImplemented
}

type cursorStopHookAdapter struct{}

func NewCursorStopHookAdapter() Adapter { return &cursorStopHookAdapter{} }

func (a *cursorStopHookAdapter) Source() Source { return SourceCursor }

func (a *cursorStopHookAdapter) Normalize(raw []byte) (Event, error) {
	return Event{}, errNotImplemented
}

type codexAppServerAdapter struct{}

func NewCodexAppServerAdapter() Adapter { return &codexAppServerAdapter{} }

func (a *codexAppServerAdapter) Source() Source { return SourceCodex }

func (a *codexAppServerAdapter) Normalize(raw []byte) (Event, error) {
	return Event{}, errNotImplemented
}
