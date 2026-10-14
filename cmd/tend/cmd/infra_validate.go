// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/annavetech/tend/internal/infra/domain"
	"github.com/annavetech/tend/internal/infra/port"
	"github.com/annavetech/tend/internal/infra/validator"
	"github.com/annavetech/tend/internal/shared/output"
)

var (
	infraValidateType   string
	infraValidateFormat string
)

var infraValidateCmd = &cobra.Command{
	Use:   "validate [target]",
	Short: "Validate infrastructure definitions",
	Long: `Validate Terraform plans, Helm charts/releases, and Kubernetes YAML manifests.

Target auto-detection:
  *.json / *.tfplan           terraform plan analysis
  directory with Chart.yaml   helm chart lint
  *.yaml / *.yml / directory  Kubernetes manifest validation
  (no target)                 helm release status check (requires kubeconfig)

Examples:
  tend infra validate plan.json
  tend infra validate plan.tfplan
  tend infra validate ./charts/myapp
  tend infra validate ./k8s --type k8s
  tend infra validate --type helm
  tend infra validate --format table plan.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInfraValidate,
}

func init() {
	infraCmd.AddCommand(infraValidateCmd)
	infraValidateCmd.Flags().StringVar(&infraValidateType, "type", "", "validate type: terraform, helm, k8s (default: auto-detect)")
	infraValidateCmd.Flags().StringVar(&infraValidateFormat, "format", "plain", "output format: plain, json, table")
}

func runInfraValidate(cmd *cobra.Command, args []string) error {
	outFmt, err := output.ParseFormat(infraValidateFormat)
	if err != nil {
		return err
	}
	p := output.New(outFmt)

	target := ""
	if len(args) > 0 {
		target = args[0]
	}

	opts := port.ValidateOptions{Type: infraValidateType}

	v := validator.New()
	result, err := v.Validate(cmd.Context(), target, opts)
	if err != nil {
		return err
	}

	if err := printWarnings(cmd.ErrOrStderr(), result.Warnings); err != nil {
		return err
	}
	printValidationResult(p, result)
	if err := p.Err(); err != nil {
		return err
	}
	if !result.Passed {
		return errProblemsFound
	}
	return nil
}

func printValidationResult(p *output.Printer, result *domain.ValidationResult) {
	switch p.Format() {
	case output.FormatJSON:
		p.JSON(result)
	case output.FormatTable:
		printValidationTable(p, result)
	default:
		printValidationPlain(p, result)
	}
}

func printValidationPlain(p *output.Printer, result *domain.ValidationResult) {
	status := "PASSED"
	if !result.Passed {
		status = "FAILED"
	}
	p.Section(fmt.Sprintf("Infra validation: %s", result.Target))
	p.KeyVal("validated at", result.ValidatedAt.Format("2006-01-02 15:04:05"))
	p.KeyVal("result", fmt.Sprintf("%s (%d issue(s))", status, len(result.Issues)))
	p.Line("")

	if len(result.Issues) == 0 {
		p.Line("  No issues found.")
		p.Line("")
		return
	}

	// Group by severity for display order.
	order := []string{
		domain.SeverityCritical,
		domain.SeverityHigh,
		domain.SeverityMedium,
		domain.SeverityLow,
		domain.SeverityInfo,
	}
	byServ := make(map[string][]domain.ValidationIssue)
	for _, issue := range result.Issues {
		byServ[issue.Severity] = append(byServ[issue.Severity], issue)
	}

	rank := 1
	for _, sev := range order {
		for _, issue := range byServ[sev] {
			p.Linef("  [%d] %-8s  %s  %s", rank, strings.ToUpper(sev), issue.Rule, issue.Message)
			rank++
		}
	}
	p.Line("")
}

func printValidationTable(p *output.Printer, result *domain.ValidationResult) {
	status := "PASSED"
	if !result.Passed {
		status = "FAILED"
	}
	p.Section(fmt.Sprintf("Infra validation: %s", result.Target))
	p.KeyVal("result", fmt.Sprintf("%s (%d issue(s))", status, len(result.Issues)))
	p.Line("")

	if len(result.Issues) == 0 {
		p.Line("  No issues found.")
		return
	}

	headers := []string{"#", "SEVERITY", "RULE", "MESSAGE", "RESOURCE"}
	rows := make([][]string, len(result.Issues))
	for i, issue := range result.Issues {
		msg := issue.Message
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		res := issue.Resource
		if len(res) > 40 {
			res = "..." + res[len(res)-37:]
		}
		rows[i] = []string{
			fmt.Sprintf("%d", i+1),
			strings.ToUpper(issue.Severity),
			issue.Rule,
			msg,
			res,
		}
	}
	p.Table(headers, rows)
}
