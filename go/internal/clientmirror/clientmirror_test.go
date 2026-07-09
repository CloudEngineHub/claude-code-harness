package clientmirror

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func schemaPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	return filepath.Join(repoRoot, SchemaRelPath)
}

func TestMirrorStateSchema_ValidFingerprint(t *testing.T) {
	schema := schemaPath(t)
	state := State{
		SchemaVersion: SchemaVersion,
		Fingerprint:   "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Healthy:       true,
		Reason:        ReasonInSync,
		Mirrors: []MirrorEntry{
			{
				Root:       "codex/.codex/skills",
				Status:     ReasonInSync,
				DriftCount: 0,
			},
		},
		TS: time.Now().Unix(),
	}
	if err := ValidateState(state, schema); err != nil {
		t.Fatalf("valid fingerprint state rejected: %v", err)
	}
}

func TestMirrorStateSchema_RejectExtraProperty(t *testing.T) {
	schema := schemaPath(t)
	raw := map[string]any{
		"schema_version": SchemaVersion,
		"fingerprint":    "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"healthy":        true,
		"reason":         ReasonInSync,
		"mirrors":        []any{},
		"unexpected":     true,
	}
	if err := validateSchemaInstance(raw, schema); err == nil {
		t.Fatal("expected schema reject for extra property")
	}
}

func validateSchemaInstance(instance any, schemaPath string) error {
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return err
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(SchemaURL, schemaDoc); err != nil {
		return err
	}
	schema, err := compiler.Compile(SchemaURL)
	if err != nil {
		return err
	}
	return schema.Validate(instance)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func writeSkillDir(t *testing.T, root, relSkill, body string) {
	t.Helper()
	dir := filepath.Join(root, relSkill)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestClientMirror_InSync(t *testing.T) {
	root := t.TempDir()
	body := "---\nname: demo\n---\n\n# Demo\n"
	writeSkillDir(t, root, "skills/demo", body)
	writeSkillDir(t, root, "codex/.codex/skills/demo", body)
	writeSkillDir(t, root, "opencode/skills/demo", body)

	state, err := Scan(root, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if state.Reason != ReasonInSync {
		t.Fatalf("reason = %q, want %q; mirrors=%+v", state.Reason, ReasonInSync, state.Mirrors)
	}
	if !state.Healthy {
		t.Fatalf("expected healthy state, got %+v", state)
	}
	if state.Fingerprint == "" || !strings.HasPrefix(state.Fingerprint, "sha256:") {
		t.Fatalf("fingerprint = %q", state.Fingerprint)
	}
}

func TestClientMirror_Drift(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "skills/demo", "---\nname: demo\n---\n\n# Demo SSOT\n")
	writeSkillDir(t, root, "codex/.codex/skills/demo", "---\nname: demo\n---\n\n# Demo Mirror Drift\n")

	state, err := Scan(root, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if state.Reason != ReasonDrift {
		t.Fatalf("reason = %q, want %q", state.Reason, ReasonDrift)
	}
	if state.Healthy {
		t.Fatal("expected unhealthy drift state")
	}

	drifts, err := Diff(root)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(drifts) == 0 {
		t.Fatal("expected drift messages")
	}
}

func TestClientMirror_MissingMirrorRoot(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "skills/demo", "---\nname: demo\n---\n\n# Demo\n")
	writeSkillDir(t, root, "codex/.codex/skills/demo", "---\nname: demo\n---\n\n# Demo\n")
	writeSkillDir(t, root, "opencode/skills/demo", "---\nname: demo\n---\n\n# Demo\n")
	// .agents/skills intentionally absent

	state, err := Scan(root, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var agents MirrorEntry
	found := false
	for _, mirror := range state.Mirrors {
		if mirror.Root == ".agents/skills" {
			agents = mirror
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing .agents/skills entry")
	}
	if agents.Status != ReasonNotConfigured {
		t.Fatalf(".agents status = %q, want %q", agents.Status, ReasonNotConfigured)
	}
	if agents.DriftCount != 0 {
		t.Fatalf(".agents drift_count = %d, want 0", agents.DriftCount)
	}
	if state.Reason != ReasonInSync {
		t.Fatalf("overall reason = %q, want %q (missing root must not count as drift)", state.Reason, ReasonInSync)
	}
}

func TestClientMirror_RepoHeadScan(t *testing.T) {
	root := repoRoot(t)
	state, err := Scan(root, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan repo root: %v", err)
	}
	if state.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version = %q", state.SchemaVersion)
	}
	if err := ValidateState(state, schemaPath(t)); err != nil {
		t.Fatalf("repo scan state invalid: %v", err)
	}
}
