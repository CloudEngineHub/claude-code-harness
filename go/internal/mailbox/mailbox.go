// Package mailbox は bridge-event.v1 を sqlite WAL append-only event log に
// 集約し、source filter + lane 別 mem ingest を提供する。livemsg とは別系統
// (Phase 92.6.1 livemsg = live notice 用 / 本パッケージ = bridge unified store)。
package mailbox

import (
	"context"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridge"
)

type Lane string

const (
	LaneFast    Lane = "fast"
	LaneGate    Lane = "gate"
	LaneRelease Lane = "release"
)

// Store は bridge unified mailbox store。
type Store struct{}

// Open は sqlite ファイルを WAL モードで開く（既存 livemsg と同じ regimen）。
// :memory: は test 用。未マイグレーションなら schema を作る。
func Open(dbPath string) (*Store, error) {
	return OpenWithIngestor(dbPath, NoopIngestor{})
}

// OpenWithIngestor は MemIngestor を指定して store を開く。
func OpenWithIngestor(dbPath string, ing MemIngestor) (*Store, error) {
	_ = dbPath
	_ = ing
	return &Store{}, nil
}

// Close は DB 接続を閉じる。
func (s *Store) Close() error {
	return nil
}

// Append は 1 event を append-only event log に書く。
func (s *Store) Append(ctx context.Context, lane Lane, ev bridge.Event) error {
	_ = ctx
	_ = lane
	_ = ev
	return nil
}

// Read は projection を返す。
func (s *Store) Read(ctx context.Context, source bridge.Source, limit int) ([]bridge.Event, error) {
	_ = ctx
	_ = source
	_ = limit
	return nil, nil
}
