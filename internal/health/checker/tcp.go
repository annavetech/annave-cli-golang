// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package checker

import (
	"context"
	"fmt"
	"net"
	"time"

	"annave.tech/cli/internal/health/domain"
)

// dialFunc matches net.Dialer.DialContext.
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// checkTCP dials a TCP address and reports whether the port is reachable.
// The error reports a failure to close the connection; the result is set either way.
func checkTCP(ctx context.Context, target domain.ServiceTarget) (domain.HealthResult, error) {
	d := net.Dialer{Timeout: target.Timeout}
	return checkTCPWith(ctx, target, d.DialContext)
}

func checkTCPWith(ctx context.Context, target domain.ServiceTarget, dial dialFunc) (result domain.HealthResult, err error) {
	start := time.Now()
	result = domain.HealthResult{Target: target, CheckedAt: start}

	conn, dialErr := dial(ctx, "tcp", target.Address)
	result.Latency = time.Since(start)

	if dialErr != nil {
		if ctx.Err() != nil || isTimeout(dialErr) {
			result.Status = domain.StatusTimeout
			result.Message = "connection timed out"
		} else {
			result.Status = domain.StatusDown
			result.Message = trimError(dialErr.Error())
		}
		return result, nil
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close connection to %s: %w", target.Name, cerr)
		}
	}()

	result.Status = domain.StatusUp
	result.Message = fmt.Sprintf("TCP %s open", target.Address)
	return result, nil
}
