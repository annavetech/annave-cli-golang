// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package validator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// commandRunner runs the external tools the validator calls (helm, terraform).
type commandRunner interface {
	LookPath(file string) (string, error)
	// Output returns stdout of name run in dir; an empty dir means the current directory.
	Output(ctx context.Context, dir, name string, args ...string) ([]byte, error)
	// CombinedOutput returns stdout and stderr of name run in the current directory.
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

// allowedTools are the only programs execRunner starts.
var allowedTools = map[string]bool{"helm": true, "terraform": true}

// execRunner runs commands with os/exec.
type execRunner struct{}

func (execRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }

func (execRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd, err := toolCommand(ctx, name, args)
	if err != nil {
		return nil, err
	}
	cmd.Dir = dir
	return cmd.Output()
}

func (execRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd, err := toolCommand(ctx, name, args)
	if err != nil {
		return nil, err
	}
	return cmd.CombinedOutput()
}

// toolCommand builds a Cmd for helm or terraform that is killed when ctx ends; any other program is refused.
func toolCommand(ctx context.Context, name string, args []string) (*exec.Cmd, error) {
	base := filepath.Base(name)
	if !allowedTools[strings.TrimSuffix(base, filepath.Ext(base))] {
		return nil, fmt.Errorf("refusing to run %q: only helm and terraform are allowed", name)
	}
	return exec.CommandContext(ctx, name, args...), nil //nolint:gosec // G204: name is checked to be helm or terraform above; no shell is used.
}
