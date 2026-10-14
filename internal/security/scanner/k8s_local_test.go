// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package scanner

import (
	"os"
	"strings"
	"testing"

	"annave.tech/cli/internal/security/domain"
)

func localFindings(t *testing.T, fixture string) []domain.Finding {
	t.Helper()
	root, err := os.OpenRoot("testdata")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	findings, err := checkManifestFile(root, fixture, fixture)
	if err != nil {
		t.Fatal(err)
	}
	return findings
}

func TestK8sLocalCronJobPrivileged(t *testing.T) {
	got := findByID(localFindings(t, "cronjob-privileged.yaml"), "K8S004")
	if len(got) != 1 {
		t.Fatalf("want 1 K8S004 for the CronJob container, got %+v", got)
	}
}

func TestK8sLocalPodLevelRunAsNonRoot(t *testing.T) {
	if got := findByID(localFindings(t, "deployment-pod-nonroot.yaml"), "K8S005"); len(got) != 0 {
		t.Fatalf("pod-level runAsNonRoot should clear K8S005, got %+v", got)
	}
}

func TestK8sLocalInitAndEphemeralContainers(t *testing.T) {
	findings := localFindings(t, "pod-init-ephemeral.yaml")

	priv := findByID(findings, "K8S004")
	if len(priv) != 2 {
		t.Fatalf("want K8S004 for init and ephemeral containers, got %+v", priv)
	}
	if !strings.Contains(priv[0].File, "init container: setup") || !strings.Contains(priv[1].File, "ephemeral container: shell") {
		t.Errorf("unexpected refs: %q, %q", priv[0].File, priv[1].File)
	}
	// Init and ephemeral containers cannot have readiness probes; ephemeral ones cannot set limits.
	if got := findByID(findings, "K8S007"); len(got) != 0 {
		t.Errorf("K8S007 must not fire for init or ephemeral containers, got %+v", got)
	}
	if got := findByID(findings, "K8S006"); len(got) != 0 {
		t.Errorf("K8S006 must not fire here, got %+v", got)
	}
}
