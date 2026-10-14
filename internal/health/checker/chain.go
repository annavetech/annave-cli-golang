// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package checker

import (
	"context"
	"errors"

	"annave.tech/cli/internal/health/domain"
)

// CheckChain runs targets sequentially, stopping at the first non-up result.
// Useful for dependency chains: only check B if A is healthy.
// The error joins any connection close failures; the results are complete either way.
func CheckChain(ctx context.Context, targets []domain.ServiceTarget) ([]domain.HealthResult, error) {
	results := make([]domain.HealthResult, 0, len(targets))
	var errs []error
	for _, t := range targets {
		r, err := checkOne(ctx, t)
		if err != nil {
			errs = append(errs, err)
		}
		results = append(results, r)
		if r.Status != domain.StatusUp {
			break
		}
	}
	return results, errors.Join(errs...)
}
