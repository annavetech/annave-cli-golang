// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"annave.tech/cli/internal/cleanup/domain"
	"annave.tech/cli/internal/cleanup/port"
	"annave.tech/cli/internal/shared/config"
)

// systemNamespaces are always skipped when scanning for idle resources.
var systemNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

// K8sScanner implements port.Scanner.
type K8sScanner struct{}

func New() *K8sScanner { return &K8sScanner{} }

func newClientset(opts port.ScanOptions) (*kubernetes.Clientset, string, error) {
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

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, "", fmt.Errorf("cannot load kubeconfig: %w", err)
	}
	currentContext := rawConfig.CurrentContext
	if opts.KubeContext != "" {
		currentContext = opts.KubeContext
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("cannot build k8s client config: %w", err)
	}

	cs, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, "", fmt.Errorf("cannot create k8s clientset: %w", err)
	}

	return cs, currentContext, nil
}

func (s *K8sScanner) Scan(ctx context.Context, opts port.ScanOptions) (*domain.CleanupPlan, error) {
	cs, currentContext, err := newClientset(opts)
	if err != nil {
		return nil, err
	}

	if opts.MaxPendingAge == 0 {
		opts.MaxPendingAge = 10 * time.Minute
	}

	plan := &domain.CleanupPlan{
		Context:   currentContext,
		Namespace: opts.Namespace,
		ScannedAt: time.Now(),
		DryRun:    opts.DryRun,
	}

	type scannerFn func(context.Context, *kubernetes.Clientset, port.ScanOptions) ([]domain.IdleResource, error)
	scanners := []scannerFn{scanPods, scanPVCs, scanConfigMaps, scanNamespaces}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, fn := range scanners {
		wg.Add(1)
		go func(f scannerFn) {
			defer wg.Done()
			resources, err := f(ctx, cs, opts)
			if err != nil {
				return // best-effort: skip failed scanner
			}
			mu.Lock()
			plan.Resources = append(plan.Resources, resources...)
			mu.Unlock()
		}(fn)
	}
	wg.Wait()

	if max := config.App.Limits.Cleanup.MaxResources; len(plan.Resources) > max {
		plan.Resources = plan.Resources[:max]
	}

	return plan, nil
}
