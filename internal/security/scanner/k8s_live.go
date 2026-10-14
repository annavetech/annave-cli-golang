// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
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
	return auditPods(ctx, cs, opts.Namespace)
}

// auditPods checks every pod in namespace, or in all namespaces when namespace is empty.
func auditPods(ctx context.Context, cs kubernetes.Interface, namespace string) ([]domain.Finding, error) {
	pods, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
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

	for i := range spec.InitContainers {
		findings = append(findings, checkLiveContainer(&spec.InitContainers[i], initContainer, ref, spec.SecurityContext)...)
	}
	for i := range spec.Containers {
		findings = append(findings, checkLiveContainer(&spec.Containers[i], appContainer, ref, spec.SecurityContext)...)
	}
	for i := range spec.EphemeralContainers {
		// EphemeralContainerCommon has the same fields as Container.
		c := (*corev1.Container)(&spec.EphemeralContainers[i].EphemeralContainerCommon)
		findings = append(findings, checkLiveContainer(c, ephemeralContainer, ref, spec.SecurityContext)...)
	}

	return findings
}

// checkLiveContainer applies the container rules that the Kubernetes API allows for kind.
func checkLiveContainer(c *corev1.Container, kind containerKind, ref string, podSC *corev1.PodSecurityContext) []domain.Finding {
	cref := fmt.Sprintf("%s (%s: %s)", ref, kind.label(), c.Name)
	var findings []domain.Finding

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
	if isRunningAsRoot(podSC, c.SecurityContext) {
		findings = append(findings, domain.Finding{
			ID: "K8S005", Title: "Container may run as root",
			Severity: domain.SeverityHigh, File: cref,
			Detail:      "No runAsNonRoot: true or runAsUser set — container may run as UID 0.",
			Remediation: "Set securityContext.runAsNonRoot: true and a non-zero runAsUser.",
		})
	}

	// no resource limits; ephemeral containers cannot set resources
	if kind != ephemeralContainer && len(c.Resources.Limits) == 0 {
		findings = append(findings, domain.Finding{
			ID: "K8S006", Title: "No CPU/memory limits set",
			Severity: domain.SeverityMedium, File: cref,
			Detail:      "Container has no resource limits — can consume all node resources.",
			Remediation: "Set resources.limits.cpu and resources.limits.memory on every container.",
		})
	}

	// missing readiness probe; only app containers can have one
	if kind == appContainer && c.ReadinessProbe == nil {
		findings = append(findings, domain.Finding{
			ID: "K8S007", Title: "Missing readiness probe",
			Severity: domain.SeverityLow, File: cref,
			Detail:      "Container has no readinessProbe — traffic is sent before the app is ready.",
			Remediation: "Add a readinessProbe (httpGet, tcpSocket, or exec) to gate traffic correctly.",
		})
	}

	return findings
}

// isRunningAsRoot reports whether a container may run as UID 0. Container settings override
// pod settings; an explicit UID decides, and runAsNonRoot applies only when no UID is set.
func isRunningAsRoot(podSC *corev1.PodSecurityContext, cSC *corev1.SecurityContext) bool {
	var nonRoot *bool
	var uid *int64
	if podSC != nil {
		nonRoot, uid = podSC.RunAsNonRoot, podSC.RunAsUser
	}
	if cSC != nil {
		if cSC.RunAsNonRoot != nil {
			nonRoot = cSC.RunAsNonRoot
		}
		if cSC.RunAsUser != nil {
			uid = cSC.RunAsUser
		}
	}
	if uid != nil {
		return *uid == 0
	}
	return nonRoot == nil || !*nonRoot
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
