// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"annave.tech/cli/internal/doc/domain"
	"annave.tech/cli/internal/doc/port"
	"annave.tech/cli/internal/doc/searcher"
	"annave.tech/cli/internal/shared/output"
)

var (
	docSearchPath   string
	docSearchFormat string
	docSearchOpen   bool
	docSearchExt    []string
)

var docSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search documentation files by full-text query",
	Long: `Full-text search across .md, .txt, .rst, .html, .yaml, and .json files.

Multiple words are treated as AND terms. Prefix matching is supported.

Examples:
  annave doc search authentication
  annave doc search "database connection" --ext md --ext txt
  annave doc search timeout --path ./docs --open`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDocSearch,
}

func init() {
	docCmd.AddCommand(docSearchCmd)
	docSearchCmd.Flags().StringVarP(&docSearchPath, "path", "p", ".", "root directory to search")
	docSearchCmd.Flags().StringVar(&docSearchFormat, "format", "plain", "output format: plain, json, table")
	docSearchCmd.Flags().BoolVar(&docSearchOpen, "open", false, "open the top result in $EDITOR")
	docSearchCmd.Flags().StringArrayVar(&docSearchExt, "ext", nil, "file extensions to include (e.g. --ext md --ext txt)")
}

func runDocSearch(cmd *cobra.Command, args []string) error {
	outFmt, err := output.ParseFormat(docSearchFormat)
	if err != nil {
		return err
	}
	p := output.New(outFmt)

	raw := strings.Join(args, " ")
	query := searcher.ParseQuery(raw)

	opts := port.SearchOptions{
		Root: docSearchPath,
		Exts: docSearchExt,
	}

	s := searcher.New()
	results, err := s.Search(cmd.Context(), query, opts)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		if p.Format() == output.FormatJSON {
			p.JSON([]domain.SearchResult{})
		} else {
			p.Linef("No results for %q in %s", raw, docSearchPath)
		}
		return p.Err()
	}

	if docSearchOpen {
		top := results[0]
		if err := docNotice(cmd, p, fmt.Sprintf("Opening: %s (line %d)", top.File.RelPath, top.Line)); err != nil {
			return err
		}
		return searcher.OpenInEditor(top.File.Path, top.Line)
	}

	printDocResults(p, results, query)
	return p.Err()
}

// docNotice prints msg to stdout, or to stderr in JSON mode so stdout stays valid JSON.
func docNotice(cmd *cobra.Command, p *output.Printer, msg string) error {
	if p.Format() != output.FormatJSON {
		p.Line(msg)
		return p.Err()
	}
	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), msg); err != nil {
		return fmt.Errorf("write notice: %w", err)
	}
	return nil
}

func printDocResults(p *output.Printer, results []domain.SearchResult, query domain.SearchQuery) {
	switch p.Format() {
	case output.FormatJSON:
		p.JSON(results)
	case output.FormatTable:
		printDocTable(p, results)
	default:
		printDocPlain(p, results, query)
	}
}

func printDocPlain(p *output.Printer, results []domain.SearchResult, query domain.SearchQuery) {
	p.Section(fmt.Sprintf("Doc search — %q  (%d result(s))", query.Raw, len(results)))
	p.Line("")
	for i, r := range results {
		p.Linef("  [%d] %s", i+1, r.File.RelPath)
		if r.File.Title != "" {
			p.Linef("       %s", r.File.Title)
		}
		if r.Excerpt != "" {
			p.Linef("       %s", r.Excerpt)
		}
		p.Linef("       line %-5d  score %.1f  %s", r.Line, r.Score, string(r.File.Format))
		p.Line("")
	}
}

func printDocTable(p *output.Printer, results []domain.SearchResult) {
	headers := []string{"#", "FILE", "TITLE", "LINE", "SCORE"}
	rows := make([][]string, len(results))
	for i, r := range results {
		title := r.File.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		relPath := r.File.RelPath
		if len(relPath) > 50 {
			relPath = "..." + relPath[len(relPath)-47:]
		}
		rows[i] = []string{
			fmt.Sprintf("%d", i+1),
			relPath,
			title,
			fmt.Sprintf("%d", r.Line),
			fmt.Sprintf("%.1f", r.Score),
		}
	}
	p.Table(headers, rows)
}
