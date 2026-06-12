package breezingmem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const searchMaxResults = 3

type searchRequest struct {
	Query   string `json:"query"`
	Project string `json:"project"`
	K       int    `json:"k"`
}

type searchResponse struct {
	Results []searchResultItem `json:"results"`
}

type searchResultItem struct {
	Observation searchObservation `json:"observation"`
	Score       float64           `json:"score"`
}

type searchObservation struct {
	ID        string `json:"id"`
	Summary   string `json:"summary"`
	Decision  string `json:"decision"`
	Outcome   string `json:"outcome"`
	DecidedAt string `json:"decided_at"`
}

// SimilarPastDecision is one harness-mem search hit with similarity score.
type SimilarPastDecision struct {
	Summary   string  `json:"summary"`
	Decision  string  `json:"decision"`
	Outcome   string  `json:"outcome"`
	DecidedAt string  `json:"decided_at"`
	MemID     string  `json:"mem_id"`
	Score     float64 `json:"score"`
}

// SearchSimilar queries harness-mem for similar past decisions (fail-open).
func (c *Client) SearchSimilar(ctx context.Context, project, query string) []SimilarPastDecision {
	if !c.configured() {
		return []SimilarPastDecision{}
	}
	body, err := json.Marshal(searchRequest{
		Query:   query,
		Project: project,
		K:       searchMaxResults,
	})
	if err != nil {
		return []SimilarPastDecision{}
	}

	var resp searchResponse
	if err := c.postJSON(ctx, "/v1/search", body, &resp); err != nil {
		fmt.Fprintf(c.logger(), "breezing-mem: search skipped (unreachable)\n")
		return []SimilarPastDecision{}
	}

	out := make([]SimilarPastDecision, 0, searchMaxResults)
	for i, item := range resp.Results {
		if i >= searchMaxResults {
			break
		}
		obs := item.Observation
		out = append(out, SimilarPastDecision{
			Summary:   obs.Summary,
			Decision:  obs.Decision,
			Outcome:   obs.Outcome,
			DecidedAt: obs.DecidedAt,
			MemID:     obs.ID,
			Score:     item.Score,
		})
	}
	return out
}

func (c *Client) postJSON(ctx context.Context, path string, body []byte, dest interface{}) error {
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
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(raw, dest)
}
