// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package checker

import (
	"context"
	"fmt"
	"net"
	"time"

	"annave.tech/cli/internal/health/domain"
)

// checkTCP dials a TCP address and reports whether the port is reachable.
func checkTCP(ctx context.Context, target domain.ServiceTarget) domain.HealthResult {
	start := time.Now()
	result := domain.HealthResult{Target: target, CheckedAt: start}

	d := net.Dialer{Timeout: target.Timeout}
	conn, err := d.DialContext(ctx, "tcp", target.Address)
	result.Latency = time.Since(start)

	if err != nil {
		if ctx.Err() != nil || isTimeout(err) {
			result.Status = domain.StatusTimeout
			result.Message = "connection timed out"
		} else {
			result.Status = domain.StatusDown
			result.Message = trimError(err.Error())
		}
		return result
	}
	conn.Close()

	result.Status = domain.StatusUp
	result.Message = fmt.Sprintf("TCP %s open", target.Address)
	return result
}
