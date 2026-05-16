// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"regexp"
	"sort"
	"time"

	"annave.tech/cli/internal/log/domain"
)

var (
	uuidNorm   = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	hexNorm    = regexp.MustCompile(`\b[0-9a-fA-F]{16,}\b`)
	ipNorm     = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	numberNorm = regexp.MustCompile(`\b\d+\b`)
	quotedNorm = regexp.MustCompile(`"[^"]{8,}"`)
)

// normalizeMessage replaces variable parts (UUIDs, IPs, numbers) with placeholders to group similar messages.
func normalizeMessage(msg string) string {
	msg = uuidNorm.ReplaceAllString(msg, "<uuid>")
	msg = hexNorm.ReplaceAllString(msg, "<hex>")
	msg = ipNorm.ReplaceAllString(msg, "<ip>")
	msg = quotedNorm.ReplaceAllString(msg, "<value>")
	msg = numberNorm.ReplaceAllString(msg, "<N>")
	return msg
}

// cluster accumulates statistics for a group of messages sharing one normalised template.
type cluster struct {
	template  string
	count     int
	firstLine int
	lastLine  int
	firstSeen time.Time
	lastSeen  time.Time
	samples   []string
}

// detectClusters groups messages by normalised template and flags recurring patterns.
func detectClusters(entries []domain.LogEntry, minCount int) []domain.Anomaly {
	clusters := make(map[string]*cluster)

	for _, e := range entries {
		if e.Level != "error" && e.Level != "critical" && e.Level != "warn" {
			continue
		}
		normalized := normalizeMessage(e.Message)
		if normalized == e.Message {
			continue // no variable parts; identical messages already handled by patterns
		}
		key := e.Level + "|" + normalized
		c, ok := clusters[key]
		if !ok {
			c = &cluster{
				template:  normalized,
				firstLine: e.Line,
				firstSeen: e.Timestamp,
			}
			clusters[key] = c
		}
		c.count++
		c.lastLine = e.Line
		if !e.Timestamp.IsZero() {
			c.lastSeen = e.Timestamp
		}
		if len(c.samples) < 3 {
			c.samples = append(c.samples, e.Message)
		}
	}

	var anomalies []domain.Anomaly
	for _, c := range clusters {
		if c.count < minCount {
			continue
		}
		anomalies = append(anomalies, domain.Anomaly{
			Kind:        domain.AnomalyCluster,
			Message:     c.template,
			Count:       c.count,
			FirstLine:   c.firstLine,
			LastLine:    c.lastLine,
			FirstSeen:   c.firstSeen,
			LastSeen:    c.lastSeen,
			SampleLines: c.samples,
		})
	}

	sort.Slice(anomalies, func(i, j int) bool {
		return anomalies[i].Count > anomalies[j].Count
	})
	return anomalies
}
