package judgmentledger

import (
	"sort"
	"strings"
)

const defaultSearchLimit = 3

// Search finds up to limit records for project using string-match ranking.
// Ranking: count of case-insensitive substring hits across question, answer,
// rationale, and tags (Lead decision: string-match, max 3).
func Search(path, project, query string, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	records, err := LoadByProject(path, project)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	query = strings.TrimSpace(strings.ToLower(query))
	type scored struct {
		rec   Record
		score int
	}
	var ranked []scored
	for _, rec := range records {
		score := matchScore(rec, query)
		if query == "" || score > 0 {
			ranked = append(ranked, scored{rec: rec, score: score})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].rec.DecidedAt > ranked[j].rec.DecidedAt
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]Record, len(ranked))
	for i, s := range ranked {
		out[i] = s.rec
	}
	return out, nil
}

func matchScore(rec Record, query string) int {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return 1
	}
	if strings.Contains(strings.ToLower(rec.Question), query) ||
		strings.Contains(strings.ToLower(rec.Answer), query) ||
		strings.Contains(strings.ToLower(rec.Rationale), query) {
		return 10
	}
	score := 0
	for _, token := range queryTokens(query) {
		for _, field := range []string{rec.Question, rec.Answer, rec.Rationale} {
			if strings.Contains(strings.ToLower(field), token) {
				score++
				break
			}
		}
		for _, tag := range rec.Tags {
			if strings.Contains(strings.ToLower(tag), token) {
				score++
				break
			}
		}
	}
	return score
}

func queryTokens(query string) []string {
	var tokens []string
	for _, part := range strings.Fields(query) {
		part = strings.ToLower(strings.TrimSpace(part))
		if len([]rune(part)) >= 2 {
			tokens = append(tokens, part)
		}
	}
	return tokens
}
