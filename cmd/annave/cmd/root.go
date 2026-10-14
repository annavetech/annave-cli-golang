// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("annave version %s\n", Version)
	},
}

// Execute runs the root cobra command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
