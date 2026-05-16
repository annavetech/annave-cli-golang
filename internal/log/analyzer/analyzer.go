// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"io"
	"time"

	"annave.tech/cli/internal/log/domain"
	"annave.tech/cli/internal/log/port"
	"annave.tech/cli/internal/shared/config"
)

// LogAnalyzer implements port.Analyzer.
type LogAnalyzer struct{}

// New returns a ready-to-use LogAnalyzer.
func New() *LogAnalyzer { return &LogAnalyzer{} }

func (a *LogAnalyzer) Analyze(r io.Reader, filename string, opts port.AnalyzeOptions) (*domain.AnalysisReport, error) {
	maxLines := opts.MaxLines
	if maxLines == 0 {
		maxLines = config.App.Limits.Log.MaxLines
	}

	entries, format, totalLines, parseErrors := parseLines(r, maxLines)
	filtered := filterEntries(entries, opts.Since, opts.MinLevel)

	const minRepeat = 3
	patterns := detectPatterns(filtered, minRepeat)
	spikes := detectSpikes(filtered)
	clusters := detectClusters(filtered, minRepeat)

	all := mergeAndSort(patterns, spikes, clusters)
	findings := rankFindings(all, len(filtered))

	var tr domain.TimeRange
	for _, e := range filtered {
		if e.Timestamp.IsZero() {
			continue
		}
		if tr.From.IsZero() || e.Timestamp.Before(tr.From) {
			tr.From = e.Timestamp
		}
		if e.Timestamp.After(tr.To) {
			tr.To = e.Timestamp
		}
	}

	return &domain.AnalysisReport{
		File:        filename,
		Format:      format,
		TotalLines:  totalLines,
		ParsedLines: len(entries),
		TimeRange:   tr,
		Findings:    findings,
		ParseErrors: parseErrors,
	}, nil
}

// filterEntries removes entries that pre-date since or fall below minLevel.
func filterEntries(entries []domain.LogEntry, since time.Time, minLevel string) []domain.LogEntry {
	if since.IsZero() && minLevel == "" {
		return entries
	}
	minRank := levelRank(minLevel)
	result := make([]domain.LogEntry, 0, len(entries))
	for _, e := range entries {
		if !since.IsZero() && !e.Timestamp.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		if minLevel != "" && levelRank(e.Level) < minRank {
			continue
		}
		result = append(result, e)
	}
	return result
}

// levelRank maps a level string to a numeric rank for comparison.
func levelRank(level string) int {
	switch level {
	case "debug", "trace":
		return 0
	case "info", "":
		return 1
	case "warn":
		return 2
	case "error":
		return 3
	case "critical", "fatal":
		return 4
	}
	return 0
}
