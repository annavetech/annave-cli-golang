// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package validator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"annave.tech/cli/internal/infra/domain"
)

// deprecatedAPIVersions maps old apiVersions to their replacement.
var deprecatedAPIVersions = map[string]string{
	"apps/v1beta1":                  "apps/v1",
	"apps/v1beta2":                  "apps/v1",
	"extensions/v1beta1":            "apps/v1 or networking.k8s.io/v1",
	"networking.k8s.io/v1beta1":     "networking.k8s.io/v1",
	"rbac.authorization.k8s.io/v1beta1": "rbac.authorization.k8s.io/v1",
	"apiextensions.k8s.io/v1beta1":  "apiextensions.k8s.io/v1",
	"policy/v1beta1":                "policy/v1",
}

// infraManifest is a minimal K8s manifest structure for infra checks.
// Overlaps intentionally with security scanner's equivalent — each module owns its own types.
type infraManifest struct {
	APIVersion string    `json:"apiVersion"`
	Kind       string    `json:"kind"`
	Metadata   infraMeta `json:"metadata"`
	Spec       infraSpec `json:"spec"`
}

type infraMeta struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type infraSpec struct {
	Replicas int              `json:"replicas"`
	// Pod spec (for bare Pods)
	Containers []infraContainer `json:"containers"`
	// Workload wrapper
	Template struct {
		Spec struct {
			Containers []infraContainer `json:"containers"`
		} `json:"spec"`
	} `json:"template"`
}

type infraContainer struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Resources struct {
		Requests map[string]string `json:"requests"`
		Limits   map[string]string `json:"limits"`
	} `json:"resources"`
	LivenessProbe  *struct{} `json:"livenessProbe"`
	ReadinessProbe *struct{} `json:"readinessProbe"`
}

var skipInfraDirs = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	".git": true, ".angular": true,
}

// validateK8sYAML checks a YAML file or directory for common Kubernetes configuration problems.
func validateK8sYAML(target string) (*domain.ValidationResult, error) {
	result := &domain.ValidationResult{
		Target:      target,
		ValidatedAt: time.Now(),
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", target, err)
	}

	if info.IsDir() {
		issues, err := walkAndCheckManifests(target)
		if err != nil {
			return nil, err
		}
		result.Issues = issues
	} else {
		rel := filepath.Base(target)
		issues, err := checkInfraManifestFile(target, rel)
		if err != nil {
			return nil, err
		}
		result.Issues = issues
	}

	result.Passed = !hasHigherThan(result.Issues, domain.SeverityMedium)
	return result, nil
}

// walkAndCheckManifests walks a directory tree and validates all YAML files found.
func walkAndCheckManifests(root string) ([]domain.ValidationIssue, error) {
	var issues []domain.ValidationIssue
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipInfraDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		ff, err := checkInfraManifestFile(path, rel)
		if err != nil {
			return nil
		}
		issues = append(issues, ff...)
		return nil
	})
	return issues, err
}

// checkInfraManifestFile splits multi-document YAML on --- and validates each document.
func checkInfraManifestFile(path, relPath string) ([]domain.ValidationIssue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var issues []domain.ValidationIssue
	for i, doc := range strings.Split(string(data), "\n---") {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		ref := fmt.Sprintf("%s[%d]", relPath, i)
		ff, err := checkInfraDoc([]byte(doc), ref)
		if err != nil {
			continue
		}
		issues = append(issues, ff...)
	}
	return issues, nil
}

// checkInfraDoc applies infra validation rules to a single YAML document.
func checkInfraDoc(data []byte, ref string) ([]domain.ValidationIssue, error) {
	var m infraManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Kind == "" && m.APIVersion == "" {
		return nil, nil
	}

	name := m.Metadata.Name
	if m.Metadata.Namespace != "" {
		name = m.Metadata.Namespace + "/" + name
	}
	docRef := fmt.Sprintf("%s (%s/%s)", ref, m.Kind, name)

	var issues []domain.ValidationIssue

	// Missing metadata.name.
	if m.Metadata.Name == "" {
		issues = append(issues, domain.ValidationIssue{
			Severity: domain.SeverityHigh,
			Rule:     "K8S101",
			Message:  fmt.Sprintf("%s: missing metadata.name", docRef),
			Resource: docRef,
		})
	}

	// Deprecated apiVersion.
	if replacement, ok := deprecatedAPIVersions[m.APIVersion]; ok {
		issues = append(issues, domain.ValidationIssue{
			Severity: domain.SeverityHigh,
			Rule:     "K8S102",
			Message:  fmt.Sprintf("%s: apiVersion %q is deprecated — use %s", docRef, m.APIVersion, replacement),
			Resource: docRef,
		})
	}

	// Single replica on a Deployment (potential SPOF).
	if m.Kind == "Deployment" && m.Spec.Replicas == 1 {
		issues = append(issues, domain.ValidationIssue{
			Severity: domain.SeverityLow,
			Rule:     "K8S103",
			Message:  fmt.Sprintf("%s: replicas: 1 — single replica is a single point of failure", docRef),
			Resource: docRef,
		})
	}

	// Per-container checks.
	containers := m.Spec.Containers
	if workloadKinds[m.Kind] {
		containers = m.Spec.Template.Spec.Containers
	}
	for _, c := range containers {
		cref := fmt.Sprintf("%s (container: %s)", docRef, c.Name)

		// Unpinned image tag.
		if strings.HasSuffix(c.Image, ":latest") || (!strings.Contains(c.Image, ":") && c.Image != "") {
			issues = append(issues, domain.ValidationIssue{
				Severity: domain.SeverityMedium,
				Rule:     "K8S104",
				Message:  fmt.Sprintf("%s: image %q uses :latest or no tag — non-deterministic deployments", cref, c.Image),
				Resource: cref,
			})
		}

		// No resource requests.
		if len(c.Resources.Requests) == 0 {
			issues = append(issues, domain.ValidationIssue{
				Severity: domain.SeverityMedium,
				Rule:     "K8S105",
				Message:  fmt.Sprintf("%s: no resource requests set — scheduler cannot make optimal placement decisions", cref),
				Resource: cref,
			})
		}

		// No liveness probe.
		if c.LivenessProbe == nil {
			issues = append(issues, domain.ValidationIssue{
				Severity: domain.SeverityLow,
				Rule:     "K8S106",
				Message:  fmt.Sprintf("%s: no livenessProbe — unhealthy pods will not be restarted automatically", cref),
				Resource: cref,
			})
		}
	}

	return issues, nil
}

// workloadKinds maps kinds that use spec.template.spec.containers instead of spec.containers.
var workloadKinds = map[string]bool{
	"Deployment": true, "StatefulSet": true, "DaemonSet": true,
	"ReplicaSet": true, "Job": true,
}
