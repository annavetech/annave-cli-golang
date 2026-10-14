// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package main

import (
	"os"

	"annave.tech/cli/cmd/annave/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
