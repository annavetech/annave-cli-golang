// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/annavetech/tend/internal/cost/analyzer"
	"github.com/annavetech/tend/internal/cost/domain"
	"github.com/annavetech/tend/internal/cost/port"
	"github.com/annavetech/tend/internal/shared/output"
)

var (
	costScanProvider string
	costScanSince    string
	costScanFormat   string
)

var costScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan cloud costs and detect anomalies",
	Long: `Fetch cost data from your cloud provider and surface anomalies.

AWS auth uses the standard credential chain:
  1. AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY environment variables
  2. ~/.aws/credentials file
  3. IAM instance role (when running on EC2/ECS/Lambda)

Note: AWS Cost Explorer API charges $0.01 per API request.

Examples:
  tend cost scan
  tend cost scan --since 2026-04-01
  tend cost scan --provider aws --format table`,
	RunE: runCostScan,
}

func init() {
	costCmd.AddCommand(costScanCmd)
	costScanCmd.Flags().StringVar(&costScanProvider, "provider", "aws", "cloud provider: aws, gcp, azure")
	costScanCmd.Flags().StringVar(&costScanSince, "since", "", "billing period start date (YYYY-MM-DD); default: 30 days ago")
	costScanCmd.Flags().StringVar(&costScanFormat, "format", "plain", "output format: plain, json, table")
}

func runCostScan(cmd *cobra.Command, args []string) error {
	outFmt, err := output.ParseFormat(costScanFormat)
	if err != nil {
		return err
	}
	p := output.New(outFmt)

	var since time.Time
	if costScanSince != "" {
		since, err = time.Parse("2006-01-02", costScanSince)
		if err != nil {
			return fmt.Errorf("invalid --since value %q: use YYYY-MM-DD format", costScanSince)
		}
	}

	opts := port.AnalyzeOptions{
		Provider: costScanProvider,
		Since:    since,
	}

	a := analyzer.New()
	report, err := a.Analyze(cmd.Context(), opts)
	if err != nil {
		return err
	}

	printCostReport(p, report)
	if err := p.Err(); err != nil {
		return err
	}
	if len(report.Anomalies) > 0 {
		return errProblemsFound
	}
	return nil
}

func printCostReport(p *output.Printer, report *domain.CostReport) {
	switch p.Format() {
	case output.FormatJSON:
		p.JSON(report)
	case output.FormatTable:
		printCostTable(p, report)
	default:
		printCostPlain(p, report)
	}
}

func printCostPlain(p *output.Printer, report *domain.CostReport) {
	p.Section(fmt.Sprintf("Cost analysis: %s", strings.ToUpper(report.Provider)))
	p.KeyVal("period", report.Period)
	p.KeyVal("total cost", fmt.Sprintf("$%.2f %s", report.TotalCost, report.Currency))
	p.KeyVal("scanned at", report.ScannedAt.Format("2006-01-02 15:04:05"))

	// Forecast surfaced via ByResource.
	for _, r := range report.ByResource {
		if strings.HasPrefix(r.Resource, "forecast") {
			p.KeyVal(r.Resource, fmt.Sprintf("$%.2f %s", r.Amount, r.Currency))
		}
	}
	p.Line("")

	// Top services.
	if len(report.Records) > 0 {
		p.Linef("  TOP SERVICES BY COST (%d):\n", len(report.Records))
		limit := len(report.Records)
		if limit > 15 {
			limit = 15
		}
		for _, rec := range report.Records[:limit] {
			pct := 0.0
			if report.TotalCost > 0 {
				pct = rec.Amount / report.TotalCost * 100
			}
			p.Linef("    %-45s  $%9.2f  (%4.1f%%)", rec.Service, rec.Amount, pct)
		}
		if len(report.Records) > 15 {
			p.Linef("    ... and %d more services", len(report.Records)-15)
		}
		p.Line("")
	}

	// Anomalies.
	if len(report.Anomalies) > 0 {
		p.Linef("  ANOMALIES (%d):\n", len(report.Anomalies))
		for i, a := range report.Anomalies {
			p.Linef("  [%d] %s: %s", i+1, a.Service, a.Message)
		}
		p.Line("")
	} else {
		p.Line("  No cost anomalies detected.\n")
	}
}

func printCostTable(p *output.Printer, report *domain.CostReport) {
	p.Section(fmt.Sprintf("Cost analysis: %s", strings.ToUpper(report.Provider)))
	p.KeyVal("period", report.Period)
	p.KeyVal("total cost", fmt.Sprintf("$%.2f %s", report.TotalCost, report.Currency))
	p.Line("")

	if len(report.Records) == 0 {
		p.Line("  No cost data found for this period.")
		return
	}

	headers := []string{"SERVICE", "COST (USD)", "% OF TOTAL", "ANOMALY"}
	rows := make([][]string, len(report.Records))

	anomalySet := make(map[string]string)
	for _, a := range report.Anomalies {
		anomalySet[a.Service] = fmt.Sprintf("+%.0f%%", a.DeltaPct)
	}

	for i, rec := range report.Records {
		pct := 0.0
		if report.TotalCost > 0 {
			pct = rec.Amount / report.TotalCost * 100
		}
		svc := rec.Service
		if len(svc) > 40 {
			svc = svc[:37] + "..."
		}
		rows[i] = []string{
			svc,
			fmt.Sprintf("$%.2f", rec.Amount),
			fmt.Sprintf("%.1f%%", pct),
			anomalySet[rec.Service],
		}
	}
	p.Table(headers, rows)
}
