package nightwatch

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestLoadStaleConfig_Literals(t *testing.T) {
	root := repoRoot(t)
	cfg, err := LoadStaleConfig(filepath.Join(root, ConfigRelPath))
	if err != nil {
		t.Fatalf("LoadStaleConfig: %v", err)
	}
	if cfg.StaleTaskHours != 72 {
		t.Fatalf("stale_task_hours = %d, want 72", cfg.StaleTaskHours)
	}
	if cfg.OpenDecisionHours != 168 {
		t.Fatalf("open_decision_hours = %d, want 168", cfg.OpenDecisionHours)
	}

	data, err := os.ReadFile(filepath.Join(root, ConfigRelPath))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "stale_task_hours: 72") {
		t.Fatal("expected stale_task_hours: 72 literal in config template")
	}
	if !strings.Contains(body, "open_decision_hours: 168") {
		t.Fatal("expected open_decision_hours: 168 literal in config template")
	}
}

func TestDetectStaleTasks_StaleWIP(t *testing.T) {
	dir := t.TempDir()
	plansPath := filepath.Join(dir, "Plans.md")
	content := "| 99.1.1 | stale task | DoD | deps | cc:WIP |\n"
	if err := os.WriteFile(plansPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-80 * time.Hour)
	if err := os.Chtimes(plansPath, old, old); err != nil {
		t.Fatal(err)
	}

	stale, err := DetectStaleTasks(plansPath, 72, time.Now())
	if err != nil {
		t.Fatalf("DetectStaleTasks: %v", err)
	}
	if len(stale) != 1 {
		t.Fatalf("got %d stale tasks, want 1", len(stale))
	}
	if stale[0].TaskID != "99.1.1" {
		t.Fatalf("task_id = %q", stale[0].TaskID)
	}
}

func TestDetectOpenDecisions_StaleOpen(t *testing.T) {
	dir := t.TempDir()
	decisionsPath := filepath.Join(dir, "decisions.md")
	oldDate := time.Now().Add(-200 * time.Hour).Format("2006-01-02")
	content := "## " + oldDate + ": Pending auth provider #decision\n\n**Status**: Open\n"
	if err := os.WriteFile(decisionsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	open, err := DetectOpenDecisions(decisionsPath, 168, time.Now())
	if err != nil {
		t.Fatalf("DetectOpenDecisions: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("got %d open decisions, want 1", len(open))
	}
	if !strings.Contains(open[0].Title, "Pending auth provider") {
		t.Fatalf("title = %q", open[0].Title)
	}
}

func TestUnresolvedLoopsFromMailbox_RequestWithoutResponse(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "mailbox.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE bridge_events (
  event_id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  event_type TEXT NOT NULL,
  lane TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  ts INTEGER NOT NULL
);`); err != nil {
		t.Fatal(err)
	}
	oldTS := time.Now().Add(-3 * time.Hour).UnixNano()
	_, err = db.Exec(
		`INSERT INTO bridge_events (event_id, source, event_type, lane, payload_json, ts)
		 VALUES ('evt-req', 'cc', 'advisor-request', 'fast', '{"task_id":"t1","trigger_hash":"abc"}', ?)`, oldTS)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	loops, err := UnresolvedLoopsFromMailbox(dbPath, time.Now())
	if err != nil {
		t.Fatalf("UnresolvedLoopsFromMailbox: %v", err)
	}
	if len(loops) != 1 {
		t.Fatalf("got %d loops, want 1", len(loops))
	}
	if loops[0].TaskID != "t1" {
		t.Fatalf("task_id = %q", loops[0].TaskID)
	}
}
