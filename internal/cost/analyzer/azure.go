// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package analyzer

import (
	"context"

	"github.com/annavetech/tend/internal/cost/domain"
	"github.com/annavetech/tend/internal/cost/port"
	shared "github.com/annavetech/tend/internal/shared/errors"
)

// azureAnalyzer is a stub until the Azure Cost Management API integration exists.
type azureAnalyzer struct{}

func newAzure() *azureAnalyzer { return &azureAnalyzer{} }

func (a *azureAnalyzer) Analyze(_ context.Context, _ port.AnalyzeOptions) (*domain.CostReport, error) {
	return nil, shared.New(
		shared.ErrCodeNotImplemented,
		shared.StageRuntime,
		"Azure cost analysis is not implemented yet. Use --provider aws.",
	)
}
