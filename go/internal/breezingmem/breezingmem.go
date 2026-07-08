package breezingmem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	platformHarness = "harness"
	briefFilePath   = "breezing/brief-card.v1.json"
	briefKind       = "decisions_md"
)

// Event type constants for breezing mem write layer v0.
const (
	EventRunStarted      = "breezing_run_started"
	EventBriefConfirmed  = "breezing_brief_confirmed"
	EventWorkerResult    = "breezing_worker_result"
	EventAggregationDone = "breezing_aggregation_completed"
)

// Client posts breezing lifecycle events to harness-mem (fail-open).
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Logger     io.Writer
	homeDir    func() (string, error)
}

type recordRequest struct {
	Event recordEvent `json:"event"`
}

type recordEvent struct {
	Platform  string `json:"platform"`
	Project   string `json:"project"`
	SessionID string `json:"session_id"`
	EventType string `json:"event_type"`
	Content   string `json:"content"`
}

type ingestRequest struct {
	FilePath  string `json:"file_path"`
	Content   string `json:"content"`
	Kind      string `json:"kind,omitempty"`
	Project   string `json:"project"`
	Platform  string `json:"platform"`
	SessionID string `json:"session_id"`
}

// New returns a Client with env-derived BaseURL and a 1s HTTP timeout.
func New() *Client {
	host := os.Getenv("HARNESS_MEM_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("HARNESS_MEM_PORT")
	if port == "" {
		port = "37888"
	}
	return &Client{
		BaseURL:    "http://" + host + ":" + port,
		HTTPClient: &http.Client{Timeout: time.Second},
		Logger:     os.Stderr,
		homeDir:    os.UserHomeDir,
	}
}

// RecordEvent POSTs one event to /v1/events/record. Fail-open: not-configured is
// silent; unreachable logs one warning line and returns.
func (c *Client) RecordEvent(ctx context.Context, eventType, project, sessionID, content string) {
	if !c.configured() {
		return
	}
	body, err := json.Marshal(recordRequest{
		Event: recordEvent{
			Platform:  platformHarness,
			Project:   project,
			SessionID: sessionID,
			EventType: eventType,
			Content:   content,
		},
	})
	if err != nil {
		return
	}
	if err := c.post(ctx, "/v1/events/record", body); err != nil {
		fmt.Fprintf(c.logger(), "breezing-mem: record skipped (unreachable)\n")
	}
}

// IngestBrief POSTs confirmed brief JSON to /v1/ingest/knowledge-file with the
// same 3-state fail-open behavior as RecordEvent.
func (c *Client) IngestBrief(ctx context.Context, project, sessionID string, briefJSON []byte) {
	if !c.configured() {
		return
	}
	body, err := json.Marshal(ingestRequest{
		FilePath:  briefFilePath,
		Content:   string(briefJSON),
		Kind:      briefKind,
		Project:   project,
		Platform:  platformHarness,
		SessionID: sessionID,
	})
	if err != nil {
		return
	}
	if err := c.post(ctx, "/v1/ingest/knowledge-file", body); err != nil {
		fmt.Fprintf(c.logger(), "breezing-mem: ingest skipped (unreachable)\n")
	}
}

func (c *Client) configured() bool {
	homeDir := c.homeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}
	home, err := homeDir()
	if err != nil {
		return false
	}
	harnessMemHome := os.Getenv("HARNESS_MEM_HOME")
	if harnessMemHome == "" {
		harnessMemHome = filepath.Join(home, ".harness-mem")
	}
	claudeMem := filepath.Join(home, ".claude-mem")
	if _, err := os.Stat(harnessMemHome); os.IsNotExist(err) {
		if _, legacyErr := os.Stat(claudeMem); os.IsNotExist(legacyErr) {
			return false
		}
	}
	return true
}

func (c *Client) logger() io.Writer {
	if c.Logger != nil {
		return c.Logger
	}
	return io.Discard
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: time.Second}
}

func (c *Client) post(ctx context.Context, path string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := os.Getenv("HARNESS_MEM_ADMIN_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
}
