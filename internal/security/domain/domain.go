// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// Severity is the impact level of a security finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// AuditType selects which scan to run.
type AuditType string

const (
	AuditTypeSecrets   AuditType = "secrets"   // secret / credential scanning
	AuditTypeSAST      AuditType = "sast"      // static analysis (Go + TypeScript)
	AuditTypeK8sLive   AuditType = "k8s-live"  // live cluster misconfiguration
	AuditTypeK8sLocal  AuditType = "k8s-local" // local YAML manifest validation
	AuditTypeContainer AuditType = "container" // image scanning (stub)
)

// AuditTarget describes what is being audited.
type AuditTarget struct {
	Path string    `json:"path"`
	Type AuditType `json:"type"`
}

// Finding is one security issue discovered during an audit.
type Finding struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Severity    Severity `json:"severity"`
	File        string   `json:"file,omitempty"`
	Line        int      `json:"line,omitempty"`
	Detail      string   `json:"detail,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

// AuditReport is the top-level result returned by the Auditor.
type AuditReport struct {
	Target    AuditTarget      `json:"target"`
	Findings  []Finding        `json:"findings"`
	ScannedAt time.Time        `json:"scanned_at"`
	Summary   map[Severity]int `json:"summary"`
}
