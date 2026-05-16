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
type Printer struct {
	out    io.Writer
	errOut io.Writer
	format Format
}

// New creates a Printer writing to stdout/stderr.
func New(format Format) *Printer {
	return &Printer{out: os.Stdout, errOut: os.Stderr, format: format}
}

// Format returns the configured output format.
func (p *Printer) Format() Format { return p.format }

// Line prints a single line to stdout.
func (p *Printer) Line(msg string) {
	fmt.Fprintln(p.out, msg)
}

// Linef prints a formatted line to stdout.
func (p *Printer) Linef(format string, args ...any) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

// JSON encodes v as indented JSON to stdout.
func (p *Printer) JSON(v any) {
	enc := json.NewEncoder(p.out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// Table prints headers and rows as a tab-aligned table.
func (p *Printer) Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(p.out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = strings.Repeat("-", len(headers[i]))
	}
	fmt.Fprintln(w, strings.Join(sep, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// Error prints an error to stderr.
func (p *Printer) Error(err error) {
	fmt.Fprintf(p.errOut, "error: %v\n", err)
}

// InDevelopment prints a consistent "in development" notice.
func (p *Printer) InDevelopment(module string) {
	fmt.Fprintf(p.out, "\n  annave %s is in development.\n\n", module)
	fmt.Fprintf(p.out, "  This module has been architected and will ship in a future release.\n")
	fmt.Fprintf(p.out, "  Follow updates at https://www.annave.tech\n\n")
}

// Section prints a labelled section header.
func (p *Printer) Section(label string) {
	fmt.Fprintf(p.out, "\n%s\n%s\n", label, strings.Repeat("─", len(label)))
}

// KeyVal prints a key: value pair.
func (p *Printer) KeyVal(key, val string) {
	fmt.Fprintf(p.out, "  %-20s %s\n", key+":", val)
}
