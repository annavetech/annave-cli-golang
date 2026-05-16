// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"

	"annave.tech/cli/internal/cost/domain"
	"annave.tech/cli/internal/cost/port"
)

const (
	anomalyThresholdPct = 20.0 // flag if cost is >20% above rolling average
	anomalyThresholdAbs = 5.0  // and absolute increase is >$5 (noise filter)
	rollingWindowDays   = 7    // days used for rolling average baseline
)

// AWSAnalyzer implements port.CostAnalyzer using the AWS Cost Explorer API.
type AWSAnalyzer struct{}

// newAWS returns an AWSAnalyzer.
func newAWS() *AWSAnalyzer { return &AWSAnalyzer{} }

func (a *AWSAnalyzer) Analyze(ctx context.Context, opts port.AnalyzeOptions) (*domain.CostReport, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load AWS credentials: %w — ensure AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are set, or configure ~/.aws/credentials", err)
	}

	ce := costexplorer.NewFromConfig(cfg)

	since, until := resolvePeriod(opts.Since)

	// --- daily costs grouped by SERVICE ---
	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(since),
			End:   aws.String(until),
		},
		Granularity: cetypes.GranularityDaily,
		GroupBy: []cetypes.GroupDefinition{{
			Type: cetypes.GroupDefinitionTypeDimension,
			Key:  aws.String("SERVICE"),
		}},
		Metrics: []string{"BlendedCost"},
	}

	usage, err := ce.GetCostAndUsage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("AWS Cost Explorer API error: %w", err)
	}

	// Build a time-series map: service → []float64 (daily costs, chronological).
	type dayCost struct {
		date   string
		amount float64
	}
	// Preserve insertion order by collecting dates first.
	var dates []string
	dateSet := make(map[string]bool)
	serviceSeries := make(map[string]map[string]float64) // service → date → amount

	for _, result := range usage.ResultsByTime {
		date := aws.ToString(result.TimePeriod.Start)
		if !dateSet[date] {
			dates = append(dates, date)
			dateSet[date] = true
		}
		for _, group := range result.Groups {
			if len(group.Keys) == 0 {
				continue
			}
			svc := group.Keys[0]
			amt := parseAmount(group.Metrics["BlendedCost"].Amount)
			if _, ok := serviceSeries[svc]; !ok {
				serviceSeries[svc] = make(map[string]float64)
			}
			serviceSeries[svc][date] += amt
		}
	}

	// Aggregate records and detect anomalies.
	var records []domain.CostRecord
	var anomalies []domain.CostAnomaly
	totalCost := 0.0

	for svc, dateCosts := range serviceSeries {
		// Build ordered slice.
		costs := make([]float64, len(dates))
		svcTotal := 0.0
		for i, d := range dates {
			costs[i] = dateCosts[d]
			svcTotal += dateCosts[d]
		}
		totalCost += svcTotal

		if svcTotal < 0.01 {
			continue // skip effectively-zero services
		}

		records = append(records, domain.CostRecord{
			Service:  svc,
			Amount:   roundCents(svcTotal),
			Currency: domain.CurrencyUSD,
			Period:   since + " → " + lastDay(until),
		})

		// Anomaly detection: compare last day to rolling average of previous window.
		if len(costs) > rollingWindowDays {
			last := costs[len(costs)-1]
			window := costs[len(costs)-1-rollingWindowDays : len(costs)-1]
			avg := mean(window)
			if avg > 0 {
				deltaPct := (last - avg) / avg * 100
				if deltaPct > anomalyThresholdPct && (last-avg) > anomalyThresholdAbs {
					anomalies = append(anomalies, domain.CostAnomaly{
						Service:  svc,
						Message:  fmt.Sprintf("%.0f%% above %d-day average ($%.2f/day vs $%.2f/day avg)", deltaPct, rollingWindowDays, last, avg),
						Expected: roundCents(avg),
						Actual:   roundCents(last),
						DeltaPct: roundCents(deltaPct),
					})
				}
			}
		}
	}

	// Sort records by cost descending.
	sort.Slice(records, func(i, j int) bool {
		return records[i].Amount > records[j].Amount
	})
	// Sort anomalies by delta descending.
	sort.Slice(anomalies, func(i, j int) bool {
		return anomalies[i].DeltaPct > anomalies[j].DeltaPct
	})

	// --- optional forecast ---
	forecast := tryForecast(ctx, ce, until)

	report := &domain.CostReport{
		Provider:  "aws",
		Period:    since + " → " + lastDay(until),
		TotalCost: roundCents(totalCost),
		Currency:  domain.CurrencyUSD,
		Records:   records,
		Anomalies: anomalies,
		ScannedAt: time.Now(),
	}
	if forecast != "" {
		// Surface forecast as a zero-cost record with a special period label
		// so the CLI can display it without changing the domain model.
		report.ByResource = []domain.ResourceAttribution{{
			Resource: "forecast (next 30 days)",
			Amount:   parseForecast(forecast),
			Currency: domain.CurrencyUSD,
		}}
	}
	return report, nil
}

// tryForecast attempts a 30-day forecast; returns empty string on any error.
func tryForecast(ctx context.Context, ce *costexplorer.Client, until string) string {
	tomorrow := dateAdd(until, 1)
	endForecast := dateAdd(until, 30)

	out, err := ce.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(tomorrow),
			End:   aws.String(endForecast),
		},
		Granularity: cetypes.GranularityMonthly,
		Metric:      cetypes.MetricBlendedCost,
	})
	if err != nil {
		return "" // forecast is best-effort; insufficient data is common
	}
	if out.Total != nil && out.Total.Amount != nil {
		return aws.ToString(out.Total.Amount)
	}
	return ""
}

// --- helpers ---

// resolvePeriod returns the start and end dates for the query, clamped to yesterday.
func resolvePeriod(since time.Time) (start, end string) {
	// Cost Explorer end date is exclusive and cannot exceed yesterday.
	yesterday := time.Now().AddDate(0, 0, -1)
	end = yesterday.Format("2006-01-02")

	if since.IsZero() {
		start = yesterday.AddDate(0, 0, -30).Format("2006-01-02")
	} else {
		start = since.Format("2006-01-02")
	}
	return
}

func parseAmount(s *string) float64 {
	if s == nil {
		return 0
	}
	v, _ := strconv.ParseFloat(*s, 64)
	return v
}

func parseForecast(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return roundCents(v)
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func roundCents(v float64) float64 {
	return math.Round(v*100) / 100
}

func lastDay(exclusiveEnd string) string {
	t, err := time.Parse("2006-01-02", exclusiveEnd)
	if err != nil {
		return exclusiveEnd
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}

func dateAdd(date string, days int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}
