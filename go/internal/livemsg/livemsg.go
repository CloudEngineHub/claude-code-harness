// Package livemsg は CCH 自前の live-notice message store を提供する（Mode 2 用）。
//
// Inspired by the agmsg pattern (MIT license). agmsg は append-only event log と
// read-projection の組み合わせで durable な agent-to-agent messaging を実現する
// 軽量パターンであり、本パッケージはその「append-only events + projection」モデルを
// 参考に CCH 文脈へ翻案している。
//
// 設計境界:
//   - durable work-handoff = harness-mem signal（別系統、本パッケージは触らない）
//   - live notice messaging = 本パッケージ（mem 未導入でも動く）
package livemsg

import (
	"context"
	"time"
)

// Store は live-notice message の append-only event store。
type Store struct{}

// Message は projection された 1 件のメッセージ。
type Message struct {
	ID        string
	Team      string
	FromAgent string
	ToAgent   string
	Subject   string
	Body      string
	CreatedAt time.Time
}

// Open は sqlite ファイル（WAL mode）を開き、未マイグレーションなら schema を作る。
func Open(dbPath string) (*Store, error) {
	return nil, errNotImplemented
}

// Close は DB 接続を閉じる。
func (s *Store) Close() error {
	return errNotImplemented
}

// Send は message_sent event を 1 行 append し、生成された Message ID を返す。
func (s *Store) Send(ctx context.Context, team, from, to, subject, body string) (string, error) {
	return "", errNotImplemented
}

// MarkRead は message_read event を 1 行 append する。
func (s *Store) MarkRead(ctx context.Context, team, messageID, byAgent string) error {
	return errNotImplemented
}

// Inbox は (team, agent) 宛 unread message を CreatedAt 昇順で返す。
func (s *Store) Inbox(ctx context.Context, team, agent string) ([]Message, error) {
	return nil, errNotImplemented
}

// History は (team, agent) が受信した全 message を CreatedAt 昇順で返す。
func (s *Store) History(ctx context.Context, team, agent string) ([]Message, error) {
	return nil, errNotImplemented
}

var errNotImplemented = &notImplementedError{}

type notImplementedError struct{}

func (e *notImplementedError) Error() string { return "livemsg: not implemented" }
