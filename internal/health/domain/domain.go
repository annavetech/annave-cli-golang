// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// CheckType identifies the protocol used for a health check.
type CheckType string

const (
	CheckHTTP CheckType = "http"
	CheckTCP  CheckType = "tcp"
	CheckDNS  CheckType = "dns"
)

// Status is the outcome of a single check.
type Status string

const (
	StatusUp      Status = "up"
	StatusDown    Status = "down"
	StatusTimeout Status = "timeout"
	StatusUnknown Status = "unknown"
)

// ServiceTarget describes a single service to check.
type ServiceTarget struct {
	Name         string        // human label; defaults to the raw address string
	Address      string        // resolved address passed to the checker
	CheckType    CheckType
	Timeout      time.Duration
	ExpectStatus int    // HTTP only: expected status code (0 = any 2xx)
	ExpectBody   string // HTTP only: required substring in response body
}

// HealthResult is the outcome of checking one target.
type HealthResult struct {
	Target    ServiceTarget `json:"target"`
	Status    Status        `json:"status"`
	Latency   time.Duration `json:"latency_ms"`
	Message   string        `json:"message,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

// HealthReport is the full output of a health check run.
type HealthReport struct {
	CheckedAt time.Time      `json:"checked_at"`
	Results   []HealthResult `json:"results"`
	Up        int            `json:"up"`
	Down      int            `json:"down"`
	Total     int            `json:"total"`
}
