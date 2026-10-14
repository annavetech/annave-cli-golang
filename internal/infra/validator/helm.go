// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"annave.tech/cli/internal/infra/domain"
)

// helmRelease mirrors the JSON output of `helm list -o json`.
type helmRelease struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Revision   string `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version"`
}

// unhealthyStatuses are helm release statuses that indicate a problem.
var unhealthyStatuses = map[string]string{
	"failed":           "release failed — last deployment did not complete successfully",
	"pending-install":  "release stuck in pending-install",
	"pending-upgrade":  "release stuck in pending-upgrade",
	"pending-rollback": "release stuck in pending-rollback",
	"uninstalling":     "release is being uninstalled",
	"unknown":          "release is in an unknown state",
}

// validateHelm inspects deployed Helm releases (empty target) or lints a chart directory.
func validateHelm(ctx context.Context, run commandRunner, target string) (*domain.ValidationResult, error) {
	helmBin, err := run.LookPath("helm")
	if err != nil {
		return nil, fmt.Errorf("helm binary not found in PATH — install helm to use infra validate with Helm")
	}

	result := &domain.ValidationResult{
		Target:      target,
		ValidatedAt: time.Now(),
	}

	if target == "" || target == "." {
		// No target → inspect deployed releases in the cluster.
		issues, err := listReleases(ctx, run, helmBin)
		if err != nil {
			return nil, err
		}
		result.Issues = issues
	} else {
		// Target is a chart directory → lint it.
		abs, err := filepath.Abs(target)
		if err != nil {
			return nil, err
		}
		issues, err := lintChart(ctx, run, helmBin, abs)
		if err != nil {
			return nil, err
		}
		result.Issues = issues
	}

	result.Passed = !hasHigherThan(result.Issues, domain.SeverityMedium)
	return result, nil
}

// listReleases runs helm list -A --all and flags unhealthy or superseded releases.
// Without --all, helm lists only deployed and failed releases.
func listReleases(ctx context.Context, run commandRunner, helmBin string) ([]domain.ValidationIssue, error) {
	out, err := run.Output(ctx, "", helmBin, "list", "-A", "-o", "json", "--all")
	if err != nil {
		return nil, fmt.Errorf("helm list failed: %w", err)
	}

	var releases []helmRelease
	if err := json.Unmarshal(out, &releases); err != nil {
		return nil, fmt.Errorf("cannot parse helm list output: %w", err)
	}

	var issues []domain.ValidationIssue
	for _, r := range releases {
		ref := fmt.Sprintf("%s/%s", r.Namespace, r.Name)

		if msg, bad := unhealthyStatuses[strings.ToLower(r.Status)]; bad {
			issues = append(issues, domain.ValidationIssue{
				Severity: domain.SeverityHigh,
				Rule:     "HELM001",
				Message:  fmt.Sprintf("Release %s: %s", ref, msg),
				Resource: ref,
			})
		}

		// Warn on superseded (old release left behind).
		if strings.ToLower(r.Status) == "superseded" {
			issues = append(issues, domain.ValidationIssue{
				Severity: domain.SeverityLow,
				Rule:     "HELM002",
				Message:  fmt.Sprintf("Release %s is superseded — consider pruning old revisions", ref),
				Resource: ref,
			})
		}
	}
	return issues, nil
}

// lintChart runs helm lint against a chart directory and maps its output to ValidationIssues.
func lintChart(ctx context.Context, run commandRunner, helmBin, chartPath string) ([]domain.ValidationIssue, error) {
	// helm lint exits non-zero on errors; capture combined output regardless.
	out, runErr := run.CombinedOutput(ctx, helmBin, "lint", chartPath)

	var issues []domain.ValidationIssue
	lintErrors := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var sev, rule string
		switch {
		case strings.HasPrefix(line, "[ERROR]"):
			sev = domain.SeverityHigh
			rule = "HELM003"
			line = strings.TrimPrefix(line, "[ERROR]")
			lintErrors++
		case strings.HasPrefix(line, "[WARNING]"):
			sev = domain.SeverityMedium
			rule = "HELM004"
			line = strings.TrimPrefix(line, "[WARNING]")
		case strings.HasPrefix(line, "[INFO]"):
			sev = domain.SeverityInfo
			rule = "HELM005"
			line = strings.TrimPrefix(line, "[INFO]")
		default:
			continue
		}

		issues = append(issues, domain.ValidationIssue{
			Severity: sev,
			Rule:     rule,
			Message:  strings.TrimSpace(line),
			Resource: chartPath,
		})
	}
	// A failure without an [ERROR] line (such as a missing chart) would otherwise pass.
	if runErr != nil && lintErrors == 0 {
		return nil, fmt.Errorf("helm lint failed: %w: %s", runErr, strings.TrimSpace(string(out)))
	}
	return issues, nil
}
