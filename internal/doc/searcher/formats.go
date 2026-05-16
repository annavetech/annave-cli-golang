// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package searcher

import (
	"path/filepath"
	"strings"

	"annave.tech/cli/internal/doc/domain"
)

// supportedFormats maps lowercase file extensions to DocFormat.
var supportedFormats = map[string]domain.DocFormat{
	".md":   domain.FormatMarkdown,
	".mdx":  domain.FormatMarkdown,
	".txt":  domain.FormatText,
	".rst":  domain.FormatRST,
	".html": domain.FormatHTML,
	".htm":  domain.FormatHTML,
	".yaml": domain.FormatYAML,
	".yml":  domain.FormatYAML,
	".json": domain.FormatJSON,
}

// fileFormat returns the DocFormat for a filename.
// ok=false means the file should be skipped (unsupported or filtered by exts).
func fileFormat(name string, exts []string) (domain.DocFormat, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	format, ok := supportedFormats[ext]
	if !ok {
		return "", false
	}
	if len(exts) == 0 {
		return format, true
	}
	for _, e := range exts {
		if strings.TrimPrefix(strings.ToLower(e), ".") == strings.TrimPrefix(ext, ".") {
			return format, true
		}
	}
	return "", false
}
