// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package port

import (
	"context"
	"io"
	"time"

	"github.com/annavetech/tend/internal/log/domain"
)

// AnalyzeOptions controls filtering applied before analysis.
type AnalyzeOptions struct {
	Since    time.Time
	MinLevel string
	MaxLines int
}

// Analyzer reads a log stream and returns a ranked findings report.
type Analyzer interface {
	Analyze(ctx context.Context, r io.Reader, filename string, opts AnalyzeOptions) (*domain.AnalysisReport, error)
}
