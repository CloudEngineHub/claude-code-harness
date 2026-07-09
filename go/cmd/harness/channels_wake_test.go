package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/channelswake"

	_ "modernc.org/sqlite"
)

func TestRunChannelsWakeCheck_NotConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_BRIDGE_HOME", home)

	var stdout, stderr stringsBuffer
	code := runChannelsWakeCommand([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, stderr.String())
	}

	var result channelswake.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !result.Healthy || result.Reason != channelswake.ReasonNotConfigured {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunChannelsWakeCheck_Unreachable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_BRIDGE_HOME", home)

	writeJSONFile(t, filepath.Join(home, "channels.json"), map[string]string{
		"socket_path": filepath.Join(home, "missing.sock"),
		"mailbox_db":  filepath.Join(home, "mailbox.db"),
	})
	writeMinimalMailbox(t, filepath.Join(home, "mailbox.db"))

	var stdout, stderr stringsBuffer
	code := runChannelsWakeCommand([]string{"check"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1 for unreachable, got %d", code)
	}

	var result channelswake.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Healthy || result.Reason != channelswake.ReasonDaemonUnreachable {
		t.Fatalf("unexpected result: %+v", result)
	}
}

type stringsBuffer struct {
	buf []byte
}

func (b *stringsBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *stringsBuffer) Bytes() []byte  { return b.buf }
func (b *stringsBuffer) String() string { return string(b.buf) }

func writeJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func writeMinimalMailbox(t *testing.T, dbPath string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
CREATE TABLE bridge_events (
  event_id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  event_type TEXT NOT NULL,
  lane TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  ts INTEGER NOT NULL
);
INSERT INTO bridge_events (event_id, source, event_type, lane, payload_json, ts)
VALUES ('evt-1', 'cc', 'stop', 'fast', '{}', 1);`); err != nil {
		t.Fatal(err)
	}
}
