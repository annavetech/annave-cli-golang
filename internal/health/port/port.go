// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package port

import (
	"context"

	"annave.tech/cli/internal/health/domain"
)

// CheckOptions configures a health check run.
type CheckOptions struct {
	Concurrency int // 0 = auto (min of target count and config limit)
}

// Checker is the port for service health checking.
type Checker interface {
	Check(ctx context.Context, targets []domain.ServiceTarget, opts CheckOptions) (*domain.HealthReport, error)
}
