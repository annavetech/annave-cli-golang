// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"annave.tech/cli/internal/log/domain"
)

// mergeAndSort merges all anomaly lists with spikes first, then by count descending.
func mergeAndSort(patterns, spikes, clusters []domain.Anomaly) []domain.Anomaly {
	all := make([]domain.Anomaly, 0, len(patterns)+len(spikes)+len(clusters))
	all = append(all, spikes...)
	all = append(all, patterns...)
	all = append(all, clusters...)

	sort.SliceStable(all, func(i, j int) bool {
		iSpike := all[i].Kind == domain.AnomalySpike
		jSpike := all[j].Kind == domain.AnomalySpike
		if iSpike != jSpike {
			return iSpike
		}
		return all[i].Count > all[j].Count
	})
	return all
}

// rankFindings converts anomalies into user-facing findings with severity and summary.
func rankFindings(anomalies []domain.Anomaly, totalEntries int) []domain.Finding {
	var findings []domain.Finding
	for rank, a := range anomalies {
		findings = append(findings, domain.Finding{
			Rank:     rank + 1,
			Severity: severityFor(a, totalEntries),
			Summary:  summaryFor(a),
			Detail:   detailFor(a),
			Count:    a.Count,
			Anomaly:  a,
		})
	}
	return findings
}

// severityFor assigns severity based on anomaly kind and its share of total entries.
func severityFor(a domain.Anomaly, total int) domain.Severity {
	pct := 0.0
	if total > 0 {
		pct = float64(a.Count) / float64(total) * 100
	}
	switch a.Kind {
	case domain.AnomalySpike:
		if a.Count >= 50 {
			return domain.SeverityCritical
		}
		return domain.SeverityHigh
	default:
		switch {
		case pct >= 10 || a.Count >= 1000:
			return domain.SeverityCritical
		case pct >= 5 || a.Count >= 100:
			return domain.SeverityHigh
		case pct >= 1 || a.Count >= 20:
			return domain.SeverityMedium
		default:
			return domain.SeverityLow
		}
	}
}

// summaryFor formats a short human-readable summary for an anomaly.
func summaryFor(a domain.Anomaly) string {
	msg := a.Message
	if len(msg) > 80 {
		msg = msg[:77] + "..."
	}
	switch a.Kind {
	case domain.AnomalySpike:
		return msg
	case domain.AnomalyPattern:
		return fmt.Sprintf("repeated %d×: %s", a.Count, msg)
	case domain.AnomalyCluster:
		return fmt.Sprintf("cluster %d×: %s", a.Count, msg)
	}
	return msg
}

// detailFor builds the detail text showing line range and a sample.
func detailFor(a domain.Anomaly) string {
	var sb strings.Builder
	if a.FirstLine > 0 {
		if a.FirstLine == a.LastLine {
			fmt.Fprintf(&sb, "line %d", a.FirstLine)
		} else {
			fmt.Fprintf(&sb, "lines %d–%d", a.FirstLine, a.LastLine)
		}
	}
	if len(a.SampleLines) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sample := a.SampleLines[0]
		if len(sample) > 120 {
			sample = sample[:117] + "..."
		}
		sb.WriteString("sample: " + sample)
	}
	return sb.String()
}
