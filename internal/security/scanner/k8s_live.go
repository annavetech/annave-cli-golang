// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"annave.tech/cli/internal/security/domain"
	"annave.tech/cli/internal/security/port"
)

// scanK8sLive queries the cluster API for pods and checks each for security misconfigurations.
func scanK8sLive(ctx context.Context, opts port.AuditOptions) ([]domain.Finding, error) {
	cs, err := newSecurityClient(opts)
	if err != nil {
		return nil, err
	}

	pods, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	var findings []domain.Finding
	for _, pod := range pods.Items {
		findings = append(findings, checkPodSecurity(pod)...)
	}
	return findings, nil
}

// checkPodSecurity inspects a pod spec and returns a finding for each security violation.
func checkPodSecurity(pod corev1.Pod) []domain.Finding {
	ref := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	var findings []domain.Finding

	spec := pod.Spec

	// hostNetwork / hostPID
	if spec.HostNetwork {
		findings = append(findings, domain.Finding{
			ID: "K8S001", Title: "Pod uses hostNetwork",
			Severity: domain.SeverityHigh, File: ref,
			Detail:      "Pod shares the node's network namespace — can access all node network interfaces.",
			Remediation: "Remove hostNetwork: true unless absolutely required. Use a Service instead.",
		})
	}
	if spec.HostPID {
		findings = append(findings, domain.Finding{
			ID: "K8S002", Title: "Pod uses hostPID",
			Severity: domain.SeverityHigh, File: ref,
			Detail:      "Pod can see all processes on the node.",
			Remediation: "Remove hostPID: true unless absolutely required.",
		})
	}

	// hostPath volumes
	for _, vol := range spec.Volumes {
		if vol.HostPath != nil {
			findings = append(findings, domain.Finding{
				ID: "K8S003", Title: "hostPath volume mount",
				Severity: domain.SeverityMedium, File: ref,
				Detail:      fmt.Sprintf("Volume %q mounts host path %q", vol.Name, vol.HostPath.Path),
				Remediation: "Use PersistentVolumeClaims instead of hostPath. hostPath mounts expose node filesystem.",
			})
		}
	}

	// per-container checks
	for _, c := range spec.Containers {
		cref := fmt.Sprintf("%s (container: %s)", ref, c.Name)

		// privileged
		if c.SecurityContext != nil && c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
			findings = append(findings, domain.Finding{
				ID: "K8S004", Title: "Privileged container",
				Severity: domain.SeverityCritical, File: cref,
				Detail:      "Container runs with full host privileges — equivalent to root on the node.",
				Remediation: "Remove privileged: true. Use specific Linux capabilities instead.",
			})
		}

		// running as root
		if isRunningAsRoot(spec.SecurityContext, c.SecurityContext) {
			findings = append(findings, domain.Finding{
				ID: "K8S005", Title: "Container may run as root",
				Severity: domain.SeverityHigh, File: cref,
				Detail:      "No runAsNonRoot: true or runAsUser set — container may run as UID 0.",
				Remediation: "Set securityContext.runAsNonRoot: true and a non-zero runAsUser.",
			})
		}

		// no resource limits
		if c.Resources.Limits == nil || len(c.Resources.Limits) == 0 {
			findings = append(findings, domain.Finding{
				ID: "K8S006", Title: "No CPU/memory limits set",
				Severity: domain.SeverityMedium, File: cref,
				Detail:      "Container has no resource limits — can consume all node resources.",
				Remediation: "Set resources.limits.cpu and resources.limits.memory on every container.",
			})
		}

		// missing readiness probe
		if c.ReadinessProbe == nil {
			findings = append(findings, domain.Finding{
				ID: "K8S007", Title: "Missing readiness probe",
				Severity: domain.SeverityLow, File: cref,
				Detail:      "Container has no readinessProbe — traffic is sent before the app is ready.",
				Remediation: "Add a readinessProbe (httpGet, tcpSocket, or exec) to gate traffic correctly.",
			})
		}
	}

	return findings
}

// isRunningAsRoot returns true when neither the pod nor container security context
// prevents running as root.
func isRunningAsRoot(podSC *corev1.PodSecurityContext, cSC *corev1.SecurityContext) bool {
	// If pod-level runAsNonRoot is set and true, we're fine.
	if podSC != nil && podSC.RunAsNonRoot != nil && *podSC.RunAsNonRoot {
		return false
	}
	// If pod-level runAsUser is a known non-root UID, we're fine.
	if podSC != nil && podSC.RunAsUser != nil && *podSC.RunAsUser > 0 {
		return false
	}
	// Container-level overrides.
	if cSC != nil {
		if cSC.RunAsNonRoot != nil && *cSC.RunAsNonRoot {
			return false
		}
		if cSC.RunAsUser != nil && *cSC.RunAsUser > 0 {
			return false
		}
	}
	return true
}

// newSecurityClient builds a Kubernetes clientset from opts or the default kubeconfig resolution.
func newSecurityClient(opts port.AuditOptions) (*kubernetes.Clientset, error) {
	kubeconfigPath := opts.KubeconfigPath
	if kubeconfigPath == "" {
		if kc := os.Getenv("KUBECONFIG"); kc != "" {
			kubeconfigPath = kc
		} else {
			home, _ := os.UserHomeDir()
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath

	overrides := &clientcmd.ConfigOverrides{}
	if opts.KubeContext != "" {
		overrides.CurrentContext = opts.KubeContext
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot build k8s client config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create k8s clientset: %w", err)
	}
	return cs, nil
}
