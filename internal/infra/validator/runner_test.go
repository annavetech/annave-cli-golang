// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package validator

import (
	"strings"
	"testing"
)

func TestToolCommandAllowsOnlyHelmAndTerraform(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "/usr/local/bin/helm"},
		{name: "terraform"},
		{name: "terraform.exe"},
		{name: "/bin/sh", wantErr: true},
		{name: "helmfile", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := toolCommand(t.Context(), tt.name, []string{"lint", "a b"})
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "refusing to run") {
					t.Fatalf("toolCommand(%q) error = %v, want refusal", tt.name, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("toolCommand(%q) = %v", tt.name, err)
			}
			if got := cmd.Args[1:]; len(got) != 2 || got[0] != "lint" || got[1] != "a b" {
				t.Fatalf("Args = %q, want the arguments unchanged after the name", cmd.Args)
			}
		})
	}
}

func TestToolCommandIsBoundToContext(t *testing.T) {
	cmd, err := toolCommand(t.Context(), "helm", []string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Cancel == nil {
		t.Fatal("helm and terraform must be killed when the context ends")
	}
}
