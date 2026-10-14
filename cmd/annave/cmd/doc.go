// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package cmd

import "github.com/spf13/cobra"

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Search local documentation",
	Long:  "Search .md, .txt, .rst, .html, .yaml, and .json files by full-text query.",
}

func init() {
	rootCmd.AddCommand(docCmd)
}
