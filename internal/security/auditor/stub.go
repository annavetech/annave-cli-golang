// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package auditor

import (
	"context"

	"github.com/annavetech/tend/internal/security/domain"
	"github.com/annavetech/tend/internal/security/port"
	shared "github.com/annavetech/tend/internal/shared/errors"
)

// StubAuditor satisfies port.Auditor while the module is in development.
type StubAuditor struct{}

func New() *StubAuditor { return &StubAuditor{} }

func (s *StubAuditor) Audit(_ context.Context, _ domain.AuditTarget, _ port.AuditOptions) (*domain.AuditReport, error) {
	return nil, shared.NotImplemented("security audit")
}
