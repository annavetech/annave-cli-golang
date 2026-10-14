// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Format defines the output format for a command.
type Format string

const (
	FormatPlain Format = "plain"
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// ParseFormat parses the --format flag value.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "plain", "":
		return FormatPlain, nil
	case "json":
		return FormatJSON, nil
	case "table":
		return FormatTable, nil
	default:
		return "", fmt.Errorf("unknown format %q — use plain, json, or table", s)
	}
}

// Printer writes formatted output to stdout/stderr.
// The first write error is kept and stops later writes; Err returns it.
type Printer struct {
	out    io.Writer
	errOut io.Writer
	format Format
	err    error
}

// New creates a Printer writing to stdout/stderr.
func New(format Format) *Printer {
	return &Printer{out: os.Stdout, errOut: os.Stderr, format: format}
}

// Format returns the configured output format.
func (p *Printer) Format() Format { return p.format }

// Err returns the first write error, or nil.
func (p *Printer) Err() error { return p.err }

func (p *Printer) write(w io.Writer, s string) {
	if p.err != nil {
		return
	}
	if _, err := io.WriteString(w, s); err != nil {
		p.err = fmt.Errorf("write output: %w", err)
	}
}

// Line prints a single line to stdout.
func (p *Printer) Line(msg string) {
	p.write(p.out, msg+"\n")
}

// Linef prints a formatted line to stdout.
func (p *Printer) Linef(format string, args ...any) {
	p.write(p.out, fmt.Sprintf(format+"\n", args...))
}

// JSON encodes v as indented JSON to stdout.
func (p *Printer) JSON(v any) {
	if p.err != nil {
		return
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		p.err = fmt.Errorf("encode json: %w", err)
		return
	}
	p.write(p.out, string(b)+"\n")
}

// Table prints headers and rows as a tab-aligned table.
func (p *Printer) Table(headers []string, rows [][]string) {
	if p.err != nil {
		return
	}
	// The table is aligned in memory and written once, so one write error stops it.
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	p.write(w, strings.Join(headers, "\t")+"\n")
	sep := make([]string, len(headers))
	for i, h := range headers {
		sep[i] = strings.Repeat("-", len(h))
	}
	p.write(w, strings.Join(sep, "\t")+"\n")
	for _, row := range rows {
		p.write(w, strings.Join(row, "\t")+"\n")
	}
	if err := w.Flush(); err != nil && p.err == nil {
		p.err = fmt.Errorf("write output: %w", err)
	}
	p.write(p.out, buf.String())
}

// Error prints an error to stderr.
func (p *Printer) Error(err error) {
	p.write(p.errOut, fmt.Sprintf("error: %v\n", err))
}

// InDevelopment prints a consistent "in development" notice.
func (p *Printer) InDevelopment(module string) {
	p.write(p.out, fmt.Sprintf("\n  annave %s is in development.\n\n", module))
	p.write(p.out, "  This module has been architected and will ship in a future release.\n")
	p.write(p.out, "  Follow updates at https://www.annave.tech\n\n")
}

// Section prints a labelled section header.
func (p *Printer) Section(label string) {
	p.write(p.out, fmt.Sprintf("\n%s\n%s\n", label, strings.Repeat("─", len(label))))
}

// KeyVal prints a key: value pair.
func (p *Printer) KeyVal(key, val string) {
	p.write(p.out, fmt.Sprintf("  %-20s %s\n", key+":", val))
}
