// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"annave.tech/cli/internal/shared/output"
)

// Version is the current release tag embedded at build time.
const Version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "annave",
	Short: "ANNÁVE developer tools",
	Long: `ANNÁVE CLI — engineering tools for log analysis, service health checking,
documentation search, Kubernetes cleanup, and more.

  annave log analyze [file]     — find what's broken in your logs
  annave health check [targets] — check service uptime and latency
  annave doc search [query]     — search documentation from the terminal
  annave cleanup scan           — find unused Kubernetes resources

Run annave <command> --help for details on any command.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("annave version %s\n", Version)
	},
}

// Exit codes returned by Execute.
const (
	exitOK       = 0
	exitError    = 1
	exitProblems = 2
)

// errProblemsFound is returned by RunE after results are printed when findings,
// down targets or a failed validation exist; Execute maps it to exit code 2.
var errProblemsFound = errors.New("problems found")

// wrapArgsOnce runs wrapArgs after every subcommand is registered.
var wrapArgsOnce sync.Once

// usageError marks a flag or argument error, which gets a pointer to --help.
type usageError struct {
	err  error
	path string
}

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

func newUsageError(c *cobra.Command, err error) error {
	return &usageError{err: err, path: c.CommandPath()}
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// After the first signal, restore default handling so a second Ctrl-C kills the process.
	go func() {
		<-ctx.Done()
		stop()
	}()
	return execute(ctx, os.Args[1:])
}

// execute runs the root command with args under ctx and prints at most one error.
func execute(ctx context.Context, args []string) int {
	wrapArgsOnce.Do(func() { wrapArgs(rootCmd) })
	// Cobra keeps a subcommand context from an earlier run, so set it on every call.
	setContext(rootCmd, ctx)
	rootCmd.SetArgs(args)
	c, err := rootCmd.ExecuteC()
	if err == nil {
		return exitOK
	}
	if errors.Is(err, errProblemsFound) {
		return exitProblems
	}
	var ue *usageError
	if !errors.As(err, &ue) && c != nil && !c.Runnable() {
		// A command that cannot run fails only on command lookup.
		ue = &usageError{err: err, path: c.CommandPath()}
	}
	if werr := printError(os.Stdout, os.Stderr, c, err, ue); werr != nil {
		// Neither stream takes the message; the exit code still reports the failure.
		return exitError
	}
	return exitError
}

// printError writes err as JSON to stdout in JSON mode, else as one line to stderr.
func printError(stdout, stderr io.Writer, c *cobra.Command, err error, ue *usageError) error {
	if isJSONFormat(c) {
		b, merr := json.Marshal(struct {
			Error string `json:"error"`
		}{err.Error()})
		if merr != nil {
			return merr
		}
		if _, werr := fmt.Fprintf(stdout, "%s\n", b); werr != nil {
			return werr
		}
	} else if _, werr := fmt.Fprintf(stderr, "error: %v\n", err); werr != nil {
		return werr
	}
	if ue != nil {
		if _, werr := fmt.Fprintf(stderr, "Run '%s --help' for usage.\n", ue.path); werr != nil {
			return werr
		}
	}
	return nil
}

// isJSONFormat reports whether c was run with a valid --format json.
func isJSONFormat(c *cobra.Command) bool {
	if c == nil {
		return false
	}
	f := c.Flags().Lookup("format")
	if f == nil {
		return false
	}
	format, err := output.ParseFormat(f.Value.String())
	return err == nil && format == output.FormatJSON
}

// setContext sets ctx on c and all its children.
func setContext(c *cobra.Command, ctx context.Context) {
	c.SetContext(ctx)
	for _, child := range c.Commands() {
		setContext(child, ctx)
	}
}

// wrapArgs marks argument validation errors of c and its children as usage errors.
func wrapArgs(c *cobra.Command) {
	if validate := c.Args; validate != nil {
		c.Args = func(cmd *cobra.Command, args []string) error {
			if err := validate(cmd, args); err != nil {
				return newUsageError(cmd, err)
			}
			return nil
		}
	}
	for _, child := range c.Commands() {
		wrapArgs(child)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetFlagErrorFunc(newUsageError)
}
