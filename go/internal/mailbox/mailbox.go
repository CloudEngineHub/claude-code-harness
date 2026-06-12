// Package mailbox は bridge-event.v1 を sqlite WAL append-only event log に
// 集約し、source filter + lane 別 mem ingest を提供する。Phase 92.6.1 の
// live-notice 系統とは別パッケージ（本パッケージ = bridge unified store）。
package mailbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridge"

	_ "modernc.org/sqlite" // SQLite ドライバ登録（副作用のみ使用）
)

type Lane string

const (
	LaneFast    Lane = "fast"
	LaneGate    Lane = "gate"
	LaneRelease Lane = "release"
)

// Store は bridge unified mailbox store。
type Store struct {
	db     *sql.DB
	ingest MemIngestor
}

const schemaDDL = `
CREATE TABLE IF NOT EXISTS bridge_events (
  event_id     TEXT PRIMARY KEY,
  source       TEXT NOT NULL CHECK(source IN ('cc','cursor','codex')),
  event_type   TEXT NOT NULL,
  lane         TEXT NOT NULL CHECK(lane IN ('fast','gate','release')),
  payload_json TEXT NOT NULL,
  ts           INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS bridge_events_source_ts ON bridge_events(source, ts);
CREATE INDEX IF NOT EXISTS bridge_events_lane ON bridge_events(lane, ts);
`

var idSeq atomic.Uint64

// Open は sqlite ファイルを WAL モードで開く。:memory: は test 用。
func Open(dbPath string) (*Store, error) {
	return OpenWithIngestor(dbPath, NoopIngestor{})
}

// OpenWithIngestor は MemIngestor を指定して store を開く。
func OpenWithIngestor(dbPath string, ing MemIngestor) (*Store, error) {
	if ing == nil {
		ing = NoopIngestor{}
	}

	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("mailbox: mkdir %s: %w", dir, err)
			}
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("mailbox: open db %s: %w", dbPath, err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("mailbox: set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("mailbox: set busy_timeout: %w", err)
	}

	store := &Store{db: db, ingest: ing}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) initSchema() error {
	if _, err := s.db.Exec(schemaDDL); err != nil {
		return fmt.Errorf("mailbox: init schema: %w", err)
	}
	return nil
}

// Close は DB 接続を閉じる。
func (s *Store) Close() error {
	return s.db.Close()
}

func newEventID() string {
	seq := idSeq.Add(1)
	return fmt.Sprintf("evt-%d-%06d", time.Now().UnixNano(), seq%1_000_000)
}

func validateLane(l Lane) error {
	switch l {
	case LaneFast, LaneGate, LaneRelease:
		return nil
	default:
		return fmt.Errorf("mailbox: invalid lane %q", l)
	}
}

// Append は 1 event を append-only event log に書く。
func (s *Store) Append(ctx context.Context, lane Lane, ev bridge.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateLane(lane); err != nil {
		return err
	}

	payloadJSON := "{}"
	if ev.Payload != nil {
		b, err := json.Marshal(ev.Payload)
		if err != nil {
			return fmt.Errorf("mailbox: marshal payload: %w", err)
		}
		payloadJSON = string(b)
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO bridge_events (event_id, source, event_type, lane, payload_json, ts)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		newEventID(), string(ev.Source), ev.EventType, string(lane), payloadJSON, ev.TS,
	)
	if err != nil {
		return fmt.Errorf("mailbox: append: %w", err)
	}

	go s.runIngest(context.Background(), lane, ev)
	return nil
}

func (s *Store) runIngest(ctx context.Context, lane Lane, ev bridge.Event) {
	record := func(name string, fn func(context.Context, Lane, bridge.Event) error) {
		if err := fn(ctx, lane, ev); err != nil {
			log.Printf("mailbox: ingest %s (lane=%s source=%s type=%s): %v", name, lane, ev.Source, ev.EventType, err)
		}
	}

	record("record", s.ingest.Record)

	switch lane {
	case LaneGate:
		record("audit", s.ingest.Audit)
	case LaneRelease:
		record("alert", s.ingest.Alert)
	}
}

// Read は projection を返す。
func (s *Store) Read(ctx context.Context, source bridge.Source, limit int) ([]bridge.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	query := `SELECT source, event_type, payload_json, ts FROM bridge_events`
	args := []interface{}{}

	if source != "" {
		query += ` WHERE source = ?`
		args = append(args, string(source))
	}
	query += ` ORDER BY ts ASC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mailbox: read: %w", err)
	}
	defer rows.Close()

	var events []bridge.Event
	for rows.Next() {
		var (
			src         string
			eventType   string
			payloadJSON string
			ts          int64
		)
		if err := rows.Scan(&src, &eventType, &payloadJSON, &ts); err != nil {
			return nil, fmt.Errorf("mailbox: scan event: %w", err)
		}

		payload := map[string]interface{}{}
		if payloadJSON != "" && payloadJSON != "{}" {
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
				return nil, fmt.Errorf("mailbox: unmarshal payload: %w", err)
			}
		}

		events = append(events, bridge.Event{
			Source:    bridge.Source(src),
			EventType: eventType,
			Payload:   payload,
			TS:        ts,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mailbox: iterate events: %w", err)
	}
	if events == nil {
		return []bridge.Event{}, nil
	}
	return events, nil
}
