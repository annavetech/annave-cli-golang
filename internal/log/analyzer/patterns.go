// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"sort"
	"time"

	"annave.tech/cli/internal/log/domain"
)

// messageCount accumulates occurrence statistics for a single message template.
type messageCount struct {
	message   string
	count     int
	firstLine int
	lastLine  int
	firstSeen time.Time
	lastSeen  time.Time
	samples   []string
}

// detectPatterns finds error/warn messages that repeat more than minCount times.
func detectPatterns(entries []domain.LogEntry, minCount int) []domain.Anomaly {
	counts := make(map[string]*messageCount)

	for _, e := range entries {
		if e.Level != "error" && e.Level != "critical" && e.Level != "warn" {
			continue
		}
		key := e.Level + "|" + e.Message
		mc, ok := counts[key]
		if !ok {
			mc = &messageCount{
				message:   e.Message,
				firstLine: e.Line,
				firstSeen: e.Timestamp,
			}
			counts[key] = mc
		}
		mc.count++
		mc.lastLine = e.Line
		if !e.Timestamp.IsZero() {
			mc.lastSeen = e.Timestamp
		}
		if len(mc.samples) < 3 {
			mc.samples = append(mc.samples, e.Raw)
		}
	}

	var anomalies []domain.Anomaly
	for _, mc := range counts {
		if mc.count < minCount {
			continue
		}
		anomalies = append(anomalies, domain.Anomaly{
			Kind:        domain.AnomalyPattern,
			Message:     mc.message,
			Count:       mc.count,
			FirstLine:   mc.firstLine,
			LastLine:    mc.lastLine,
			FirstSeen:   mc.firstSeen,
			LastSeen:    mc.lastSeen,
			SampleLines: mc.samples,
		})
	}

	sort.Slice(anomalies, func(i, j int) bool {
		return anomalies[i].Count > anomalies[j].Count
	})
	return anomalies
}
