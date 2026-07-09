package judgmentledger

import (
	"fmt"
	"strings"
)

const defaultRecallLimit = 3

// PastDecision mirrors judgment-card.v1 similar_past_decisions items.
type PastDecision struct {
	Summary   string `json:"summary"`
	Decision  string `json:"decision"`
	Outcome   string `json:"outcome"`
	DecidedAt string `json:"decided_at"`
	MemID     string `json:"mem_id"`
}

// RecallSimilar returns up to limit past decisions ranked by string-match against question.
func RecallSimilar(path, project, question string, limit int) ([]PastDecision, error) {
	if limit <= 0 {
		limit = defaultRecallLimit
	}
	matches, err := Search(path, project, question, limit)
	if err != nil {
		return nil, err
	}
	out := make([]PastDecision, 0, len(matches))
	for _, rec := range matches {
		out = append(out, recordToPastDecision(rec))
	}
	return out, nil
}

func recordToPastDecision(rec Record) PastDecision {
	summary := rec.Question
	if len(summary) > 120 {
		summary = summary[:117] + "..."
	}
	outcome := rec.Rationale
	if outcome == "" {
		outcome = "recorded"
	}
	return PastDecision{
		Summary:   summary,
		Decision:  rec.Answer,
		Outcome:   outcome,
		DecidedAt: rec.DecidedAt,
		MemID:     fmt.Sprintf("judgment-ledger:%s", rec.ID),
	}
}

// SummarizeQuestion returns a trimmed question for search/recall queries.
func SummarizeQuestion(question string) string {
	q := strings.TrimSpace(question)
	if len(q) > 200 {
		return q[:200]
	}
	return q
}
