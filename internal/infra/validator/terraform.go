// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"annave.tech/cli/internal/infra/domain"
)

// tfPlan is the minimal structure of `terraform show -json` output.
type tfPlan struct {
	FormatVersion    string            `json:"format_version"`
	TerraformVersion string            `json:"terraform_version"`
	ResourceChanges  []tfResourceChange `json:"resource_changes"`
}

type tfResourceChange struct {
	Address string   `json:"address"`
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Change  tfChange `json:"change"`
}

type tfChange struct {
	Actions []string `json:"actions"`
}

// destructiveDataRisk maps resource types whose deletion destroys irreplaceable data.
var destructiveDataRisk = map[string]string{
	"aws_db_instance":               "RDS database — deletion destroys all data permanently",
	"aws_rds_cluster":               "RDS Aurora cluster — deletion destroys all data permanently",
	"aws_dynamodb_table":            "DynamoDB table — deletion destroys all items permanently",
	"aws_s3_bucket":                 "S3 bucket — may contain critical data",
	"aws_elasticsearch_domain":      "Elasticsearch domain and all indices",
	"aws_opensearch_domain":         "OpenSearch domain and all indices",
	"google_sql_database_instance":  "Cloud SQL instance — deletion destroys all data",
	"google_bigtable_instance":      "Bigtable instance and all tables",
	"azurerm_sql_server":            "Azure SQL Server and all databases",
	"azurerm_cosmosdb_account":      "CosmosDB account and all data",
}

// iamRisk maps IAM-related resource types that affect security posture.
var iamRisk = map[string]string{
	"aws_iam_role":                  "IAM role",
	"aws_iam_policy":                "IAM policy",
	"aws_iam_user":                  "IAM user",
	"aws_iam_role_policy":           "IAM inline role policy",
	"aws_iam_role_policy_attachment": "IAM role policy attachment",
	"google_project_iam_binding":    "GCP project IAM binding",
	"google_project_iam_member":     "GCP project IAM member",
	"azurerm_role_assignment":       "Azure RBAC role assignment",
}

// networkRisk maps network security resource types.
var networkRisk = map[string]string{
	"aws_security_group":                "Security group",
	"aws_security_group_rule":           "Security group rule",
	"aws_network_acl":                   "Network ACL",
	"google_compute_firewall":           "GCP compute firewall rule",
	"azurerm_network_security_group":    "Azure network security group",
	"azurerm_network_security_rule":     "Azure NSG rule",
}

func validateTerraform(target string) (*domain.ValidationResult, error) {
	result := &domain.ValidationResult{
		Target:      target,
		ValidatedAt: time.Now(),
	}

	data, err := loadTerraformJSON(target)
	if err != nil {
		return nil, err
	}

	var plan tfPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("not a valid terraform plan JSON: %w", err)
	}
	if len(plan.ResourceChanges) == 0 && plan.FormatVersion == "" {
		return nil, fmt.Errorf("file does not look like a terraform show -json output")
	}

	for _, rc := range plan.ResourceChanges {
		issues := classifyChange(rc)
		result.Issues = append(result.Issues, issues...)
	}

	result.Passed = !hasHigherThan(result.Issues, domain.SeverityMedium)
	return result, nil
}

func loadTerraformJSON(target string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(target))

	// Already a JSON file — parse directly.
	if ext == ".json" {
		data, err := os.ReadFile(target)
		if err != nil {
			return nil, fmt.Errorf("cannot read plan file: %w", err)
		}
		return data, nil
	}

	// Binary .tfplan — shell out to terraform show -json.
	tfBin, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("terraform binary not found in PATH — run `terraform show -json plan.tfplan > plan.json` and pass the JSON file instead")
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(tfBin, "show", "-json", abs)
	cmd.Dir = filepath.Dir(abs)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform show -json failed: %w", err)
	}
	return out, nil
}

func classifyChange(rc tfResourceChange) []domain.ValidationIssue {
	actions := rc.Change.Actions
	if len(actions) == 0 || (len(actions) == 1 && (actions[0] == "no-op" || actions[0] == "read")) {
		return nil
	}

	isDelete := containsAction(actions, "delete")
	isReplace := len(actions) > 1 && containsAction(actions, "create") && containsAction(actions, "delete")
	isCreate := len(actions) == 1 && actions[0] == "create"
	isUpdate := len(actions) == 1 && actions[0] == "update"

	var issues []domain.ValidationIssue

	// Data destruction risk.
	if desc, ok := destructiveDataRisk[rc.Type]; ok {
		if isDelete {
			issues = append(issues, domain.ValidationIssue{
				Severity: domain.SeverityCritical,
				Rule:     "TF001",
				Message:  fmt.Sprintf("Destructive delete: %s — %s", rc.Address, desc),
				Resource: rc.Address,
			})
		} else if isReplace {
			issues = append(issues, domain.ValidationIssue{
				Severity: domain.SeverityCritical,
				Rule:     "TF002",
				Message:  fmt.Sprintf("Destructive replace: %s — %s (will be destroyed and recreated)", rc.Address, desc),
				Resource: rc.Address,
			})
		}
	}

	// IAM risk.
	if desc, ok := iamRisk[rc.Type]; ok {
		sev := domain.SeverityHigh
		if isDelete || isReplace {
			sev = domain.SeverityCritical
		} else if isCreate {
			sev = domain.SeverityMedium
		}
		_ = isUpdate
		issues = append(issues, domain.ValidationIssue{
			Severity: sev,
			Rule:     "TF003",
			Message:  fmt.Sprintf("IAM change: %s is a %s", rc.Address, desc),
			Resource: rc.Address,
		})
	}

	// Network security risk.
	if desc, ok := networkRisk[rc.Type]; ok {
		sev := domain.SeverityHigh
		if isDelete || isReplace {
			sev = domain.SeverityCritical
		} else if isCreate {
			sev = domain.SeverityMedium
		}
		issues = append(issues, domain.ValidationIssue{
			Severity: sev,
			Rule:     "TF004",
			Message:  fmt.Sprintf("Network security change: %s is a %s", rc.Address, desc),
			Resource: rc.Address,
		})
	}

	// Generic destructive change (not data/IAM/network).
	if (isDelete || isReplace) && len(issues) == 0 {
		issues = append(issues, domain.ValidationIssue{
			Severity: domain.SeverityHigh,
			Rule:     "TF005",
			Message:  fmt.Sprintf("Destructive change (%s): %s", strings.Join(actions, "+"), rc.Address),
			Resource: rc.Address,
		})
	}

	return issues
}

func containsAction(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}
