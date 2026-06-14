package nightwatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func schemaPath(t *testing.T) string {
	t.Helper()
	return DefaultSchemaPath(repoRoot(t))
}

func TestNightWatchReportSchema_Valid(t *testing.T) {
	report := Report{
		SchemaVersion: "night-watch-report.v1",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Health:        ReportHealth{Healthy: true, Reason: ReasonNotConfigured},
		UnresolvedLoops: []UnresolvedLoop{},
		StaleTasks:      []StaleTask{},
		OpenDecisions:   []OpenDecision{},
	}
	if err := ValidateReport(report, schemaPath(t)); err != nil {
		t.Fatalf("valid report rejected: %v", err)
	}
}

func TestNightWatchReportSchema_RejectExtraProperty(t *testing.T) {
	report := Report{
		SchemaVersion:   "night-watch-report.v1",
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Health:          ReportHealth{Healthy: true, Reason: ""},
		UnresolvedLoops: []UnresolvedLoop{},
		StaleTasks:      []StaleTask{},
		OpenDecisions:   []OpenDecision{},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["unexpected_field"] = true
	if err := validateReportMap(raw, schemaPath(t)); err == nil {
		t.Fatal("expected schema reject for extra property")
	}
}

func TestNightWatchReportSchema_AdditionalPropertiesFalse(t *testing.T) {
	data, err := os.ReadFile(schemaPath(t))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}
	id, _ := schema["$id"].(string)
	if id == "" || !strings.Contains(id, "night-watch-report.v1") {
		t.Fatalf("$id = %q, want night-watch-report.v1", id)
	}
	if schema["additionalProperties"] != false {
		t.Fatalf("additionalProperties = %v, want false", schema["additionalProperties"])
	}
}

func TestBuildReport_DryRunValidJSON(t *testing.T) {
	root := repoRoot(t)
	report, err := BuildReport(BuildReportOptions{
		RepoRoot: root,
		DryRun:   true,
		Now:      time.Date(2026, 6, 14, 2, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if !report.DryRun {
		t.Fatal("expected dry_run=true")
	}
	if report.SchemaVersion != "night-watch-report.v1" {
		t.Fatalf("schema_version = %q", report.SchemaVersion)
	}
}
