// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package config

import (
	"bytes"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestLimitsYAMLHasOnlyKnownKeys fails when limits.yaml holds a key that no Limits field reads.
func TestLimitsYAMLHasOnlyKnownKeys(t *testing.T) {
	dec := yaml.NewDecoder(bytes.NewReader(limitsYAML))
	dec.KnownFields(true)
	var l Limits
	if err := dec.Decode(&l); err != nil {
		t.Fatalf("limits.yaml: %v", err)
	}
	if l.Log.MaxLines <= 0 || l.Health.TimeoutSeconds <= 0 || l.Health.MaxTargets <= 0 ||
		l.Doc.MaxFileSizeMB <= 0 || l.Doc.MaxResults <= 0 || l.Doc.MaxDepth <= 0 ||
		l.Cleanup.TimeoutSeconds <= 0 || l.Cleanup.MaxResources <= 0 {
		t.Fatalf("limits.yaml leaves a limit unset: %+v", l)
	}
}
