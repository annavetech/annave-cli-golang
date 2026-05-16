// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package searcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenFile opens path in the OS default application (browser for HTML, editor for text).
func OpenFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// OpenInEditor opens path at the given line in $EDITOR (or a detected fallback).
// line=0 means no line hint.
func OpenInEditor(path string, line int) error {
	editor := resolveEditor()
	args := editorArgs(editor, path, line)
	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resolveEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	for _, e := range []string{"code", "nvim", "vim", "nano"} {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}
	return "vi"
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
