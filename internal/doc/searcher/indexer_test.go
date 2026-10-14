// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package searcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/annavetech/tend/internal/doc/port"
)

func indexHas(t *testing.T, root, term string) bool {
	t.Helper()
	idx, err := buildIndex(context.Background(), port.SearchOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	return len(idx.Terms[term]) > 0
}

func TestIndexHiddenRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".hidden")
	if err := os.MkdirAll(filepath.Join(root, ".nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "h.md"), []byte("findme\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".nested", "n.md"), []byte("skipme\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if !indexHas(t, root, "findme") {
		t.Error("word in .hidden/h.md not indexed")
	}
	if indexHas(t, root, "skipme") {
		t.Error("hidden subdirectory below the root must still be skipped")
	}
}

func TestIndexWordAfterLongLine(t *testing.T) {
	root := t.TempDir()
	content := `{"blob":"` + strings.Repeat("x", 70*1024) + "\"}\nfindme\n"
	if err := os.WriteFile(filepath.Join(root, "big.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if !indexHas(t, root, "findme") {
		t.Error("word after a 70 KB line not indexed")
	}
}

func TestIndexStopsOnUnopenableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can open any file")
	}
	root := t.TempDir()
	path := filepath.Join(root, "locked.md")
	if err := os.WriteFile(path, []byte("text\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	_, err := buildIndex(context.Background(), port.SearchOptions{Root: root})
	if err == nil || !strings.Contains(err.Error(), "locked.md") {
		t.Fatalf("want an error naming locked.md, got %v", err)
	}
}

func TestIndexSkipsNonRegularMarkdown(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dir.md"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "b.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "dir.md"), filepath.Join(root, "dirlink.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ok.md"), []byte("findme\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !indexHas(t, root, "findme") {
		t.Error("ok.md not indexed")
	}
}

func TestIndexSingleFileRoot(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.md")
	if err := os.WriteFile(file, []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !indexHas(t, file, "needle") {
		t.Fatal("want the single file root indexed")
	}
}

func TestIndexMissingRootIsEmpty(t *testing.T) {
	idx, err := buildIndex(context.Background(), port.SearchOptions{Root: filepath.Join(t.TempDir(), "missing")})
	if err != nil || len(idx.Files) != 0 {
		t.Fatalf("files=%v err=%v, want an empty index", idx.Files, err)
	}
}

func TestSearchExcerptReadsThroughRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("intro\n  the needle line  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	results, err := New().Search(context.Background(), ParseQuery("needle"), port.SearchOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Excerpt != "the needle line" || results[0].Line != 2 {
		t.Fatalf("results = %+v, want one excerpt from line 2", results)
	}
}

func TestSearchExcerptForFileRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "real")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "a.md")
	if err := os.WriteFile(file, []byte("intro\nthe needle line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	results, err := New().Search(context.Background(), ParseQuery("needle"), port.SearchOptions{Root: file})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Excerpt != "the needle line" {
		t.Fatalf("results = %+v, want the excerpt from line 2", results)
	}
}
