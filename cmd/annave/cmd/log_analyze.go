// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"annave.tech/cli/internal/log/analyzer"
	"annave.tech/cli/internal/log/domain"
	"annave.tech/cli/internal/log/port"
	"annave.tech/cli/internal/shared/output"
)

var (
	logAnalyzeStdin  bool
	logAnalyzeFormat string
	logAnalyzeSince  string
	logAnalyzeLevel  string
)

var logAnalyzeCmd = &cobra.Command{
	Use:   "analyze [file]",
	Short: "Analyze a log file for errors, spikes, and patterns",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogAnalyze,
}

func init() {
	logCmd.AddCommand(logAnalyzeCmd)
	logAnalyzeCmd.Flags().BoolVar(&logAnalyzeStdin, "stdin", false, "read from stdin instead of a file")
	logAnalyzeCmd.Flags().StringVar(&logAnalyzeFormat, "format", "plain", "output format: plain, json, table")
	logAnalyzeCmd.Flags().StringVar(&logAnalyzeSince, "since", "", "only include entries after this time (e.g. 1h, 30m, or RFC3339)")
	logAnalyzeCmd.Flags().StringVar(&logAnalyzeLevel, "level", "", "minimum log level: debug, info, warn, error, critical")
}

func runLogAnalyze(cmd *cobra.Command, args []string) (retErr error) {
	outFmt, err := output.ParseFormat(logAnalyzeFormat)
	if err != nil {
		return err
	}
	p := output.New(outFmt)

	minLevel, err := analyzer.ParseLevel(logAnalyzeLevel)
	if err != nil {
		return newUsageError(cmd, err)
	}
	if !logAnalyzeStdin && len(args) == 0 {
		return newUsageError(cmd, errors.New("no input: pass a file, \"-\" or --stdin"))
	}

	in := os.Stdin
	filename := "<stdin>"
	if !logAnalyzeStdin && args[0] != "-" {
		filename = args[0]
		f, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("cannot open %q: %w", filename, err)
		}
		defer func() {
			if err := f.Close(); err != nil && retErr == nil {
				retErr = fmt.Errorf("close %q: %w", filename, err)
			}
		}()
		in = f
	}

	var since time.Time
	if logAnalyzeSince != "" {
		since, err = parseSince(logAnalyzeSince)
		if err != nil {
			return fmt.Errorf("invalid --since value %q: %w", logAnalyzeSince, err)
		}
	}

	opts := port.AnalyzeOptions{
		Since:    since,
		MinLevel: minLevel,
	}

	report, err := analyzeInput(cmd.Context(), in, filename, opts)
	if err != nil {
		return err
	}

	printReport(p, report)
	if err := p.Err(); err != nil {
		return err
	}
	if len(report.Findings) > 0 {
		return errProblemsFound
	}
	return nil
}

// deadlineReader is an input whose blocked reads can be ended by a deadline.
type deadlineReader interface {
	io.Reader
	SetReadDeadline(t time.Time) error
}

// analyzeInput runs the analysis and returns ctx.Err() as soon as ctx ends, even mid-read.
func analyzeInput(ctx context.Context, r io.Reader, filename string, opts port.AnalyzeOptions) (*domain.AnalysisReport, error) {
	if dr, ok := r.(deadlineReader); ok {
		err := dr.SetReadDeadline(time.Time{})
		if err == nil {
			return analyzeWithDeadline(ctx, dr, filename, opts)
		}
		if !errors.Is(err, os.ErrNoDeadline) {
			return nil, fmt.Errorf("prepare reading %q: %w", filename, err)
		}
	}
	// Without deadlines (such as a terminal), the read runs in a goroutine that is
	// abandoned on cancel and ends when the process exits.
	type outcome struct {
		report *domain.AnalysisReport
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		report, err := analyzer.New().Analyze(ctx, r, filename, opts)
		done <- outcome{report, err}
	}()
	select {
	case o := <-done:
		return o.report, o.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// analyzeWithDeadline sets a read deadline when ctx ends so a blocked read returns.
func analyzeWithDeadline(ctx context.Context, dr deadlineReader, filename string, opts port.AnalyzeOptions) (*domain.AnalysisReport, error) {
	deadlineErr := make(chan error, 1)
	stopDeadline := context.AfterFunc(ctx, func() { deadlineErr <- dr.SetReadDeadline(time.Now()) })
	report, err := analyzer.New().Analyze(ctx, dr, filename, opts)
	if !stopDeadline() {
		if derr := <-deadlineErr; derr != nil {
			return nil, errors.Join(ctx.Err(), fmt.Errorf("stop reading %q: %w", filename, derr))
		}
	}
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	return report, nil
}

func printReport(p *output.Printer, report *domain.AnalysisReport) {
	switch p.Format() {
	case output.FormatJSON:
		p.JSON(report)
	case output.FormatTable:
		printReportTable(p, report)
	default:
		printReportPlain(p, report)
	}
}

func printReportPlain(p *output.Printer, report *domain.AnalysisReport) {
	p.Section(fmt.Sprintf("Log analysis — %s", report.File))
	p.KeyVal("format", string(report.Format))
	p.KeyVal("lines parsed", fmt.Sprintf("%d / %d", report.ParsedLines, report.TotalLines))
	if report.Truncated > 0 {
		p.KeyVal("truncated lines", fmt.Sprintf("%d (cut to 1 MiB)", report.Truncated))
	}
	if !report.TimeRange.From.IsZero() {
		p.KeyVal("time range", fmt.Sprintf("%s → %s",
			report.TimeRange.From.Format("2006-01-02 15:04:05"),
			report.TimeRange.To.Format("2006-01-02 15:04:05"),
		))
	}

	if len(report.Findings) == 0 {
		p.Line("\n  No anomalies detected.")
		return
	}

	p.Linef("\n  %d finding(s):\n", len(report.Findings))
	for _, f := range report.Findings {
		p.Linef("  [%d] %-8s  %s", f.Rank, strings.ToUpper(string(f.Severity)), f.Summary)
		if f.Detail != "" {
			for _, line := range strings.Split(f.Detail, "\n") {
				p.Linef("           %s", line)
			}
		}
	}
	p.Line("")
}

func printReportTable(p *output.Printer, report *domain.AnalysisReport) {
	p.Section(fmt.Sprintf("Log analysis — %s", report.File))
	p.KeyVal("format", string(report.Format))
	p.KeyVal("lines parsed", fmt.Sprintf("%d / %d", report.ParsedLines, report.TotalLines))

	if len(report.Findings) == 0 {
		p.Line("\n  No anomalies detected.")
		return
	}

	headers := []string{"RANK", "SEVERITY", "COUNT", "SUMMARY"}
	rows := make([][]string, len(report.Findings))
	for i, f := range report.Findings {
		summary := f.Summary
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		rows[i] = []string{
			fmt.Sprintf("%d", f.Rank),
			strings.ToUpper(string(f.Severity)),
			fmt.Sprintf("%d", f.Count),
			summary,
		}
	}
	p.Table(headers, rows)
}

func parseSince(s string) (time.Time, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("use a duration (1h, 30m) or a date (2006-01-02)")
}
