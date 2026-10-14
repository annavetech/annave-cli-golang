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

// gcpAnalyzer is a stub until the GCP billing API integration exists.
type gcpAnalyzer struct{}

func newGCP() *gcpAnalyzer { return &gcpAnalyzer{} }

func (g *gcpAnalyzer) Analyze(_ context.Context, _ port.AnalyzeOptions) (*domain.CostReport, error) {
	return nil, shared.New(
		shared.ErrCodeNotImplemented,
		shared.StageRuntime,
		"GCP cost analysis is not implemented yet. Use --provider aws.",
	)
}
