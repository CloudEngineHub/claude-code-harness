package scopeleash

import (
	"reflect"
	"testing"
)

// samplePlan mimics a Plans.md v2 pipe table. The 101.7 row's DoD mentions two
// real path tokens plus prose and a command; only the file paths must be mined.
const samplePlan = `# Plans.md

## Phase 101

| Task | 内容 | DoD | Depends | Status |
|------|------|-----|---------|--------|
| 101.6 | 別タスク | ` + "`go/internal/other/baz.go`" + ` を編集 | - | cc:todo |
| 101.7 | 文章規範パイロット gate を ` + "`go/internal/writingnorms/scan.go`" + ` に実装 | (a) ` + "`scripts/check-writing-norms.sh`" + ` が JP 面を scan、(b) test ` + "`tests/test-writing-norms-gate.sh`" + ` が red→green、(c) 決定性のみ | 101.4 | cc:todo |
`

func TestScopeLeash_AutoInferredFromPlan(t *testing.T) {
	got, err := InferScopeFromPlan(samplePlan, "101.7")
	if err != nil {
		t.Fatalf("InferScopeFromPlan: unexpected error: %v", err)
	}
	want := []string{
		"go/internal/writingnorms/scan.go",
		"scripts/check-writing-norms.sh",
		"tests/test-writing-norms-gate.sh",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scope auto-inference mismatch (zero human input expected)\n got: %v\nwant: %v", got, want)
	}
	// It must NOT leak the neighboring task's path, proving row isolation.
	for _, p := range got {
		if p == "go/internal/other/baz.go" {
			t.Fatalf("inferred scope leaked another task's path: %v", got)
		}
	}
}

func TestScopeLeash_AutoInferred_NoHumanList_ProseIgnored(t *testing.T) {
	// A row with prose but no path tokens yields an empty scope, not prose words.
	plan := "| 9.9 | これはただの説明文で、決定性のみと書いてある | 完了条件を満たす | - | cc:todo |"
	got, err := InferScopeFromPlan(plan, "9.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("prose-only row should yield empty scope, got %v", got)
	}
}

func TestScopeLeash_OutOfScopeWriteAlarms(t *testing.T) {
	scope := []string{"go/internal/writingnorms/scan.go", "tests/test-writing-norms-gate.sh"}
	root := "/project"

	// In-scope write → allowed (no alarm).
	if !CheckWrite(scope, "/project/go/internal/writingnorms/scan.go", root) {
		t.Fatalf("in-scope write was wrongly flagged out-of-scope")
	}
	// Out-of-scope write → alarm (CheckWrite returns false).
	if CheckWrite(scope, "/project/go/internal/other/baz.go", root) {
		t.Fatalf("out-of-scope write was NOT flagged (alarm missed)")
	}
	// Repo-relative target (no projectRoot) also works.
	if !CheckWrite(scope, "tests/test-writing-norms-gate.sh", "") {
		t.Fatalf("repo-relative in-scope write was wrongly flagged")
	}
}

func TestScopeLeash_OutOfScopeWrite_DirPrefix(t *testing.T) {
	// A declared directory entry covers files beneath it.
	scope := []string{"go/internal/writingnorms"}
	if !CheckWrite(scope, "go/internal/writingnorms/scan.go", "") {
		t.Fatalf("dir-prefix scope should cover nested file")
	}
	if CheckWrite(scope, "go/internal/writingnormsX/scan.go", "") {
		t.Fatalf("dir-prefix must not match a sibling with shared name prefix")
	}
}

func TestScopeLeash_DroppedScopeFlagged(t *testing.T) {
	scope := []string{"a.go", "b.go", "scripts/c.sh"}
	touched := []string{"a.go", "scripts/c.sh"}
	got := DroppedScope(scope, touched)
	want := []string{"b.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dropped-scope detection mismatch\n got: %v\nwant: %v", got, want)
	}

	// Nothing dropped when all declared paths were touched.
	if d := DroppedScope(scope, []string{"a.go", "b.go", "scripts/c.sh"}); len(d) != 0 {
		t.Fatalf("expected no dropped scope, got %v", d)
	}
}
