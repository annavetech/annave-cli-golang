// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package searcher

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/annavetech/tend/internal/doc/domain"
	"github.com/annavetech/tend/internal/doc/port"
	"github.com/annavetech/tend/internal/shared/config"
)

// DocSearcher implements port.Searcher.
type DocSearcher struct{}

func New() *DocSearcher { return &DocSearcher{} }

func (s *DocSearcher) Search(ctx context.Context, query domain.SearchQuery, opts port.SearchOptions) ([]domain.SearchResult, error) {
	idx, err := buildIndex(ctx, opts)
	if err != nil {
		return nil, err
	}
	return search(ctx, idx, query, opts)
}

// ParseQuery splits the raw query string into individual terms.
func ParseQuery(raw string) domain.SearchQuery {
	return domain.SearchQuery{Terms: strings.Fields(raw), Raw: raw}
}

// buildIndex walks opts.Root and builds an inverted index over all matching files.
func buildIndex(ctx context.Context, opts port.SearchOptions) (*domain.Index, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	maxSize := int64(config.App.Limits.Doc.MaxFileSizeMB) * 1024 * 1024
	maxDepth := config.App.Limits.Doc.MaxDepth

	idx := &domain.Index{
		Root:  absRoot,
		Terms: make(map[string][]domain.Posting),
	}

	// Files are opened through an os.Root so a path cannot leave the search root.
	// A root that cannot be read yields an empty index, as the walk below skips it.
	rootDir := absRoot
	info, err := os.Lstat(absRoot)
	if err != nil {
		return idx, nil
	}
	if !info.IsDir() {
		rootDir = filepath.Dir(absRoot)
	}
	rootFS, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Depth check
		relPath, _ := filepath.Rel(absRoot, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if d.IsDir() {
			// The root is searched even when its own name is hidden or a toolchain name.
			if depth >= maxDepth || (path != absRoot && shouldSkipDir(d.Name())) {
				return filepath.SkipDir
			}
			return nil
		}

		format, ok := fileFormat(d.Name(), opts.Exts)
		if !ok {
			return nil
		}

		info, err := d.Info()
		// d.Info does not follow symlinks, so links, devices and pipes are skipped here.
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxSize || info.Size() == 0 {
			return nil
		}

		name, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("open %s: %w", relPath, err)
		}
		f, err := rootFS.Open(name)
		if err != nil {
			return fmt.Errorf("open %s: %w", relPath, err)
		}

		fileIdx := len(idx.Files)
		docFile := domain.DocFile{
			Path:    path,
			RelPath: relPath,
			Format:  format,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), int(maxSize)+1)
		lineNum := 0
		firstHeading := ""

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if firstHeading == "" {
				if h := extractTitle(line, format); h != "" {
					firstHeading = h
				}
			}
			for _, token := range tokenize(line) {
				idx.Terms[token] = append(idx.Terms[token], domain.Posting{
					FileIndex: fileIdx,
					Line:      lineNum,
				})
			}
		}
		scanErr := scanner.Err()
		if err := f.Close(); err != nil && scanErr == nil {
			scanErr = err
		}
		if scanErr != nil {
			return fmt.Errorf("read %s: %w", relPath, scanErr)
		}

		if firstHeading != "" {
			docFile.Title = firstHeading
		} else {
			docFile.Title = filenameTitle(d.Name())
		}
		idx.Files = append(idx.Files, docFile)
		return nil
	})
	if cerr := rootFS.Close(); cerr != nil && err == nil {
		err = cerr
	}

	return idx, err
}

var skipDirs = map[string]bool{
	".git": true, ".svn": true, "node_modules": true, "vendor": true,
	".cache": true, "__pycache__": true, ".next": true, "dist": true,
	"build": true, ".angular": true, ".idea": true, ".vscode": true,
}

// shouldSkipDir returns true for hidden directories and known toolchain dirs.
func shouldSkipDir(name string) bool {
	return skipDirs[name] || strings.HasPrefix(name, ".")
}

// tokenize splits s into lowercase tokens of 3+ characters, deduplicated.
func tokenize(s string) []string {
	seen := make(map[string]bool)
	var tokens []string
	start := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		isWord := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
		if isWord {
			if start < 0 {
				start = i
			}
		} else {
			if start >= 0 {
				tok := strings.ToLower(s[start:i])
				if len(tok) >= 3 && !seen[tok] {
					seen[tok] = true
					tokens = append(tokens, tok)
				}
				start = -1
			}
		}
	}
	if start >= 0 {
		tok := strings.ToLower(s[start:])
		if len(tok) >= 3 && !seen[tok] {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}

func extractTitle(line string, format domain.DocFormat) string {
	switch format {
	case domain.FormatMarkdown:
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	case domain.FormatHTML:
		lower := strings.ToLower(line)
		if i := strings.Index(lower, "<h1"); i >= 0 {
			if j := strings.Index(lower[i:], ">"); j >= 0 {
				rest := line[i+j+1:]
				if k := strings.Index(strings.ToLower(rest), "</h1>"); k >= 0 {
					return strings.TrimSpace(rest[:k])
				}
			}
		}
	}
	return ""
}

func filenameTitle(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return strings.ReplaceAll(strings.ReplaceAll(base, "-", " "), "_", " ")
}
