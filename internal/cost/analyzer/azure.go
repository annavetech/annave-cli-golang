// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package analyzer

import (
	"context"

	"annave.tech/cli/internal/cost/domain"
	"annave.tech/cli/internal/cost/port"
	shared "annave.tech/cli/internal/shared/errors"
)

// azureAnalyzer is a stub — Azure Cost Management API integration is planned.
type azureAnalyzer struct{}

func newAzure() *azureAnalyzer { return &azureAnalyzer{} }

func (a *azureAnalyzer) Analyze(_ context.Context, _ port.AnalyzeOptions) (*domain.CostReport, error) {
	return nil, shared.New(
		shared.ErrCodeNotImplemented,
		shared.StageRuntime,
		"Azure cost analysis is not implemented yet. Use --provider aws.",
	)
}
