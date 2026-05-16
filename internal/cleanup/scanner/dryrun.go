// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"fmt"
	"time"

	"annave.tech/cli/internal/cleanup/domain"
)

// GroupByType groups idle resources by their ResourceType.
func GroupByType(resources []domain.IdleResource) map[domain.ResourceType][]domain.IdleResource {
	groups := make(map[domain.ResourceType][]domain.IdleResource)
	for _, r := range resources {
		k := r.Resource.Kind
		groups[k] = append(groups[k], r)
	}
	return groups
}

// SortedTypes returns resource types in a stable, meaningful display order.
func SortedTypes(groups map[domain.ResourceType][]domain.IdleResource) []domain.ResourceType {
	order := []domain.ResourceType{
		domain.ResourcePod,
		domain.ResourcePVC,
		domain.ResourceConfigMap,
		domain.ResourceNamespace,
	}
	var result []domain.ResourceType
	for _, t := range order {
		if _, ok := groups[t]; ok {
			result = append(result, t)
		}
	}
	return result
}

// FormatAge returns a compact human-readable duration string.
func FormatAge(hours int) string {
	d := time.Duration(hours) * time.Hour
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d >= time.Hour:
		return fmt.Sprintf("%dh", hours)
	default:
		return "<1h"
	}
}
