package companionresult

import (
	"encoding/json"
	"testing"
)

func TestNewSetsSchemaBackendTaskID(t *testing.T) {
	r := New("codex", "35.6.1")
	if r.Schema != SchemaID {
		t.Errorf("Schema = %q, want %q", r.Schema, SchemaID)
	}
	if r.Backend != "codex" {
		t.Errorf("Backend = %q, want codex", r.Backend)
	}
	if r.TaskID != "35.6.1" {
		t.Errorf("TaskID = %q, want 35.6.1", r.TaskID)
	}
	if r.FilesChanged == nil {
		t.Error("FilesChanged should be non-nil empty slice, got nil")
	}
}

func TestMarshalParseRoundTrip(t *testing.T) {
	orig := Normalize("cursor", "t1", 0, "edited two files\n", "", 1234)
	orig.FilesChanged = []string{"src/a.go", "src/b.go"}

	b, err := orig.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got.Schema != orig.Schema || got.Backend != orig.Backend ||
		got.TaskID != orig.TaskID || got.Success != orig.Success ||
		got.ExitCode != orig.ExitCode || got.Summary != orig.Summary ||
		got.DurationMs != orig.DurationMs {
		t.Errorf("round-trip scalar mismatch:\n  got  %+v\n  want %+v", got, orig)
	}
	if len(got.FilesChanged) != 2 || got.FilesChanged[0] != "src/a.go" || got.FilesChanged[1] != "src/b.go" {
		t.Errorf("FilesChanged round-trip mismatch: %#v", got.FilesChanged)
	}
}

func TestMarshalRendersFilesChangedAsEmptyArrayNotNull(t *testing.T) {
	b, err := New("codex", "t1").Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Confirm the wire form carries an empty array, never JSON null, so
	// downstream consumers can index without a nil guard.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if string(raw["files_changed"]) != "[]" {
		t.Errorf("files_changed wire form = %s, want []", raw["files_changed"])
	}
}

func TestParseRejectsWrongSchema(t *testing.T) {
	bad := []byte(`{"schema":"something-else.v1","backend":"codex","task_id":"t1"}`)
	if _, err := Parse(bad); err == nil {
		t.Fatal("Parse should reject a wrong schema, got nil error")
	}
}

func TestParseRejectsMissingSchema(t *testing.T) {
	bad := []byte(`{"backend":"codex","task_id":"t1"}`)
	if _, err := Parse(bad); err == nil {
		t.Fatal("Parse should reject a missing schema, got nil error")
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	if _, err := Parse([]byte("{not json")); err == nil {
		t.Fatal("Parse should reject invalid JSON, got nil error")
	}
}

func TestNormalizeExitZeroIsSuccess(t *testing.T) {
	r := Normalize("codex", "t1", 0, "all good\n", "", 10)
	if !r.Success {
		t.Error("exit 0 should be Success=true")
	}
	if r.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", r.ExitCode)
	}
	if r.Summary != "all good" {
		t.Errorf("Summary = %q, want %q", r.Summary, "all good")
	}
	if r.DurationMs != 10 {
		t.Errorf("DurationMs = %d, want 10", r.DurationMs)
	}
}

func TestNormalizeNonZeroIsFailure(t *testing.T) {
	r := Normalize("codex", "t1", 2, "", "boom: it broke\n", 5)
	if r.Success {
		t.Error("exit 2 should be Success=false")
	}
	if r.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", r.ExitCode)
	}
	// Summary falls back to the first stderr line when stdout is empty.
	if r.Summary != "boom: it broke" {
		t.Errorf("Summary = %q, want %q", r.Summary, "boom: it broke")
	}
}

func TestNormalizeSummaryPrefersFirstNonEmptyStdoutLine(t *testing.T) {
	stdout := "\n\n   \nfirst real line\nsecond line\n"
	r := Normalize("cursor", "t1", 0, stdout, "stderr line", 1)
	if r.Summary != "first real line" {
		t.Errorf("Summary = %q, want %q", r.Summary, "first real line")
	}
}

func TestNormalizeSummaryIsCapped(t *testing.T) {
	long := ""
	for i := 0; i < 500; i++ {
		long += "x"
	}
	r := Normalize("codex", "t1", 0, long, "", 1)
	runes := []rune(r.Summary)
	// summaryCap content + 1 ellipsis rune.
	if len(runes) != summaryCap+1 {
		t.Errorf("capped summary len = %d, want %d", len(runes), summaryCap+1)
	}
}

func TestNormalizeExtractsFilePaths(t *testing.T) {
	stdout := "Applied patch:\nsrc/foo.go\ninternal/bar/baz.ts\nsrc/foo.go\njust prose with spaces\nno_extension_token\n"
	r := Normalize("codex", "t1", 0, stdout, "", 1)
	want := []string{"src/foo.go", "internal/bar/baz.ts"}
	if len(r.FilesChanged) != len(want) {
		t.Fatalf("FilesChanged = %#v, want %#v", r.FilesChanged, want)
	}
	for i := range want {
		if r.FilesChanged[i] != want[i] {
			t.Errorf("FilesChanged[%d] = %q, want %q", i, r.FilesChanged[i], want[i])
		}
	}
}

func TestNormalizeDoesNotFabricateFilePaths(t *testing.T) {
	// No path-like tokens → empty (non-nil) slice, never invented.
	r := Normalize("cursor", "t1", 0, "Done. Nothing to report.\n", "", 1)
	if r.FilesChanged == nil {
		t.Fatal("FilesChanged should be non-nil empty slice")
	}
	if len(r.FilesChanged) != 0 {
		t.Errorf("FilesChanged should be empty, got %#v", r.FilesChanged)
	}
}

// TestNormalizeBothBackendsYieldSchema proves the normalization layer makes
// codex AND cursor sub-runs produce the same companion-result.v1 schema id.
func TestNormalizeBothBackendsYieldSchema(t *testing.T) {
	for _, backend := range []string{"codex", "cursor"} {
		r := Normalize(backend, "t1", 0, "ok\n", "", 1)
		if r.Schema != SchemaID {
			t.Errorf("backend %s: Schema = %q, want %q", backend, r.Schema, SchemaID)
		}
		if r.Backend != backend {
			t.Errorf("backend field = %q, want %q", r.Backend, backend)
		}
		// And it survives a marshal/parse cycle as companion-result.v1.
		b, err := r.Marshal()
		if err != nil {
			t.Fatalf("backend %s: Marshal: %v", backend, err)
		}
		parsed, err := Parse(b)
		if err != nil {
			t.Fatalf("backend %s: Parse: %v", backend, err)
		}
		if parsed.Schema != SchemaID {
			t.Errorf("backend %s: parsed Schema = %q, want %q", backend, parsed.Schema, SchemaID)
		}
	}
}
