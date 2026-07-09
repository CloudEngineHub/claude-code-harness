package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/pkg/hookproto"
)

type transcriptLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

type reviewResultV1 struct {
	SchemaVersion   string        `json:"schema_version"`
	GeneratedAt     string        `json:"generated_at"`
	Verdict         string        `json:"verdict"`
	ReviewerProfile string        `json:"reviewer_profile"`
	Task            interface{}   `json:"task"`
	Type            interface{}   `json:"type"`
	CommitHash      interface{}   `json:"commit_hash"`
	Checks          []interface{} `json:"checks"`
	Gaps            []reviewGap   `json:"gaps"`
	Followups       []interface{} `json:"followups"`
	DualReview      interface{}   `json:"dual_review"`
}

type reviewGap struct {
	Severity string `json:"severity,omitempty"`
}

func persistReviewerResultBackstop(input hookproto.HookInput) error {
	if input.TranscriptPath == "" {
		return nil
	}
	text, err := lastAssistantText(input.TranscriptPath)
	if err != nil || strings.TrimSpace(text) == "" {
		return nil
	}
	block := extractReviewResultJSON(text)
	if block == "" {
		return nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(block), &raw); err != nil {
		return nil
	}
	if raw["schema_version"] != "review-result.v1" {
		return nil
	}

	cwd := input.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if cwd == "" {
		return nil
	}
	result := normalizeReviewResult(raw)
	stateDir := filepath.Join(cwd, ".claude", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	out := filepath.Join(stateDir, "review-result.json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	if result.Verdict == "APPROVE" {
		legacy := map[string]interface{}{
			"approved_at": result.GeneratedAt,
			"judgment":    "APPROVE",
			"commit_hash": nil,
		}
		legacyData, _ := json.MarshalIndent(legacy, "", "  ")
		legacyData = append(legacyData, '\n')
		if err := os.WriteFile(filepath.Join(stateDir, "review-approved.json"), legacyData, 0o644); err != nil {
			return err
		}
	} else {
		_ = os.Remove(filepath.Join(stateDir, "review-approved.json"))
	}
	return appendSubagentStopPersistAudit(stateDir, input.SessionID, result.Verdict)
}

func lastAssistantText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	last := ""
	for scanner.Scan() {
		var line transcriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "assistant" {
			continue
		}
		var parts []string
		for _, c := range line.Message.Content {
			if c.Type == "text" && c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
		if len(parts) > 0 {
			last = strings.Join(parts, "\n")
		}
	}
	return last, scanner.Err()
}

func extractReviewResultJSON(text string) string {
	if block := extractFencedReviewJSON(text); block != "" {
		return block
	}
	return extractBalancedReviewJSON(text)
}

func extractFencedReviewJSON(text string) string {
	lines := strings.Split(text, "\n")
	inBlock := false
	var b strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock && trimmed == "```json" {
			inBlock = true
			b.Reset()
			continue
		}
		if inBlock && trimmed == "```" {
			candidate := b.String()
			if strings.Contains(candidate, "review-result.v1") {
				return strings.TrimSpace(candidate)
			}
			inBlock = false
			continue
		}
		if inBlock {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return ""
}

func extractBalancedReviewJSON(text string) string {
	for start := 0; start < len(text); start++ {
		if text[start] != '{' {
			continue
		}
		depth := 0
		inString := false
		escaped := false
		for i := start; i < len(text); i++ {
			c := text[i]
			if inString {
				if escaped {
					escaped = false
				} else if c == '\\' {
					escaped = true
				} else if c == '"' {
					inString = false
				}
				continue
			}
			switch c {
			case '"':
				inString = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					candidate := text[start : i+1]
					if strings.Contains(candidate, "review-result.v1") && json.Valid([]byte(candidate)) {
						return candidate
					}
					start = i
					break
				}
			}
		}
	}
	return ""
}

func normalizeReviewResult(raw map[string]interface{}) reviewResultV1 {
	verdict := normalizeReviewVerdict(stringValue(raw["verdict"], stringValue(raw["judgment"], "REQUEST_CHANGES")))
	gaps := normalizeReviewGaps(raw["gaps"])
	for _, gap := range gaps {
		switch strings.ToLower(gap.Severity) {
		case "critical", "high", "major":
			verdict = "REQUEST_CHANGES"
		}
	}
	return reviewResultV1{
		SchemaVersion:   "review-result.v1",
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Verdict:         verdict,
		ReviewerProfile: stringValue(raw["reviewer_profile"], "static"),
		Task:            nullable(raw["task"]),
		Type:            nullable(firstNonNil(raw["type"], raw["review_type"])),
		CommitHash:      nullable(raw["commit_hash"]),
		Checks:          interfaceSlice(raw["checks"]),
		Gaps:            gaps,
		Followups:       append(interfaceSlice(raw["followups"]), interfaceSlice(raw["recommendations"])...),
		DualReview:      nullable(raw["dual_review"]),
	}
}

func normalizeReviewVerdict(v string) string {
	switch v {
	case "approve":
		return "APPROVE"
	case "needs-attention":
		return "REQUEST_CHANGES"
	case "APPROVE", "REQUEST_CHANGES":
		return v
	default:
		return "REQUEST_CHANGES"
	}
}

func normalizeReviewGaps(v interface{}) []reviewGap {
	items := interfaceSlice(v)
	gaps := make([]reviewGap, 0, len(items))
	for _, item := range items {
		switch x := item.(type) {
		case string:
			gaps = append(gaps, reviewGap{Severity: "major"})
		case map[string]interface{}:
			gaps = append(gaps, reviewGap{Severity: stringValue(x["severity"], "major")})
		}
	}
	return gaps
}

func appendSubagentStopPersistAudit(stateDir, sessionID, verdict string) error {
	rec := map[string]interface{}{
		"ts":          time.Now().UTC().Format(time.RFC3339),
		"session_id":  sessionID,
		"verdict":     verdict,
		"commit_hash": nil,
		"source":      "subagentstop-reviewer-persist",
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(stateDir, "subagentstop-persist-audit.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

func stringValue(v interface{}, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func nullable(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	return v
}

func firstNonNil(values ...interface{}) interface{} {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func interfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	if s, ok := v.([]interface{}); ok {
		return s
	}
	return []interface{}{v}
}
