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

// scanPVCs lists PersistentVolumeClaims and flags those that are lost, long-pending, or unmounted.
func scanPVCs(ctx context.Context, cs *kubernetes.Clientset, opts port.ScanOptions) ([]domain.IdleResource, error) {
	pvcs, err := cs.CoreV1().PersistentVolumeClaims(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list PVCs: %w", err)
	}

	// Build set of PVC names currently mounted by pods.
	pods, err := cs.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods for PVC check: %w", err)
	}
	// mounted is keyed as namespace/name to match PVC references in pod volumes.
	mounted := make(map[string]bool)
	for _, pod := range pods.Items {
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				mounted[pod.Namespace+"/"+vol.PersistentVolumeClaim.ClaimName] = true
			}
		}
	}

	var idle []domain.IdleResource
	for _, pvc := range pvcs.Items {
		age := time.Since(pvc.CreationTimestamp.Time)
		res := domain.K8sResource{
			Kind:      domain.ResourcePVC,
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
			AgeHours:  int(age.Hours()),
			Labels:    pvc.Labels,
		}

		switch pvc.Status.Phase {
		case corev1.ClaimLost:
			idle = append(idle, domain.IdleResource{
				Resource: res,
				Reason:   domain.ReasonPVCNotBound,
				Detail:   "underlying storage lost",
			})

		case corev1.ClaimPending:
			if age > time.Hour {
				idle = append(idle, domain.IdleResource{
					Resource: res,
					Reason:   domain.ReasonPVCNotBound,
					Detail:   fmt.Sprintf("pending for %s — no matching PV", age.Round(time.Minute)),
				})
			}

		case corev1.ClaimBound:
			// Flag bound PVCs not mounted by any running pod (stale storage).
			if !mounted[pvc.Namespace+"/"+pvc.Name] && age > 24*time.Hour {
				idle = append(idle, domain.IdleResource{
					Resource: res,
					Reason:   domain.ReasonOrphaned,
					Detail:   "bound but not mounted by any pod",
				})
			}
		}
	}
	return idle, nil
}
