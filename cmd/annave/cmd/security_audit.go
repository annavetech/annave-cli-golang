// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"annave.tech/cli/internal/security/domain"
	"annave.tech/cli/internal/security/port"
	"annave.tech/cli/internal/security/scanner"
	"annave.tech/cli/internal/shared/output"
)

var (
	securityAuditType       string
	securityAuditFormat     string
	securityAuditKubeconfig string
	securityAuditContext    string
	securityAuditNamespace  string
)

var securityAuditCmd = &cobra.Command{
	Use:   "audit [path]",
	Short: "Audit a path for security issues",
	Long: `Scan for secrets, SAST issues, or Kubernetes misconfigurations.

Scan types:
  secrets    Scan for hardcoded credentials, keys, and tokens (default)
  sast       Static analysis for Go and TypeScript security patterns
  k8s-live   Check running cluster workloads via kubeconfig
  k8s-local  Validate local Kubernetes YAML manifests

Examples:
  annave security audit .
  annave security audit ./src --type sast
  annave security audit --type k8s-live --context production
  annave security audit ./k8s --type k8s-local --format table`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSecurityAudit,
}

func init() {
	securityCmd.AddCommand(securityAuditCmd)
	securityAuditCmd.Flags().StringVar(&securityAuditType, "type", "secrets", "scan type: secrets, sast, k8s-live, k8s-local")
	securityAuditCmd.Flags().StringVar(&securityAuditFormat, "format", "plain", "output format: plain, json, table")
	securityAuditCmd.Flags().StringVar(&securityAuditKubeconfig, "kubeconfig", "", "kubeconfig file (k8s-live only)")
	securityAuditCmd.Flags().StringVar(&securityAuditContext, "context", "", "kubeconfig context (k8s-live only)")
	securityAuditCmd.Flags().StringVar(&securityAuditNamespace, "namespace", "", "namespace to scan (k8s-live only, default: all namespaces)")
}

func runSecurityAudit(cmd *cobra.Command, args []string) error {
	outFmt, err := output.ParseFormat(securityAuditFormat)
	if err != nil {
		return err
	}
	p := output.New(outFmt)

	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	auditType, err := parseAuditType(securityAuditType)
	if err != nil {
		return err
	}

	target := domain.AuditTarget{Path: path, Type: auditType}
	opts := port.AuditOptions{
		Type:           auditType,
		KubeconfigPath: securityAuditKubeconfig,
		KubeContext:    securityAuditContext,
		Namespace:      securityAuditNamespace,
	}

	s := scanner.New()
	report, err := s.Audit(context.Background(), target, opts)
	if err != nil {
		return err
	}

	if err := printWarnings(cmd.ErrOrStderr(), report.Warnings); err != nil {
		return err
	}
	printAuditReport(p, report)
	return p.Err()
}

// printWarnings writes one "warning:" line per entry to w.
func printWarnings(w io.Writer, warnings []string) error {
	for _, msg := range warnings {
		if _, err := fmt.Fprintf(w, "warning: %s\n", msg); err != nil {
			return fmt.Errorf("write warning: %w", err)
		}
	}
	return nil
}

func parseAuditType(s string) (domain.AuditType, error) {
	switch strings.ToLower(s) {
	case "secrets", "":
		return domain.AuditTypeSecrets, nil
	case "sast":
		return domain.AuditTypeSAST, nil
	case "k8s-live":
		return domain.AuditTypeK8sLive, nil
	case "k8s-local":
		return domain.AuditTypeK8sLocal, nil
	default:
		return "", fmt.Errorf("unknown type %q — use secrets, sast, k8s-live, or k8s-local", s)
	}
}

func printAuditReport(p *output.Printer, report *domain.AuditReport) {
	switch p.Format() {
	case output.FormatJSON:
		p.JSON(report)
	case output.FormatTable:
		printAuditTable(p, report)
	default:
		printAuditPlain(p, report)
	}
}

func printAuditPlain(p *output.Printer, report *domain.AuditReport) {
	p.Section(fmt.Sprintf("Security audit — %s [%s]", report.Target.Path, report.Target.Type))
	p.KeyVal("scanned at", report.ScannedAt.Format("2006-01-02 15:04:05"))
	p.KeyVal("findings", fmt.Sprintf("%d", len(report.Findings)))

	for sev, count := range report.Summary {
		p.KeyVal("  "+string(sev), fmt.Sprintf("%d", count))
	}
	p.Line("")

	if len(report.Findings) == 0 {
		p.Line("  No issues found.")
		p.Line("")
		return
	}

	for i, f := range report.Findings {
		p.Linef("  [%d] %-8s  %s  %s", i+1, strings.ToUpper(string(f.Severity)), f.ID, f.Title)
		if f.File != "" {
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			p.Linef("       file: %s", loc)
		}
		if f.Detail != "" {
			p.Linef("       detail: %s", f.Detail)
		}
		if f.Remediation != "" {
			p.Linef("       fix: %s", f.Remediation)
		}
		p.Line("")
	}
}

func printAuditTable(p *output.Printer, report *domain.AuditReport) {
	p.Section(fmt.Sprintf("Security audit — %s [%s]", report.Target.Path, report.Target.Type))
	p.KeyVal("findings", fmt.Sprintf("%d", len(report.Findings)))
	p.Line("")

	if len(report.Findings) == 0 {
		p.Line("  No issues found.")
		return
	}

	headers := []string{"#", "SEVERITY", "RULE", "TITLE", "FILE", "LINE"}
	rows := make([][]string, len(report.Findings))
	for i, f := range report.Findings {
		loc := "-"
		if f.Line > 0 {
			loc = fmt.Sprintf("%d", f.Line)
		}
		file := f.File
		if len(file) > 40 {
			file = "..." + file[len(file)-37:]
		}
		title := f.Title
		if len(title) > 35 {
			title = title[:32] + "..."
		}
		rows[i] = []string{
			fmt.Sprintf("%d", i+1),
			strings.ToUpper(string(f.Severity)),
			f.ID,
			title,
			file,
			loc,
		}
	}
	p.Table(headers, rows)
}
