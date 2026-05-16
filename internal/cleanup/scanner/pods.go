// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"annave.tech/cli/internal/cleanup/domain"
	"annave.tech/cli/internal/cleanup/port"
)

// scanPods lists pods in the target namespace and returns those that are idle or unhealthy.
func scanPods(ctx context.Context, cs *kubernetes.Clientset, opts port.ScanOptions) ([]domain.IdleResource, error) {
	pods, err := cs.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	var idle []domain.IdleResource
	for _, pod := range pods.Items {
		if systemNamespaces[pod.Namespace] {
			continue
		}
		if r, ok := classifyPod(pod, opts.MaxPendingAge); ok {
			idle = append(idle, r)
		}
	}
	return idle, nil
}

// classifyPod inspects pod phase and container statuses; returns false when the pod is healthy.
func classifyPod(pod corev1.Pod, maxPendingAge time.Duration) (domain.IdleResource, bool) {
	age := time.Since(pod.CreationTimestamp.Time)
	res := domain.K8sResource{
		Kind:      domain.ResourcePod,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		AgeHours:  int(age.Hours()),
		Labels:    pod.Labels,
	}

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		return domain.IdleResource{Resource: res, Reason: domain.ReasonCompleted, Detail: "job pod completed"}, true

	case corev1.PodFailed:
		msg := pod.Status.Message
		if msg == "" {
			msg = pod.Status.Reason
		}
		return domain.IdleResource{Resource: res, Reason: domain.ReasonFailed, Detail: msg}, true

	case corev1.PodPending:
		if age > maxPendingAge {
			return domain.IdleResource{
				Resource: res,
				Reason:   domain.ReasonPendingTooLong,
				Detail:   fmt.Sprintf("pending for %s", age.Round(time.Second)),
			}, true
		}

	case corev1.PodRunning:
		for _, cst := range pod.Status.ContainerStatuses {
			if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CrashLoopBackOff" {
				return domain.IdleResource{
					Resource: res,
					Reason:   domain.ReasonCrashLoopBackOff,
					Detail:   fmt.Sprintf("container %s: %d restarts", cst.Name, cst.RestartCount),
				}, true
			}
		}
	}
	return domain.IdleResource{}, false
}
