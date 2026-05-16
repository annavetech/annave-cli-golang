// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// ResourceType identifies the Kubernetes resource kind.
type ResourceType string

const (
	ResourcePod       ResourceType = "Pod"
	ResourcePVC       ResourceType = "PersistentVolumeClaim"
	ResourceConfigMap ResourceType = "ConfigMap"
	ResourceNamespace ResourceType = "Namespace"
)

// IdleReason is the classifier assigned to an idle resource.
type IdleReason string

const (
	ReasonCrashLoopBackOff IdleReason = "CrashLoopBackOff"
	ReasonCompleted        IdleReason = "Completed"
	ReasonFailed           IdleReason = "Failed"
	ReasonPendingTooLong   IdleReason = "PendingTooLong"
	ReasonPVCNotBound      IdleReason = "NotBound"
	ReasonOrphaned         IdleReason = "Orphaned"
	ReasonEmpty            IdleReason = "Empty"
)

// K8sResource identifies a Kubernetes object by kind, name, and namespace.
type K8sResource struct {
	Kind      ResourceType      `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	AgeHours  int               `json:"age_hours"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// IdleResource pairs a K8sResource with the reason it was flagged as idle.
type IdleResource struct {
	Resource K8sResource `json:"resource"`
	Reason   IdleReason  `json:"reason"`
	Detail   string      `json:"detail,omitempty"`
}

// CleanupPlan is the top-level result returned by the Scanner.
type CleanupPlan struct {
	Context   string         `json:"context"`
	Namespace string         `json:"namespace,omitempty"`
	Resources []IdleResource `json:"resources"`
	ScannedAt time.Time      `json:"scanned_at"`
	DryRun    bool           `json:"dry_run"`
}
