// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"sync"
	"time"

	"annave.tech/cli/internal/security/domain"
	"annave.tech/cli/internal/security/port"
)

// SecurityScanner implements port.Auditor with real scanners.
type SecurityScanner struct{}

func New() *SecurityScanner { return &SecurityScanner{} }

func (s *SecurityScanner) Audit(ctx context.Context, target domain.AuditTarget, opts port.AuditOptions) (*domain.AuditReport, error) {
	var findings []domain.Finding
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	run := func(fn func() ([]domain.Finding, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ff, err := fn()
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
				return
			}
			findings = append(findings, ff...)
		}()
	}

	switch opts.Type {
	case domain.AuditTypeSAST:
		// Run Go and TypeScript SAST concurrently.
		run(func() ([]domain.Finding, error) { return scanSASTGo(ctx, target.Path) })
		run(func() ([]domain.Finding, error) { return scanSASTTypeScript(ctx, target.Path) })

	case domain.AuditTypeK8sLive:
		run(func() ([]domain.Finding, error) { return scanK8sLive(ctx, opts) })

	case domain.AuditTypeK8sLocal:
		run(func() ([]domain.Finding, error) { return scanK8sLocal(ctx, target.Path) })

	default: // secrets is the default
		run(func() ([]domain.Finding, error) { return scanSecrets(ctx, target.Path) })
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	return buildAuditReport(target, opts.Type, findings), nil
}

// buildAuditReport assembles the final AuditReport from raw findings.
func buildAuditReport(target domain.AuditTarget, auditType domain.AuditType, findings []domain.Finding) *domain.AuditReport {
	t := auditType
	if t == "" {
		t = domain.AuditTypeSecrets
	}
	summary := make(map[domain.Severity]int)
	for _, f := range findings {
		summary[f.Severity]++
	}
	return &domain.AuditReport{
		Target:    domain.AuditTarget{Path: target.Path, Type: t},
		Findings:  findings,
		ScannedAt: time.Now(),
		Summary:   summary,
	}
}
