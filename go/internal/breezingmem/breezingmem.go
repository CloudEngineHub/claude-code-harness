package breezingmem

import (
	"context"
	"net/http"
	"os"
)

// Client posts breezing lifecycle events to harness-mem (fail-open).
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Logger     interface{ Write([]byte) (int, error) }
	homeDir    func() (string, error)
}

// Event type constants for breezing mem write layer v0.
const (
	EventRunStarted      = "breezing_run_started"
	EventBriefConfirmed  = "breezing_brief_confirmed"
	EventWorkerResult    = "breezing_worker_result"
	EventAggregationDone = "breezing_aggregation_completed"
)

// New returns a Client with env-derived BaseURL. RED stub: no HTTP yet.
func New() *Client {
	return &Client{
		homeDir: os.UserHomeDir,
	}
}

// RecordEvent is a RED stub — tests expect real POST behavior.
func (c *Client) RecordEvent(_ context.Context, _, _, _, _ string) {}

// IngestBrief is a RED stub — tests expect real POST behavior.
func (c *Client) IngestBrief(_ context.Context, _, _ string, _ []byte) {}
