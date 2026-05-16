// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// ValidateType identifies the infra target type.
type ValidateType string

const (
	ValidateTypeTerraform ValidateType = "terraform"
	ValidateTypeHelm      ValidateType = "helm"
	ValidateTypeK8s       ValidateType = "k8s"
)

// Severity constants for ValidationIssue.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// ChangeType is the Terraform action applied to a resource.
type ChangeType string

const (
	ChangeAdd     ChangeType = "add"
	ChangeUpdate  ChangeType = "update"
	ChangeDelete  ChangeType = "delete"
	ChangeReplace ChangeType = "replace"
)

// Change describes one resource change in a Terraform plan.
type Change struct {
	Resource   string     `json:"resource"`
	ChangeType ChangeType `json:"change_type"`
	Before     string     `json:"before,omitempty"`
	After      string     `json:"after,omitempty"`
}

// TerraformPlan holds the parsed contents of a terraform show -json plan file.
type TerraformPlan struct {
	PlanFile  string    `json:"plan_file"`
	Changes   []Change  `json:"changes"`
	ParsedAt  time.Time `json:"parsed_at"`
}

// HelmRelease is a deployed Helm release as reported by helm list.
type HelmRelease struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Chart     string `json:"chart"`
	Version   string `json:"version"`
	Status    string `json:"status"`
}

// ValidationIssue is a single rule violation found during validation.
type ValidationIssue struct {
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Resource string `json:"resource,omitempty"`
}

// ValidationResult is the top-level result returned by the Validator.
type ValidationResult struct {
	Target      string            `json:"target"`
	Passed      bool              `json:"passed"`
	Issues      []ValidationIssue `json:"issues"`
	ValidatedAt time.Time         `json:"validated_at"`
}
