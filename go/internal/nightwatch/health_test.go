package nightwatch

import (
	"database/sql"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/eventstore"

	_ "modernc.org/sqlite"
)

func writeBridgeChannelsConfig(t *testing.T, home, socketPath, mailboxDB string) {
	t.Helper()
	cfg := map[string]string{
		"socket_path": socketPath,
		"mailbox_db":  mailboxDB,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "channels.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func writeNightWatchConfig(t *testing.T, home string, enabled bool) {
	t.Helper()
	data, err := json.Marshal(map[string]bool{"enabled": enabled})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "night-watch.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func startUnixSocket(t *testing.T) string {
	t.Helper()
	socketDir, err := os.MkdirTemp("/tmp", "nw")
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

func writeMailboxDB(t *testing.T, dbPath string) {
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
	if err := eventstore.EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
}

func TestNightWatch_NotConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_NIGHT_WATCH_HOME", home)
	t.Setenv("HARNESS_BRIDGE_HOME", filepath.Join(home, "bridge"))
	t.Setenv("NIGHT_WATCH_ENABLED", "false")

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

func TestNightWatch_Unreachable(t *testing.T) {
	home := t.TempDir()
	bridgeHome := filepath.Join(home, "bridge")
	t.Setenv("HARNESS_NIGHT_WATCH_HOME", home)
	t.Setenv("HARNESS_BRIDGE_HOME", bridgeHome)
	t.Setenv("NIGHT_WATCH_ENABLED", "true")

	mailbox := filepath.Join(bridgeHome, "mailbox.db")
	writeMailboxDB(t, mailbox)
	writeBridgeChannelsConfig(t, bridgeHome, filepath.Join(bridgeHome, "missing.sock"), mailbox)

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

func TestNightWatch_Healthy(t *testing.T) {
	home := t.TempDir()
	bridgeHome := filepath.Join(home, "bridge")
	t.Setenv("HARNESS_NIGHT_WATCH_HOME", home)
	t.Setenv("HARNESS_BRIDGE_HOME", bridgeHome)
	t.Setenv("NIGHT_WATCH_ENABLED", "true")

	socketPath := startUnixSocket(t)
	mailbox := filepath.Join(bridgeHome, "mailbox.db")
	writeMailboxDB(t, mailbox)
	writeBridgeChannelsConfig(t, bridgeHome, socketPath, mailbox)

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

func TestNightWatch_Corrupted(t *testing.T) {
	home := t.TempDir()
	bridgeHome := filepath.Join(home, "bridge")
	t.Setenv("HARNESS_NIGHT_WATCH_HOME", home)
	t.Setenv("HARNESS_BRIDGE_HOME", bridgeHome)
	t.Setenv("NIGHT_WATCH_ENABLED", "true")

	if err := os.MkdirAll(bridgeHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bridgeHome, "channels.json"), []byte("{bad json"), 0600); err != nil {
		t.Fatal(err)
	}

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

func TestNightWatch_ReasonLiteralsPresent(t *testing.T) {
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
