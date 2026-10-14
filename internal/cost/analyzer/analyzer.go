// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package analyzer

import (
	"context"
	"fmt"

	"github.com/annavetech/tend/internal/cost/domain"
	"github.com/annavetech/tend/internal/cost/port"
)

// CostAnalyzer implements port.CostAnalyzer, routing to the correct provider adapter.
type CostAnalyzer struct{}

// New returns a CostAnalyzer routed to the correct provider adapter.
func New() *CostAnalyzer { return &CostAnalyzer{} }

func (c *CostAnalyzer) Analyze(ctx context.Context, opts port.AnalyzeOptions) (*domain.CostReport, error) {
	switch opts.Provider {
	case "aws", "":
		return newAWS().Analyze(ctx, opts)
	case "gcp":
		return newGCP().Analyze(ctx, opts)
	case "azure":
		return newAzure().Analyze(ctx, opts)
	default:
		return nil, fmt.Errorf("unknown provider %q: use aws, gcp, or azure", opts.Provider)
	}
}
