// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package checker

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"annave.tech/cli/internal/health/domain"
	"annave.tech/cli/internal/health/port"
	"annave.tech/cli/internal/shared/config"
)

// defaultResolver is package-level so dns.go can use it without importing net separately.
var defaultResolver = net.DefaultResolver

// HealthChecker implements port.Checker.
type HealthChecker struct{}

func New() *HealthChecker { return &HealthChecker{} }

func (hc *HealthChecker) Check(ctx context.Context, targets []domain.ServiceTarget, opts port.CheckOptions) (*domain.HealthReport, error) {
	if len(targets) == 0 {
		return &domain.HealthReport{CheckedAt: time.Now()}, nil
	}

	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = min(len(targets), config.App.Limits.Health.MaxTargets)
	}

	sem := make(chan struct{}, concurrency)
	results := make([]domain.HealthResult, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, target domain.ServiceTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = checkOne(ctx, target)
		}(i, t)
	}
	wg.Wait()

	return buildReport(results), nil
}

func checkOne(ctx context.Context, target domain.ServiceTarget) domain.HealthResult {
	switch target.CheckType {
	case domain.CheckHTTP:
		return checkHTTP(ctx, target)
	case domain.CheckTCP:
		return checkTCP(ctx, target)
	case domain.CheckDNS:
		return checkDNS(ctx, target)
	}
	return domain.HealthResult{
		Target:    target,
		Status:    domain.StatusUnknown,
		Message:   "unknown check type",
		CheckedAt: time.Now(),
	}
}

func buildReport(results []domain.HealthResult) *domain.HealthReport {
	r := &domain.HealthReport{
		CheckedAt: time.Now(),
		Results:   results,
		Total:     len(results),
	}
	for _, res := range results {
		if res.Status == domain.StatusUp {
			r.Up++
		} else {
			r.Down++
		}
	}
	return r
}

// ParseTarget parses a target string into a ServiceTarget.
//
//	https://example.com      → HTTP check
//	tcp://host:port          → TCP check
//	dns://hostname           → DNS check
//	host:port                → TCP (auto-detected)
//	hostname                 → DNS (auto-detected)
func ParseTarget(s string, defaultTimeout time.Duration) domain.ServiceTarget {
	t := domain.ServiceTarget{
		Name:    s,
		Address: s,
		Timeout: defaultTimeout,
	}

	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://"):
		t.CheckType = domain.CheckHTTP
	case strings.HasPrefix(lower, "tcp://"):
		t.CheckType = domain.CheckTCP
		t.Address = s[6:]
	case strings.HasPrefix(lower, "dns://"):
		t.CheckType = domain.CheckDNS
		t.Address = s[6:]
	default:
		if strings.Contains(s, ":") {
			t.CheckType = domain.CheckTCP
		} else {
			t.CheckType = domain.CheckDNS
		}
	}
	return t
}
