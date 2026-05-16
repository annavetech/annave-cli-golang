// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package port

import (
	"context"

	"annave.tech/cli/internal/doc/domain"
)

// SearchOptions configures a doc search run.
type SearchOptions struct {
	Root       string   // directory to index (default: ".")
	Exts       []string // file extensions to include (empty = all supported)
	MaxResults int      // 0 = use config default
}

// Searcher is the port for documentation search.
type Searcher interface {
	Search(ctx context.Context, query domain.SearchQuery, opts SearchOptions) ([]domain.SearchResult, error)
}
