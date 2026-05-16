// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package auditor

import (
	"context"

	"annave.tech/cli/internal/security/domain"
	"annave.tech/cli/internal/security/port"
	shared "annave.tech/cli/internal/shared/errors"
)

// StubAuditor satisfies port.Auditor while the module is in development.
type StubAuditor struct{}

func New() *StubAuditor { return &StubAuditor{} }

func (s *StubAuditor) Audit(_ context.Context, _ domain.AuditTarget, _ port.AuditOptions) (*domain.AuditReport, error) {
	return nil, shared.NotImplemented("security audit")
}
