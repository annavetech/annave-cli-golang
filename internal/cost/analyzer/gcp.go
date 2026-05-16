// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"context"
	"fmt"

	"annave.tech/cli/internal/cost/domain"
	"annave.tech/cli/internal/cost/port"
	shared "annave.tech/cli/internal/shared/errors"
)

// gcpAnalyzer is a stub — GCP billing API integration is planned.
type gcpAnalyzer struct{}

func newGCP() *gcpAnalyzer { return &gcpAnalyzer{} }

func (g *gcpAnalyzer) Analyze(_ context.Context, _ port.AnalyzeOptions) (*domain.CostReport, error) {
	return nil, shared.New(
		shared.ErrCodeNotImplemented,
		shared.StageRuntime,
		fmt.Sprintf("GCP cost analysis is not yet implemented — use --provider aws"),
	)
}
