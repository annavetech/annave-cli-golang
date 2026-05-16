// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package analyzer

import (
	"fmt"
	"sort"
	"time"

	"annave.tech/cli/internal/log/domain"
)

// detectSpikes delegates to time-based or line-based spike detection depending on whether timestamps are present.
func detectSpikes(entries []domain.LogEntry) []domain.Anomaly {
	var errorEntries []domain.LogEntry
	for _, e := range entries {
		if e.Level == "error" || e.Level == "critical" {
			errorEntries = append(errorEntries, e)
		}
	}
	if len(errorEntries) < 5 {
		return nil
	}
	for _, e := range errorEntries {
		if !e.Timestamp.IsZero() {
			return detectTimeSpikes(errorEntries)
		}
	}
	return detectLineSpikes(errorEntries, len(entries))
}

// detectTimeSpikes finds one-minute windows with ≥3× the average error rate.
func detectTimeSpikes(errors []domain.LogEntry) []domain.Anomaly {
	type bucket struct {
		start time.Time
		count int
		first domain.LogEntry
		last  domain.LogEntry
	}

	bmap := make(map[int64]*bucket)
	for _, e := range errors {
		if e.Timestamp.IsZero() {
			continue
		}
		minute := e.Timestamp.Truncate(time.Minute).Unix()
		b, ok := bmap[minute]
		if !ok {
			b = &bucket{start: e.Timestamp.Truncate(time.Minute), first: e}
			bmap[minute] = b
		}
		b.count++
		b.last = e
	}

	if len(bmap) < 2 {
		return nil
	}

	counts := make([]int, 0, len(bmap))
	for _, b := range bmap {
		counts = append(counts, b.count)
	}
	avg := average(counts)
	if avg == 0 {
		return nil
	}

	var anomalies []domain.Anomaly
	for _, b := range bmap {
		ratio := float64(b.count) / avg
		if ratio >= 3.0 && b.count >= 5 {
			anomalies = append(anomalies, domain.Anomaly{
				Kind:      domain.AnomalySpike,
				Message:   fmt.Sprintf("%.0f× error spike at %s (%d errors in 1 minute)", ratio, b.start.Format("15:04"), b.count),
				Count:     b.count,
				FirstLine: b.first.Line,
				LastLine:  b.last.Line,
				FirstSeen: b.first.Timestamp,
				LastSeen:  b.last.Timestamp,
			})
		}
	}

	sort.Slice(anomalies, func(i, j int) bool {
		return anomalies[i].Count > anomalies[j].Count
	})
	return anomalies
}

// detectLineSpikes finds line-range windows with ≥3× the average error density.
func detectLineSpikes(errors []domain.LogEntry, totalLines int) []domain.Anomaly {
	windowSize := totalLines / 10
	if windowSize < 10 {
		windowSize = 10
	}

	type window struct {
		start int
		end   int
		count int
	}

	windows := make(map[int]*window)
	for _, e := range errors {
		w := e.Line / windowSize
		win, ok := windows[w]
		if !ok {
			win = &window{start: w * windowSize, end: (w+1)*windowSize - 1}
			windows[w] = win
		}
		win.count++
	}

	counts := make([]int, 0, len(windows))
	for _, w := range windows {
		counts = append(counts, w.count)
	}
	avg := average(counts)
	if avg == 0 {
		return nil
	}

	var anomalies []domain.Anomaly
	for _, win := range windows {
		ratio := float64(win.count) / avg
		if ratio >= 3.0 && win.count >= 5 {
			anomalies = append(anomalies, domain.Anomaly{
				Kind:      domain.AnomalySpike,
				Message:   fmt.Sprintf("%.0f× error spike at lines %d–%d (%d errors)", ratio, win.start, win.end, win.count),
				Count:     win.count,
				FirstLine: win.start,
				LastLine:  win.end,
			})
		}
	}

	sort.Slice(anomalies, func(i, j int) bool {
		return anomalies[i].Count > anomalies[j].Count
	})
	return anomalies
}

// average returns the mean of a slice of ints.
func average(nums []int) float64 {
	if len(nums) == 0 {
		return 0
	}
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return float64(sum) / float64(len(nums))
}
