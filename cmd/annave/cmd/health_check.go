// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"annave.tech/cli/internal/health/checker"
	"annave.tech/cli/internal/health/domain"
	"annave.tech/cli/internal/health/port"
	"annave.tech/cli/internal/shared/config"
	"annave.tech/cli/internal/shared/output"
)

var (
	healthCheckTimeout string
	healthCheckFormat  string
	healthCheckChain   bool
)

var healthCheckCmd = &cobra.Command{
	Use:   "check [targets...]",
	Short: "Check uptime and latency of services",
	Long: `Check HTTP, TCP, and DNS targets concurrently.

Target formats:
  https://api.example.com   HTTP/HTTPS (expects 2xx)
  tcp://db.internal:5432    TCP port connectivity
  dns://api.example.com     DNS resolution
  db.internal:5432          TCP (auto-detected from host:port)
  api.example.com           DNS (auto-detected from plain hostname)`,
	Args: cobra.MinimumNArgs(1),
	RunE: runHealthCheck,
}

func init() {
	healthCmd.AddCommand(healthCheckCmd)
	healthCheckCmd.Flags().StringVar(&healthCheckTimeout, "timeout", "10s", "per-target timeout (e.g. 5s, 10s)")
	healthCheckCmd.Flags().StringVar(&healthCheckFormat, "format", "plain", "output format: plain, json, table")
	healthCheckCmd.Flags().BoolVar(&healthCheckChain, "chain", false, "stop at first failure (dependency chain mode)")
}

func runHealthCheck(cmd *cobra.Command, args []string) error {
	outFmt, err := output.ParseFormat(healthCheckFormat)
	if err != nil {
		return err
	}
	p := output.New(outFmt)

	timeout, err := time.ParseDuration(healthCheckTimeout)
	if err != nil {
		return fmt.Errorf("invalid --timeout %q: %w", healthCheckTimeout, err)
	}
	maxTimeout := time.Duration(config.App.Limits.Health.TimeoutSeconds) * time.Second
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	targets := make([]domain.ServiceTarget, len(args))
	for i, arg := range args {
		targets[i] = checker.ParseTarget(arg, timeout)
	}

	// Global deadline: per-target timeout × targets for chain mode, otherwise timeout + buffer.
	var globalTimeout time.Duration
	if healthCheckChain {
		globalTimeout = timeout * time.Duration(len(targets)+1)
	} else {
		globalTimeout = timeout + 5*time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	// checkErr reports close failures; the report is still printed before it is returned.
	var report *domain.HealthReport
	var checkErr error
	if healthCheckChain {
		var results []domain.HealthResult
		results, checkErr = checker.CheckChain(ctx, targets)
		report = buildHealthReport(results)
	} else {
		c := checker.New()
		report, checkErr = c.Check(ctx, targets, port.CheckOptions{})
		if report == nil {
			return checkErr
		}
	}

	printHealthReport(p, report)
	return errors.Join(checkErr, p.Err())
}

func buildHealthReport(results []domain.HealthResult) *domain.HealthReport {
	r := &domain.HealthReport{
		CheckedAt: time.Now(),
		Results:   results,
		Total:     len(results),
	}
	for _, res := range results {
		if res.Status == domain.StatusUp {
			r.Up++
		} else {
			r.Down++
		}
	}
	return r
}

func printHealthReport(p *output.Printer, report *domain.HealthReport) {
	switch p.Format() {
	case output.FormatJSON:
		p.JSON(report)
	case output.FormatTable:
		printHealthTable(p, report)
	default:
		printHealthPlain(p, report)
	}
}

func printHealthPlain(p *output.Printer, report *domain.HealthReport) {
	p.Section(fmt.Sprintf("Health check — %s", report.CheckedAt.Format("2006-01-02 15:04:05")))
	p.KeyVal("summary", fmt.Sprintf("%d total  %d up  %d down", report.Total, report.Up, report.Down))
	p.Line("")

	for _, r := range report.Results {
		icon := "✓"
		if r.Status != domain.StatusUp {
			icon = "✗"
		}
		lat := ""
		if r.Latency > 0 {
			lat = fmt.Sprintf("  %s", r.Latency.Round(time.Millisecond))
		}
		p.Linef("  %s  %-45s  %-8s  %s%s",
			icon, r.Target.Name, strings.ToUpper(string(r.Status)), r.Message, lat)
	}
	p.Line("")
}

func printHealthTable(p *output.Printer, report *domain.HealthReport) {
	p.Section(fmt.Sprintf("Health check — %s", report.CheckedAt.Format("2006-01-02 15:04:05")))
	p.KeyVal("summary", fmt.Sprintf("%d total  %d up  %d down", report.Total, report.Up, report.Down))
	p.Line("")

	headers := []string{"STATUS", "TARGET", "TYPE", "LATENCY", "MESSAGE"}
	rows := make([][]string, len(report.Results))
	for i, r := range report.Results {
		lat := "-"
		if r.Latency > 0 {
			lat = r.Latency.Round(time.Millisecond).String()
		}
		rows[i] = []string{
			strings.ToUpper(string(r.Status)),
			r.Target.Name,
			string(r.Target.CheckType),
			lat,
			r.Message,
		}
	}
	p.Table(headers, rows)
}
