// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// Severity is the impact level of a log finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// LogFormat identifies the log line structure detected in the input.
type LogFormat string

const (
	FormatJSON   LogFormat = "json"
	FormatNginx  LogFormat = "nginx"
	FormatSyslog LogFormat = "syslog"
	FormatPlain  LogFormat = "plain"
)

// LogEntry is a parsed line from a log file.
type LogEntry struct {
	Line      int
	Timestamp time.Time
	Level     string
	Message   string
	Raw       string
	Format    LogFormat
}

// AnomalyKind classifies the detection method that produced an anomaly.
type AnomalyKind string

const (
	AnomalySpike   AnomalyKind = "spike"
	AnomalyPattern AnomalyKind = "pattern"
	AnomalyCluster AnomalyKind = "cluster"
)

// Anomaly is a detected irregularity in the log stream.
type Anomaly struct {
	Kind        AnomalyKind `json:"kind"`
	Message     string      `json:"message"`
	Count       int         `json:"count"`
	FirstLine   int         `json:"first_line"`
	LastLine    int         `json:"last_line"`
	FirstSeen   time.Time   `json:"first_seen,omitempty"`
	LastSeen    time.Time   `json:"last_seen,omitempty"`
	SampleLines []string    `json:"sample_lines,omitempty"`
}

// Finding is a ranked, user-facing result produced by the analysis.
type Finding struct {
	Rank     int      `json:"rank"`
	Severity Severity `json:"severity"`
	Summary  string   `json:"summary"`
	Detail   string   `json:"detail,omitempty"`
	Count    int      `json:"count"`
	Anomaly  Anomaly  `json:"anomaly"`
}

// TimeRange represents the span of timestamps seen in the analysed log.
type TimeRange struct {
	From time.Time `json:"from,omitempty"`
	To   time.Time `json:"to,omitempty"`
}

// AnalysisReport is the top-level result returned by the Analyzer.
type AnalysisReport struct {
	File        string    `json:"file"`
	Format      LogFormat `json:"format"`
	TotalLines  int       `json:"total_lines"`
	ParsedLines int       `json:"parsed_lines"`
	TimeRange   TimeRange `json:"time_range,omitempty"`
	Findings    []Finding `json:"findings"`
	ParseErrors int       `json:"parse_errors,omitempty"`
}
