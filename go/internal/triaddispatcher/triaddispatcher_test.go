package triaddispatcher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezingmem"
)

func configuredHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	return home
}

func newMemClient(t *testing.T, memHome string, server *httptest.Server) *breezingmem.Client {
	t.Helper()
	t.Setenv("HARNESS_MEM_HOME", memHome)
	mem := breezingmem.New()
	mem.BaseURL = server.URL
	mem.HTTPClient = server.Client()
	mem.Logger = io.Discard
	return mem
}

func searchServer(t *testing.T, decisions ...string) *httptest.Server {
	t.Helper()
	results := make([]map[string]interface{}, 0, len(decisions))
	for i, decision := range decisions {
		results = append(results, map[string]interface{}{
			"observation": map[string]interface{}{
				"id":         "mem-" + decision,
				"summary":    "past dispatch",
				"decision":   decision,
				"outcome":    "shipped",
				"decided_at": "2026-06-01T00:00:00Z",
			},
			"score": 0.9 - float64(i)*0.05,
		})
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	}))
}

func TestDispatchFor_DefaultResolverBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	home := configuredHome(t)
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	mem := newMemClient(t, home, server)

	got := DispatchFor(context.Background(), mem, BackendCodex, "/repo", "add login form")
	if got.Backend != BackendCodex {
		t.Fatalf("backend = %q, want codex", got.Backend)
	}
	if len(got.SimilarPastResults) != 0 {
		t.Fatalf("expected empty similar results, got %d", len(got.SimilarPastResults))
	}
}

func TestDispatchFor_PastDecisionMajorityNoted(t *testing.T) {
	server := searchServer(t, "cursor", "cursor", "claude")
	defer server.Close()

	home := configuredHome(t)
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	mem := newMemClient(t, home, server)

	got := DispatchFor(context.Background(), mem, BackendClaude, "/repo", "UI polish")
	if got.Backend != BackendClaude {
		t.Fatalf("backend = %q, want resolver default claude", got.Backend)
	}
	if !strings.Contains(got.Reason, "cursor") {
		t.Fatalf("reason should note majority backend cursor, got %q", got.Reason)
	}
	if len(got.SimilarPastResults) != 3 {
		t.Fatalf("similar results = %d, want 3", len(got.SimilarPastResults))
	}
}

func TestDispatchFor_MemUnreachable_StillReturnsBackend(t *testing.T) {
	memHome := configuredHome(t)
	if err := os.MkdirAll(memHome, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HARNESS_MEM_HOME", memHome)
	mem := breezingmem.New()
	mem.BaseURL = "http://127.0.0.1:1"
	mem.HTTPClient = &http.Client{Timeout: 0}
	mem.Logger = io.Discard

	got := DispatchFor(context.Background(), mem, BackendCursor, "/repo", "refactor hooks")
	if got.Backend != BackendCursor {
		t.Fatalf("backend = %q, want cursor even when mem unreachable", got.Backend)
	}
}
