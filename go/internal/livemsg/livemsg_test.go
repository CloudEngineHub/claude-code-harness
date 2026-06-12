package livemsg_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/Chachamaru127/claude-code-harness/go/internal/livemsg"
)

func tempDB(t *testing.T) string {
	dir := t.TempDir()
	return filepath.Join(dir, "livemsg.db")
}

func openTestStore(t *testing.T) (*livemsg.Store, string) {
	path := tempDB(t)
	store, err := livemsg.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return store, path
}

func countEvents(t *testing.T, dbPath string, eventType string) int {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	var n int
	q := "SELECT COUNT(*) FROM livemsg_events"
	if eventType != "" {
		err = db.QueryRow(q+" WHERE event_type = ?", eventType).Scan(&n)
	} else {
		err = db.QueryRow(q).Scan(&n)
	}
	if err != nil {
		t.Fatalf("count events: %v", err)
	}
	return n
}

func TestStore_Send_AppendsEvent(t *testing.T) {
	store, path := openTestStore(t)
	defer store.Close()

	ctx := context.Background()
	_, err := store.Send(ctx, "team-a", "agent-1", "agent-2", "hello", "body")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if got := countEvents(t, path, "message_sent"); got != 1 {
		t.Fatalf("message_sent count = %d, want 1", got)
	}
}

func TestStore_MarkRead_AppendsEvent_NoDelete(t *testing.T) {
	store, path := openTestStore(t)
	defer store.Close()

	ctx := context.Background()
	msgID, err := store.Send(ctx, "team-a", "agent-1", "agent-2", "subj", "body")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	sentBefore := countEvents(t, path, "message_sent")
	if sentBefore != 1 {
		t.Fatalf("sent before MarkRead = %d, want 1", sentBefore)
	}

	if err := store.MarkRead(ctx, "team-a", msgID, "agent-2"); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	if got := countEvents(t, path, "message_read"); got != 1 {
		t.Fatalf("message_read count = %d, want 1", got)
	}
	if got := countEvents(t, path, "message_sent"); got != sentBefore {
		t.Fatalf("message_sent count = %d, want %d (append-only: no delete)", got, sentBefore)
	}
}

func TestStore_Inbox_ProjectsReadFlag(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()

	ctx := context.Background()
	team := "team-a"
	to := "agent-2"

	msgID, err := store.Send(ctx, team, "agent-1", to, "subj", "body")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	inbox, err := store.Inbox(ctx, team, to)
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("Inbox len = %d, want 1", len(inbox))
	}
	if inbox[0].ID != msgID {
		t.Fatalf("Inbox[0].ID = %q, want %q", inbox[0].ID, msgID)
	}

	if err := store.MarkRead(ctx, team, msgID, to); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	inbox, err = store.Inbox(ctx, team, to)
	if err != nil {
		t.Fatalf("Inbox after read: %v", err)
	}
	if len(inbox) != 0 {
		t.Fatalf("Inbox after read len = %d, want 0", len(inbox))
	}

	history, err := store.History(ctx, team, to)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("History len = %d, want 1", len(history))
	}
}

func TestStore_MarkRead_Idempotent(t *testing.T) {
	store, path := openTestStore(t)
	defer store.Close()

	ctx := context.Background()
	team := "team-a"
	to := "agent-2"

	msgID, err := store.Send(ctx, team, "agent-1", to, "subj", "body")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := store.MarkRead(ctx, team, msgID, to); err != nil {
			t.Fatalf("MarkRead #%d: %v", i, err)
		}
	}

	readCount := countEvents(t, path, "message_read")
	if readCount < 3 {
		t.Fatalf("message_read count = %d, want >= 3", readCount)
	}

	inbox, err := store.Inbox(ctx, team, to)
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	if len(inbox) != 0 {
		t.Fatalf("Inbox len = %d, want 0", len(inbox))
	}

	history, err := store.History(ctx, team, to)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("History len = %d, want 1", len(history))
	}
}

func TestStore_Inbox_OrderedByCreatedAt(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()

	ctx := context.Background()
	team := "team-a"
	to := "agent-2"

	var ids []string
	for i := 0; i < 3; i++ {
		id, err := store.Send(ctx, team, "agent-1", to, "subj", "body")
		if err != nil {
			t.Fatalf("Send #%d: %v", i, err)
		}
		ids = append(ids, id)
	}

	inbox, err := store.Inbox(ctx, team, to)
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	if len(inbox) != 3 {
		t.Fatalf("Inbox len = %d, want 3", len(inbox))
	}
	for i := 0; i < 3; i++ {
		if inbox[i].ID != ids[i] {
			t.Fatalf("inbox[%d].ID = %q, want %q", i, inbox[i].ID, ids[i])
		}
		if i > 0 && !inbox[i].CreatedAt.After(inbox[i-1].CreatedAt) && inbox[i].CreatedAt.Equal(inbox[i-1].CreatedAt) == false {
			// allow equal or after
		}
		if i > 0 && inbox[i].CreatedAt.Before(inbox[i-1].CreatedAt) {
			t.Fatalf("inbox[%d] CreatedAt before inbox[%d]", i, i-1)
		}
	}
}

func TestStore_HarnessMemNotRequired(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// harness-mem ディレクトリは作らない（mem 非依存契約）
	memPath := filepath.Join(home, ".harness-mem")
	if _, err := os.Stat(memPath); err == nil {
		t.Fatalf("unexpected harness-mem at %s", memPath)
	}

	dbPath := filepath.Join(t.TempDir(), "livemsg.db")
	store, err := livemsg.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	msgID, err := store.Send(ctx, "team", "a1", "a2", "s", "b")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := store.MarkRead(ctx, "team", msgID, "a2"); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	inbox, err := store.Inbox(ctx, "team", "a2")
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	if len(inbox) != 0 {
		t.Fatalf("Inbox len = %d, want 0", len(inbox))
	}
}

func TestStore_NoSignalAPIReferences(t *testing.T) {
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	forbidden := []string{"signal_send", "signal_read", "signal_ack", "workgraph"}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", e.Name(), err)
		}
		content := string(data)
		for _, token := range forbidden {
			if strings.Contains(content, token) {
				t.Fatalf("file %s contains forbidden token %q", e.Name(), token)
			}
		}
	}
}

func TestStore_AttributionInSource(t *testing.T) {
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	var foundAgmsg, foundMIT bool
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", e.Name(), err)
		}
		content := string(data)
		if strings.Contains(content, "agmsg") {
			foundAgmsg = true
		}
		if strings.Contains(content, "MIT") {
			foundMIT = true
		}
	}
	if !foundAgmsg {
		t.Fatal("no .go file contains attribution token \"agmsg\"")
	}
	if !foundMIT {
		t.Fatal("no .go file contains attribution token \"MIT\"")
	}
}
