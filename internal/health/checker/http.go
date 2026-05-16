// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"annave.tech/cli/internal/health/domain"
)

// checkHTTP performs an HTTP(S) GET against target and returns a HealthResult.
func checkHTTP(ctx context.Context, target domain.ServiceTarget) domain.HealthResult {
	start := time.Now()
	result := domain.HealthResult{Target: target, CheckedAt: start}

	client := &http.Client{
		Timeout: target.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.Address, nil)
	if err != nil {
		result.Status = domain.StatusDown
		result.Message = err.Error()
		return result
	}
	req.Header.Set("User-Agent", "annave-health/1.0")

	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		if ctx.Err() != nil || isTimeout(err) {
			result.Status = domain.StatusTimeout
			result.Message = "request timed out"
		} else {
			result.Status = domain.StatusDown
			result.Message = trimError(err.Error())
		}
		return result
	}
	defer resp.Body.Close()

	// Read body only if we need a substring match (up to 64 KB).
	var body string
	if target.ExpectBody != "" {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		body = string(b)
	}

	statusOK := false
	if target.ExpectStatus == 0 {
		statusOK = resp.StatusCode >= 200 && resp.StatusCode < 300
	} else {
		statusOK = resp.StatusCode == target.ExpectStatus
	}

	if !statusOK {
		result.Status = domain.StatusDown
		result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result
	}

	if target.ExpectBody != "" && !strings.Contains(body, target.ExpectBody) {
		result.Status = domain.StatusDown
		result.Message = fmt.Sprintf("body does not contain %q", target.ExpectBody)
		return result
	}

	result.Status = domain.StatusUp
	result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return result
}

// isTimeout returns true when the error represents a network or context timeout.
func isTimeout(err error) bool {
	type timeoutErr interface{ Timeout() bool }
	if t, ok := err.(timeoutErr); ok {
		return t.Timeout()
	}
	return strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded")
}

// trimError extracts the most useful segment of a verbose net/http error string.
func trimError(s string) string {
	// net/http wraps errors verbosely; return the last meaningful segment.
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}
