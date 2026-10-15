// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package validator

import "testing"

func TestClassifyChangeDataResource(t *testing.T) {
	cases := []struct {
		name    string
		actions []string
		rule    string
	}{
		{"delete", []string{"delete"}, "TF001"},
		{"replace delete first", []string{"delete", "create"}, "TF002"},
		{"replace create first", []string{"create", "delete"}, "TF002"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rc := tfResourceChange{Address: "aws_db_instance.main", Type: "aws_db_instance", Change: tfChange{Actions: c.actions}}
			issues := classifyChange(rc)
			if len(issues) != 1 || issues[0].Rule != c.rule {
				t.Fatalf("actions %v: got %+v, want one %s issue", c.actions, issues, c.rule)
			}
		})
	}
}
