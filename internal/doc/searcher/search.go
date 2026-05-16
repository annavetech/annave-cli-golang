// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package searcher

import (
	"bufio"
	"context"
	"os"
	"sort"
	"strings"

	"annave.tech/cli/internal/doc/domain"
	"annave.tech/cli/internal/doc/port"
	"annave.tech/cli/internal/shared/config"
)

func search(ctx context.Context, idx *domain.Index, query domain.SearchQuery, opts port.SearchOptions) []domain.SearchResult {
	if len(query.Terms) == 0 || len(idx.Files) == 0 {
		return nil
	}

	maxResults := opts.MaxResults
	if maxResults == 0 {
		maxResults = config.App.Limits.Doc.MaxResults
	}

	scores := make(map[int]float64) // fileIndex → cumulative score
	bestLine := make(map[int]int)   // fileIndex → line of first match

	for _, term := range query.Terms {
		lower := strings.ToLower(term)

		// Exact match
		if postings, ok := idx.Terms[lower]; ok {
			for _, p := range postings {
				scores[p.FileIndex] += 1.0
				if _, seen := bestLine[p.FileIndex]; !seen {
					bestLine[p.FileIndex] = p.Line
				}
			}
			continue
		}

		// Prefix match (partial credit)
		for indexTerm, postings := range idx.Terms {
			if strings.HasPrefix(indexTerm, lower) {
				for _, p := range postings {
					scores[p.FileIndex] += 0.5
					if _, seen := bestLine[p.FileIndex]; !seen {
						bestLine[p.FileIndex] = p.Line
					}
				}
			}
		}
	}

	type candidate struct {
		fileIndex int
		score     float64
		line      int
	}
	candidates := make([]candidate, 0, len(scores))
	for fileIdx, score := range scores {
		candidates = append(candidates, candidate{fileIdx, score, bestLine[fileIdx]})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		// Tiebreak: prefer more recently modified files
		return idx.Files[candidates[i].fileIndex].ModTime.After(idx.Files[candidates[j].fileIndex].ModTime)
	})

	if len(candidates) > maxResults {
		candidates = candidates[:maxResults]
	}

	results := make([]domain.SearchResult, 0, len(candidates))
	for _, c := range candidates {
		if ctx.Err() != nil {
			break
		}
		file := idx.Files[c.fileIndex]
		results = append(results, domain.SearchResult{
			File:    file,
			Score:   c.score,
			Excerpt: extractExcerpt(file.Path, c.line, query.Terms),
			Line:    c.line,
		})
	}
	return results
}

// extractExcerpt reads the matching line from disk and returns a trimmed snippet.
func extractExcerpt(path string, targetLine int, terms []string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum == targetLine {
			line := strings.TrimSpace(scanner.Text())
			if len(line) > 200 {
				line = line[:197] + "..."
			}
			return line
		}
	}
	return ""
}
