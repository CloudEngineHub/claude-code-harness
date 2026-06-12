package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/livemsg"
)

type inboxCheckResult struct {
	Team     string                   `json:"team"`
	Agent    string                   `json:"agent"`
	Unread   int                      `json:"unread"`
	Messages []inboxCheckMessageEntry `json:"messages"`
}

type inboxCheckMessageEntry struct {
	ID        string `json:"id"`
	Team      string `json:"team"`
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

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
		"check",
		"--team", "team-a",
		"--agent", "agent-1",
		"--db", missing,
	})
	if code != 0 {
		t.Fatalf("expected exit 0 for missing DB (fail-open), got %d; stderr path ok", code)
	}

	var result inboxCheckResult
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

	args := []string{"check", "--team", team, "--agent", agent, "--db", dbPath}

	out1, code1 := runInboxCheckCapture(args)
	if code1 != 0 {
		t.Fatalf("first check exit = %d, want 0", code1)
	}
	var first inboxCheckResult
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
	var second inboxCheckResult
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

func TestInboxCheck_StableJSONOrder(t *testing.T) {
	team := "team-a"
	agent := "agent-2"

	runOnce := func() string {
		dbPath := seedInboxDB(t, team, "agent-1", agent, 2)
		args := []string{"check", "--team", team, "--agent", agent, "--db", dbPath}
		out, code := runInboxCheckCapture(args)
		if code != 0 {
			t.Fatalf("check exit = %d", code)
		}
		return out
	}

	a := runOnce()
	b := runOnce()
	if a != b {
		t.Errorf("inbox check JSON not bytes-identical:\nfirst:\n%s\nsecond:\n%s", a, b)
	}
}
