// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func privilegedSC() *corev1.SecurityContext {
	yes := true
	return &corev1.SecurityContext{Privileged: &yes}
}

func TestK8sLiveInitAndEphemeralContainers(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "p"},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: "setup", SecurityContext: privilegedSC()}},
			EphemeralContainers: []corev1.EphemeralContainer{{
				EphemeralContainerCommon: corev1.EphemeralContainerCommon{Name: "shell", SecurityContext: privilegedSC()},
			}},
		},
	}
	findings := checkPodSecurity(pod)

	priv := findByID(findings, "K8S004")
	if len(priv) != 2 {
		t.Fatalf("want K8S004 for init and ephemeral containers, got %+v", priv)
	}
	if !strings.Contains(priv[0].File, "init container: setup") || !strings.Contains(priv[1].File, "ephemeral container: shell") {
		t.Errorf("unexpected refs: %q, %q", priv[0].File, priv[1].File)
	}
	if got := findByID(findings, "K8S007"); len(got) != 0 {
		t.Errorf("K8S007 must not fire for init or ephemeral containers, got %+v", got)
	}
}

func TestContainerRunAsUserZeroOverridesPod(t *testing.T) {
	yes, uid, root := true, int64(1000), int64(0)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "p"},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: &yes, RunAsUser: &uid},
			Containers:      []corev1.Container{{Name: "c", SecurityContext: &corev1.SecurityContext{RunAsUser: &root}}},
		},
	}
	if got := findByID(checkPodSecurity(pod), "K8S005"); len(got) != 1 {
		t.Fatalf("k8s-live: want K8S005, got %+v", got)
	}

	local, err := checkManifestDoc([]byte(`apiVersion: v1
kind: Pod
metadata:
  name: p
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
  containers:
    - name: c
      image: nginx:1.27
      securityContext:
        runAsUser: 0
`), "pod.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := findByID(local, "K8S005"); len(got) != 1 {
		t.Fatalf("k8s-local: want K8S005, got %+v", got)
	}
}

func TestK8sLiveNamespaceFilter(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`))
	}))
	defer srv.Close()
	cs, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	for _, ns := range []string{"a", ""} {
		if _, err := auditPods(context.Background(), cs, ns); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"/api/v1/namespaces/a/pods", "/api/v1/pods"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("requested %v, want %v", paths, want)
	}
}
