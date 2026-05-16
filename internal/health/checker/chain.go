// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package checker

import (
	"context"

	"annave.tech/cli/internal/health/domain"
)

// CheckChain runs targets sequentially, stopping at the first non-up result.
// Useful for dependency chains: only check B if A is healthy.
func CheckChain(ctx context.Context, targets []domain.ServiceTarget) []domain.HealthResult {
	results := make([]domain.HealthResult, 0, len(targets))
	for _, t := range targets {
		r := checkOne(ctx, t)
		results = append(results, r)
		if r.Status != domain.StatusUp {
			break
		}
	}
	return results
}
