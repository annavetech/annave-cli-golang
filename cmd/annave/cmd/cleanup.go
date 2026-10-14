// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import "github.com/spf13/cobra"

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Scan for idle Kubernetes resources",
	Long:  "Scan Kubernetes clusters for idle pods, stale PVCs, orphaned ConfigMaps, and empty namespaces.",
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
