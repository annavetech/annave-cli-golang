// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package port

import (
	"context"
	"time"

	"annave.tech/cli/internal/cost/domain"
)

// AnalyzeOptions parameterises the cost analysis.
type AnalyzeOptions struct {
	Provider string    // aws, gcp, azure
	Since    time.Time // start of billing period
}

// CostAnalyzer fetches cost data for the configured provider and period.
type CostAnalyzer interface {
	Analyze(ctx context.Context, opts AnalyzeOptions) (*domain.CostReport, error)
}
