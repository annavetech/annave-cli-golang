// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package cmd

import "github.com/spf13/cobra"

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check service health",
	Long:  "Check uptime and latency of HTTP, TCP, and DNS targets.",
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
