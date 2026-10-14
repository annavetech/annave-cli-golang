// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
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

// skipNamespaces are never flagged as empty — they always exist for a reason.
var skipNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
	"default":         true,
}

// scanNamespaces returns empty namespaces, plus one error per namespace that could not be checked.
func scanNamespaces(ctx context.Context, cs *kubernetes.Clientset, opts port.ScanOptions) ([]domain.IdleResource, []error, error) {
	nsList, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("list namespaces: %w", err)
	}

	var idle []domain.IdleResource
	var nsErrs []error
	for _, nsObj := range nsList.Items {
		if opts.Namespace != "" && nsObj.Name != opts.Namespace {
			continue
		}
		if skipNamespaces[nsObj.Name] {
			continue
		}
		age := time.Since(nsObj.CreationTimestamp.Time)
		if age < 24*time.Hour {
			continue // too recently created to flag
		}

		empty, detail, err := isNamespaceEmpty(ctx, cs, nsObj.Name)
		if err != nil {
			nsErrs = append(nsErrs, fmt.Errorf("namespace %s not checked: %w", nsObj.Name, err))
			continue
		}
		if !empty {
			continue
		}

		idle = append(idle, domain.IdleResource{
			Resource: domain.K8sResource{
				Kind:     domain.ResourceNamespace,
				Name:     nsObj.Name,
				AgeHours: int(age.Hours()),
			},
			Reason: domain.ReasonEmpty,
			Detail: detail,
		})
	}
	return idle, nsErrs, nil
}

// isNamespaceEmpty returns true when a namespace has no workloads or services.
// Uses Limit:1 queries so it exits early.
func isNamespaceEmpty(ctx context.Context, cs *kubernetes.Clientset, ns string) (bool, string, error) {
	one := metav1.ListOptions{Limit: 1}

	pods, err := cs.CoreV1().Pods(ns).List(ctx, one)
	if err != nil {
		return false, "", err
	}
	if len(pods.Items) > 0 {
		return false, "", nil
	}

	deps, err := cs.AppsV1().Deployments(ns).List(ctx, one)
	if err != nil {
		return false, "", err
	}
	if len(deps.Items) > 0 {
		return false, "", nil
	}

	sts, err := cs.AppsV1().StatefulSets(ns).List(ctx, one)
	if err != nil {
		return false, "", err
	}
	if len(sts.Items) > 0 {
		return false, "", nil
	}

	svcs, err := cs.CoreV1().Services(ns).List(ctx, one)
	if err != nil {
		return false, "", err
	}
	if len(svcs.Items) > 0 {
		return false, "", nil
	}

	return true, fmt.Sprintf("no pods, deployments, statefulsets, or services in %s", ns), nil
}
