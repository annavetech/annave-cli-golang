// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"annave.tech/cli/internal/cleanup/domain"
	"annave.tech/cli/internal/cleanup/port"
)

// orphanAge is the minimum age before a ConfigMap is considered a candidate.
const orphanAge = 30 * 24 * time.Hour

// systemCMNames are well-known system ConfigMap names, always skipped.
var systemCMNames = map[string]bool{
	"kube-root-ca.crt": true,
}

func scanConfigMaps(ctx context.Context, cs *kubernetes.Clientset, opts port.ScanOptions) ([]domain.IdleResource, error) {
	cms, err := cs.CoreV1().ConfigMaps(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list configmaps: %w", err)
	}

	var idle []domain.IdleResource
	for _, cm := range cms.Items {
		if systemNamespaces[cm.Namespace] {
			continue
		}
		if systemCMNames[cm.Name] {
			continue
		}
		if len(cm.OwnerReferences) > 0 {
			continue // managed by a controller, not orphaned
		}
		age := time.Since(cm.CreationTimestamp.Time)
		if age < orphanAge {
			continue
		}
		idle = append(idle, domain.IdleResource{
			Resource: domain.K8sResource{
				Kind:      domain.ResourceConfigMap,
				Name:      cm.Name,
				Namespace: cm.Namespace,
				AgeHours:  int(age.Hours()),
				Labels:    cm.Labels,
			},
			Reason: domain.ReasonOrphaned,
			Detail: fmt.Sprintf("no owner references, %d days old", int(age.Hours()/24)),
		})
	}
	return idle, nil
}
