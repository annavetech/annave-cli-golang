// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
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
}

// Auditor runs a security scan against a target and returns findings.
type Auditor interface {
	Audit(ctx context.Context, target domain.AuditTarget, opts AuditOptions) (*domain.AuditReport, error)
}
