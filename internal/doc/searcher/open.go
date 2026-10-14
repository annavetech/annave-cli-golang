// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package searcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OpenFile opens path in the OS default application (browser for HTML, editor for text).
func OpenFile(path string) error {
	if err := validateFilePath(path); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = command("open", path)
	case "linux":
		cmd = command("xdg-open", path)
	case "windows":
		cmd = command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// OpenInEditor opens path at the given line in $EDITOR (or a detected fallback).
// line=0 means no line hint.
func OpenInEditor(path string, line int) error {
	cmd, err := editorCommand(path, line)
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// editorCommand builds the editor Cmd for path: the program, the extra words of its
// $EDITOR or $VISUAL value, then the line hint and path.
func editorCommand(path string, line int) (*exec.Cmd, error) {
	if err := validateFilePath(path); err != nil {
		return nil, err
	}
	editor, extra := resolveEditor()
	return command(editor, append(extra, editorArgs(editorName(editor), path, line)...)...), nil
}

// command builds a Cmd; no shell parses the name or the arguments.
func command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...) //nolint:gosec // G204: runs the user's own $EDITOR/$VISUAL or the OS opener on a validated absolute path.
}

// validateFilePath accepts only an absolute path to an existing regular file,
// so the program it is passed to cannot read it as an option.
func validateFilePath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("open %q: path must be absolute", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("open %q: not a regular file", path)
	}
	return nil
}

// resolveEditor returns the first word of $EDITOR or $VISUAL as the program, exactly as given,
// and the remaining words as extra arguments; without either it uses the first editor found on PATH.
func resolveEditor() (string, []string) {
	for _, env := range []string{"EDITOR", "VISUAL"} {
		fields := strings.Fields(os.Getenv(env))
		if len(fields) > 0 {
			return fields[0], fields[1:]
		}
	}
	for _, e := range []string{"code", "nvim", "vim", "nano"} {
		if _, err := exec.LookPath(e); err == nil {
			return e, nil
		}
	}
	return "vi", nil
}

// editorName reduces a program path such as "/usr/bin/nvim" to its name.
func editorName(program string) string {
	base := filepath.Base(program)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func editorArgs(editor, path string, line int) []string {
	if line <= 0 {
		return []string{path}
	}
	switch {
	case strings.Contains(editor, "nvim") || strings.Contains(editor, "vim"):
		return []string{fmt.Sprintf("+%d", line), path}
	case strings.Contains(editor, "code"):
		return []string{"--goto", fmt.Sprintf("%s:%d", path, line)}
	case strings.Contains(editor, "nano"):
		return []string{fmt.Sprintf("+%d", line), path}
	default:
		return []string{path}
	}
}
