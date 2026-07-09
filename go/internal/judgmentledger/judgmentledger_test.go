package judgmentledger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func schemaPath(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	return DefaultSchemaPath(root)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, SchemaRelPath)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", wd)
		}
		dir = parent
	}
}

func TestJudgmentLedger_Append(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "judgment-ledger.jsonl")
	schema := schemaPath(t)

	err := Append(AppendOpts{
		LedgerPath: ledger,
		SchemaPath: schema,
		Record: Record{
			ID:        "jl-test-001",
			Project:   "demo-project",
			DecidedAt: "2026-06-14T00:00:00Z",
			Question:  "Redis or Postgres for cache?",
			Answer:    "redis",
			Rationale: "scale requirement",
			CardRef:   "/tmp/card.json",
			Tags:      []string{"judgment-card"},
		},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	data, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if !strings.Contains(string(data), `"id":"jl-test-001"`) {
		t.Fatalf("ledger line missing id: %s", data)
	}

	records, err := LoadByProject(ledger, "demo-project")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Answer != "redis" {
		t.Fatalf("answer = %q, want redis", records[0].Answer)
	}
}

func TestJudgmentLedger_SchemaReject(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "judgment-ledger.jsonl")
	schema := schemaPath(t)

	err := Append(AppendOpts{
		LedgerPath: ledger,
		SchemaPath: schema,
		Record: Record{
			ID:        "jl-invalid",
			Project:   "demo-project",
			DecidedAt: "2026-06-14T00:00:00Z",
			Question:  "",
			Answer:    "a",
			Rationale: "",
			CardRef:   "card.json",
			Tags:      []string{},
		},
	})
	if err == nil {
		t.Fatal("expected schema reject for empty question")
	}
	if _, statErr := os.Stat(ledger); statErr == nil {
		t.Fatal("invalid record must not be written")
	}
}

func TestJudgmentLedger_FailOpen(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	badLedger := filepath.Join(blocker, "ledger.jsonl")
	schema := schemaPath(t)

	AppendFailOpen(AppendOpts{
		LedgerPath: badLedger,
		SchemaPath: schema,
		Record: Record{
			ID:        "jl-failopen",
			Project:   "demo",
			DecidedAt: "2026-06-14T00:00:00Z",
			Question:  "q",
			Answer:    "a",
			Rationale: "",
			CardRef:   "c.json",
			Tags:      []string{},
		},
	})
}

func TestJudgmentLedger_ProjectScope(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "judgment-ledger.jsonl")
	schema := schemaPath(t)

	for _, spec := range []struct {
		id, project string
	}{
		{"jl-a", "proj-a"},
		{"jl-b", "proj-b"},
		{"jl-a2", "proj-a"},
	} {
		if err := Append(AppendOpts{
			LedgerPath: ledger,
			SchemaPath: schema,
			Record: Record{
				ID:        spec.id,
				Project:   spec.project,
				DecidedAt: "2026-06-14T00:00:00Z",
				Question:  "question",
				Answer:    "answer",
				Rationale: "",
				CardRef:   "card.json",
				Tags:      []string{},
			},
		}); err != nil {
			t.Fatalf("append %s: %v", spec.id, err)
		}
	}

	records, err := LoadByProject(ledger, "proj-a")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("proj-a count = %d, want 2", len(records))
	}
	for _, rec := range records {
		if rec.Project != "proj-a" {
			t.Fatalf("unexpected project %q in scoped result", rec.Project)
		}
	}
}

func seedLedger(t *testing.T, ledger, schema string, records []Record) {
	t.Helper()
	for _, rec := range records {
		if err := Append(AppendOpts{LedgerPath: ledger, SchemaPath: schema, Record: rec}); err != nil {
			t.Fatalf("seed append: %v", err)
		}
	}
}

func TestJudgmentLedgerIndex_Top3(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "judgment-ledger.jsonl")
	schema := schemaPath(t)

	seedLedger(t, ledger, schema, []Record{
		{ID: "1", Project: "p", DecidedAt: "2026-06-01T00:00:00Z", Question: "redis cache sizing", Answer: "a1", CardRef: "c1", Tags: []string{}},
		{ID: "2", Project: "p", DecidedAt: "2026-06-02T00:00:00Z", Question: "redis persistence mode", Answer: "a2", CardRef: "c2", Tags: []string{}},
		{ID: "3", Project: "p", DecidedAt: "2026-06-03T00:00:00Z", Question: "redis cluster mode", Answer: "a3", CardRef: "c3", Tags: []string{}},
		{ID: "4", Project: "p", DecidedAt: "2026-06-04T00:00:00Z", Question: "unrelated topic", Answer: "a4", CardRef: "c4", Tags: []string{}},
	})

	hits, err := Search(ledger, "p", "redis", 3)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("got %d hits, want 3", len(hits))
	}
	for _, hit := range hits {
		if !strings.Contains(strings.ToLower(hit.Question), "redis") {
			t.Fatalf("unexpected hit without redis: %+v", hit)
		}
	}
}

func TestJudgmentLedgerIndex_ProjectScopeIsolation(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "judgment-ledger.jsonl")
	schema := schemaPath(t)

	seedLedger(t, ledger, schema, []Record{
		{ID: "a1", Project: "alpha", DecidedAt: "2026-06-01T00:00:00Z", Question: "shared redis keyword", Answer: "x", CardRef: "c1", Tags: []string{}},
		{ID: "b1", Project: "beta", DecidedAt: "2026-06-02T00:00:00Z", Question: "shared redis keyword", Answer: "y", CardRef: "c2", Tags: []string{}},
	})

	hits, err := Search(ledger, "alpha", "redis", 3)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1", len(hits))
	}
	if hits[0].Project != "alpha" {
		t.Fatalf("project = %q, want alpha", hits[0].Project)
	}
}

func TestJudgmentLedgerIndex_EmptyCorpus(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "missing.jsonl")

	hits, err := Search(ledger, "demo", "redis", 3)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("got %d hits, want 0", len(hits))
	}
}

func TestJudgmentLedger_RecallSimilar(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "judgment-ledger.jsonl")
	schema := schemaPath(t)

	seedLedger(t, ledger, schema, []Record{
		{ID: "r1", Project: "p", DecidedAt: "2026-06-01T00:00:00Z", Question: "Use Redis for session store?", Answer: "yes", Rationale: "latency", CardRef: "c1", Tags: []string{}},
	})

	past, err := RecallSimilar(ledger, "p", "session store", 3)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(past) != 1 {
		t.Fatalf("got %d past decisions, want 1", len(past))
	}
	if past[0].MemID != "judgment-ledger:r1" {
		t.Fatalf("mem_id = %q", past[0].MemID)
	}
	if past[0].Decision != "yes" {
		t.Fatalf("decision = %q", past[0].Decision)
	}
}
