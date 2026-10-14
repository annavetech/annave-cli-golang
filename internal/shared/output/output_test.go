// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// failWriter fails every write and counts the calls.
type failWriter struct{ calls int }

func (w *failWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, errors.New("disk full")
}

func TestPrinterWritesOutput(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{out: &buf, errOut: &buf, format: FormatPlain}
	p.Line("a")
	p.Linef("b %d", 1)
	p.KeyVal("k", "v")
	p.JSON(map[string]int{"n": 1})
	p.Table([]string{"H1", "H2"}, [][]string{{"x", "y"}})
	if err := p.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
	want := "a\nb 1\n  k:                   v\n{\n  \"n\": 1\n}\nH1   H2\n--   --\nx    y\n"
	if got := buf.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestPrinterKeepsFirstWriteError(t *testing.T) {
	tests := []struct {
		name  string
		write func(p *Printer)
	}{
		{"line", func(p *Printer) { p.Line("a") }},
		{"linef", func(p *Printer) { p.Linef("%s", "a") }},
		{"json", func(p *Printer) { p.JSON([]int{1}) }},
		{"table", func(p *Printer) { p.Table([]string{"H"}, [][]string{{"x"}}) }},
		{"section", func(p *Printer) { p.Section("s") }},
		{"keyval", func(p *Printer) { p.KeyVal("k", "v") }},
		{"indevelopment", func(p *Printer) { p.InDevelopment("m") }},
		{"error", func(p *Printer) { p.Error(errors.New("x")) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &failWriter{}
			p := &Printer{out: w, errOut: w, format: FormatPlain}
			tt.write(p)
			p.Line("after")
			err := p.Err()
			if err == nil || !strings.Contains(err.Error(), "disk full") {
				t.Fatalf("Err() = %v, want the write error", err)
			}
			if w.calls != 1 {
				t.Fatalf("writes after the first error: got %d calls, want 1", w.calls)
			}
		})
	}
}

func TestSectionUnderlineMatchesTitleWidth(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{out: &buf, errOut: &buf, format: FormatPlain}
	p.Section("Café: Señor")
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	want := "\nCafé: Señor\n-----------\n"
	if got := buf.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
