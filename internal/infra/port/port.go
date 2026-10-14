// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package port

import (
	"context"

	"github.com/annavetech/tend/internal/infra/domain"
)

// ValidateOptions parameterises the infra validation.
type ValidateOptions struct {
	Type ValidateType // if empty, auto-detected from target
}

type ValidateType = string // alias to domain.ValidateType values

// Validator analyses an infra target and returns a ValidationResult.
type Validator interface {
	Validate(ctx context.Context, target string, opts ValidateOptions) (*domain.ValidationResult, error)
}
