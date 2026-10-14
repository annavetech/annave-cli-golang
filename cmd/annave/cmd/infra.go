// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import "github.com/spf13/cobra"

var infraCmd = &cobra.Command{
	Use:   "infra",
	Short: "Validate Terraform plans and Helm releases (in development)",
	Long:  "Parse and validate Terraform plans, Helm releases, and infrastructure configurations.",
}

func init() {
	rootCmd.AddCommand(infraCmd)
}
