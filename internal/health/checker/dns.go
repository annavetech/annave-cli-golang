// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package checker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"annave.tech/cli/internal/health/domain"
)

// checkDNS resolves target.Address using the system resolver.
func checkDNS(ctx context.Context, target domain.ServiceTarget) domain.HealthResult {
	start := time.Now()
	result := domain.HealthResult{Target: target, CheckedAt: start}

	addrs, err := defaultResolver.LookupHost(ctx, target.Address)
	result.Latency = time.Since(start)

	if err != nil {
		if ctx.Err() != nil {
			result.Status = domain.StatusTimeout
			result.Message = "DNS resolution timed out"
		} else {
			result.Status = domain.StatusDown
			result.Message = trimError(err.Error())
		}
		return result
	}

	result.Status = domain.StatusUp
	result.Message = fmt.Sprintf("%d address(es): %s", len(addrs), strings.Join(addrs, ", "))
	return result
}
