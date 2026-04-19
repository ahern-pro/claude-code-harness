package memory

import (
	"regexp"
	"sort"
	"strings"
)

const (
	defaultMaxResults = 5
	scanMaxFiles      = 100
)

var (
	asciiTokenPattern = regexp.MustCompile(`[A-Za-z0-9_]+`)
	hanCharPattern    = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}]`)
)

func FindRelevantMemories(query, cwd string, maxResults int) ([]*MemoryHeader, error) {
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}

	tokens := tokenize(query)
	if len(tokens) == 0 {
		return []*MemoryHeader{}, nil
	}

	headers, err := ScanMemoryFiles(cwd, scanMaxFiles)
	if err != nil {
		return nil, err
	}

	type scored struct {
		score  float64
		header *MemoryHeader
	}
	var ranked []scored
	for _, header := range headers {
		meta := strings.ToLower(header.Title + " " + header.Description)
		body := strings.ToLower(header.BodyPreview)

		// Metadata matches are weighted 2x; body matches 1x.
		var metaHits, bodyHits int
		for t := range tokens {
			if strings.Contains(meta, t) {
				metaHits++
			}
			if strings.Contains(body, t) {
				bodyHits++
			}
		}
		score := float64(metaHits)*2.0 + float64(bodyHits)
		if score > 0 {
			ranked = append(ranked, scored{score: score, header: header})
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].header.ModifiedAt > ranked[j].header.ModifiedAt
	})

	if len(ranked) > maxResults {
		ranked = ranked[:maxResults]
	}
	out := make([]*MemoryHeader, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.header)
	}
	return out, nil
}

func tokenize(text string) map[string]struct{} {
	tokens := make(map[string]struct{})
	lower := strings.ToLower(text)
	for _, t := range asciiTokenPattern.FindAllString(lower, -1) {
		if len(t) >= 3 {
			tokens[t] = struct{}{}
		}
	}
	for _, ch := range hanCharPattern.FindAllString(text, -1) {
		tokens[ch] = struct{}{}
	}
	return tokens
}
