package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/pkg/hookproto"
)

func TestSubagentStopReviewerPersistBackstopWritesReviewResult(t *testing.T) {
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, ".claude", "state"), 0o755); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}
	transcript := filepath.Join(cwd, "transcript.jsonl")
	reviewBlock := `{"schema_version":"review-result.v1","verdict":"APPROVE","reviewer_profile":"static","task":"109.1a","gaps":[],"followups":[]}`
	line := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role": "assistant",
			"content": []map[string]string{{
				"type": "text",
				"text": "review complete\n```json\n" + reviewBlock + "\n```",
			}},
		},
	}
	data, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("marshal transcript: %v", err)
	}
	if err := os.WriteFile(transcript, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	input := hookproto.HookInput{
		SessionID:      "sess-109",
		TranscriptPath: transcript,
		CWD:            cwd,
		ToolName:       "Task",
		ToolInput:      map[string]interface{}{},
	}
	if err := persistReviewerResultBackstop(input); err != nil {
		t.Fatalf("persistReviewerResultBackstop: %v", err)
	}

	out := filepath.Join(cwd, ".claude", "state", "review-result.json")
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("review-result.json not written: %v", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("review-result.json is not JSON: %v", err)
	}
	if obj["schema_version"] != "review-result.v1" || obj["verdict"] != "APPROVE" {
		t.Fatalf("unexpected review result: %s", got)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude", "state", "review-approved.json")); err != nil {
		t.Fatalf("legacy approval file not written: %v", err)
	}
}

func TestSubagentStopReviewerPersistBackstopNoopsWithoutSchema(t *testing.T) {
	cwd := t.TempDir()
	transcript := filepath.Join(cwd, "transcript.jsonl")
	if err := os.WriteFile(transcript, []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"worker done"}]}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	input := hookproto.HookInput{TranscriptPath: transcript, CWD: cwd, ToolName: "Task", ToolInput: map[string]interface{}{}}
	if err := persistReviewerResultBackstop(input); err != nil {
		t.Fatalf("noop should not error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude", "state", "review-result.json")); !os.IsNotExist(err) {
		t.Fatalf("review-result.json should not be written for non-reviewer text")
	}
}
