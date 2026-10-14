// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package searcher

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveEditorFromEnv(t *testing.T) {
	tests := []struct {
		editor, visual string
		want           string
		wantExtra      []string
	}{
		{editor: "nvim", want: "nvim"},
		{editor: "/usr/local/bin/vim", want: "/usr/local/bin/vim"},
		{editor: "code --wait", want: "code", wantExtra: []string{"--wait"}},
		{editor: "/opt/bin/ed -s", want: "/opt/bin/ed", wantExtra: []string{"-s"}},
		{visual: "kate --new", want: "kate", wantExtra: []string{"--new"}},
		{editor: "nano", visual: "kate", want: "nano"},
	}
	for _, tt := range tests {
		t.Run(tt.editor+"|"+tt.visual, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editor)
			t.Setenv("VISUAL", tt.visual)
			got, extra := resolveEditor()
			if got != tt.want || strings.Join(extra, "|") != strings.Join(tt.wantExtra, "|") {
				t.Fatalf("resolveEditor() = %q, %q; want %q, %q", got, extra, tt.want, tt.wantExtra)
			}
		})
	}
}

func TestOpenInEditorRunsFullPathEditor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as the editor")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + filepath.Join(dir, "args.txt") + "\"\n"
	if err := root.WriteFile("myeditor.sh", []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := root.Chmod("myeditor.sh", 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EDITOR", filepath.Join(dir, "myeditor.sh")+" --wait -n")
	t.Setenv("VISUAL", "")
	if err := OpenInEditor(path, 0); err != nil {
		t.Fatal(err)
	}
	got, err := root.ReadFile("args.txt")
	if err != nil {
		t.Fatalf("the script did not run: %v", err)
	}
	if want := "--wait\n-n\n" + path + "\n"; string(got) != want {
		t.Fatalf("script args = %q, want %q", got, want)
	}
}

func TestEditorCommandKeepsExtraWords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		editor string
		line   int
		want   []string
	}{
		{"vi --wait", 0, []string{"vi", "--wait", path}},
		{"/usr/bin/vim -u NONE", 4, []string{"/usr/bin/vim", "-u", "NONE", "+4", path}},
		{"code --wait --new-window", 2, []string{"code", "--wait", "--new-window", "--goto", path + ":2"}},
		{"  nano  ", 0, []string{"nano", path}},
	}
	for _, tt := range tests {
		t.Run(tt.editor, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editor)
			t.Setenv("VISUAL", "")
			cmd, err := editorCommand(path, tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(cmd.Args, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("Args = %q, want %q", cmd.Args, tt.want)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.md")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{"absolute file", file, ""},
		{"relative", "a.md", "must be absolute"},
		{"option-like", "-c", "must be absolute"},
		{"missing", filepath.Join(dir, "missing.md"), "no such file"},
		{"directory", dir, "not a regular file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateFilePath(%q) = %v, want nil", tt.path, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateFilePath(%q) = %v, want error containing %q", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestCommandKeepsPathAsOneArgument(t *testing.T) {
	path := "/tmp/a b; rm -rf x.md"
	cmd := command("vim", editorArgs("vim", path, 3)...)
	want := []string{"vim", "+3", path}
	if strings.Join(cmd.Args, "|") != strings.Join(want, "|") {
		t.Fatalf("Args = %q, want %q", cmd.Args, want)
	}
}
