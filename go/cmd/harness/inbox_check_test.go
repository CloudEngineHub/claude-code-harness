package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/livemsg"
)

func seedInboxDB(t *testing.T, team, from, to string, count int) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "livemsg.db")
	store, err := livemsg.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	for i := 0; i < count; i++ {
		subject := "subj"
		body := "body"
		if _, err := store.Send(ctx, team, from, to, subject, body); err != nil {
			t.Fatalf("Send #%d: %v", i, err)
		}
	}
	return dbPath
}

func runInboxCheckCapture(args []string) (string, int) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInboxCheckCommand(args, &stdout, &stderr)
	return stdout.String(), code
}

func TestInboxCheck_EmptyDB(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-livemsg.db")
	out, code := runInboxCheckCapture([]string{
		"--team", "team-a",
		"--agent", "agent-1",
		"--db", missing,
	})
	if code != 0 {
		t.Fatalf("expected exit 0 for missing DB (fail-open), got %d; stderr path ok", code)
	}

	var result inboxCheckOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if result.Unread != 0 {
		t.Errorf("unread = %d, want 0", result.Unread)
	}
	if result.Team != "team-a" {
		t.Errorf("team = %q, want team-a", result.Team)
	}
	if result.Agent != "agent-1" {
		t.Errorf("agent = %q, want agent-1", result.Agent)
	}
	if len(result.Messages) != 0 {
		t.Errorf("messages len = %d, want 0", len(result.Messages))
	}
}

func TestInboxCheck_ReadsAndMarksRead(t *testing.T) {
	team := "team-a"
	agent := "agent-2"
	dbPath := seedInboxDB(t, team, "agent-1", agent, 2)

	args := []string{"--team", team, "--agent", agent, "--db", dbPath}

	out1, code1 := runInboxCheckCapture(args)
	if code1 != 0 {
		t.Fatalf("first check exit = %d, want 0", code1)
	}
	var first inboxCheckOutput
	if err := json.Unmarshal([]byte(out1), &first); err != nil {
		t.Fatalf("first JSON: %v\nraw: %s", err, out1)
	}
	if first.Unread != 2 {
		t.Fatalf("first unread = %d, want 2", first.Unread)
	}
	if len(first.Messages) != 2 {
		t.Fatalf("first messages len = %d, want 2", len(first.Messages))
	}

	out2, code2 := runInboxCheckCapture(args)
	if code2 != 0 {
		t.Fatalf("second check exit = %d, want 0", code2)
	}
	var second inboxCheckOutput
	if err := json.Unmarshal([]byte(out2), &second); err != nil {
		t.Fatalf("second JSON: %v\nraw: %s", err, out2)
	}
	if second.Unread != 0 {
		t.Fatalf("second unread = %d, want 0 after MarkRead", second.Unread)
	}
	if len(second.Messages) != 0 {
		t.Fatalf("second messages len = %d, want 0", len(second.Messages))
	}
}

func copyDBFile(t *testing.T, src string) string {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "livemsg-copy.db")
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return dst
}

func TestInboxCheck_StableJSONOrder(t *testing.T) {
	team := "team-a"
	agent := "agent-2"
	src := seedInboxDB(t, team, "agent-1", agent, 2)

	runOnCopy := func() string {
		dbPath := copyDBFile(t, src)
		args := []string{"--team", team, "--agent", agent, "--db", dbPath}
		out, code := runInboxCheckCapture(args)
		if code != 0 {
			t.Fatalf("check exit = %d", code)
		}
		return out
	}

	a := runOnCopy()
	b := runOnCopy()
	if a != b {
		t.Errorf("inbox check JSON not bytes-identical:\nfirst:\n%s\nsecond:\n%s", a, b)
	}
}
