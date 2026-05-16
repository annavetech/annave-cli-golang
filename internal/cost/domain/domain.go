// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// Currency is the billing currency code.
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
)

// CostRecord is the aggregated cost for one service over the query period.
type CostRecord struct {
	Service  string   `json:"service"`
	Resource string   `json:"resource"`
	Amount   float64  `json:"amount"`
	Currency Currency `json:"currency"`
	Period   string   `json:"period"` // e.g. "2026-04"
}

// CostAnomaly is a service whose recent cost has spiked beyond the rolling average.
type CostAnomaly struct {
	Service  string  `json:"service"`
	Message  string  `json:"message"`
	Expected float64 `json:"expected"`
	Actual   float64 `json:"actual"`
	DeltaPct float64 `json:"delta_pct"` // percentage above expected
}

// ResourceAttribution tags a cost amount to a specific resource or forecast label.
type ResourceAttribution struct {
	Resource string            `json:"resource"`
	Tags     map[string]string `json:"tags,omitempty"`
	Amount   float64           `json:"amount"`
	Currency Currency          `json:"currency"`
}

// CostReport is the top-level result returned by the CostAnalyzer.
type CostReport struct {
	Provider  string               `json:"provider"`
	Period    string               `json:"period"`
	TotalCost float64              `json:"total_cost"`
	Currency  Currency             `json:"currency"`
	Records   []CostRecord         `json:"records"`
	Anomalies []CostAnomaly        `json:"anomalies"`
	ByResource []ResourceAttribution `json:"by_resource,omitempty"`
	ScannedAt time.Time            `json:"scanned_at"`
}
