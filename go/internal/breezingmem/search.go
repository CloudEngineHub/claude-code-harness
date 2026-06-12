package breezingmem

import "context"

// SimilarPastDecision is one harness-mem search hit with similarity score.
type SimilarPastDecision struct {
	Summary   string  `json:"summary"`
	Decision  string  `json:"decision"`
	Outcome   string  `json:"outcome"`
	DecidedAt string  `json:"decided_at"`
	MemID     string  `json:"mem_id"`
	Score     float64 `json:"score"`
}

// SearchSimilar is a RED-phase stub; implementation follows in GREEN.
func (c *Client) SearchSimilar(ctx context.Context, project, query string) []SimilarPastDecision {
	_ = c
	_ = ctx
	_ = project
	_ = query
	return nil
}
