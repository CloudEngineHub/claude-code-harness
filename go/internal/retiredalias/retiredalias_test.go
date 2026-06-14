package retiredalias

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "templates", "registry", "retired-aliases.v1.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root from test file")
		}
		dir = parent
	}
}

func TestRetiredAlias_RegistrySchemaValid(t *testing.T) {
	root := testRepoRoot(t)
	registryPath := filepath.Join(root, "templates", "registry", "retired-aliases.v1.yaml")
	schemaPath := filepath.Join(root, "templates", "schemas", "retired-alias.v1.json")

	if err := ValidateRegistrySchema(registryPath, schemaPath); err != nil {
		t.Fatalf("registry schema validation failed: %v", err)
	}
}

func TestRetiredAlias_ResidueDetected(t *testing.T) {
	root := testRepoRoot(t)
	fixtureRoot := filepath.Join(root, "tests", "fixtures", "retired-alias")

	reg := &Registry{
		Version: 1,
		Entries: []Entry{
			{
				ID:      "fixture-path",
				Kind:    KindPath,
				Pattern: "core/src/guardrails/rules.ts",
			},
		},
	}

	hits, err := Scan(fixtureRoot, reg, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected residue hits in fixture, got 0")
	}
	found := false
	for _, h := range hits {
		if strings.Contains(h.File, "residue-sample.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected hit in residue-sample.txt, got: %+v", hits)
	}
}

func TestRetiredAlias_AllowlistRespected(t *testing.T) {
	root := testRepoRoot(t)
	fixtureRoot := filepath.Join(root, "tests", "fixtures", "retired-alias")

	reg := &Registry{
		Version: 1,
		Entries: []Entry{
			{
				ID:      "fixture-concept",
				Kind:    KindConcept,
				Pattern: "TypeScript guardrail engine",
				Allowlist: []string{
					"allowlisted/",
				},
			},
		},
	}

	hits, err := Scan(fixtureRoot, reg, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	for _, h := range hits {
		if strings.Contains(h.File, "allowlisted") {
			t.Fatalf("allowlisted path should not be reported: %+v", h)
		}
	}
}

func TestRetiredAlias_HeadZeroHits(t *testing.T) {
	root := testRepoRoot(t)
	registryPath := DefaultRegistryPath(root)

	reg, err := LoadRegistry(registryPath)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}

	hits, err := Scan(root, reg, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(hits) > 0 {
		var lines []string
		for _, h := range hits {
			lines = append(lines, h.String())
		}
		t.Fatalf("expected 0 hits on HEAD, got %d:\n%s", len(hits), strings.Join(lines, "\n"))
	}
}
