// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package domain

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestHealthResultLatencyIsMilliseconds(t *testing.T) {
	r := HealthResult{Status: StatusUp, Latency: 470 * time.Microsecond}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	ms, ok := got["latency_ms"].(float64)
	if !ok || math.Abs(ms-0.47) > 1e-9 {
		t.Fatalf("latency_ms = %v, want 0.47 in %s", got["latency_ms"], b)
	}
	if got["status"] != "up" {
		t.Fatalf("status = %v, want up in %s", got["status"], b)
	}
}

func TestHealthReportLatencyIsMilliseconds(t *testing.T) {
	rep := HealthReport{Results: []HealthResult{{Latency: 142 * time.Millisecond}}}
	b, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Results []struct {
			LatencyMS float64 `json:"latency_ms"`
		} `json:"results"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || got.Results[0].LatencyMS != 142 {
		t.Fatalf("results = %+v, want latency_ms 142 in %s", got.Results, b)
	}
}

func TestServiceTargetJSONKeys(t *testing.T) {
	r := HealthResult{Target: ServiceTarget{
		Name: "tcp://db:5432", Address: "db:5432", CheckType: CheckTCP, Timeout: 1500 * time.Millisecond,
	}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Target map[string]any `json:"target"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"name": "tcp://db:5432", "address": "db:5432", "check_type": "tcp",
		"timeout_ms": 1500.0, "expect_status": 0.0, "expect_body": "",
	}
	if len(got.Target) != len(want) {
		t.Fatalf("target = %v, want keys %v", got.Target, want)
	}
	for k, v := range want {
		if got.Target[k] != v {
			t.Fatalf("target[%q] = %v, want %v in %s", k, got.Target[k], v, b)
		}
	}
}
