package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "templates", "schemas", "bridge-event.v1.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found (templates/schemas/bridge-event.v1.json missing)")
		}
		dir = parent
	}
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(repoRoot(t), "tests", "fixtures", "bridge", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func schemaPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "templates", "schemas", "bridge-event.v1.json")
}
