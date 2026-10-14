// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package port

import (
	"context"

	"annave.tech/cli/internal/security/domain"
)

// AuditOptions parameterises the security audit.
type AuditOptions struct {
	Type           domain.AuditType
	KubeconfigPath string
	KubeContext    string
	Namespace      string // k8s-live only; empty means all namespaces
}

// Auditor runs a security scan against a target and returns findings.
type Auditor interface {
	Audit(ctx context.Context, target domain.AuditTarget, opts AuditOptions) (*domain.AuditReport, error)
}
