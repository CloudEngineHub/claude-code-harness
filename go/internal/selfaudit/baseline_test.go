package selfaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const emptyDenySHA256 = "4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945"

func TestComputeDenyHash_OrderIndependent(t *testing.T) {
	a := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)","Write(.claude/settings*)"]}}`)
	b := []byte(`{"permissions":{"deny":["Write(.claude/settings*)","Bash(sudo:*)","Edit(.claude/settings*)"]}}`)

	hashA, entriesA, err := ComputeDenyHash(a)
	if err != nil {
		t.Fatalf("ComputeDenyHash(a): %v", err)
	}
	hashB, entriesB, err := ComputeDenyHash(b)
	if err != nil {
		t.Fatalf("ComputeDenyHash(b): %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hash mismatch: %q vs %q", hashA, hashB)
	}
	if len(entriesA) != 3 || len(entriesB) != 3 {
		t.Fatalf("entries len = %d / %d, want 3", len(entriesA), len(entriesB))
	}
}

func TestComputeDenyHash_DuplicateRemoved(t *testing.T) {
	raw := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Bash(sudo:*)","Edit(.claude/settings*)"]}}`)
	hash, entries, err := ComputeDenyHash(raw)
	if err != nil {
		t.Fatalf("ComputeDenyHash: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2 (duplicates removed)", len(entries))
	}
	single := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)"]}}`)
	hashSingle, _, err := ComputeDenyHash(single)
	if err != nil {
		t.Fatalf("ComputeDenyHash(single): %v", err)
	}
	if hash != hashSingle {
		t.Fatalf("hash with duplicates = %q, want %q", hash, hashSingle)
	}
}

func TestComputeDenyHash_EmptyDeny(t *testing.T) {
	raw := []byte(`{"permissions":{"deny":[]}}`)
	hash, entries, err := ComputeDenyHash(raw)
	if err != nil {
		t.Fatalf("ComputeDenyHash: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries len = %d, want 0", len(entries))
	}
	if hash != emptyDenySHA256 {
		t.Fatalf("hash = %q, want %q", hash, emptyDenySHA256)
	}
}

func TestVerifyDenyNotRegressed_NoChange_OK(t *testing.T) {
	settings := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)"]}}`)
	hash, entries, err := ComputeDenyHash(settings)
	if err != nil {
		t.Fatalf("ComputeDenyHash: %v", err)
	}
	baseline := DenyBaseline{
		Version:         "deny-baseline.v1",
		CanonicalSHA256: hash,
		Entries:         entries,
	}
	ok, reason, err := VerifyDenyNotRegressed(baseline, settings)
	if err != nil {
		t.Fatalf("VerifyDenyNotRegressed: %v", err)
	}
	if !ok {
		t.Fatalf("ok = false, reason = %q", reason)
	}
	if reason != "" {
		t.Fatalf("reason = %q, want empty", reason)
	}
}

func TestVerifyDenyNotRegressed_AddedEntry_OK(t *testing.T) {
	baselineSettings := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)"]}}`)
	currentSettings := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)","Write(.claude/settings*)"]}}`)
	hash, entries, err := ComputeDenyHash(baselineSettings)
	if err != nil {
		t.Fatalf("ComputeDenyHash: %v", err)
	}
	baseline := DenyBaseline{
		Version:         "deny-baseline.v1",
		CanonicalSHA256: hash,
		Entries:         entries,
	}
	ok, reason, err := VerifyDenyNotRegressed(baseline, currentSettings)
	if err != nil {
		t.Fatalf("VerifyDenyNotRegressed: %v", err)
	}
	if !ok {
		t.Fatalf("ok = false, reason = %q", reason)
	}
}

func TestVerifyDenyNotRegressed_RemovedEntry_NG(t *testing.T) {
	baselineSettings := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)","Write(.claude/settings*)"]}}`)
	currentSettings := []byte(`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)"]}}`)
	hash, entries, err := ComputeDenyHash(baselineSettings)
	if err != nil {
		t.Fatalf("ComputeDenyHash: %v", err)
	}
	baseline := DenyBaseline{
		Version:         "deny-baseline.v1",
		CanonicalSHA256: hash,
		Entries:         entries,
	}
	ok, reason, err := VerifyDenyNotRegressed(baseline, currentSettings)
	if err != nil {
		t.Fatalf("VerifyDenyNotRegressed: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false for removed entry")
	}
	if !strings.Contains(reason, "Write(.claude/settings*)") {
		t.Fatalf("reason = %q, want removed entry name", reason)
	}
}

func TestVerifyDenyNotRegressed_RenamedPattern_NG(t *testing.T) {
	baselineSettings := []byte(`{"permissions":{"deny":["Edit(.claude/settings*)"]}}`)
	currentSettings := []byte(`{"permissions":{"deny":["Edit(.claude/settings.json)"]}}`)
	hash, entries, err := ComputeDenyHash(baselineSettings)
	if err != nil {
		t.Fatalf("ComputeDenyHash: %v", err)
	}
	baseline := DenyBaseline{
		Version:         "deny-baseline.v1",
		CanonicalSHA256: hash,
		Entries:         entries,
	}
	ok, reason, err := VerifyDenyNotRegressed(baseline, currentSettings)
	if err != nil {
		t.Fatalf("VerifyDenyNotRegressed: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false for renamed pattern")
	}
	if !strings.Contains(reason, "Edit(.claude/settings*)") {
		t.Fatalf("reason = %q, want missing baseline entry", reason)
	}
}

func TestLoadBaseline_FileMissing_NoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-baseline.json")
	baseline, loaded, err := LoadBaseline(path)
	if err != nil {
		t.Fatalf("LoadBaseline: %v", err)
	}
	if loaded {
		t.Fatal("loaded = true, want false for missing file")
	}
	if baseline.Version != "" || baseline.CanonicalSHA256 != "" || len(baseline.Entries) != 0 {
		t.Fatalf("expected zero DenyBaseline, got %+v", baseline)
	}
}

func TestBaseline_NeverWritesRealUserSettings(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	forbiddenPairs := []struct {
		a, b string
	}{
		{"os.WriteFile", "UserHomeDir"},
		{"os.WriteFile", "os.Getenv(\"HOME\")"},
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".go") || strings.HasSuffix(ent.Name(), "_test.go") {
			continue
		}
		if ent.Name() != "baseline.go" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, ent.Name()))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		for _, pair := range forbiddenPairs {
			if strings.Contains(content, pair.a) && strings.Contains(content, pair.b) {
				t.Errorf("%s: forbidden combination %q + %q", ent.Name(), pair.a, pair.b)
			}
		}
	}
}
