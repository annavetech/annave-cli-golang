// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// DocFormat identifies the file format of a document.
type DocFormat string

const (
	FormatMarkdown DocFormat = "md"
	FormatText     DocFormat = "txt"
	FormatRST      DocFormat = "rst"
	FormatHTML     DocFormat = "html"
	FormatYAML     DocFormat = "yaml"
	FormatJSON     DocFormat = "json"
)

// DocFile represents an indexed document.
type DocFile struct {
	Path    string    `json:"path"`
	RelPath string    `json:"rel_path"`
	Format  DocFormat `json:"format"`
	Title   string    `json:"title,omitempty"`
	Size    int64     `json:"size_bytes"`
	ModTime time.Time `json:"mod_time"`
}

// SearchQuery holds the parsed query.
type SearchQuery struct {
	Terms []string `json:"terms"`
	Raw   string   `json:"raw"`
}

// SearchResult represents a single match returned to the caller.
type SearchResult struct {
	File    DocFile `json:"file"`
	Score   float64 `json:"score"`
	Excerpt string  `json:"excerpt,omitempty"`
	Line    int     `json:"line"`
}

// Posting records a term occurrence within a specific file.
type Posting struct {
	FileIndex int
	Line      int
}

// Index is the in-memory inverted index built by the indexer.
type Index struct {
	Root  string
	Files []DocFile
	Terms map[string][]Posting // lowercase term → postings
}
