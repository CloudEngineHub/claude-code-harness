package mailbox_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridge"
	"github.com/Chachamaru127/claude-code-harness/go/internal/mailbox"
)

type fakeIngestor struct {
	mu    sync.Mutex
	rec   int
	audit int
	alert int
	err   error
}

func (f *fakeIngestor) Record(context.Context, mailbox.Lane, bridge.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec++
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeIngestor) Audit(context.Context, mailbox.Lane, bridge.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.audit++
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeIngestor) Alert(context.Context, mailbox.Lane, bridge.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alert++
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeIngestor) counts() (record, audit, alert int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rec, f.audit, f.alert
}

func (f *fakeIngestor) waitCounts(t *testing.T, wantRec, wantAudit, wantAlert int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec, audit, alert := f.counts()
		if rec == wantRec && audit == wantAudit && alert == wantAlert {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	rec, audit, alert := f.counts()
	t.Fatalf("ingestor counts = (%d,%d,%d), want (%d,%d,%d)", rec, audit, alert, wantRec, wantAudit, wantAlert)
}

func openTestStore(t *testing.T, ing mailbox.MemIngestor) *mailbox.Store {
	t.Helper()
	store, err := mailbox.OpenWithIngestor(":memory:", ing)
	if err != nil {
		t.Fatalf("OpenWithIngestor: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestStore_Append_ThreeSourcesUnifiedOrder(t *testing.T) {
	store := openTestStore(t, mailbox.NoopIngestor{})
	ctx := context.Background()

	events := []bridge.Event{
		{Source: bridge.SourceCC, EventType: "ping", Payload: map[string]interface{}{"n": 1}, TS: 100},
		{Source: bridge.SourceCursor, EventType: "stop", Payload: map[string]interface{}{"n": 2}, TS: 200},
		{Source: bridge.SourceCodex, EventType: "done", Payload: map[string]interface{}{"n": 3}, TS: 300},
	}

	for _, ev := range events {
		if err := store.Append(ctx, mailbox.LaneFast, ev); err != nil {
			t.Fatalf("Append(%s): %v", ev.Source, err)
		}
	}

	got, err := store.Read(ctx, "", 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(Read) = %d, want 3", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].TS < got[i-1].TS {
			t.Fatalf("TS order broken at %d: %d < %d", i, got[i].TS, got[i-1].TS)
		}
	}
	if got[0].Source != bridge.SourceCC || got[2].Source != bridge.SourceCodex {
		t.Fatalf("unexpected order/sources: %+v", got)
	}
}

func TestStore_Read_FilterBySource(t *testing.T) {
	store := openTestStore(t, mailbox.NoopIngestor{})
	ctx := context.Background()

	for _, ev := range []bridge.Event{
		{Source: bridge.SourceCC, EventType: "a", Payload: nil, TS: 1},
		{Source: bridge.SourceCursor, EventType: "b", Payload: nil, TS: 2},
		{Source: bridge.SourceCodex, EventType: "c", Payload: nil, TS: 3},
	} {
		if err := store.Append(ctx, mailbox.LaneFast, ev); err != nil {
			t.Fatal(err)
		}
	}

	got, err := store.Read(ctx, bridge.SourceCursor, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Source != bridge.SourceCursor || got[0].EventType != "b" {
		t.Fatalf("got %+v", got[0])
	}
}

func TestStore_Read_LimitRespected(t *testing.T) {
	store := openTestStore(t, mailbox.NoopIngestor{})
	ctx := context.Background()

	for i := int64(1); i <= 5; i++ {
		ev := bridge.Event{Source: bridge.SourceCC, EventType: "tick", Payload: nil, TS: i * 10}
		if err := store.Append(ctx, mailbox.LaneFast, ev); err != nil {
			t.Fatal(err)
		}
	}

	got, err := store.Read(ctx, "", 2)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].TS != 10 || got[1].TS != 20 {
		t.Fatalf("expected first two by TS, got %+v", got)
	}
}

func TestStore_AppendIsAppendOnly(t *testing.T) {
	mailboxDir := filepath.Join("..", "mailbox")
	entries, err := os.ReadDir(mailboxDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(mailboxDir, e.Name()))
		if readErr != nil {
			t.Fatalf("ReadFile %s: %v", e.Name(), readErr)
		}
		upper := strings.ToUpper(string(body))
		if strings.Contains(upper, "DELETE FROM") || strings.Contains(upper, "UPDATE ") {
			t.Fatalf("%s contains DELETE/UPDATE SQL (append-only violation)", e.Name())
		}
	}

	store := openTestStore(t, mailbox.NoopIngestor{})
	ctx := context.Background()
	ev := bridge.Event{Source: bridge.SourceCC, EventType: "once", Payload: map[string]interface{}{"k": "v"}, TS: 42}
	if err := store.Append(ctx, mailbox.LaneFast, ev); err != nil {
		t.Fatal(err)
	}
	all, err := store.Read(ctx, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("len after append = %d, want 1", len(all))
	}
	if err := store.Append(ctx, mailbox.LaneFast, ev); err != nil {
		t.Fatal(err)
	}
	all, err = store.Read(ctx, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("append-only: len = %d, want 2 distinct rows", len(all))
	}
}

func TestStore_LaneFast_IngestRecordsOnly(t *testing.T) {
	ing := &fakeIngestor{}
	store := openTestStore(t, ing)
	ctx := context.Background()
	ev := bridge.Event{Source: bridge.SourceCC, EventType: "x", Payload: nil, TS: 1}

	if err := store.Append(ctx, mailbox.LaneFast, ev); err != nil {
		t.Fatal(err)
	}
	ing.waitCounts(t, 1, 0, 0)
}

func TestStore_LaneGate_RecordsPlusAudit(t *testing.T) {
	ing := &fakeIngestor{}
	store := openTestStore(t, ing)
	ctx := context.Background()
	ev := bridge.Event{Source: bridge.SourceCC, EventType: "x", Payload: nil, TS: 1}

	if err := store.Append(ctx, mailbox.LaneGate, ev); err != nil {
		t.Fatal(err)
	}
	ing.waitCounts(t, 1, 1, 0)
}

func TestStore_LaneRelease_RecordsPlusAlert(t *testing.T) {
	ing := &fakeIngestor{}
	store := openTestStore(t, ing)
	ctx := context.Background()
	ev := bridge.Event{Source: bridge.SourceCC, EventType: "x", Payload: nil, TS: 1}

	if err := store.Append(ctx, mailbox.LaneRelease, ev); err != nil {
		t.Fatal(err)
	}
	ing.waitCounts(t, 1, 0, 1)
}

func TestStore_FailOpen_IngestError(t *testing.T) {
	ing := &fakeIngestor{err: errors.New("mem down")}
	store := openTestStore(t, ing)
	ctx := context.Background()
	ev := bridge.Event{Source: bridge.SourceCC, EventType: "x", Payload: nil, TS: 99}

	if err := store.Append(ctx, mailbox.LaneGate, ev); err != nil {
		t.Fatalf("Append should succeed (fail-open): %v", err)
	}

	got, err := store.Read(ctx, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("bridge_events row count = %d, want 1", len(got))
	}
	ing.waitCounts(t, 1, 1, 0)
}

func TestStore_NotConfiguredHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	store, err := mailbox.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	ev := bridge.Event{Source: bridge.SourceCodex, EventType: "boot", Payload: nil, TS: 7}
	if err := store.Append(ctx, mailbox.LaneFast, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := store.Read(ctx, bridge.SourceCodex, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestStore_LaneEnumValidation(t *testing.T) {
	store := openTestStore(t, mailbox.NoopIngestor{})
	ctx := context.Background()
	ev := bridge.Event{Source: bridge.SourceCC, EventType: "x", Payload: nil, TS: 1}

	err := store.Append(ctx, mailbox.Lane("bogus"), ev)
	if err == nil {
		t.Fatal("expected error for bogus lane")
	}
}

func TestStore_NoLivemsgCoupling_GrepAudit(t *testing.T) {
	mailboxDir := filepath.Join("..", "mailbox")
	entries, err := os.ReadDir(mailboxDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	const forbidden = "livemsg"
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(mailboxDir, name))
		if readErr != nil {
			t.Fatalf("ReadFile %s: %v", name, readErr)
		}
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("%s contains %q token (coupling audit)", name, forbidden)
		}
	}

	// shell grep でも二重確認（_test.go 除外）
	mailboxAbs, err := filepath.Abs(mailboxDir)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("grep", "-r", forbidden, mailboxAbs, "--include=*.go", "--exclude=*_test.go")
	out, grepErr := cmd.CombinedOutput()
	if grepErr == nil && len(strings.TrimSpace(string(out))) > 0 {
		t.Fatalf("forbidden token found in mailbox package:\n%s", out)
	}
	if grepErr != nil {
		exitErr, ok := grepErr.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("grep failed: %v\n%s", grepErr, out)
		}
	}
}
