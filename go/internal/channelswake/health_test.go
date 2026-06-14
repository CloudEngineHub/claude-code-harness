package channelswake

import (
	"database/sql"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func writeChannelsConfig(t *testing.T, home string, cfg channelsConfig) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "channels.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func writeMailboxDB(t *testing.T, dbPath string, lastTS int64) {
	t.Helper()
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS bridge_events (
  event_id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  event_type TEXT NOT NULL,
  lane TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  ts INTEGER NOT NULL
);`); err != nil {
		t.Fatal(err)
	}
	if lastTS > 0 {
		if _, err := db.Exec(
			`INSERT INTO bridge_events (event_id, source, event_type, lane, payload_json, ts)
			 VALUES ('evt-1', 'cc', 'stop', 'fast', '{}', ?)`, lastTS); err != nil {
			t.Fatal(err)
		}
	}
}

func startUnixSocket(t *testing.T) string {
	t.Helper()
	socketDir, err := os.MkdirTemp("/tmp", "cw")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })
	path := filepath.Join(socketDir, "b.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	return path
}

func TestChannelsWake_NotConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_BRIDGE_HOME", home)

	result, code := CheckWithExit()
	if code != 0 {
		t.Fatalf("expected exit 0 for not-configured, got %d", code)
	}
	if !result.Healthy {
		t.Fatalf("expected healthy=true for not-configured, got false")
	}
	if result.Reason != ReasonNotConfigured {
		t.Fatalf("expected reason=%q, got %q", ReasonNotConfigured, result.Reason)
	}
}

func TestChannelsWake_Unreachable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_BRIDGE_HOME", home)

	mailbox := filepath.Join(home, "mailbox.db")
	writeMailboxDB(t, mailbox, time.Now().UnixNano())
	writeChannelsConfig(t, home, channelsConfig{
		SocketPath: filepath.Join(home, "missing.sock"),
		MailboxDB:  mailbox,
	})

	result, code := CheckWithExit()
	if code == 0 {
		t.Fatal("expected non-zero exit for daemon-unreachable")
	}
	if result.Healthy {
		t.Fatal("expected healthy=false")
	}
	if result.Reason != ReasonDaemonUnreachable {
		t.Fatalf("expected reason=%q, got %q", ReasonDaemonUnreachable, result.Reason)
	}
}

func TestChannelsWake_Healthy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_BRIDGE_HOME", home)

	socketPath := startUnixSocket(t)
	mailbox := filepath.Join(home, "mailbox.db")
	writeMailboxDB(t, mailbox, time.Now().UnixNano())
	writeChannelsConfig(t, home, channelsConfig{
		SocketPath: socketPath,
		MailboxDB:  mailbox,
	})

	result, code := CheckWithExit()
	if code != 0 {
		t.Fatalf("expected exit 0 for healthy, got %d (reason=%q)", code, result.Reason)
	}
	if !result.Healthy {
		t.Fatalf("expected healthy=true, got false (reason=%q)", result.Reason)
	}
	if result.Reason != "" {
		t.Fatalf("expected empty reason for healthy, got %q", result.Reason)
	}
}

func TestChannelsWake_Corrupted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_BRIDGE_HOME", home)

	socketPath := startUnixSocket(t)
	writeChannelsConfig(t, home, channelsConfig{
		SocketPath:        socketPath,
		MailboxDB:         filepath.Join(home, "missing-mailbox.db"),
		StaleAfterSeconds: 60,
	})

	result, code := CheckWithExit()
	if code == 0 {
		t.Fatal("expected non-zero exit for corrupted")
	}
	if result.Healthy {
		t.Fatal("expected healthy=false")
	}
	if result.Reason != ReasonCorrupted {
		t.Fatalf("expected reason=%q, got %q", ReasonCorrupted, result.Reason)
	}
}

func TestChannelsWake_Corrupted_StaleMailbox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_BRIDGE_HOME", home)

	socketPath := startUnixSocket(t)
	mailbox := filepath.Join(home, "mailbox.db")
	oldTS := time.Now().Add(-2 * time.Hour).UnixNano()
	writeMailboxDB(t, mailbox, oldTS)
	writeChannelsConfig(t, home, channelsConfig{
		SocketPath:        socketPath,
		MailboxDB:         mailbox,
		StaleAfterSeconds: 60,
	})

	result, code := CheckWithExit()
	if code == 0 {
		t.Fatal("expected non-zero exit for stale mailbox")
	}
	if result.Reason != ReasonCorrupted {
		t.Fatalf("expected reason=%q, got %q", ReasonCorrupted, result.Reason)
	}
}

func TestChannelsWake_ReasonLiteralsPresent(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "health.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	for _, reason := range []string{ReasonNotConfigured, ReasonDaemonUnreachable, ReasonCorrupted} {
		if !strings.Contains(body, reason) {
			t.Fatalf("expected health.go to contain reason literal %q", reason)
		}
	}
}
