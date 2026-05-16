// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"annave.tech/cli/internal/infra/domain"
	"annave.tech/cli/internal/infra/port"
)

// InfraValidator implements port.Validator.
type InfraValidator struct{}

func New() *InfraValidator { return &InfraValidator{} }

func (v *InfraValidator) Validate(_ context.Context, target string, opts port.ValidateOptions) (*domain.ValidationResult, error) {
	vtype := opts.Type
	if vtype == "" {
		vtype = detectType(target)
	}

	switch vtype {
	case string(domain.ValidateTypeTerraform):
		return validateTerraform(target)
	case string(domain.ValidateTypeHelm):
		return validateHelm(target)
	case string(domain.ValidateTypeK8s):
		return validateK8sYAML(target)
	default:
		return nil, fmt.Errorf("unknown validate type %q — use terraform, helm, or k8s", vtype)
	}
}

// detectType infers the validation type from the target path.
func detectType(target string) string {
	if target == "" || target == "." {
		return string(domain.ValidateTypeHelm) // default cluster check
	}

	info, err := os.Stat(target)
	if err != nil {
		return string(domain.ValidateTypeK8s) // assume YAML file
	}

	if info.IsDir() {
		// Directory containing Chart.yaml → helm chart.
		if _, err := os.Stat(filepath.Join(target, "Chart.yaml")); err == nil {
			return string(domain.ValidateTypeHelm)
		}
		return string(domain.ValidateTypeK8s)
	}

	ext := strings.ToLower(filepath.Ext(target))
	switch ext {
	case ".tfplan":
		return string(domain.ValidateTypeTerraform)
	case ".json":
		return string(domain.ValidateTypeTerraform)
	case ".yaml", ".yml":
		return string(domain.ValidateTypeK8s)
	}
	return string(domain.ValidateTypeK8s)
}

// hasHigherThan returns true if any issue has severity higher than threshold.
// Severity order: critical > high > medium > low > info
func hasHigherThan(issues []domain.ValidationIssue, threshold string) bool {
	rank := map[string]int{
		domain.SeverityCritical: 4,
		domain.SeverityHigh:     3,
		domain.SeverityMedium:   2,
		domain.SeverityLow:      1,
		domain.SeverityInfo:     0,
	}
	thresholdRank := rank[threshold]
	for _, issue := range issues {
		if rank[issue.Severity] >= thresholdRank {
			return true
		}
	}
	return false
}
