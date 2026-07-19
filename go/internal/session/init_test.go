package session

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitHandler_Subagent(t *testing.T) {
	// サブエージェント時は軽量初期化
	h := &InitHandler{}
	inp := `{"agent_type":"subagent","session_id":"cc-123"}`
	var out bytes.Buffer
	err := h.Handle(strings.NewReader(inp), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp initResponse
	if err := json.Unmarshal(bytes.TrimRight(out.Bytes(), "\n"), &resp); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out.String())
	}
	if resp.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("expected HookEventName=SessionStart, got %q", resp.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "subagent") {
		t.Errorf("expected subagent context, got %q", resp.HookSpecificOutput.AdditionalContext)
	}
}

func TestInitHandler_NewSession(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")

	// Plans.md なし
	h := &InitHandler{
		StateDir:  stateDir,
		PlansFile: filepath.Join(dir, "Plans.md"),
	}

	inp := `{"session_id":"cc-456","agent_type":"","cwd":"` + dir + `"}`
	var out bytes.Buffer
	err := h.Handle(strings.NewReader(inp), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp initResponse
	if err := json.Unmarshal(bytes.TrimRight(out.Bytes(), "\n"), &resp); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out.String())
	}
	if resp.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("expected HookEventName=SessionStart, got %q", resp.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "Plans.md") {
		t.Errorf("expected Plans.md info in context, got %q", resp.HookSpecificOutput.AdditionalContext)
	}
	// マーカー凡例が含まれるか
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "cc:TODO") {
		t.Errorf("expected marker legend in context")
	}

	// session.json が作成されたか確認
	sessionFile := filepath.Join(stateDir, "session.json")
	if _, err := os.Stat(sessionFile); err != nil {
		t.Errorf("session.json not created: %v", err)
	}
}

func TestInitHandler_WithPlans(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	plansFile := filepath.Join(dir, "Plans.md")

	// Plans.md を作成
	content := `# Plans
| Task | 内容 | DoD | Depends | Status |
|---|---|---|---|---|
| task1 | A | d | - | cc:WIP |
| task2 | B | d | - | cc:TODO |
| task3 | C | d | - | cc:TODO |
| task4 | D | d | - | pm:依頼中 |
`
	if err := os.WriteFile(plansFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	h := &InitHandler{
		StateDir:  stateDir,
		PlansFile: plansFile,
	}

	var out bytes.Buffer
	err := h.Handle(strings.NewReader(`{"cwd":"`+dir+`"}`), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp initResponse
	if err := json.Unmarshal(bytes.TrimRight(out.Bytes(), "\n"), &resp); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out.String())
	}

	ctx := resp.HookSpecificOutput.AdditionalContext
	// WIP=1 (cc:WIP) + pm:依頼中=1 → 進行中 2
	// TODO=2
	if !strings.Contains(ctx, "進行中 2") {
		t.Errorf("expected 進行中 2 in context, got %q", ctx)
	}
	if !strings.Contains(ctx, "未着手 2") {
		t.Errorf("expected 未着手 2 in context, got %q", ctx)
	}
}

func TestInitHandler_SymlinkSessionFile(t *testing.T) {
	// シンボリックリンクの場合はセキュリティエラー（エラーは無視してレスポンスを返す）
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		t.Fatal(err)
	}

	// session.json をシンボリックリンクにする
	realFile := filepath.Join(dir, "real-session.json")
	if err := os.WriteFile(realFile, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	sessionLink := filepath.Join(stateDir, "session.json")
	if err := os.Symlink(realFile, sessionLink); err != nil {
		t.Skip("symlink creation not supported")
	}

	h := &InitHandler{
		StateDir:  stateDir,
		PlansFile: filepath.Join(dir, "Plans.md"),
	}

	var out bytes.Buffer
	// エラーにならず、レスポンスが返る
	err := h.Handle(strings.NewReader(`{}`), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// レスポンスが valid JSON であること
	var resp initResponse
	if json.Unmarshal(bytes.TrimRight(out.Bytes(), "\n"), &resp) != nil {
		t.Errorf("invalid JSON output: %s", out.String())
	}
}

func TestInitHandler_EmptyInput(t *testing.T) {
	dir := t.TempDir()
	h := &InitHandler{
		StateDir:  filepath.Join(dir, "state"),
		PlansFile: filepath.Join(dir, "Plans.md"),
	}

	var out bytes.Buffer
	err := h.Handle(strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp initResponse
	if json.Unmarshal(bytes.TrimRight(out.Bytes(), "\n"), &resp) != nil {
		t.Errorf("invalid JSON output: %s", out.String())
	}
	if resp.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("expected HookEventName=SessionStart")
	}
}

func TestBuildPlansInfo_StatusCellCounts(t *testing.T) {
	dir := t.TempDir()
	plansFile := filepath.Join(dir, "Plans.md")
	content := "| T1 | work | dod | - | cc:wip |\n| T2 | wait | dod | - | cc:todo |\n| T3 | pm | dod | - | pm:依頼中 |\n"
	if err := os.WriteFile(plansFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got := buildPlansInfo(plansFile)
	want := "Plans.md: 進行中 2 / 未着手 1"
	if got != want {
		t.Errorf("buildPlansInfo() = %q, want %q", got, want)
	}
}
