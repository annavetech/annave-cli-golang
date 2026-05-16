// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"

	"annave.tech/cli/internal/security/domain"
)

// k8sManifest is a minimal representation of any Kubernetes resource manifest.
type k8sManifest struct {
	APIVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   k8sMeta     `json:"metadata"`
	Spec       k8sSpecWrap `json:"spec"`
}

type k8sMeta struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// k8sSpecWrap handles both Pod specs (spec.containers) and workload specs
// (spec.template.spec.containers) in one struct.
type k8sSpecWrap struct {
	HostNetwork bool           `json:"hostNetwork"`
	HostPID     bool           `json:"hostPID"`
	Containers  []k8sContainer `json:"containers"`
	Volumes     []k8sVol       `json:"volumes"`
	Template    struct {
		Spec struct {
			HostNetwork bool           `json:"hostNetwork"`
			HostPID     bool           `json:"hostPID"`
			Containers  []k8sContainer `json:"containers"`
			Volumes     []k8sVol       `json:"volumes"`
		} `json:"spec"`
	} `json:"template"`
}

type k8sContainer struct {
	Name      string `json:"name"`
	Image     string `json:"image"`
	Resources struct {
		Limits map[string]string `json:"limits"`
	} `json:"resources"`
	SecurityContext struct {
		Privileged   *bool  `json:"privileged"`
		RunAsUser    *int64 `json:"runAsUser"`
		RunAsNonRoot *bool  `json:"runAsNonRoot"`
	} `json:"securityContext"`
	ReadinessProbe *struct{} `json:"readinessProbe"`
}

type k8sVol struct {
	Name     string `json:"name"`
	HostPath *struct {
		Path string `json:"path"`
	} `json:"hostPath"`
}

// workloadKinds that have spec.template.spec.containers (vs Pod which has spec.containers).
var workloadKinds = map[string]bool{
	"Deployment": true, "StatefulSet": true, "DaemonSet": true,
	"ReplicaSet": true, "Job": true, "CronJob": true,
}

// scanK8sLocal walks root for YAML files and checks each manifest for security misconfigurations.
func scanK8sLocal(ctx context.Context, root string) ([]domain.Finding, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var findings []domain.Finding

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if skipSecretDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		rel, _ := filepath.Rel(absRoot, path)
		ff, err := checkManifestFile(path, rel)
		if err != nil {
			return nil
		}
		findings = append(findings, ff...)
		return nil
	})
	return findings, err
}

// checkManifestFile splits a multi-document YAML file on --- and checks each document.
func checkManifestFile(path, relPath string) ([]domain.Finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// YAML files can contain multiple documents separated by ---.
	var findings []domain.Finding
	for i, doc := range strings.Split(string(data), "\n---") {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		ff, err := checkManifestDoc([]byte(doc), fmt.Sprintf("%s[%d]", relPath, i))
		if err != nil {
			continue // skip malformed documents
		}
		findings = append(findings, ff...)
	}
	return findings, nil
}

// checkManifestDoc unmarshals one YAML document and applies all K8s security rules.
func checkManifestDoc(data []byte, ref string) ([]domain.Finding, error) {
	var m k8sManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Kind == "" {
		return nil, nil
	}

	name := m.Metadata.Name
	if m.Metadata.Namespace != "" {
		name = m.Metadata.Namespace + "/" + name
	}
	docRef := fmt.Sprintf("%s (%s/%s)", ref, m.Kind, name)

	// Extract the pod spec regardless of whether this is a Pod or a workload.
	var (
		hostNetwork bool
		hostPID     bool
		containers  []k8sContainer
		volumes     []k8sVol
	)
	if workloadKinds[m.Kind] {
		hostNetwork = m.Spec.Template.Spec.HostNetwork
		hostPID = m.Spec.Template.Spec.HostPID
		containers = m.Spec.Template.Spec.Containers
		volumes = m.Spec.Template.Spec.Volumes
	} else {
		hostNetwork = m.Spec.HostNetwork
		hostPID = m.Spec.HostPID
		containers = m.Spec.Containers
		volumes = m.Spec.Volumes
	}

	var findings []domain.Finding

	if hostNetwork {
		findings = append(findings, domain.Finding{
			ID: "K8S001", Title: "Pod uses hostNetwork",
			Severity: domain.SeverityHigh, File: docRef,
			Detail:      "Manifest sets hostNetwork: true — shares node network namespace.",
			Remediation: "Remove hostNetwork: true unless absolutely required.",
		})
	}
	if hostPID {
		findings = append(findings, domain.Finding{
			ID: "K8S002", Title: "Pod uses hostPID",
			Severity: domain.SeverityHigh, File: docRef,
			Detail:      "Manifest sets hostPID: true — can see all node processes.",
			Remediation: "Remove hostPID: true unless absolutely required.",
		})
	}
	for _, vol := range volumes {
		if vol.HostPath != nil {
			findings = append(findings, domain.Finding{
				ID: "K8S003", Title: "hostPath volume mount",
				Severity: domain.SeverityMedium, File: docRef,
				Detail:      fmt.Sprintf("Volume %q mounts host path %q", vol.Name, vol.HostPath.Path),
				Remediation: "Use PersistentVolumeClaims instead of hostPath.",
			})
		}
	}

	for _, c := range containers {
		cref := fmt.Sprintf("%s (container: %s)", docRef, c.Name)

		if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
			findings = append(findings, domain.Finding{
				ID: "K8S004", Title: "Privileged container",
				Severity: domain.SeverityCritical, File: cref,
				Detail:      "privileged: true grants full host access.",
				Remediation: "Remove privileged: true. Use specific Linux capabilities instead.",
			})
		}

		runAsRoot := true
		if c.SecurityContext.RunAsNonRoot != nil && *c.SecurityContext.RunAsNonRoot {
			runAsRoot = false
		}
		if c.SecurityContext.RunAsUser != nil && *c.SecurityContext.RunAsUser > 0 {
			runAsRoot = false
		}
		if runAsRoot {
			findings = append(findings, domain.Finding{
				ID: "K8S005", Title: "Container may run as root",
				Severity: domain.SeverityHigh, File: cref,
				Detail:      "No runAsNonRoot: true or non-zero runAsUser specified.",
				Remediation: "Set securityContext.runAsNonRoot: true and a non-zero runAsUser.",
			})
		}

		if len(c.Resources.Limits) == 0 {
			findings = append(findings, domain.Finding{
				ID: "K8S006", Title: "No CPU/memory limits set",
				Severity: domain.SeverityMedium, File: cref,
				Detail:      "Container has no resources.limits defined.",
				Remediation: "Set resources.limits.cpu and resources.limits.memory.",
			})
		}

		if c.ReadinessProbe == nil {
			findings = append(findings, domain.Finding{
				ID: "K8S007", Title: "Missing readiness probe",
				Severity: domain.SeverityLow, File: cref,
				Detail:      "No readinessProbe defined.",
				Remediation: "Add a readinessProbe to gate traffic until the app is ready.",
			})
		}

		// image:latest
		if strings.HasSuffix(c.Image, ":latest") || !strings.Contains(c.Image, ":") {
			findings = append(findings, domain.Finding{
				ID: "K8S008", Title: "Unpinned image tag",
				Severity: domain.SeverityLow, File: cref,
				Detail:      fmt.Sprintf("Image %q uses :latest or has no tag — non-deterministic deploys.", c.Image),
				Remediation: "Pin images to a specific digest or version tag (e.g. image:1.2.3).",
			})
		}
	}

	return findings, nil
}
