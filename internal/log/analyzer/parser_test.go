// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package analyzer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/annavetech/tend/internal/log/port"
)

func TestParseLinesContinuesAfterOverlongLine(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&b, "ERROR before %d\n", i)
	}
	b.WriteString(strings.Repeat("x", 2*1024*1024) + "\n")
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&b, "ERROR after %d\n", i)
	}

	entries, _, total, _, truncated, err := parseLines(t.Context(), strings.NewReader(b.String()), 0)
	if err != nil {
		t.Fatal(err)
	}
	errorsSeen := 0
	for _, e := range entries {
		if e.Level == "error" {
			errorsSeen++
		}
	}
	if errorsSeen != 13 || total != 14 || truncated != 1 {
		t.Fatalf("got %d error entries, %d lines, %d truncated; want 13, 14, 1", errorsSeen, total, truncated)
	}
	if got := len(entries[3].Message); got != maxLogLineBytes {
		t.Errorf("truncated line kept %d bytes, want %d", got, maxLogLineBytes)
	}
}

func TestParseLinesLastLineWithoutNewline(t *testing.T) {
	entries, _, total, _, truncated, err := parseLines(t.Context(), strings.NewReader("ERROR a\r\nERROR b"), 0)
	if err != nil || total != 2 || truncated != 0 || len(entries) != 2 || entries[0].Message != "ERROR a" {
		t.Fatalf("entries=%+v total=%d truncated=%d err=%v", entries, total, truncated, err)
	}
}

func TestAnalyzeReturnsReadError(t *testing.T) {
	readErr := errors.New("disk failure")
	_, err := New().Analyze(t.Context(), iotest.ErrReader(readErr), "x.log", port.AnalyzeOptions{})
	if !errors.Is(err, readErr) {
		t.Fatalf("want the read error, got %v", err)
	}
}

func TestAnalyzeStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := New().Analyze(ctx, strings.NewReader("ERROR a\nERROR b\n"), "app.log", port.AnalyzeOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Analyze error = %v, want context.Canceled", err)
	}
}
