// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package scanner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/annavetech/tend/internal/cleanup/port"
)

func writeKubeconfig(t *testing.T, server string) string {
	t.Helper()
	cfg := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: %s
contexts:
- name: ctx
  context:
    cluster: c
    user: u
current-context: ctx
users:
- name: u
  user: {}
`, server)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanUnreachableClusterReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	plan, err := New().Scan(ctx, port.ScanOptions{KubeconfigPath: writeKubeconfig(t, "https://127.0.0.1:1")})
	if err == nil {
		t.Fatalf("want an error for an unreachable cluster, got plan %+v", plan)
	}
}

// fakeAPI serves empty lists, one old namespace "stale", and 403 for the paths in forbidden.
func fakeAPI(t *testing.T, forbidden ...string) *httptest.Server {
	t.Helper()
	kinds := map[string]string{
		"pods": "PodList", "persistentvolumeclaims": "PersistentVolumeClaimList",
		"configmaps": "ConfigMapList", "deployments": "DeploymentList",
		"statefulsets": "StatefulSetList", "services": "ServiceList",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		for _, p := range forbidden {
			if r.URL.Path == p {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"Status","status":"Failure","reason":"Forbidden","code":403}`))
				return
			}
		}
		if r.URL.Path == "/api/v1/namespaces" {
			_, _ = fmt.Fprintf(w, `{"apiVersion":"v1","kind":"NamespaceList","items":[{"metadata":{"name":"stale","creationTimestamp":%q}}]}`,
				time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339))
			return
		}
		seg := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		apiVersion := "v1"
		if strings.HasPrefix(r.URL.Path, "/apis/apps/") {
			apiVersion = "apps/v1"
		}
		_, _ = fmt.Fprintf(w, `{"apiVersion":%q,"kind":%q,"items":[]}`, apiVersion, kinds[seg])
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestScanPartialFailureWarns(t *testing.T) {
	srv := fakeAPI(t, "/api/v1/pods", "/api/v1/namespaces/stale/pods")

	plan, err := New().Scan(context.Background(), port.ScanOptions{KubeconfigPath: writeKubeconfig(t, srv.URL)})
	if err != nil {
		t.Fatal(err)
	}
	warnings := strings.Join(plan.Warnings, "\n")
	if !strings.Contains(warnings, "pods not scanned") {
		t.Errorf("want a warning for the failed pod scan, got %q", warnings)
	}
	if !strings.Contains(warnings, "namespace stale not checked") {
		t.Errorf("want a warning for namespace stale, got %q", warnings)
	}
}

func TestScanHealthyClusterHasNoWarnings(t *testing.T) {
	srv := fakeAPI(t)

	plan, err := New().Scan(context.Background(), port.ScanOptions{KubeconfigPath: writeKubeconfig(t, srv.URL)})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Warnings) != 0 || len(plan.Resources) != 1 {
		t.Fatalf("want one empty namespace and no warnings, got %+v", plan)
	}
}
