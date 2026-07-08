package breezingmem

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const testPlatform = "harness"

type recordedEvent struct {
	EventType string
	Project   string
	SessionID string
	Content   string
	Platform  string
}

func configuredHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".harness-mem"), 0o700); err != nil {
		t.Fatal(err)
	}
	return home
}

func newTestClient(t *testing.T, home string, server *httptest.Server, logger io.Writer) *Client {
	t.Helper()
	return &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Logger:     logger,
		homeDir:    func() (string, error) { return home, nil },
	}
}

func decodeRecordBody(t *testing.T, body []byte) recordedEvent {
	t.Helper()
	var req struct {
		Event struct {
			Platform  string `json:"platform"`
			Project   string `json:"project"`
			SessionID string `json:"session_id"`
			EventType string `json:"event_type"`
			Content   string `json:"content"`
		} `json:"event"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("invalid record body: %v\n%s", err, body)
	}
	return recordedEvent{
		Platform:  req.Event.Platform,
		Project:   req.Event.Project,
		SessionID: req.Event.SessionID,
		EventType: req.Event.EventType,
		Content:   req.Event.Content,
	}
}

func TestBreezingMem_Healthy(t *testing.T) {
	var mu sync.Mutex
	var bodies [][]byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/events/record" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	home := configuredHome(t)
	client := newTestClient(t, home, server, io.Discard)

	ctx := context.Background()
	project := "/tmp/test-project"
	sessionID := "sess-healthy"
	cases := []struct {
		eventType string
		content   string
	}{
		{EventRunStarted, `{"backend":"codex"}`},
		{EventBriefConfirmed, `{"brief":"ok"}`},
		{EventWorkerResult, `{"task_id":"t1","success":true,"exit_code":0}`},
		{EventAggregationDone, `{"task_count":1}`},
	}

	for _, tc := range cases {
		client.RecordEvent(ctx, tc.eventType, project, sessionID, tc.content)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != len(cases) {
		t.Fatalf("got %d POST bodies, want %d", len(bodies), len(cases))
	}

	for i, tc := range cases {
		got := decodeRecordBody(t, bodies[i])
		if got.Platform != testPlatform {
			t.Errorf("event[%d] platform = %q, want %q", i, got.Platform, testPlatform)
		}
		if got.Project != project {
			t.Errorf("event[%d] project = %q, want %q", i, got.Project, project)
		}
		if got.SessionID != sessionID {
			t.Errorf("event[%d] session_id = %q, want %q", i, got.SessionID, sessionID)
		}
		if got.EventType != tc.eventType {
			t.Errorf("event[%d] event_type = %q, want %q", i, got.EventType, tc.eventType)
		}
		if got.Content != tc.content {
			t.Errorf("event[%d] content = %q, want %q", i, got.Content, tc.content)
		}
	}
}

func TestBreezingMem_NotConfigured(t *testing.T) {
	home := t.TempDir() // no ~/.harness-mem or ~/.claude-mem
	var logBuf bytes.Buffer

	client := &Client{
		BaseURL:    "http://127.0.0.1:1",
		HTTPClient: &http.Client{Timeout: 0},
		Logger:     &logBuf,
		homeDir:    func() (string, error) { return home, nil },
	}

	ctx := context.Background()
	client.RecordEvent(ctx, EventRunStarted, "proj", "sess", `{}`)
	client.IngestBrief(ctx, "proj", "sess", []byte(`{"brief":true}`))

	if logBuf.Len() != 0 {
		t.Fatalf("not-configured must emit 0 warning lines, got %q", logBuf.String())
	}
}

func TestBreezingMem_Unreachable(t *testing.T) {
	home := configuredHome(t)
	var logBuf bytes.Buffer

	client := &Client{
		BaseURL:    "http://127.0.0.1:1",
		HTTPClient: &http.Client{Timeout: 0},
		Logger:     &logBuf,
		homeDir:    func() (string, error) { return home, nil },
	}

	ctx := context.Background()
	client.RecordEvent(ctx, EventWorkerResult, "proj", "sess", `{}`)

	lines := strings.Split(strings.TrimSpace(logBuf.String()), "\n")
	if len(lines) != 1 || lines[0] == "" {
		t.Fatalf("unreachable must emit exactly 1 warning line, got %q", logBuf.String())
	}
	if !strings.Contains(lines[0], "breezing-mem: record skipped (unreachable)") {
		t.Fatalf("warning = %q, want breezing-mem: record skipped (unreachable)", lines[0])
	}
}

func TestBreezingMem_IngestBrief(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ingest/knowledge-file" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotPath = r.URL.Path
		gotBody = body
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	home := configuredHome(t)
	client := newTestClient(t, home, server, io.Discard)

	brief := []byte(`{"schema":"brief-card.v1","question":"ship?"}`)
	client.IngestBrief(context.Background(), "/repo/proj", "sess-ingest", brief)

	mu.Lock()
	defer mu.Unlock()
	if gotPath != "/v1/ingest/knowledge-file" {
		t.Fatalf("path = %q, want /v1/ingest/knowledge-file", gotPath)
	}

	var payload struct {
		FilePath  string `json:"file_path"`
		Content   string `json:"content"`
		Kind      string `json:"kind"`
		Project   string `json:"project"`
		Platform  string `json:"platform"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("invalid ingest body: %v\n%s", err, gotBody)
	}
	if payload.FilePath == "" {
		t.Fatal("file_path must be set")
	}
	if payload.Content != string(brief) {
		t.Fatalf("content = %q, want %q", payload.Content, string(brief))
	}
	if payload.Project != "/repo/proj" {
		t.Fatalf("project = %q, want /repo/proj", payload.Project)
	}
	if payload.SessionID != "sess-ingest" {
		t.Fatalf("session_id = %q, want sess-ingest", payload.SessionID)
	}
	if payload.Platform != testPlatform {
		t.Fatalf("platform = %q, want %q", payload.Platform, testPlatform)
	}
}

func TestBreezingMem_NoSignalAPIReferences(t *testing.T) {
	data, err := os.ReadFile("breezingmem.go")
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(data))
	for _, needle := range []string{"/v1/signals", "signal_send", "signal_read", "signal_ack"} {
		if strings.Contains(lower, needle) {
			t.Fatalf("breezingmem.go must not reference signal API %q", needle)
		}
	}
}
