// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package domain

import (
	"encoding/json"
	"time"
)

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
	Name         string        `json:"name"`    // human label; defaults to the raw address string
	Address      string        `json:"address"` // resolved address passed to the checker
	CheckType    CheckType     `json:"check_type"`
	Timeout      time.Duration `json:"-"`             // written as timeout_ms by MarshalJSON
	ExpectStatus int           `json:"expect_status"` // HTTP only: expected status code (0 = any 2xx)
	ExpectBody   string        `json:"expect_body"`   // HTTP only: required substring in response body
}

// MarshalJSON writes the timeout as timeout_ms in milliseconds.
func (t ServiceTarget) MarshalJSON() ([]byte, error) {
	type target ServiceTarget
	return json.Marshal(struct {
		target
		TimeoutMS float64 `json:"timeout_ms"`
	}{target(t), durationMS(t.Timeout)})
}

// HealthResult is the outcome of checking one target.
type HealthResult struct {
	Target    ServiceTarget `json:"target"`
	Status    Status        `json:"status"`
	Latency   time.Duration `json:"latency_ms"`
	Message   string        `json:"message,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

// MarshalJSON writes latency_ms in milliseconds; time.Duration alone would marshal as nanoseconds.
func (r HealthResult) MarshalJSON() ([]byte, error) {
	type result HealthResult
	return json.Marshal(struct {
		result
		Latency float64 `json:"latency_ms"`
	}{result(r), durationMS(r.Latency)})
}

// durationMS converts d to milliseconds.
func durationMS(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// HealthReport is the full output of a health check run.
type HealthReport struct {
	CheckedAt time.Time      `json:"checked_at"`
	Results   []HealthResult `json:"results"`
	Up        int            `json:"up"`
	Down      int            `json:"down"`
	Total     int            `json:"total"`
}
