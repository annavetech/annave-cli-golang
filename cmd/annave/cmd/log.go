// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import "github.com/spf13/cobra"

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Analyze log files",
	Long:  "Parse and analyze log files to surface errors, spikes, and repeated patterns.",
}

func init() {
	rootCmd.AddCommand(logCmd)
}
