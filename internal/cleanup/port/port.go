// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package port

import (
	"context"
	"time"

	"annave.tech/cli/internal/cleanup/domain"
)

// ScanOptions parameterises the cleanup scan.
type ScanOptions struct {
	Namespace      string
	KubeContext    string
	KubeconfigPath string
	DryRun         bool
	MaxPendingAge  time.Duration // how long a pod may stay Pending before it's flagged (default 10m)
}

// Scanner reads the cluster and returns a CleanupPlan of idle resources.
type Scanner interface {
	Scan(ctx context.Context, opts ScanOptions) (*domain.CleanupPlan, error)
}
