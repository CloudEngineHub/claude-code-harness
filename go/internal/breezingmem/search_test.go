package breezingmem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func searchResultPayload(id, summary, decision, outcome, decidedAt string, score float64) map[string]interface{} {
	return map[string]interface{}{
		"observation": map[string]interface{}{
			"id":         id,
			"summary":    summary,
			"decision":   decision,
			"outcome":    outcome,
			"decided_at": decidedAt,
		},
		"score": score,
	}
}

func newSearchServer(t *testing.T, results []map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	}))
}

func TestSearchSimilar_Healthy(t *testing.T) {
	results := []map[string]interface{}{
		searchResultPayload("mem-1", "Rate limit", "cursor", "shipped", "2026-06-01T00:00:00Z", 0.95),
		searchResultPayload("mem-2", "Auth flow", "codex", "shipped", "2026-06-02T00:00:00Z", 0.88),
		searchResultPayload("mem-3", "Dashboard", "claude", "wait", "2026-06-03T00:00:00Z", 0.75),
	}
	server := newSearchServer(t, results)
	defer server.Close()

	home := configuredHome(t)
	client := newTestClient(t, home, server, io.Discard)

	got := client.SearchSimilar(context.Background(), "/repo/proj", "implement rate limit")
	if len(got) != 3 {
		t.Fatalf("got %d results, want 3", len(got))
	}
	if got[0].MemID != "mem-1" || got[0].Score != 0.95 {
		t.Fatalf("first result = %+v, want mem_id=mem-1 score=0.95", got[0])
	}
	if got[0].Summary != "Rate limit" || got[0].Decision != "cursor" {
		t.Fatalf("first result fields = %+v", got[0])
	}
}

func TestSearchSimilar_NotConfigured(t *testing.T) {
	home := t.TempDir()
	var logBuf bytes.Buffer
	client := &Client{
		BaseURL:    "http://127.0.0.1:1",
		HTTPClient: &http.Client{Timeout: 0},
		Logger:     &logBuf,
		homeDir:    func() (string, error) { return home, nil },
	}

	got := client.SearchSimilar(context.Background(), "proj", "query")
	if len(got) != 0 {
		t.Fatalf("got %d results, want 0", len(got))
	}
	if logBuf.Len() != 0 {
		t.Fatalf("not-configured must emit 0 warning lines, got %q", logBuf.String())
	}
}

func TestSearchSimilar_Unreachable(t *testing.T) {
	home := configuredHome(t)
	var logBuf bytes.Buffer
	client := &Client{
		BaseURL:    "http://127.0.0.1:1",
		HTTPClient: &http.Client{Timeout: 0},
		Logger:     &logBuf,
		homeDir:    func() (string, error) { return home, nil },
	}

	got := client.SearchSimilar(context.Background(), "proj", "query")
	if len(got) != 0 {
		t.Fatalf("unreachable must return empty slice, got %d", len(got))
	}
	lines := strings.Split(strings.TrimSpace(logBuf.String()), "\n")
	if len(lines) != 1 || lines[0] == "" {
		t.Fatalf("unreachable must emit exactly 1 warning line, got %q", logBuf.String())
	}
	if !strings.Contains(lines[0], "breezing-mem: search skipped (unreachable)") {
		t.Fatalf("warning = %q, want breezing-mem: search skipped (unreachable)", lines[0])
	}
}

func TestSearchSimilar_MoreThan3_Capped(t *testing.T) {
	results := make([]map[string]interface{}, 0, 5)
	for i := 1; i <= 5; i++ {
		results = append(results, searchResultPayload(
			fmt.Sprintf("mem-%d", i),
			"summary",
			"cursor",
			"outcome",
			"2026-06-01T00:00:00Z",
			float64(i)/10,
		))
	}
	server := newSearchServer(t, results)
	defer server.Close()

	home := configuredHome(t)
	client := newTestClient(t, home, server, io.Discard)
	got := client.SearchSimilar(context.Background(), "proj", "query")
	if len(got) != 3 {
		t.Fatalf("got %d results, want cap at 3", len(got))
	}
}

func TestSearchSimilar_Zero_OK(t *testing.T) {
	server := newSearchServer(t, nil)
	defer server.Close()

	home := configuredHome(t)
	client := newTestClient(t, home, server, io.Discard)
	got := client.SearchSimilar(context.Background(), "proj", "query")
	if len(got) != 0 {
		t.Fatalf("got %d results, want 0", len(got))
	}
}

func TestSearchSimilar_NoWorkgraphCall_GrepAudit(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(data))
		for _, needle := range []string{"workgraph", "signal_send"} {
			if strings.Contains(lower, needle) {
				t.Fatalf("%s must not reference %q", entry.Name(), needle)
			}
		}
	}
}
