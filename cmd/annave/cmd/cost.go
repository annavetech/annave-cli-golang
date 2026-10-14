// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import "github.com/spf13/cobra"

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Analyze cloud spending and detect anomalies (in development)",
	Long:  "Connect to cloud provider billing APIs to surface cost anomalies and resource attribution.",
}

func init() {
	rootCmd.AddCommand(costCmd)
}
