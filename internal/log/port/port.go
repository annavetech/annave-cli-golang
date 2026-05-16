// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package port

import (
	"io"
	"time"

	"annave.tech/cli/internal/log/domain"
)

// AnalyzeOptions controls filtering applied before analysis.
type AnalyzeOptions struct {
	Since    time.Time
	MinLevel string
	MaxLines int
}

// Analyzer reads a log stream and returns a ranked findings report.
type Analyzer interface {
	Analyze(r io.Reader, filename string, opts AnalyzeOptions) (*domain.AnalysisReport, error)
}
