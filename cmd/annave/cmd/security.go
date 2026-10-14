// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import "github.com/spf13/cobra"

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Audit code, secrets, containers, and cluster config (in development)",
	Long:  "SAST, secret scanning, container image analysis, and Kubernetes misconfiguration detection.",
}

func init() {
	rootCmd.AddCommand(securityCmd)
}
