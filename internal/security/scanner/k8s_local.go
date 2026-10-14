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

	corev1 "k8s.io/api/core/v1"
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

// k8sSpecWrap holds a Pod spec (spec), a workload template (spec.template.spec),
// and a CronJob template (spec.jobTemplate.spec.template.spec) in one struct.
type k8sSpecWrap struct {
	k8sPodSpec
	Template    k8sPodTemplate `json:"template"`
	JobTemplate struct {
		Spec struct {
			Template k8sPodTemplate `json:"template"`
		} `json:"spec"`
	} `json:"jobTemplate"`
}

type k8sPodTemplate struct {
	Spec k8sPodSpec `json:"spec"`
}

type k8sPodSpec struct {
	HostNetwork     bool `json:"hostNetwork"`
	HostPID         bool `json:"hostPID"`
	SecurityContext struct {
		RunAsUser    *int64 `json:"runAsUser"`
		RunAsNonRoot *bool  `json:"runAsNonRoot"`
	} `json:"securityContext"`
	InitContainers      []k8sContainer `json:"initContainers"`
	Containers          []k8sContainer `json:"containers"`
	EphemeralContainers []k8sContainer `json:"ephemeralContainers"`
	Volumes             []k8sVol       `json:"volumes"`
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

// containerKind tells which pod spec list a container came from; the API allows different fields per list.
type containerKind int

const (
	appContainer containerKind = iota
	initContainer
	ephemeralContainer
)

// label returns the prefix used in finding references, such as "init container".
func (k containerKind) label() string {
	switch k {
	case initContainer:
		return "init container"
	case ephemeralContainer:
		return "ephemeral container"
	default:
		return "container"
	}
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
	"ReplicaSet": true, "Job": true,
}

// podSpec returns the pod spec of m for its kind.
func (m *k8sManifest) podSpec() *k8sPodSpec {
	switch {
	case m.Kind == "CronJob":
		return &m.Spec.JobTemplate.Spec.Template.Spec
	case workloadKinds[m.Kind]:
		return &m.Spec.Template.Spec
	default:
		return &m.Spec.k8sPodSpec
	}
}

// scanK8sLocal walks root for YAML files and checks each manifest for security misconfigurations.
// A file that cannot be read, such as a symlink pointing outside root, is named in a warning.
func scanK8sLocal(ctx context.Context, root string) (findings []domain.Finding, warnings []string, err error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}

	rootFS, rootDir, err := openWalkRoot(absRoot)
	if err != nil || rootFS == nil {
		return nil, nil, err
	}
	defer closeRoot(rootFS, &err)

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
		name, _ := filepath.Rel(rootDir, path)
		ff, err := checkManifestFile(rootFS, name, rel)
		if err != nil {
			warnings = append(warnings, skippedFileWarning(rel, err))
			return nil
		}
		findings = append(findings, ff...)
		return nil
	})
	return findings, warnings, err
}

// checkManifestFile splits a multi-document YAML file on --- and checks each document.
func checkManifestFile(root *os.Root, name, relPath string) ([]domain.Finding, error) {
	data, err := root.ReadFile(name)
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

	spec := m.podSpec()
	podSC := &corev1.PodSecurityContext{
		RunAsUser:    spec.SecurityContext.RunAsUser,
		RunAsNonRoot: spec.SecurityContext.RunAsNonRoot,
	}

	var findings []domain.Finding

	if spec.HostNetwork {
		findings = append(findings, domain.Finding{
			ID: "K8S001", Title: "Pod uses hostNetwork",
			Severity: domain.SeverityHigh, File: docRef,
			Detail:      "Manifest sets hostNetwork: true — shares node network namespace.",
			Remediation: "Remove hostNetwork: true unless absolutely required.",
		})
	}
	if spec.HostPID {
		findings = append(findings, domain.Finding{
			ID: "K8S002", Title: "Pod uses hostPID",
			Severity: domain.SeverityHigh, File: docRef,
			Detail:      "Manifest sets hostPID: true — can see all node processes.",
			Remediation: "Remove hostPID: true unless absolutely required.",
		})
	}
	for _, vol := range spec.Volumes {
		if vol.HostPath != nil {
			findings = append(findings, domain.Finding{
				ID: "K8S003", Title: "hostPath volume mount",
				Severity: domain.SeverityMedium, File: docRef,
				Detail:      fmt.Sprintf("Volume %q mounts host path %q", vol.Name, vol.HostPath.Path),
				Remediation: "Use PersistentVolumeClaims instead of hostPath.",
			})
		}
	}

	for _, group := range []struct {
		kind       containerKind
		containers []k8sContainer
	}{
		{initContainer, spec.InitContainers},
		{appContainer, spec.Containers},
		{ephemeralContainer, spec.EphemeralContainers},
	} {
		for _, c := range group.containers {
			findings = append(findings, checkLocalContainer(c, group.kind, docRef, podSC)...)
		}
	}

	return findings, nil
}

// checkLocalContainer applies the container rules that the Kubernetes API allows for kind.
func checkLocalContainer(c k8sContainer, kind containerKind, docRef string, podSC *corev1.PodSecurityContext) []domain.Finding {
	cref := fmt.Sprintf("%s (%s: %s)", docRef, kind.label(), c.Name)
	var findings []domain.Finding

	if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
		findings = append(findings, domain.Finding{
			ID: "K8S004", Title: "Privileged container",
			Severity: domain.SeverityCritical, File: cref,
			Detail:      "privileged: true grants full host access.",
			Remediation: "Remove privileged: true. Use specific Linux capabilities instead.",
		})
	}

	cSC := &corev1.SecurityContext{
		RunAsUser:    c.SecurityContext.RunAsUser,
		RunAsNonRoot: c.SecurityContext.RunAsNonRoot,
	}
	if isRunningAsRoot(podSC, cSC) {
		findings = append(findings, domain.Finding{
			ID: "K8S005", Title: "Container may run as root",
			Severity: domain.SeverityHigh, File: cref,
			Detail:      "No runAsNonRoot: true or non-zero runAsUser specified.",
			Remediation: "Set securityContext.runAsNonRoot: true and a non-zero runAsUser.",
		})
	}

	// Ephemeral containers cannot set resources; only app containers can have probes.
	if kind != ephemeralContainer && len(c.Resources.Limits) == 0 {
		findings = append(findings, domain.Finding{
			ID: "K8S006", Title: "No CPU/memory limits set",
			Severity: domain.SeverityMedium, File: cref,
			Detail:      "Container has no resources.limits defined.",
			Remediation: "Set resources.limits.cpu and resources.limits.memory.",
		})
	}

	if kind == appContainer && c.ReadinessProbe == nil {
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

	return findings
}
