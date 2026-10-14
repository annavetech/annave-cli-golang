// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/annavetech/tend/internal/cleanup/domain"
	"github.com/annavetech/tend/internal/cleanup/port"
	"github.com/annavetech/tend/internal/cleanup/scanner"
	"github.com/annavetech/tend/internal/shared/config"
	"github.com/annavetech/tend/internal/shared/output"
)

var (
	cleanupScanNamespace  string
	cleanupScanContext    string
	cleanupScanKubeconfig string
	cleanupScanDryRun     bool
	cleanupScanFormat     string
)

var cleanupScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for idle Kubernetes resources",
	Long:  "Scan the cluster for idle pods, stale PVCs, orphaned ConfigMaps, and empty namespaces.",
	RunE:  runCleanupScan,
}

func init() {
	cleanupCmd.AddCommand(cleanupScanCmd)
	cleanupScanCmd.Flags().StringVarP(&cleanupScanNamespace, "namespace", "n", "", "limit scan to a specific namespace (default: all namespaces)")
	cleanupScanCmd.Flags().StringVar(&cleanupScanContext, "context", "", "kubeconfig context to use")
	cleanupScanCmd.Flags().StringVar(&cleanupScanKubeconfig, "kubeconfig", "", "path to kubeconfig file")
	cleanupScanCmd.Flags().BoolVar(&cleanupScanDryRun, "dry-run", true, "preview only, make no changes")
	cleanupScanCmd.Flags().StringVar(&cleanupScanFormat, "format", "plain", "output format: plain, json, table")
}

func runCleanupScan(cmd *cobra.Command, args []string) error {
	outFmt, err := output.ParseFormat(cleanupScanFormat)
	if err != nil {
		return err
	}
	p := output.New(outFmt)

	opts := port.ScanOptions{
		Namespace:      cleanupScanNamespace,
		KubeContext:    cleanupScanContext,
		KubeconfigPath: cleanupScanKubeconfig,
		DryRun:         cleanupScanDryRun,
		MaxPendingAge:  10 * time.Minute,
	}

	timeout := time.Duration(config.App.Limits.Cleanup.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	s := scanner.New()
	plan, err := s.Scan(ctx, opts)
	if err != nil {
		return err
	}

	if err := printWarnings(cmd.ErrOrStderr(), plan.Warnings); err != nil {
		return err
	}
	printCleanupPlan(p, plan)
	if err := p.Err(); err != nil {
		return err
	}
	if len(plan.Resources) > 0 {
		return errProblemsFound
	}
	return nil
}

func printCleanupPlan(p *output.Printer, plan *domain.CleanupPlan) {
	switch p.Format() {
	case output.FormatJSON:
		p.JSON(plan)
	case output.FormatTable:
		printCleanupTable(p, plan)
	default:
		printCleanupPlain(p, plan)
	}
}

func printCleanupPlain(p *output.Printer, plan *domain.CleanupPlan) {
	header := "Cleanup scan"
	if plan.DryRun {
		header += " [DRY RUN]"
	}
	p.Section(fmt.Sprintf("%s: %s", header, plan.ScannedAt.Format("2006-01-02 15:04:05")))
	p.KeyVal("context", plan.Context)
	ns := plan.Namespace
	if ns == "" {
		ns = "all namespaces"
	}
	p.KeyVal("namespace", ns)
	p.KeyVal("findings", fmt.Sprintf("%d idle resource(s)", len(plan.Resources)))
	p.Line("")

	if len(plan.Resources) == 0 {
		p.Line("  No idle resources found.")
		p.Line("")
		return
	}

	groups := scanner.GroupByType(plan.Resources)
	for _, rType := range scanner.SortedTypes(groups) {
		p.Linef("  %s (%d):", rType, len(groups[rType]))
		for _, r := range groups[rType] {
			name := r.Resource.Name
			if r.Resource.Namespace != "" {
				name = r.Resource.Namespace + "/" + name
			}
			p.Linef("    %-50s  %-20s  %s",
				name, string(r.Reason), r.Detail)
		}
		p.Line("")
	}
}

func printCleanupTable(p *output.Printer, plan *domain.CleanupPlan) {
	header := "Cleanup scan"
	if plan.DryRun {
		header += " [DRY RUN]"
	}
	p.Section(fmt.Sprintf("%s: %s", header, plan.ScannedAt.Format("2006-01-02 15:04:05")))
	p.KeyVal("context", plan.Context)
	p.KeyVal("findings", fmt.Sprintf("%d idle resource(s)", len(plan.Resources)))
	p.Line("")

	if len(plan.Resources) == 0 {
		p.Line("  No idle resources found.")
		return
	}

	headers := []string{"KIND", "NAMESPACE/NAME", "REASON", "AGE", "DETAIL"}
	rows := make([][]string, len(plan.Resources))
	for i, r := range plan.Resources {
		name := r.Resource.Name
		if r.Resource.Namespace != "" {
			name = r.Resource.Namespace + "/" + name
		}
		detail := r.Detail
		if len(detail) > 50 {
			detail = detail[:47] + "..."
		}
		rows[i] = []string{
			string(r.Resource.Kind),
			name,
			string(r.Reason),
			scanner.FormatAge(r.Resource.AgeHours),
			detail,
		}
	}
	p.Table(headers, rows)
}
